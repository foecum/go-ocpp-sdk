package csms_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/foecum/go-ocpp-sdk/chargepoint/v16"
	"github.com/foecum/go-ocpp-sdk/csms/v16"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/types"
	"github.com/foecum/go-ocpp-sdk/ws"
)

// handler is a csms.CoreHandler built from optional functions. Methods without
// a function fall back to the embedded NotSupported implementation.
type handler struct {
	csms.UnimplementedCoreHandler
	authorize          func(context.Context, *csms.ChargePoint, *core.AuthorizeRequest) (*core.AuthorizeConfirmation, error)
	boot               func(context.Context, *csms.ChargePoint, *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error)
	dataTransfer       func(context.Context, *csms.ChargePoint, *core.DataTransferRequest) (*core.DataTransferConfirmation, error)
	heartbeat          func(context.Context, *csms.ChargePoint, *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error)
	meterValues        func(context.Context, *csms.ChargePoint, *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error)
	startTransaction   func(context.Context, *csms.ChargePoint, *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error)
	statusNotification func(context.Context, *csms.ChargePoint, *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error)
	stopTransaction    func(context.Context, *csms.ChargePoint, *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error)
}

func (h *handler) Authorize(ctx context.Context, cp *csms.ChargePoint, r *core.AuthorizeRequest) (*core.AuthorizeConfirmation, error) {
	if h.authorize != nil {
		return h.authorize(ctx, cp, r)
	}
	return h.UnimplementedCoreHandler.Authorize(ctx, cp, r)
}
func (h *handler) BootNotification(ctx context.Context, cp *csms.ChargePoint, r *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
	if h.boot != nil {
		return h.boot(ctx, cp, r)
	}
	return h.UnimplementedCoreHandler.BootNotification(ctx, cp, r)
}
func (h *handler) DataTransfer(ctx context.Context, cp *csms.ChargePoint, r *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	if h.dataTransfer != nil {
		return h.dataTransfer(ctx, cp, r)
	}
	return h.UnimplementedCoreHandler.DataTransfer(ctx, cp, r)
}
func (h *handler) Heartbeat(ctx context.Context, cp *csms.ChargePoint, r *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
	if h.heartbeat != nil {
		return h.heartbeat(ctx, cp, r)
	}
	return h.UnimplementedCoreHandler.Heartbeat(ctx, cp, r)
}
func (h *handler) MeterValues(ctx context.Context, cp *csms.ChargePoint, r *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	if h.meterValues != nil {
		return h.meterValues(ctx, cp, r)
	}
	return h.UnimplementedCoreHandler.MeterValues(ctx, cp, r)
}
func (h *handler) StartTransaction(ctx context.Context, cp *csms.ChargePoint, r *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	if h.startTransaction != nil {
		return h.startTransaction(ctx, cp, r)
	}
	return h.UnimplementedCoreHandler.StartTransaction(ctx, cp, r)
}
func (h *handler) StatusNotification(ctx context.Context, cp *csms.ChargePoint, r *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
	if h.statusNotification != nil {
		return h.statusNotification(ctx, cp, r)
	}
	return h.UnimplementedCoreHandler.StatusNotification(ctx, cp, r)
}
func (h *handler) StopTransaction(ctx context.Context, cp *csms.ChargePoint, r *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	if h.stopTransaction != nil {
		return h.stopTransaction(ctx, cp, r)
	}
	return h.UnimplementedCoreHandler.StopTransaction(ctx, cp, r)
}

// okHandler accepts everything.
func okHandler() *handler {
	accepted := types.IDTagInfo{Status: types.AuthorizationStatusAccepted}
	return &handler{
		authorize: func(context.Context, *csms.ChargePoint, *core.AuthorizeRequest) (*core.AuthorizeConfirmation, error) {
			return &core.AuthorizeConfirmation{IDTagInfo: accepted}, nil
		},
		boot: func(context.Context, *csms.ChargePoint, *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
			return &core.BootNotificationConfirmation{Status: types.RegistrationStatusAccepted, CurrentTime: types.Now(), Interval: 300}, nil
		},
		dataTransfer: func(context.Context, *csms.ChargePoint, *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
			return &core.DataTransferConfirmation{Status: types.DataTransferStatusAccepted, Data: "pong"}, nil
		},
		heartbeat: func(context.Context, *csms.ChargePoint, *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
			return &core.HeartbeatConfirmation{CurrentTime: types.Now()}, nil
		},
		meterValues: func(context.Context, *csms.ChargePoint, *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
			return &core.MeterValuesConfirmation{}, nil
		},
		startTransaction: func(context.Context, *csms.ChargePoint, *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
			return &core.StartTransactionConfirmation{IDTagInfo: accepted, TransactionID: 42}, nil
		},
		statusNotification: func(context.Context, *csms.ChargePoint, *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
			return &core.StatusNotificationConfirmation{}, nil
		},
		stopTransaction: func(context.Context, *csms.ChargePoint, *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
			return &core.StopTransactionConfirmation{IDTagInfo: &accepted}, nil
		},
	}
}

