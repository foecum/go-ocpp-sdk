package core

import "github.com/foecum/go-ocpp-sdk/v16/types"

// DataTransferAction is the action name of the DataTransfer message.
const DataTransferAction = "DataTransfer"

// DataTransferRequest is the DataTransfer.req PDU, which either side can send
// to exchange vendor-specific data.
type DataTransferRequest struct {
	// VendorID identifies the vendor-specific implementation (at most 255 characters).
	VendorID string `json:"vendorId" ocpp:"required"`
	// MessageID is an additional identifier (at most 50 characters).
	MessageID string `json:"messageId,omitempty"`
	// Data is text of undefined length and format.
	Data string `json:"data,omitempty"`
}

// Action implements ocpp.Request.
func (DataTransferRequest) Action() string { return DataTransferAction }

// Validate implements ocpp.Request.
func (r DataTransferRequest) Validate() error {
	return types.First(
		types.CheckString("vendorId", r.VendorID, types.CiString255, true),
		types.CheckString("messageId", r.MessageID, types.CiString50, false),
	)
}

// DataTransferConfirmation is the DataTransfer.conf PDU.
type DataTransferConfirmation struct {
	// Status is the outcome.
	Status types.DataTransferStatus `json:"status" ocpp:"required"`
	// Data is the response data, of undefined length and format.
	Data string `json:"data,omitempty"`
}

// Action implements ocpp.Response.
func (DataTransferConfirmation) Action() string { return DataTransferAction }

// Validate implements ocpp.Response.
func (c DataTransferConfirmation) Validate() error { return types.CheckEnum("status", c.Status, true) }
