package chargepoint_test

import (
	"context"
	"errors"
	"fmt"
	"net"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/chargepoint/v16"
	"github.com/foecum/go-ocpp-sdk/csms/v16"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// resetHandler answers Reset. Embedding UnimplementedCoreHandler makes every other
// call from the Central System answer NotSupported.
type resetHandler struct {
	chargepoint.UnimplementedCoreHandler
}

func (resetHandler) Reset(ctx context.Context, req *core.ResetRequest) (*core.ResetConfirmation, error) {
	fmt.Println("reset requested:", req.Type)
	return &core.ResetConfirmation{Status: types.ResetStatusAccepted}, nil
}

// A ChargePoint connects with Run (which also reconnects), sends calls with
// typed methods and answers the Central System through its handlers.
func Example() {
	ctx := context.Background()
	srv, err := csms.New(csms.Config{})
	if err != nil {
		panic(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	go srv.Serve(ln)
	defer srv.Shutdown(ctx)

	station, err := chargepoint.New(chargepoint.Config{
		URL:  "ws://" + ln.Addr().String() + "/ocpp",
		ID:   "CS001",
		Core: resetHandler{},
	})
	if err != nil {
		panic(err)
	}
	go station.Run(ctx)
	defer station.Close()
	if err := station.WaitConnected(ctx); err != nil {
		panic(err)
	}

	// This Central System has no handlers, so it answers NotSupported.
	_, err = station.Heartbeat(ctx, core.HeartbeatRequest{})
	var callErr *ocpp.CallError
	fmt.Println(errors.As(err, &callErr), callErr.Code)

	// The Central System can call the station back.
	for cp, ok := srv.ChargePoint("CS001"); ok; ok = false {
		reset, err := cp.Reset(ctx, core.ResetRequest{Type: types.ResetTypeSoft})
		fmt.Println(reset.Status, err)
	}
	// Output:
	// true NotSupported
	// reset requested: Soft
	// Accepted <nil>
}

// Calls made while the station is not connected fail at once. Nothing is queued
// or retried for you: use Connected and WaitConnected to build what you need.
func ExampleChargePoint_offline() {
	station, err := chargepoint.New(chargepoint.Config{URL: "ws://csms.example.com/ocpp", ID: "CS001"})
	if err != nil {
		panic(err)
	}
	_, err = station.Heartbeat(context.Background(), core.HeartbeatRequest{})
	fmt.Println(station.Connected(), errors.Is(err, ocpp.ErrNotConnected))
	// Output: false true
}
