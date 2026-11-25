package firmware

import "github.com/foecum/go-ocpp-sdk/v16/types"

// UpdateFirmwareAction is the action name of the UpdateFirmware message.
const UpdateFirmwareAction = "UpdateFirmware"

// UpdateFirmwareRequest is the UpdateFirmware.req PDU, sent by the Central
// System to tell the Charge Point to download and install new firmware.
type UpdateFirmwareRequest struct {
	// Location is the URI to retrieve the firmware from.
	Location string `json:"location" ocpp:"required"`
	// Retries is how many times to try the download before giving up. If absent
	// the Charge Point decides.
	Retries *int `json:"retries,omitempty"`
	// RetrieveDate is the time after which the Charge Point may retrieve the
	// firmware.
	RetrieveDate types.DateTime `json:"retrieveDate" ocpp:"required"`
	// RetryInterval is the number of seconds between attempts. If absent the
	// Charge Point decides.
	RetryInterval *int `json:"retryInterval,omitempty"`
}

// Action implements ocpp.Request.
func (UpdateFirmwareRequest) Action() string { return UpdateFirmwareAction }

// Validate implements ocpp.Request.
func (r UpdateFirmwareRequest) Validate() error {
	return types.First(
		types.CheckURI("location", r.Location, true),
		types.CheckMinPtr("retries", r.Retries, 0),
		types.CheckDateTime("retrieveDate", r.RetrieveDate),
		types.CheckMinPtr("retryInterval", r.RetryInterval, 0),
	)
}

// UpdateFirmwareConfirmation is the UpdateFirmware.conf PDU. It has no fields.
type UpdateFirmwareConfirmation struct{}

// Action implements ocpp.Response.
func (UpdateFirmwareConfirmation) Action() string { return UpdateFirmwareAction }

// Validate implements ocpp.Response.
func (UpdateFirmwareConfirmation) Validate() error { return nil }
