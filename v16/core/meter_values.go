package core

import (
	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// MeterValuesAction is the action name of the MeterValues message.
const MeterValuesAction = "MeterValues"

// MeterValuesRequest is the MeterValues.req PDU, sent by the Charge Point to
// report meter readings.
type MeterValuesRequest struct {
	// ConnectorID is the connector the values belong to; 0 is the main meter.
	ConnectorID int `json:"connectorId" ocpp:"required"`
	// TransactionID is the transaction the samples relate to, if any.
	TransactionID *int `json:"transactionId,omitempty"`
	// MeterValue holds one or more sets of sampled values.
	MeterValue []types.MeterValue `json:"meterValue" ocpp:"required"`
}

// Action implements ocpp.Request.
func (MeterValuesRequest) Action() string { return MeterValuesAction }

// Validate implements ocpp.Request.
func (r MeterValuesRequest) Validate() error {
	if err := types.CheckMin("connectorId", r.ConnectorID, 0); err != nil {
		return err
	}
	if len(r.MeterValue) == 0 {
		return ocpp.Required("meterValue")
	}
	return types.ValidateMeterValues("meterValue", r.MeterValue)
}

// MeterValuesConfirmation is the MeterValues.conf PDU. It has no fields.
type MeterValuesConfirmation struct{}

// Action implements ocpp.Response.
func (MeterValuesConfirmation) Action() string { return MeterValuesAction }

// Validate implements ocpp.Response.
func (MeterValuesConfirmation) Validate() error { return nil }
