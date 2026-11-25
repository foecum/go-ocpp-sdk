package chargepoint_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/chargepoint/v16"
	"github.com/foecum/go-ocpp-sdk/csms/v16"
	"github.com/foecum/go-ocpp-sdk/ocppj"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/types"
	"github.com/foecum/go-ocpp-sdk/ws"
)

func TestNewValidatesConfig(t *testing.T) {
	good := chargepoint.Config{URL: "ws://localhost/ocpp", ID: "CS1"}
	if _, err := chargepoint.New(good); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*chargepoint.Config){
		"no id":      func(c *chargepoint.Config) { c.ID = "" },
		"colon id":   func(c *chargepoint.Config) { c.ID = "a:b" },
		"bad scheme": func(c *chargepoint.Config) { c.URL = "http://localhost/ocpp" },
		"no url":     func(c *chargepoint.Config) { c.URL = "" },
	} {
		cfg := good
		mutate(&cfg)
		if _, err := chargepoint.New(cfg); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestNotConnectedBeforeRun(t *testing.T) {
	cp, err := chargepoint.New(chargepoint.Config{URL: "ws://127.0.0.1:1/ocpp", ID: "CS1"})
	if err != nil {
		t.Fatal(err)
	}
	if cp.Connected() {
		t.Error("connected before Run")
	}
	if _, err := cp.Heartbeat(context.Background(), core.HeartbeatRequest{}); !errors.Is(err, ocpp.ErrNotConnected) {
		t.Errorf("got %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if err := cp.WaitConnected(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("WaitConnected: %v", err)
	}
}

func TestRunCloseAndHooks(t *testing.T) {
	_, ev, url := startCSMS(t, nil)
	var connects, disconnects counter
	disc := make(chan error, 4)
	cp, err := chargepoint.New(chargepoint.Config{URL: url, ID: "CS1", Reconnect: fastReconnect,
		OnConnect:    connects.inc,
		OnDisconnect: func(err error) { disconnects.inc(); disc <- err }})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cp.Run(context.Background()) }()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := cp.WaitConnected(ctx); err != nil {
		t.Fatal(err)
	}
	if err := cp.WaitConnected(ctx); err != nil { // already connected: returns at once
		t.Fatal(err)
	}
	recv(t, ev.connected)
	if !cp.Connected() || connects.get() != 1 {
		t.Errorf("connected=%v connects=%d", cp.Connected(), connects.get())
	}
	if err := cp.Close(); err != nil {
		t.Fatal(err)
	}
	if err := recv(t, done); !errors.Is(err, ocpp.ErrClosed) {
		t.Errorf("Run returned %v", err)
	}
	if err := recv(t, disc); err != nil || disconnects.get() != 1 || cp.Connected() {
		t.Errorf("disconnect err=%v n=%d connected=%v", err, disconnects.get(), cp.Connected())
	}
	_ = cp.Close() // idempotent
	if _, err := cp.Authorize(context.Background(), core.AuthorizeRequest{IDTag: "x"}); !errors.Is(err, ocpp.ErrNotConnected) {
		t.Errorf("after close: %v", err)
	}
}

func TestRunStopsWhenContextEnds(t *testing.T) {
	_, _, url := startCSMS(t, nil)
	cp, err := chargepoint.New(chargepoint.Config{URL: url, ID: "CS1", Reconnect: fastReconnect})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- cp.Run(ctx) }()
	wctx, wcancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer wcancel()
	if err := cp.WaitConnected(wctx); err != nil {
		t.Fatal(err)
	}
	cancel()
	if err := recv(t, done); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v", err)
	}
}

// TestEveryChargePointCall exercises each typed call from the station.
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
		"StopTransaction": func() (any, error) { return cp.StopTransaction(ctx, core.StopTransactionRequest{Timestamp: now}) },
	}
	for name, f := range checks {
		if got, err := f(); err != nil || got == nil {
			t.Errorf("%s: %v %v", name, got, err)
		}
	}
	if len(checks) != 8 {
		t.Errorf("expected 8 calls, have %d", len(checks))
	}
	if _, err := cp.StartTransaction(ctx, core.StartTransactionRequest{ConnectorID: 0, IDTag: "x", Timestamp: now}); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("invalid request: %v", err)
	}
}

