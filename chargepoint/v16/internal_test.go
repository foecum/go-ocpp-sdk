package chargepoint

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"context"

	"github.com/coder/websocket"

	"github.com/foecum/go-ocpp-sdk/ocppj"
	"github.com/foecum/go-ocpp-sdk/ws"
)

func fastBackoff() ws.Reconnect {
	return ws.Reconnect{WaitMinimum: 5 * time.Millisecond, RandomRange: 5 * time.Millisecond, RepeatTimes: 2}
}

func TestNewReportsInternalFailure(t *testing.T) {
	boom := errors.New("boom")
	old := newRegistry
	newRegistry = func() (*ocppj.Registry, error) { return nil, boom }
	defer func() { newRegistry = old }()
	if _, err := New(Config{URL: "ws://localhost/ocpp", ID: "CS1"}); !errors.Is(err, boom) {
		t.Errorf("got %v", err)
	}
}

func TestConnectionIsDroppedIfItsEndpointCannotBeBuilt(t *testing.T) {
	old := newEndpoint
	newEndpoint = func(ocppj.Config) (*ocppj.Endpoint, error) { return nil, errors.New("boom") }
	defer func() { newEndpoint = old }()

	accepted := make(chan struct{}, 8)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"ocpp1.6"}})
		if err != nil {
			return
		}
		accepted <- struct{}{}
		// Read so the station's close handshake completes.
		_, _, _ = c.Read(r.Context())
		_ = c.CloseNow()
	}))
	defer ts.Close()

	cp, err := New(Config{URL: "ws" + strings.TrimPrefix(ts.URL, "http") + "/ocpp", ID: "CS1",
		Reconnect: fastBackoff()})
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = cp.Run(context.Background()) }()
	defer cp.Close()

	// It connects, finds it cannot serve the connection, drops it, and tries again.
	for i := 0; i < 2; i++ {
		select {
		case <-accepted:
		case <-time.After(3 * time.Second):
			t.Fatal("the station did not retry")
		}
	}
	if cp.Connected() {
		t.Error("a connection without an endpoint must not count as connected")
	}
}
