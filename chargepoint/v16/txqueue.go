package chargepoint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// Defaults for TransactionQueueConfig.
const (
	// DefaultTransactionMessageAttempts is how many times a transaction-related
	// message is tried before it is dropped.
	DefaultTransactionMessageAttempts = 3
	// DefaultTransactionMessageRetryInterval is the base wait before a retry.
	DefaultTransactionMessageRetryInterval = 60 * time.Second
	// DefaultTransactionQueueSize is how many messages may wait to be delivered.
	DefaultTransactionQueueSize = 10000
)

// A TransactionStore keeps the state of a TransactionQueue between runs of the
// program. Load is called once, by NewTransactionQueue; Save is called after every
// change, so it should be quick. Implementations need not be safe for
// concurrent use: the queue calls them one at a time.
type TransactionStore interface {
	// Load returns the saved state, or the zero state if there is none yet.
	Load() (QueueState, error)
	// Save replaces the saved state.
	Save(QueueState) error
}

// QueueState is what a TransactionStore keeps.
type QueueState struct {
	// NextSeq numbers the next message, NextLocal the next transaction handle.
	NextSeq   uint64 `json:"nextSeq"`
	NextLocal uint64 `json:"nextLocal"`
	// Items are the undelivered messages, oldest first.
	Items []QueuedMessage `json:"items,omitempty"`
	// Resolved maps the local handle of a transaction whose StartTransaction was
	// answered to the transactionId the Central System assigned, until its
	// StopTransaction has been dealt with.
	Resolved map[uint64]int `json:"resolved,omitempty"`
}

// QueuedMessage is one undelivered transaction-related message.
type QueuedMessage struct {
	Seq    uint64 `json:"seq"`
	Action string `json:"action"`
	// Local is the handle of the transaction the message belongs to, or 0 when
	// the message carries a transactionId of its own.
	Local uint64 `json:"local,omitempty"`
	// Payload is the request as JSON.
	Payload json.RawMessage `json:"payload"`
	// Attempts counts the failed deliveries so far.
	Attempts int `json:"attempts,omitempty"`
}

// FileStore returns a TransactionStore that keeps the queue in a JSON file,
// replaced atomically on every save. It survives a restart of the program.
func FileStore(path string) TransactionStore { return fileStore(path) }

type fileStore string

