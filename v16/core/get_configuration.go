package core

import (
	"fmt"

	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// GetConfigurationAction is the action name of the GetConfiguration message.
const GetConfigurationAction = "GetConfiguration"

// GetConfigurationRequest is the GetConfiguration.req PDU, sent by the Central
// System to read configuration keys.
type GetConfigurationRequest struct {
	// Key lists the keys to read (each at most 50 characters). If empty, all
	// keys are returned.
	Key []string `json:"key,omitempty"`
}

// Action implements ocpp.Request.
func (GetConfigurationRequest) Action() string { return GetConfigurationAction }

// Validate implements ocpp.Request.
func (r GetConfigurationRequest) Validate() error { return checkKeys("key", r.Key) }

func checkKeys(field string, keys []string) error {
	for i, k := range keys {
		if err := types.CheckString(fmt.Sprintf("%s[%d]", field, i), k, types.CiString50, true); err != nil {
			return err
		}
	}
	return nil
}

// GetConfigurationConfirmation is the GetConfiguration.conf PDU.
type GetConfigurationConfirmation struct {
	// ConfigurationKey lists the requested or known keys.
	ConfigurationKey []types.KeyValue `json:"configurationKey,omitempty"`
	// UnknownKey lists requested keys the Charge Point does not know.
	UnknownKey []string `json:"unknownKey,omitempty"`
}

// Action implements ocpp.Response.
func (GetConfigurationConfirmation) Action() string { return GetConfigurationAction }

// Validate implements ocpp.Response.
func (c GetConfigurationConfirmation) Validate() error {
	for i, kv := range c.ConfigurationKey {
		if err := kv.Validate(); err != nil {
			return types.Prefix(fmt.Sprintf("configurationKey[%d]", i), err)
		}
	}
	return checkKeys("unknownKey", c.UnknownKey)
}
