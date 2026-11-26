package ocppj

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

func TestNewEndpointConfig(t *testing.T) {
	if _, err := NewEndpoint(Config{Send: func(context.Context, []byte) error { return nil }}); err == nil {
		t.Error("missing registry should fail")
	}
	if _, err := NewEndpoint(Config{Registry: NewRegistry(ocpp.V16)}); err == nil {
		t.Error("missing send should fail")
	}
	r := newRig(t, nil)
	if r.e.timeout != DefaultTimeout || r.e.maxQueue != DefaultQueueSize || r.e.ids == nil || r.e.log == nil {
		t.Error("defaults not applied")
	}
	r = newRig(t, func(c *Config) {
		c.Timeout, c.QueueSize, c.Logger = time.Second, 3, slog.Default()
		c.IDGenerator = IDGeneratorFunc(func() string { return "x" })
	})
	if r.e.timeout != time.Second || r.e.maxQueue != 3 || r.e.log != slog.Default() {
		t.Error("explicit config ignored")
	}
}

func TestCallSuccess(t *testing.T) {
	r := newRig(t, nil)
	ch := async(func() (ocpp.Response, error) { return r.e.Call(context.Background(), pingReq{Message: "hi"}) })
	m := r.next(t)
	c, ok := m.(*Call)
	if !ok || c.Action != "Ping" || string(c.Payload) != `{"message":"hi"}` || len(c.UniqueID) != 36 {
		t.Fatalf("unexpected frame %#v", m)
	}
	r.respond(c.UniqueID, `{"reply":"pong"}`)
	res := get(t, ch)
	if res.err != nil || res.resp.(*pingResp).Reply != "pong" {
		t.Fatalf("%v %v", res.resp, res.err)
	}
}

func TestCallRejectsBadRequests(t *testing.T) {
	r := newRig(t, nil)
	ctx := context.Background()
	if _, err := r.e.Call(ctx, nil); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("nil: %v", err)
	}
	if _, err := r.e.Call(ctx, pingReq{}); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("invalid: %v", err)
	}
	if _, err := r.e.Call(ctx, otherReq{}); !errors.Is(err, ocpp.ErrUnknownAction) {
		t.Errorf("unregistered: %v", err)
	}
	if _, err := r.e.Call(ctx, unencodableReq{F: func() {}}); err == nil {
		t.Error("unencodable payload should fail")
	}
	rr := newRig(t, func(c *Config) {
		c.IDGenerator = IDGeneratorFunc(func() string { return strings.Repeat("x", 40) })
	})
	if _, err := rr.e.Call(ctx, pingReq{Message: "hi"}); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("long id: %v", err)
	}
	r.expectNone(t)
}

type otherReq struct{}

func (otherReq) Action() string  { return "Other" }
func (otherReq) Validate() error { return nil }

func TestCallErrorResponse(t *testing.T) {
	r := newRig(t, nil)
	ch := async(func() (ocpp.Response, error) { return r.e.Call(context.Background(), pingReq{Message: "hi"}) })
	id := r.next(t).ID()
	r.e.HandleFrame(fmt.Appendf(nil, `[4,%q,"NotSupported","no",{"why":1}]`, id))
	res := get(t, ch)
	var ce *ocpp.CallError
	if !errors.Is(res.err, ocpp.ErrCallError) || !errors.As(res.err, &ce) ||
		ce.Code != ocpp.NotSupported || ce.Description != "no" || string(ce.Details) != `{"why":1}` {
		t.Fatalf("got %v", res.err)
	}
}

func TestCallInvalidResponsePayload(t *testing.T) {
	r := newRig(t, nil)
	ch := async(func() (ocpp.Response, error) { return r.e.Call(context.Background(), pingReq{Message: "hi"}) })
	r.respond(r.next(t).ID(), `{"reply":"bad"}`)
	res := get(t, ch)
	if !errors.Is(res.err, ocpp.ErrValidation) {
		t.Fatalf("got %v", res.err)
	}
	// The slot is free again.
	ch = async(func() (ocpp.Response, error) { return r.e.Call(context.Background(), pingReq{Message: "hi"}) })
	r.respond(r.next(t).ID(), `{"reply":"ok"}`)
	if res := get(t, ch); res.err != nil {
		t.Fatal(res.err)
	}
}

