package csms_test

import (
	"context"
	"fmt"
	"net"

	"github.com/foecum/go-ocpp-sdk/chargepoint/v16"
	"github.com/foecum/go-ocpp-sdk/csms/v16"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// bootHandler answers BootNotification. Embedding UnimplementedCoreHandler makes
// every other call answer NotSupported.
type bootHandler struct{ csms.UnimplementedCoreHandler }

func (bootHandler) BootNotification(ctx context.Context, cp *csms.ChargePoint, req *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
	fmt.Printf("boot from %s: %s %s\n", cp.ID(), req.ChargePointVendor, req.ChargePointModel)
	return &core.BootNotificationConfirmation{
		Status:      types.RegistrationStatusAccepted,
		CurrentTime: types.Now(),
		Interval:    300,
	}, nil
}

// A Central System is an http.Handler (it also has Serve and ListenAndServe).
// Once a charge point is connected, call it with the typed methods.
func Example() {
	ctx := context.Background()
	srv, err := csms.New(csms.Config{Core: bootHandler{}})
	if err != nil {
		panic(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	go srv.Serve(ln)
	defer srv.Shutdown(ctx)

	// A charging station to talk to; see package chargepoint.
	station, err := chargepoint.New(chargepoint.Config{URL: "ws://" + ln.Addr().String() + "/ocpp", ID: "CS001"})
	if err != nil {
		panic(err)
	}
	go station.Run(ctx)
	defer station.Close()
	if err := station.WaitConnected(ctx); err != nil {
		panic(err)
	}
	if _, err := station.BootNotification(ctx, core.BootNotificationRequest{ChargePointVendor: "Acme", ChargePointModel: "X1"}); err != nil {
		panic(err)
	}

	cp, ok := srv.ChargePoint("CS001")
	fmt.Println(ok, srv.ChargePoints())

	// The station has no handler for Reset, so it answers NotSupported.
	_, err = cp.Reset(ctx, core.ResetRequest{Type: types.ResetTypeSoft})
	fmt.Println(err)
	// Output:
	// boot from CS001: Acme X1
	// true [CS001]
	// ocpp: CALLERROR NotSupported: Reset is not supported
}
