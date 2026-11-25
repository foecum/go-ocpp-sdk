package security

import (
	"fmt"

	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// GetInstalledCertificateIdsAction is the action name of the
// GetInstalledCertificateIds message.
const GetInstalledCertificateIdsAction = "GetInstalledCertificateIds"

// GetInstalledCertificateIdsRequest is the GetInstalledCertificateIds.req PDU,
// sent by the Central System to list the certificates installed on the Charge
// Point.
type GetInstalledCertificateIdsRequest struct {
	// CertificateType says which kind of certificate is requested.
	CertificateType CertificateUse `json:"certificateType" ocpp:"required"`
}

// Action implements ocpp.Request.
func (GetInstalledCertificateIdsRequest) Action() string { return GetInstalledCertificateIdsAction }

// Validate implements ocpp.Request.
func (r GetInstalledCertificateIdsRequest) Validate() error {
	return types.CheckEnum("certificateType", r.CertificateType, true)
}

// GetInstalledCertificateIdsConfirmation is the GetInstalledCertificateIds.conf PDU.
type GetInstalledCertificateIdsConfirmation struct {
	// Status says whether the Charge Point can process the request.
	Status GetInstalledCertificateStatus `json:"status" ocpp:"required"`
	// CertificateHashData lists the installed certificates. The specification
	// allows zero or more entries.
	CertificateHashData []CertificateHashData `json:"certificateHashData,omitempty"`
}

// Action implements ocpp.Response.
func (GetInstalledCertificateIdsConfirmation) Action() string {
	return GetInstalledCertificateIdsAction
}

// Validate implements ocpp.Response.
func (c GetInstalledCertificateIdsConfirmation) Validate() error {
	if err := types.CheckEnum("status", c.Status, true); err != nil {
		return err
	}
	for i, h := range c.CertificateHashData {
		if err := h.Validate(); err != nil {
			return types.Prefix(fmt.Sprintf("certificateHashData[%d]", i), err)
		}
	}
	return nil
}
