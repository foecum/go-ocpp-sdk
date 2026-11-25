package security

import "github.com/foecum/go-ocpp-sdk/v16/types"

// LogStatusNotificationAction is the action name of the LogStatusNotification message.
const LogStatusNotificationAction = "LogStatusNotification"

// LogStatusNotificationRequest is the LogStatusNotification.req PDU, sent by
// the Charge Point to report the progress of a log upload.
type LogStatusNotificationRequest struct {
	// Status is the status of the upload.
	Status UploadLogStatus `json:"status" ocpp:"required"`
	// RequestID is the id from the GetLog request that started the upload.
	RequestID *int `json:"requestId,omitempty"`
}

// Action implements ocpp.Request.
func (LogStatusNotificationRequest) Action() string { return LogStatusNotificationAction }

// Validate implements ocpp.Request.
func (r LogStatusNotificationRequest) Validate() error {
	return types.CheckEnum("status", r.Status, true)
}

// LogStatusNotificationConfirmation is the LogStatusNotification.conf PDU. It
// has no fields.
type LogStatusNotificationConfirmation struct{}

// Action implements ocpp.Response.
func (LogStatusNotificationConfirmation) Action() string { return LogStatusNotificationAction }

// Validate implements ocpp.Response.
func (LogStatusNotificationConfirmation) Validate() error { return nil }
