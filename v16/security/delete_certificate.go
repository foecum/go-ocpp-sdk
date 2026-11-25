package security

import "github.com/foecum/go-ocpp-sdk/v16/types"

// DeleteCertificateAction is the action name of the DeleteCertificate message.
const DeleteCertificateAction = "DeleteCertificate"

// DeleteCertificateRequest is the DeleteCertificate.req PDU, sent by the
// Central System to delete an installed certificate.
type DeleteCertificateRequest struct {
	// CertificateHashData identifies the certificate to delete.
	CertificateHashData CertificateHashData `json:"certificateHashData" ocpp:"required"`
}

// Action implements ocpp.Request.
func (DeleteCertificateRequest) Action() string { return DeleteCertificateAction }

// Validate implements ocpp.Request.
func (r DeleteCertificateRequest) Validate() error {
	return types.Prefix("certificateHashData", r.CertificateHashData.Validate())
}

// DeleteCertificateConfirmation is the DeleteCertificate.conf PDU.
type DeleteCertificateConfirmation struct {
	// Status says whether the certificate was deleted.
	Status DeleteCertificateStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Response.
func (DeleteCertificateConfirmation) Action() string { return DeleteCertificateAction }

// Validate implements ocpp.Response.
func (c DeleteCertificateConfirmation) Validate() error {
	return types.CheckEnum("status", c.Status, true)
}
