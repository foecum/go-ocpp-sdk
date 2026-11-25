package reservation

import "github.com/foecum/go-ocpp-sdk/v16/types"

// ReserveNowAction is the action name of the ReserveNow message.
const ReserveNowAction = "ReserveNow"

// ReserveNowRequest is the ReserveNow.req PDU, sent by the Central System to
// reserve a connector for an identifier.
type ReserveNowRequest struct {
	// ConnectorID is the connector to reserve; 0 means any connector.
	ConnectorID int `json:"connectorId" ocpp:"required"`
	// ExpiryDate is when the reservation ends.
	ExpiryDate types.DateTime `json:"expiryDate" ocpp:"required"`
	// IDTag is the identifier the reservation is for (at most 20 characters).
	IDTag string `json:"idTag" ocpp:"required"`
	// ParentIDTag is the parent identifier (at most 20 characters).
	ParentIDTag string `json:"parentIdTag,omitempty"`
	// ReservationID is the unique id of the reservation.
	ReservationID int `json:"reservationId" ocpp:"required"`
}

// Action implements ocpp.Request.
func (ReserveNowRequest) Action() string { return ReserveNowAction }

// Validate implements ocpp.Request.
func (r ReserveNowRequest) Validate() error {
	return types.First(
		types.CheckMin("connectorId", r.ConnectorID, 0),
		types.CheckDateTime("expiryDate", r.ExpiryDate),
		types.CheckString("idTag", r.IDTag, types.CiString20, true),
		types.CheckString("parentIdTag", r.ParentIDTag, types.CiString20, false),
	)
}

// ReserveNowConfirmation is the ReserveNow.conf PDU.
type ReserveNowConfirmation struct {
	// Status is the outcome.
	Status types.ReservationStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Response.
func (ReserveNowConfirmation) Action() string { return ReserveNowAction }

// Validate implements ocpp.Response.
func (c ReserveNowConfirmation) Validate() error { return types.CheckEnum("status", c.Status, true) }
