package reservation_test

import (
	"errors"
	"strings"
	"testing"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/schematest"
	"github.com/foecum/go-ocpp-sdk/v16/reservation"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

var samples = map[string]schematest.Sample{
	"ReserveNow": {
		ReqMin:  `{"connectorId":0,"expiryDate":"2026-01-02T03:04:05Z","idTag":"04A1B2C3","reservationId":7}`,
		ReqFull: `{"connectorId":1,"expiryDate":"2026-01-02T03:04:05.500Z","idTag":"04A1B2C3","parentIdTag":"PARENT01","reservationId":7}`,
		ConfMin: `{"status":"Accepted"}`, ConfFull: `{"status":"Occupied"}`,
	},
	"CancelReservation": {
		ReqMin: `{"reservationId":7}`, ReqFull: `{"reservationId":7}`,
		ConfMin: `{"status":"Accepted"}`, ConfFull: `{"status":"Rejected"}`,
	},
}

func TestAgainstOfficialSchemas(t *testing.T) {
	features := reservation.Features()
	schematest.Run(t, "../../docs/spec/1.6/schemas/json", schematest.NewRegistry(t, features), samples)
	schematest.CheckActions(t, features)
	if len(features) != len(samples) {
		t.Errorf("%d features, %d samples", len(features), len(samples))
	}
}

func TestValidationRules(t *testing.T) {
	now := types.Now()
	ok := reservation.ReserveNowRequest{ConnectorID: 0, ExpiryDate: now, IDTag: "x", ReservationID: 1}
	with := func(f func(*reservation.ReserveNowRequest)) reservation.ReserveNowRequest { r := ok; f(&r); return r }
	tests := []struct {
		name  string
		v     interface{ Validate() error }
		field string
	}{
		{"ReserveNow ok", ok, ""},
		{"ReserveNow negative connector", with(func(r *reservation.ReserveNowRequest) { r.ConnectorID = -1 }), "connectorId"},
		{"ReserveNow no expiry", with(func(r *reservation.ReserveNowRequest) { r.ExpiryDate = types.DateTime{} }), "expiryDate"},
		{"ReserveNow no idTag", with(func(r *reservation.ReserveNowRequest) { r.IDTag = "" }), "idTag"},
		{"ReserveNow parentIdTag", with(func(r *reservation.ReserveNowRequest) { r.ParentIDTag = strings.Repeat("p", 21) }), "parentIdTag"},
		{"ReserveNow.conf", reservation.ReserveNowConfirmation{Status: types.ReservationStatusFaulted}, ""},
		{"ReserveNow.conf missing", reservation.ReserveNowConfirmation{}, "status"},
		{"CancelReservation", reservation.CancelReservationRequest{ReservationID: 0}, ""},
		{"CancelReservation.conf", reservation.CancelReservationConfirmation{Status: types.CancelReservationStatusAccepted}, ""},
		{"CancelReservation.conf bad", reservation.CancelReservationConfirmation{Status: "Maybe"}, "status"},
	}
	for _, tt := range tests {
		err := tt.v.Validate()
		if tt.field == "" {
			if err != nil {
				t.Errorf("%s: %v", tt.name, err)
			}
			continue
		}
		var ve *ocpp.ValidationError
		if !errors.As(err, &ve) || ve.Field != tt.field {
			t.Errorf("%s: got %v, want an error on %q", tt.name, err, tt.field)
		}
	}
}
