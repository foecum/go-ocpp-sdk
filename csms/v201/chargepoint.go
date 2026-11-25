package csms

import (
	"context"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/csmscore"
)

// ChargePoint is one connected charging station. It is tied to a single
// connection: after the station reconnects, look it up again with
// CSMS.ChargePoint. It is safe for concurrent use.
type ChargePoint struct {
	session *csmscore.Session
}

// ID returns the charge point identity from the connection URL.
func (cp *ChargePoint) ID() string { return cp.session.ID() }

// Version returns the OCPP version of the connection, always ocpp.V201.
func (cp *ChargePoint) Version() ocpp.Version { return cp.session.Version() }

// RemoteAddr returns the charge point's network address.
func (cp *ChargePoint) RemoteAddr() string { return cp.session.RemoteAddr() }

// Done is closed when the connection has ended.
func (cp *ChargePoint) Done() <-chan struct{} { return cp.session.Done() }

// Close closes the connection gracefully. The charge point will normally
// reconnect.
func (cp *ChargePoint) Close() error { return cp.session.Close() }

func call[Resp any](ctx context.Context, cp *ChargePoint, req ocpp.Request) (*Resp, error) {
	return csmscore.Call[Resp](ctx, cp.session, req)
}
