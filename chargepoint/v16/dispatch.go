package chargepoint

import (
	"context"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/rpc"
	"github.com/foecum/go-ocpp-sdk/v16/core"
)

// dispatch routes an incoming request to the handler for its profile. A
// request the SDK knows but nobody handles is NotSupported.
func (c *ChargePoint) dispatch(ctx context.Context, req ocpp.Request) (ocpp.Response, error) {
	return rpc.Safe(c.log, req.Action(), func() (ocpp.Response, error) {
		h := c.cfg.Core
		if h == nil {
			return nil, rpc.NotSupported(req.Action())
		}
		switch r := req.(type) {
		case *core.ChangeAvailabilityRequest:
			return rpc.Return(h.ChangeAvailability(ctx, r))
		case *core.ChangeConfigurationRequest:
			return rpc.Return(h.ChangeConfiguration(ctx, r))
		case *core.ClearCacheRequest:
			return rpc.Return(h.ClearCache(ctx, r))
		case *core.DataTransferRequest:
			return rpc.Return(h.DataTransfer(ctx, r))
		case *core.GetConfigurationRequest:
			return rpc.Return(h.GetConfiguration(ctx, r))
		case *core.RemoteStartTransactionRequest:
			return rpc.Return(h.RemoteStartTransaction(ctx, r))
		case *core.RemoteStopTransactionRequest:
			return rpc.Return(h.RemoteStopTransaction(ctx, r))
		case *core.ResetRequest:
			return rpc.Return(h.Reset(ctx, r))
		case *core.UnlockConnectorRequest:
			return rpc.Return(h.UnlockConnector(ctx, r))
		}
		// A message the SDK knows but the Central System does not send to a
		// charge point (for example BootNotification).
		return nil, rpc.NotSupported(req.Action())
	})
}
