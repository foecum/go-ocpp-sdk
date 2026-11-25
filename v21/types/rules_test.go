package types_test

import (
	"strings"
	"testing"

	"github.com/foecum/go-ocpp-sdk/v21/types"
)

func ptr[T any](v T) *T { return &v }

// Constraints of OCPP 2.1 part 2 and its errata that the JSON schemas lack.
func TestConstraintsOfTheSpecificationText(t *testing.T) {
	tests := []struct {
		name string
		v    interface{ Validate() error }
		want string // empty: valid
	}{
		// Errata 2026-09, 2.28: pctMaxDischargePower is required, a percentage.
		{"max discharge without a percentage", types.LimitMaxDischarge{Priority: 1}, "pctMaxDischargePower"},
		{"max discharge at 0", types.LimitMaxDischarge{Priority: 1, PctMaxDischargePower: ptr(0.0)}, ""},
		{"max discharge at 100", types.LimitMaxDischarge{Priority: 1, PctMaxDischargePower: ptr(100.0)}, ""},
		{"max discharge above 100", types.LimitMaxDischarge{Priority: 1, PctMaxDischargePower: ptr(100.5)}, "pctMaxDischargePower"},
		{"max discharge below 0", types.LimitMaxDischarge{Priority: 1, PctMaxDischargePower: ptr(-1.0)}, "pctMaxDischargePower"},

		{"10 minute mean without delay", types.VoltageParams{Hv10MinMeanValue: ptr(253.0)}, "hv10MinMeanTripDelay"},
		{"10 minute delay without mean", types.VoltageParams{Hv10MinMeanTripDelay: ptr(5.0)}, "hv10MinMeanValue"},
		{"10 minute mean with delay", types.VoltageParams{Hv10MinMeanValue: ptr(253.0), Hv10MinMeanTripDelay: ptr(5.0)}, ""},
		{"neither 10 minute value", types.VoltageParams{}, ""},

		{"time of day 18:00", types.TariffConditions{StartTimeOfDay: "08:00", EndTimeOfDay: "18:00"}, ""},
		{"time of day 24:00", types.TariffConditions{EndTimeOfDay: "24:00"}, "endTimeOfDay"},
		{"time of day without leading zero", types.TariffConditions{StartTimeOfDay: "8:00"}, "startTimeOfDay"},
		{"dates", types.TariffConditions{ValidFromDate: "2026-01-02", ValidToDate: "2026-12-31"}, ""},
		{"date in the wrong order", types.TariffConditions{ValidFromDate: "02-01-2026"}, "validFromDate"},
		{"fixed time of day", types.TariffConditionsFixed{StartTimeOfDay: "08:00", ValidToDate: "2026-12-31"}, ""},
		{"fixed bad time of day", types.TariffConditionsFixed{StartTimeOfDay: "25:00"}, "startTimeOfDay"},
		{"fixed bad date", types.TariffConditionsFixed{ValidToDate: "2026-13-01"}, "validToDate"},
	}
	for _, tt := range tests {
		err := tt.v.Validate()
		switch {
		case tt.want == "" && err != nil:
			t.Errorf("%s: unexpectedly rejected: %v", tt.name, err)
		case tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)):
			t.Errorf("%s: want an error containing %q, got %v", tt.name, tt.want, err)
		}
	}
}
