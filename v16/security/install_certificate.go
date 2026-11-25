package security

import "github.com/foecum/go-ocpp-sdk/v16/types"

// InstallCertificateAction is the action name of the InstallCertificate message.
const InstallCertificateAction = "InstallCertificate"

// InstallCertificateRequest is the InstallCertificate.req PDU, sent by the
// Central System to install a root certificate on the Charge Point.
type InstallCertificateRequest struct {
	// CertificateType says which trust store the certificate goes in.
	CertificateType CertificateUse `json:"certificateType" ocpp:"required"`
	// Certificate is a PEM encoded X.509 certificate (at most 5500 characters).
	Certificate string `json:"certificate" ocpp:"required"`
}

// Action implements ocpp.Request.
func (InstallCertificateRequest) Action() string { return InstallCertificateAction }

// Validate implements ocpp.Request.
func (r InstallCertificateRequest) Validate() error {
	return types.First(
		types.CheckEnum("certificateType", r.CertificateType, true),
		checkPEM("certificate", r.Certificate, 5500, 1, pemCertificate),
	)
}

// InstallCertificateConfirmation is the InstallCertificate.conf PDU.
type InstallCertificateConfirmation struct {
	// Status says whether the installation succeeded.
	Status CertificateStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Response.
func (InstallCertificateConfirmation) Action() string { return InstallCertificateAction }

// Validate implements ocpp.Response.
func (c InstallCertificateConfirmation) Validate() error {
	return types.CheckEnum("status", c.Status, true)
}
