package chargepoint

import (
	"context"

	"github.com/foecum/go-ocpp-sdk/internal/rpc"
	"github.com/foecum/go-ocpp-sdk/v16/firmware"
	"github.com/foecum/go-ocpp-sdk/v16/localauth"
	"github.com/foecum/go-ocpp-sdk/v16/remotetrigger"
	"github.com/foecum/go-ocpp-sdk/v16/reservation"
	"github.com/foecum/go-ocpp-sdk/v16/smartcharging"
)

// FirmwareHandler handles the Firmware Management calls the Central System
// sends. Embed UnimplementedFirmwareHandler to implement only some of them.
type FirmwareHandler interface {
	// GetDiagnostics answers a GetDiagnostics request.
	GetDiagnostics(ctx context.Context, req *firmware.GetDiagnosticsRequest) (*firmware.GetDiagnosticsConfirmation, error)
	// UpdateFirmware answers an UpdateFirmware request.
	UpdateFirmware(ctx context.Context, req *firmware.UpdateFirmwareRequest) (*firmware.UpdateFirmwareConfirmation, error)
}

// LocalAuthHandler handles the Local Auth List Management calls the Central
// System sends.
type LocalAuthHandler interface {
	// GetLocalListVersion answers a GetLocalListVersion request.
	GetLocalListVersion(ctx context.Context, req *localauth.GetLocalListVersionRequest) (*localauth.GetLocalListVersionConfirmation, error)
	// SendLocalList answers a SendLocalList request.
	SendLocalList(ctx context.Context, req *localauth.SendLocalListRequest) (*localauth.SendLocalListConfirmation, error)
}

// ReservationHandler handles the Reservation calls the Central System sends.
type ReservationHandler interface {
	// ReserveNow answers a ReserveNow request.
	ReserveNow(ctx context.Context, req *reservation.ReserveNowRequest) (*reservation.ReserveNowConfirmation, error)
	// CancelReservation answers a CancelReservation request.
	CancelReservation(ctx context.Context, req *reservation.CancelReservationRequest) (*reservation.CancelReservationConfirmation, error)
}

// SmartChargingHandler handles the Smart Charging calls the Central System sends.
type SmartChargingHandler interface {
	// SetChargingProfile answers a SetChargingProfile request.
	SetChargingProfile(ctx context.Context, req *smartcharging.SetChargingProfileRequest) (*smartcharging.SetChargingProfileConfirmation, error)
	// ClearChargingProfile answers a ClearChargingProfile request.
	ClearChargingProfile(ctx context.Context, req *smartcharging.ClearChargingProfileRequest) (*smartcharging.ClearChargingProfileConfirmation, error)
	// GetCompositeSchedule answers a GetCompositeSchedule request.
	GetCompositeSchedule(ctx context.Context, req *smartcharging.GetCompositeScheduleRequest) (*smartcharging.GetCompositeScheduleConfirmation, error)
}

// RemoteTriggerHandler handles the Remote Trigger calls the Central System sends.
type RemoteTriggerHandler interface {
	// TriggerMessage answers a TriggerMessage request.
	TriggerMessage(ctx context.Context, req *remotetrigger.TriggerMessageRequest) (*remotetrigger.TriggerMessageConfirmation, error)
}

// UnimplementedFirmwareHandler implements FirmwareHandler by answering every
// call with a NotSupported CALLERROR.
type UnimplementedFirmwareHandler struct{}

// UnimplementedLocalAuthHandler implements LocalAuthHandler by answering every
// call with a NotSupported CALLERROR.
type UnimplementedLocalAuthHandler struct{}

// UnimplementedReservationHandler implements ReservationHandler by answering
// every call with a NotSupported CALLERROR.
type UnimplementedReservationHandler struct{}

// UnimplementedSmartChargingHandler implements SmartChargingHandler by
// answering every call with a NotSupported CALLERROR.
type UnimplementedSmartChargingHandler struct{}

// UnimplementedRemoteTriggerHandler implements RemoteTriggerHandler by
// answering every call with a NotSupported CALLERROR.
type UnimplementedRemoteTriggerHandler struct{}

var (
	_ FirmwareHandler      = UnimplementedFirmwareHandler{}
	_ LocalAuthHandler     = UnimplementedLocalAuthHandler{}
	_ ReservationHandler   = UnimplementedReservationHandler{}
	_ SmartChargingHandler = UnimplementedSmartChargingHandler{}
	_ RemoteTriggerHandler = UnimplementedRemoteTriggerHandler{}
)

// GetDiagnostics returns NotSupported.
func (UnimplementedFirmwareHandler) GetDiagnostics(context.Context, *firmware.GetDiagnosticsRequest) (*firmware.GetDiagnosticsConfirmation, error) {
	return nil, rpc.NotSupported(firmware.GetDiagnosticsAction)
}

// UpdateFirmware returns NotSupported.
func (UnimplementedFirmwareHandler) UpdateFirmware(context.Context, *firmware.UpdateFirmwareRequest) (*firmware.UpdateFirmwareConfirmation, error) {
	return nil, rpc.NotSupported(firmware.UpdateFirmwareAction)
}

// GetLocalListVersion returns NotSupported.
func (UnimplementedLocalAuthHandler) GetLocalListVersion(context.Context, *localauth.GetLocalListVersionRequest) (*localauth.GetLocalListVersionConfirmation, error) {
	return nil, rpc.NotSupported(localauth.GetLocalListVersionAction)
}

// SendLocalList returns NotSupported.
func (UnimplementedLocalAuthHandler) SendLocalList(context.Context, *localauth.SendLocalListRequest) (*localauth.SendLocalListConfirmation, error) {
	return nil, rpc.NotSupported(localauth.SendLocalListAction)
}

// ReserveNow returns NotSupported.
func (UnimplementedReservationHandler) ReserveNow(context.Context, *reservation.ReserveNowRequest) (*reservation.ReserveNowConfirmation, error) {
	return nil, rpc.NotSupported(reservation.ReserveNowAction)
}

// CancelReservation returns NotSupported.
func (UnimplementedReservationHandler) CancelReservation(context.Context, *reservation.CancelReservationRequest) (*reservation.CancelReservationConfirmation, error) {
	return nil, rpc.NotSupported(reservation.CancelReservationAction)
}

// SetChargingProfile returns NotSupported.
func (UnimplementedSmartChargingHandler) SetChargingProfile(context.Context, *smartcharging.SetChargingProfileRequest) (*smartcharging.SetChargingProfileConfirmation, error) {
	return nil, rpc.NotSupported(smartcharging.SetChargingProfileAction)
}

// ClearChargingProfile returns NotSupported.
func (UnimplementedSmartChargingHandler) ClearChargingProfile(context.Context, *smartcharging.ClearChargingProfileRequest) (*smartcharging.ClearChargingProfileConfirmation, error) {
	return nil, rpc.NotSupported(smartcharging.ClearChargingProfileAction)
}

// GetCompositeSchedule returns NotSupported.
func (UnimplementedSmartChargingHandler) GetCompositeSchedule(context.Context, *smartcharging.GetCompositeScheduleRequest) (*smartcharging.GetCompositeScheduleConfirmation, error) {
	return nil, rpc.NotSupported(smartcharging.GetCompositeScheduleAction)
}

// TriggerMessage returns NotSupported.
func (UnimplementedRemoteTriggerHandler) TriggerMessage(context.Context, *remotetrigger.TriggerMessageRequest) (*remotetrigger.TriggerMessageConfirmation, error) {
	return nil, rpc.NotSupported(remotetrigger.TriggerMessageAction)
}
