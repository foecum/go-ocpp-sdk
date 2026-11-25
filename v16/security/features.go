package security

import (
	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/featureset"
)

// Features returns the feature of every security extension message, ready to
// register in an ocppj.Registry.
func Features() []ocpp.Feature {
	return featureset.Build(
		featureset.Pair{Req: CertificateSignedRequest{}, Resp: CertificateSignedConfirmation{}},
		featureset.Pair{Req: DeleteCertificateRequest{}, Resp: DeleteCertificateConfirmation{}},
		featureset.Pair{Req: ExtendedTriggerMessageRequest{}, Resp: ExtendedTriggerMessageConfirmation{}},
		featureset.Pair{Req: GetInstalledCertificateIdsRequest{}, Resp: GetInstalledCertificateIdsConfirmation{}},
		featureset.Pair{Req: GetLogRequest{}, Resp: GetLogConfirmation{}},
		featureset.Pair{Req: InstallCertificateRequest{}, Resp: InstallCertificateConfirmation{}},
		featureset.Pair{Req: LogStatusNotificationRequest{}, Resp: LogStatusNotificationConfirmation{}},
		featureset.Pair{Req: SecurityEventNotificationRequest{}, Resp: SecurityEventNotificationConfirmation{}},
		featureset.Pair{Req: SignCertificateRequest{}, Resp: SignCertificateConfirmation{}},
		featureset.Pair{Req: SignedFirmwareStatusNotificationRequest{}, Resp: SignedFirmwareStatusNotificationConfirmation{}},
		featureset.Pair{Req: SignedUpdateFirmwareRequest{}, Resp: SignedUpdateFirmwareConfirmation{}},
	)
}
