// Package csms is the high-level OCPP 2.0.1 Central System (CSMS) API.
//
// Create a CSMS with New, give it a handler for each functional block you
// support and mount it as an http.Handler (or call ListenAndServe). Charging
// stations connect to ws://host/path/<identity>.
//
//	type handler struct{ csms.UnimplementedProvisioningHandler }
//
//	func (handler) BootNotification(ctx context.Context, cp *csms.ChargePoint,
//		req *provisioning.BootNotificationRequest) (*provisioning.BootNotificationResponse, error) {
//		return &provisioning.BootNotificationResponse{
//			Status:      types.RegistrationStatusEnumAccepted,
//			CurrentTime: ocpp.Now(),
//			Interval:    300,
//		}, nil
//	}
//
//	srv, _ := csms.New(csms.Config{Provisioning: handler{}})
//	http.Handle("/ocpp/", srv)
//
// A connected station is a *ChargePoint; look it up with CSMS.ChargePoint and call
// it with its typed methods, for example cp.GetVariables.
package csms
