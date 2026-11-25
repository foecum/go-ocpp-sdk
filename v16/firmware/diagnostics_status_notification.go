package firmware

import "github.com/foecum/go-ocpp-sdk/v16/types"

// DiagnosticsStatusNotificationAction is the action name of the
// DiagnosticsStatusNotification message.
const DiagnosticsStatusNotificationAction = "DiagnosticsStatusNotification"

// DiagnosticsStatusNotificationRequest is the DiagnosticsStatusNotification.req
// PDU, sent by the Charge Point to report the progress of a diagnostics upload.
type DiagnosticsStatusNotificationRequest struct {
	// Status is the status of the upload.
	Status types.DiagnosticsStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Request.
func (DiagnosticsStatusNotificationRequest) Action() string {
	return DiagnosticsStatusNotificationAction
}

// Validate implements ocpp.Request.
func (r DiagnosticsStatusNotificationRequest) Validate() error {
	return types.CheckEnum("status", r.Status, true)
}

// DiagnosticsStatusNotificationConfirmation is the
// DiagnosticsStatusNotification.conf PDU. It has no fields.
type DiagnosticsStatusNotificationConfirmation struct{}

// Action implements ocpp.Response.
func (DiagnosticsStatusNotificationConfirmation) Action() string {
	return DiagnosticsStatusNotificationAction
}

// Validate implements ocpp.Response.
func (DiagnosticsStatusNotificationConfirmation) Validate() error { return nil }
