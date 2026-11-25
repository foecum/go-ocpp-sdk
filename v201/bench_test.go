package v201_test

import (
	"encoding/json"
	"testing"

	"github.com/foecum/go-ocpp-sdk/v201"
)

// BenchmarkDecodeBootNotification measures decoding and validating a realistic
// 2.0.1 request.
func BenchmarkDecodeBootNotification(b *testing.B) {
	reg, err := v201.NewRegistry()
	if err != nil {
		b.Fatal(err)
	}
	payload := json.RawMessage(`{"reason":"PowerUp","chargingStation":{"model":"X1","vendorName":"Acme","serialNumber":"SN0001","firmwareVersion":"1.2.3"}}`)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := reg.DecodeRequest("BootNotification", payload); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecodeTransactionEvent is the heaviest common 2.0.1 message: nested
// objects, arrays and timestamps.
func BenchmarkDecodeTransactionEvent(b *testing.B) {
	reg, err := v201.NewRegistry()
	if err != nil {
		b.Fatal(err)
	}
	payload := json.RawMessage(`{"eventType":"Updated","timestamp":"2026-11-25T12:00:00.000Z","triggerReason":"MeterValuePeriodic","seqNo":3,` +
		`"transactionInfo":{"transactionId":"TX-1","chargingState":"Charging"},"evse":{"id":1,"connectorId":1},"idToken":{"idToken":"04A1B2C3","type":"ISO14443"},` +
		`"meterValue":[{"timestamp":"2026-11-25T12:00:00.000Z","sampledValue":[{"value":1500,"measurand":"Energy.Active.Import.Register","unitOfMeasure":{"unit":"Wh"}},` +
		`{"value":16,"measurand":"Current.Import","phase":"L1","unitOfMeasure":{"unit":"A"}}]}]}`)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := reg.DecodeRequest("TransactionEvent", payload); err != nil {
			b.Fatal(err)
		}
	}
}
