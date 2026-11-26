package chargepoint

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/ocppj"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// fakeCP records the calls the queue makes and answers them from scripts.
type fakeCP struct {
	mu        sync.Mutex
	calls     []string
	stopIDs   []int
	meterIDs  []*int
	connected chan struct{} // closed while "connected"
	failures  map[string][]error
	nextTx    int
}

func newFakeCP(online bool) *fakeCP {
	f := &fakeCP{connected: make(chan struct{}), failures: map[string][]error{}, nextTx: 100}
	if online {
		close(f.connected)
	}
	return f
}

func (f *fakeCP) goOnline() {
	f.mu.Lock()
	defer f.mu.Unlock()
	select {
	case <-f.connected:
	default:
		close(f.connected)
	}
}

func (f *fakeCP) WaitConnected(ctx context.Context) error {
	f.mu.Lock()
	c := f.connected
	f.mu.Unlock()
	select {
	case <-c:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// next pops the scripted failure for action, if any.
func (f *fakeCP) next(action string) error {
	f.calls = append(f.calls, action)
	if errs := f.failures[action]; len(errs) > 0 {
		f.failures[action] = errs[1:]
		return errs[0]
	}
	return nil
}

func (f *fakeCP) StartTransaction(_ context.Context, req core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.next("StartTransaction"); err != nil {
		return nil, err
	}
	f.nextTx++
	return &core.StartTransactionConfirmation{TransactionID: f.nextTx, IDTagInfo: types.IDTagInfo{Status: types.AuthorizationStatusAccepted}}, nil
}

func (f *fakeCP) StopTransaction(_ context.Context, req core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.next("StopTransaction"); err != nil {
		return nil, err
	}
	f.stopIDs = append(f.stopIDs, req.TransactionID)
	return &core.StopTransactionConfirmation{}, nil
}

func (f *fakeCP) MeterValues(_ context.Context, req core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.next("MeterValues"); err != nil {
		return nil, err
	}
	f.meterIDs = append(f.meterIDs, req.TransactionID)
	return &core.MeterValuesConfirmation{}, nil
}

func (f *fakeCP) sent() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

func start() core.StartTransactionRequest {
	return core.StartTransactionRequest{ConnectorID: 1, IDTag: "TAG", MeterStart: 0, Timestamp: types.Now()}
}

func stop() core.StopTransactionRequest {
	return core.StopTransactionRequest{MeterStop: 10, Timestamp: types.Now()}
}

func meter() core.MeterValuesRequest {
	return core.MeterValuesRequest{ConnectorID: 1, MeterValue: []types.MeterValue{{
		Timestamp: types.Now(), SampledValue: []types.SampledValue{{Value: "5"}}}}}
}

func fastConfig() TransactionQueueConfig {
	return TransactionQueueConfig{Attempts: 3, RetryInterval: 5 * time.Millisecond}
}

func runQueue(t *testing.T, q *TransactionQueue) (stopFn func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- q.Run(ctx) }()
	return func() {
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Errorf("Run returned %v", err)
		}
	}
}

func waitUntil(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timeout waiting for " + what)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestMessagesAreDeliveredInOrderWithTheAssignedTransactionID(t *testing.T) {
	cp := newFakeCP(true)
	q, err := newTransactionQueue(cp, fastConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer runQueue(t, q)()
	tx, err := q.StartTransaction(start())
	if err != nil {
		t.Fatal(err)
	}
	if err := q.MeterValues(tx, meter()); err != nil {
		t.Fatal(err)
	}
	if err := q.StopTransaction(tx, stop()); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, "delivery", func() bool { return q.Len() == 0 })
	if got := cp.sent(); len(got) != 3 || got[0] != "StartTransaction" || got[1] != "MeterValues" || got[2] != "StopTransaction" {
		t.Errorf("sent %v", got)
	}
	select {
	case <-tx.Started():
	default:
		t.Error("Started must be closed")
	}
	if id, ok := tx.TransactionID(); !ok || id != 101 {
		t.Errorf("transaction id %d %v", id, ok)
	}
	if info := tx.IDTagInfo(); info == nil || info.Status != types.AuthorizationStatusAccepted {
		t.Errorf("idTagInfo %+v", info)
	}
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if len(cp.stopIDs) != 1 || cp.stopIDs[0] != 101 || cp.meterIDs[0] == nil || *cp.meterIDs[0] != 101 {
		t.Errorf("the assigned id was not filled in: stop %v meter %v", cp.stopIDs, cp.meterIDs)
	}
	if _, ok := q.Transaction(tx.LocalID()); ok {
		t.Error("an ended transaction is forgotten")
	}
}

