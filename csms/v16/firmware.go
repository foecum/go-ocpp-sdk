package csms

import (
	"context"

	"github.com/foecum/go-ocpp-sdk/internal/rpc"
	"github.com/foecum/go-ocpp-sdk/v16/firmware"
)

// FirmwareHandler handles the Firmware Management profile calls a charge point
// sends to the Central System. Embed UnimplementedFirmwareHandler to implement
// only some of them.
type FirmwareHandler interface {
	// DiagnosticsStatusNotification answers a DiagnosticsStatusNotification request.
	DiagnosticsStatusNotification(ctx context.Context, cp *ChargePoint, req *firmware.DiagnosticsStatusNotificationRequest) (*firmware.DiagnosticsStatusNotificationConfirmation, error)
	// FirmwareStatusNotification answers a FirmwareStatusNotification request.
	FirmwareStatusNotification(ctx context.Context, cp *ChargePoint, req *firmware.FirmwareStatusNotificationRequest) (*firmware.FirmwareStatusNotificationConfirmation, error)
}

// UnimplementedFirmwareHandler implements FirmwareHandler by answering every
// call with a NotSupported CALLERROR.
type UnimplementedFirmwareHandler struct{}

var _ FirmwareHandler = UnimplementedFirmwareHandler{}

// DiagnosticsStatusNotification returns NotSupported.
func (UnimplementedFirmwareHandler) DiagnosticsStatusNotification(context.Context, *ChargePoint, *firmware.DiagnosticsStatusNotificationRequest) (*firmware.DiagnosticsStatusNotificationConfirmation, error) {
	return nil, rpc.NotSupported(firmware.DiagnosticsStatusNotificationAction)
}

// FirmwareStatusNotification returns NotSupported.
func (UnimplementedFirmwareHandler) FirmwareStatusNotification(context.Context, *ChargePoint, *firmware.FirmwareStatusNotificationRequest) (*firmware.FirmwareStatusNotificationConfirmation, error) {
	return nil, rpc.NotSupported(firmware.FirmwareStatusNotificationAction)
}
