package smartcharging_test

import (
	"errors"
	"testing"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/schematest"
	"github.com/foecum/go-ocpp-sdk/v16/smartcharging"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

const schedule = `{"duration":3600,"startSchedule":"2026-01-02T03:04:05Z","chargingRateUnit":"A",` +
	`"chargingSchedulePeriod":[{"startPeriod":0,"limit":16.5,"numberPhases":3}],"minChargingRate":6.5}`

var samples = map[string]schematest.Sample{
	"SetChargingProfile": {
		ReqMin: `{"connectorId":1,"csChargingProfiles":{"chargingProfileId":1,"stackLevel":0,"chargingProfilePurpose":"TxProfile",` +
			`"chargingProfileKind":"Relative","chargingSchedule":{"chargingRateUnit":"A","chargingSchedulePeriod":[{"startPeriod":0,"limit":16}]}}}`,
		ReqFull: `{"connectorId":0,"csChargingProfiles":{"chargingProfileId":2,"transactionId":9,"stackLevel":3,` +
			`"chargingProfilePurpose":"ChargePointMaxProfile","chargingProfileKind":"Recurring","recurrencyKind":"Weekly",` +
			`"validFrom":"2026-01-01T00:00:00Z","validTo":"2026-12-31T23:59:59Z","chargingSchedule":` + schedule + `}}`,
		ConfMin: `{"status":"Accepted"}`, ConfFull: `{"status":"NotSupported"}`,
	},
	"ClearChargingProfile": {
		ReqMin: `{}`, ReqFull: `{"id":1,"connectorId":0,"chargingProfilePurpose":"TxDefaultProfile","stackLevel":2}`,
		ConfMin: `{"status":"Accepted"}`, ConfFull: `{"status":"Unknown"}`,
	},
	"GetCompositeSchedule": {
		ReqMin: `{"connectorId":1,"duration":3600}`, ReqFull: `{"connectorId":0,"duration":86400,"chargingRateUnit":"W"}`,
		ConfMin:  `{"status":"Rejected"}`,
		ConfFull: `{"status":"Accepted","connectorId":1,"scheduleStart":"2026-01-02T03:04:05Z","chargingSchedule":` + schedule + `}`,
	},
}

func TestAgainstOfficialSchemas(t *testing.T) {
	features := smartcharging.Features()
	schematest.Run(t, "../../docs/spec/1.6/schemas/json", schematest.NewRegistry(t, features), samples)
	schematest.CheckActions(t, features)
	if len(features) != len(samples) {
		t.Errorf("%d features, %d samples", len(features), len(samples))
	}
}

func intp(i int) *int { return &i }

func profile() types.ChargingProfile {
	return types.ChargingProfile{
		ChargingProfileID: 1, ChargingProfilePurpose: types.ChargingProfilePurposeTypeTxProfile,
		ChargingProfileKind: types.ChargingProfileKindTypeAbsolute,
		ChargingSchedule: types.ChargingSchedule{ChargingRateUnit: types.ChargingRateUnitTypeA,
			ChargingSchedulePeriod: []types.ChargingSchedulePeriod{{Limit: 16}}},
	}
}

func goodSchedule() *types.ChargingSchedule {
	p := profile()
	return &p.ChargingSchedule
}

func TestValidationRules(t *testing.T) {
	badProfile := profile()
	badProfile.StackLevel = -1
	badSchedule := &types.ChargingSchedule{}
	tests := []struct {
		name  string
		v     interface{ Validate() error }
		field string
	}{
		{"SetChargingProfile ok", smartcharging.SetChargingProfileRequest{ConnectorID: 0, CSChargingProfiles: profile()}, ""},
		{"SetChargingProfile negative connector", smartcharging.SetChargingProfileRequest{ConnectorID: -1, CSChargingProfiles: profile()}, "connectorId"},
		{"SetChargingProfile bad profile", smartcharging.SetChargingProfileRequest{CSChargingProfiles: badProfile}, "csChargingProfiles.stackLevel"},
		{"SetChargingProfile.conf", smartcharging.SetChargingProfileConfirmation{Status: types.ChargingProfileStatusAccepted}, ""},
		{"SetChargingProfile.conf missing", smartcharging.SetChargingProfileConfirmation{}, "status"},

		{"ClearChargingProfile empty clears all", smartcharging.ClearChargingProfileRequest{}, ""},
		{"ClearChargingProfile full", smartcharging.ClearChargingProfileRequest{ID: intp(1), ConnectorID: intp(0), StackLevel: intp(0),
			ChargingProfilePurpose: types.ChargingProfilePurposeTypeTxProfile}, ""},
		{"ClearChargingProfile connector", smartcharging.ClearChargingProfileRequest{ConnectorID: intp(-1)}, "connectorId"},
		{"ClearChargingProfile purpose", smartcharging.ClearChargingProfileRequest{ChargingProfilePurpose: "Other"}, "chargingProfilePurpose"},
		{"ClearChargingProfile stack level", smartcharging.ClearChargingProfileRequest{StackLevel: intp(-1)}, "stackLevel"},
		{"ClearChargingProfile.conf", smartcharging.ClearChargingProfileConfirmation{Status: types.ClearChargingProfileStatusUnknown}, ""},
		{"ClearChargingProfile.conf missing", smartcharging.ClearChargingProfileConfirmation{}, "status"},

		{"GetCompositeSchedule ok", smartcharging.GetCompositeScheduleRequest{ConnectorID: 0, Duration: 60, ChargingRateUnit: types.ChargingRateUnitTypeW}, ""},
		{"GetCompositeSchedule connector", smartcharging.GetCompositeScheduleRequest{ConnectorID: -1}, "connectorId"},
		{"GetCompositeSchedule duration", smartcharging.GetCompositeScheduleRequest{Duration: -1}, "duration"},
		{"GetCompositeSchedule unit", smartcharging.GetCompositeScheduleRequest{ChargingRateUnit: "kW"}, "chargingRateUnit"},
		{"GetCompositeSchedule.conf minimal", smartcharging.GetCompositeScheduleConfirmation{Status: types.GetCompositeScheduleStatusRejected}, ""},
		{"GetCompositeSchedule.conf full", smartcharging.GetCompositeScheduleConfirmation{Status: types.GetCompositeScheduleStatusAccepted,
			ConnectorID: intp(1), ScheduleStart: types.Now().Ptr(), ChargingSchedule: goodSchedule()}, ""},
		{"GetCompositeSchedule.conf status", smartcharging.GetCompositeScheduleConfirmation{}, "status"},
		{"GetCompositeSchedule.conf connector", smartcharging.GetCompositeScheduleConfirmation{Status: types.GetCompositeScheduleStatusAccepted, ConnectorID: intp(-1)}, "connectorId"},
		{"GetCompositeSchedule.conf schedule", smartcharging.GetCompositeScheduleConfirmation{Status: types.GetCompositeScheduleStatusAccepted, ChargingSchedule: badSchedule}, "chargingSchedule.chargingRateUnit"},
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
