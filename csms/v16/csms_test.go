package csms_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/chargepoint/v16"
	"github.com/foecum/go-ocpp-sdk/csms/v16"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/types"
	"github.com/foecum/go-ocpp-sdk/ws"
)

// TestTransactionFlow runs a charging session end to end over a real WebSocket:
// the station boots, reports status, authorises, starts a transaction, sends
// meter values and stops; the CSMS then remote-starts and resets the station.
func TestTransactionFlow(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	note := func(s string) { mu.Lock(); seen = append(seen, s); mu.Unlock() }

	h := okHandler()
	h.boot = func(_ context.Context, cp *csms.ChargePoint, r *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
		note("boot:" + cp.ID() + ":" + r.ChargePointVendor)
		return &core.BootNotificationConfirmation{Status: types.RegistrationStatusAccepted, CurrentTime: types.Now(), Interval: 300}, nil
	}
	h.statusNotification = func(_ context.Context, _ *csms.ChargePoint, r *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
		note("status:" + string(r.Status))
		return &core.StatusNotificationConfirmation{}, nil
	}
	h.authorize = func(_ context.Context, _ *csms.ChargePoint, r *core.AuthorizeRequest) (*core.AuthorizeConfirmation, error) {
		note("authorize:" + r.IDTag)
		return &core.AuthorizeConfirmation{IDTagInfo: types.IDTagInfo{Status: types.AuthorizationStatusAccepted}}, nil
	}
	h.startTransaction = func(_ context.Context, _ *csms.ChargePoint, r *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
		note(fmt.Sprintf("start:%d:%d", r.ConnectorID, r.MeterStart))
		return &core.StartTransactionConfirmation{TransactionID: 1001, IDTagInfo: types.IDTagInfo{Status: types.AuthorizationStatusAccepted}}, nil
	}
	h.meterValues = func(_ context.Context, _ *csms.ChargePoint, r *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
		note(fmt.Sprintf("meter:%d:%s", *r.TransactionID, r.MeterValue[0].SampledValue[0].Value))
		return &core.MeterValuesConfirmation{}, nil
	}
	h.stopTransaction = func(_ context.Context, _ *csms.ChargePoint, r *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
		note(fmt.Sprintf("stop:%d:%d:%s", r.TransactionID, r.MeterStop, r.Reason))
		return &core.StopTransactionConfirmation{}, nil
	}
	srv, _, url := startCSMS(t, func(c *csms.Config) { c.Core = h })

	remoteStarted := make(chan string, 1)
	resetKind := make(chan types.ResetType, 1)
	st := okStation()
	st.remoteStartTransaction = func(_ context.Context, r *core.RemoteStartTransactionRequest) (*core.RemoteStartTransactionConfirmation, error) {
		remoteStarted <- r.IDTag
		return &core.RemoteStartTransactionConfirmation{Status: types.RemoteStartStopStatusAccepted}, nil
	}
	st.reset = func(_ context.Context, r *core.ResetRequest) (*core.ResetConfirmation, error) {
		resetKind <- r.Type
		return &core.ResetConfirmation{Status: types.ResetStatusAccepted}, nil
	}
	cp := startStation(t, url, "CS001", func(c *chargepoint.Config) { c.Core = st })
	ctx := context.Background()

	boot, err := cp.BootNotification(ctx, core.BootNotificationRequest{ChargePointVendor: "Acme", ChargePointModel: "X1"})
	if err != nil || boot.Status != types.RegistrationStatusAccepted || boot.Interval != 300 {
		t.Fatalf("boot: %v %v", boot, err)
	}
	if _, err := cp.StatusNotification(ctx, core.StatusNotificationRequest{ConnectorID: 1, ErrorCode: types.ChargePointErrorCodeNoError, Status: types.ChargePointStatusAvailable}); err != nil {
		t.Fatal(err)
	}
	auth, err := cp.Authorize(ctx, core.AuthorizeRequest{IDTag: "04A1B2C3"})
	if err != nil || auth.IDTagInfo.Status != types.AuthorizationStatusAccepted {
		t.Fatalf("authorize: %v %v", auth, err)
	}
	start, err := cp.StartTransaction(ctx, core.StartTransactionRequest{ConnectorID: 1, IDTag: "04A1B2C3", MeterStart: 100, Timestamp: types.Now()})
	if err != nil || start.TransactionID != 1001 {
		t.Fatalf("start: %v %v", start, err)
	}
	tx := start.TransactionID
	if _, err := cp.MeterValues(ctx, core.MeterValuesRequest{ConnectorID: 1, TransactionID: &tx, MeterValue: []types.MeterValue{{
		Timestamp: types.Now(), SampledValue: []types.SampledValue{{Value: "250", Measurand: types.MeasurandEnergyActiveImportRegister}}}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := cp.StopTransaction(ctx, core.StopTransactionRequest{TransactionID: tx, MeterStop: 900, Timestamp: types.Now(), Reason: types.ReasonLocal}); err != nil {
		t.Fatal(err)
	}

	want := []string{"boot:CS001:Acme", "status:Available", "authorize:04A1B2C3", "start:1:100", "meter:1001:250", "stop:1001:900:Local"}
	mu.Lock()
	got := append([]string(nil), seen...)
	mu.Unlock()
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("CSMS saw %v, want %v", got, want)
	}

	// Now the CSMS drives the station.
	handle := lookup(t, srv, "CS001")
	rs, err := handle.RemoteStartTransaction(ctx, core.RemoteStartTransactionRequest{IDTag: "REMOTE01"})
	if err != nil || rs.Status != types.RemoteStartStopStatusAccepted || <-remoteStarted != "REMOTE01" {
		t.Fatalf("remote start: %v %v", rs, err)
	}
	rst, err := handle.Reset(ctx, core.ResetRequest{Type: types.ResetTypeSoft})
	if err != nil || rst.Status != types.ResetStatusAccepted || <-resetKind != types.ResetTypeSoft {
		t.Fatalf("reset: %v %v", rst, err)
	}
}

// TestEveryCSMSCall exercises each typed call from the CSMS to a station.
func TestEveryCSMSCall(t *testing.T) {
	srv, _, url := startCSMS(t, nil)
	startStation(t, url, "CS1", nil)
	cp := lookup(t, srv, "CS1")
	ctx := context.Background()

	checks := map[string]func() (any, error){
		"ChangeAvailability": func() (any, error) {
			return cp.ChangeAvailability(ctx, core.ChangeAvailabilityRequest{ConnectorID: 0, Type: types.AvailabilityTypeOperative})
		},
		"ChangeConfiguration": func() (any, error) {
			return cp.ChangeConfiguration(ctx, core.ChangeConfigurationRequest{Key: "k", Value: "v"})
		},
		"ClearCache":   func() (any, error) { return cp.ClearCache(ctx, core.ClearCacheRequest{}) },
		"DataTransfer": func() (any, error) { return cp.DataTransfer(ctx, core.DataTransferRequest{VendorID: "v"}) },
		"GetConfiguration": func() (any, error) {
			return cp.GetConfiguration(ctx, core.GetConfigurationRequest{Key: []string{"HeartbeatInterval"}})
		},
		"RemoteStartTransaction": func() (any, error) {
			return cp.RemoteStartTransaction(ctx, core.RemoteStartTransactionRequest{IDTag: "x"})
		},
		"RemoteStopTransaction": func() (any, error) {
			return cp.RemoteStopTransaction(ctx, core.RemoteStopTransactionRequest{TransactionID: 5})
		},
		"Reset":           func() (any, error) { return cp.Reset(ctx, core.ResetRequest{Type: types.ResetTypeHard}) },
		"UnlockConnector": func() (any, error) { return cp.UnlockConnector(ctx, core.UnlockConnectorRequest{ConnectorID: 1}) },
	}
	for name, f := range checks {
		got, err := f()
		if err != nil || got == nil {
			t.Errorf("%s: %v %v", name, got, err)
		}
	}
	if len(checks) != 9 {
		t.Errorf("expected 9 calls, have %d", len(checks))
	}
	// Invalid requests never leave the CSMS.
	if _, err := cp.UnlockConnector(ctx, core.UnlockConnectorRequest{ConnectorID: 0}); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("got %v", err)
	}
}

// TestEveryChargePointCall exercises each handler on the CSMS side.
func TestEveryChargePointCall(t *testing.T) {
	_, _, url := startCSMS(t, nil)
	cp := startStation(t, url, "CS1", nil)
	ctx := context.Background()
	now := types.Now()

	checks := map[string]func() (any, error){
		"Authorize": func() (any, error) { return cp.Authorize(ctx, core.AuthorizeRequest{IDTag: "x"}) },
		"BootNotification": func() (any, error) {
			return cp.BootNotification(ctx, core.BootNotificationRequest{ChargePointVendor: "v", ChargePointModel: "m"})
		},
		"DataTransfer": func() (any, error) { return cp.DataTransfer(ctx, core.DataTransferRequest{VendorID: "v"}) },
		"Heartbeat":    func() (any, error) { return cp.Heartbeat(ctx, core.HeartbeatRequest{}) },
		"MeterValues": func() (any, error) {
			return cp.MeterValues(ctx, core.MeterValuesRequest{ConnectorID: 1, MeterValue: []types.MeterValue{{Timestamp: now, SampledValue: []types.SampledValue{{Value: "1"}}}}})
		},
		"StartTransaction": func() (any, error) {
			return cp.StartTransaction(ctx, core.StartTransactionRequest{ConnectorID: 1, IDTag: "x", Timestamp: now})
		},
		"StatusNotification": func() (any, error) {
			return cp.StatusNotification(ctx, core.StatusNotificationRequest{ErrorCode: types.ChargePointErrorCodeNoError, Status: types.ChargePointStatusAvailable})
		},
		"StopTransaction": func() (any, error) {
			return cp.StopTransaction(ctx, core.StopTransactionRequest{Timestamp: now})
		},
	}
	for name, f := range checks {
		got, err := f()
		if err != nil || got == nil {
			t.Errorf("%s: %v %v", name, got, err)
		}
	}
	if len(checks) != 8 {
		t.Errorf("expected 8 calls, have %d", len(checks))
	}
}

func callError(t *testing.T, err error) *ocpp.CallError {
	t.Helper()
	var ce *ocpp.CallError
	if !errors.As(err, &ce) {
		t.Fatalf("expected a CALLERROR, got %v", err)
	}
	return ce
}

func TestNotSupportedWhenNoHandler(t *testing.T) {
	// No Core handler at all.
	_, _, url := startCSMS(t, func(c *csms.Config) { c.Core = nil })
	cp := startStation(t, url, "CS1", nil)
	_, err := cp.BootNotification(context.Background(), core.BootNotificationRequest{ChargePointVendor: "v", ChargePointModel: "m"})
	if ce := callError(t, err); ce.Code != ocpp.NotSupported {
		t.Errorf("nil handler: %s", ce.Code)
	}

	// A handler that embeds the Unimplemented one and implements nothing.
	_, _, url2 := startCSMS(t, func(c *csms.Config) { c.Core = &handler{} })
	cp2 := startStation(t, url2, "CS2", nil)
	ctx := context.Background()
	now := types.Now()
	for name, f := range map[string]func() error{
		"Authorize": func() error { _, e := cp2.Authorize(ctx, core.AuthorizeRequest{IDTag: "x"}); return e },
		"BootNotification": func() error {
			_, e := cp2.BootNotification(ctx, core.BootNotificationRequest{ChargePointVendor: "v", ChargePointModel: "m"})
			return e
		},
		"DataTransfer": func() error { _, e := cp2.DataTransfer(ctx, core.DataTransferRequest{VendorID: "v"}); return e },
		"Heartbeat":    func() error { _, e := cp2.Heartbeat(ctx, core.HeartbeatRequest{}); return e },
		"MeterValues": func() error {
			_, e := cp2.MeterValues(ctx, core.MeterValuesRequest{ConnectorID: 1, MeterValue: []types.MeterValue{{Timestamp: now, SampledValue: []types.SampledValue{{Value: "1"}}}}})
			return e
		},
		"StartTransaction": func() error {
			_, e := cp2.StartTransaction(ctx, core.StartTransactionRequest{ConnectorID: 1, IDTag: "x", Timestamp: now})
			return e
		},
		"StatusNotification": func() error {
			_, e := cp2.StatusNotification(ctx, core.StatusNotificationRequest{ErrorCode: types.ChargePointErrorCodeNoError, Status: types.ChargePointStatusAvailable})
			return e
		},
		"StopTransaction": func() error { _, e := cp2.StopTransaction(ctx, core.StopTransactionRequest{Timestamp: now}); return e },
	} {
		if ce := callError(t, f()); ce.Code != ocpp.NotSupported {
			t.Errorf("%s: %s", name, ce.Code)
		}
	}
}

func TestHandlerErrorsBecomeCallErrors(t *testing.T) {
	h := okHandler()
	var mode string
	h.boot = func(context.Context, *csms.ChargePoint, *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
		switch mode {
		case "calllerror":
			return nil, &ocpp.CallError{Code: ocpp.SecurityError, Description: "denied", Details: []byte(`{"why":"banned"}`)}
		case "plain":
			return nil, errors.New("database is down")
		case "panic":
			panic("kaboom")
		}
		return nil, nil // nil confirmation, nil error
	}
	_, _, url := startCSMS(t, func(c *csms.Config) { c.Core = h })
	cp := startStation(t, url, "CS1", nil)
	boot := func() error {
		_, err := cp.BootNotification(context.Background(), core.BootNotificationRequest{ChargePointVendor: "v", ChargePointModel: "m"})
		return err
	}
	mode = "calllerror"
	if ce := callError(t, boot()); ce.Code != ocpp.SecurityError || ce.Description != "denied" || string(ce.Details) != `{"why":"banned"}` {
		t.Errorf("%+v", ce)
	}
	mode = "plain"
	ce := callError(t, boot())
	if ce.Code != ocpp.InternalError || ce.Description == "database is down" {
		t.Errorf("plain error must be InternalError without leaking its text: %+v", ce)
	}
	mode = "panic"
	if ce := callError(t, boot()); ce.Code != ocpp.InternalError {
		t.Errorf("panic: %+v", ce)
	}
	mode = "nil"
	if ce := callError(t, boot()); ce.Code != ocpp.InternalError {
		t.Errorf("nil confirmation: %+v", ce)
	}
	// The connection survived all of that.
	mode = "calllerror"
	_ = boot()
	if !cp.Connected() {
		t.Error("station disconnected")
	}
}

func TestCallTimeout(t *testing.T) {
	block := make(chan struct{})
	t.Cleanup(func() { close(block) })
	st := okStation()
	st.reset = func(ctx context.Context, _ *core.ResetRequest) (*core.ResetConfirmation, error) {
		select {
		case <-block:
		case <-ctx.Done():
		}
		return &core.ResetConfirmation{Status: types.ResetStatusAccepted}, nil
	}
	srv, _, url := startCSMS(t, func(c *csms.Config) { c.CallTimeout = 50 * time.Millisecond })
	startStation(t, url, "CS1", func(c *chargepoint.Config) { c.Core = st })
	cp := lookup(t, srv, "CS1")
	start := time.Now()
	_, err := cp.Reset(context.Background(), core.ResetRequest{Type: types.ResetTypeSoft})
	if !errors.Is(err, ocpp.ErrTimeout) || time.Since(start) > 2*time.Second {
		t.Errorf("got %v after %v", err, time.Since(start))
	}
}

func TestDisconnectDuringCall(t *testing.T) {
	// CSMS -> station call interrupted when the station goes away.
	entered := make(chan struct{})
	st := okStation()
	st.reset = func(ctx context.Context, _ *core.ResetRequest) (*core.ResetConfirmation, error) {
		close(entered)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	srv, ev, url := startCSMS(t, nil) // default 30s call timeout: the error must come much sooner
	station := startStation(t, url, "CS1", func(c *chargepoint.Config) { c.Core = st })
	cp := lookup(t, srv, "CS1")
	done := make(chan error, 1)
	go func() {
		_, err := cp.Reset(context.Background(), core.ResetRequest{Type: types.ResetTypeSoft})
		done <- err
	}()
	<-entered
	start := time.Now()
	_ = station.Close()
	err := recv(t, done)
	if !errors.Is(err, ocpp.ErrNotConnected) || time.Since(start) > 3*time.Second {
		t.Errorf("got %v after %v", err, time.Since(start))
	}
	recv(t, ev.disconnected)

	// Station -> CSMS call interrupted when the CSMS drops the connection.
	entered2 := make(chan struct{})
	h := okHandler()
	h.heartbeat = func(ctx context.Context, _ *csms.ChargePoint, _ *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
		close(entered2)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	srv2, _, url2 := startCSMS(t, func(c *csms.Config) { c.Core = h })
	station2 := startStation(t, url2, "CS2", func(c *chargepoint.Config) { c.Reconnect = ws.Reconnect{WaitMinimum: time.Hour} })
	done2 := make(chan error, 1)
	go func() {
		_, err := station2.Heartbeat(context.Background(), core.HeartbeatRequest{})
		done2 <- err
	}()
	<-entered2
	_ = lookup(t, srv2, "CS2").Close()
	if err := recv(t, done2); !errors.Is(err, ocpp.ErrNotConnected) {
		t.Errorf("station side: %v", err)
	}
}

// TestReconnectAndResend drops a station's connection, checks calls fail fast
// while it is down, then retries after it has reconnected.
func TestReconnectAndResend(t *testing.T) {
	var beats counter
	h := okHandler()
	h.heartbeat = func(context.Context, *csms.ChargePoint, *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
		beats.inc()
		return &core.HeartbeatConfirmation{CurrentTime: types.Now()}, nil
	}
	srv, ev, url := startCSMS(t, func(c *csms.Config) { c.Core = h })
	var connects, disconnects counter
	station := startStation(t, url, "CS1", func(c *chargepoint.Config) {
		c.Reconnect = ws.Reconnect{WaitMinimum: 300 * time.Millisecond}
		c.OnConnect = connects.inc
		c.OnDisconnect = func(error) { disconnects.inc() }
	})
	recv(t, ev.connected)
	ctx := context.Background()
	if _, err := station.Heartbeat(ctx, core.HeartbeatRequest{}); err != nil {
		t.Fatal(err)
	}

	_ = lookup(t, srv, "CS1").Close() // the CSMS drops the connection
	recv(t, ev.disconnected)
	deadline := time.Now().Add(2 * time.Second)
	for station.Connected() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if station.Connected() {
		t.Fatal("station still thinks it is connected")
	}
	// Down: calls fail at once and nothing is queued.
	start := time.Now()
	if _, err := station.Heartbeat(ctx, core.HeartbeatRequest{}); !errors.Is(err, ocpp.ErrNotConnected) || time.Since(start) > time.Second {
		t.Errorf("while down: %v after %v", err, time.Since(start))
	}
	if beats.get() != 1 {
		t.Errorf("the failed call must not have reached the CSMS (%d heartbeats)", beats.get())
	}
	// Reconnected: the application resends and it goes through.
	wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := station.WaitConnected(wctx); err != nil {
		t.Fatal(err)
	}
	recv(t, ev.connected)
	if _, err := station.Heartbeat(ctx, core.HeartbeatRequest{}); err != nil {
		t.Fatalf("after reconnect: %v", err)
	}
	if beats.get() != 2 || connects.get() != 2 || disconnects.get() != 1 {
		t.Errorf("heartbeats=%d connects=%d disconnects=%d", beats.get(), connects.get(), disconnects.get())
	}
}

func TestDuplicateIdentityReplacesOldConnection(t *testing.T) {
	srv, ev, url := startCSMS(t, nil)
	first := startStation(t, url, "CS1", func(c *chargepoint.Config) { c.Reconnect = ws.Reconnect{WaitMinimum: time.Hour} })
	recv(t, ev.connected)
	oldHandle := lookup(t, srv, "CS1")

	startStation(t, url, "CS1", nil)
	recv(t, ev.connected)
	d := recv(t, ev.disconnected)
	if d.cp != oldHandle || !errors.Is(d.err, ws.ErrReplaced) {
		t.Errorf("old connection ended with %v", d.err)
	}
	if got := srv.ChargePointIDs(); len(got) != 1 || got[0] != "CS1" {
		t.Errorf("connected: %v", got)
	}
	// The old handle is dead, the old station knows it.
	<-oldHandle.Done()
	if _, err := oldHandle.Reset(context.Background(), core.ResetRequest{Type: types.ResetTypeSoft}); !errors.Is(err, ocpp.ErrNotConnected) {
		t.Errorf("old handle: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for first.Connected() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if first.Connected() {
		t.Error("the replaced station should be disconnected")
	}
}

func TestChargePointHandleAndListing(t *testing.T) {
	srv, ev, url := startCSMS(t, nil)
	if _, ok := srv.ChargePoint("nope"); ok {
		t.Error("unknown charge point found")
	}
	for _, id := range []string{"B", "A", "C"} {
		startStation(t, url, id, nil)
		recv(t, ev.connected)
	}
	var each []string
	srv.ForEach(func(cp *csms.ChargePoint) bool { each = append(each, cp.ID()); return true })
	if fmt.Sprint(each) != "[A B C]" {
		t.Errorf("ForEach visited %v", each)
	}
	if got := srv.ChargePointIDs(); fmt.Sprint(got) != "[A B C]" {
		t.Errorf("ChargePointIDs() = %v", got)
	}
	cp := lookup(t, srv, "B")
	if cp.ID() != "B" || cp.Version() != ocpp.V16 || cp.RemoteAddr() == "" {
		t.Errorf("%q %q %q", cp.ID(), cp.Version(), cp.RemoteAddr())
	}
	select {
	case <-cp.Done():
		t.Error("Done closed while connected")
	default:
	}
	if err := cp.Close(); err != nil {
		t.Fatal(err)
	}
	<-cp.Done()
	if d := recv(t, ev.disconnected); d.cp.ID() != "B" || d.err != nil {
		t.Errorf("disconnect: %+v", d)
	}
}

func TestBasicAuthentication(t *testing.T) {
	_, ev, url := startCSMS(t, func(c *csms.Config) {
		c.Authenticator = ws.BasicAuthenticator(func(id string) (string, bool) { return "correct-password-16", id == "CS1" })
	})
	bad, err := chargepoint.New(chargepoint.Config{URL: url, ID: "CS1", Password: "wrong-password-0000", Reconnect: fastReconnect})
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = bad.Run(context.Background()) }()
	t.Cleanup(func() { _ = bad.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	if err := bad.WaitConnected(ctx); err == nil {
		t.Error("wrong password must not connect")
	}
	startStation(t, url, "CS1", func(c *chargepoint.Config) { c.Password = "correct-password-16" })
	recv(t, ev.connected)
}

func TestConcurrentStations(t *testing.T) {
	const stations = 50
	var boots, beats counter
	h := okHandler()
	h.boot = func(_ context.Context, cp *csms.ChargePoint, _ *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
		boots.inc()
		return &core.BootNotificationConfirmation{Status: types.RegistrationStatusAccepted, CurrentTime: types.Now(), Interval: 300}, nil
	}
	h.heartbeat = func(context.Context, *csms.ChargePoint, *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
		beats.inc()
		return &core.HeartbeatConfirmation{CurrentTime: types.Now()}, nil
	}
	srv, _, url := startCSMS(t, func(c *csms.Config) { c.Core = h })
	var wg sync.WaitGroup
	for i := 0; i < stations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cp := startStation(t, url, fmt.Sprintf("CS%03d", i), nil)
			if _, err := cp.BootNotification(context.Background(), core.BootNotificationRequest{ChargePointVendor: "v", ChargePointModel: "m"}); err != nil {
				t.Errorf("station %d: %v", i, err)
				return
			}
			for j := 0; j < 5; j++ {
				if _, err := cp.Heartbeat(context.Background(), core.HeartbeatRequest{}); err != nil {
					t.Errorf("station %d heartbeat: %v", i, err)
				}
			}
		}()
	}
	wg.Wait()
	if boots.get() != stations || beats.get() != stations*5 || len(srv.ChargePointIDs()) != stations {
		t.Errorf("boots=%d beats=%d connected=%d", boots.get(), beats.get(), len(srv.ChargePointIDs()))
	}
	// The CSMS can address each one.
	for i := 0; i < stations; i += 10 {
		if _, err := lookup(t, srv, fmt.Sprintf("CS%03d", i)).Reset(context.Background(), core.ResetRequest{Type: types.ResetTypeSoft}); err != nil {
			t.Errorf("reset %d: %v", i, err)
		}
	}
}

// TestHandlerCanCallBack: a handler uses its ChargePoint to call the station
// while still handling the station's request.
func TestHandlerCanCallBack(t *testing.T) {
	keys := make(chan string, 1)
	h := okHandler()
	h.boot = func(ctx context.Context, cp *csms.ChargePoint, _ *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
		conf, err := cp.GetConfiguration(ctx, core.GetConfigurationRequest{})
		if err != nil {
			return nil, err
		}
		keys <- conf.ConfigurationKey[0].Key
		return &core.BootNotificationConfirmation{Status: types.RegistrationStatusAccepted, CurrentTime: types.Now(), Interval: 60}, nil
	}
	_, _, url := startCSMS(t, func(c *csms.Config) { c.Core = h })
	cp := startStation(t, url, "CS1", nil)
	if _, err := cp.BootNotification(context.Background(), core.BootNotificationRequest{ChargePointVendor: "v", ChargePointModel: "m"}); err != nil {
		t.Fatal(err)
	}
	if got := recv(t, keys); got != "HeartbeatInterval" {
		t.Errorf("got %q", got)
	}
}

func TestServeAndShutdown(t *testing.T) {
	ev := newEvents()
	srv, err := csms.New(csms.Config{Core: okHandler(), OnDisconnect: func(cp *csms.ChargePoint, err error) { ev.disconnected <- disconnect{cp, err} }})
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	startStation(t, "ws://"+ln.Addr().String()+"/ocpp", "CS1", func(c *chargepoint.Config) { c.Reconnect = ws.Reconnect{WaitMinimum: time.Hour} })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if err := recv(t, serveErr); !errors.Is(err, http.ErrServerClosed) {
		t.Errorf("Serve returned %v", err)
	}
	if d := recv(t, ev.disconnected); d.err != nil {
		t.Errorf("shutdown should be a clean close: %v", d.err)
	}
	if err := srv.ListenAndServe("127.0.0.1:99999"); err == nil {
		t.Error("a bad address must fail")
	}
}

func TestListenAndServeUntilShutdown(t *testing.T) {
	srv, err := csms.New(csms.Config{})
	if err != nil {
		t.Fatal(err)
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe("127.0.0.1:0") }()
	time.Sleep(50 * time.Millisecond)
	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := recv(t, errc); !errors.Is(err, http.ErrServerClosed) {
		t.Errorf("got %v", err)
	}
}
