package security

import "github.com/foecum/go-ocpp-sdk/v16/types"

// SecurityEventNotificationAction is the action name of the
// SecurityEventNotification message.
const SecurityEventNotificationAction = "SecurityEventNotification"

// SecurityEventNotificationRequest is the SecurityEventNotification.req PDU,
// sent by the Charge Point to report a security event. Critical events (see
// IsCriticalEvent) should be pushed to the Central System.
type SecurityEventNotificationRequest struct {
	// Type is the type of the event (at most 50 characters). The Event constants
	// list the types the whitepaper defines.
	Type string `json:"type" ocpp:"required"`
	// Timestamp is when the event occurred.
	Timestamp types.DateTime `json:"timestamp" ocpp:"required"`
	// TechInfo is additional information about the event (at most 255 characters).
	TechInfo string `json:"techInfo,omitempty"`
}

// Action implements ocpp.Request.
func (SecurityEventNotificationRequest) Action() string { return SecurityEventNotificationAction }

// Validate implements ocpp.Request.
func (r SecurityEventNotificationRequest) Validate() error {
	return types.First(
		types.CheckString("type", r.Type, types.CiString50, true),
		types.CheckDateTime("timestamp", r.Timestamp),
		types.CheckString("techInfo", r.TechInfo, types.CiString255, false),
	)
}

// SecurityEventNotificationConfirmation is the SecurityEventNotification.conf
// PDU. It has no fields.
type SecurityEventNotificationConfirmation struct{}

// Action implements ocpp.Response.
func (SecurityEventNotificationConfirmation) Action() string {
	return SecurityEventNotificationAction
}

// Validate implements ocpp.Response.
func (SecurityEventNotificationConfirmation) Validate() error { return nil }
