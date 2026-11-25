package ws

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

var fastReconnect = Reconnect{WaitMinimum: 5 * time.Millisecond, RandomRange: 5 * time.Millisecond, RepeatTimes: 2}

// clientRec collects client callbacks.
type clientRec struct {
	frames       chan string
	connected    chan *Conn
	disconnected chan disconnect
}

func newClientRec() *clientRec {
	return &clientRec{frames: make(chan string, 100), connected: make(chan *Conn, 100), disconnected: make(chan disconnect, 100)}
}

func (r *clientRec) apply(cfg *ClientConfig) {
	cfg.OnConnect = func(c *Conn) { r.connected <- c }
	cfg.OnFrame = func(_ *Conn, f []byte) { r.frames <- string(f) }
	cfg.OnDisconnect = func(c *Conn, err error) { r.disconnected <- disconnect{c, err} }
}

// startClient builds a Client against base and runs it. The returned channel
// yields Run's result.
func startClient(t testing.TB, base string, mutate func(*ClientConfig)) (*Client, *clientRec, <-chan error) {
	t.Helper()
	rec := newClientRec()
	cfg := ClientConfig{URL: base + "/ocpp", ID: "CS1", Versions: []ocpp.Version{ocpp.V201}, Reconnect: fastReconnect}
	rec.apply(&cfg)
	if mutate != nil {
		mutate(&cfg)
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- c.Run(context.Background()) }()
	t.Cleanup(func() { _ = c.Close() })
	return c, rec, done
}

