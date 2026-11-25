package localauth

import (
	ocpp "github.com/foecum/go-ocpp-sdk"
)

// GetLocalListVersionAction is the action name of the GetLocalListVersion message.
const GetLocalListVersionAction = "GetLocalListVersion"

// GetLocalListVersionRequest is the GetLocalListVersion.req PDU, sent by the
// Central System to read the version of the Charge Point's local authorisation
// list. It has no fields.
type GetLocalListVersionRequest struct{}

// Action implements ocpp.Request.
func (GetLocalListVersionRequest) Action() string { return GetLocalListVersionAction }

// Validate implements ocpp.Request.
func (GetLocalListVersionRequest) Validate() error { return nil }

// GetLocalListVersionConfirmation is the GetLocalListVersion.conf PDU.
type GetLocalListVersionConfirmation struct {
	// ListVersion is the version of the local list: 0 means the list is empty
	// and -1 means the Charge Point does not support local lists.
	ListVersion int `json:"listVersion" ocpp:"required"`
}

// Action implements ocpp.Response.
func (GetLocalListVersionConfirmation) Action() string { return GetLocalListVersionAction }

// Validate implements ocpp.Response.
func (c GetLocalListVersionConfirmation) Validate() error {
	if c.ListVersion < -1 {
		return ocpp.Invalid("listVersion", "must be -1 or more")
	}
	return nil
}
