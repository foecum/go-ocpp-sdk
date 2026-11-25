package core

import (
	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// RemoteStartTransactionAction is the action name of the RemoteStartTransaction message.
const RemoteStartTransactionAction = "RemoteStartTransaction"

// RemoteStartTransactionRequest is the RemoteStartTransaction.req PDU, sent by
// the Central System to start a transaction on a Charge Point.
type RemoteStartTransactionRequest struct {
	// ConnectorID is the connector to start on. If present it must be greater than 0.
	ConnectorID *int `json:"connectorId,omitempty"`
	// IDTag is the identifier the Charge Point must use to start the transaction
	// (at most 20 characters).
	IDTag string `json:"idTag" ocpp:"required"`
	// ChargingProfile is a profile for this transaction. Its purpose must be TxProfile.
	ChargingProfile *types.ChargingProfile `json:"chargingProfile,omitempty"`
}

// Action implements ocpp.Request.
func (RemoteStartTransactionRequest) Action() string { return RemoteStartTransactionAction }

// Validate implements ocpp.Request.
func (r RemoteStartTransactionRequest) Validate() error {
	if err := types.First(
		types.CheckMinPtr("connectorId", r.ConnectorID, 1),
		types.CheckString("idTag", r.IDTag, types.CiString20, true),
	); err != nil {
		return err
	}
	if p := r.ChargingProfile; p != nil {
		if err := p.Validate(); err != nil {
			return types.Prefix("chargingProfile", err)
		}
		if p.ChargingProfilePurpose != types.ChargingProfilePurposeTypeTxProfile {
			return ocpp.Invalid("chargingProfile.chargingProfilePurpose", "must be TxProfile")
		}
	}
	return nil
}

// RemoteStartTransactionConfirmation is the RemoteStartTransaction.conf PDU.
type RemoteStartTransactionConfirmation struct {
	// Status says whether the Charge Point accepts the request.
	Status types.RemoteStartStopStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Response.
func (RemoteStartTransactionConfirmation) Action() string { return RemoteStartTransactionAction }

// Validate implements ocpp.Response.
func (c RemoteStartTransactionConfirmation) Validate() error {
	return types.CheckEnum("status", c.Status, true)
}
