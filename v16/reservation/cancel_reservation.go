package reservation

import "github.com/foecum/go-ocpp-sdk/v16/types"

// CancelReservationAction is the action name of the CancelReservation message.
const CancelReservationAction = "CancelReservation"

// CancelReservationRequest is the CancelReservation.req PDU, sent by the
// Central System to cancel a reservation.
type CancelReservationRequest struct {
	// ReservationID is the id of the reservation to cancel.
	ReservationID int `json:"reservationId" ocpp:"required"`
}

// Action implements ocpp.Request.
func (CancelReservationRequest) Action() string { return CancelReservationAction }

// Validate implements ocpp.Request.
func (CancelReservationRequest) Validate() error { return nil }

// CancelReservationConfirmation is the CancelReservation.conf PDU.
type CancelReservationConfirmation struct {
	// Status is the outcome.
	Status types.CancelReservationStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Response.
func (CancelReservationConfirmation) Action() string { return CancelReservationAction }

// Validate implements ocpp.Response.
func (c CancelReservationConfirmation) Validate() error {
	return types.CheckEnum("status", c.Status, true)
}
