package cpcore

import (
	"context"
	"errors"
	"net/http"
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

func fastBackoff() ws.Reconnect {
	return ws.Reconnect{WaitMinimum: 5 * time.Millisecond, RandomRange: 5 * time.Millisecond, RepeatTimes: 2}
}

func testConfig(t *testing.T, url string) Config {
	t.Helper()
	reg, err := v16.NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	return Config{
		Version:  ocpp.V16,
		Registry: reg,
		Dispatch: func(ctx context.Context, req ocpp.Request) (ocpp.Response, error) {
			return &core.ClearCacheConfirmation{Status: "Accepted"}, nil
		},
		URL:       url,
		ID:        "CS1",
		Reconnect: fastBackoff(),
	}
}

func TestNewReportsClientFailure(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Error("an empty config must be rejected")
	}
}

func TestCallWhileDisconnected(t *testing.T) {
	s, err := New(testConfig(t, "ws://localhost:1/ocpp"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Connected() || s.Endpoint() != nil {
		t.Error("must start disconnected")
	}
	if _, err := Call[core.HeartbeatConfirmation](context.Background(), s, core.HeartbeatRequest{}); !errors.Is(err, ocpp.ErrNotConnected) {
		t.Errorf("got %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := s.WaitConnected(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got %v", err)
	}
	// A frame with no connection is ignored.
	s.onFrame(nil, []byte(`[3,"1",{}]`))
}

func TestConnectCallDispatchAndReconnect(t *testing.T) {
	conns := make(chan *websocket.Conn, 4)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"ocpp1.6"}})
		if err != nil {
			return
		}
		conns <- c
		<-r.Context().Done()
	}))
	defer ts.Close()

	cfg := testConfig(t, "ws"+strings.TrimPrefix(ts.URL, "http")+"/ocpp")
	up := make(chan struct{}, 4)
	down := make(chan error, 4)
	cfg.OnConnect = func() { up <- struct{}{} }
	cfg.OnDisconnect = func(err error) { down <- err }
	s, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = s.Run(context.Background()) }()
	defer s.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.WaitConnected(ctx); err != nil {
		t.Fatal(err)
	}
	<-up
	if !s.Connected() {
		t.Error("must be connected")
	}
	srv := <-conns

	// Incoming call is dispatched and answered.
	if err := srv.Write(ctx, websocket.MessageText, []byte(`[2,"9","ClearCache",{}]`)); err != nil {
		t.Fatal(err)
	}
	_, data, err := srv.Read(ctx)
	if err != nil || string(data) != `[3,"9",{"status":"Accepted"}]` {
		t.Fatalf("reply %s %v", data, err)
	}

	// Outgoing call.
	go func() {
		_, data, err := srv.Read(ctx)
		if err != nil {
			return
		}
		id := strings.Split(string(data), `"`)[1]
		_ = srv.Write(ctx, websocket.MessageText, []byte(`[3,"`+id+`",{"currentTime":"2026-01-01T00:00:00Z"}]`))
	}()
	if _, err := Call[core.HeartbeatConfirmation](ctx, s, core.HeartbeatRequest{}); err != nil {
		t.Fatal(err)
	}

	// The server drops the connection: hook fires and the station reconnects.
	_ = srv.CloseNow()
	select {
	case <-down:
	case <-time.After(3 * time.Second):
		t.Fatal("OnDisconnect not called")
	}
	if err := s.WaitConnected(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-up:
	case <-time.After(3 * time.Second):
		t.Fatal("no reconnect")
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

	s, err := New(testConfig(t, "ws"+strings.TrimPrefix(ts.URL, "http")+"/ocpp"))
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = s.Run(context.Background()) }()
	defer s.Close()

	// It connects, finds it cannot serve the connection, drops it, and tries again.
	for i := 0; i < 2; i++ {
		select {
		case <-accepted:
		case <-time.After(3 * time.Second):
			t.Fatal("the station did not retry")
		}
	}
	if s.Connected() {
		t.Error("a connection without an endpoint must not count as connected")
	}
}

func TestSendWhileDisconnected(t *testing.T) {
	s, err := New(testConfig(t, "ws://localhost:1/ocpp"))
	if err != nil {
		t.Fatal(err)
	}
	msg := diagnostics.NotifyPeriodicEventStream{ID: 1, Basetime: ocpp.Now(), Data: []types.StreamDataElement{{V: "1"}}}
	if err := Send(context.Background(), s, msg); !errors.Is(err, ocpp.ErrNotConnected) {
		t.Errorf("got %v", err)
	}
}

// An OCPP 2.1 station sends unconfirmed messages and receives them through DispatchSend.
func TestUnconfirmedMessages(t *testing.T) {
	conns := make(chan *websocket.Conn, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"ocpp2.1"}})
		if err != nil {
			return
		}
		conns <- c
		<-r.Context().Done()
	}))
	defer ts.Close()

	reg, err := v21.NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	got := make(chan ocpp.Request, 1)
	s, err := New(Config{
		Version: ocpp.V21, Registry: reg, URL: "ws" + strings.TrimPrefix(ts.URL, "http") + "/ocpp", ID: "CS1", Reconnect: fastBackoff(),
		Dispatch:     func(context.Context, ocpp.Request) (ocpp.Response, error) { return nil, errors.New("unused") },
		DispatchSend: func(_ context.Context, m ocpp.Request) error { got <- m; return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = s.Run(context.Background()) }()
	defer s.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.WaitConnected(ctx); err != nil {
		t.Fatal(err)
	}
	srv := <-conns

	msg := diagnostics.NotifyPeriodicEventStream{ID: 4, Basetime: ocpp.Now(), Data: []types.StreamDataElement{{V: "1"}}}
	if err := Send(ctx, s, msg); err != nil {
		t.Fatal(err)
	}
	_, data, err := srv.Read(ctx)
	if err != nil || !strings.HasPrefix(string(data), `[6,"`) || !strings.Contains(string(data), `"NotifyPeriodicEventStream"`) {
		t.Fatalf("frame %s %v", data, err)
	}
	// The other side's SEND reaches DispatchSend.
	if err := srv.Write(ctx, websocket.MessageText, []byte(`[6,"x1","NotifyPeriodicEventStream",{"id":9,"pending":0,"basetime":"2026-01-01T00:00:00Z","data":[{"t":0,"v":"1"}]}]`)); err != nil {
		t.Fatal(err)
	}
	select {
	case m := <-got:
		if m.(*diagnostics.NotifyPeriodicEventStream).ID != 9 {
			t.Errorf("%+v", m)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("DispatchSend not called")
	}
}
