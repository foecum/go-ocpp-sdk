package v16_test

import (
	"encoding/json"
	"testing"

	"github.com/foecum/go-ocpp-sdk/v16"
)

// FuzzDecode feeds arbitrary payloads to the decoder of every OCPP 1.6 message.
// It must not panic, and whatever it accepts must validate and survive being
// marshalled and decoded again.
func FuzzDecode(f *testing.F) {
	reg, err := v16.NewRegistry()
	if err != nil {
		f.Fatal(err)
	}
	for _, s := range []string{`{}`, `null`, `{"idTag":"A"}`, `{"connectorId":1,"idTag":"A","meterStart":0,"timestamp":"2026-11-25T12:00:00Z"}`,
		`{"chargePointVendor":"a","chargePointModel":"b"}`, `{"transactionId":1,"meterStop":2,"timestamp":"2026-11-25T12:00:00.5Z"}`, `[]`} {
		for i := range len(v16.Features()) {
			f.Add(uint8(i), s)
		}
	}
	var features []string
	for _, ft := range v16.Features() {
		features = append(features, ft.Action())
	}
	f.Fuzz(func(t *testing.T, idx uint8, payload string) {
		action := features[int(idx)%len(features)]
		if req, err := reg.DecodeRequest(action, json.RawMessage(payload)); err == nil {
			if err := req.Validate(); err != nil {
				t.Fatalf("%s: accepted request does not validate: %v", action, err)
			}
			out, err := json.Marshal(req)
			if err != nil {
				t.Fatalf("%s: cannot marshal accepted request: %v", action, err)
			}
			if _, err := reg.DecodeRequest(action, out); err != nil {
				t.Fatalf("%s: %s does not decode again: %v", action, out, err)
			}
		}
		if resp, err := reg.DecodeResponse(action, json.RawMessage(payload)); err == nil {
			if err := resp.Validate(); err != nil {
				t.Fatalf("%s: accepted response does not validate: %v", action, err)
			}
			out, err := json.Marshal(resp)
			if err != nil {
				t.Fatalf("%s: cannot marshal accepted response: %v", action, err)
			}
			if _, err := reg.DecodeResponse(action, out); err != nil {
				t.Fatalf("%s: %s does not decode again: %v", action, out, err)
			}
		}
	})
}