func TestOneOutstandingCallAndOrdering(t *testing.T) {
	r := newRig(t, nil)
	ctx := context.Background()
	var chans []<-chan result
	var ids []string
	for i := 0; i < 3; i++ {
		msg := fmt.Sprintf("m%d", i)
		chans = append(chans, async(func() (ocpp.Response, error) { return r.e.Call(ctx, pingReq{Message: msg}) }))
		// Wait until each call is either sent or queued, to fix the order.
		if i == 0 {
			ids = append(ids, r.next(t).ID())
		} else {
			waitQueue(t, r.e, i)
		}
	}
	r.expectNone(t) // only one CALL outstanding
	for i := 0; i < 3; i++ {
		r.respond(ids[i], fmt.Sprintf(`{"reply":"r%d"}`, i))
		if got := get(t, chans[i]); got.err != nil || got.resp.(*pingResp).Reply != fmt.Sprintf("r%d", i) {
			t.Fatalf("call %d: %v %v", i, got.resp, got.err)
		}
		if i < 2 {
			c := r.next(t).(*Call)
			if !strings.Contains(string(c.Payload), fmt.Sprintf("m%d", i+1)) {
				t.Fatalf("out of order: %s", c.Payload)
			}
			ids = append(ids, c.UniqueID)
		}
	}
}

func TestQueueFull(t *testing.T) {
	r := newRig(t, func(c *Config) { c.QueueSize = 1 })
	ctx := context.Background()
	c1 := async(func() (ocpp.Response, error) { return r.e.Call(ctx, pingReq{Message: "1"}) })
	id1 := r.next(t).ID()
	c2 := async(func() (ocpp.Response, error) { return r.e.Call(ctx, pingReq{Message: "2"}) })
	waitQueue(t, r.e, 1)
	if _, err := r.e.Call(ctx, pingReq{Message: "3"}); !errors.Is(err, ocpp.ErrQueueFull) {
		t.Fatalf("got %v", err)
	}
	r.respond(id1, `{"reply":"a"}`)
	get(t, c1)
	r.respond(r.next(t).ID(), `{"reply":"b"}`)
	if res := get(t, c2); res.err != nil {
		t.Fatal(res.err)
	}
}

func TestTimeoutThenNextCall(t *testing.T) {
	r := newRig(t, func(c *Config) { c.Timeout = 30 * time.Millisecond })
	ctx := context.Background()
	c1 := async(func() (ocpp.Response, error) { return r.e.Call(ctx, pingReq{Message: "1"}) })
	id1 := r.next(t).ID()
	c2 := async(func() (ocpp.Response, error) { return r.e.Call(ctx, pingReq{Message: "2"}) })
	waitQueue(t, r.e, 1)
	if res := get(t, c1); !errors.Is(res.err, ocpp.ErrTimeout) {
		t.Fatalf("got %v", res.err)
	}
	id2 := r.next(t).ID()              // queue advanced after the timeout
	r.respond(id1, `{"reply":"late"}`) // late response for the timed-out call is dropped
	r.respond(id2, `{"reply":"ok"}`)
	if res := get(t, c2); res.err != nil || res.resp.(*pingResp).Reply != "ok" {
		t.Fatalf("%v %v", res.resp, res.err)
	}
}

