package chargepoint

import (
	"context"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/rpc"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/firmware"
	"github.com/foecum/go-ocpp-sdk/v16/localauth"
	"github.com/foecum/go-ocpp-sdk/v16/remotetrigger"
	"github.com/foecum/go-ocpp-sdk/v16/reservation"
	"github.com/foecum/go-ocpp-sdk/v16/smartcharging"
)

// dispatch routes an incoming request to the handler for its profile. A
// request the SDK knows but nobody handles is NotSupported.
func (c *ChargePoint) dispatch(ctx context.Context, req ocpp.Request) (ocpp.Response, error) {
	return rpc.Safe(c.log, req.Action(), func() (ocpp.Response, error) {
		switch r := req.(type) {
		case *core.ChangeAvailabilityRequest:
			if h := c.cfg.Core; h != nil {
				return rpc.Return(h.ChangeAvailability(ctx, r))
			}
		case *core.ChangeConfigurationRequest:
			if h := c.cfg.Core; h != nil {
				return rpc.Return(h.ChangeConfiguration(ctx, r))
			}
		case *core.ClearCacheRequest:
			if h := c.cfg.Core; h != nil {
				return rpc.Return(h.ClearCache(ctx, r))
			}
		case *core.DataTransferRequest:
			if h := c.cfg.Core; h != nil {
				return rpc.Return(h.DataTransfer(ctx, r))
			}
		case *core.GetConfigurationRequest:
			if h := c.cfg.Core; h != nil {
				return rpc.Return(h.GetConfiguration(ctx, r))
			}
		case *core.RemoteStartTransactionRequest:
			if h := c.cfg.Core; h != nil {
				return rpc.Return(h.RemoteStartTransaction(ctx, r))
			}
		case *core.RemoteStopTransactionRequest:
			if h := c.cfg.Core; h != nil {
				return rpc.Return(h.RemoteStopTransaction(ctx, r))
			}
		case *core.ResetRequest:
			if h := c.cfg.Core; h != nil {
				return rpc.Return(h.Reset(ctx, r))
			}
		case *core.UnlockConnectorRequest:
			if h := c.cfg.Core; h != nil {
				return rpc.Return(h.UnlockConnector(ctx, r))
			}
		case *firmware.GetDiagnosticsRequest:
			if h := c.cfg.Firmware; h != nil {
				return rpc.Return(h.GetDiagnostics(ctx, r))
			}
		case *firmware.UpdateFirmwareRequest:
			if h := c.cfg.Firmware; h != nil {
				return rpc.Return(h.UpdateFirmware(ctx, r))
			}
		case *localauth.GetLocalListVersionRequest:
			if h := c.cfg.LocalAuth; h != nil {
				return rpc.Return(h.GetLocalListVersion(ctx, r))
			}
		case *localauth.SendLocalListRequest:
			if h := c.cfg.LocalAuth; h != nil {
				return rpc.Return(h.SendLocalList(ctx, r))
			}
		case *reservation.ReserveNowRequest:
			if h := c.cfg.Reservation; h != nil {
				return rpc.Return(h.ReserveNow(ctx, r))
			}
		case *reservation.CancelReservationRequest:
			if h := c.cfg.Reservation; h != nil {
				return rpc.Return(h.CancelReservation(ctx, r))
			}
		case *smartcharging.SetChargingProfileRequest:
			if h := c.cfg.SmartCharging; h != nil {
				return rpc.Return(h.SetChargingProfile(ctx, r))
			}
		case *smartcharging.ClearChargingProfileRequest:
			if h := c.cfg.SmartCharging; h != nil {
				return rpc.Return(h.ClearChargingProfile(ctx, r))
			}
		case *smartcharging.GetCompositeScheduleRequest:
			if h := c.cfg.SmartCharging; h != nil {
				return rpc.Return(h.GetCompositeSchedule(ctx, r))
			}
		case *remotetrigger.TriggerMessageRequest:
			if h := c.cfg.RemoteTrigger; h != nil {
				return rpc.Return(h.TriggerMessage(ctx, r))
			}
		}
		// No handler for the profile, or a message the SDK knows but the
		// Central System does not send to a charge point (for example
		// BootNotification).
		return nil, rpc.NotSupported(req.Action())
	})
}