func TestNewClientValidation(t *testing.T) {
	ok := func(*Conn, []byte) {}
	base := ClientConfig{URL: "ws://localhost/ocpp", ID: "CS1", Versions: []ocpp.Version{ocpp.V16}, OnFrame: ok}
	mutations := map[string]func(*ClientConfig){
		"no OnFrame":  func(c *ClientConfig) { c.OnFrame = nil },
		"empty id":    func(c *ClientConfig) { c.ID = "" },
		"colon in id": func(c *ClientConfig) { c.ID = "a:b" },
		"no versions": func(c *ClientConfig) { c.Versions = nil },
		"bad version": func(c *ClientConfig) { c.Versions = []ocpp.Version{"x"} },
		"unparsable":  func(c *ClientConfig) { c.URL = "ws://a b/" },
		"http scheme": func(c *ClientConfig) { c.URL = "http://localhost/ocpp" },
		"no host":     func(c *ClientConfig) { c.URL = "ws:///ocpp" },
	}
	for name, m := range mutations {
		cfg := base
		m(&cfg)
		if _, err := NewClient(cfg); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	c, err := NewClient(base)
	if err != nil || c.cfg.HandshakeTimeout != DefaultHandshakeTimeout || c.header.Get("Authorization") != "" {
		t.Errorf("defaults: %v", err)
	}
}

func TestClientURLAndAuthHeader(t *testing.T) {
	cfg := func(u, id, pw string) ClientConfig {
		return ClientConfig{URL: u, ID: id, Password: pw, Versions: []ocpp.Version{ocpp.V16}, OnFrame: func(*Conn, []byte) {}}
	}
	tests := []struct{ url, id, want string }{
		{"ws://csms.example.com/ocpp", "CS001", "ws://csms.example.com/ocpp/CS001"},
		{"ws://csms.example.com/ocpp/", "CS001", "ws://csms.example.com/ocpp/CS001"},
		{"wss://csms.example.com", "CS001", "wss://csms.example.com/CS001"},
		{"wss://csms.example.com:8443/a/b", "RDAM|123", "wss://csms.example.com:8443/a/b/RDAM%7C123"},
	}
	for _, tt := range tests {
		c, err := NewClient(cfg(tt.url, tt.id, ""))
		if err != nil || c.url != tt.want {
			t.Errorf("%s + %s: got %q (%v), want %q", tt.url, tt.id, c.url, err, tt.want)
		}
	}
	c, _ := NewClient(cfg("ws://h/ocpp", "CS1", "secret-password-16"))
	if got := c.header.Get("Authorization"); got != "Basic Q1MxOnNlY3JldC1wYXNzd29yZC0xNg==" {
		t.Errorf("auth header %q", got)
	}
}

func TestClientConnectsAndExchangesFrames(t *testing.T) {
	_, srec, base := startServer(t, nil)
	c, rec, _ := startClient(t, base, func(cfg *ClientConfig) { cfg.Versions = []ocpp.Version{ocpp.V16, ocpp.V201} })
	if err := c.Send(context.Background(), []byte("early")); !errors.Is(err, ocpp.ErrNotConnected) && c.Conn() == nil {
		t.Errorf("send before connect: %v", err)
	}
	conn := recv(t, rec.connected)
	sconn := recv(t, srec.connected)
	// The client offers 1.6 first, but the server prefers 2.0.1 and its order wins.
	if conn.ID() != "CS1" || conn.Version() != ocpp.V201 || sconn.Version() != ocpp.V201 {
		t.Errorf("negotiated %q / %q", conn.Version(), sconn.Version())
	}
	if c.Conn() != conn {
		t.Error("Conn() mismatch")
	}
	ctx := context.Background()
	if err := c.Send(ctx, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	if got := recv(t, srec.frames); got != "CS1:hello" {
		t.Errorf("server got %q", got)
	}
	if err := sconn.Write(ctx, []byte("world")); err != nil {
		t.Fatal(err)
	}
	if got := recv(t, rec.frames); got != "world" {
		t.Errorf("client got %q", got)
	}
}

func TestClientBasicAuth(t *testing.T) {
	_, srec, base := startServer(t, func(c *ServerConfig) {
		c.Authenticator = BasicAuthenticator(func(id string) (string, bool) { return "correct-password-16", id == "CS1" })
	})
	_, rec, _ := startClient(t, base, func(c *ClientConfig) { c.Password = "wrong-password-0000" })
	select {
	case <-rec.connected:
		t.Fatal("wrong password must not connect")
	case <-time.After(100 * time.Millisecond):
	}
	_, rec2, _ := startClient(t, base, func(c *ClientConfig) { c.Password = "correct-password-16" })
	recv(t, rec2.connected)
	recv(t, srec.connected)
}

func TestClientKeepsRetryingOnSubprotocolMismatch(t *testing.T) {
	_, _, base := startServer(t, func(c *ServerConfig) { c.Versions = []ocpp.Version{ocpp.V201} })
	_, rec, done := startClient(t, base, func(c *ClientConfig) { c.Versions = []ocpp.Version{ocpp.V16} })
	select {
	case <-rec.connected:
		t.Fatal("must not connect without a common subprotocol")
	case err := <-done:
		t.Fatalf("Run returned: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestClientReconnectsAfterServerDropsConnection(t *testing.T) {
	s, srec, base := startServer(t, nil)
	_, rec, _ := startClient(t, base, nil)
	first := recv(t, rec.connected)
	sconn := recv(t, srec.connected)
	_ = sconn.closeWith(nil, websocket.StatusInternalError, "drop")
	d := recv(t, rec.disconnected)
	if d.c != first || d.err == nil {
		t.Errorf("disconnect: %v", d.err)
	}
	second := recv(t, rec.connected)
	if second == first {
		t.Error("expected a new connection")
	}
	recv(t, srec.connected)
	if _, ok := s.Conn("CS1"); !ok {
		t.Error("server should hold the new connection")
	}
}

func TestClientReconnectsAfterServerRestart(t *testing.T) {
	ln := mustListen(t)
	addr := ln.Addr().String()
	newSrv := func(l net.Listener) (*Server, *recorder) {
		rec := newRecorder()
		cfg := ServerConfig{Versions: []ocpp.Version{ocpp.V201}}
		rec.apply(&cfg)
		s, err := NewServer(cfg)
		if err != nil {
			t.Fatal(err)
		}
		go func() { _ = s.Serve(l) }()
		return s, rec
	}
	s1, srec1 := newSrv(ln)
	_, rec, _ := startClient(t, "ws://"+addr, nil)
	recv(t, rec.connected)
	recv(t, srec1.connected)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s1.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	// The client was told 1001 (going away): a clean close, then it retries.
	if d := recv(t, rec.disconnected); d.err != nil {
		t.Errorf("going-away close should not be an error: %v", d.err)
	}
	time.Sleep(60 * time.Millisecond) // a few failed attempts against the dead port

	ln2, err := net.Listen("tcp", addr)
	if err != nil {
		t.Skipf("port was taken: %v", err)
	}
	s2, srec2 := newSrv(ln2)
	t.Cleanup(func() { _ = s2.Shutdown(context.Background()) })
	recv(t, rec.connected)
	recv(t, srec2.connected)
}

func TestClientCloseIsGraceful(t *testing.T) {
	_, srec, base := startServer(t, nil)
	c, rec, done := startClient(t, base, nil)
	recv(t, rec.connected)
	recv(t, srec.connected)
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, ocpp.ErrClosed) {
		t.Errorf("Run returned %v", err)
	}
	if d := recv(t, srec.disconnected); d.err != nil {
		t.Errorf("server saw %v, want a normal close", d.err)
	}
	if d := recv(t, rec.disconnected); d.err != nil {
		t.Errorf("client saw %v", d.err)
	}
	if c.Conn() != nil {
		t.Error("Conn should be nil after Close")
	}
	if err := c.Send(context.Background(), []byte("x")); !errors.Is(err, ocpp.ErrNotConnected) {
		t.Errorf("send after close: %v", err)
	}
	_ = c.Close() // idempotent
	if err := c.Run(context.Background()); !errors.Is(err, ocpp.ErrClosed) {
		t.Errorf("Run after Close: %v", err)
	}
}

func TestClientCloseBeforeRunAndDoubleRun(t *testing.T) {
	cfg := ClientConfig{URL: "ws://127.0.0.1:1/ocpp", ID: "CS1", Versions: []ocpp.Version{ocpp.V16},
		OnFrame: func(*Conn, []byte) {}, Reconnect: fastReconnect}
	c, _ := NewClient(cfg)
	_ = c.Close()
	if err := c.Run(context.Background()); !errors.Is(err, ocpp.ErrClosed) {
		t.Errorf("got %v", err)
	}

	c, _ = NewClient(cfg)
	done := make(chan error, 1)
	go func() { done <- c.Run(context.Background()) }()
	deadline := time.Now().Add(2 * time.Second)
	for {
		c.mu.Lock()
		running := c.running
		c.mu.Unlock()
		if running {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("client did not start")
		}
		time.Sleep(time.Millisecond)
	}
	if err := c.Run(context.Background()); err == nil || errors.Is(err, ocpp.ErrClosed) {
		t.Errorf("second Run should be refused, got %v", err)
	}
	_ = c.Close()
	<-done
}

func TestClientStopsWhenContextEnds(t *testing.T) {
	_, srec, base := startServer(t, nil)
	rec := newClientRec()
	cfg := ClientConfig{URL: base + "/ocpp", ID: "CS1", Versions: []ocpp.Version{ocpp.V201}, Reconnect: fastReconnect}
	rec.apply(&cfg)
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()
	recv(t, rec.connected)
	recv(t, srec.connected)
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Errorf("got %v", err)
	}
	// Cancelling while waiting between attempts also stops Run.
	c2, _ := NewClient(ClientConfig{URL: "ws://127.0.0.1:1/ocpp", ID: "CS2", Versions: []ocpp.Version{ocpp.V16},
		OnFrame: func(*Conn, []byte) {}, Reconnect: Reconnect{WaitMinimum: time.Hour}})
	ctx2, cancel2 := context.WithCancel(context.Background())
	done2 := make(chan error, 1)
	go func() { done2 <- c2.Run(ctx2) }()
	time.Sleep(50 * time.Millisecond)
	cancel2()
	if err := <-done2; !errors.Is(err, context.Canceled) {
		t.Errorf("got %v", err)
	}
}

func TestClientDetectsDeadServerWithPing(t *testing.T) {
	// A server that accepts the handshake but never reads, so never answers pings.
	var accepted atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"ocpp2.0.1"}})
		if err != nil {
			return
		}
		accepted.Add(1)
		defer c.CloseNow()
		<-r.Context().Done()
	}))
	defer ts.Close()
	_, rec, _ := startClient(t, "ws"+strings.TrimPrefix(ts.URL, "http"), func(c *ClientConfig) {
		c.PingInterval, c.PongTimeout = 20*time.Millisecond, 40*time.Millisecond
	})
	recv(t, rec.connected)
	if d := recv(t, rec.disconnected); !errors.Is(d.err, ErrPingTimeout) {
		t.Errorf("got %v", d.err)
	}
	recv(t, rec.connected) // and it reconnects
	// The server's handler counts the connection in its own goroutine, which may
	// run a moment after the client reports it, so wait for it.
	deadline := time.Now().Add(3 * time.Second)
	for accepted.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if accepted.Load() < 2 {
		t.Errorf("accepted %d connections", accepted.Load())
	}
}

