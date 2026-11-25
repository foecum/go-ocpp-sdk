package security

import "github.com/foecum/go-ocpp-sdk/v16/types"

// SignedUpdateFirmwareAction is the action name of the SignedUpdateFirmware message.
const SignedUpdateFirmwareAction = "SignedUpdateFirmware"

// SignedUpdateFirmwareRequest is the SignedUpdateFirmware.req PDU, sent by the
// Central System to have the Charge Point download, verify and install signed
// firmware.
type SignedUpdateFirmwareRequest struct {
	// Retries is how many times to try the download. If absent the Charge Point
	// decides.
	Retries *int `json:"retries,omitempty"`
	// RetryInterval is the number of seconds between attempts. If absent the
	// Charge Point decides.
	RetryInterval *int `json:"retryInterval,omitempty"`
	// RequestID identifies this request; the Charge Point repeats it in its
	// SignedFirmwareStatusNotification messages.
	RequestID int `json:"requestId" ocpp:"required"`
	// Firmware describes the firmware to install.
	Firmware Firmware `json:"firmware" ocpp:"required"`
}

// Action implements ocpp.Request.
func (SignedUpdateFirmwareRequest) Action() string { return SignedUpdateFirmwareAction }

// Validate implements ocpp.Request.
func (r SignedUpdateFirmwareRequest) Validate() error {
	if err := types.First(
		types.CheckMinPtr("retries", r.Retries, 0),
		types.CheckMinPtr("retryInterval", r.RetryInterval, 0),
	); err != nil {
		return err
	}
	return types.Prefix("firmware", r.Firmware.Validate())
}

// SignedUpdateFirmwareConfirmation is the SignedUpdateFirmware.conf PDU.
type SignedUpdateFirmwareConfirmation struct {
	// Status says whether the Charge Point accepted the request.
	Status SignedUpdateFirmwareStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Response.
func (SignedUpdateFirmwareConfirmation) Action() string { return SignedUpdateFirmwareAction }

// Validate implements ocpp.Response.
func (c SignedUpdateFirmwareConfirmation) Validate() error {
	return types.CheckEnum("status", c.Status, true)
}
