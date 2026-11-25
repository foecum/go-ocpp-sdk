package firmware

import "github.com/foecum/go-ocpp-sdk/v16/types"

// GetDiagnosticsAction is the action name of the GetDiagnostics message.
const GetDiagnosticsAction = "GetDiagnostics"

// GetDiagnosticsRequest is the GetDiagnostics.req PDU, sent by the Central
// System to ask the Charge Point to upload its diagnostics.
type GetDiagnosticsRequest struct {
	// Location is the URI of the directory to upload the diagnostics file to.
	Location string `json:"location" ocpp:"required"`
	// Retries is how many times to try the upload before giving up. If absent
	// the Charge Point decides.
	Retries *int `json:"retries,omitempty"`
	// RetryInterval is the number of seconds between attempts. If absent the
	// Charge Point decides.
	RetryInterval *int `json:"retryInterval,omitempty"`
	// StartTime is the oldest logging information to include.
	StartTime *types.DateTime `json:"startTime,omitempty"`
	// StopTime is the latest logging information to include.
	StopTime *types.DateTime `json:"stopTime,omitempty"`
}

// Action implements ocpp.Request.
func (GetDiagnosticsRequest) Action() string { return GetDiagnosticsAction }

// Validate implements ocpp.Request.
func (r GetDiagnosticsRequest) Validate() error {
	return types.First(
		types.CheckURI("location", r.Location, true),
		types.CheckMinPtr("retries", r.Retries, 0),
		types.CheckMinPtr("retryInterval", r.RetryInterval, 0),
	)
}

// GetDiagnosticsConfirmation is the GetDiagnostics.conf PDU.
type GetDiagnosticsConfirmation struct {
	// FileName is the name of the file that will be uploaded (at most 255
	// characters). It is absent if no diagnostics are available.
	FileName string `json:"fileName,omitempty"`
}

// Action implements ocpp.Response.
func (GetDiagnosticsConfirmation) Action() string { return GetDiagnosticsAction }

// Validate implements ocpp.Response.
func (c GetDiagnosticsConfirmation) Validate() error {
	return types.CheckString("fileName", c.FileName, types.CiString255, false)
}
