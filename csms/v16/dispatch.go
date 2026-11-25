package csms

import (
	"context"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/rpc"
	"github.com/foecum/go-ocpp-sdk/v16/core"
)

// dispatch routes an incoming request to the handler for its profile. A
// request the SDK knows but nobody handles is NotSupported.
func (s *CSMS) dispatch(ctx context.Context, cp *ChargePoint, req ocpp.Request) (ocpp.Response, error) {
	return rpc.Safe(s.log, req.Action(), func() (ocpp.Response, error) {
		h := s.cfg.Core
		if h == nil {
			return nil, rpc.NotSupported(req.Action())
		}
		switch r := req.(type) {
		case *core.AuthorizeRequest:
			return rpc.Return(h.Authorize(ctx, cp, r))
		case *core.BootNotificationRequest:
			return rpc.Return(h.BootNotification(ctx, cp, r))
		case *core.DataTransferRequest:
			return rpc.Return(h.DataTransfer(ctx, cp, r))
		case *core.HeartbeatRequest:
			return rpc.Return(h.Heartbeat(ctx, cp, r))
		case *core.MeterValuesRequest:
			return rpc.Return(h.MeterValues(ctx, cp, r))
		case *core.StartTransactionRequest:
			return rpc.Return(h.StartTransaction(ctx, cp, r))
		case *core.StatusNotificationRequest:
			return rpc.Return(h.StatusNotification(ctx, cp, r))
		case *core.StopTransactionRequest:
			return rpc.Return(h.StopTransaction(ctx, cp, r))
		}
		// A message the SDK knows but a charge point does not send to the
		// Central System (for example ChangeAvailability).
		return nil, rpc.NotSupported(req.Action())
	})
}