func (f fileStore) Load() (QueueState, error) {
	var st QueueState
	data, err := os.ReadFile(string(f))
	if errors.Is(err, os.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal(data, &st); err != nil {
		return QueueState{}, fmt.Errorf("%s: %w", f, err)
	}
	return st, nil
}

func (f fileStore) Save(st QueueState) error {
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(string(f)), filepath.Base(string(f))+".*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), string(f))
}

// memoryStore keeps nothing: the queue lives and dies with the program.
type memoryStore struct{}

func (memoryStore) Load() (QueueState, error) { return QueueState{}, nil }
func (memoryStore) Save(QueueState) error     { return nil }

// Discarded describes a message the queue gave up on.
type Discarded struct {
	// Action is the message's action: StartTransaction, StopTransaction or MeterValues.
	Action string
	// Attempts is how often delivery was tried.
	Attempts int
	// Err is why it was dropped: the last failure, or ErrTransactionUnknown.
	Err error
}

// ErrTransactionUnknown is the reason a message is dropped when the
// StartTransaction of its transaction was dropped, so its transactionId will
// never be known.
var ErrTransactionUnknown = errors.New("chargepoint: transaction was never started at the Central System")

// TransactionQueueConfig configures a TransactionQueue. Every field is optional.
type TransactionQueueConfig struct {
	// Attempts is how many times a message the Central System fails to process
	// (a CALLERROR, a timeout or an unusable answer) is sent before it is
	// dropped: the TransactionMessageAttempts configuration key. Default:
	// DefaultTransactionMessageAttempts.
	Attempts int
	// RetryInterval is the wait before the first retry; before the n-th retry
	// the queue waits n times this: the TransactionMessageRetryInterval
	// configuration key. Default: DefaultTransactionMessageRetryInterval.
	RetryInterval time.Duration
	// MaxQueued bounds the messages waiting for delivery. Default:
	// DefaultTransactionQueueSize.
	MaxQueued int
	// Store keeps the queue across restarts. Default: in memory only.
	Store TransactionStore
	// OnDiscard is told about every message the queue drops.
	OnDiscard func(Discarded)
	// Logger receives diagnostics. Default: discard.
	Logger *slog.Logger
}

// transactionCaller is the part of a ChargePoint the queue uses.
type transactionCaller interface {
	WaitConnected(ctx context.Context) error
	StartTransaction(ctx context.Context, req core.StartTransactionRequest) (*core.StartTransactionConfirmation, error)
	StopTransaction(ctx context.Context, req core.StopTransactionRequest) (*core.StopTransactionConfirmation, error)
	MeterValues(ctx context.Context, req core.MeterValuesRequest) (*core.MeterValuesConfirmation, error)
}

// A TransactionQueue delivers the transaction-related messages of OCPP 1.6
// (StartTransaction, StopTransaction and MeterValues) the way section 3.7 of the
// specification asks:
//
//   - in chronological order: a message is sent only after the ones before it
//     were delivered;
//   - while the charge point is offline they wait and are sent after it
//     reconnects, timestamps unchanged;
//   - a message the Central System fails to process is sent again, up to the
//     configured number of attempts, waiting longer each time, and is dropped
//     after the last one so the queue cannot get stuck.
//
// The SDK does none of this on its own: calls on a ChargePoint fail fast while
// it is offline. An application that wants this behaviour hands its
// transaction-related messages to the queue instead of calling the ChargePoint
// directly, and runs the queue with Run. Messages that are not
// transaction-related (Authorize, StatusNotification and so on) are still sent
// directly and need not wait.
//
// A transaction started while offline has no transactionId yet. Its
// StartTransaction returns a Transaction handle; give it to StopTransaction and
// MeterValues, and the queue fills in the transactionId once the Central System
// has assigned it.
//
// A TransactionQueue is safe for concurrent use.
type TransactionQueue struct {
	cp       transactionCaller
	attempts int
	interval time.Duration
	max      int
	store    TransactionStore
	onDrop   func(Discarded)
	log      *slog.Logger

	wake chan struct{}

	mu    sync.Mutex
	state QueueState
	txs   map[uint64]*Transaction
}

// NewTransactionQueue creates a queue that delivers through cp. It loads what
// the configured Store kept. Call Run to start delivering.
func NewTransactionQueue(cp *ChargePoint, cfg TransactionQueueConfig) (*TransactionQueue, error) {
	return newTransactionQueue(cp, cfg)
}

func newTransactionQueue(cp transactionCaller, cfg TransactionQueueConfig) (*TransactionQueue, error) {
	q := &TransactionQueue{
		cp: cp, attempts: cfg.Attempts, interval: cfg.RetryInterval, max: cfg.MaxQueued,
		store: cfg.Store, onDrop: cfg.OnDiscard, log: cfg.Logger,
		wake: make(chan struct{}, 1), txs: map[uint64]*Transaction{},
	}
	if q.attempts <= 0 {
		q.attempts = DefaultTransactionMessageAttempts
	}
	if q.interval <= 0 {
		q.interval = DefaultTransactionMessageRetryInterval
	}
	if q.max <= 0 {
		q.max = DefaultTransactionQueueSize
	}
	if q.store == nil {
		q.store = memoryStore{}
	}
	if q.log == nil {
		q.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	st, err := q.store.Load()
	if err != nil {
		return nil, fmt.Errorf("chargepoint: load the transaction queue: %w", err)
	}
	q.state = st
	if q.state.Resolved == nil {
		q.state.Resolved = map[uint64]int{}
	}
	for local, id := range q.state.Resolved {
		t := newTransaction(local)
		t.resolve(id, nil)
		q.txs[local] = t
	}
	// A transaction whose StartTransaction is still queued has a handle too.
	for _, m := range q.state.Items {
		if m.Action == core.StartTransactionAction {
			q.txs[m.Local] = newTransaction(m.Local)
		}
	}
	if len(q.state.Items) > 0 {
		q.signal()
	}
	return q, nil
}

// Transaction is the handle of a transaction started through a TransactionQueue.
// It is safe for concurrent use.
type Transaction struct {
	local uint64
	done  chan struct{}

	mu      sync.Mutex
	id      int
	info    *types.IDTagInfo
	started bool
}

func newTransaction(local uint64) *Transaction {
	return &Transaction{local: local, done: make(chan struct{})}
}

func (t *Transaction) resolve(id int, info *types.IDTagInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.started {
		return
	}
	t.id, t.info, t.started = id, info, true
	close(t.done)
}

// LocalID is a number that identifies the transaction in the queue and in its
// store, whatever the Central System assigns. It is not the transactionId.
func (t *Transaction) LocalID() uint64 { return t.local }

// Started is closed once the Central System has answered the transaction's
// StartTransaction.
func (t *Transaction) Started() <-chan struct{} { return t.done }

// TransactionID returns the transactionId the Central System assigned, and
// whether it has by now.
func (t *Transaction) TransactionID() (int, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.id, t.started
}

// IDTagInfo returns the authorization the Central System gave in its answer to
// StartTransaction. It is nil when the transaction was restored from a store, or
// has not been answered yet.
func (t *Transaction) IDTagInfo() *types.IDTagInfo {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.info
}

// Transaction returns the handle of a transaction by its LocalID, for example
// after a restart with a persistent store.
func (q *TransactionQueue) Transaction(local uint64) (*Transaction, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	t, ok := q.txs[local]
	return t, ok
}

// Len returns how many messages wait for delivery.
func (q *TransactionQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.state.Items)
}

// StartTransaction queues a StartTransaction message and returns the handle of
// the transaction. The request's timestamp is what is sent, so set it when the
// transaction really starts.
func (q *TransactionQueue) StartTransaction(req core.StartTransactionRequest) (*Transaction, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	local := q.state.NextLocal + 1
	if err := q.enqueue(core.StartTransactionAction, local, req); err != nil {
		return nil, err
	}
	q.state.NextLocal = local
	t := newTransaction(local)
	q.txs[local] = t
	return t, q.saveLocked()
}

// StopTransaction queues a StopTransaction message. For a transaction started
// through the queue pass its handle: the transactionId of req is then replaced
// by the one the Central System assigned. Pass nil for a transaction the
// application started itself, whose transactionId is in req.
func (q *TransactionQueue) StopTransaction(tx *Transaction, req core.StopTransactionRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if err := q.enqueue(core.StopTransactionAction, localOf(tx), req); err != nil {
		return err
	}
	return q.saveLocked()
}

// MeterValues queues a MeterValues message; tx is as for StopTransaction. With
// a nil tx the message is sent as it is, whether or not its transactionId is set.
func (q *TransactionQueue) MeterValues(tx *Transaction, req core.MeterValuesRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if err := q.enqueue(core.MeterValuesAction, localOf(tx), req); err != nil {
		return err
	}
	return q.saveLocked()
}

func localOf(tx *Transaction) uint64 {
	if tx == nil {
		return 0
	}
	return tx.local
}

// enqueue appends a message; the caller holds q.mu and saves.
func (q *TransactionQueue) enqueue(action string, local uint64, req any) error {
	if len(q.state.Items) >= q.max {
		return ocpp.ErrQueueFull
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("chargepoint: encode %s: %w", action, err)
	}
	q.state.NextSeq++
	q.state.Items = append(q.state.Items, QueuedMessage{Seq: q.state.NextSeq, Action: action, Local: local, Payload: payload})
	q.signal()
	return nil
}

func (q *TransactionQueue) signal() {
	select {
	case q.wake <- struct{}{}:
	default:
	}
}

// saveLocked persists the state; the caller holds q.mu. A store failure is
// returned to the caller that changed the queue.
func (q *TransactionQueue) saveLocked() error {
	if err := q.store.Save(q.state); err != nil {
		return fmt.Errorf("chargepoint: save the transaction queue: %w", err)
	}
	return nil
}

func (q *TransactionQueue) head() (QueuedMessage, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.state.Items) == 0 {
		return QueuedMessage{}, false
	}
	return q.state.Items[0], true
}