func TestMessagesWaitWhileOfflineAndKeepTheirOrder(t *testing.T) {
	cp := newFakeCP(false)
	q, _ := newTransactionQueue(cp, fastConfig())
	defer runQueue(t, q)()
	tx, _ := q.StartTransaction(start())
	_ = q.MeterValues(tx, meter())
	_ = q.StopTransaction(tx, stop())
	time.Sleep(30 * time.Millisecond)
	if got := cp.sent(); len(got) != 0 || q.Len() != 3 {
		t.Fatalf("sent %v while offline, %d queued", got, q.Len())
	}
	cp.goOnline()
	waitUntil(t, "delivery after reconnecting", func() bool { return q.Len() == 0 })
	if got := cp.sent(); len(got) != 3 || got[0] != "StartTransaction" || got[2] != "StopTransaction" {
		t.Errorf("sent %v", got)
	}
}

func TestACallThatFailsBecauseTheConnectionIsLostIsNotAFailureOfTheCentralSystem(t *testing.T) {
	cp := newFakeCP(true)
	cp.failures["StopTransaction"] = []error{ocpp.ErrNotConnected, ocpp.ErrClosed, ocpp.ErrQueueFull, ocpp.ErrNotConnected}
	var dropped []Discarded
	cfg := fastConfig()
	cfg.Attempts = 1 // a single processing failure would drop the message
	cfg.OnDiscard = func(d Discarded) { dropped = append(dropped, d) }
	q, _ := newTransactionQueue(cp, cfg)
	defer runQueue(t, q)()
	_ = q.StopTransaction(nil, stop())
	waitUntil(t, "delivery", func() bool { return q.Len() == 0 })
	if len(dropped) != 0 {
		t.Errorf("dropped %+v", dropped)
	}
	if got := cp.sent(); len(got) != 5 {
		t.Errorf("sent %v, want four failed sends and one that worked", got)
	}
}

func TestAFailedMessageIsRetriedAndThenDropped(t *testing.T) {
	cp := newFakeCP(true)
	boom := &ocpp.CallError{Code: ocpp.InternalError, Description: "no"}
	cp.failures["StopTransaction"] = []error{boom, boom, boom, boom}
	var dropped []Discarded
	cfg := fastConfig()
	cfg.OnDiscard = func(d Discarded) { dropped = append(dropped, d) }
	q, _ := newTransactionQueue(cp, cfg)
	defer runQueue(t, q)()
	_ = q.StopTransaction(nil, stop()) // fails 3 times and is dropped
	_ = q.MeterValues(nil, meter())    // the queue goes on with the next one
	waitUntil(t, "both messages dealt with", func() bool { return q.Len() == 0 })
	if got := cp.sent(); len(got) != 4 || got[0] != "StopTransaction" || got[3] != "MeterValues" {
		t.Errorf("sent %v", got)
	}
	if len(dropped) != 1 || dropped[0].Action != "StopTransaction" || dropped[0].Attempts != 3 || !errors.Is(dropped[0].Err, ocpp.ErrCallError) {
		t.Errorf("dropped %+v", dropped)
	}
}

func TestAMessageThatSucceedsOnTheSecondTryIsDelivered(t *testing.T) {
	cp := newFakeCP(true)
	cp.failures["MeterValues"] = []error{ocpp.ErrTimeout}
	q, _ := newTransactionQueue(cp, fastConfig())
	defer runQueue(t, q)()
	_ = q.MeterValues(nil, meter())
	waitUntil(t, "delivery", func() bool { return q.Len() == 0 })
	if got := cp.sent(); len(got) != 2 {
		t.Errorf("sent %v", got)
	}
}

func TestTheWaitBeforeARetryGrowsWithEveryAttempt(t *testing.T) {
	cp := newFakeCP(true)
	cp.failures["MeterValues"] = []error{ocpp.ErrTimeout, ocpp.ErrTimeout}
	cfg := TransactionQueueConfig{Attempts: 3, RetryInterval: 40 * time.Millisecond}
	q, _ := newTransactionQueue(cp, cfg)
	defer runQueue(t, q)()
	begin := time.Now()
	_ = q.MeterValues(nil, meter())
	waitUntil(t, "delivery", func() bool { return q.Len() == 0 })
	// 40 ms before the first retry and 80 ms before the second.
	if elapsed := time.Since(begin); elapsed < 110*time.Millisecond {
		t.Errorf("delivered after %v, want at least 120ms of waiting", elapsed)
	}
}

