package smartcharging

import "github.com/foecum/go-ocpp-sdk/v16/types"

// ClearChargingProfileAction is the action name of the ClearChargingProfile message.
const ClearChargingProfileAction = "ClearChargingProfile"

// ClearChargingProfileRequest is the ClearChargingProfile.req PDU, sent by the
// Central System to remove charging profiles. Every field is optional: the
// request clears the profile with the given ID, or every profile that matches
// all the other fields given. With no fields at all, every profile is cleared.
type ClearChargingProfileRequest struct {
	// ID is the id of the profile to clear.
	ID *int `json:"id,omitempty"`
	// ConnectorID restricts the request to one connector; 0 is the overall
	// Charge Point profile. Absent means all connectors.
	ConnectorID *int `json:"connectorId,omitempty"`
	// ChargingProfilePurpose restricts the request to one purpose.
	ChargingProfilePurpose types.ChargingProfilePurposeType `json:"chargingProfilePurpose,omitempty"`
	// StackLevel restricts the request to one stack level.
	StackLevel *int `json:"stackLevel,omitempty"`
}

// Action implements ocpp.Request.
func (ClearChargingProfileRequest) Action() string { return ClearChargingProfileAction }

// Validate implements ocpp.Request.
func (r ClearChargingProfileRequest) Validate() error {
	return types.First(
		types.CheckMinPtr("connectorId", r.ConnectorID, 0),
		types.CheckEnum("chargingProfilePurpose", r.ChargingProfilePurpose, false),
		types.CheckMinPtr("stackLevel", r.StackLevel, 0),
	)
}

// ClearChargingProfileConfirmation is the ClearChargingProfile.conf PDU.
type ClearChargingProfileConfirmation struct {
	// Status is Accepted if a profile was cleared, otherwise Unknown.
	Status types.ClearChargingProfileStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Response.
func (ClearChargingProfileConfirmation) Action() string { return ClearChargingProfileAction }

// Validate implements ocpp.Response.
func (c ClearChargingProfileConfirmation) Validate() error {
	return types.CheckEnum("status", c.Status, true)
}
