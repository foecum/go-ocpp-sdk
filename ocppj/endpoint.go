package ocppj

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

// Defaults for Config.
const (
	// DefaultTimeout is how long a sent CALL waits for its response.
	DefaultTimeout = 30 * time.Second
	// DefaultQueueSize is how many calls may wait behind the outstanding one.
	DefaultQueueSize = 1000
)

// RequestHandler handles an incoming, decoded CALL and returns the response.
// Returning a *ocpp.CallError sends that CALLERROR; any other error is sent as
// an InternalError CALLERROR. It runs in its own goroutine, so it may itself
// call Endpoint.Call.
type RequestHandler func(ctx context.Context, req ocpp.Request) (ocpp.Response, error)

// SendHandler handles an incoming SEND, an unconfirmed message of OCPP 2.1.
// Nothing is sent back whatever it returns; an error is only logged. It runs in
// its own goroutine.
type SendHandler func(ctx context.Context, msg ocpp.Request) error

// Config configures an Endpoint.
type Config struct {
	// Registry is required and decides the OCPP version.
	Registry *Registry
	// Send writes one frame to the connection. It is required. It should
	// return an error matching ocpp.ErrNotConnected when there is no connection.
	Send func(ctx context.Context, frame []byte) error
	// Handler handles incoming CALLs. If nil, every CALL is answered with
	// NotImplemented.
	Handler RequestHandler
	// SendHandler handles incoming SEND messages (OCPP 2.1). If nil, they are
	// logged and dropped.
	SendHandler SendHandler
	// ResultErrorHandler is told about CALLRESULTERROR frames (OCPP 2.1), with
	// which the peer reports that it could not process a CALLRESULT of ours. If
	// nil, they are only logged.
	ResultErrorHandler func(*CallResultErrorFrame)
	// IDGenerator creates message IDs. Default: NewUUIDGenerator.
	IDGenerator IDGenerator
	// Timeout is the per-call response timeout, counted from when the CALL is
	// sent. Default: DefaultTimeout.
	Timeout time.Duration
	// QueueSize bounds the calls waiting behind the outstanding one. Default:
	// DefaultQueueSize.
	QueueSize int
	// Logger receives diagnostics. Default: discard.
	Logger *slog.Logger
}

type outcome struct {
	resp ocpp.Response
	err  error
}

type pending struct {
	id     string
	action string
	frame  []byte
	ctx    context.Context
	done   chan outcome // buffered, receives exactly one outcome
	once   sync.Once
	timer  *time.Timer
}

func (p *pending) finish(resp ocpp.Response, err error) {
	p.once.Do(func() { p.done <- outcome{resp, err} })
}

// Endpoint is one side of an OCPP-J connection. It sends CALLs one at a time
// (the spec allows a single outstanding CALL per direction), queueing the
// rest, correlates responses by message ID, times calls out, and dispatches
// incoming CALLs to a handler. It does no I/O itself: the transport supplies
// Config.Send and feeds received frames to HandleFrame. All methods are safe
// for concurrent use.
type Endpoint struct {
	reg      *Registry
	send     func(context.Context, []byte) error
	handler  RequestHandler
	sendH    SendHandler
	resultH  func(*CallResultErrorFrame)
	ids      IDGenerator
	timeout  time.Duration
	maxQueue int
	log      *slog.Logger

	baseCtx context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	mu      sync.Mutex
	current *pending
	queue   []*pending
	closed  bool

	// Incoming SEND messages are handled one at a time, in arrival order, because
	// the values of a stream only make sense in order.
	sends      []*Send
	sendActive bool // a goroutine is draining sends
}

