package chargepoint

import (
	"context"

	"github.com/foecum/go-ocpp-sdk/internal/rpc"
	"github.com/foecum/go-ocpp-sdk/v16/security"
)

// SecurityHandler handles the security extension calls the Central System
// sends. Embed UnimplementedSecurityHandler to implement only some of them.
type SecurityHandler interface {
	// CertificateSigned answers a CertificateSigned request.
	CertificateSigned(ctx context.Context, req *security.CertificateSignedRequest) (*security.CertificateSignedConfirmation, error)
	// DeleteCertificate answers a DeleteCertificate request.
	DeleteCertificate(ctx context.Context, req *security.DeleteCertificateRequest) (*security.DeleteCertificateConfirmation, error)
	// ExtendedTriggerMessage answers an ExtendedTriggerMessage request.
	ExtendedTriggerMessage(ctx context.Context, req *security.ExtendedTriggerMessageRequest) (*security.ExtendedTriggerMessageConfirmation, error)
	// GetInstalledCertificateIds answers a GetInstalledCertificateIds request.
	GetInstalledCertificateIds(ctx context.Context, req *security.GetInstalledCertificateIdsRequest) (*security.GetInstalledCertificateIdsConfirmation, error)
	// GetLog answers a GetLog request.
	GetLog(ctx context.Context, req *security.GetLogRequest) (*security.GetLogConfirmation, error)
	// InstallCertificate answers an InstallCertificate request.
	InstallCertificate(ctx context.Context, req *security.InstallCertificateRequest) (*security.InstallCertificateConfirmation, error)
	// SignedUpdateFirmware answers a SignedUpdateFirmware request.
	SignedUpdateFirmware(ctx context.Context, req *security.SignedUpdateFirmwareRequest) (*security.SignedUpdateFirmwareConfirmation, error)
}

// UnimplementedSecurityHandler implements SecurityHandler by answering every
// call with a NotSupported CALLERROR.
type UnimplementedSecurityHandler struct{}

var _ SecurityHandler = UnimplementedSecurityHandler{}

// CertificateSigned returns NotSupported.
func (UnimplementedSecurityHandler) CertificateSigned(context.Context, *security.CertificateSignedRequest) (*security.CertificateSignedConfirmation, error) {
	return nil, rpc.NotSupported(security.CertificateSignedAction)
}

// DeleteCertificate returns NotSupported.
func (UnimplementedSecurityHandler) DeleteCertificate(context.Context, *security.DeleteCertificateRequest) (*security.DeleteCertificateConfirmation, error) {
	return nil, rpc.NotSupported(security.DeleteCertificateAction)
}

// ExtendedTriggerMessage returns NotSupported.
func (UnimplementedSecurityHandler) ExtendedTriggerMessage(context.Context, *security.ExtendedTriggerMessageRequest) (*security.ExtendedTriggerMessageConfirmation, error) {
	return nil, rpc.NotSupported(security.ExtendedTriggerMessageAction)
}

// GetInstalledCertificateIds returns NotSupported.
func (UnimplementedSecurityHandler) GetInstalledCertificateIds(context.Context, *security.GetInstalledCertificateIdsRequest) (*security.GetInstalledCertificateIdsConfirmation, error) {
	return nil, rpc.NotSupported(security.GetInstalledCertificateIdsAction)
}

// GetLog returns NotSupported.
func (UnimplementedSecurityHandler) GetLog(context.Context, *security.GetLogRequest) (*security.GetLogConfirmation, error) {
	return nil, rpc.NotSupported(security.GetLogAction)
}

// InstallCertificate returns NotSupported.
func (UnimplementedSecurityHandler) InstallCertificate(context.Context, *security.InstallCertificateRequest) (*security.InstallCertificateConfirmation, error) {
	return nil, rpc.NotSupported(security.InstallCertificateAction)
}

// SignedUpdateFirmware returns NotSupported.
func (UnimplementedSecurityHandler) SignedUpdateFirmware(context.Context, *security.SignedUpdateFirmwareRequest) (*security.SignedUpdateFirmwareConfirmation, error) {
	return nil, rpc.NotSupported(security.SignedUpdateFirmwareAction)
}
