package types_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/foecum/go-ocpp-sdk/v201/types"
)

func TestCustomDataKeepsExtraProperties(t *testing.T) {
	var c types.CustomData
	in := `{"vendorId":"acme","tier":3,"note":"x","nested":{"a":[1,2]}}`
	if err := json.Unmarshal([]byte(in), &c); err != nil {
		t.Fatal(err)
	}
	if c.VendorID != "acme" || len(c.Extra) != 3 || string(c.Extra["tier"]) != "3" {
		t.Fatalf("%+v", c)
	}
	out, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var a, b map[string]any
	_ = json.Unmarshal([]byte(in), &a)
	_ = json.Unmarshal(out, &b)
	if len(a) != len(b) || b["vendorId"] != "acme" || b["tier"].(float64) != 3 {
		t.Errorf("round trip lost data: %s", out)
	}
	// Only vendorId: no extras.
	var plain types.CustomData
	if err := json.Unmarshal([]byte(`{"vendorId":"v"}`), &plain); err != nil || plain.Extra != nil {
		t.Errorf("%+v %v", plain, err)
	}
	if out, _ := json.Marshal(plain); string(out) != `{"vendorId":"v"}` {
		t.Errorf("%s", out)
	}
	// An extra named vendorId cannot override the field.
	spoof := types.CustomData{VendorID: "real", Extra: map[string]json.RawMessage{"vendorId": json.RawMessage(`"fake"`)}}
	if out, _ := json.Marshal(spoof); !strings.Contains(string(out), `"real"`) || strings.Contains(string(out), "fake") {
		t.Errorf("%s", out)
	}
}

func TestCustomDataErrorsAndValidation(t *testing.T) {
	var c types.CustomData
	for _, bad := range []string{`"text"`, `[1]`, `{"vendorId":5}`} {
		if err := json.Unmarshal([]byte(bad), &c); err == nil {
			t.Errorf("%s: expected an error", bad)
		}
	}
	if err := (types.CustomData{VendorID: "acme"}).Validate(); err != nil {
		t.Error(err)
	}
	if err := (types.CustomData{}).Validate(); err == nil {
		t.Error("vendorId is required")
	}
	if err := (types.CustomData{VendorID: strings.Repeat("v", 256)}).Validate(); err == nil {
		t.Error("vendorId is limited to 255 characters")
	}
}