// NewEndpoint creates an Endpoint.
func NewEndpoint(cfg Config) (*Endpoint, error) {
	if cfg.Registry == nil || cfg.Send == nil {
		return nil, errors.New("ocppj: Config needs a Registry and a Send function")
	}
	e := &Endpoint{
		reg: cfg.Registry, send: cfg.Send, handler: cfg.Handler, sendH: cfg.SendHandler, resultH: cfg.ResultErrorHandler, ids: cfg.IDGenerator,
		timeout: cfg.Timeout, maxQueue: cfg.QueueSize, log: cfg.Logger,
	}
	if e.ids == nil {
		e.ids = NewUUIDGenerator()
	}
	if e.timeout <= 0 {
		e.timeout = DefaultTimeout
	}
	if e.maxQueue <= 0 {
		e.maxQueue = DefaultQueueSize
	}
	if e.log == nil {
		e.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	e.baseCtx, e.cancel = context.WithCancel(context.Background())
	return e, nil
}

// Call sends req and waits for the response. It returns:
//   - an error matching ocpp.ErrValidation if req is invalid;
//   - ocpp.ErrClosed if the endpoint is closed, ocpp.ErrQueueFull if too many
//     calls are waiting;
//   - ocpp.ErrTimeout if no response arrives within the timeout;
//   - *ocpp.CallError (matching ocpp.ErrCallError) if the peer answers with a
//     CALLERROR;
//   - ctx.Err() if ctx ends first;
//   - the transport's error if sending fails.
//
// The action must be registered. Calls from several goroutines are queued
// first-come first-served.
func (e *Endpoint) Call(ctx context.Context, req ocpp.Request) (ocpp.Response, error) {
	if req == nil {
		return nil, ocpp.Required("request")
	}
	f, ok := e.reg.Lookup(req.Action())
	if !ok {
		return nil, fmt.Errorf("%w: %q", ocpp.ErrUnknownAction, req.Action())
	}
	if ocpp.IsOneWay(f) {
		return nil, fmt.Errorf("%w: %q is an unconfirmed message, use Send", ocpp.ErrUnknownAction, req.Action())
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	payload, err := marshalPayload(req)
	if err != nil {
		return nil, err
	}
	p := &pending{id: e.ids.NewID(), action: req.Action(), ctx: ctx, done: make(chan outcome, 1)}
	if p.frame, err = Marshal(&Call{UniqueID: p.id, Action: p.action, Payload: payload}); err != nil {
		return nil, err
	}

	e.mu.Lock()
	switch {
	case e.closed:
		e.mu.Unlock()
		return nil, ocpp.ErrClosed
	case len(e.queue) >= e.maxQueue:
		e.mu.Unlock()
		return nil, ocpp.ErrQueueFull
	}
	e.queue = append(e.queue, p)
	e.mu.Unlock()
	e.pump()

	select {
	case o := <-p.done:
		return o.resp, o.err
	case <-ctx.Done():
		e.abandon(p)
		return nil, ctx.Err()
	}
}

// Send sends msg as an unconfirmed SEND frame (OCPP 2.1) and returns once it
// is written. A SEND is not queued behind CALLs and does not wait for any
// answer: the spec lets a SEND go out while a CALL is outstanding. It returns
// an error matching ocpp.ErrValidation if msg is invalid, ocpp.ErrUnknownAction
// if its action is unknown or is not an unconfirmed message, ocpp.ErrClosed if
// the endpoint is closed, or the transport's error.
func (e *Endpoint) Send(ctx context.Context, msg ocpp.Request) error {
	if msg == nil {
		return ocpp.Required("message")
	}
	f, ok := e.reg.Lookup(msg.Action())
	if !ok {
		return fmt.Errorf("%w: %q", ocpp.ErrUnknownAction, msg.Action())
	}
	if !ocpp.IsOneWay(f) {
		return fmt.Errorf("%w: %q is not an unconfirmed message, use Call", ocpp.ErrUnknownAction, msg.Action())
	}
	if err := msg.Validate(); err != nil {
		return err
	}
	payload, err := marshalPayload(msg)
	if err != nil {
		return err
	}
	frame, err := Marshal(&Send{UniqueID: e.ids.NewID(), Action: msg.Action(), Payload: payload})
	if err != nil {
		return err
	}
	e.mu.Lock()
	closed := e.closed
	e.mu.Unlock()
	if closed {
		return ocpp.ErrClosed
	}
	if err := e.send(ctx, frame); err != nil {
		return fmt.Errorf("ocppj: send %s: %w", msg.Action(), err)
	}
	return nil
}

// pump sends queued calls while no call is outstanding.
func (e *Endpoint) pump() {
	for {
		e.mu.Lock()
		if e.closed || e.current != nil || len(e.queue) == 0 {
			e.mu.Unlock()
			return
		}
		p := e.queue[0]
		e.queue = e.queue[1:]
		if p.ctx.Err() != nil { // caller gave up while queued
			e.mu.Unlock()
			p.finish(nil, p.ctx.Err())
			continue
		}
		e.current = p
		e.mu.Unlock()

		if err := e.send(p.ctx, p.frame); err != nil {
			e.mu.Lock()
			if e.current == p {
				e.current = nil
			}
			e.mu.Unlock()
			p.finish(nil, fmt.Errorf("ocppj: send %s: %w", p.action, err))
			continue
		}
		e.mu.Lock()
		if e.current == p { // not already answered or abandoned
			p.timer = time.AfterFunc(e.timeout, func() { e.expire(p) })
		}
		e.mu.Unlock()
		return
	}
}

// expire fails p with a timeout and moves on.
func (e *Endpoint) expire(p *pending) {
	if e.release(p) {
		p.finish(nil, fmt.Errorf("ocppj: %s %s: %w", p.action, p.id, ocpp.ErrTimeout))
		e.pump()
	}
}

// abandon removes a call whose caller's context ended.
func (e *Endpoint) abandon(p *pending) {
	e.mu.Lock()
	for i, q := range e.queue {
		if q == p {
			e.queue = append(e.queue[:i], e.queue[i+1:]...)
			break
		}
	}
	e.mu.Unlock()
	if e.release(p) {
		e.pump()
	}
}

// release clears p as the outstanding call; it reports whether p was it.
func (e *Endpoint) release(p *pending) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.current != p {
		return false
	}
	e.current = nil
	if p.timer != nil {
		p.timer.Stop()
	}
	return true
}

// HandleFrame processes one frame received from the peer. Incoming CALLs are
// handled in a new goroutine; responses complete the matching Call.
func (e *Endpoint) HandleFrame(data []byte) {
	m, err := Parse(e.reg.Version(), data)
	if err != nil {
		var pe *ParseError
		errors.As(err, &pe)
		e.log.Warn("ocppj: invalid frame", "err", err)
		if pe.ShouldReply() {
			e.reply(&CallErrorFrame{UniqueID: pe.ID, Code: pe.Code, Description: pe.Err.Error()})
		}
		return
	}
	switch m := m.(type) {
	case *Call:
		e.mu.Lock()
		if e.closed {
			e.mu.Unlock()
			return
		}
		e.wg.Add(1)
		e.mu.Unlock()
		go func() {
			defer e.wg.Done()
			e.serve(m)
		}()
	case *Send:
		e.enqueueSend(m)
	case *CallResultErrorFrame:
		e.log.Warn("ocppj: peer could not process our response", "id", m.UniqueID, "code", m.Code, "description", m.Description)
		if e.resultH != nil {
			e.resultH(m)
		}
	case *CallResult:
		e.complete(m.UniqueID, func(p *pending) (ocpp.Response, error) {
			resp, err := e.reg.DecodeResponse(p.action, m.Payload)
			if err != nil {
				e.rejectResult(m.UniqueID, err)
				return nil, fmt.Errorf("ocppj: %s response: %w", p.action, err)
			}
			return resp, nil
		})
	case *CallErrorFrame:
		e.complete(m.UniqueID, func(*pending) (ocpp.Response, error) {
			return nil, &ocpp.CallError{Code: m.Code, Description: m.Description, Details: m.Details}
		})
	}
}

// complete matches a response to the outstanding call and finishes it.
func (e *Endpoint) complete(id string, result func(*pending) (ocpp.Response, error)) {
	e.mu.Lock()
	p := e.current
	if p == nil || p.id != id {
		e.mu.Unlock()
		e.log.Warn("ocppj: response with unknown message ID", "id", id)
		return
	}
	e.current = nil
	if p.timer != nil {
		p.timer.Stop()
	}
	e.mu.Unlock()
	p.finish(result(p))
	e.pump()
}

// enqueueSend queues an incoming SEND for the goroutine that hands them to the
// handler in order. When more than the queue size are waiting, the newest is dropped.
func (e *Endpoint) enqueueSend(m *Send) {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return
	}
	if len(e.sends) >= e.maxQueue {
		e.mu.Unlock()
		e.log.Warn("ocppj: dropping SEND, queue full", "action", m.Action)
		return
	}
	e.sends = append(e.sends, m)
	if e.sendActive {
		e.mu.Unlock()
		return
	}
	e.sendActive = true
	e.wg.Add(1)
	e.mu.Unlock()
	go e.drainSends()
}