func TestMessagesOfATransactionThatNeverStartedAreDropped(t *testing.T) {
	cp := newFakeCP(true)
	boom := &ocpp.CallError{Code: ocpp.InternalError}
	cp.failures["StartTransaction"] = []error{boom, boom, boom}
	var dropped []Discarded
	cfg := fastConfig()
	cfg.OnDiscard = func(d Discarded) { dropped = append(dropped, d) }
	q, _ := newTransactionQueue(cp, cfg)
	defer runQueue(t, q)()
	tx, _ := q.StartTransaction(start())
	_ = q.MeterValues(tx, meter())
	_ = q.StopTransaction(tx, stop())
	waitUntil(t, "everything dealt with", func() bool { return q.Len() == 0 })
	if len(dropped) != 3 || !errors.Is(dropped[1].Err, ErrTransactionUnknown) || !errors.Is(dropped[2].Err, ErrTransactionUnknown) {
		t.Errorf("dropped %+v", dropped)
	}
	if _, ok := tx.TransactionID(); ok {
		t.Error("a transaction that never started has no id")
	}
}

func TestAnInvalidPersistedMessageIsDropped(t *testing.T) {
	cp := newFakeCP(true)
	store := &memStore{state: QueueState{NextSeq: 1, Items: []QueuedMessage{{Seq: 1, Action: "MeterValues", Payload: []byte(`{"connectorId":"x"}`)}}}}
	var dropped []Discarded
	cfg := fastConfig()
	cfg.Store = store
	cfg.OnDiscard = func(d Discarded) { dropped = append(dropped, d) }
	q, err := newTransactionQueue(cp, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer runQueue(t, q)()
	waitUntil(t, "the message to be dropped", func() bool { return q.Len() == 0 })
	if len(dropped) != 1 || !errors.Is(dropped[0].Err, ocpp.ErrValidation) {
		t.Errorf("dropped %+v", dropped)
	}
	if len(cp.sent()) != 0 {
		t.Errorf("sent %v", cp.sent())
	}
	for _, action := range []string{"StartTransaction", "StopTransaction"} {
		store := &memStore{state: QueueState{NextSeq: 1, Items: []QueuedMessage{{Seq: 1, Action: action, Payload: []byte(`[]`)}}}}
		cfg.Store = store
		q, _ := newTransactionQueue(cp, cfg)
		stopFn := runQueue(t, q)
		waitUntil(t, action+" to be dropped", func() bool { return q.Len() == 0 })
		stopFn()
	}
}

func TestEnqueueRejectsInvalidMessagesAndAFullQueue(t *testing.T) {
	cp := newFakeCP(false)
	cfg := fastConfig()
	cfg.MaxQueued = 2
	q, _ := newTransactionQueue(cp, cfg)
	if _, err := q.StartTransaction(core.StartTransactionRequest{}); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("start: %v", err)
	}
	if err := q.StopTransaction(nil, core.StopTransactionRequest{}); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("stop: %v", err)
	}
	if err := q.MeterValues(nil, core.MeterValuesRequest{}); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("meter: %v", err)
	}
	if _, err := q.StartTransaction(start()); err != nil {
		t.Fatal(err)
	}
	if err := q.MeterValues(nil, meter()); err != nil {
		t.Fatal(err)
	}
	if _, err := q.StartTransaction(start()); !errors.Is(err, ocpp.ErrQueueFull) {
		t.Errorf("start on a full queue: %v", err)
	}
	if err := q.StopTransaction(nil, stop()); !errors.Is(err, ocpp.ErrQueueFull) {
		t.Errorf("stop on a full queue: %v", err)
	}
	if err := q.MeterValues(nil, meter()); !errors.Is(err, ocpp.ErrQueueFull) {
		t.Errorf("meter on a full queue: %v", err)
	}
}

func TestDefaults(t *testing.T) {
	q, err := newTransactionQueue(newFakeCP(false), TransactionQueueConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if q.attempts != DefaultTransactionMessageAttempts || q.interval != DefaultTransactionMessageRetryInterval || q.max != DefaultTransactionQueueSize {
		t.Errorf("%d %v %d", q.attempts, q.interval, q.max)
	}
	var _ = NewTransactionQueue // the exported constructor wraps a ChargePoint
}

func TestRunStopsWhenTheContextEnds(t *testing.T) {
	q, _ := newTransactionQueue(newFakeCP(false), fastConfig())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := q.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("idle: %v", err)
	}
	_ = q.MeterValues(nil, meter())
	if err := q.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("offline: %v", err)
	}
	// Cancelling during the wait before a retry.
	cp := newFakeCP(true)
	cp.failures["MeterValues"] = []error{ocpp.ErrTimeout}
	q, _ = newTransactionQueue(cp, TransactionQueueConfig{Attempts: 3, RetryInterval: time.Hour})
	_ = q.MeterValues(nil, meter())
	ctx, cancel = context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- q.Run(ctx) }()
	waitUntil(t, "the first attempt", func() bool { return len(cp.sent()) == 1 })
	time.Sleep(10 * time.Millisecond)
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Errorf("retry wait: %v", err)
	}
	// Cancelling while the call itself fails with the context error.
	cp = newFakeCP(true)
	cp.failures["MeterValues"] = []error{context.Canceled}
	q, _ = newTransactionQueue(cp, fastConfig())
	_ = q.MeterValues(nil, meter())
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	if err := q.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("call: %v", err)
	}
}

