package ocppj_test

import (
	"context"
	"fmt"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/ocppj"
	"github.com/foecum/go-ocpp-sdk/v16"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

func ExampleParse() {
	msg, err := ocppj.Parse(ocpp.V16, []byte(`[2,"19223201","Heartbeat",{}]`))
	if err != nil {
		panic(err)
	}
	call := msg.(*ocppj.Call)
	fmt.Println(call.Type(), call.ID(), call.Action)
	// Output: 2 19223201 Heartbeat
}

// A frame that cannot be parsed yields a *ParseError carrying what is needed to
// answer with a CALLERROR.
func ExampleParse_invalid() {
	_, err := ocppj.Parse(ocpp.V16, []byte(`[2,"7","Heartbeat"]`))
	pe := err.(*ocppj.ParseError)
	fmt.Println(pe.ID, pe.Code, pe.ShouldReply())
	// Output: 7 ProtocolError true
}

func ExampleMarshal() {
	frame, err := ocppj.Marshal(&ocppj.CallResult{UniqueID: "19223201", Payload: []byte(`{"status":"Accepted"}`)})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(frame))
	// Output: [3,"19223201",{"status":"Accepted"}]
}

func ExampleNewUUIDGenerator() {
	id := ocppj.NewUUIDGenerator().NewID()
	fmt.Println(len(id) <= ocppj.MaxIDLength)
	// Output: true
}

// An Endpoint sends calls one at a time and matches the answers. It does no I/O:
// the transport supplies Send and feeds received frames to HandleFrame. Here two
// endpoints are joined back to back.
func ExampleEndpoint() {
	var station, csms *ocppj.Endpoint
	newEndpoint := func(peer **ocppj.Endpoint, handler ocppj.RequestHandler) *ocppj.Endpoint {
		reg, err := v16.NewRegistry()
		if err != nil {
			panic(err)
		}
		ep, err := ocppj.NewEndpoint(ocppj.Config{
			Registry: reg,
			Handler:  handler,
			Send:     func(_ context.Context, frame []byte) error { (*peer).HandleFrame(frame); return nil },
		})
		if err != nil {
			panic(err)
		}
		return ep
	}
	station = newEndpoint(&csms, nil)
	csms = newEndpoint(&station, func(_ context.Context, req ocpp.Request) (ocpp.Response, error) {
		return &core.BootNotificationConfirmation{Status: types.RegistrationStatusAccepted, CurrentTime: types.Now(), Interval: 300}, nil
	})
	defer station.Close()
	defer csms.Close()

	resp, err := station.Call(context.Background(), core.BootNotificationRequest{ChargePointVendor: "Acme", ChargePointModel: "X1"})
	if err != nil {
		panic(err)
	}
	boot := resp.(*core.BootNotificationConfirmation)
	fmt.Println(boot.Status, boot.Interval)
	// Output: Accepted 300
}
