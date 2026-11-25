package core

import "github.com/foecum/go-ocpp-sdk/v16/types"

// UnlockConnectorAction is the action name of the UnlockConnector message.
const UnlockConnectorAction = "UnlockConnector"

// UnlockConnectorRequest is the UnlockConnector.req PDU, sent by the Central
// System to unlock a connector.
type UnlockConnectorRequest struct {
	// ConnectorID is the connector to unlock; it must be greater than 0.
	ConnectorID int `json:"connectorId" ocpp:"required"`
}

// Action implements ocpp.Request.
func (UnlockConnectorRequest) Action() string { return UnlockConnectorAction }

// Validate implements ocpp.Request.
func (r UnlockConnectorRequest) Validate() error {
	return types.CheckMin("connectorId", r.ConnectorID, 1)
}

// UnlockConnectorConfirmation is the UnlockConnector.conf PDU.
type UnlockConnectorConfirmation struct {
	// Status is the outcome.
	Status types.UnlockStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Response.
func (UnlockConnectorConfirmation) Action() string { return UnlockConnectorAction }

// Validate implements ocpp.Response.
func (c UnlockConnectorConfirmation) Validate() error {
	return types.CheckEnum("status", c.Status, true)
}
