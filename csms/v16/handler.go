package csms

import (
	"context"

	"github.com/foecum/go-ocpp-sdk/internal/rpc"
	"github.com/foecum/go-ocpp-sdk/v16/core"
)

// CoreHandler handles the Core profile calls a charge point sends to the
// Central System. Embed UnimplementedCoreHandler to implement only some of them.
type CoreHandler interface {
	// Authorize answers an Authorize request.
	Authorize(ctx context.Context, cp *ChargePoint, req *core.AuthorizeRequest) (*core.AuthorizeConfirmation, error)
	// BootNotification answers a BootNotification request.
	BootNotification(ctx context.Context, cp *ChargePoint, req *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error)
	// DataTransfer answers a DataTransfer request.
	DataTransfer(ctx context.Context, cp *ChargePoint, req *core.DataTransferRequest) (*core.DataTransferConfirmation, error)
	// Heartbeat answers a Heartbeat request.
	Heartbeat(ctx context.Context, cp *ChargePoint, req *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error)
	// MeterValues answers a MeterValues request.
	MeterValues(ctx context.Context, cp *ChargePoint, req *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error)
	// StartTransaction answers a StartTransaction request.
	StartTransaction(ctx context.Context, cp *ChargePoint, req *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error)
	// StatusNotification answers a StatusNotification request.
	StatusNotification(ctx context.Context, cp *ChargePoint, req *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error)
	// StopTransaction answers a StopTransaction request.
	StopTransaction(ctx context.Context, cp *ChargePoint, req *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error)
}

// UnimplementedCoreHandler implements CoreHandler by answering every call with
// a NotSupported CALLERROR. Embed it in your handler and override what you
// support.
type UnimplementedCoreHandler struct{}

var _ CoreHandler = UnimplementedCoreHandler{}

// Authorize returns NotSupported.
func (UnimplementedCoreHandler) Authorize(context.Context, *ChargePoint, *core.AuthorizeRequest) (*core.AuthorizeConfirmation, error) {
	return nil, rpc.NotSupported(core.AuthorizeAction)
}

// BootNotification returns NotSupported.
func (UnimplementedCoreHandler) BootNotification(context.Context, *ChargePoint, *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
	return nil, rpc.NotSupported(core.BootNotificationAction)
}

// DataTransfer returns NotSupported.
func (UnimplementedCoreHandler) DataTransfer(context.Context, *ChargePoint, *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	return nil, rpc.NotSupported(core.DataTransferAction)
}

// Heartbeat returns NotSupported.
func (UnimplementedCoreHandler) Heartbeat(context.Context, *ChargePoint, *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
	return nil, rpc.NotSupported(core.HeartbeatAction)
}

// MeterValues returns NotSupported.
func (UnimplementedCoreHandler) MeterValues(context.Context, *ChargePoint, *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	return nil, rpc.NotSupported(core.MeterValuesAction)
}

// StartTransaction returns NotSupported.
func (UnimplementedCoreHandler) StartTransaction(context.Context, *ChargePoint, *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	return nil, rpc.NotSupported(core.StartTransactionAction)
}

// StatusNotification returns NotSupported.
func (UnimplementedCoreHandler) StatusNotification(context.Context, *ChargePoint, *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
	return nil, rpc.NotSupported(core.StatusNotificationAction)
}

// StopTransaction returns NotSupported.
func (UnimplementedCoreHandler) StopTransaction(context.Context, *ChargePoint, *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	return nil, rpc.NotSupported(core.StopTransactionAction)
}
