package core

import "github.com/foecum/go-ocpp-sdk/v16/types"

// StopTransactionAction is the action name of the StopTransaction message.
const StopTransactionAction = "StopTransaction"

// StopTransactionRequest is the StopTransaction.req PDU, sent by the Charge
// Point when a transaction ends.
type StopTransactionRequest struct {
	// IDTag is the identifier that asked to stop charging, if known (at most 20
	// characters).
	IDTag string `json:"idTag,omitempty"`
	// MeterStop is the meter reading in Wh at the end.
	MeterStop int `json:"meterStop" ocpp:"required"`
	// Timestamp is when the transaction stopped.
	Timestamp types.DateTime `json:"timestamp" ocpp:"required"`
	// TransactionID is the id received in StartTransaction.conf.
	TransactionID int `json:"transactionId" ocpp:"required"`
	// Reason is why the transaction stopped. It may only be omitted for "Local".
	Reason types.Reason `json:"reason,omitempty"`
	// TransactionData holds usage details relevant for billing.
	TransactionData []types.MeterValue `json:"transactionData,omitempty"`
}

// Action implements ocpp.Request.
func (StopTransactionRequest) Action() string { return StopTransactionAction }

// Validate implements ocpp.Request.
func (r StopTransactionRequest) Validate() error {
	return types.First(
		types.CheckString("idTag", r.IDTag, types.CiString20, false),
		types.CheckDateTime("timestamp", r.Timestamp),
		types.CheckEnum("reason", r.Reason, false),
		types.ValidateMeterValues("transactionData", r.TransactionData),
	)
}

// StopTransactionConfirmation is the StopTransaction.conf PDU.
type StopTransactionConfirmation struct {
	// IDTagInfo says whether the identifier is authorised. It is absent if the
	// transaction stopped without an identifier.
	IDTagInfo *types.IDTagInfo `json:"idTagInfo,omitempty"`
}

// Action implements ocpp.Response.
func (StopTransactionConfirmation) Action() string { return StopTransactionAction }

// Validate implements ocpp.Response.
func (c StopTransactionConfirmation) Validate() error {
	if c.IDTagInfo == nil {
		return nil
	}
	return types.Prefix("idTagInfo", c.IDTagInfo.Validate())
}