// TestEveryHandlerIsCalled checks the dispatch to each CoreHandler method.
func TestEveryHandlerIsCalled(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	note := func(s string) { mu.Lock(); seen = append(seen, s); mu.Unlock() }
	st := okStation()
	st.changeAvailability = func(_ context.Context, r *core.ChangeAvailabilityRequest) (*core.ChangeAvailabilityConfirmation, error) {
		note(fmt.Sprintf("ChangeAvailability:%d:%s", r.ConnectorID, r.Type))
		return &core.ChangeAvailabilityConfirmation{Status: types.AvailabilityStatusScheduled}, nil
	}
	st.changeConfiguration = func(_ context.Context, r *core.ChangeConfigurationRequest) (*core.ChangeConfigurationConfirmation, error) {
		note("ChangeConfiguration:" + r.Key + "=" + r.Value)
		return &core.ChangeConfigurationConfirmation{Status: types.ConfigurationStatusRebootRequired}, nil
	}
	st.clearCache = func(context.Context, *core.ClearCacheRequest) (*core.ClearCacheConfirmation, error) {
		note("ClearCache")
		return &core.ClearCacheConfirmation{Status: types.ClearCacheStatusAccepted}, nil
	}
	st.dataTransfer = func(_ context.Context, r *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
		note("DataTransfer:" + r.VendorID)
		return &core.DataTransferConfirmation{Status: types.DataTransferStatusAccepted}, nil
	}
	st.getConfiguration = func(_ context.Context, r *core.GetConfigurationRequest) (*core.GetConfigurationConfirmation, error) {
		note(fmt.Sprintf("GetConfiguration:%v", r.Key))
		return &core.GetConfigurationConfirmation{UnknownKey: r.Key}, nil
	}
	st.remoteStartTransaction = func(_ context.Context, r *core.RemoteStartTransactionRequest) (*core.RemoteStartTransactionConfirmation, error) {
		note("RemoteStartTransaction:" + r.IDTag)
		return &core.RemoteStartTransactionConfirmation{Status: types.RemoteStartStopStatusAccepted}, nil
	}
	st.remoteStopTransaction = func(_ context.Context, r *core.RemoteStopTransactionRequest) (*core.RemoteStopTransactionConfirmation, error) {
		note(fmt.Sprintf("RemoteStopTransaction:%d", r.TransactionID))
		return &core.RemoteStopTransactionConfirmation{Status: types.RemoteStartStopStatusRejected}, nil
	}
	st.reset = func(_ context.Context, r *core.ResetRequest) (*core.ResetConfirmation, error) {
		note("Reset:" + string(r.Type))
		return &core.ResetConfirmation{Status: types.ResetStatusAccepted}, nil
	}
	st.unlockConnector = func(_ context.Context, r *core.UnlockConnectorRequest) (*core.UnlockConnectorConfirmation, error) {
		note(fmt.Sprintf("UnlockConnector:%d", r.ConnectorID))
		return &core.UnlockConnectorConfirmation{Status: types.UnlockStatusUnlockFailed}, nil
	}
	srv, _, url := startCSMS(t, nil)
	startStation(t, url, "CS1", func(c *chargepoint.Config) { c.Core = st })
	cp := lookup(t, srv, "CS1")
	ctx := context.Background()

	if r, err := cp.ChangeAvailability(ctx, core.ChangeAvailabilityRequest{ConnectorID: 2, Type: types.AvailabilityTypeInoperative}); err != nil || r.Status != types.AvailabilityStatusScheduled {
		t.Errorf("%v %v", r, err)
	}
	if r, err := cp.ChangeConfiguration(ctx, core.ChangeConfigurationRequest{Key: "K", Value: "V"}); err != nil || r.Status != types.ConfigurationStatusRebootRequired {
		t.Errorf("%v %v", r, err)
	}
	if _, err := cp.ClearCache(ctx, core.ClearCacheRequest{}); err != nil {
		t.Error(err)
	}
	if _, err := cp.DataTransfer(ctx, core.DataTransferRequest{VendorID: "acme"}); err != nil {
		t.Error(err)
	}
	if r, err := cp.GetConfiguration(ctx, core.GetConfigurationRequest{Key: []string{"A"}}); err != nil || len(r.UnknownKey) != 1 {
		t.Errorf("%v %v", r, err)
	}
	if _, err := cp.RemoteStartTransaction(ctx, core.RemoteStartTransactionRequest{IDTag: "T"}); err != nil {
		t.Error(err)
	}
	if r, err := cp.RemoteStopTransaction(ctx, core.RemoteStopTransactionRequest{TransactionID: 9}); err != nil || r.Status != types.RemoteStartStopStatusRejected {
		t.Errorf("%v %v", r, err)
	}
	if _, err := cp.Reset(ctx, core.ResetRequest{Type: types.ResetTypeHard}); err != nil {
		t.Error(err)
	}
	if r, err := cp.UnlockConnector(ctx, core.UnlockConnectorRequest{ConnectorID: 3}); err != nil || r.Status != types.UnlockStatusUnlockFailed {
		t.Errorf("%v %v", r, err)
	}
	want := "[ChangeAvailability:2:Inoperative ChangeConfiguration:K=V ClearCache DataTransfer:acme GetConfiguration:[A] " +
		"RemoteStartTransaction:T RemoteStopTransaction:9 Reset:Hard UnlockConnector:3]"
	mu.Lock()
	defer mu.Unlock()
	if fmt.Sprint(seen) != want {
		t.Errorf("handlers saw\n%v\nwant\n%s", seen, want)
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
	for name, core0 := range map[string]chargepoint.CoreHandler{"nil handler": nil, "unimplemented embed": &station{}} {
		srv, _, url := startCSMS(t, nil)
		startStation(t, url, "CS1", func(c *chargepoint.Config) { c.Core = core0 })
		cp := lookup(t, srv, "CS1")
		ctx := context.Background()
		for call, f := range map[string]func() error{
			"ChangeAvailability": func() error {
				_, e := cp.ChangeAvailability(ctx, core.ChangeAvailabilityRequest{Type: types.AvailabilityTypeOperative})
				return e
			},
			"ChangeConfiguration": func() error { _, e := cp.ChangeConfiguration(ctx, core.ChangeConfigurationRequest{Key: "k"}); return e },
			"ClearCache":          func() error { _, e := cp.ClearCache(ctx, core.ClearCacheRequest{}); return e },
			"DataTransfer":        func() error { _, e := cp.DataTransfer(ctx, core.DataTransferRequest{VendorID: "v"}); return e },
			"GetConfiguration":    func() error { _, e := cp.GetConfiguration(ctx, core.GetConfigurationRequest{}); return e },
			"RemoteStartTransaction": func() error {
				_, e := cp.RemoteStartTransaction(ctx, core.RemoteStartTransactionRequest{IDTag: "x"})
				return e
			},
			"RemoteStopTransaction": func() error { _, e := cp.RemoteStopTransaction(ctx, core.RemoteStopTransactionRequest{}); return e },
			"Reset":                 func() error { _, e := cp.Reset(ctx, core.ResetRequest{Type: types.ResetTypeSoft}); return e },
			"UnlockConnector":       func() error { _, e := cp.UnlockConnector(ctx, core.UnlockConnectorRequest{ConnectorID: 1}); return e },
		} {
			if ce := callError(t, f()); ce.Code != ocpp.NotSupported {
				t.Errorf("%s / %s: %s", name, call, ce.Code)
			}
		}
	}
}

func TestHandlerErrorsBecomeCallErrors(t *testing.T) {
	var mode string
	st := okStation()
	st.reset = func(context.Context, *core.ResetRequest) (*core.ResetConfirmation, error) {
		switch mode {
		case "calllerror":
			return nil, &ocpp.CallError{Code: ocpp.GenericError, Description: "nope"}
		case "plain":
			return nil, errors.New("disk on fire")
		case "panic":
			panic("kaboom")
		}
		return nil, nil
	}
	srv, _, url := startCSMS(t, nil)
	cp0 := startStation(t, url, "CS1", func(c *chargepoint.Config) { c.Core = st })
	cp := lookup(t, srv, "CS1")
	reset := func() error {
		_, err := cp.Reset(context.Background(), core.ResetRequest{Type: types.ResetTypeSoft})
		return err
	}
	mode = "calllerror"
	if ce := callError(t, reset()); ce.Code != ocpp.GenericError || ce.Description != "nope" {
		t.Errorf("%+v", ce)
	}
	for _, m := range []string{"plain", "panic", "nil"} {
		mode = m
		if ce := callError(t, reset()); ce.Code != ocpp.InternalError || strings.Contains(ce.Description, "fire") || strings.Contains(ce.Description, "kaboom") {
			t.Errorf("%s: %+v", m, ce)
		}
	}
	if !cp0.Connected() {
		t.Error("the station must survive handler failures")
	}
}

// rawCSMS is a Central System that speaks raw frames, to check how a station
// answers messages a well-behaved CSMS would not send.
func rawCSMS(t *testing.T) (url string, conns <-chan *websocket.Conn) {
	t.Helper()
	ch := make(chan *websocket.Conn, 4)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"ocpp1.6"}})
		if err != nil {
			return
		}
		ch <- c
		<-r.Context().Done()
		_ = c.CloseNow()
	}))
	t.Cleanup(ts.Close)
	return "ws" + strings.TrimPrefix(ts.URL, "http") + "/ocpp", ch
}

