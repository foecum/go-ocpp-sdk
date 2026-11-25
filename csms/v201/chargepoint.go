package csms

import (
	"context"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/csmscore"
)

// ChargePoint is one connected charging station. It is tied to a single
// connection: after the station reconnects, look it up again with
// CSMS.ChargePoint. It is safe for concurrent use.
//
// The embedded session provides ID, Version (always ocpp.V201), RemoteAddr, Done
// and Close. The methods that call the station, one per message the CSMS sends,
// are in calls_gen.go.
type ChargePoint struct {
	*csmscore.Session
}

func call[Resp any](ctx context.Context, cp *ChargePoint, req ocpp.Request) (*Resp, error) {
	return csmscore.Call[Resp](ctx, cp.Session, req)
}
