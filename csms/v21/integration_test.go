package csms_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/chargepoint/v21"
	"github.com/foecum/go-ocpp-sdk/csms/v21"
	"github.com/foecum/go-ocpp-sdk/v21/authorization"
	"github.com/foecum/go-ocpp-sdk/v21/availability"
	"github.com/foecum/go-ocpp-sdk/v21/diagnostics"
	"github.com/foecum/go-ocpp-sdk/v21/provisioning"
	"github.com/foecum/go-ocpp-sdk/v21/remotecontrol"
	"github.com/foecum/go-ocpp-sdk/v21/tariffandcost"
	"github.com/foecum/go-ocpp-sdk/v21/transactions"
	"github.com/foecum/go-ocpp-sdk/v21/types"
)

// csmsHandlers is the Central System side of the scenario.
type provisioningHandler struct {
	csms.UnimplementedProvisioningHandler
}

func (provisioningHandler) BootNotification(ctx context.Context, cp *csms.ChargePoint, req *provisioning.BootNotificationRequest) (*provisioning.BootNotificationResponse, error) {
	return &provisioning.BootNotificationResponse{Status: types.RegistrationStatusAccepted, CurrentTime: ocpp.Now(), Interval: 300}, nil
}

type authorizationHandler struct {
	csms.UnimplementedAuthorizationHandler
}

func (authorizationHandler) Authorize(ctx context.Context, cp *csms.ChargePoint, req *authorization.AuthorizeRequest) (*authorization.AuthorizeResponse, error) {
	status := types.AuthorizationStatusAccepted
	if req.IDToken.IDToken == "BAD" {
		status = types.AuthorizationStatusBlocked
	}
	return &authorization.AuthorizeResponse{IDTokenInfo: types.IDTokenInfo{Status: status}}, nil
}

type availabilityHandler struct {
	csms.UnimplementedAvailabilityHandler
}

func (availabilityHandler) StatusNotification(ctx context.Context, cp *csms.ChargePoint, req *availability.StatusNotificationRequest) (*availability.StatusNotificationResponse, error) {
	return &availability.StatusNotificationResponse{}, nil
}

type transactionsHandler struct {
	csms.UnimplementedTransactionsHandler
	mu     sync.Mutex
	events []types.TransactionEvent
}

func (h *transactionsHandler) TransactionEvent(ctx context.Context, cp *csms.ChargePoint, req *transactions.TransactionEventRequest) (*transactions.TransactionEventResponse, error) {
	h.mu.Lock()
	h.events = append(h.events, req.EventType)
	h.mu.Unlock()
	return &transactions.TransactionEventResponse{}, nil
}

// Station side.
type remoteControl struct {
	chargepoint.UnimplementedRemoteControlHandler
	started chan int
}

func (h remoteControl) RequestStartTransaction(ctx context.Context, req *remotecontrol.RequestStartTransactionRequest) (*remotecontrol.RequestStartTransactionResponse, error) {
	h.started <- req.RemoteStartID
	return &remotecontrol.RequestStartTransactionResponse{Status: types.RequestStartStopStatusAccepted}, nil
}

type provisioningStation struct {
	chargepoint.UnimplementedProvisioningHandler
}

func (provisioningStation) GetVariables(ctx context.Context, req *provisioning.GetVariablesRequest) (*provisioning.GetVariablesResponse, error) {
	out := &provisioning.GetVariablesResponse{}
	for _, d := range req.GetVariableData {
		out.GetVariableResult = append(out.GetVariableResult, types.GetVariableResult{
			AttributeStatus: types.GetVariableStatusAccepted, Component: d.Component, Variable: d.Variable, AttributeValue: "42"})
	}
	return out, nil
}

func (provisioningStation) SetVariables(ctx context.Context, req *provisioning.SetVariablesRequest) (*provisioning.SetVariablesResponse, error) {
	out := &provisioning.SetVariablesResponse{}
	for _, d := range req.SetVariableData {
		out.SetVariableResult = append(out.SetVariableResult, types.SetVariableResult{
			AttributeStatus: types.SetVariableStatusAccepted, Component: d.Component, Variable: d.Variable})
	}
	return out, nil
}

