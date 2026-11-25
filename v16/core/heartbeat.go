package core

import "github.com/foecum/go-ocpp-sdk/v16/types"

// HeartbeatAction is the action name of the Heartbeat message.
const HeartbeatAction = "Heartbeat"

// HeartbeatRequest is the Heartbeat.req PDU, sent by the Charge Point to show
// it is alive and to get the Central System's time. It has no fields.
type HeartbeatRequest struct{}

// Action implements ocpp.Request.
func (HeartbeatRequest) Action() string { return HeartbeatAction }

// Validate implements ocpp.Request.
func (HeartbeatRequest) Validate() error { return nil }

// HeartbeatConfirmation is the Heartbeat.conf PDU.
type HeartbeatConfirmation struct {
	// CurrentTime is the Central System's current time.
	CurrentTime types.DateTime `json:"currentTime" ocpp:"required"`
}

// Action implements ocpp.Response.
func (HeartbeatConfirmation) Action() string { return HeartbeatAction }

// Validate implements ocpp.Response.
func (c HeartbeatConfirmation) Validate() error {
	return types.CheckDateTime("currentTime", c.CurrentTime)
}
