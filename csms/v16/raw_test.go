package csms_test

import (
	"context"
	"testing"
	"time"

	"github.com/coder/websocket"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/ocppj"
)

// rawStation is a charge point that speaks raw frames, to test how the CSMS
// answers messages a well-behaved station would not send.
func rawStation(t *testing.T, url, id string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, url+"/"+id, &websocket.DialOptions{Subprotocols: []string{"ocpp1.6"}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.CloseNow() })
	return c
}

// exchange sends one frame and returns the CALLERROR it provokes.
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

func TestCSMSAnswersBadFramesWithTheRightCodes(t *testing.T) {
	_, _, url := startCSMS(t, nil)
	c := rawStation(t, url, "RAW1")
	tests := []struct {
		name, frame string
		id          string
		code        ocpp.ErrorCode
	}{
		{"unknown action", `[2,"a1","MakeCoffee",{}]`, "a1", ocpp.NotImplemented},
		{"known but Central System only", `[2,"a2","ChangeAvailability",{"connectorId":1,"type":"Operative"}]`, "a2", ocpp.NotSupported},
		{"missing required field", `[2,"a3","BootNotification",{"chargePointModel":"x"}]`, "a3", ocpp.OccurenceConstraintViolation},
		{"wrong type", `[2,"a4","BootNotification",{"chargePointVendor":5,"chargePointModel":"x"}]`, "a4", ocpp.TypeConstraintViolation},
		{"bad enum", `[2,"a5","StatusNotification",{"connectorId":1,"errorCode":"NoError","status":"Bogus"}]`, "a5", ocpp.PropertyConstraintViolation},
		{"string too long", `[2,"a6","Authorize",{"idTag":"012345678901234567890"}]`, "a6", ocpp.PropertyConstraintViolation},
		{"not JSON", `this is not json`, "-1", ocpp.FormationViolation},
	}
	for _, tt := range tests {
		ce := exchange(t, c, tt.frame)
		if ce.UniqueID != tt.id || ce.Code != tt.code {
			t.Errorf("%s: got id=%q code=%q, want id=%q code=%q", tt.name, ce.UniqueID, ce.Code, tt.id, tt.code)
		}
		if !ce.Code.IsValid(ocpp.V16) {
			t.Errorf("%s: %q is not a 1.6 error code", tt.name, ce.Code)
		}
	}
}