// TestChargingSessionScenario runs a charging station against a Central System:
// boot, status, authorize, a transaction (Started, Updated, Ended) and calls from
// the CSMS (RequestStartTransaction, GetVariables, SetVariables).
func TestChargingSessionScenario(t *testing.T) {
	tx := &transactionsHandler{}
	connected := make(chan string, 1)
	disconnected := make(chan string, 1)
	srv, err := csms.New(csms.Config{
		Provisioning:  provisioningHandler{},
		Authorization: authorizationHandler{},
		Availability:  availabilityHandler{},
		Transactions:  tx,
		OnConnect:     func(cp *csms.ChargePoint) { connected <- cp.ID() },
		OnDisconnect:  func(cp *csms.ChargePoint, err error) { disconnected <- cp.ID() },
	})
	if err != nil {
		t.Fatal(err)
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	served := make(chan error, 1)
	go func() { served <- srv.Serve(l) }()

	started := make(chan int, 1)
	cp, err := chargepoint.New(chargepoint.Config{
		URL:           "ws://" + l.Addr().String() + "/ocpp",
		ID:            "CS001",
		RemoteControl: remoteControl{started: started},
		Provisioning:  provisioningStation{},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	ran := make(chan error, 1)
	go func() { ran <- cp.Run(ctx) }()
	if err := cp.WaitConnected(ctx); err != nil {
		t.Fatal(err)
	}
	if id := <-connected; id != "CS001" {
		t.Fatalf("connected %q", id)
	}

	boot, err := cp.BootNotification(ctx, provisioning.BootNotificationRequest{
		Reason:          types.BootReasonPowerUp,
		ChargingStation: types.ChargingStation{Model: "X1", VendorName: "Acme"},
	})
	if err != nil || boot.Status != types.RegistrationStatusAccepted || boot.Interval != 300 {
		t.Fatalf("boot: %+v %v", boot, err)
	}
	if _, err := cp.StatusNotification(ctx, availability.StatusNotificationRequest{
		Timestamp: ocpp.Now(), ConnectorStatus: types.ConnectorStatusAvailable, EVSEID: 1, ConnectorID: 1}); err != nil {
		t.Fatal(err)
	}
	// Heartbeat has no handler: NotSupported.
	var ce *ocpp.CallError
	if _, err := cp.Heartbeat(ctx, availability.HeartbeatRequest{}); !errors.As(err, &ce) || ce.Code != ocpp.NotSupported {
		t.Errorf("heartbeat: %v", err)
	}

	token := types.IDToken{IDToken: "TAG1", Type: "ISO14443"}
	auth, err := cp.Authorize(ctx, authorization.AuthorizeRequest{IDToken: token})
	if err != nil || auth.IDTokenInfo.Status != types.AuthorizationStatusAccepted {
		t.Fatalf("authorize: %+v %v", auth, err)
	}
	bad, err := cp.Authorize(ctx, authorization.AuthorizeRequest{IDToken: types.IDToken{IDToken: "BAD", Type: "ISO14443"}})
	if err != nil || bad.IDTokenInfo.Status != types.AuthorizationStatusBlocked {
		t.Fatalf("authorize BAD: %+v %v", bad, err)
	}

	for i, step := range []struct {
		event   types.TransactionEvent
		trigger types.TriggerReason
	}{
		{types.TransactionEventStarted, types.TriggerReasonAuthorized},
		{types.TransactionEventUpdated, types.TriggerReasonMeterValuePeriodic},
		{types.TransactionEventEnded, types.TriggerReasonEVDeparted},
	} {
		_, err := cp.TransactionEvent(ctx, transactions.TransactionEventRequest{
			EventType: step.event, Timestamp: ocpp.Now(), TriggerReason: step.trigger, SeqNo: i,
			TransactionInfo: types.Transaction{TransactionID: "TX1"}, IDToken: &token,
		})
		if err != nil {
			t.Fatalf("transaction event %s: %v", step.event, err)
		}
	}
	tx.mu.Lock()
	got := append([]types.TransactionEvent(nil), tx.events...)
	tx.mu.Unlock()
	if len(got) != 3 || got[0] != types.TransactionEventStarted || got[1] != types.TransactionEventUpdated || got[2] != types.TransactionEventEnded {
		t.Errorf("events %v", got)
	}

	// The CSMS calls the station.
	station, ok := srv.ChargePoint("CS001")
	if !ok {
		t.Fatal("station not listed")
	}
	if ids := srv.ChargePointIDs(); len(ids) != 1 || ids[0] != "CS001" {
		t.Errorf("ids %v", ids)
	}
	if _, ok := srv.ChargePoint("nobody"); ok {
		t.Error("unknown station found")
	}
	rs, err := station.RequestStartTransaction(ctx, remotecontrol.RequestStartTransactionRequest{IDToken: token, RemoteStartID: 7})
	if err != nil || rs.Status != types.RequestStartStopStatusAccepted {
		t.Fatalf("remote start: %+v %v", rs, err)
	}
	if id := <-started; id != 7 {
		t.Errorf("remote start id %d", id)
	}
	comp := types.Component{Name: "OCPPCommCtrlr"}
	variable := types.Variable{Name: "HeartbeatInterval"}
	gv, err := station.GetVariables(ctx, provisioning.GetVariablesRequest{GetVariableData: []types.GetVariableData{{Component: comp, Variable: variable}}})
	if err != nil || len(gv.GetVariableResult) != 1 || gv.GetVariableResult[0].AttributeValue != "42" {
		t.Fatalf("get variables: %+v %v", gv, err)
	}
	sv, err := station.SetVariables(ctx, provisioning.SetVariablesRequest{SetVariableData: []types.SetVariableData{{Component: comp, Variable: variable, AttributeValue: "60"}}})
	if err != nil || len(sv.SetVariableResult) != 1 || sv.SetVariableResult[0].AttributeStatus != types.SetVariableStatusAccepted {
		t.Fatalf("set variables: %+v %v", sv, err)
	}
	// A request that is invalid is rejected before it is sent.
	if _, err := station.GetVariables(ctx, provisioning.GetVariablesRequest{}); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("empty GetVariables: %v", err)
	}
	// A call the station has no handler for is answered NotSupported.
	if _, err := station.Reset(ctx, provisioning.ResetRequest{Type: types.ResetImmediate}); !errors.As(err, &ce) || ce.Code != ocpp.NotSupported {
		t.Errorf("reset: %v", err)
	}

	if err := cp.Close(); err != nil && !errors.Is(err, ocpp.ErrClosed) {
		t.Errorf("close: %v", err)
	}
	if id := <-disconnected; id != "CS001" {
		t.Errorf("disconnected %q", id)
	}
	shutdownCtx, c2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer c2()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Errorf("shutdown: %v", err)
	}
	if err := <-served; !errors.Is(err, http.ErrServerClosed) {
		t.Errorf("serve: %v", err)
	}
}

func TestListenAndServe(t *testing.T) {
	srv, err := csms.New(csms.Config{})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- srv.ListenAndServe("127.0.0.1:0") }()
	time.Sleep(100 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, http.ErrServerClosed) {
		t.Errorf("got %v", err)
	}
}