// station is a chargepoint.CoreHandler built from optional functions.
type station struct {
	chargepoint.UnimplementedCoreHandler
	changeAvailability     func(context.Context, *core.ChangeAvailabilityRequest) (*core.ChangeAvailabilityConfirmation, error)
	changeConfiguration    func(context.Context, *core.ChangeConfigurationRequest) (*core.ChangeConfigurationConfirmation, error)
	clearCache             func(context.Context, *core.ClearCacheRequest) (*core.ClearCacheConfirmation, error)
	dataTransfer           func(context.Context, *core.DataTransferRequest) (*core.DataTransferConfirmation, error)
	getConfiguration       func(context.Context, *core.GetConfigurationRequest) (*core.GetConfigurationConfirmation, error)
	remoteStartTransaction func(context.Context, *core.RemoteStartTransactionRequest) (*core.RemoteStartTransactionConfirmation, error)
	remoteStopTransaction  func(context.Context, *core.RemoteStopTransactionRequest) (*core.RemoteStopTransactionConfirmation, error)
	reset                  func(context.Context, *core.ResetRequest) (*core.ResetConfirmation, error)
	unlockConnector        func(context.Context, *core.UnlockConnectorRequest) (*core.UnlockConnectorConfirmation, error)
}

func (h *station) ChangeAvailability(ctx context.Context, r *core.ChangeAvailabilityRequest) (*core.ChangeAvailabilityConfirmation, error) {
	if h.changeAvailability != nil {
		return h.changeAvailability(ctx, r)
	}
	return h.UnimplementedCoreHandler.ChangeAvailability(ctx, r)
}
func (h *station) ChangeConfiguration(ctx context.Context, r *core.ChangeConfigurationRequest) (*core.ChangeConfigurationConfirmation, error) {
	if h.changeConfiguration != nil {
		return h.changeConfiguration(ctx, r)
	}
	return h.UnimplementedCoreHandler.ChangeConfiguration(ctx, r)
}
func (h *station) ClearCache(ctx context.Context, r *core.ClearCacheRequest) (*core.ClearCacheConfirmation, error) {
	if h.clearCache != nil {
		return h.clearCache(ctx, r)
	}
	return h.UnimplementedCoreHandler.ClearCache(ctx, r)
}
func (h *station) DataTransfer(ctx context.Context, r *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	if h.dataTransfer != nil {
		return h.dataTransfer(ctx, r)
	}
	return h.UnimplementedCoreHandler.DataTransfer(ctx, r)
}
func (h *station) GetConfiguration(ctx context.Context, r *core.GetConfigurationRequest) (*core.GetConfigurationConfirmation, error) {
	if h.getConfiguration != nil {
		return h.getConfiguration(ctx, r)
	}
	return h.UnimplementedCoreHandler.GetConfiguration(ctx, r)
}
func (h *station) RemoteStartTransaction(ctx context.Context, r *core.RemoteStartTransactionRequest) (*core.RemoteStartTransactionConfirmation, error) {
	if h.remoteStartTransaction != nil {
		return h.remoteStartTransaction(ctx, r)
	}
	return h.UnimplementedCoreHandler.RemoteStartTransaction(ctx, r)
}
func (h *station) RemoteStopTransaction(ctx context.Context, r *core.RemoteStopTransactionRequest) (*core.RemoteStopTransactionConfirmation, error) {
	if h.remoteStopTransaction != nil {
		return h.remoteStopTransaction(ctx, r)
	}
	return h.UnimplementedCoreHandler.RemoteStopTransaction(ctx, r)
}
func (h *station) Reset(ctx context.Context, r *core.ResetRequest) (*core.ResetConfirmation, error) {
	if h.reset != nil {
		return h.reset(ctx, r)
	}
	return h.UnimplementedCoreHandler.Reset(ctx, r)
}
func (h *station) UnlockConnector(ctx context.Context, r *core.UnlockConnectorRequest) (*core.UnlockConnectorConfirmation, error) {
	if h.unlockConnector != nil {
		return h.unlockConnector(ctx, r)
	}
	return h.UnimplementedCoreHandler.UnlockConnector(ctx, r)
}

