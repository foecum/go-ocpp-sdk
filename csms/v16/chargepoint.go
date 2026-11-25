package csms

import (
	"context"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/rpc"
	"github.com/foecum/go-ocpp-sdk/ocppj"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/ws"
)

// ChargePoint is one connected charge point. It is tied to a single
// connection: after the charge point reconnects, look it up again with
// CSMS.ChargePoint. It is safe for concurrent use.
type ChargePoint struct {
	id   string
	conn *ws.Conn
	ep   *ocppj.Endpoint
}

// ID returns the charge point identity from the connection URL.
func (cp *ChargePoint) ID() string { return cp.id }

// Version returns the OCPP version of the connection, always ocpp.V16.
func (cp *ChargePoint) Version() ocpp.Version { return cp.conn.Version() }

// RemoteAddr returns the charge point's network address.
func (cp *ChargePoint) RemoteAddr() string { return cp.conn.RemoteAddr() }

// Done is closed when the connection has ended.
func (cp *ChargePoint) Done() <-chan struct{} { return cp.conn.Done() }

// Close closes the connection gracefully. The charge point will normally
// reconnect.
func (cp *ChargePoint) Close() error { return cp.conn.Close() }

func call[Resp any](ctx context.Context, cp *ChargePoint, req ocpp.Request) (*Resp, error) {
	return rpc.Call[Resp](ctx, cp.ep, req)
}

// ChangeAvailability asks the charge point to change a connector's (or its own)
// availability.
func (cp *ChargePoint) ChangeAvailability(ctx context.Context, req core.ChangeAvailabilityRequest) (*core.ChangeAvailabilityConfirmation, error) {
	return call[core.ChangeAvailabilityConfirmation](ctx, cp, req)
}

// ChangeConfiguration asks the charge point to change a configuration key.
func (cp *ChargePoint) ChangeConfiguration(ctx context.Context, req core.ChangeConfigurationRequest) (*core.ChangeConfigurationConfirmation, error) {
	return call[core.ChangeConfigurationConfirmation](ctx, cp, req)
}

// ClearCache asks the charge point to clear its authorisation cache.
func (cp *ChargePoint) ClearCache(ctx context.Context, req core.ClearCacheRequest) (*core.ClearCacheConfirmation, error) {
	return call[core.ClearCacheConfirmation](ctx, cp, req)
}

// DataTransfer sends vendor-specific data to the charge point.
func (cp *ChargePoint) DataTransfer(ctx context.Context, req core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	return call[core.DataTransferConfirmation](ctx, cp, req)
}

// GetConfiguration reads configuration keys from the charge point.
func (cp *ChargePoint) GetConfiguration(ctx context.Context, req core.GetConfigurationRequest) (*core.GetConfigurationConfirmation, error) {
	return call[core.GetConfigurationConfirmation](ctx, cp, req)
}

// RemoteStartTransaction asks the charge point to start a transaction.
func (cp *ChargePoint) RemoteStartTransaction(ctx context.Context, req core.RemoteStartTransactionRequest) (*core.RemoteStartTransactionConfirmation, error) {
	return call[core.RemoteStartTransactionConfirmation](ctx, cp, req)
}

// RemoteStopTransaction asks the charge point to stop a transaction.
func (cp *ChargePoint) RemoteStopTransaction(ctx context.Context, req core.RemoteStopTransactionRequest) (*core.RemoteStopTransactionConfirmation, error) {
	return call[core.RemoteStopTransactionConfirmation](ctx, cp, req)
}

// Reset asks the charge point to reset.
func (cp *ChargePoint) Reset(ctx context.Context, req core.ResetRequest) (*core.ResetConfirmation, error) {
	return call[core.ResetConfirmation](ctx, cp, req)
}

// UnlockConnector asks the charge point to unlock a connector.
func (cp *ChargePoint) UnlockConnector(ctx context.Context, req core.UnlockConnectorRequest) (*core.UnlockConnectorConfirmation, error) {
	return call[core.UnlockConnectorConfirmation](ctx, cp, req)
}
