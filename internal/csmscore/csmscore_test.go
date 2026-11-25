package csmscore

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/ocppj"
	"github.com/foecum/go-ocpp-sdk/v16"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v21"
	"github.com/foecum/go-ocpp-sdk/v21/diagnostics"
	"github.com/foecum/go-ocpp-sdk/v21/types"
	"github.com/foecum/go-ocpp-sdk/ws"
)

type handle struct{ *Session }

func newTestServer(t *testing.T, mod func(*Config[handle])) (*Server[handle], string) {
	t.Helper()
	reg, err := v16.NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config[handle]{
		Version:   ocpp.V16,
		Registry:  reg,
		NewHandle: func(s *Session) *handle { return &handle{s} },
		Dispatch: func(ctx context.Context, h *handle, req ocpp.Request) (ocpp.Response, error) {
			return &core.HeartbeatConfirmation{CurrentTime: ocpp.Now()}, nil
		},
	}
	if mod != nil {
		mod(&cfg)
	}
	srv, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv)
	t.Cleanup(func() { ts.Close(); _ = srv.Shutdown(context.Background()) })
	return srv, "ws" + strings.TrimPrefix(ts.URL, "http")
}

func dial(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{Subprotocols: []string{"ocpp1.6"}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.CloseNow() })
	return c
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timeout waiting for " + what)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestNewReportsServerFailure(t *testing.T) {
	boom := errors.New("boom")
	old := newWSServer
	newWSServer = func(ws.ServerConfig) (*ws.Server, error) { return nil, boom }
	defer func() { newWSServer = old }()
	if _, err := New(Config[handle]{}); !errors.Is(err, boom) {
		t.Errorf("got %v", err)
	}
}

