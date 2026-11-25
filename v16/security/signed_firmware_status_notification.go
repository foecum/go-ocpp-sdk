package security

import "github.com/foecum/go-ocpp-sdk/v16/types"

// SignedFirmwareStatusNotificationAction is the action name of the
// SignedFirmwareStatusNotification message.
const SignedFirmwareStatusNotificationAction = "SignedFirmwareStatusNotification"

// SignedFirmwareStatusNotificationRequest is the
// SignedFirmwareStatusNotification.req PDU, sent by the Charge Point to report
// the progress of a signed firmware update.
type SignedFirmwareStatusNotificationRequest struct {
	// Status is the progress of the update.
	Status SignedFirmwareStatus `json:"status" ocpp:"required"`
	// RequestID is the id from the SignedUpdateFirmware request that started the
	// update. It may only be absent if the message was triggered while no update
	// is in progress.
	RequestID *int `json:"requestId,omitempty"`
}

// Action implements ocpp.Request.
func (SignedFirmwareStatusNotificationRequest) Action() string {
	return SignedFirmwareStatusNotificationAction
}

// Validate implements ocpp.Request.
func (r SignedFirmwareStatusNotificationRequest) Validate() error {
	return types.CheckEnum("status", r.Status, true)
}

// SignedFirmwareStatusNotificationConfirmation is the
// SignedFirmwareStatusNotification.conf PDU. It has no fields.
type SignedFirmwareStatusNotificationConfirmation struct{}

// Action implements ocpp.Response.
func (SignedFirmwareStatusNotificationConfirmation) Action() string {
	return SignedFirmwareStatusNotificationAction
}

// Validate implements ocpp.Response.
func (SignedFirmwareStatusNotificationConfirmation) Validate() error { return nil }
