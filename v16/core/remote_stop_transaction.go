package core

import "github.com/foecum/go-ocpp-sdk/v16/types"

// RemoteStopTransactionAction is the action name of the RemoteStopTransaction message.
const RemoteStopTransactionAction = "RemoteStopTransaction"

// RemoteStopTransactionRequest is the RemoteStopTransaction.req PDU, sent by
// the Central System to stop a transaction.
type RemoteStopTransactionRequest struct {
	// TransactionID is the transaction to stop.
	TransactionID int `json:"transactionId" ocpp:"required"`
}

// Action implements ocpp.Request.
func (RemoteStopTransactionRequest) Action() string { return RemoteStopTransactionAction }

// Validate implements ocpp.Request.
func (RemoteStopTransactionRequest) Validate() error { return nil }

// RemoteStopTransactionConfirmation is the RemoteStopTransaction.conf PDU.
type RemoteStopTransactionConfirmation struct {
	// Status says whether the Charge Point accepts the request.
	Status types.RemoteStartStopStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Response.
func (RemoteStopTransactionConfirmation) Action() string { return RemoteStopTransactionAction }

// Validate implements ocpp.Response.
func (c RemoteStopTransactionConfirmation) Validate() error {
	return types.CheckEnum("status", c.Status, true)
}
