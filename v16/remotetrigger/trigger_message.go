package remotetrigger

import (
	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/featureset"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// TriggerMessageAction is the action name of the TriggerMessage message.
const TriggerMessageAction = "TriggerMessage"

// TriggerMessageRequest is the TriggerMessage.req PDU, sent by the Central
// System to ask the Charge Point to send a message.
type TriggerMessageRequest struct {
	// RequestedMessage is the message to send.
	RequestedMessage types.MessageTrigger `json:"requestedMessage" ocpp:"required"`
	// ConnectorID is the connector the request applies to, if any. When present
	// it must be greater than 0.
	ConnectorID *int `json:"connectorId,omitempty"`
}

// Action implements ocpp.Request.
func (TriggerMessageRequest) Action() string { return TriggerMessageAction }

// Validate implements ocpp.Request.
func (r TriggerMessageRequest) Validate() error {
	return types.First(
		types.CheckEnum("requestedMessage", r.RequestedMessage, true),
		types.CheckMinPtr("connectorId", r.ConnectorID, 1),
	)
}

// TriggerMessageConfirmation is the TriggerMessage.conf PDU.
type TriggerMessageConfirmation struct {
	// Status says whether the Charge Point will send the message.
	Status types.TriggerMessageStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Response.
func (TriggerMessageConfirmation) Action() string { return TriggerMessageAction }

// Validate implements ocpp.Response.
func (c TriggerMessageConfirmation) Validate() error {
	return types.CheckEnum("status", c.Status, true)
}

// Features returns the feature of the TriggerMessage message, ready to register
// in an ocppj.Registry.
func Features() []ocpp.Feature {
	return featureset.Build(featureset.Pair{Req: TriggerMessageRequest{}, Resp: TriggerMessageConfirmation{}})
}
