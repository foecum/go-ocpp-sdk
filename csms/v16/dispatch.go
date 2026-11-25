package csms

import (
	"context"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/rpc"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/firmware"
)

// dispatch routes an incoming request to the handler for its profile. A
// request the SDK knows but nobody handles is NotSupported.
func (s *CSMS) dispatch(ctx context.Context, cp *ChargePoint, req ocpp.Request) (ocpp.Response, error) {
	return rpc.Safe(s.log, req.Action(), func() (ocpp.Response, error) {
		switch r := req.(type) {
		case *core.AuthorizeRequest:
			if h := s.cfg.Core; h != nil {
				return rpc.Return(h.Authorize(ctx, cp, r))
			}
		case *core.BootNotificationRequest:
			if h := s.cfg.Core; h != nil {
				return rpc.Return(h.BootNotification(ctx, cp, r))
			}
		case *core.DataTransferRequest:
			if h := s.cfg.Core; h != nil {
				return rpc.Return(h.DataTransfer(ctx, cp, r))
			}
		case *core.HeartbeatRequest:
			if h := s.cfg.Core; h != nil {
				return rpc.Return(h.Heartbeat(ctx, cp, r))
			}
		case *core.MeterValuesRequest:
			if h := s.cfg.Core; h != nil {
				return rpc.Return(h.MeterValues(ctx, cp, r))
			}
		case *core.StartTransactionRequest:
			if h := s.cfg.Core; h != nil {
				return rpc.Return(h.StartTransaction(ctx, cp, r))
			}
		case *core.StatusNotificationRequest:
			if h := s.cfg.Core; h != nil {
				return rpc.Return(h.StatusNotification(ctx, cp, r))
			}
		case *core.StopTransactionRequest:
			if h := s.cfg.Core; h != nil {
				return rpc.Return(h.StopTransaction(ctx, cp, r))
			}
		case *firmware.DiagnosticsStatusNotificationRequest:
			if h := s.cfg.Firmware; h != nil {
				return rpc.Return(h.DiagnosticsStatusNotification(ctx, cp, r))
			}
		case *firmware.FirmwareStatusNotificationRequest:
			if h := s.cfg.Firmware; h != nil {
				return rpc.Return(h.FirmwareStatusNotification(ctx, cp, r))
			}
		}
		// No handler for the profile, or a message the SDK knows but a charge
		// point does not send to the Central System (for example
		// ChangeAvailability).
		return nil, rpc.NotSupported(req.Action())
	})
}
