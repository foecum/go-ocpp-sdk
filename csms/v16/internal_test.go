package csms

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/foecum/go-ocpp-sdk/ocppj"
	"github.com/foecum/go-ocpp-sdk/ws"
)

// These tests replace the construction seams to reach error paths that cannot
// happen with the fixed arguments the package uses.

func TestNewReportsInternalFailures(t *testing.T) {
	boom := errors.New("boom")

	old := newRegistry
	newRegistry = func() (*ocppj.Registry, error) { return nil, boom }
	_, err := New(Config{})
	newRegistry = old
	if !errors.Is(err, boom) {
		t.Errorf("registry failure: %v", err)
	}

	oldWS := newWSServer
	newWSServer = func(ws.ServerConfig) (*ws.Server, error) { return nil, boom }
	_, err = New(Config{})
	newWSServer = oldWS
	if !errors.Is(err, boom) {
		t.Errorf("server failure: %v", err)
	}
}

func TestConnectionIsRefusedIfItsEndpointCannotBeBuilt(t *testing.T) {
	old := newEndpoint
	newEndpoint = func(ocppj.Config) (*ocppj.Endpoint, error) { return nil, errors.New("boom") }
	defer func() { newEndpoint = old }()

	hooks := make(chan struct{}, 2)
	srv, err := New(Config{
		OnConnect:    func(*ChargePoint) { hooks <- struct{}{} },
		OnDisconnect: func(*ChargePoint, error) { hooks <- struct{}{} },
	})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv)
	defer ts.Close()
	defer func() { _ = srv.Shutdown(context.Background()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(ts.URL, "http")+"/ocpp/CS1", &websocket.DialOptions{Subprotocols: []string{"ocpp1.6"}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()

	// The server registers the connection, fails to build its endpoint and
	// closes it. While it waits for us to acknowledge, the CSMS knows the
	// connection but has no ChargePoint for it.
	deadline := time.Now().Add(3 * time.Second)
	for len(srv.ChargePoints()) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("connection never registered")
		}
		time.Sleep(time.Millisecond)
	}
	if cp, ok := srv.ChargePoint("CS1"); ok || cp != nil {
		t.Error("a connection without an endpoint must not be exposed")
	}
	// Reading lets the close handshake finish; the disconnect then has no
	// ChargePoint to report and must not call the hooks.
	_, _, _ = c.Read(ctx)
	deadline = time.Now().Add(3 * time.Second)
	for len(srv.ChargePoints()) != 0 {
		if time.Now().After(deadline) {
			t.Fatal("connection never removed")
		}
		time.Sleep(time.Millisecond)
	}
	select {
	case <-hooks:
		t.Error("hooks must not fire for a connection that never became a ChargePoint")
	case <-time.After(50 * time.Millisecond):
	}
}
