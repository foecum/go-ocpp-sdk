package smartcharging_test

import (
	"fmt"

	"github.com/foecum/go-ocpp-sdk/v16/smartcharging"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// The first charging period of a schedule must start at 0.
func ExampleSetChargingProfileRequest_Validate() {
	req := smartcharging.SetChargingProfileRequest{
		ConnectorID: 1,
		CSChargingProfiles: types.ChargingProfile{
			ChargingProfileID:      1,
			ChargingProfilePurpose: types.ChargingProfilePurposeTypeTxProfile,
			ChargingProfileKind:    types.ChargingProfileKindTypeRelative,
			ChargingSchedule: types.ChargingSchedule{
				ChargingRateUnit:       types.ChargingRateUnitTypeA,
				ChargingSchedulePeriod: []types.ChargingSchedulePeriod{{StartPeriod: 60, Limit: 16}},
			},
		},
	}
	fmt.Println(req.Validate())
	// Output: ocpp: invalid field "csChargingProfiles.chargingSchedule.chargingSchedulePeriod[0].startPeriod": the first period must start at 0
}
