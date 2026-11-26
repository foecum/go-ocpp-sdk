package ws

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

func TestNewServerValidation(t *testing.T) {
	ok := func(*Conn, []byte) {}
	for name, cfg := range map[string]ServerConfig{
		"no OnFrame":      {Versions: []ocpp.Version{ocpp.V16}},
		"no versions":     {OnFrame: ok},
		"unknown version": {Versions: []ocpp.Version{"ocpp9"}, OnFrame: ok},
	} {
		if _, err := NewServer(cfg); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	s, err := NewServer(ServerConfig{Versions: []ocpp.Version{ocpp.V16}, OnFrame: ok})
	if err != nil || s.opts.pingInterval != DefaultPingInterval || s.opts.pongTimeout != DefaultPongTimeout ||
		s.opts.readLimit != DefaultReadLimit || s.cfg.Logger == nil {
		t.Errorf("defaults: %+v %v", s.opts, err)
	}
}

func TestHandshakeAndFrames(t *testing.T) {
	s, rec, base := startServer(t, nil)
	c := mustDial(t, base, "CS001", "ocpp2.0.1")
	if c.Subprotocol() != "ocpp2.0.1" {
		t.Fatalf("subprotocol %q", c.Subprotocol())
	}
	conn := recv(t, rec.connected)
	if conn.ID() != "CS001" || conn.Version() != ocpp.V201 || conn.RemoteAddr() == "" {
		t.Errorf("conn %q %q %q", conn.ID(), conn.Version(), conn.RemoteAddr())
	}
	if got, ok := s.Conn("CS001"); !ok || got != conn {
		t.Error("not registered")
	}
	if _, ok := s.Conn("nope"); ok {
		t.Error("unknown id found")
	}

	ctx := context.Background()
	// Client to server.
	for i := 0; i < 3; i++ {
		if err := c.Write(ctx, websocket.MessageText, fmt.Appendf(nil, "m%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 3; i++ {
		if got := recv(t, rec.frames); got != fmt.Sprintf("CS001:m%d", i) {
			t.Errorf("frame %d: %q", i, got)
		}
	}
	// Server to client.
	if err := conn.Write(ctx, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	typ, data, err := c.Read(ctx)
	if err != nil || typ != websocket.MessageText || string(data) != "hello" {
		t.Fatalf("%v %q %v", typ, data, err)
	}

	// Normal close from the client.
	if err := c.Close(websocket.StatusNormalClosure, ""); err != nil {
		t.Fatal(err)
	}
	d := recv(t, rec.disconnected)
	if d.c != conn || d.err != nil {
		t.Errorf("disconnect: %v", d.err)
	}
	<-conn.Done()
	if _, ok := s.Conn("CS001"); ok {
		t.Error("still registered")
	}
	if err := conn.Write(ctx, []byte("x")); !errors.Is(err, ocpp.ErrNotConnected) {
		t.Errorf("write after close: %v", err)
	}
	if conn.Close() == nil {
		t.Log("close of a finished connection is a no-op")
	}
}

func TestSubprotocolNegotiation(t *testing.T) {
	tests := []struct {
		name     string
		versions []ocpp.Version
		offer    []string
		want     string
	}{
		{"server preference wins", []ocpp.Version{ocpp.V201, ocpp.V16}, []string{"ocpp1.6", "ocpp2.0.1"}, "ocpp2.0.1"},
		{"server preference 1.6 first", []ocpp.Version{ocpp.V16, ocpp.V201}, []string{"ocpp2.0.1", "ocpp1.6"}, "ocpp1.6"},
		{"server filters", []ocpp.Version{ocpp.V16}, []string{"ocpp2.0.1", "ocpp1.6"}, "ocpp1.6"},
		{"1.6 only client", []ocpp.Version{ocpp.V201, ocpp.V16}, []string{"ocpp1.6"}, "ocpp1.6"},
	}
	for _, tt := range tests {
		_, rec, base := startServer(t, func(c *ServerConfig) { c.Versions = tt.versions })
		c := mustDial(t, base, "CS1", tt.offer...)
		if c.Subprotocol() != tt.want {
			t.Errorf("%s: got %q want %q", tt.name, c.Subprotocol(), tt.want)
		}
		if conn := recv(t, rec.connected); string(conn.Version()) != tt.want {
			t.Errorf("%s: conn version %q", tt.name, conn.Version())
		}
	}
}

func TestSubprotocolMismatchClosesAfterHandshake(t *testing.T) {
	_, rec, base := startServer(t, func(c *ServerConfig) { c.Versions = []ocpp.Version{ocpp.V201} })
	for _, offer := range [][]string{{"ocpp1.6"}, {"mqtt"}, nil} {
		c, _, err := rawDial(t, base, "CS1", nil, offer...)
		if err != nil {
			t.Fatalf("handshake must complete: %v", err)
		}
		if c.Subprotocol() != "" {
			t.Errorf("unexpected subprotocol %q", c.Subprotocol())
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, _, err = c.Read(ctx)
		cancel()
		if websocket.CloseStatus(err) != websocket.StatusProtocolError {
			t.Errorf("offer %v: close status %d (%v)", offer, websocket.CloseStatus(err), err)
		}
	}
	select {
	case <-rec.connected:
		t.Error("OnConnect must not fire without a subprotocol")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestInvalidIdentities(t *testing.T) {
	_, _, base := startServer(t, nil)
	long := make([]byte, 49)
	for i := range long {
		long[i] = 'a'
	}
	for name, id := range map[string]string{
		"empty":    "",
		"colon":    "a%3Ab",
		"slash":    "a%2Fb",
		"too long": string(long),
	} {
		_, resp, err := rawDial(t, base, id, nil, "ocpp2.0.1")
		if err == nil || resp == nil || resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: got %v %v", name, resp, err)
		}
	}
	// Percent-encoded identities are decoded.
	_, rec, base2 := startServer(t, nil)
	mustDial(t, base2, "RDAM%7C123", "ocpp2.0.1")
	if c := recv(t, rec.connected); c.ID() != "RDAM|123" {
		t.Errorf("id %q", c.ID())
	}
	// Exactly 48 characters is fine.
	mustDial(t, base2, string(long[:48]), "ocpp2.0.1")
	recv(t, rec.connected)
}

func TestValidateID(t *testing.T) {
	for _, id := range []string{"CS001", "a-b_c.d", "x"} {
		if err := validateID(id); err != nil {
			t.Errorf("%q: %v", id, err)
		}
	}
	if err := validateID("a:b"); !errors.Is(err, ErrInvalidID) {
		t.Error("colon should be invalid")
	}
	if _, err := idFromPath("/a/b/%zz"); !errors.Is(err, ErrInvalidID) {
		t.Error("bad escape should be invalid")
	}
}

func TestDuplicateConnectionReplacesOld(t *testing.T) {
	s, rec, base := startServer(t, nil)
	old := mustDial(t, base, "CS1", "ocpp2.0.1")
	oldConn := recv(t, rec.connected)
	oldRead := make(chan error, 1)
	go func() { _, _, err := old.Read(context.Background()); oldRead <- err }()

	nw := mustDial(t, base, "CS1", "ocpp2.0.1")
	newConn := recv(t, rec.connected)

	d := recv(t, rec.disconnected)
	if d.c != oldConn || !errors.Is(d.err, ErrReplaced) {
		t.Errorf("old connection: %v", d.err)
	}
	// The old client sees a policy-violation close.
	if err := recv(t, oldRead); websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
		t.Errorf("old client: %v", err)
	}
	ctx := context.Background()
	if got, _ := s.Conn("CS1"); got != newConn {
		t.Error("registry should hold the newest connection")
	}
	// The new connection still works, and the old one leaving did not unregister it.
	if err := nw.Write(ctx, websocket.MessageText, []byte("hi")); err != nil {
		t.Fatal(err)
	}
	if got := recv(t, rec.frames); got != "CS1:hi" {
		t.Errorf("got %q", got)
	}
	if ids := s.IDs(); len(ids) != 1 || ids[0] != "CS1" {
		t.Errorf("ids %v", ids)
	}
}

func TestBinaryFrameRejected(t *testing.T) {
	_, rec, base := startServer(t, nil)
	c := mustDial(t, base, "CS1", "ocpp2.0.1")
	recv(t, rec.connected)
	go func() { _, _, _ = c.Read(context.Background()) }() // lets the close handshake complete
	if err := c.Write(context.Background(), websocket.MessageBinary, []byte{1, 2}); err != nil {
		t.Fatal(err)
	}
	d := recv(t, rec.disconnected)
	if !errors.Is(d.err, ErrBinaryFrame) {
		t.Errorf("got %v", d.err)
	}
}

func TestReadLimit(t *testing.T) {
	_, rec, base := startServer(t, func(c *ServerConfig) { c.ReadLimit = 64 })
	c := mustDial(t, base, "CS1", "ocpp2.0.1")
	recv(t, rec.connected)
	_ = c.Write(context.Background(), websocket.MessageText, make([]byte, 200))
	d := recv(t, rec.disconnected)
	if d.err == nil || !strings.Contains(d.err.Error(), "too big") {
		t.Errorf("got %v", d.err)
	}
}

func TestAbruptDisconnectReportsError(t *testing.T) {
	_, rec, base := startServer(t, nil)
	c := mustDial(t, base, "CS1", "ocpp2.0.1")
	recv(t, rec.connected)
	_ = c.CloseNow()
	if d := recv(t, rec.disconnected); d.err == nil {
		t.Error("abrupt close should carry an error")
	}
}

func TestPingTimeoutClosesDeadPeer(t *testing.T) {
	_, rec, base := startServer(t, func(c *ServerConfig) {
		c.PingInterval = 20 * time.Millisecond
		c.PongTimeout = 40 * time.Millisecond
	})
	// This client never reads, so it never answers pings.
	mustDial(t, base, "CS1", "ocpp2.0.1")
	recv(t, rec.connected)
	if d := recv(t, rec.disconnected); !errors.Is(d.err, ErrPingTimeout) {
		t.Errorf("got %v", d.err)
	}
}

func TestPingKeepsHealthyConnectionAlive(t *testing.T) {
	_, rec, base := startServer(t, func(c *ServerConfig) {
		c.PingInterval = 15 * time.Millisecond
		c.PongTimeout = time.Second
	})
	c := mustDial(t, base, "CS1", "ocpp2.0.1")
	conn := recv(t, rec.connected)
	// A reading client answers pings automatically.
	go func() { _, _, _ = c.Read(context.Background()) }()
	select {
	case <-conn.Done():
		t.Fatalf("healthy connection dropped: %v", conn.Err())
	case <-time.After(200 * time.Millisecond):
	}
}

func TestPingDisabled(t *testing.T) {
	s, _, _ := startServer(t, func(c *ServerConfig) { c.PingInterval = -1 })
	if s.opts.pingInterval >= 0 {
		t.Errorf("ping should stay disabled, got %v", s.opts.pingInterval)
	}
}

func TestAuthenticatorStatusCodes(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{ErrUnauthorized, 401},
		{errors.New("something else"), 401},
		{ErrUnknownID, 404},
		{fmt.Errorf("wrapped: %w", ErrForbidden), 403},
	}
	for _, tt := range tests {
		_, _, base := startServer(t, func(c *ServerConfig) {
			c.Authenticator = func(*http.Request, string) error { return tt.err }
		})
		_, resp, err := rawDial(t, base, "CS1", nil, "ocpp2.0.1")
		if err == nil || resp.StatusCode != tt.want {
			t.Errorf("%v: status %v err %v", tt.err, resp, err)
		}
		if tt.want == 401 && resp.Header.Get("WWW-Authenticate") == "" {
			t.Error("missing WWW-Authenticate")
		}
	}
}

func TestConcurrentConnections(t *testing.T) {
	const n = 100
	s, rec, base := startServer(t, nil)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := fmt.Sprintf("CS%03d", i)
			c, _, err := rawDial(t, base, id, nil, "ocpp2.0.1")
			if err != nil {
				t.Errorf("%s: %v", id, err)
				return
			}
			for j := 0; j < 5; j++ {
				if err := c.Write(context.Background(), websocket.MessageText, []byte(fmt.Sprint(j))); err != nil {
					t.Errorf("%s: %v", id, err)
				}
			}
		}()
	}
	wg.Wait()
	for i := 0; i < n*5; i++ {
		recv(t, rec.frames)
	}
	if got := len(s.IDs()); got != n {
		t.Errorf("%d connected, want %d", got, n)
	}
}

func TestServeAndGracefulShutdown(t *testing.T) {
	rec := newRecorder()
	cfg := ServerConfig{Versions: []ocpp.Version{ocpp.V201}}
	rec.apply(&cfg)
	s, err := NewServer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	serveErr := make(chan error, 1)
	ln := mustListen(t)
	go func() { serveErr <- s.Serve(ln) }()
	base := "ws://" + ln.Addr().String()

	var clients []*websocket.Conn
	for i := 0; i < 3; i++ {
		clients = append(clients, mustDial(t, base, fmt.Sprintf("CS%d", i), "ocpp2.0.1"))
		recv(t, rec.connected)
	}
	// Clients must read to see the close frame.
	closeCodes := make(chan websocket.StatusCode, 3)
	for _, c := range clients {
		go func() {
			_, _, err := c.Read(context.Background())
			closeCodes <- websocket.CloseStatus(err)
		}()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		t.Errorf("Serve returned %v", err)
	}
	for i := 0; i < 3; i++ {
		if code := recv(t, closeCodes); code != websocket.StatusGoingAway {
			t.Errorf("close code %d, want 1001", code)
		}
		if d := recv(t, rec.disconnected); d.err != nil {
			t.Errorf("shutdown disconnect error: %v", d.err)
		}
	}
	if len(s.IDs()) != 0 {
		t.Error("connections remain")
	}
	// After shutdown new handshakes and Serve calls are refused.
	if err := s.Serve(mustListen(t)); !errors.Is(err, http.ErrServerClosed) {
		t.Errorf("Serve after shutdown: %v", err)
	}
}

func TestShutdownRefusesNewHandshakes(t *testing.T) {
	s, _, base := startServer(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	_, resp, err := rawDial(t, base, "CS1", nil, "ocpp2.0.1")
	if err == nil || resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("got %v %v", resp, err)
	}
}

func TestShutdownTimeoutCutsConnections(t *testing.T) {
	s, rec, base := startServer(t, nil)
	// This client never reads, so it never acknowledges the close frame.
	mustDial(t, base, "CS1", "ocpp2.0.1")
	recv(t, rec.connected)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := s.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got %v", err)
	}
	recv(t, rec.disconnected)
}
