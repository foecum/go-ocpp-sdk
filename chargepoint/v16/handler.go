package chargepoint

import (
	"context"

	"github.com/foecum/go-ocpp-sdk/internal/rpc"
	"github.com/foecum/go-ocpp-sdk/v16/core"
)

// CoreHandler handles the Core profile calls the Central System sends to the
// charge point. Embed UnimplementedCoreHandler to implement only some of them.
type CoreHandler interface {
	// ChangeAvailability answers a ChangeAvailability request.
	ChangeAvailability(ctx context.Context, req *core.ChangeAvailabilityRequest) (*core.ChangeAvailabilityConfirmation, error)
	// ChangeConfiguration answers a ChangeConfiguration request.
	ChangeConfiguration(ctx context.Context, req *core.ChangeConfigurationRequest) (*core.ChangeConfigurationConfirmation, error)
	// ClearCache answers a ClearCache request.
	ClearCache(ctx context.Context, req *core.ClearCacheRequest) (*core.ClearCacheConfirmation, error)
	// DataTransfer answers a DataTransfer request.
	DataTransfer(ctx context.Context, req *core.DataTransferRequest) (*core.DataTransferConfirmation, error)
	// GetConfiguration answers a GetConfiguration request.
	GetConfiguration(ctx context.Context, req *core.GetConfigurationRequest) (*core.GetConfigurationConfirmation, error)
	// RemoteStartTransaction answers a RemoteStartTransaction request.
	RemoteStartTransaction(ctx context.Context, req *core.RemoteStartTransactionRequest) (*core.RemoteStartTransactionConfirmation, error)
	// RemoteStopTransaction answers a RemoteStopTransaction request.
	RemoteStopTransaction(ctx context.Context, req *core.RemoteStopTransactionRequest) (*core.RemoteStopTransactionConfirmation, error)
	// Reset answers a Reset request.
	Reset(ctx context.Context, req *core.ResetRequest) (*core.ResetConfirmation, error)
	// UnlockConnector answers an UnlockConnector request.
	UnlockConnector(ctx context.Context, req *core.UnlockConnectorRequest) (*core.UnlockConnectorConfirmation, error)
}

// UnimplementedCoreHandler implements CoreHandler by answering every call with
// a NotSupported CALLERROR. Embed it in your handler and override what you
// support.
type UnimplementedCoreHandler struct{}

var _ CoreHandler = UnimplementedCoreHandler{}

// ChangeAvailability returns NotSupported.
func (UnimplementedCoreHandler) ChangeAvailability(context.Context, *core.ChangeAvailabilityRequest) (*core.ChangeAvailabilityConfirmation, error) {
	return nil, rpc.NotSupported(core.ChangeAvailabilityAction)
}

// ChangeConfiguration returns NotSupported.
func (UnimplementedCoreHandler) ChangeConfiguration(context.Context, *core.ChangeConfigurationRequest) (*core.ChangeConfigurationConfirmation, error) {
	return nil, rpc.NotSupported(core.ChangeConfigurationAction)
}

// ClearCache returns NotSupported.
func (UnimplementedCoreHandler) ClearCache(context.Context, *core.ClearCacheRequest) (*core.ClearCacheConfirmation, error) {
	return nil, rpc.NotSupported(core.ClearCacheAction)
}

// DataTransfer returns NotSupported.
func (UnimplementedCoreHandler) DataTransfer(context.Context, *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	return nil, rpc.NotSupported(core.DataTransferAction)
}

// GetConfiguration returns NotSupported.
func (UnimplementedCoreHandler) GetConfiguration(context.Context, *core.GetConfigurationRequest) (*core.GetConfigurationConfirmation, error) {
	return nil, rpc.NotSupported(core.GetConfigurationAction)
}

// RemoteStartTransaction returns NotSupported.
func (UnimplementedCoreHandler) RemoteStartTransaction(context.Context, *core.RemoteStartTransactionRequest) (*core.RemoteStartTransactionConfirmation, error) {
	return nil, rpc.NotSupported(core.RemoteStartTransactionAction)
}

// RemoteStopTransaction returns NotSupported.
func (UnimplementedCoreHandler) RemoteStopTransaction(context.Context, *core.RemoteStopTransactionRequest) (*core.RemoteStopTransactionConfirmation, error) {
	return nil, rpc.NotSupported(core.RemoteStopTransactionAction)
}

// Reset returns NotSupported.
func (UnimplementedCoreHandler) Reset(context.Context, *core.ResetRequest) (*core.ResetConfirmation, error) {
	return nil, rpc.NotSupported(core.ResetAction)
}

// UnlockConnector returns NotSupported.
func (UnimplementedCoreHandler) UnlockConnector(context.Context, *core.UnlockConnectorRequest) (*core.UnlockConnectorConfirmation, error) {
	return nil, rpc.NotSupported(core.UnlockConnectorAction)
}
