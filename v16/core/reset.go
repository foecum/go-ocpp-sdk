package core

import "github.com/foecum/go-ocpp-sdk/v16/types"

// ResetAction is the action name of the Reset message.
const ResetAction = "Reset"

// ResetRequest is the Reset.req PDU, sent by the Central System to reset the
// Charge Point.
type ResetRequest struct {
	// Type is Hard or Soft.
	Type types.ResetType `json:"type" ocpp:"required"`
}

// Action implements ocpp.Request.
func (ResetRequest) Action() string { return ResetAction }

// Validate implements ocpp.Request.
func (r ResetRequest) Validate() error { return types.CheckEnum("type", r.Type, true) }

// ResetConfirmation is the Reset.conf PDU.
type ResetConfirmation struct {
	// Status says whether the Charge Point will reset.
	Status types.ResetStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Response.
func (ResetConfirmation) Action() string { return ResetAction }

// Validate implements ocpp.Response.
func (c ResetConfirmation) Validate() error { return types.CheckEnum("status", c.Status, true) }
