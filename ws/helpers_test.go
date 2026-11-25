package ws

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

// recorder collects server callbacks.
type recorder struct {
	frames       chan string
	connected    chan *Conn
	disconnected chan disconnect
}

type disconnect struct {
	c   *Conn
	err error
}

func newRecorder() *recorder {
	return &recorder{frames: make(chan string, 100), connected: make(chan *Conn, 100), disconnected: make(chan disconnect, 100)}
}

func (r *recorder) apply(cfg *ServerConfig) {
	cfg.OnConnect = func(c *Conn) { r.connected <- c }
	cfg.OnFrame = func(c *Conn, f []byte) { r.frames <- c.ID() + ":" + string(f) }
	cfg.OnDisconnect = func(c *Conn, err error) { r.disconnected <- disconnect{c, err} }
}

func recv[T any](t testing.TB, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for event")
		panic("unreachable")
	}
}

// startServer runs a Server behind httptest and returns it with its ws:// base URL.
func startServer(t testing.TB, mutate func(*ServerConfig)) (*Server, *recorder, string) {
	t.Helper()
	rec := newRecorder()
	cfg := ServerConfig{Versions: []ocpp.Version{ocpp.V201, ocpp.V16}}
	rec.apply(&cfg)
	if mutate != nil {
		mutate(&cfg)
	}
	s, err := NewServer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
		ts.Close()
	})
	return s, rec, "ws" + strings.TrimPrefix(ts.URL, "http")
}

// rawDial connects with the coder library directly, as a charge point would.
func rawDial(t testing.TB, base, id string, header http.Header, protos ...string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c, resp, err := websocket.Dial(ctx, base+"/ocpp/"+id, &websocket.DialOptions{Subprotocols: protos, HTTPHeader: header})
	if c != nil {
		t.Cleanup(func() { _ = c.CloseNow() })
	}
	return c, resp, err
}

func mustDial(t testing.TB, base, id string, protos ...string) *websocket.Conn {
	t.Helper()
	c, _, err := rawDial(t, base, id, nil, protos...)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func mustListen(t testing.TB) net.Listener {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	return l
}
