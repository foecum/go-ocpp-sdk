package csms

import (
	"context"

	"github.com/foecum/go-ocpp-sdk/internal/rpc"
	"github.com/foecum/go-ocpp-sdk/v16/security"
)

// SecurityHandler handles the security extension calls a charge point sends to
// the Central System. Embed UnimplementedSecurityHandler to implement only some
// of them.
type SecurityHandler interface {
	// LogStatusNotification answers a LogStatusNotification request.
	LogStatusNotification(ctx context.Context, cp *ChargePoint, req *security.LogStatusNotificationRequest) (*security.LogStatusNotificationConfirmation, error)
	// SecurityEventNotification answers a SecurityEventNotification request.
	SecurityEventNotification(ctx context.Context, cp *ChargePoint, req *security.SecurityEventNotificationRequest) (*security.SecurityEventNotificationConfirmation, error)
	// SignCertificate answers a SignCertificate request. To deliver the signed
	// certificate, call ChargePoint.CertificateSigned afterwards (not from within
	// the handler's reply).
	SignCertificate(ctx context.Context, cp *ChargePoint, req *security.SignCertificateRequest) (*security.SignCertificateConfirmation, error)
	// SignedFirmwareStatusNotification answers a SignedFirmwareStatusNotification request.
	SignedFirmwareStatusNotification(ctx context.Context, cp *ChargePoint, req *security.SignedFirmwareStatusNotificationRequest) (*security.SignedFirmwareStatusNotificationConfirmation, error)
}

// UnimplementedSecurityHandler implements SecurityHandler by answering every
// call with a NotSupported CALLERROR.
type UnimplementedSecurityHandler struct{}

var _ SecurityHandler = UnimplementedSecurityHandler{}

// LogStatusNotification returns NotSupported.
func (UnimplementedSecurityHandler) LogStatusNotification(context.Context, *ChargePoint, *security.LogStatusNotificationRequest) (*security.LogStatusNotificationConfirmation, error) {
	return nil, rpc.NotSupported(security.LogStatusNotificationAction)
}

// SecurityEventNotification returns NotSupported.
func (UnimplementedSecurityHandler) SecurityEventNotification(context.Context, *ChargePoint, *security.SecurityEventNotificationRequest) (*security.SecurityEventNotificationConfirmation, error) {
	return nil, rpc.NotSupported(security.SecurityEventNotificationAction)
}

// SignCertificate returns NotSupported.
func (UnimplementedSecurityHandler) SignCertificate(context.Context, *ChargePoint, *security.SignCertificateRequest) (*security.SignCertificateConfirmation, error) {
	return nil, rpc.NotSupported(security.SignCertificateAction)
}

// SignedFirmwareStatusNotification returns NotSupported.
func (UnimplementedSecurityHandler) SignedFirmwareStatusNotification(context.Context, *ChargePoint, *security.SignedFirmwareStatusNotificationRequest) (*security.SignedFirmwareStatusNotificationConfirmation, error) {
	return nil, rpc.NotSupported(security.SignedFirmwareStatusNotificationAction)
}
