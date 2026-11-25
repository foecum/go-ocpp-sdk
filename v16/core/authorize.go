package core

import "github.com/foecum/go-ocpp-sdk/v16/types"

// AuthorizeAction is the action name of the Authorize message.
const AuthorizeAction = "Authorize"

// AuthorizeRequest is the Authorize.req PDU, sent by the Charge Point to ask
// whether an identifier may start charging.
type AuthorizeRequest struct {
	// IDTag is the identifier to authorise (at most 20 characters).
	IDTag string `json:"idTag" ocpp:"required"`
}

// Action implements ocpp.Request.
func (AuthorizeRequest) Action() string { return AuthorizeAction }

// Validate implements ocpp.Request.
func (r AuthorizeRequest) Validate() error {
	return types.CheckString("idTag", r.IDTag, types.CiString20, true)
}

// AuthorizeConfirmation is the Authorize.conf PDU.
type AuthorizeConfirmation struct {
	// IDTagInfo says whether the identifier is authorised.
	IDTagInfo types.IDTagInfo `json:"idTagInfo" ocpp:"required"`
}

// Action implements ocpp.Response.
func (AuthorizeConfirmation) Action() string { return AuthorizeAction }

// Validate implements ocpp.Response.
func (c AuthorizeConfirmation) Validate() error {
	return types.Prefix("idTagInfo", c.IDTagInfo.Validate())
}
