// Package ocpp is a Go SDK for OCPP (Open Charge Point Protocol), the protocol
// between EV charging stations and their central management system.
//
// The SDK supports both roles, Charge Point (client) and Central System / CSMS
// (server), over OCPP-J (JSON over WebSocket). This root package holds the
// pieces shared by every OCPP version: the Version constants, the Request,
// Response and Feature interfaces, the CALLERROR codes and the common errors.
//
// Framing lives in package ocppj, transport in ws. Version-specific message
// types live in the v16 and v201 packages; the high-level APIs live in
// chargepoint and csms.
//
// # Errors
//
// Branch on the sentinel errors with errors.Is:
//
//	if errors.Is(err, ocpp.ErrTimeout) { /* retry */ }
//
// A CALLERROR sent by the peer is returned as *CallError:
//
//	var ce *ocpp.CallError
//	if errors.As(err, &ce) { log.Println(ce.Code, ce.Description) }
package ocpp
