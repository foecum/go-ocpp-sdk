// Package chargepoint is the high-level OCPP 1.6 Charge Point (charging
// station) API.
//
// A ChargePoint connects to a Central System, keeps the connection up,
// reconnecting with the back-off OCPP-J specifies, sends calls with typed
// methods, and answers the Central System's calls through your handler:
//
//	type handler struct{ chargepoint.UnimplementedCoreHandler }
//
//	func (handler) Reset(ctx context.Context, req *core.ResetRequest) (*core.ResetConfirmation, error) {
//		return &core.ResetConfirmation{Status: types.ResetStatusAccepted}, nil
//	}
//
//	cp, err := chargepoint.New(chargepoint.Config{
//		URL:  "wss://csms.example.com/ocpp",
//		ID:   "CS001",
//		Core: handler{},
//	})
//	go cp.Run(ctx)
//
//	if err := cp.WaitConnected(ctx); err == nil {
//		conf, err := cp.BootNotification(ctx, core.BootNotificationRequest{
//			ChargePointVendor: "Acme",
//			ChargePointModel:  "X1",
//		})
//		_ = conf
//		_ = err
//	}
//
// The ChargePoint never sends BootNotification or Heartbeat by itself; the
// application decides when to (a BootNotification after every start, and
// Heartbeats at the interval the Central System returns).
//
// # Offline behaviour
//
// Calls made while the connection is down fail at once with an error matching
// ocpp.ErrNotConnected. So does a call that was waiting for its answer when the
// connection dropped; the Central System may or may not have processed it.
// Nothing is queued or retried for you: use Connected and WaitConnected to
// build whatever offline queue your station needs (OCPP 1.6 requires
// transaction messages to be retried).
package chargepoint