func TestContextCancellation(t *testing.T) {
	r := newRig(t, nil)
	// Cancelled while outstanding: the slot is released and the queue moves on.
	ctx1, cancel1 := context.WithCancel(context.Background())
	c1 := async(func() (ocpp.Response, error) { return r.e.Call(ctx1, pingReq{Message: "1"}) })
	r.next(t)
	c2 := async(func() (ocpp.Response, error) { return r.e.Call(context.Background(), pingReq{Message: "2"}) })
	waitQueue(t, r.e, 1)
	cancel1()
	if res := get(t, c1); !errors.Is(res.err, context.Canceled) {
		t.Fatalf("got %v", res.err)
	}
	id2 := r.next(t).ID()
	// Cancelled while queued: removed without ever being sent.
	ctx3, cancel3 := context.WithCancel(context.Background())
	c3 := async(func() (ocpp.Response, error) { return r.e.Call(ctx3, pingReq{Message: "3"}) })
	waitQueue(t, r.e, 1)
	cancel3()
	if res := get(t, c3); !errors.Is(res.err, context.Canceled) {
		t.Fatalf("got %v", res.err)
	}
	waitQueue(t, r.e, 0)
	r.respond(id2, `{"reply":"ok"}`)
	get(t, c2)
	r.expectNone(t)
	// Already cancelled before the call.
	ctx4, cancel4 := context.WithCancel(context.Background())
	cancel4()
	if _, err := r.e.Call(ctx4, pingReq{Message: "4"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	r.expectNone(t)
}

func TestSendFailure(t *testing.T) {
	r := newRig(t, nil)
	r.setSendErr(ocpp.ErrNotConnected)
	if _, err := r.e.Call(context.Background(), pingReq{Message: "1"}); !errors.Is(err, ocpp.ErrNotConnected) {
		t.Fatalf("got %v", err)
	}
	r.setSendErr(nil)
	ch := async(func() (ocpp.Response, error) { return r.e.Call(context.Background(), pingReq{Message: "2"}) })
	r.respond(r.next(t).ID(), `{"reply":"ok"}`)
	if res := get(t, ch); res.err != nil {
		t.Fatal(res.err)
	}
}

func TestResponseArrivesDuringSend(t *testing.T) {
	var e *Endpoint
	e, err := NewEndpoint(Config{
		Registry: newRegistry(t, ocpp.V201),
		Send: func(_ context.Context, frame []byte) error {
			m, _ := Parse(ocpp.V201, frame)
			e.HandleFrame(fmt.Appendf(nil, `[3,%q,{"reply":"fast"}]`, m.ID()))
			return nil
		},
		Timeout: 20 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	resp, err := e.Call(context.Background(), pingReq{Message: "hi"})
	if err != nil || resp.(*pingResp).Reply != "fast" {
		t.Fatalf("%v %v", resp, err)
	}
	time.Sleep(40 * time.Millisecond) // no stray timeout fires afterwards
}

func TestUnexpectedResponsesIgnored(t *testing.T) {
	r := newRig(t, nil)
	r.respond("nobody", `{}`)                                    // nothing outstanding
	r.e.HandleFrame([]byte(`[4,"nobody","GenericError","",{}]`)) // ditto
	ch := async(func() (ocpp.Response, error) { return r.e.Call(context.Background(), pingReq{Message: "hi"}) })
	id := r.next(t).ID()
	r.respond("wrong-id", `{"reply":"x"}`) // different ID is ignored
	r.respond(id, `{"reply":"ok"}`)
	if res := get(t, ch); res.err != nil {
		t.Fatal(res.err)
	}
}

func TestClose(t *testing.T) {
	r := newRig(t, nil)
	ctx := context.Background()
	c1 := async(func() (ocpp.Response, error) { return r.e.Call(ctx, pingReq{Message: "1"}) })
	r.next(t)
	c2 := async(func() (ocpp.Response, error) { return r.e.Call(ctx, pingReq{Message: "2"}) })
	waitQueue(t, r.e, 1)
	r.e.Close()
	r.e.Close() // idempotent
	for _, c := range []<-chan result{c1, c2} {
		if res := get(t, c); !errors.Is(res.err, ocpp.ErrClosed) {
			t.Errorf("got %v", res.err)
		}
	}
	if _, err := r.e.Call(ctx, pingReq{Message: "3"}); !errors.Is(err, ocpp.ErrClosed) {
		t.Errorf("after close: %v", err)
	}
	r.e.HandleFrame([]byte(`[2,"1","Ping",{"message":"hi"}]`)) // ignored
	r.expectNone(t)
}

func TestCloseWaitsForHandlers(t *testing.T) {
	started := make(chan struct{})
	var finished atomic.Bool
	r := newRig(t, func(c *Config) {
		c.Handler = func(ctx context.Context, _ ocpp.Request) (ocpp.Response, error) {
			close(started)
			<-ctx.Done() // Close cancels the handler context
			finished.Store(true)
			return nil, ctx.Err()
		}
	})
	r.e.HandleFrame([]byte(`[2,"1","Ping",{"message":"hi"}]`))
	<-started
	r.e.Close()
	if !finished.Load() {
		t.Error("Close returned before the handler did")
	}
	r.e.Close()
}

func TestServeIncomingCalls(t *testing.T) {
	long := strings.Repeat("é", 300)
	tests := []struct {
		name     string
		handler  RequestHandler
		frame    string
		wantCode ocpp.ErrorCode // empty: expect CALLRESULT
		wantDesc string
	}{
		{"ok", func(_ context.Context, req ocpp.Request) (ocpp.Response, error) {
			return &pingResp{Reply: strings.ToUpper(req.(*pingReq).Message)}, nil
		}, `[2,"c1","Ping",{"message":"hi"}]`, "", ""},
		{"no handler", nil, `[2,"c1","Ping",{"message":"hi"}]`, ocpp.NotImplemented, ""},
		{"unknown action", okHandler, `[2,"c1","Nope",{}]`, ocpp.NotImplemented, ""},
		{"bad payload type", okHandler, `[2,"c1","Ping",{"message":5}]`, ocpp.TypeConstraintViolation, ""},
		{"missing field", okHandler, `[2,"c1","Ping",{}]`, ocpp.OccurrenceConstraintViolation, ""},
		{"call error from handler", func(context.Context, ocpp.Request) (ocpp.Response, error) {
			return nil, &ocpp.CallError{Code: ocpp.SecurityError, Description: "denied", Details: json.RawMessage(`{"a":1}`)}
		}, `[2,"c1","Ping",{"message":"hi"}]`, ocpp.SecurityError, "denied"},
		{"long description truncated", func(context.Context, ocpp.Request) (ocpp.Response, error) {
			return nil, &ocpp.CallError{Code: ocpp.GenericError, Description: long}
		}, `[2,"c1","Ping",{"message":"hi"}]`, ocpp.GenericError, strings.Repeat("é", 255)},
		{"plain error", func(context.Context, ocpp.Request) (ocpp.Response, error) {
			return nil, errBoom
		}, `[2,"c1","Ping",{"message":"hi"}]`, ocpp.InternalError, ""},
		{"nil response", func(context.Context, ocpp.Request) (ocpp.Response, error) {
			return nil, nil
		}, `[2,"c1","Ping",{"message":"hi"}]`, ocpp.InternalError, "handler returned no response"},
		{"invalid response", func(context.Context, ocpp.Request) (ocpp.Response, error) {
			return &pingResp{Reply: "bad"}, nil
		}, `[2,"c1","Ping",{"message":"hi"}]`, ocpp.InternalError, "invalid response"},
		{"unencodable response", func(context.Context, ocpp.Request) (ocpp.Response, error) {
			return &unencodableResp{F: func() {}}, nil
		}, `[2,"c1","Unencodable",{}]`, ocpp.InternalError, "cannot encode response"},
		{"null payload", okHandler, `[2,"c1","Ping",null]`, ocpp.OccurrenceConstraintViolation, ""},
	}
	for _, tt := range tests {
		r := newRig(t, func(c *Config) { c.Handler = tt.handler })
		r.e.HandleFrame([]byte(tt.frame))
		m := r.next(t)
		if m.ID() != "c1" {
			t.Errorf("%s: reply ID %q", tt.name, m.ID())
		}
		if tt.wantCode == "" {
			if res, ok := m.(*CallResult); !ok || string(res.Payload) != `{"reply":"HI"}` {
				t.Errorf("%s: got %#v", tt.name, m)
			}
			continue
		}
		ce, ok := m.(*CallErrorFrame)
		if !ok || ce.Code != tt.wantCode || (tt.wantDesc != "" && ce.Description != tt.wantDesc) {
			t.Errorf("%s: got %#v", tt.name, m)
		}
		if tt.name == "call error from handler" && string(ce.Details) != `{"a":1}` {
			t.Errorf("details lost: %s", ce.Details)
		}
	}
}

func okHandler(_ context.Context, req ocpp.Request) (ocpp.Response, error) {
	return &pingResp{Reply: "ok"}, nil
}

func TestInvalidIncomingFrames(t *testing.T) {
	tests := []struct {
		name     string
		v        ocpp.Version
		frame    string
		wantID   string
		wantCode ocpp.ErrorCode // empty: no reply
	}{
		{"garbage 2.0.1", ocpp.V201, `garbage`, "-1", ocpp.RpcFrameworkError},
		{"garbage 1.6", ocpp.V16, `garbage`, "-1", ocpp.FormationViolation},
		{"short call", ocpp.V201, `[2,"9","Ping"]`, "9", ocpp.ProtocolError},
		{"unknown type 2.0.1", ocpp.V201, `[7,"9","Ping",{}]`, "9", ocpp.MessageTypeNotSupported},
		{"unknown type 1.6", ocpp.V16, `[7,"9","Ping",{}]`, "", ""},
		{"bad call result", ocpp.V201, `[3,"9"]`, "", ""},
		{"bad call error", ocpp.V201, `[4,"9"]`, "", ""},
	}
	for _, tt := range tests {
		var sent []Message
		var mu sync.Mutex
		e, err := NewEndpoint(Config{
			Registry: newRegistry(t, tt.v),
			Send: func(_ context.Context, f []byte) error {
				m, perr := Parse(tt.v, f)
				if perr != nil {
					t.Errorf("%s: sent invalid frame %s", tt.name, f)
				}
				mu.Lock()
				sent = append(sent, m)
				mu.Unlock()
				return nil
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		e.HandleFrame([]byte(tt.frame))
		e.Close()
		mu.Lock()
		if tt.wantCode == "" {
			if len(sent) != 0 {
				t.Errorf("%s: unexpected reply %#v", tt.name, sent)
			}
		} else if len(sent) != 1 || sent[0].ID() != tt.wantID || sent[0].(*CallErrorFrame).Code != tt.wantCode {
			t.Errorf("%s: got %#v", tt.name, sent)
		}
		mu.Unlock()
	}
}

func TestReplyFailuresAreLogged(t *testing.T) {
	// Send fails: the reply is dropped without panicking.
	r := newRig(t, func(c *Config) { c.Handler = okHandler })
	r.setSendErr(ocpp.ErrNotConnected)
	r.e.HandleFrame([]byte(`[2,"c1","Ping",{"message":"hi"}]`))
	r.e.Close()
	r.expectNone(t)

	// Unmarshalable reply: a handler CallErrorFrame without a code.
	r = newRig(t, func(c *Config) {
		c.Handler = func(context.Context, ocpp.Request) (ocpp.Response, error) {
			return nil, &ocpp.CallError{Description: "no code"}
		}
	})
	r.e.HandleFrame([]byte(`[2,"c1","Ping",{"message":"hi"}]`))
	r.e.Close()
	r.expectNone(t)
}

// link connects two endpoints back to back.
func link(t *testing.T, ha, hb RequestHandler) (a, b *Endpoint) {
	t.Helper()
	var err error
	a, err = NewEndpoint(Config{Registry: newRegistry(t, ocpp.V201), Handler: ha,
		Send: func(_ context.Context, f []byte) error { b.HandleFrame(f); return nil }})
	if err != nil {
		t.Fatal(err)
	}
	b, err = NewEndpoint(Config{Registry: newRegistry(t, ocpp.V201), Handler: hb,
		Send: func(_ context.Context, f []byte) error { a.HandleFrame(f); return nil }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close(); b.Close() })
	return a, b
}

func TestEndToEndBothDirections(t *testing.T) {
	var a, b *Endpoint
	a, b = link(t,
		func(context.Context, ocpp.Request) (ocpp.Response, error) { return &pingResp{Reply: "from-a"}, nil },
		func(ctx context.Context, req ocpp.Request) (ocpp.Response, error) {
			// The handler calls back into the peer while handling a call.
			inner, err := b.Call(ctx, pingReq{Message: "inner"})
			if err != nil {
				return nil, err
			}
			return &pingResp{Reply: "b+" + inner.(*pingResp).Reply}, nil
		})
	resp, err := a.Call(context.Background(), pingReq{Message: "outer"})
	if err != nil || resp.(*pingResp).Reply != "b+from-a" {
		t.Fatalf("%v %v", resp, err)
	}
}

func TestConcurrentCalls(t *testing.T) {
	var inFlight, maxInFlight atomic.Int32
	a, _ := link(t, nil, func(_ context.Context, req ocpp.Request) (ocpp.Response, error) {
		n := inFlight.Add(1)
		defer inFlight.Add(-1)
		for {
			m := maxInFlight.Load()
			if n <= m || maxInFlight.CompareAndSwap(m, n) {
				break
			}
		}
		time.Sleep(time.Millisecond)
		return &pingResp{Reply: req.(*pingReq).Message}, nil
	})
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg := fmt.Sprintf("m%d", i)
			resp, err := a.Call(context.Background(), pingReq{Message: msg})
			if err != nil || resp.(*pingResp).Reply != msg {
				t.Errorf("call %d: %v %v", i, resp, err)
			}
		}()
	}
	wg.Wait()
	if maxInFlight.Load() != 1 {
		t.Errorf("handler saw %d concurrent calls, want 1", maxInFlight.Load())
	}
}