// memStore is a TransactionStore that can fail.
type memStore struct {
	state   QueueState
	loadErr error
	saveErr error
	saves   int
}

func (m *memStore) Load() (QueueState, error) { return m.state, m.loadErr }
func (m *memStore) Save(st QueueState) error {
	m.saves++
	if m.saveErr != nil {
		return m.saveErr
	}
	m.state = st
	return nil
}

func TestStoreFailures(t *testing.T) {
	boom := errors.New("disk full")
	cfg := fastConfig()
	cfg.Store = &memStore{loadErr: boom}
	if _, err := newTransactionQueue(newFakeCP(true), cfg); !errors.Is(err, boom) {
		t.Errorf("load: %v", err)
	}
	st := &memStore{saveErr: boom}
	cfg.Store = st
	q, err := newTransactionQueue(newFakeCP(false), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.StartTransaction(start()); !errors.Is(err, boom) {
		t.Errorf("start: %v", err)
	}
	if err := q.StopTransaction(nil, stop()); !errors.Is(err, boom) {
		t.Errorf("stop: %v", err)
	}
	if err := q.MeterValues(nil, meter()); !errors.Is(err, boom) {
		t.Errorf("meter: %v", err)
	}
	// A save that fails while delivering is only logged: the delivery goes on.
	cp := newFakeCP(true)
	cp.failures["MeterValues"] = []error{ocpp.ErrTimeout}
	q, _ = newTransactionQueue(cp, cfg)
	_ = q.MeterValues(nil, meter())
	defer runQueue(t, q)()
	waitUntil(t, "delivery", func() bool { return q.Len() == 0 })
}

func TestTheQueueSurvivesARestart(t *testing.T) {
	store := &memStore{}
	cfg := fastConfig()
	cfg.Store = store

	// First run: offline, so a whole transaction is queued.
	q, _ := newTransactionQueue(newFakeCP(false), cfg)
	tx, _ := q.StartTransaction(start())
	_ = q.MeterValues(tx, meter())
	local := tx.LocalID()
	if store.saves != 2 || len(store.state.Items) != 2 {
		t.Fatalf("saved %d times, %d items", store.saves, len(store.state.Items))
	}

	// Second run: the same store, now online. Only the first message is delivered
	// before the program "stops" again; its id must be remembered.
	cp := newFakeCP(true)
	q, err := newTransactionQueue(cp, cfg)
	if err != nil {
		t.Fatal(err)
	}
	tx2, ok := q.Transaction(local)
	if !ok {
		t.Fatal("the handle of a queued transaction is restored")
	}
	stopFn := runQueue(t, q)
	waitUntil(t, "delivery", func() bool { return q.Len() == 0 })
	stopFn()
	if id, ok := tx2.TransactionID(); !ok || id != 101 {
		t.Errorf("id %d %v", id, ok)
	}
	if tx2.IDTagInfo() == nil {
		t.Error("the answer of the Central System is kept for a live handle")
	}
	if got := cp.sent(); len(got) != 2 || got[0] != "StartTransaction" || got[1] != "MeterValues" {
		t.Errorf("sent %v", got)
	}

	// Third run: the transaction is known to the store; stopping it works with a restored handle.
	cp = newFakeCP(true)
	q, _ = newTransactionQueue(cp, cfg)
	tx3, ok := q.Transaction(local)
	if !ok {
		t.Fatal("a started transaction is restored")
	}
	if id, ok := tx3.TransactionID(); !ok || id != 101 {
		t.Errorf("restored id %d %v", id, ok)
	}
	if tx3.IDTagInfo() != nil {
		t.Error("a restored handle has no idTagInfo")
	}
	stopFn = runQueue(t, q)
	_ = q.StopTransaction(tx3, stop())
	waitUntil(t, "delivery", func() bool { return q.Len() == 0 })
	stopFn()
	cp.mu.Lock()
	ids := append([]int(nil), cp.stopIDs...)
	cp.mu.Unlock()
	if len(ids) != 1 || ids[0] != 101 {
		t.Errorf("stop ids %v", ids)
	}
	if len(store.state.Resolved) != 0 {
		t.Errorf("an ended transaction is forgotten: %v", store.state.Resolved)
	}
}

func TestFileStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue.json")
	fs := FileStore(path)
	if st, err := fs.Load(); err != nil || len(st.Items) != 0 {
		t.Fatalf("a missing file is an empty state: %+v %v", st, err)
	}
	want := QueueState{NextSeq: 3, NextLocal: 2, Items: []QueuedMessage{{Seq: 3, Action: "MeterValues", Local: 2, Payload: []byte(`{"a":1}`), Attempts: 1}}, Resolved: map[uint64]int{2: 55}}
	if err := fs.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := fs.Load()
	if err != nil || got.NextSeq != 3 || got.NextLocal != 2 || len(got.Items) != 1 || got.Items[0].Attempts != 1 || got.Resolved[2] != 55 {
		t.Errorf("%+v %v", got, err)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("the temporary file must be gone: %v", entries)
	}
	// A broken file is an error, not an empty queue.
	if err := os.WriteFile(path, []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Load(); err == nil {
		t.Error("a corrupt file must be reported")
	}
	// An unreadable path and an unwritable directory.
	if _, err := FileStore(t.TempDir()).Load(); err == nil {
		t.Error("a directory cannot be read as a file")
	}
	if err := FileStore(filepath.Join(t.TempDir(), "missing", "q.json")).Save(want); err == nil {
		t.Error("saving into a missing directory must fail")
	}
	// The queue works end to end on a file store.
	cfg := fastConfig()
	cfg.Store = FileStore(filepath.Join(t.TempDir(), "q.json"))
	cp := newFakeCP(true)
	q, err := newTransactionQueue(cp, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer runQueue(t, q)()
	_ = q.MeterValues(nil, meter())
	waitUntil(t, "delivery", func() bool { return q.Len() == 0 })
}

// An answer the Central System sends that cannot be used (it fails decoding or
// validation) is a failure to process the message: the message is retried, not
// dropped at once.
func TestAnUnusableAnswerIsRetried(t *testing.T) {
	cp := newFakeCP(true)
	bad := fmt.Errorf("ocppj: StartTransaction response: %w",
		&ocppj.DecodeError{Code: ocpp.OccurrenceConstraintViolation, Err: ocpp.Required("idTagInfo")})
	cp.failures["StartTransaction"] = []error{bad}
	var dropped []Discarded
	cfg := fastConfig()
	cfg.OnDiscard = func(d Discarded) { dropped = append(dropped, d) }
	q, _ := newTransactionQueue(cp, cfg)
	defer runQueue(t, q)()
	tx, _ := q.StartTransaction(start())
	_ = q.StopTransaction(tx, stop())
	waitUntil(t, "delivery", func() bool { return q.Len() == 0 })
	if len(dropped) != 0 {
		t.Errorf("dropped %+v", dropped)
	}
	if got := cp.sent(); len(got) != 3 || got[0] != "StartTransaction" || got[1] != "StartTransaction" || got[2] != "StopTransaction" {
		t.Errorf("sent %v, want the start twice and then the stop", got)
	}
	// After the last attempt an unusable answer drops the message like any failure.
	cp = newFakeCP(true)
	cp.failures["MeterValues"] = []error{bad, bad, bad}
	dropped = nil
	q, _ = newTransactionQueue(cp, cfg)
	defer runQueue(t, q)()
	_ = q.MeterValues(nil, meter())
	waitUntil(t, "the message to be dealt with", func() bool { return q.Len() == 0 })
	if len(dropped) != 1 || dropped[0].Attempts != 3 {
		t.Errorf("dropped %+v", dropped)
	}
}

func TestFileStoreSyncFailures(t *testing.T) {
	// A store in a directory that disappears cannot save, and leaves no temporary file.
	dir := filepath.Join(t.TempDir(), "gone")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	fs := FileStore(filepath.Join(dir, "q.json"))
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	if err := fs.Save(QueueState{}); err == nil {
		t.Error("saving into a removed directory must fail")
	}
	// Renaming onto a directory fails; the temporary file is removed.
	dir = t.TempDir()
	target := filepath.Join(dir, "q.json")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := FileStore(target).Save(QueueState{}); err == nil {
		t.Error("renaming onto a directory must fail")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("a temporary file was left behind: %v", entries)
	}
	if err := syncDir(filepath.Join(dir, "missing")); err == nil {
		t.Error("a missing directory cannot be synced")
	}
}
