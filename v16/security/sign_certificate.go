package security

import "github.com/foecum/go-ocpp-sdk/v16/types"

// SignCertificateAction is the action name of the SignCertificate message.
const SignCertificateAction = "SignCertificate"

// SignCertificateRequest is the SignCertificate.req PDU, sent by the Charge
// Point to ask the Central System's Certificate Authority to sign its public key
// into a certificate.
type SignCertificateRequest struct {
	// CSR is the PEM encoded certificate signing request (RFC 2986), at most
	// 5500 characters.
	CSR string `json:"csr" ocpp:"required"`
}

// Action implements ocpp.Request.
func (SignCertificateRequest) Action() string { return SignCertificateAction }

// Validate implements ocpp.Request.
func (r SignCertificateRequest) Validate() error {
	return checkPEM("csr", r.CSR, 5500, 1, pemCSR, pemCSRLegacy)
}

// SignCertificateConfirmation is the SignCertificate.conf PDU.
type SignCertificateConfirmation struct {
	// Status says whether the Central System can process the request.
	Status GenericStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Response.
func (SignCertificateConfirmation) Action() string { return SignCertificateAction }

// Validate implements ocpp.Response.
func (c SignCertificateConfirmation) Validate() error {
	return types.CheckEnum("status", c.Status, true)
}
