package ws

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/ocppj"
)

type pingReq struct {
	Message string `json:"message"`
}

func (pingReq) Action() string { return "Ping" }
func (r pingReq) Validate() error {
	if r.Message == "" {
		return ocpp.Required("message")
	}
	return nil
}

type pingResp struct {
	Reply string `json:"reply"`
}

func (pingResp) Action() string  { return "Ping" }
func (pingResp) Validate() error { return nil }

func pingRegistry(t testing.TB) *ocppj.Registry {
	t.Helper()
	f, err := ocpp.NewFeature(pingReq{}, pingResp{})
	if err != nil {
		t.Fatal(err)
	}
	reg := ocppj.NewRegistry(ocpp.V201)
	if err := reg.Register(f); err != nil {
		t.Fatal(err)
	}
	return reg
}

// TestOCPPJOverWebSocket wires ocppj endpoints to both transports and makes
// calls in both directions over a real WebSocket.
func TestOCPPJOverWebSocket(t *testing.T) {
	var mu sync.Mutex
	endpoints := map[string]*ocppj.Endpoint{}
	newEndpoint := func(send func(context.Context, []byte) error, h ocppj.RequestHandler) *ocppj.Endpoint {
		ep, err := ocppj.NewEndpoint(ocppj.Config{Registry: pingRegistry(t), Send: send, Handler: h, Timeout: 2 * time.Second})
		if err != nil {
			t.Fatal(err)
		}
		return ep
	}

	// CSMS: one endpoint per connection.
	srec := newRecorder()
	_, _, base := startServer(t, func(cfg *ServerConfig) {
		cfg.OnConnect = func(c *Conn) {
			ep := newEndpoint(c.Write, func(_ context.Context, req ocpp.Request) (ocpp.Response, error) {
				return &pingResp{Reply: "csms:" + req.(*pingReq).Message}, nil
			})
			mu.Lock()
			endpoints[c.ID()] = ep
			mu.Unlock()
			srec.connected <- c
		}
		cfg.OnFrame = func(c *Conn, f []byte) {
			mu.Lock()
			ep := endpoints[c.ID()]
			mu.Unlock()
			ep.HandleFrame(f)
		}
		cfg.OnDisconnect = func(c *Conn, err error) {
			mu.Lock()
			ep := endpoints[c.ID()]
			delete(endpoints, c.ID())
			mu.Unlock()
			ep.Close()
			srec.disconnected <- disconnect{c, err}
		}
	})

	// Charging station.
	var station *ocppj.Endpoint
	cl, rec, _ := startClient(t, base, func(cfg *ClientConfig) {
		cfg.OnFrame = func(_ *Conn, f []byte) { station.HandleFrame(f) }
	})
	station = newEndpoint(cl.Send, func(_ context.Context, req ocpp.Request) (ocpp.Response, error) {
		return &pingResp{Reply: "station:" + req.(*pingReq).Message}, nil
	})
	t.Cleanup(station.Close)

	recv(t, rec.connected)
	sconn := recv(t, srec.connected)
	ctx := context.Background()

	// Station -> CSMS.
	resp, err := station.Call(ctx, pingReq{Message: "hello"})
	if err != nil || resp.(*pingResp).Reply != "csms:hello" {
		t.Fatalf("%v %v", resp, err)
	}
	// CSMS -> station.
	mu.Lock()
	ep := endpoints[sconn.ID()]
	mu.Unlock()
	resp, err = ep.Call(ctx, pingReq{Message: "hi"})
	if err != nil || resp.(*pingResp).Reply != "station:hi" {
		t.Fatalf("%v %v", resp, err)
	}
	// Invalid requests never reach the wire.
	if _, err := station.Call(ctx, pingReq{}); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("got %v", err)
	}

	// After the connection drops, calls fail fast with ErrNotConnected.
	_ = sconn.closeWith(nil, 1011, "drop")
	recv(t, rec.disconnected)
	if _, err := station.Call(ctx, pingReq{Message: "x"}); err == nil || (!errors.Is(err, ocpp.ErrNotConnected) && !errors.Is(err, ocpp.ErrTimeout)) {
		t.Errorf("call while disconnected: %v", err)
	}
	// ...and work again once the client has reconnected.
	recv(t, rec.connected)
	recv(t, srec.connected)
	resp, err = station.Call(ctx, pingReq{Message: "again"})
	if err != nil || resp.(*pingResp).Reply != "csms:again" {
		t.Fatalf("after reconnect: %v %v", resp, err)
	}
}
