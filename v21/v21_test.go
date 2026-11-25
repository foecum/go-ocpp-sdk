package v21_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/ocppj"
	"github.com/foecum/go-ocpp-sdk/v21"
	"github.com/foecum/go-ocpp-sdk/v21/types"
)

func TestNewRegistry(t *testing.T) {
	reg, err := v21.NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if reg.Version() != ocpp.V21 || reg.Len() != 91 || reg.Len() != len(v21.Features()) {
		t.Errorf("version %s, %d actions", reg.Version(), reg.Len())
	}
	for _, action := range []string{"BootNotification", "Authorize", "TransactionEvent", "GetVariables", "SetVariables",
		"RequestStartTransaction", "SetChargingProfile", "NotifyReport", "Get15118EVCertificate", "DataTransfer",
		"SetDERControl", "BatterySwap", "AFRRSignal", "NotifyPeriodicEventStream", "VatNumberValidation"} {
		if _, ok := reg.Lookup(action); !ok {
			t.Errorf("%s is not registered", action)
		}
	}
}

// real-world style payloads that must decode, validate and encode back unchanged.
var golden = map[string]struct {
	action  string
	request bool
	payload string
}{
	"boot request": {"BootNotification", true,
		`{"reason":"PowerUp","chargingStation":{"model":"SingleSocketCharger","vendorName":"VendorX","serialNumber":"SN-1","firmwareVersion":"1.0","modem":{"iccid":"8944500102198304826","imsi":"310150123456789"}}}`},
	"boot response": {"BootNotification", false,
		`{"currentTime":"2026-01-02T03:04:05Z","interval":300,"status":"Accepted"}`},
	"authorize": {"Authorize", true,
		`{"idToken":{"idToken":"04A1B2C3D4","type":"ISO14443"}}`},
	"authorize response": {"Authorize", false,
		`{"idTokenInfo":{"status":"Accepted","cacheExpiryDateTime":"2026-02-01T00:00:00Z","chargingPriority":1,"language1":"en"}}`},
	"transaction started": {"TransactionEvent", true,
		`{"eventType":"Started","timestamp":"2026-01-02T03:04:05.250Z","triggerReason":"Authorized","seqNo":0,` +
			`"transactionInfo":{"transactionId":"tx-0001","chargingState":"Charging"},"evse":{"id":1,"connectorId":1},` +
			`"idToken":{"idToken":"04A1B2C3D4","type":"ISO14443"},` +
			`"meterValue":[{"timestamp":"2026-01-02T03:04:05Z","sampledValue":[{"value":0,"measurand":"Energy.Active.Import.Register","unitOfMeasure":{"unit":"Wh"}}]}]}`},
	"get variables": {"GetVariables", true,
		`{"getVariableData":[{"component":{"name":"OCPPCommCtrlr"},"variable":{"name":"HeartbeatInterval"}},{"component":{"name":"EVSE","evse":{"id":1}},"variable":{"name":"Power"},"attributeType":"Actual"}]}`},
	"set variables": {"SetVariables", true,
		`{"setVariableData":[{"attributeValue":"300","component":{"name":"OCPPCommCtrlr"},"variable":{"name":"HeartbeatInterval"}},{"attributeValue":"","component":{"name":"X"},"variable":{"name":"List"}}]}`},
	"request start": {"RequestStartTransaction", true,
		`{"idToken":{"idToken":"REMOTE01","type":"Central"},"remoteStartId":7,"evseId":1}`},
	"periodic event stream (unconfirmed)": {"NotifyPeriodicEventStream", true,
		`{"id":123,"pending":0,"basetime":"2024-08-27T12:30:40Z","data":[{"t":0,"v":"230.4"},{"t":5,"v":"230.2"}]}`},
	"set default tariff": {"SetDefaultTariff", true,
		`{"evseId":0,"tariff":{"tariffId":"T1","currency":"EUR","energy":{"prices":[{"priceKwh":0.3}]}}}`},
	"vendor extension kept": {"DataTransfer", true,
		`{"vendorId":"com.acme","messageId":"ping","data":{"n":1},"customData":{"vendorId":"com.acme","tier":3}}`},
}

func TestGoldenPayloads(t *testing.T) {
	reg, err := v21.NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	for name, g := range golden {
		var v any
		if g.request {
			v, err = reg.DecodeRequest(g.action, []byte(g.payload))
		} else {
			v, err = reg.DecodeResponse(g.action, []byte(g.payload))
		}
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		out, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		var want, got any
		_ = json.Unmarshal([]byte(g.payload), &want)
		_ = json.Unmarshal(out, &got)
		if !reflect.DeepEqual(want, got) {
			t.Errorf("%s: round trip changed the payload\n in: %s\nout: %s", name, g.payload, out)
		}
	}
}

func TestUnconfirmedMessagesHaveNoResponse(t *testing.T) {
	reg, _ := v21.NewRegistry()
	f, ok := reg.Lookup("NotifyPeriodicEventStream")
	if !ok || !ocpp.IsOneWay(f) || f.ResponseType() != nil || f.RequestType().Name() != "NotifyPeriodicEventStream" {
		t.Errorf("%v", f)
	}
	if _, err := reg.DecodeResponse("NotifyPeriodicEventStream", []byte(`{}`)); err == nil {
		t.Error("an unconfirmed message has no response to decode")
	}
	oneWay := 0
	for _, f := range v21.Features() {
		if ocpp.IsOneWay(f) {
			oneWay++
		}
	}
	if oneWay != 1 {
		t.Errorf("%d unconfirmed messages, want 1", oneWay)
	}
}

func TestDecodeErrorCodes(t *testing.T) {
	reg, _ := v21.NewRegistry()
	tests := []struct {
		name, action, payload string
		code                  ocpp.ErrorCode
	}{
		{"missing field", "BootNotification", `{"reason":"PowerUp"}`, ocpp.OccurrenceConstraintViolation},
		{"missing nested field", "BootNotification", `{"reason":"PowerUp","chargingStation":{"model":"M"}}`, ocpp.OccurrenceConstraintViolation},
		{"bad enum", "BootNotification", `{"reason":"Sleepy","chargingStation":{"model":"M","vendorName":"V"}}`, ocpp.PropertyConstraintViolation},
		{"wrong type", "Heartbeat", `{"customData":"x"}`, ocpp.TypeConstraintViolation},
		{"not JSON at all", "Heartbeat", `{"customData":`, ocpp.FormatViolation},
		{"empty required list", "GetVariables", `{"getVariableData":[]}`, ocpp.PropertyConstraintViolation},
		{"string too long", "Authorize", `{"idToken":{"idToken":"` + strings.Repeat("x", 256) + `","type":"Local"}}`, ocpp.PropertyConstraintViolation},
	}
	for _, tt := range tests {
		_, err := reg.DecodeRequest(tt.action, []byte(tt.payload))
		var de *ocppj.DecodeError
		if !errors.As(err, &de) || de.Code != tt.code {
			t.Errorf("%s: got %v, want code %s", tt.name, err, tt.code)
		}
	}
}

func TestMessagesValidateBeforeSending(t *testing.T) {
	// A zero Request is missing its required fields.
	reg, _ := v21.NewRegistry()
	f, _ := reg.Lookup("TransactionEvent")
	req := reflect.New(f.RequestType()).Interface().(ocpp.Request)
	if err := req.Validate(); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("got %v", err)
	}
	_ = types.Now()
}