// okStation accepts everything.
func okStation() *station {
	return &station{
		changeAvailability: func(context.Context, *core.ChangeAvailabilityRequest) (*core.ChangeAvailabilityConfirmation, error) {
			return &core.ChangeAvailabilityConfirmation{Status: types.AvailabilityStatusAccepted}, nil
		},
		changeConfiguration: func(context.Context, *core.ChangeConfigurationRequest) (*core.ChangeConfigurationConfirmation, error) {
			return &core.ChangeConfigurationConfirmation{Status: types.ConfigurationStatusAccepted}, nil
		},
		clearCache: func(context.Context, *core.ClearCacheRequest) (*core.ClearCacheConfirmation, error) {
			return &core.ClearCacheConfirmation{Status: types.ClearCacheStatusAccepted}, nil
		},
		dataTransfer: func(context.Context, *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
			return &core.DataTransferConfirmation{Status: types.DataTransferStatusAccepted, Data: "pong"}, nil
		},
		getConfiguration: func(context.Context, *core.GetConfigurationRequest) (*core.GetConfigurationConfirmation, error) {
			return &core.GetConfigurationConfirmation{ConfigurationKey: []types.KeyValue{{Key: "HeartbeatInterval", Value: "300"}}}, nil
		},
		remoteStartTransaction: func(context.Context, *core.RemoteStartTransactionRequest) (*core.RemoteStartTransactionConfirmation, error) {
			return &core.RemoteStartTransactionConfirmation{Status: types.RemoteStartStopStatusAccepted}, nil
		},
		remoteStopTransaction: func(context.Context, *core.RemoteStopTransactionRequest) (*core.RemoteStopTransactionConfirmation, error) {
			return &core.RemoteStopTransactionConfirmation{Status: types.RemoteStartStopStatusAccepted}, nil
		},
		reset: func(context.Context, *core.ResetRequest) (*core.ResetConfirmation, error) {
			return &core.ResetConfirmation{Status: types.ResetStatusAccepted}, nil
		},
		unlockConnector: func(context.Context, *core.UnlockConnectorRequest) (*core.UnlockConnectorConfirmation, error) {
			return &core.UnlockConnectorConfirmation{Status: types.UnlockStatusUnlocked}, nil
		},
	}
}

var fastReconnect = ws.Reconnect{WaitMinimum: 5 * time.Millisecond, RandomRange: 5 * time.Millisecond, RepeatTimes: 2}

// events collects hook callbacks.
type events struct {
	connected    chan *csms.ChargePoint
	disconnected chan disconnect
}

type disconnect struct {
	cp  *csms.ChargePoint
	err error
}

func newEvents() *events {
	return &events{connected: make(chan *csms.ChargePoint, 200), disconnected: make(chan disconnect, 200)}
}

func recv[T any](t testing.TB, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for an event")
		panic("unreachable")
	}
}

// startCSMS serves a CSMS over httptest and returns it with the ws:// endpoint
// URL charge points should use.
func startCSMS(t testing.TB, mutate func(*csms.Config)) (*csms.CSMS, *events, string) {
	t.Helper()
	ev := newEvents()
	cfg := csms.Config{
		Core:         okHandler(),
		OnConnect:    func(cp *csms.ChargePoint) { ev.connected <- cp },
		OnDisconnect: func(cp *csms.ChargePoint, err error) { ev.disconnected <- disconnect{cp, err} },
	}
	if mutate != nil {
		mutate(&cfg)
	}
	srv, err := csms.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		ts.Close()
	})
	return srv, ev, "ws" + strings.TrimPrefix(ts.URL, "http") + "/ocpp"
}

// startStation runs a charge point against url and waits until it is connected.
func startStation(t testing.TB, url, id string, mutate func(*chargepoint.Config)) *chargepoint.ChargePoint {
	t.Helper()
	cfg := chargepoint.Config{URL: url, ID: id, Core: okStation(), Reconnect: fastReconnect}
	if mutate != nil {
		mutate(&cfg)
	}
	cp, err := chargepoint.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = cp.Run(context.Background()) }()
	t.Cleanup(func() { _ = cp.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := cp.WaitConnected(ctx); err != nil {
		t.Fatalf("station %s did not connect: %v", id, err)
	}
	return cp
}

// lookup waits until the CSMS knows the charge point.
func lookup(t testing.TB, srv *csms.CSMS, id string) *csms.ChargePoint {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if cp, ok := srv.ChargePoint(id); ok {
			return cp
		}
		if time.Now().After(deadline) {
			t.Fatalf("charge point %s never appeared", id)
		}
		time.Sleep(time.Millisecond)
	}
}

type counter struct {
	mu sync.Mutex
	n  int
}

func (c *counter) inc() { c.mu.Lock(); c.n++; c.mu.Unlock() }
func (c *counter) get() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}