func exchange(t *testing.T, c *websocket.Conn, frame string) *ocppj.CallErrorFrame {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := c.Write(ctx, websocket.MessageText, []byte(frame)); err != nil {
		t.Fatal(err)
	}
	_, data, err := c.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	m, err := ocppj.Parse(ocpp.V16, data)
	if err != nil {
		t.Fatal(err)
	}
	ce, ok := m.(*ocppj.CallErrorFrame)
	if !ok {
		t.Fatalf("expected a CALLERROR for %s, got %s", frame, data)
	}
	return ce
}

func TestStationAnswersBadFramesWithTheRightCodes(t *testing.T) {
	url, conns := rawCSMS(t)
	startStation(t, url, "CS1", nil)
	c := recv(t, conns)
	// Nobody reads on this connection after the test, so drop it before the
	// station's graceful close would wait for an acknowledgement.
	t.Cleanup(func() { _ = c.CloseNow() })
	tests := []struct {
		name, frame string
		id          string
		code        ocpp.ErrorCode
	}{
		{"unknown action", `[2,"b1","MakeCoffee",{}]`, "b1", ocpp.NotImplemented},
		{"known but a charge point action", `[2,"b2","BootNotification",{"chargePointVendor":"v","chargePointModel":"m"}]`, "b2", ocpp.NotSupported},
		{"missing required field", `[2,"b3","Reset",{}]`, "b3", ocpp.OccurenceConstraintViolation},
		{"wrong type", `[2,"b4","UnlockConnector",{"connectorId":"one"}]`, "b4", ocpp.TypeConstraintViolation},
		{"bad enum", `[2,"b5","Reset",{"type":"Lukewarm"}]`, "b5", ocpp.PropertyConstraintViolation},
		{"connector 0 for unlock", `[2,"b6","UnlockConnector",{"connectorId":0}]`, "b6", ocpp.PropertyConstraintViolation},
		{"not JSON", `nope`, "-1", ocpp.FormationViolation},
	}
	for _, tt := range tests {
		ce := exchange(t, c, tt.frame)
		if ce.UniqueID != tt.id || ce.Code != tt.code {
			t.Errorf("%s: got id=%q code=%q, want id=%q code=%q", tt.name, ce.UniqueID, ce.Code, tt.id, tt.code)
		}
	}
}

