// Package csms is the high-level OCPP 1.6 Central System (CSMS) API.
//
// A CSMS accepts WebSocket connections from charge points, answers their calls
// through the handlers you provide, and lets you call them back with typed
// methods:
//
//	type handler struct{ csms.UnimplementedCoreHandler }
//
//	func (handler) BootNotification(ctx context.Context, cp *csms.ChargePoint,
//		req *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
//		return &core.BootNotificationConfirmation{
//			Status:      types.RegistrationStatusAccepted,
//			CurrentTime: types.Now(),
//			Interval:    300,
//		}, nil
//	}
//
//	srv, _ := csms.New(csms.Config{Core: handler{}})
//	log.Fatal(http.ListenAndServe(":8080", srv)) // the CSMS is an http.Handler
//
// The charge point identity is the last path segment of the connection URL, so
// serve on a path such as "/ocpp/" and charge points connect to
// "ws://host/ocpp/CS001".
//
// # Handlers
//
// A profile with no handler (a nil Config.Core), or a method left to the
// embedded Unimplemented handler, is answered with a NotSupported CALLERROR. An
// action the SDK does not know is answered with NotImplemented. A handler may
// return a *ocpp.CallError to choose the CALLERROR sent to the charge point;
// any other error becomes InternalError. A handler that panics is answered with
// InternalError and the process keeps running.
//
// # Calling a charge point
//
// Look a connected charge point up and call it:
//
//	if cp, ok := srv.ChargePoint("CS001"); ok {
//		conf, err := cp.Reset(ctx, core.ResetRequest{Type: types.ResetTypeSoft})
//	}
//
// A ChargePoint value is tied to one connection. If the charge point
// reconnects, look it up again. Calls on a closed connection fail at once with
// an error matching ocpp.ErrNotConnected, including a call that was waiting for
// its answer when the connection dropped (the charge point may or may not have
// processed it).
package csms
