package firmware

import "github.com/foecum/go-ocpp-sdk/v16/types"

// FirmwareStatusNotificationAction is the action name of the
// FirmwareStatusNotification message.
const FirmwareStatusNotificationAction = "FirmwareStatusNotification"

// FirmwareStatusNotificationRequest is the FirmwareStatusNotification.req PDU,
// sent by the Charge Point to report the progress of a firmware update.
type FirmwareStatusNotificationRequest struct {
	// Status is the progress of the download and installation.
	Status types.FirmwareStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Request.
func (FirmwareStatusNotificationRequest) Action() string { return FirmwareStatusNotificationAction }

// Validate implements ocpp.Request.
func (r FirmwareStatusNotificationRequest) Validate() error {
	return types.CheckEnum("status", r.Status, true)
}

// FirmwareStatusNotificationConfirmation is the FirmwareStatusNotification.conf
// PDU. It has no fields.
type FirmwareStatusNotificationConfirmation struct{}

// Action implements ocpp.Response.
func (FirmwareStatusNotificationConfirmation) Action() string {
	return FirmwareStatusNotificationAction
}

// Validate implements ocpp.Response.
func (FirmwareStatusNotificationConfirmation) Validate() error { return nil }