// diagnosticsHandler receives the unconfirmed NotifyPeriodicEventStream messages
// and answers the OpenPeriodicEventStream calls.
type diagnosticsHandler struct {
	csms.UnimplementedDiagnosticsHandler
	opened chan types.ConstantStreamData
	data   chan *diagnostics.NotifyPeriodicEventStream
}

func (h diagnosticsHandler) OpenPeriodicEventStream(ctx context.Context, cp *csms.ChargePoint, req *diagnostics.OpenPeriodicEventStreamRequest) (*diagnostics.OpenPeriodicEventStreamResponse, error) {
	h.opened <- req.ConstantStreamData
	return &diagnostics.OpenPeriodicEventStreamResponse{Status: types.GenericStatusAccepted}, nil
}

func (h diagnosticsHandler) NotifyPeriodicEventStream(ctx context.Context, cp *csms.ChargePoint, msg *diagnostics.NotifyPeriodicEventStream) error {
	h.data <- msg
	return nil
}

type tariffStation struct {
	chargepoint.UnimplementedTariffAndCostHandler
	got chan types.Tariff
}

func (h tariffStation) SetDefaultTariff(ctx context.Context, req *tariffandcost.SetDefaultTariffRequest) (*tariffandcost.SetDefaultTariffResponse, error) {
	h.got <- req.Tariff
	return &tariffandcost.SetDefaultTariffResponse{Status: types.TariffSetStatusAccepted}, nil
}

