package core

import "github.com/foecum/go-ocpp-sdk/v16/types"

// StatusNotificationAction is the action name of the StatusNotification message.
const StatusNotificationAction = "StatusNotification"

// StatusNotificationRequest is the StatusNotification.req PDU, sent by the
// Charge Point when the status of a connector changes.
type StatusNotificationRequest struct {
	// ConnectorID is the connector; 0 is the Charge Point main controller.
	ConnectorID int `json:"connectorId" ocpp:"required"`
	// ErrorCode is the error reported.
	ErrorCode types.ChargePointErrorCode `json:"errorCode" ocpp:"required"`
	// Info is free-form additional information (at most 50 characters).
	Info string `json:"info,omitempty"`
	// Status is the current status.
	Status types.ChargePointStatus `json:"status" ocpp:"required"`
	// Timestamp is when the status applies; if absent, the time of receipt.
	Timestamp *types.DateTime `json:"timestamp,omitempty"`
	// VendorID identifies the vendor-specific implementation (at most 255 characters).
	VendorID string `json:"vendorId,omitempty"`
	// VendorErrorCode is a vendor-specific error code (at most 50 characters).
	VendorErrorCode string `json:"vendorErrorCode,omitempty"`
}

// Action implements ocpp.Request.
func (StatusNotificationRequest) Action() string { return StatusNotificationAction }

// Validate implements ocpp.Request.
func (r StatusNotificationRequest) Validate() error {
	return types.First(
		types.CheckMin("connectorId", r.ConnectorID, 0),
		types.CheckEnum("errorCode", r.ErrorCode, true),
		types.CheckString("info", r.Info, types.CiString50, false),
		types.CheckEnum("status", r.Status, true),
		types.CheckString("vendorId", r.VendorID, types.CiString255, false),
		types.CheckString("vendorErrorCode", r.VendorErrorCode, types.CiString50, false),
	)
}

// StatusNotificationConfirmation is the StatusNotification.conf PDU. It has no fields.
type StatusNotificationConfirmation struct{}

// Action implements ocpp.Response.
func (StatusNotificationConfirmation) Action() string { return StatusNotificationAction }

// Validate implements ocpp.Response.
func (StatusNotificationConfirmation) Validate() error { return nil }
