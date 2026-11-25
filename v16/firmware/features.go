package firmware

import (
	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/featureset"
)

// Features returns the feature of every Firmware Management message, ready to
// register in an ocppj.Registry.
func Features() []ocpp.Feature {
	return featureset.Build(
		featureset.Pair{Req: GetDiagnosticsRequest{}, Resp: GetDiagnosticsConfirmation{}},
		featureset.Pair{Req: DiagnosticsStatusNotificationRequest{}, Resp: DiagnosticsStatusNotificationConfirmation{}},
		featureset.Pair{Req: FirmwareStatusNotificationRequest{}, Resp: FirmwareStatusNotificationConfirmation{}},
		featureset.Pair{Req: UpdateFirmwareRequest{}, Resp: UpdateFirmwareConfirmation{}},
	)
}