func TestCallTimeout(t *testing.T) {
	block := make(chan struct{})
	t.Cleanup(func() { close(block) })
	h := okHandler()
	h.heartbeat = func(ctx context.Context, _ *csms.ChargePoint, _ *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
		select {
		case <-block:
		case <-ctx.Done():
		}
		return &core.HeartbeatConfirmation{CurrentTime: types.Now()}, nil
	}
	_, _, url := startCSMS(t, func(c *csms.Config) { c.Core = h })
	cp := startStation(t, url, "CS1", func(c *chargepoint.Config) { c.CallTimeout = 50 * time.Millisecond })
	start := time.Now()
	_, err := cp.Heartbeat(context.Background(), core.HeartbeatRequest{})
	if !errors.Is(err, ocpp.ErrTimeout) || time.Since(start) > 2*time.Second {
		t.Errorf("got %v after %v", err, time.Since(start))
	}
}

func TestCallContextCancellation(t *testing.T) {
	block := make(chan struct{})
	t.Cleanup(func() { close(block) })
	h := okHandler()
	h.heartbeat = func(ctx context.Context, _ *csms.ChargePoint, _ *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
		select {
		case <-block:
		case <-ctx.Done():
		}
		return nil, errors.New("cancelled")
	}
	_, _, url := startCSMS(t, func(c *csms.Config) { c.Core = h })
	cp := startStation(t, url, "CS1", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := cp.Heartbeat(ctx, core.HeartbeatRequest{}); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got %v", err)
	}
}

func TestReconnectAfterServerRestartKeepsHooksInOrder(t *testing.T) {
	srv, ev, url := startCSMS(t, nil)
	var mu sync.Mutex
	var log []string
	add := func(s string) { mu.Lock(); log = append(log, s); mu.Unlock() }
	cp := startStation(t, url, "CS1", func(c *chargepoint.Config) {
		c.OnConnect = func() { add("up") }
		c.OnDisconnect = func(error) { add("down") }
	})
	recv(t, ev.connected)
	_ = lookup(t, srv, "CS1").Close()
	recv(t, ev.disconnected)
	recv(t, ev.connected) // reconnected
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := cp.WaitConnected(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := cp.Heartbeat(context.Background(), core.HeartbeatRequest{}); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if fmt.Sprint(log) != "[up down up]" {
		t.Errorf("hooks: %v", log)
	}
}

func TestTLSAndAuthPassThrough(t *testing.T) {
	_, ev, url := startCSMS(t, func(c *csms.Config) {
		c.Authenticator = ws.BasicAuthenticator(func(id string) (string, bool) { return "correct-password-16", true })
	})
	startStation(t, url, "CS1", func(c *chargepoint.Config) { c.Password = "correct-password-16" })
	recv(t, ev.connected)
}
