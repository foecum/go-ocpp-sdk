package security

import "github.com/foecum/go-ocpp-sdk/v16/types"

// ExtendedTriggerMessageAction is the action name of the ExtendedTriggerMessage message.
const ExtendedTriggerMessageAction = "ExtendedTriggerMessage"

// ExtendedTriggerMessageRequest is the ExtendedTriggerMessage.req PDU, sent by
// the Central System to ask the Charge Point to send a message. It extends
// TriggerMessage with the messages of the security extension, for example
// SignChargePointCertificate.
type ExtendedTriggerMessageRequest struct {
	// RequestedMessage is the message to send.
	RequestedMessage ExtendedMessageTrigger `json:"requestedMessage" ocpp:"required"`
	// ConnectorID is the connector the request applies to, if any. When present
	// it must be greater than 0.
	ConnectorID *int `json:"connectorId,omitempty"`
}

// Action implements ocpp.Request.
func (ExtendedTriggerMessageRequest) Action() string { return ExtendedTriggerMessageAction }

// Validate implements ocpp.Request.
func (r ExtendedTriggerMessageRequest) Validate() error {
	return types.First(
		types.CheckEnum("requestedMessage", r.RequestedMessage, true),
		types.CheckMinPtr("connectorId", r.ConnectorID, 1),
	)
}

// ExtendedTriggerMessageConfirmation is the ExtendedTriggerMessage.conf PDU.
type ExtendedTriggerMessageConfirmation struct {
	// Status says whether the Charge Point will send the message.
	Status types.TriggerMessageStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Response.
func (ExtendedTriggerMessageConfirmation) Action() string { return ExtendedTriggerMessageAction }

// Validate implements ocpp.Response.
func (c ExtendedTriggerMessageConfirmation) Validate() error {
	return types.CheckEnum("status", c.Status, true)
}
