// Package core implements the OCPP 1.6 Core profile: the sixteen messages every
// Charge Point and Central System must support.
//
// Each message is a pair of types named after the specification: a Request
// (the ".req" PDU) and a Confirmation (the ".conf" PDU), for example
// BootNotificationRequest and BootNotificationConfirmation. Both implement
// ocpp.Request or ocpp.Response, so they can be sent with an ocppj.Endpoint
// once registered:
//
//	reg := ocppj.NewRegistry(ocpp.V16)
//	for _, f := range core.Features() {
//		if err := reg.Register(f); err != nil { /* ... */ }
//	}
//
// Packages v16 and below provide a ready-made registry.
//
// A request is validated before it is sent and after it is received. Required
// fields, string lengths, ranges and enumeration values are checked.
//
//	req := core.StartTransactionRequest{
//		ConnectorID: 1,
//		IDTag:       "04A1B2C3",
//		MeterStart:  12345,
//		Timestamp:   types.Now(),
//	}
//	err := req.Validate() // nil
package core
