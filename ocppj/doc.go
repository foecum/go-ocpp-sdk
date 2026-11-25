// Package ocppj implements OCPP-J framing and request/response correlation,
// independent of OCPP version and of the WebSocket library.
//
// Parse and Marshal convert between frames and the Call, CallResult and
// CallError messages. A Registry maps actions to request/response types, and an
// Endpoint sends CALLs one at a time, matches responses by message ID, applies
// timeouts and dispatches incoming CALLs to a handler:
//
//	reg := ocppj.NewRegistry(ocpp.V16)
//	// reg.Register(feature) for each action ...
//	ep, _ := ocppj.NewEndpoint(ocppj.Config{
//		Registry: reg,
//		Send:     func(ctx context.Context, frame []byte) error { return conn.Write(ctx, frame) },
//		Handler:  func(ctx context.Context, req ocpp.Request) (ocpp.Response, error) { /* ... */ },
//	})
//	defer ep.Close()
//	// feed every received frame to ep.HandleFrame(data)
//	resp, err := ep.Call(ctx, request)
package ocppj
