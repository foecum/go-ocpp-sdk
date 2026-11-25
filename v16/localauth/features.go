package localauth

import (
	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/featureset"
)

// Features returns the feature of every Local Auth List Management message,
// ready to register in an ocppj.Registry.
func Features() []ocpp.Feature {
	return featureset.Build(
		featureset.Pair{Req: GetLocalListVersionRequest{}, Resp: GetLocalListVersionConfirmation{}},
		featureset.Pair{Req: SendLocalListRequest{}, Resp: SendLocalListConfirmation{}},
	)
}
