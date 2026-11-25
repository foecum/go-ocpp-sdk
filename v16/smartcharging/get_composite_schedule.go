package smartcharging

import "github.com/foecum/go-ocpp-sdk/v16/types"

// GetCompositeScheduleAction is the action name of the GetCompositeSchedule message.
const GetCompositeScheduleAction = "GetCompositeSchedule"

// GetCompositeScheduleRequest is the GetCompositeSchedule.req PDU, sent by the
// Central System to ask for the schedule the Charge Point will follow.
type GetCompositeScheduleRequest struct {
	// ConnectorID is the connector; 0 asks for the expected consumption of the
	// grid connection.
	ConnectorID int `json:"connectorId" ocpp:"required"`
	// Duration is the length of the requested schedule in seconds.
	Duration int `json:"duration" ocpp:"required"`
	// ChargingRateUnit can force a power (W) or current (A) schedule.
	ChargingRateUnit types.ChargingRateUnitType `json:"chargingRateUnit,omitempty"`
}

// Action implements ocpp.Request.
func (GetCompositeScheduleRequest) Action() string { return GetCompositeScheduleAction }

// Validate implements ocpp.Request.
func (r GetCompositeScheduleRequest) Validate() error {
	return types.First(
		types.CheckMin("connectorId", r.ConnectorID, 0),
		types.CheckMin("duration", r.Duration, 0),
		types.CheckEnum("chargingRateUnit", r.ChargingRateUnit, false),
	)
}

// GetCompositeScheduleConfirmation is the GetCompositeSchedule.conf PDU.
type GetCompositeScheduleConfirmation struct {
	// Status says whether the Charge Point could process the request.
	Status types.GetCompositeScheduleStatus `json:"status" ocpp:"required"`
	// ConnectorID is the connector the schedule applies to.
	ConnectorID *int `json:"connectorId,omitempty"`
	// ScheduleStart is the start of the schedule.
	ScheduleStart *types.DateTime `json:"scheduleStart,omitempty"`
	// ChargingSchedule is the composite schedule.
	ChargingSchedule *types.ChargingSchedule `json:"chargingSchedule,omitempty"`
}

// Action implements ocpp.Response.
func (GetCompositeScheduleConfirmation) Action() string { return GetCompositeScheduleAction }

// Validate implements ocpp.Response.
func (c GetCompositeScheduleConfirmation) Validate() error {
	if err := types.First(
		types.CheckEnum("status", c.Status, true),
		types.CheckMinPtr("connectorId", c.ConnectorID, 0),
	); err != nil {
		return err
	}
	if c.ChargingSchedule != nil {
		return types.Prefix("chargingSchedule", c.ChargingSchedule.Validate())
	}
	return nil
}
