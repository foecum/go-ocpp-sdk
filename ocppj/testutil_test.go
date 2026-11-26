package ocppj

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

// pingReq / pingResp are the test feature "Ping".
type pingReq struct {
	Message string `json:"message"`
	Count   int    `json:"count,omitempty"`
}

func (pingReq) Action() string { return "Ping" }
func (r pingReq) Validate() error {
	if r.Message == "" {
		return ocpp.Required("message")
	}
	if len(r.Message) > 10 {
		return ocpp.Invalid("message", "too long")
	}
	return nil
}

type pingResp struct {
	Reply string `json:"reply"`
}

func (pingResp) Action() string { return "Ping" }
func (r pingResp) Validate() error {
	if r.Reply == "bad" {
		return ocpp.Invalid("reply", "bad value")
	}
	return nil
}

// unencodable has a field encoding/json cannot marshal.
type unencodableReq struct {
	F func() `json:"f"`
}

func (unencodableReq) Action() string  { return "Unencodable" }
func (unencodableReq) Validate() error { return nil }

type unencodableResp struct {
	F func() `json:"f"`
}

func (unencodableResp) Action() string  { return "Unencodable" }
func (unencodableResp) Validate() error { return nil }

func newRegistry(t testing.TB, v ocpp.Version) *Registry {
	t.Helper()
	r := NewRegistry(v)
	for _, p := range [][2]any{
		{pingReq{}, pingResp{}},
		{unencodableReq{}, unencodableResp{}},
	} {
		f, err := ocpp.NewFeature(p[0].(ocpp.Request), p[1].(ocpp.Response))
		if err != nil {
			t.Fatal(err)
		}
		if err := r.Register(f); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

// rig is an Endpoint whose sent frames land in a channel.
type rig struct {
	e       *Endpoint
	frames  chan []byte
	mu      sync.Mutex
	sendErr error
}

func newRig(t testing.TB, mutate func(*Config)) *rig {
	t.Helper()
	r := &rig{frames: make(chan []byte, 100)}
	cfg := Config{
		Registry: newRegistry(t, ocpp.V201),
		Send: func(_ context.Context, frame []byte) error {
			r.mu.Lock()
			err := r.sendErr
			r.mu.Unlock()
			if err != nil {
				return err
			}
			r.frames <- frame
			return nil
		},
	}
	if mutate != nil {
		mutate(&cfg)
	}
	e, err := NewEndpoint(cfg)
	if err != nil {
		t.Fatal(err)
	}
	r.e = e
	t.Cleanup(e.Close)
	return r
}

func (r *rig) setSendErr(err error) {
	r.mu.Lock()
	r.sendErr = err
	r.mu.Unlock()
}

// next waits for the next sent frame.
func (r *rig) next(t testing.TB) Message {
	t.Helper()
	select {
	case f := <-r.frames:
		m, err := Parse(ocpp.V201, f)
		if err != nil {
			t.Fatalf("sent invalid frame %s: %v", f, err)
		}
		return m
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a sent frame")
		return nil
	}
}

func (r *rig) expectNone(t testing.TB) {
	t.Helper()
	select {
	case f := <-r.frames:
		t.Fatalf("unexpected frame %s", f)
	case <-time.After(30 * time.Millisecond):
	}
}

func (r *rig) respond(id, payload string) {
	r.e.HandleFrame([]byte(fmt.Sprintf(`[3,%q,%s]`, id, payload)))
}

type result struct {
	resp ocpp.Response
	err  error
}

func async(f func() (ocpp.Response, error)) <-chan result {
	ch := make(chan result, 1)
	go func() {
		resp, err := f()
		ch <- result{resp, err}
	}()
	return ch
}

func get(t testing.TB, ch <-chan result) result {
	t.Helper()
	select {
	case r := <-ch:
		return r
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for call to return")
		return result{}
	}
}

func (e *Endpoint) queueLen() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.queue)
}

func waitQueue(t testing.TB, e *Endpoint, n int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for e.queueLen() != n {
		if time.Now().After(deadline) {
			t.Fatalf("queue length stuck at %d, want %d", e.queueLen(), n)
		}
		time.Sleep(time.Millisecond)
	}
}

var errBoom = errors.New("boom")

// goid returns the current goroutine's id, for tests only.
func goid() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	var id int64
	fmt.Sscanf(string(buf[:n]), "goroutine %d ", &id)
	return id
}
