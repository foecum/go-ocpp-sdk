package localauth

import (
	"fmt"
	"strings"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// SendLocalListAction is the action name of the SendLocalList message.
const SendLocalListAction = "SendLocalList"

// SendLocalListRequest is the SendLocalList.req PDU, sent by the Central System
// to replace or amend the Charge Point's local authorisation list.
//
// A Full update with an empty list removes every entry; a Differential update
// with an empty list changes nothing.
type SendLocalListRequest struct {
	// ListVersion is the version of the full list (Full update) or of the list
	// after the update (Differential update).
	ListVersion int `json:"listVersion" ocpp:"required"`
	// LocalAuthorizationList holds the new list (Full) or the changes
	// (Differential). Every idTag must be unique, ignoring case.
	LocalAuthorizationList []types.AuthorizationData `json:"localAuthorizationList,omitempty"`
	// UpdateType says whether this is a Full or a Differential update.
	UpdateType types.UpdateType `json:"updateType" ocpp:"required"`
}

// Action implements ocpp.Request.
func (SendLocalListRequest) Action() string { return SendLocalListAction }

// Validate implements ocpp.Request.
func (r SendLocalListRequest) Validate() error {
	if err := types.CheckEnum("updateType", r.UpdateType, true); err != nil {
		return err
	}
	seen := make(map[string]bool, len(r.LocalAuthorizationList))
	for i, entry := range r.LocalAuthorizationList {
		field := fmt.Sprintf("localAuthorizationList[%d]", i)
		if err := entry.Validate(); err != nil {
			return types.Prefix(field, err)
		}
		key := strings.ToLower(entry.IDTag) // idTags are case insensitive
		if seen[key] {
			return ocpp.Invalid(field+".idTag", "duplicate idTag in the list")
		}
		seen[key] = true
		if r.UpdateType == types.UpdateTypeFull && entry.IDTagInfo == nil {
			return ocpp.Required(field + ".idTagInfo")
		}
	}
	return nil
}

// SendLocalListConfirmation is the SendLocalList.conf PDU.
type SendLocalListConfirmation struct {
	// Status is the outcome.
	Status types.UpdateStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Response.
func (SendLocalListConfirmation) Action() string { return SendLocalListAction }

// Validate implements ocpp.Response.
func (c SendLocalListConfirmation) Validate() error { return types.CheckEnum("status", c.Status, true) }
