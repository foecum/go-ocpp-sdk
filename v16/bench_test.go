package v16_test

import (
	"encoding/json"
	"testing"

	"github.com/foecum/go-ocpp-sdk/v16"
)

// BenchmarkDecodeBootNotification measures decoding and validating a realistic
// 1.6 request.
func BenchmarkDecodeBootNotification(b *testing.B) {
	reg, err := v16.NewRegistry()
	if err != nil {
		b.Fatal(err)
	}
	payload := json.RawMessage(`{"chargePointVendor":"Acme","chargePointModel":"X1","chargePointSerialNumber":"SN0001","firmwareVersion":"1.2.3","iccid":"8944500000000000000","meterType":"MID"}`)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := reg.DecodeRequest("BootNotification", payload); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecodeMeterValues is the heaviest common 1.6 message: nested arrays
// and timestamps.
func BenchmarkDecodeMeterValues(b *testing.B) {
	reg, err := v16.NewRegistry()
	if err != nil {
		b.Fatal(err)
	}
	payload := json.RawMessage(`{"connectorId":1,"transactionId":42,"meterValue":[{"timestamp":"2026-11-25T12:00:00.000Z","sampledValue":[` +
		`{"value":"1500","measurand":"Energy.Active.Import.Register","unit":"Wh"},{"value":"16.0","measurand":"Current.Import","unit":"A","phase":"L1"},` +
		`{"value":"230.1","measurand":"Voltage","unit":"V","phase":"L1-N"}]}]}`)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := reg.DecodeRequest("MeterValues", payload); err != nil {
			b.Fatal(err)
		}
	}
}