// Run delivers the queued messages, oldest first, until ctx ends; it returns
// ctx's error. Call it once, usually in its own goroutine.
func (q *TransactionQueue) Run(ctx context.Context) error {
	for {
		m, ok := q.head()
		if !ok {
			select {
			case <-q.wake:
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if err := q.deliver(ctx, m); err != nil {
			return err
		}
	}
}

// deliver sends the head message until it is answered or dropped. It returns an
// error only when ctx ends.
func (q *TransactionQueue) deliver(ctx context.Context, m QueuedMessage) error {
	for {
		if err := q.cp.WaitConnected(ctx); err != nil {
			return err
		}
		resp, err := q.send(ctx, &m)
		switch {
		case err == nil:
			q.answered(m, resp)
			return nil
		case ctx.Err() != nil:
			return ctx.Err()
		case errors.Is(err, ocpp.ErrNotConnected) || errors.Is(err, ocpp.ErrClosed) || errors.Is(err, ocpp.ErrQueueFull):
			// Not the Central System's failure: wait for the connection and try again.
			if !sleep(ctx, 10*time.Millisecond) {
				return ctx.Err()
			}
			continue
		case errors.Is(err, ErrTransactionUnknown) || errors.Is(err, ocpp.ErrValidation):
			q.drop(m, err) // it can never be delivered
			return nil
		}
		// The Central System failed to process the message.
		m.Attempts++
		q.log.Warn("transaction message not processed", "action", m.Action, "attempt", m.Attempts, "err", err)
		if m.Attempts >= q.attempts {
			q.drop(m, err)
			return nil
		}
		q.recordAttempt(m)
		if !sleep(ctx, time.Duration(m.Attempts)*q.interval) {
			return ctx.Err()
		}
	}
}

func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// send builds and sends the request of m; the result is the confirmation.
func (q *TransactionQueue) send(ctx context.Context, m *QueuedMessage) (any, error) {
	var id int
	var known bool
	if m.Action != core.StartTransactionAction { // its Local is the transaction it starts
		var err error
		if id, known, err = q.transactionID(m.Local); err != nil {
			return nil, err
		}
	}
	switch m.Action {
	case core.StartTransactionAction:
		var req core.StartTransactionRequest
		if err := json.Unmarshal(m.Payload, &req); err != nil {
			return nil, fmt.Errorf("%w: %v", ocpp.ErrValidation, err)
		}
		return q.cp.StartTransaction(ctx, req)
	case core.StopTransactionAction:
		var req core.StopTransactionRequest
		if err := json.Unmarshal(m.Payload, &req); err != nil {
			return nil, fmt.Errorf("%w: %v", ocpp.ErrValidation, err)
		}
		if known {
			req.TransactionID = id
		}
		return q.cp.StopTransaction(ctx, req)
	default:
		var req core.MeterValuesRequest
		if err := json.Unmarshal(m.Payload, &req); err != nil {
			return nil, fmt.Errorf("%w: %v", ocpp.ErrValidation, err)
		}
		if known {
			req.TransactionID = &id
		}
		return q.cp.MeterValues(ctx, req)
	}
}

// transactionID resolves the local handle of a message: known is false when the
// message has none. A message whose transaction was never started is an error.
func (q *TransactionQueue) transactionID(local uint64) (id int, known bool, err error) {
	if local == 0 {
		return 0, false, nil
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	id, ok := q.state.Resolved[local]
	if !ok {
		return 0, false, ErrTransactionUnknown
	}
	return id, true, nil
}

// answered removes m from the queue after the Central System accepted it, and
// records what StartTransaction returned.
func (q *TransactionQueue) answered(m QueuedMessage, resp any) {
	q.mu.Lock()
	var t *Transaction
	var id int
	var info *types.IDTagInfo
	if conf, ok := resp.(*core.StartTransactionConfirmation); ok && m.Action == core.StartTransactionAction {
		id, info = conf.TransactionID, &conf.IDTagInfo
		q.state.Resolved[m.Local] = id
		t = q.txs[m.Local]
	}
	q.removeLocked(m)
	q.saveAndLog()
	q.mu.Unlock()
	if t != nil {
		t.resolve(id, info)
	}
}

// drop removes m from the queue because it cannot or will not be delivered.
func (q *TransactionQueue) drop(m QueuedMessage, reason error) {
	q.log.Error("dropping transaction message", "action", m.Action, "attempts", m.Attempts, "err", reason)
	q.mu.Lock()
	q.removeLocked(m)
	q.saveAndLog()
	q.mu.Unlock()
	if q.onDrop != nil {
		q.onDrop(Discarded{Action: m.Action, Attempts: m.Attempts, Err: reason})
	}
}

// removeLocked takes m (the head) off the queue and forgets a transaction that
// has ended; the caller holds q.mu.
func (q *TransactionQueue) removeLocked(m QueuedMessage) {
	if len(q.state.Items) > 0 && q.state.Items[0].Seq == m.Seq {
		q.state.Items = q.state.Items[1:]
	}
	if m.Action == core.StopTransactionAction && m.Local != 0 {
		delete(q.state.Resolved, m.Local)
		delete(q.txs, m.Local)
	}
}

// recordAttempt keeps the failed attempts of the head message in the store.
func (q *TransactionQueue) recordAttempt(m QueuedMessage) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.state.Items) > 0 && q.state.Items[0].Seq == m.Seq {
		q.state.Items[0].Attempts = m.Attempts
	}
	q.saveAndLog()
}

func (q *TransactionQueue) saveAndLog() {
	if err := q.saveLocked(); err != nil {
		q.log.Error("cannot save the transaction queue", "err", err)
	}
}