func TestClientOverTLSProfiles(t *testing.T) {
	ca := newTestCA(t, "CA")
	srvCert := ca.issue(t, "localhost", "CSMS", true)

	// Profile 2: TLS + Basic.
	srec, base := startTLSServer(t, func(c *ServerConfig) {
		c.Authenticator = BasicAuthenticator(func(id string) (string, bool) { return "correct-password-16", true })
	}, ServerTLSConfig(srvCert, nil))
	_, rec, _ := startClient(t, base, func(c *ClientConfig) {
		c.Password = "correct-password-16"
		c.TLSConfig = ClientTLSConfig(ca.pool, nil)
	})
	recv(t, rec.connected)
	recv(t, srec.connected)

	// Profile 3: client certificate.
	srec3, base3 := startTLSServer(t, func(c *ServerConfig) {
		c.Authenticator = ClientCertAuthenticator(ClientCertOptions{})
	}, ServerTLSConfig(srvCert, ca.pool))
	leaf := ca.issue(t, "CS1", "CSO", false)
	_, rec3, _ := startClient(t, base3, func(c *ClientConfig) { c.TLSConfig = ClientTLSConfig(ca.pool, &leaf) })
	recv(t, rec3.connected)
	recv(t, srec3.connected)

	// Untrusted server certificate: never connects.
	_, rec4, _ := startClient(t, base, func(c *ClientConfig) {
		c.Password = "correct-password-16"
		c.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	})
	select {
	case <-rec4.connected:
		t.Error("connected to an untrusted server")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestClientHandshakeTimeout(t *testing.T) {
	// A TCP server that never answers the HTTP upgrade.
	ln := mustListen(t)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			defer c.Close()
		}
	}()
	_, rec, _ := startClient(t, "ws://"+ln.Addr().String(), func(c *ClientConfig) { c.HandshakeTimeout = 50 * time.Millisecond })
	select {
	case <-rec.connected:
		t.Fatal("must not connect")
	case <-time.After(200 * time.Millisecond):
	}
}
