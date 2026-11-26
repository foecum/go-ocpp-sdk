package ocppj

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

// streamMsg is the one-way test feature "Stream" (an unconfirmed SEND message).
type streamMsg struct {
	ID int `json:"id"`
}

func (streamMsg) Action() string { return "Stream" }
func (m streamMsg) Validate() error {
	if m.ID < 0 {
		return ocpp.Invalid("id", "negative")
	}
	return nil
}

type streamUnencodable struct {
	F func() `json:"f"`
}

func (streamUnencodable) Action() string  { return "StreamBad" }
func (streamUnencodable) Validate() error { return nil }

func newRegistry21(t testing.TB) *Registry {
	t.Helper()
	r := newRegistry(t, ocpp.V21)
	for _, req := range []ocpp.Request{streamMsg{}, streamUnencodable{}} {
		f, err := ocpp.NewSendFeature(req)
		if err != nil {
			t.Fatal(err)
		}
		if err := r.Register(f); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func rig21(t testing.TB, mutate func(*Config)) *rig {
	t.Helper()
	return newRig(t, func(c *Config) {
		c.Registry = newRegistry21(t)
		if mutate != nil {
			mutate(c)
		}
	})
}

func (r *rig) next21(t testing.TB) Message {
	t.Helper()
	select {
	case f := <-r.frames:
		m, err := Parse(ocpp.V21, f)
		if err != nil {
			t.Fatalf("sent invalid frame %s: %v", f, err)
		}
		return m
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a sent frame")
		return nil
	}
}

func TestParseSendAndCallResultError(t *testing.T) {
	m, err := Parse(ocpp.V21, []byte(`[6,"7","Stream",{"id":1}]`))
	if err != nil {
		t.Fatal(err)
	}
	s, ok := m.(*Send)
	if !ok || s.UniqueID != "7" || s.Action != "Stream" || string(s.Payload) != `{"id":1}` || s.Type() != SendType || s.ID() != "7" {
		t.Errorf("%#v", m)
	}
	m, err = Parse(ocpp.V21, []byte(`[6,"7","Stream",null]`))
	if err != nil || string(m.(*Send).Payload) != `{}` {
		t.Errorf("null payload: %v %v", m, err)
	}
	m, err = Parse(ocpp.V21, []byte(`[5,"9","FormatViolation","bad",{"a":1}]`))
	if err != nil {
		t.Fatal(err)
	}
	c, ok := m.(*CallResultErrorFrame)
	if !ok || c.UniqueID != "9" || c.Code != ocpp.FormatViolation || c.Description != "bad" || string(c.Details) != `{"a":1}` || c.Type() != CallResultErrorType || c.ID() != "9" {
		t.Errorf("%#v", m)
	}
}

func TestParseSendErrors(t *testing.T) {
	for name, tc := range map[string]struct {
		frame  string
		reply  bool
		isType MessageType
	}{
		"send with 3 elements":     {`[6,"1","Stream"]`, false, SendType},
		"send action not a string": {`[6,"1",5,{}]`, false, SendType},
		"send empty action":        {`[6,"1","",{}]`, false, SendType},
		"send payload not object":  {`[6,"1","Stream",[1]]`, false, SendType},
		"result error 4 elements":  {`[5,"1","FormatViolation","x"]`, false, CallResultErrorType},
		"result error no code":     {`[5,"1","","x",{}]`, false, CallResultErrorType},
		"result error bad details": {`[5,"1","FormatViolation","x",5]`, false, CallResultErrorType},
	} {
		_, err := Parse(ocpp.V21, []byte(tc.frame))
		var pe *ParseError
		if !errors.As(err, &pe) {
			t.Errorf("%s: got %v", name, err)
			continue
		}
		if pe.Type != tc.isType || pe.ID != "1" || pe.ShouldReply() != tc.reply {
			t.Errorf("%s: %+v reply=%v", name, pe, pe.ShouldReply())
		}
	}
}

func TestSendAndCallResultErrorAreUnknownBefore21(t *testing.T) {
	for _, v := range []ocpp.Version{ocpp.V16, ocpp.V201} {
		for _, frame := range []string{`[6,"1","Stream",{}]`, `[5,"1","FormatViolation","x",{}]`} {
			_, err := Parse(v, []byte(frame))
			var pe *ParseError
			if !errors.As(err, &pe) {
				t.Fatalf("%s %s: %v", v, frame, err)
			}
			if wantReply := v == ocpp.V201; pe.ShouldReply() != wantReply {
				t.Errorf("%s %s: reply=%v", v, frame, pe.ShouldReply())
			}
		}
	}
}

func TestMarshalSendAndCallResultError(t *testing.T) {
	got, err := Marshal(&Send{UniqueID: "1", Action: "Stream", Payload: json.RawMessage(`{"id":2}`)})
	if err != nil || string(got) != `[6,"1","Stream",{"id":2}]` {
		t.Errorf("%s %v", got, err)
	}
	got, err = Marshal(&Send{UniqueID: "1", Action: "Stream"})
	if err != nil || string(got) != `[6,"1","Stream",{}]` {
		t.Errorf("%s %v", got, err)
	}
	got, err = Marshal(&CallResultErrorFrame{UniqueID: "1", Code: ocpp.FormatViolation, Description: "bad"})
	if err != nil || string(got) != `[5,"1","FormatViolation","bad",{}]` {
		t.Errorf("%s %v", got, err)
	}
	for name, m := range map[string]Message{
		"send without id":       &Send{Action: "Stream"},
		"send without action":   &Send{UniqueID: "1"},
		"send long id":          &Send{UniqueID: strings.Repeat("x", 37), Action: "Stream"},
		"result error no id":    &CallResultErrorFrame{Code: ocpp.FormatViolation},
		"result error no code":  &CallResultErrorFrame{UniqueID: "1"},
		"result error long msg": &CallResultErrorFrame{UniqueID: "1", Code: ocpp.FormatViolation, Description: strings.Repeat("x", 256)},
	} {
		if _, err := Marshal(m); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestRegistryOneWayDecode(t *testing.T) {
	reg := newRegistry21(t)
	req, err := reg.DecodeRequest("Stream", json.RawMessage(`{"id":4}`))
	if err != nil || req.(*streamMsg).ID != 4 {
		t.Errorf("%v %v", req, err)
	}
	if _, err := reg.DecodeResponse("Stream", json.RawMessage(`{}`)); !errors.Is(err, ocpp.ErrUnknownAction) {
		t.Errorf("a one-way message has no response: %v", err)
	}
}

func TestEndpointSend(t *testing.T) {
	r := rig21(t, nil)
	if err := r.e.Send(context.Background(), streamMsg{ID: 3}); err != nil {
		t.Fatal(err)
	}
	m, ok := r.next21(t).(*Send)
	if !ok || m.Action != "Stream" || string(m.Payload) != `{"id":3}` {
		t.Errorf("%#v", m)
	}
}

func TestSendIsNotBlockedByAnOutstandingCall(t *testing.T) {
	r := rig21(t, nil)
	done := async(func() (ocpp.Response, error) {
		return r.e.Call(context.Background(), pingReq{Message: "hi"})
	})
	call := r.next21(t) // the outstanding CALL
	if _, ok := call.(*Call); !ok {
		t.Fatalf("%#v", call)
	}
	if err := r.e.Send(context.Background(), streamMsg{ID: 1}); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.next21(t).(*Send); !ok {
		t.Error("the SEND must go out while the CALL waits")
	}
	r.respond(call.ID(), `{"reply":"ok"}`)
	if got := get(t, done); got.err != nil {
		t.Error(got.err)
	}
}

func TestEndpointSendErrors(t *testing.T) {
	r := rig21(t, nil)
	ctx := context.Background()
	if err := r.e.Send(ctx, nil); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("nil: %v", err)
	}
	if err := r.e.Send(ctx, unencodableReq{}); !errors.Is(err, ocpp.ErrUnknownAction) {
		t.Errorf("unknown action is Unencodable here but a normal CALL: %v", err)
	}
	if err := r.e.Send(ctx, pingReq{Message: "x"}); !errors.Is(err, ocpp.ErrUnknownAction) {
		t.Errorf("a message with a response: %v", err)
	}
	if err := r.e.Send(ctx, streamMsg{ID: -1}); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("invalid: %v", err)
	}
	if err := r.e.Send(ctx, streamUnencodable{}); err == nil {
		t.Error("unencodable payload must fail")
	}
	r.setSendErr(errBoom)
	if err := r.e.Send(ctx, streamMsg{}); !errors.Is(err, errBoom) {
		t.Errorf("transport error: %v", err)
	}
	r.setSendErr(nil)
	r.e.Close()
	if err := r.e.Send(ctx, streamMsg{}); !errors.Is(err, ocpp.ErrClosed) {
		t.Errorf("closed: %v", err)
	}
}

func TestSendNeedsAnIDAndEncodableFrame(t *testing.T) {
	r := rig21(t, func(c *Config) { c.IDGenerator = fixedID("") })
	if err := r.e.Send(context.Background(), streamMsg{}); err == nil {
		t.Error("a frame without an ID must not be sent")
	}
}

type fixedID string

func (f fixedID) NewID() string { return string(f) }

func TestCallRejectsOneWayMessage(t *testing.T) {
	r := rig21(t, nil)
	if _, err := r.e.Call(context.Background(), streamMsg{}); !errors.Is(err, ocpp.ErrUnknownAction) {
		t.Errorf("%v", err)
	}
}

func TestIncomingSend(t *testing.T) {
	got := make(chan ocpp.Request, 1)
	r := rig21(t, func(c *Config) {
		c.SendHandler = func(_ context.Context, m ocpp.Request) error {
			got <- m
			return errBoom // only logged
		}
	})
	r.e.HandleFrame([]byte(`[6,"s1","Stream",{"id":9}]`))
	select {
	case m := <-got:
		if m.(*streamMsg).ID != 9 {
			t.Errorf("%v", m)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the SEND handler did not run")
	}
	r.expectNone(t)
}

func TestIncomingSendIsNeverAnswered(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	r := rig21(t, func(c *Config) {
		c.SendHandler = func(context.Context, ocpp.Request) error {
			mu.Lock()
			calls++
			mu.Unlock()
			return nil
		}
	})
	for _, frame := range []string{
		`[6,"a","Stream",{"id":-1}]`,     // invalid payload
		`[6,"b","Unknown",{}]`,           // unknown action
		`[6,"c","Ping",{"message":"x"}]`, // a message that needs a response
		`[6,"d","Stream"]`,               // malformed
	} {
		r.e.HandleFrame([]byte(frame))
	}
	r.expectNone(t)
	mu.Lock()
	defer mu.Unlock()
	if calls != 0 {
		t.Errorf("handler ran %d times for invalid SENDs", calls)
	}
}

func TestIncomingSendWithoutHandlerIsDropped(t *testing.T) {
	r := rig21(t, nil)
	r.e.HandleFrame([]byte(`[6,"s1","Stream",{"id":1}]`))
	r.expectNone(t)
}

func TestIncomingSendAfterCloseIsIgnored(t *testing.T) {
	ran := make(chan struct{}, 1)
	r := rig21(t, func(c *Config) {
		c.SendHandler = func(context.Context, ocpp.Request) error { ran <- struct{}{}; return nil }
	})
	r.e.Close()
	r.e.HandleFrame([]byte(`[6,"s1","Stream",{"id":1}]`))
	select {
	case <-ran:
		t.Error("handler ran after Close")
	case <-time.After(30 * time.Millisecond):
	}
}

func TestOneWayMessageSentAsCallIsRejected(t *testing.T) {
	r := rig21(t, nil)
	r.e.HandleFrame([]byte(`[2,"c1","Stream",{"id":1}]`))
	ce, ok := r.next21(t).(*CallErrorFrame)
	if !ok || ce.Code != ocpp.ProtocolError || ce.UniqueID != "c1" {
		t.Errorf("%#v", ce)
	}
}

func TestUndecodableResultIsAnsweredWithCallResultError(t *testing.T) {
	r := rig21(t, nil)
	done := async(func() (ocpp.Response, error) {
		return r.e.Call(context.Background(), pingReq{Message: "hi"})
	})
	call := r.next21(t)
	r.respond(call.ID(), `{"reply":"bad"}`)
	if got := get(t, done); got.err == nil {
		t.Fatal("the call must fail")
	}
	ce, ok := r.next21(t).(*CallResultErrorFrame)
	if !ok || ce.UniqueID != call.ID() || ce.Code != ocpp.PropertyConstraintViolation {
		t.Errorf("%#v", ce)
	}
}

func TestUndecodableResultIsNotAnsweredBefore21(t *testing.T) {
	r := newRig(t, nil)
	done := async(func() (ocpp.Response, error) {
		return r.e.Call(context.Background(), pingReq{Message: "hi"})
	})
	call := r.next(t)
	r.respond(call.ID(), `{"reply":"bad"}`)
	if got := get(t, done); got.err == nil {
		t.Fatal("the call must fail")
	}
	r.expectNone(t)
}

func TestIncomingCallResultError(t *testing.T) {
	got := make(chan *CallResultErrorFrame, 1)
	r := rig21(t, func(c *Config) { c.ResultErrorHandler = func(f *CallResultErrorFrame) { got <- f } })
	r.e.HandleFrame([]byte(`[5,"x","FormatViolation","nope",{}]`))
	select {
	case f := <-got:
		if f.UniqueID != "x" || f.Code != ocpp.FormatViolation {
			t.Errorf("%#v", f)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler not called")
	}
	// Without a handler it is only logged.
	r2 := rig21(t, nil)
	r2.e.HandleFrame([]byte(`[5,"x","FormatViolation","nope",{}]`))
	r2.expectNone(t)
}

func TestReplyTruncatesCallResultErrorDescription(t *testing.T) {
	r := rig21(t, nil)
	r.e.reply(&CallResultErrorFrame{UniqueID: "1", Code: ocpp.FormatViolation, Description: strings.Repeat("é", 300)})
	ce, ok := r.next21(t).(*CallResultErrorFrame)
	if !ok || len([]rune(ce.Description)) != MaxDescriptionLength {
		t.Errorf("%#v", ce)
	}
}

// Incoming SEND messages reach the handler one at a time and in the order they arrived.
func TestIncomingSendsKeepTheirOrder(t *testing.T) {
	var mu sync.Mutex
	var order []int
	running := 0
	overlap := false
	done := make(chan struct{})
	r := rig21(t, func(c *Config) {
		c.SendHandler = func(_ context.Context, m ocpp.Request) error {
			mu.Lock()
			running++
			overlap = overlap || running > 1
			mu.Unlock()
			time.Sleep(time.Millisecond)
			mu.Lock()
			running--
			order = append(order, m.(*streamMsg).ID)
			if len(order) == 50 {
				close(done)
			}
			mu.Unlock()
			return nil
		}
	})
	for i := 0; i < 50; i++ {
		r.e.HandleFrame(fmt.Appendf(nil, `[6,"s%d","Stream",{"id":%d}]`, i, i))
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("not every SEND was handled")
	}
	mu.Lock()
	defer mu.Unlock()
	if overlap {
		t.Error("two SEND handlers ran at once")
	}
	for i, id := range order {
		if id != i {
			t.Fatalf("handled out of order: %v", order)
		}
	}
}

// A SEND queue that is full drops the newest message instead of blocking the connection.
func TestIncomingSendQueueIsBounded(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{}, 1)
	var mu sync.Mutex
	var got []int
	r := rig21(t, func(c *Config) {
		c.QueueSize = 2
		c.SendHandler = func(_ context.Context, m ocpp.Request) error {
			select {
			case started <- struct{}{}:
			default:
			}
			<-release
			mu.Lock()
			got = append(got, m.(*streamMsg).ID)
			mu.Unlock()
			return nil
		}
	})
	r.e.HandleFrame([]byte(`[6,"a","Stream",{"id":0}]`))
	<-started // the first is in the handler; the queue is empty again
	for i := 1; i <= 4; i++ {
		r.e.HandleFrame(fmt.Appendf(nil, `[6,"s%d","Stream",{"id":%d}]`, i, i))
	}
	close(release)
	deadline := time.Now().Add(3 * time.Second)
	for {
		mu.Lock()
		n := len(got)
		mu.Unlock()
		if n == 3 || time.Now().After(deadline) {
			break
		}
		time.Sleep(time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 3 || got[0] != 0 || got[1] != 1 || got[2] != 2 {
		t.Errorf("handled %v, want [0 1 2] (4 and 3 dropped)", got)
	}
}

// Incoming CALLs are served one at a time, in arrival order.
func TestIncomingCallsAreServedInOrder(t *testing.T) {
	var mu sync.Mutex
	var order []string
	running, overlap := 0, false
	r := newRig(t, func(c *Config) {
		c.Handler = func(_ context.Context, req ocpp.Request) (ocpp.Response, error) {
			mu.Lock()
			running++
			overlap = overlap || running > 1
			mu.Unlock()
			time.Sleep(time.Millisecond)
			mu.Lock()
			running--
			order = append(order, req.(*pingReq).Message)
			mu.Unlock()
			return pingResp{Reply: "ok"}, nil
		}
	})
	for i := 0; i < 20; i++ {
		r.e.HandleFrame(fmt.Appendf(nil, `[2,"c%d","Ping",{"message":"m%d"}]`, i, i))
	}
	for i := 0; i < 20; i++ {
		if res, ok := r.next(t).(*CallResult); !ok || res.UniqueID != fmt.Sprintf("c%d", i) {
			t.Fatalf("reply %d: %#v", i, res)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if overlap {
		t.Error("two handlers ran at once")
	}
	for i, m := range order {
		if m != fmt.Sprintf("m%d", i) {
			t.Fatalf("served out of order: %v", order)
		}
	}
}

// A peer that sends more CALLs than the queue holds gets GenericError for the
// extra ones instead of starting unbounded handlers.
func TestIncomingCallFloodIsRefused(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{}, 1)
	r := newRig(t, func(c *Config) {
		c.QueueSize = 2
		c.Handler = func(context.Context, ocpp.Request) (ocpp.Response, error) {
			select {
			case started <- struct{}{}:
			default:
			}
			<-release
			return pingResp{Reply: "ok"}, nil
		}
	})
	r.e.HandleFrame([]byte(`[2,"a","Ping",{"message":"x"}]`))
	<-started
	for _, id := range []string{"b", "c", "d"} {
		r.e.HandleFrame([]byte(`[2,"` + id + `","Ping",{"message":"x"}]`))
	}
	if ce, ok := r.next(t).(*CallErrorFrame); !ok || ce.UniqueID != "d" || ce.Code != ocpp.GenericError {
		t.Fatalf("%#v", ce)
	}
	close(release)
	for _, id := range []string{"a", "b", "c"} {
		if res, ok := r.next(t).(*CallResult); !ok || res.UniqueID != id {
			t.Fatalf("want the answer to %s, got %#v", id, res)
		}
	}
	// After Close nothing is queued any more.
	r.e.Close()
	r.e.HandleFrame([]byte(`[2,"z","Ping",{"message":"x"}]`))
	r.expectNone(t)
}

// Abort fails waiting calls at once and does not wait for running handlers.
func TestAbortDoesNotWaitForHandlers(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	r := newRig(t, func(c *Config) {
		c.Handler = func(ctx context.Context, _ ocpp.Request) (ocpp.Response, error) {
			close(started)
			<-release
			return pingResp{Reply: "ok"}, nil
		}
	})
	r.e.HandleFrame([]byte(`[2,"a","Ping",{"message":"x"}]`))
	<-started
	call := async(func() (ocpp.Response, error) { return r.e.Call(context.Background(), pingReq{Message: "y"}) })
	r.next(t) // our CALL went out
	aborted := make(chan struct{})
	go func() { r.e.Abort(); r.e.Abort(); close(aborted) }()
	select {
	case <-aborted:
	case <-time.After(2 * time.Second):
		t.Fatal("Abort waited for the running handler")
	}
	if got := get(t, call); !errors.Is(got.err, ocpp.ErrClosed) {
		t.Errorf("waiting call: %v", got.err)
	}
	close(release)
	r.e.Close() // now waits for the handler, which has been released
}

// The next queued CALL is written by a goroutine of its own, not by the
// goroutine that delivered the response (the transport's read loop).
func TestNextCallIsNotWrittenByTheReader(t *testing.T) {
	var mu sync.Mutex
	var writers []int64
	reader := make(chan int64, 1)
	r := newRig(t, nil)
	r.e.send = func(_ context.Context, f []byte) error {
		mu.Lock()
		writers = append(writers, goid())
		mu.Unlock()
		r.frames <- f
		return nil
	}
	first := async(func() (ocpp.Response, error) { return r.e.Call(context.Background(), pingReq{Message: "1"}) })
	c1 := r.next(t)
	second := async(func() (ocpp.Response, error) { return r.e.Call(context.Background(), pingReq{Message: "2"}) })
	waitQueue(t, r.e, 1)
	go func() {
		reader <- goid()
		r.respond(c1.ID(), `{"reply":"ok"}`)
	}()
	readerID := <-reader
	c2 := r.next(t)
	r.respond(c2.ID(), `{"reply":"ok"}`)
	get(t, first)
	get(t, second)
	mu.Lock()
	defer mu.Unlock()
	for _, w := range writers {
		if w == readerID {
			t.Error("the reader goroutine wrote the next CALL")
		}
	}
}
