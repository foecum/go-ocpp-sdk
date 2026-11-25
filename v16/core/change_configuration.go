package core

import "github.com/foecum/go-ocpp-sdk/v16/types"

// ChangeConfigurationAction is the action name of the ChangeConfiguration message.
const ChangeConfigurationAction = "ChangeConfiguration"

// ChangeConfigurationRequest is the ChangeConfiguration.req PDU, sent by the
// Central System to change a configuration key.
type ChangeConfigurationRequest struct {
	// Key is the name of the setting (at most 50 characters).
	Key string `json:"key" ocpp:"required"`
	// Value is the new value as a string (at most 500 characters). It may be
	// empty, but the field must be present.
	Value string `json:"value" ocpp:"required"`
}

// Action implements ocpp.Request.
func (ChangeConfigurationRequest) Action() string { return ChangeConfigurationAction }

// Validate implements ocpp.Request.
func (r ChangeConfigurationRequest) Validate() error {
	return types.First(
		types.CheckString("key", r.Key, types.CiString50, true),
		types.CheckString("value", r.Value, types.CiString500, false),
	)
}

// ChangeConfigurationConfirmation is the ChangeConfiguration.conf PDU.
type ChangeConfigurationConfirmation struct {
	// Status is the outcome.
	Status types.ConfigurationStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Response.
func (ChangeConfigurationConfirmation) Action() string { return ChangeConfigurationAction }

// Validate implements ocpp.Response.
func (c ChangeConfigurationConfirmation) Validate() error {
	return types.CheckEnum("status", c.Status, true)
}
