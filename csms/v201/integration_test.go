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
	"github.com/foecum/go-ocpp-sdk/chargepoint/v201"
	"github.com/foecum/go-ocpp-sdk/csms/v201"
	"github.com/foecum/go-ocpp-sdk/v201/authorization"
	"github.com/foecum/go-ocpp-sdk/v201/availability"
	"github.com/foecum/go-ocpp-sdk/v201/provisioning"
	"github.com/foecum/go-ocpp-sdk/v201/remotecontrol"
	"github.com/foecum/go-ocpp-sdk/v201/transactions"
	"github.com/foecum/go-ocpp-sdk/v201/types"
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

	token := types.IDToken{IDToken: "TAG1", Type: types.IDTokenEnumISO14443}
	auth, err := cp.Authorize(ctx, authorization.AuthorizeRequest{IDToken: token})
	if err != nil || auth.IDTokenInfo.Status != types.AuthorizationStatusAccepted {
		t.Fatalf("authorize: %+v %v", auth, err)
	}
	bad, err := cp.Authorize(ctx, authorization.AuthorizeRequest{IDToken: types.IDToken{IDToken: "BAD", Type: types.IDTokenEnumISO14443}})
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
	visited := 0
	srv.ForEach(func(cp *csms.ChargePoint) bool { visited++; return cp.ID() == "CS001" })
	if visited != 1 {
		t.Errorf("ForEach visited %d charging stations", visited)
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
