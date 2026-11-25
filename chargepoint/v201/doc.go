// Package chargepoint is the high-level OCPP 2.0.1 charging station API.
//
// Create a ChargePoint with New, give it a handler for each functional block you
// support, start it with Run and call the CSMS with its typed methods.
//
//	cp, _ := chargepoint.New(chargepoint.Config{
//		URL: "ws://localhost:8080/ocpp",
//		ID:  "CS001",
//	})
//	go cp.Run(ctx) // connects, and reconnects by itself
//	_ = cp.WaitConnected(ctx)
//	boot, err := cp.BootNotification(ctx, provisioning.BootNotificationRequest{ /* ... */ })
package chargepoint