func TestSessionLookupAndCall(t *testing.T) {
	connected := make(chan *handle, 1)
	disconnected := make(chan error, 1)
	srv, url := newTestServer(t, func(c *Config[handle]) {
		c.OnConnect = func(h *handle) { connected <- h }
		c.OnDisconnect = func(h *handle, err error) { disconnected <- err }
	})
	c := dial(t, url+"/ocpp/CS1")
	h := <-connected
	if h.ID() != "CS1" || h.Version() != ocpp.V16 || h.RemoteAddr() == "" || h.Endpoint() == nil {
		t.Errorf("session: %q %v %q", h.ID(), h.Version(), h.RemoteAddr())
	}
	if got, ok := srv.Lookup("CS1"); !ok || got != h {
		t.Error("lookup failed")
	}
	if _, ok := srv.Lookup("nope"); ok {
		t.Error("unknown id found")
	}
	if ids := srv.IDs(); len(ids) != 1 || ids[0] != "CS1" {
		t.Errorf("ids %v", ids)
	}

	// A call to the charge point is answered by our fake station.
	go func() {
		ctx := context.Background()
		_, data, err := c.Read(ctx)
		if err != nil {
			return
		}
		frame := string(data) // [2,"id","ClearCache",{}]
		id := strings.Split(frame, `"`)[1]
		_ = c.Write(ctx, websocket.MessageText, []byte(`[3,"`+id+`",{"status":"Accepted"}]`))
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := Call[core.ClearCacheConfirmation](ctx, h.Session, core.ClearCacheRequest{})
	if err != nil || resp.Status != "Accepted" {
		t.Fatalf("call: %v %v", resp, err)
	}

	// An incoming call is dispatched and answered.
	if err := c.Write(ctx, websocket.MessageText, []byte(`[2,"7","Heartbeat",{}]`)); err != nil {
		t.Fatal(err)
	}
	_, data, err := c.Read(ctx)
	if err != nil || !strings.HasPrefix(string(data), `[3,"7",{"currentTime"`) {
		t.Fatalf("reply %s %v", data, err)
	}

	if err := h.Close(); err != nil {
		t.Logf("close: %v", err)
	}
	select {
	case <-h.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("Done not closed")
	}
	_, _, _ = c.Read(ctx)
	select {
	case <-disconnected:
	case <-time.After(3 * time.Second):
		t.Fatal("OnDisconnect not called")
	}
	waitFor(t, "removal", func() bool { return len(srv.IDs()) == 0 })
}

func TestConnectionIsRefusedIfItsEndpointCannotBeBuilt(t *testing.T) {
	old := newEndpoint
	newEndpoint = func(ocppj.Config) (*ocppj.Endpoint, error) { return nil, errors.New("boom") }
	defer func() { newEndpoint = old }()

	hooks := make(chan struct{}, 2)
	srv, url := newTestServer(t, func(c *Config[handle]) {
		c.OnConnect = func(*handle) { hooks <- struct{}{} }
		c.OnDisconnect = func(*handle, error) { hooks <- struct{}{} }
	})
	c := dial(t, url+"/ocpp/CS1")

	// The server registers the connection, fails to build its endpoint and
	// closes it. While it waits for us to acknowledge, it knows the connection
	// but has no handle for it.
	waitFor(t, "registration", func() bool { return len(srv.IDs()) != 0 })
	if h, ok := srv.Lookup("CS1"); ok || h != nil {
		t.Error("a connection without an endpoint must not be exposed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _, _ = c.Read(ctx)
	waitFor(t, "removal", func() bool { return len(srv.IDs()) == 0 })
	select {
	case <-hooks:
		t.Error("hooks must not fire for a connection that never became a handle")
	case <-time.After(50 * time.Millisecond):
	}
}

// An OCPP 2.1 CSMS receives unconfirmed messages through DispatchSend and sends
// its own with Send.
func TestUnconfirmedMessages(t *testing.T) {
	reg, err := v21.NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	got := make(chan ocpp.Request, 1)
	connected := make(chan *handle, 1)
	_, url := newTestServer(t, func(c *Config[handle]) {
		c.Version, c.Registry = ocpp.V21, reg
		c.OnConnect = func(h *handle) { connected <- h }
		c.DispatchSend = func(_ context.Context, _ *handle, m ocpp.Request) error { got <- m; return nil }
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, url+"/ocpp/CS1", &websocket.DialOptions{Subprotocols: []string{"ocpp2.1"}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	h := <-connected

	if err := c.Write(ctx, websocket.MessageText, []byte(`[6,"x1","NotifyPeriodicEventStream",{"id":3,"pending":0,"basetime":"2026-01-01T00:00:00Z","data":[{"t":0,"v":"1"}]}]`)); err != nil {
		t.Fatal(err)
	}
	select {
	case m := <-got:
		if m.(*diagnostics.NotifyPeriodicEventStream).ID != 3 {
			t.Errorf("%+v", m)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("DispatchSend not called")
	}

	msg := diagnostics.NotifyPeriodicEventStream{ID: 8, Basetime: ocpp.Now(), Data: []types.StreamDataElement{{V: "1"}}}
	if err := Send(ctx, h.Session, msg); err != nil {
		t.Fatal(err)
	}
	_, data, err := c.Read(ctx)
	if err != nil || !strings.HasPrefix(string(data), `[6,"`) {
		t.Fatalf("frame %s %v", data, err)
	}
}

func TestForEach(t *testing.T) {
	connected := make(chan *handle, 3)
	srv, url := newTestServer(t, func(c *Config[handle]) { c.OnConnect = func(h *handle) { connected <- h } })
	for _, id := range []string{"CS3", "CS1", "CS2"} {
		dial(t, url+"/ocpp/"+id)
		<-connected
	}
	var got []string
	srv.ForEach(func(h *handle) bool { got = append(got, h.ID()); return true })
	if len(got) != 3 || got[0] != "CS1" || got[1] != "CS2" || got[2] != "CS3" {
		t.Errorf("visited %v", got)
	}
	got = nil
	srv.ForEach(func(h *handle) bool { got = append(got, h.ID()); return false })
	if len(got) != 1 || got[0] != "CS1" {
		t.Errorf("must stop when fn returns false: %v", got)
	}
}
