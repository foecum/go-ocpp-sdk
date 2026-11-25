package reservation

import (
	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/featureset"
)

// Features returns the feature of every Reservation message, ready to register
// in an ocppj.Registry.
func Features() []ocpp.Feature {
	return featureset.Build(
		featureset.Pair{Req: ReserveNowRequest{}, Resp: ReserveNowConfirmation{}},
		featureset.Pair{Req: CancelReservationRequest{}, Resp: CancelReservationConfirmation{}},
	)
}