// drainSends hands the queued SEND messages to the handler, one after another.
func (e *Endpoint) drainSends() {
	defer e.wg.Done()
	for {
		e.mu.Lock()
		if e.closed || len(e.sends) == 0 {
			e.sends, e.sendActive = nil, false
			e.mu.Unlock()
			return
		}
		m := e.sends[0]
		e.sends = e.sends[1:]
		e.mu.Unlock()
		e.serveSend(m)
	}
}

// rejectResult answers a CALLRESULT we could not decode with a CALLRESULTERROR,
// which only exists in OCPP 2.1.
func (e *Endpoint) rejectResult(id string, err error) {
	if e.reg.Version() != ocpp.V21 {
		return
	}
	f := e.errorFrame(id, err)
	e.reply(&CallResultErrorFrame{UniqueID: f.UniqueID, Code: f.Code, Description: f.Description, Details: f.Details})
}

// serveSend decodes an incoming SEND and gives it to the handler. Nothing is
// ever sent back.
func (e *Endpoint) serveSend(s *Send) {
	if f, ok := e.reg.Lookup(s.Action); ok && !ocpp.IsOneWay(f) {
		e.log.Warn("ocppj: dropping SEND of a message that needs a response", "action", s.Action)
		return
	}
	msg, err := e.reg.DecodeRequest(s.Action, s.Payload)
	if err != nil {
		e.log.Warn("ocppj: dropping invalid SEND", "action", s.Action, "err", err)
		return
	}
	if e.sendH == nil {
		e.log.Warn("ocppj: dropping SEND, no handler", "action", s.Action)
		return
	}
	if err := e.sendH(e.baseCtx, msg); err != nil {
		e.log.Warn("ocppj: SEND handler failed", "action", s.Action, "err", err)
	}
}

