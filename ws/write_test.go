package ws

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

// A caller whose context has already ended sends nothing, and the connection
// stays up for everyone else.
func TestWriteWithAnEndedContextKeepsTheConnection(t *testing.T) {
	_, rec, base := startServer(t, nil)
	c := mustDial(t, base, "CS1", "ocpp2.0.1")
	conn := recv(t, rec.connected)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := conn.Write(ctx, []byte("lost")); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if err := conn.Write(context.Background(), []byte("kept")); err != nil {
		t.Fatalf("the connection was closed: %v", err)
	}
	_, data, err := c.Read(context.Background())
	if err != nil || string(data) != "kept" {
		t.Errorf("%q %v", data, err)
	}
}

// A write to a peer that stops reading is not cut short by the caller's
// context, which would close the connection under everyone else's feet, but by
// the write timeout, after which the connection is considered stuck.
func TestAStuckWriteIsBoundedByTheWriteTimeout(t *testing.T) {
	_, rec, base := startServer(t, func(cfg *ServerConfig) {
		cfg.WriteTimeout = 300 * time.Millisecond
		cfg.ReadLimit = 64 << 20
	})
	_ = mustDial(t, base, "CS1", "ocpp2.0.1") // never reads
	conn := recv(t, rec.connected)
	big := bytes.Repeat([]byte("x"), 4<<20)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	var err error
	for i := 0; i < 64 && err == nil; i++ { // fill the socket buffers until a write blocks
		err = conn.Write(ctx, big)
	}
	if err == nil {
		t.Skip("the socket buffers never filled")
	}
	if errors.Is(err, context.DeadlineExceeded) && time.Since(start) < 250*time.Millisecond {
		t.Fatalf("the write ended with the caller's context after %v", time.Since(start))
	}
	select {
	case <-conn.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("a write past the write timeout must close the stuck connection")
	}
}

func TestWriteTimeoutDefaults(t *testing.T) {
	if o := (connOptions{}).withDefaults(); o.writeTimeout != DefaultWriteTimeout {
		t.Errorf("%v", o.writeTimeout)
	}
	c, err := NewClient(ClientConfig{URL: "ws://localhost/ocpp", ID: "CS1", Versions: []ocpp.Version{ocpp.V16}, OnFrame: func(*Conn, []byte) {}, WriteTimeout: time.Second})
	if err != nil || c.opts.writeTimeout != time.Second {
		t.Errorf("%v %v", c, err)
	}
}
