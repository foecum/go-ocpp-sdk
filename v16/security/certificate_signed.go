package security

import "github.com/foecum/go-ocpp-sdk/v16/types"

// CertificateSignedAction is the action name of the CertificateSigned message.
const CertificateSignedAction = "CertificateSigned"

// CertificateSignedRequest is the CertificateSigned.req PDU, sent by the
// Central System to deliver the certificate it signed after a SignCertificate
// request.
type CertificateSignedRequest struct {
	// CertificateChain is the signed PEM encoded X.509 certificate, optionally
	// followed by the sub CA certificates (at most 10000 characters).
	CertificateChain string `json:"certificateChain" ocpp:"required"`
}

// Action implements ocpp.Request.
func (CertificateSignedRequest) Action() string { return CertificateSignedAction }

// Validate implements ocpp.Request.
func (r CertificateSignedRequest) Validate() error {
	return checkPEM("certificateChain", r.CertificateChain, 10000, 1<<30, pemCertificate)
}

// CertificateSignedConfirmation is the CertificateSigned.conf PDU.
type CertificateSignedConfirmation struct {
	// Status says whether the signed certificate was accepted.
	Status CertificateSignedStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Response.
func (CertificateSignedConfirmation) Action() string { return CertificateSignedAction }

// Validate implements ocpp.Response.
func (c CertificateSignedConfirmation) Validate() error {
	return types.CheckEnum("status", c.Status, true)
}
