package core

import "github.com/foecum/go-ocpp-sdk/v16/types"

// ChangeAvailabilityAction is the action name of the ChangeAvailability message.
const ChangeAvailabilityAction = "ChangeAvailability"

// ChangeAvailabilityRequest is the ChangeAvailability.req PDU, sent by the
// Central System to make a connector or the whole Charge Point operative or
// inoperative.
type ChangeAvailabilityRequest struct {
	// ConnectorID is the connector to change; 0 means the Charge Point and all
	// its connectors.
	ConnectorID int `json:"connectorId" ocpp:"required"`
	// Type is the availability to change to.
	Type types.AvailabilityType `json:"type" ocpp:"required"`
}

// Action implements ocpp.Request.
func (ChangeAvailabilityRequest) Action() string { return ChangeAvailabilityAction }

// Validate implements ocpp.Request.
func (r ChangeAvailabilityRequest) Validate() error {
	return types.First(
		types.CheckMin("connectorId", r.ConnectorID, 0),
		types.CheckEnum("type", r.Type, true),
	)
}

// ChangeAvailabilityConfirmation is the ChangeAvailability.conf PDU.
type ChangeAvailabilityConfirmation struct {
	// Status says whether the Charge Point can perform the change.
	Status types.AvailabilityStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Response.
func (ChangeAvailabilityConfirmation) Action() string { return ChangeAvailabilityAction }

// Validate implements ocpp.Response.
func (c ChangeAvailabilityConfirmation) Validate() error {
	return types.CheckEnum("status", c.Status, true)
}
