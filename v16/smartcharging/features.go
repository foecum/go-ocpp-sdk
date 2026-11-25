package smartcharging

import (
	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/featureset"
)

// Features returns the feature of every Smart Charging message, ready to
// register in an ocppj.Registry.
func Features() []ocpp.Feature {
	return featureset.Build(
		featureset.Pair{Req: SetChargingProfileRequest{}, Resp: SetChargingProfileConfirmation{}},
		featureset.Pair{Req: ClearChargingProfileRequest{}, Resp: ClearChargingProfileConfirmation{}},
		featureset.Pair{Req: GetCompositeScheduleRequest{}, Resp: GetCompositeScheduleConfirmation{}},
	)
}
