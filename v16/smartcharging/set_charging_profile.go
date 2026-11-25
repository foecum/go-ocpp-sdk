package smartcharging

import "github.com/foecum/go-ocpp-sdk/v16/types"

// SetChargingProfileAction is the action name of the SetChargingProfile message.
const SetChargingProfileAction = "SetChargingProfile"

// SetChargingProfileRequest is the SetChargingProfile.req PDU, sent by the
// Central System to install a charging profile.
type SetChargingProfileRequest struct {
	// ConnectorID is the connector the profile applies to; 0 means an overall
	// limit for the Charge Point.
	ConnectorID int `json:"connectorId" ocpp:"required"`
	// CSChargingProfiles is the profile to install.
	CSChargingProfiles types.ChargingProfile `json:"csChargingProfiles" ocpp:"required"`
}

// Action implements ocpp.Request.
func (SetChargingProfileRequest) Action() string { return SetChargingProfileAction }

// Validate implements ocpp.Request.
func (r SetChargingProfileRequest) Validate() error {
	if err := types.CheckMin("connectorId", r.ConnectorID, 0); err != nil {
		return err
	}
	return types.Prefix("csChargingProfiles", r.CSChargingProfiles.Validate())
}

// SetChargingProfileConfirmation is the SetChargingProfile.conf PDU.
type SetChargingProfileConfirmation struct {
	// Status is the outcome.
	Status types.ChargingProfileStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Response.
func (SetChargingProfileConfirmation) Action() string { return SetChargingProfileAction }

// Validate implements ocpp.Response.
func (c SetChargingProfileConfirmation) Validate() error {
	return types.CheckEnum("status", c.Status, true)
}
