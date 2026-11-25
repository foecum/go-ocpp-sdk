package core

import (
	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/featureset"
)

// Features returns the feature (action plus request and response types) of
// every Core profile message, ready to register in an ocppj.Registry.
func Features() []ocpp.Feature {
	return featureset.Build(
		featureset.Pair{Req: AuthorizeRequest{}, Resp: AuthorizeConfirmation{}},
		featureset.Pair{Req: BootNotificationRequest{}, Resp: BootNotificationConfirmation{}},
		featureset.Pair{Req: ChangeAvailabilityRequest{}, Resp: ChangeAvailabilityConfirmation{}},
		featureset.Pair{Req: ChangeConfigurationRequest{}, Resp: ChangeConfigurationConfirmation{}},
		featureset.Pair{Req: ClearCacheRequest{}, Resp: ClearCacheConfirmation{}},
		featureset.Pair{Req: DataTransferRequest{}, Resp: DataTransferConfirmation{}},
		featureset.Pair{Req: GetConfigurationRequest{}, Resp: GetConfigurationConfirmation{}},
		featureset.Pair{Req: HeartbeatRequest{}, Resp: HeartbeatConfirmation{}},
		featureset.Pair{Req: MeterValuesRequest{}, Resp: MeterValuesConfirmation{}},
		featureset.Pair{Req: RemoteStartTransactionRequest{}, Resp: RemoteStartTransactionConfirmation{}},
		featureset.Pair{Req: RemoteStopTransactionRequest{}, Resp: RemoteStopTransactionConfirmation{}},
		featureset.Pair{Req: ResetRequest{}, Resp: ResetConfirmation{}},
		featureset.Pair{Req: StartTransactionRequest{}, Resp: StartTransactionConfirmation{}},
		featureset.Pair{Req: StatusNotificationRequest{}, Resp: StatusNotificationConfirmation{}},
		featureset.Pair{Req: StopTransactionRequest{}, Resp: StopTransactionConfirmation{}},
		featureset.Pair{Req: UnlockConnectorRequest{}, Resp: UnlockConnectorConfirmation{}},
	)
}
