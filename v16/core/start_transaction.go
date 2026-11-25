package core

import "github.com/foecum/go-ocpp-sdk/v16/types"

// StartTransactionAction is the action name of the StartTransaction message.
const StartTransactionAction = "StartTransaction"

// StartTransactionRequest is the StartTransaction.req PDU, sent by the Charge
// Point when a transaction starts.
type StartTransactionRequest struct {
	// ConnectorID is the connector used; it must be greater than 0.
	ConnectorID int `json:"connectorId" ocpp:"required"`
	// IDTag is the identifier the transaction is started for (at most 20 characters).
	IDTag string `json:"idTag" ocpp:"required"`
	// MeterStart is the meter reading in Wh at the start.
	MeterStart int `json:"meterStart" ocpp:"required"`
	// ReservationID is the reservation that ends because of this transaction.
	ReservationID *int `json:"reservationId,omitempty"`
	// Timestamp is when the transaction started.
	Timestamp types.DateTime `json:"timestamp" ocpp:"required"`
}

// Action implements ocpp.Request.
func (StartTransactionRequest) Action() string { return StartTransactionAction }

// Validate implements ocpp.Request.
func (r StartTransactionRequest) Validate() error {
	return types.First(
		types.CheckMin("connectorId", r.ConnectorID, 1),
		types.CheckString("idTag", r.IDTag, types.CiString20, true),
		types.CheckDateTime("timestamp", r.Timestamp),
	)
}

// StartTransactionConfirmation is the StartTransaction.conf PDU.
type StartTransactionConfirmation struct {
	// IDTagInfo says whether the identifier is authorised.
	IDTagInfo types.IDTagInfo `json:"idTagInfo" ocpp:"required"`
	// TransactionID is the transaction id assigned by the Central System.
	TransactionID int `json:"transactionId" ocpp:"required"`
}

// Action implements ocpp.Response.
func (StartTransactionConfirmation) Action() string { return StartTransactionAction }

// Validate implements ocpp.Response.
func (c StartTransactionConfirmation) Validate() error {
	return types.Prefix("idTagInfo", c.IDTagInfo.Validate())
}