// serve runs the handler for an incoming CALL and sends the reply.
func (e *Endpoint) serve(c *Call) {
	if f, ok := e.reg.Lookup(c.Action); ok && ocpp.IsOneWay(f) {
		e.reply(&CallErrorFrame{UniqueID: c.UniqueID, Code: ocpp.ProtocolError, Description: c.Action + " is an unconfirmed message and must be sent as SEND"})
		return
	}
	req, err := e.reg.DecodeRequest(c.Action, c.Payload)
	if err != nil {
		e.reply(e.errorFrame(c.UniqueID, err))
		return
	}
	if e.handler == nil {
		e.reply(&CallErrorFrame{UniqueID: c.UniqueID, Code: ocpp.NotImplemented, Description: "no handler for " + c.Action})
		return
	}
	resp, err := e.handler(e.baseCtx, req)
	if err != nil {
		e.reply(e.errorFrame(c.UniqueID, err))
		return
	}
	if resp == nil {
		e.reply(&CallErrorFrame{UniqueID: c.UniqueID, Code: ocpp.InternalError, Description: "handler returned no response"})
		return
	}
	if err := resp.Validate(); err != nil {
		e.log.Error("ocppj: handler produced invalid response", "action", c.Action, "err", err)
		e.reply(&CallErrorFrame{UniqueID: c.UniqueID, Code: ocpp.InternalError, Description: "invalid response"})
		return
	}
	payload, err := marshalPayload(resp)
	if err != nil {
		e.reply(&CallErrorFrame{UniqueID: c.UniqueID, Code: ocpp.InternalError, Description: "cannot encode response"})
		return
	}
	e.reply(&CallResult{UniqueID: c.UniqueID, Payload: payload})
}

// errorFrame maps an error to the CALLERROR to send.
func (e *Endpoint) errorFrame(id string, err error) *CallErrorFrame {
	var de *DecodeError
	var ce *ocpp.CallError
	switch {
	case errors.Is(err, ocpp.ErrUnknownAction):
		return &CallErrorFrame{UniqueID: id, Code: ocpp.NotImplemented, Description: err.Error()}
	case errors.As(err, &de):
		return &CallErrorFrame{UniqueID: id, Code: de.Code, Description: truncate(de.Err.Error())}
	case errors.As(err, &ce):
		return &CallErrorFrame{UniqueID: id, Code: ce.Code, Description: ce.Description, Details: ce.Details}
	}
	e.log.Error("ocppj: handler failed", "err", err)
	return &CallErrorFrame{UniqueID: id, Code: ocpp.InternalError}
}

func truncate(s string) string {
	if r := []rune(s); len(r) > MaxDescriptionLength {
		return string(r[:MaxDescriptionLength])
	}
	return s
}

// reply sends a CALLRESULT or CALLERROR; failures are logged.
func (e *Endpoint) reply(m Message) {
	switch f := m.(type) {
	case *CallErrorFrame:
		f.Description = truncate(f.Description)
	case *CallResultErrorFrame:
		f.Description = truncate(f.Description)
	}
	frame, err := Marshal(m)
	if err == nil {
		err = e.send(e.baseCtx, frame)
	}
	if err != nil {
		e.log.Warn("ocppj: cannot send reply", "id", m.ID(), "err", err)
	}
}

// Close fails all pending and queued calls with ocpp.ErrClosed, cancels the
// context of running handlers and waits for them to return. It is idempotent.
func (e *Endpoint) Close() {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		e.wg.Wait()
		return
	}
	e.closed = true
	doomed := e.queue
	if e.current != nil {
		doomed = append(doomed, e.current)
		if e.current.timer != nil {
			e.current.timer.Stop()
		}
	}
	e.queue, e.current = nil, nil
	e.mu.Unlock()

	e.cancel()
	for _, p := range doomed {
		p.finish(nil, ocpp.ErrClosed)
	}
	e.wg.Wait()
}
