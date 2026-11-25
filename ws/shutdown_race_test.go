package ws

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

// TestConnectionAcceptedDuringShutdownIsClosed reproduces the window between the
// handshake and the registration of a connection: a shutdown that starts there
// has already taken its snapshot of the open connections, so the new connection
// must close itself, or Shutdown would wait for it until its context ends.
func TestConnectionAcceptedDuringShutdownIsClosed(t *testing.T) {
	disconnected := make(chan error, 1)
	srv, err := NewServer(ServerConfig{
		Versions:     []ocpp.Version{ocpp.V201},
		OnFrame:      func(*Conn, []byte) {},
		OnDisconnect: func(_ *Conn, err error) { disconnected <- err },
	})
	if err != nil {
		t.Fatal(err)
	}
	srv.afterAccept = func() {
		srv.mu.Lock()
		srv.closing = true // Shutdown has started and taken its snapshot
		srv.mu.Unlock()
	}
	ts := httptest.NewServer(srv)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(ts.URL, "http")+"/ocpp/CS1", &websocket.DialOptions{Subprotocols: []string{"ocpp2.0.1"}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	if _, _, err := c.Read(ctx); websocket.CloseStatus(err) != websocket.StatusGoingAway {
		t.Errorf("client saw %v, want a 1001 close", err)
	}
	select {
	case err := <-disconnected:
		if err != nil {
			t.Errorf("a close initiated by the server should be a normal end, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the connection was never closed by the server")
	}
}
