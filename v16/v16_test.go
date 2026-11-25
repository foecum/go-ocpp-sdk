package v16_test

import (
	"context"
	"testing"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/ocppj"
	"github.com/foecum/go-ocpp-sdk/v16"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

func TestNewRegistry(t *testing.T) {
	reg, err := v16.NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if reg.Version() != ocpp.V16 || reg.Len() != len(v16.Features()) || reg.Len() != 39 {
		t.Errorf("version %s, %d actions", reg.Version(), reg.Len())
	}
	for _, action := range []string{"Authorize", "BootNotification", "StartTransaction", "StopTransaction", "UnlockConnector",
		"GetDiagnostics", "UpdateFirmware", "SendLocalList", "ReserveNow", "SetChargingProfile", "GetCompositeSchedule", "TriggerMessage",
		"SignCertificate", "CertificateSigned", "SecurityEventNotification", "ExtendedTriggerMessage", "GetLog", "SignedUpdateFirmware"} {
		if _, ok := reg.Lookup(action); !ok {
			t.Errorf("%s is not registered", action)
		}
	}
}

// TestEndpointsExchangeRealMessages runs typed 1.6 messages through two
// ocppj endpoints joined back to back.
func TestEndpointsExchangeRealMessages(t *testing.T) {
	var cp, cs *ocppj.Endpoint
	mk := func(peer **ocppj.Endpoint, h ocppj.RequestHandler) *ocppj.Endpoint {
		reg, err := v16.NewRegistry()
		if err != nil {
			t.Fatal(err)
		}
		ep, err := ocppj.NewEndpoint(ocppj.Config{Registry: reg, Handler: h, Timeout: 2 * time.Second,
			Send: func(_ context.Context, f []byte) error { (*peer).HandleFrame(f); return nil }})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(ep.Close)
		return ep
	}
	cp = mk(&cs, nil)
	cs = mk(&cp, func(_ context.Context, req ocpp.Request) (ocpp.Response, error) {
		switch r := req.(type) {
		case *core.BootNotificationRequest:
			return &core.BootNotificationConfirmation{Status: types.RegistrationStatusAccepted, CurrentTime: types.Now(), Interval: 60}, nil
		case *core.StartTransactionRequest:
			return &core.StartTransactionConfirmation{TransactionID: 7, IDTagInfo: types.IDTagInfo{Status: types.AuthorizationStatusAccepted}}, nil
		default:
			return nil, &ocpp.CallError{Code: ocpp.NotImplemented, Description: r.Action()}
		}
	})

	ctx := context.Background()
	resp, err := cp.Call(ctx, core.BootNotificationRequest{ChargePointVendor: "Acme", ChargePointModel: "X1"})
	if err != nil || resp.(*core.BootNotificationConfirmation).Interval != 60 {
		t.Fatalf("boot: %v %v", resp, err)
	}
	resp, err = cp.Call(ctx, core.StartTransactionRequest{ConnectorID: 1, IDTag: "04A1", MeterStart: 0, Timestamp: types.Now()})
	if err != nil || resp.(*core.StartTransactionConfirmation).TransactionID != 7 {
		t.Fatalf("start: %v %v", resp, err)
	}
	// A message the handler does not implement comes back as a CALLERROR.
	if _, err := cp.Call(ctx, core.HeartbeatRequest{}); err == nil {
		t.Error("expected a CALLERROR")
	}
	// Invalid messages are stopped before sending.
	if _, err := cp.Call(ctx, core.StartTransactionRequest{ConnectorID: 0, IDTag: "x", Timestamp: types.Now()}); err == nil {
		t.Error("connector 0 must be rejected")
	}
}
