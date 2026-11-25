package appendices_test

import (
	"fmt"
	"testing"

	"github.com/foecum/go-ocpp-sdk/v201/appendices"
)

func unique(t *testing.T, what string, values []string) {
	t.Helper()
	seen := map[string]bool{}
	for _, v := range values {
		if v == "" || seen[v] {
			t.Errorf("%s: %q is empty or listed twice", what, v)
		}
		seen[v] = true
	}
}

func TestTablesMatchTheAppendices(t *testing.T) {
	if n := len(appendices.Components); n != 82 {
		t.Errorf("%d components, want 82", n)
	}
	if n := len(appendices.Variables); n != 213 {
		t.Errorf("%d variables, want 213", n)
	}
	if n := len(appendices.DeviceModel); n != 438 {
		t.Errorf("%d device model entries, want 438", n)
	}
	if n := len(appendices.Units); n != 34 {
		t.Errorf("%d units, want 34", n)
	}
	if n := len(appendices.ReasonCodes); n != 51 {
		t.Errorf("%d reason codes, want 51", n)
	}
	if n := len(appendices.SecurityEvents); n != 21 {
		t.Errorf("%d security events, want 21", n)
	}
	unique(t, "components", appendices.Components)
	unique(t, "units", appendices.Units)
	var names, codes []string
	for _, v := range appendices.Variables {
		names = append(names, v.Name)
	}
	for _, c := range appendices.ReasonCodes {
		codes = append(codes, c.Code)
		if c.Group == "" {
			t.Errorf("reason code %s has no group", c.Code)
		}
	}
	unique(t, "variables", names)
	unique(t, "reason codes", codes)
}

func TestConstants(t *testing.T) {
	for got, want := range map[string]string{
		appendices.ComponentOCPPCommCtrlr:       "OCPPCommCtrlr",
		appendices.VariableHeartbeatInterval:    "HeartbeatInterval",
		appendices.UnitKWh:                      "kWh",
		appendices.UnitA:                        "A",
		appendices.ReasonCodeInvalidProfile:     "InvalidProfile",
		appendices.SecurityEventFirmwareUpdated: "FirmwareUpdated",
	} {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

func TestDeviceModelListsTheHeartbeatInterval(t *testing.T) {
	for _, e := range appendices.DeviceModel {
		if e.Component == appendices.ComponentOCPPCommCtrlr && e.Variable == appendices.VariableHeartbeatInterval {
			if e.Required || e.DataType != "integer" || e.Unit != appendices.UnitS {
				t.Errorf("%+v", e)
			}
			return
		}
	}
	t.Error("OCPPCommCtrlr.HeartbeatInterval is missing from the device model")
}

func TestSecurityEvents(t *testing.T) {
	if !appendices.IsKnownSecurityEvent(appendices.SecurityEventFirmwareUpdated) || appendices.IsKnownSecurityEvent("MadeUp") {
		t.Error("IsKnownSecurityEvent")
	}
	if !appendices.IsCriticalSecurityEvent(appendices.SecurityEventFirmwareUpdated) {
		t.Error("FirmwareUpdated is critical")
	}
	if appendices.IsCriticalSecurityEvent(appendices.SecurityEventFailedToAuthenticateAtCsms) {
		t.Error("FailedToAuthenticateAtCsms is not critical")
	}
	if appendices.IsCriticalSecurityEvent("MadeUp") {
		t.Error("an unknown event is not critical")
	}
	critical := 0
	for _, e := range appendices.SecurityEvents {
		if e.Critical {
			critical++
		}
	}
	if critical == 0 || critical == len(appendices.SecurityEvents) {
		t.Errorf("%d of %d critical", critical, len(appendices.SecurityEvents))
	}
}

func Example() {
	fmt.Println(appendices.ComponentOCPPCommCtrlr + "." + appendices.VariableHeartbeatInterval)
	fmt.Println(appendices.IsCriticalSecurityEvent(appendices.SecurityEventFirmwareUpdated))
	// Output:
	// OCPPCommCtrlr.HeartbeatInterval
	// true
}