// TestOCPP21Features runs the parts of OCPP 2.1 that 2.0.1 does not have: an
// unconfirmed (SEND) message from the station, which the CSMS never answers and
// which does not wait for the call before it, and a call to the station for one
// of the new messages.
func TestOCPP21Features(t *testing.T) {
	diag := diagnosticsHandler{opened: make(chan types.ConstantStreamData, 1), data: make(chan *diagnostics.NotifyPeriodicEventStream, 4)}
	srv, err := csms.New(csms.Config{Diagnostics: diag})
	if err != nil {
		t.Fatal(err)
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(l) }()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	tariffs := tariffStation{got: make(chan types.Tariff, 1)}
	cp, err := chargepoint.New(chargepoint.Config{URL: "ws://" + l.Addr().String() + "/ocpp", ID: "CS21", TariffAndCost: tariffs})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	go func() { _ = cp.Run(ctx) }()
	defer cp.Close()
	if err := cp.WaitConnected(ctx); err != nil {
		t.Fatal(err)
	}

	if _, err := cp.OpenPeriodicEventStream(ctx, diagnostics.OpenPeriodicEventStreamRequest{
		ConstantStreamData: types.ConstantStreamData{ID: 5, VariableMonitoringID: 9, Params: types.PeriodicEventStreamParams{}},
	}); err != nil {
		t.Fatal(err)
	}
	if got := <-diag.opened; got.ID != 5 {
		t.Errorf("stream %+v", got)
	}
	// Unconfirmed messages: nothing comes back, so the call returns once written.
	for i := 0; i < 3; i++ {
		err := cp.NotifyPeriodicEventStream(ctx, diagnostics.NotifyPeriodicEventStream{
			ID: 5, Pending: 2 - i, Basetime: ocpp.Now(), Data: []types.StreamDataElement{{T: 0, V: "230.4"}, {T: 5, V: "230.2"}},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 3; i++ {
		select {
		case m := <-diag.data:
			if m.ID != 5 || len(m.Data) != 2 || m.Pending != 2-i {
				t.Errorf("message %d: %+v", i, m)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("unconfirmed message %d not received", i)
		}
	}
	if err := cp.NotifyPeriodicEventStream(ctx, diagnostics.NotifyPeriodicEventStream{}); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("an invalid message must not be sent: %v", err)
	}

	station, ok := srv.ChargePoint("CS21")
	if !ok {
		t.Fatal("station not listed")
	}
	if station.Version() != ocpp.V21 {
		t.Errorf("version %q", station.Version())
	}
	resp, err := station.SetDefaultTariff(ctx, tariffandcost.SetDefaultTariffRequest{EVSEID: 0, Tariff: types.Tariff{TariffID: "T1", Currency: "EUR"}})
	if err != nil || resp.Status != types.TariffSetStatusAccepted {
		t.Fatalf("set default tariff: %+v %v", resp, err)
	}
	if got := <-tariffs.got; got.TariffID != "T1" {
		t.Errorf("tariff %+v", got)
	}
	// The station has no handler for the other new messages: NotSupported.
	var ce *ocpp.CallError
	if _, err := station.GetTariffs(ctx, tariffandcost.GetTariffsRequest{EVSEID: 1}); !errors.As(err, &ce) || ce.Code != ocpp.NotSupported {
		t.Errorf("get tariffs: %v", err)
	}
}
