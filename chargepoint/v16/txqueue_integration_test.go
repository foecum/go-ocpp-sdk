package chargepoint_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/foecum/go-ocpp-sdk/chargepoint/v16"
	"github.com/foecum/go-ocpp-sdk/csms/v16"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// A transaction that took place while the charge point was offline is delivered,
// in order and with the transactionId the Central System assigns, once the
// charge point connects.
func TestTransactionQueueDeliversAnOfflineTransaction(t *testing.T) {
	var mu sync.Mutex
	var got []string
	var stopID int
	var meterID *int
	h := okHandler()
	h.startTransaction = func(_ context.Context, _ *csms.ChargePoint, r *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
		mu.Lock()
		got = append(got, "StartTransaction")
		mu.Unlock()
		return &core.StartTransactionConfirmation{IDTagInfo: types.IDTagInfo{Status: types.AuthorizationStatusAccepted}, TransactionID: 77}, nil
	}
	h.meterValues = func(_ context.Context, _ *csms.ChargePoint, r *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
		mu.Lock()
		got, meterID = append(got, "MeterValues"), r.TransactionID
		mu.Unlock()
		return &core.MeterValuesConfirmation{}, nil
	}
	h.stopTransaction = func(_ context.Context, _ *csms.ChargePoint, r *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
		mu.Lock()
		got, stopID = append(got, "StopTransaction"), r.TransactionID
		mu.Unlock()
		return &core.StopTransactionConfirmation{}, nil
	}
	_, _, url := startCSMS(t, func(c *csms.Config) { c.Core = h })

	// The charge point exists but is not running: it is offline.
	cp, err := chargepoint.New(chargepoint.Config{URL: url, ID: "CS-OFFLINE", Core: okStation(), Reconnect: fastReconnect})
	if err != nil {
		t.Fatal(err)
	}
	q, err := chargepoint.NewTransactionQueue(cp, chargepoint.TransactionQueueConfig{Attempts: 3, RetryInterval: 5 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go func() { _ = q.Run(ctx) }()

	tx, err := q.StartTransaction(core.StartTransactionRequest{ConnectorID: 1, IDTag: "TAG", MeterStart: 0, Timestamp: types.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if err := q.MeterValues(tx, core.MeterValuesRequest{ConnectorID: 1, MeterValue: []types.MeterValue{{
		Timestamp: types.Now(), SampledValue: []types.SampledValue{{Value: "5"}}}}}); err != nil {
		t.Fatal(err)
	}
	if err := q.StopTransaction(tx, core.StopTransactionRequest{MeterStop: 10, Timestamp: types.Now()}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if q.Len() != 3 {
		t.Fatalf("%d queued while offline, want 3", q.Len())
	}

	go func() { _ = cp.Run(ctx) }()
	defer cp.Close()
	select {
	case <-tx.Started():
	case <-time.After(5 * time.Second):
		t.Fatal("the transaction was not started after connecting")
	}
	deadline := time.Now().Add(5 * time.Second)
	for q.Len() != 0 {
		if time.Now().After(deadline) {
			t.Fatalf("%d messages left", q.Len())
		}
		time.Sleep(time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 3 || got[0] != "StartTransaction" || got[1] != "MeterValues" || got[2] != "StopTransaction" {
		t.Errorf("delivered %v", got)
	}
	if stopID != 77 || meterID == nil || *meterID != 77 {
		t.Errorf("transaction ids: stop %d, meter values %v", stopID, meterID)
	}
	if id, ok := tx.TransactionID(); !ok || id != 77 {
		t.Errorf("handle id %d %v", id, ok)
	}
}
