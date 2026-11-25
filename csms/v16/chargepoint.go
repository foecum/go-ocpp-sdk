package csms

import (
	"context"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/rpc"
	"github.com/foecum/go-ocpp-sdk/ocppj"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/firmware"
	"github.com/foecum/go-ocpp-sdk/v16/localauth"
	"github.com/foecum/go-ocpp-sdk/v16/remotetrigger"
	"github.com/foecum/go-ocpp-sdk/v16/reservation"
	"github.com/foecum/go-ocpp-sdk/v16/smartcharging"
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

// GetDiagnostics asks the charge point to upload its diagnostics.
func (cp *ChargePoint) GetDiagnostics(ctx context.Context, req firmware.GetDiagnosticsRequest) (*firmware.GetDiagnosticsConfirmation, error) {
	return call[firmware.GetDiagnosticsConfirmation](ctx, cp, req)
}

// UpdateFirmware tells the charge point to download and install new firmware.
func (cp *ChargePoint) UpdateFirmware(ctx context.Context, req firmware.UpdateFirmwareRequest) (*firmware.UpdateFirmwareConfirmation, error) {
	return call[firmware.UpdateFirmwareConfirmation](ctx, cp, req)
}

// GetLocalListVersion reads the version of the charge point's local
// authorisation list.
func (cp *ChargePoint) GetLocalListVersion(ctx context.Context, req localauth.GetLocalListVersionRequest) (*localauth.GetLocalListVersionConfirmation, error) {
	return call[localauth.GetLocalListVersionConfirmation](ctx, cp, req)
}

// SendLocalList replaces or amends the charge point's local authorisation list.
func (cp *ChargePoint) SendLocalList(ctx context.Context, req localauth.SendLocalListRequest) (*localauth.SendLocalListConfirmation, error) {
	return call[localauth.SendLocalListConfirmation](ctx, cp, req)
}

// ReserveNow reserves a connector for an identifier.
func (cp *ChargePoint) ReserveNow(ctx context.Context, req reservation.ReserveNowRequest) (*reservation.ReserveNowConfirmation, error) {
	return call[reservation.ReserveNowConfirmation](ctx, cp, req)
}

// CancelReservation cancels a reservation.
func (cp *ChargePoint) CancelReservation(ctx context.Context, req reservation.CancelReservationRequest) (*reservation.CancelReservationConfirmation, error) {
	return call[reservation.CancelReservationConfirmation](ctx, cp, req)
}

// SetChargingProfile installs a charging profile.
func (cp *ChargePoint) SetChargingProfile(ctx context.Context, req smartcharging.SetChargingProfileRequest) (*smartcharging.SetChargingProfileConfirmation, error) {
	return call[smartcharging.SetChargingProfileConfirmation](ctx, cp, req)
}

// ClearChargingProfile removes charging profiles.
func (cp *ChargePoint) ClearChargingProfile(ctx context.Context, req smartcharging.ClearChargingProfileRequest) (*smartcharging.ClearChargingProfileConfirmation, error) {
	return call[smartcharging.ClearChargingProfileConfirmation](ctx, cp, req)
}

// GetCompositeSchedule asks for the schedule the charge point will follow.
func (cp *ChargePoint) GetCompositeSchedule(ctx context.Context, req smartcharging.GetCompositeScheduleRequest) (*smartcharging.GetCompositeScheduleConfirmation, error) {
	return call[smartcharging.GetCompositeScheduleConfirmation](ctx, cp, req)
}

// TriggerMessage asks the charge point to send one of its own messages.
func (cp *ChargePoint) TriggerMessage(ctx context.Context, req remotetrigger.TriggerMessageRequest) (*remotetrigger.TriggerMessageConfirmation, error) {
	return call[remotetrigger.TriggerMessageConfirmation](ctx, cp, req)
}
