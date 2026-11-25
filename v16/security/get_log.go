package security

import "github.com/foecum/go-ocpp-sdk/v16/types"

// GetLogAction is the action name of the GetLog message.
const GetLogAction = "GetLog"

// GetLogRequest is the GetLog.req PDU, sent by the Central System to ask the
// Charge Point to upload a log.
type GetLogRequest struct {
	// LogType says which log is requested.
	LogType LogType `json:"logType" ocpp:"required"`
	// RequestID identifies this request; the Charge Point repeats it in its
	// LogStatusNotification messages.
	RequestID int `json:"requestId" ocpp:"required"`
	// Retries is how many times to try the upload. If absent the Charge Point
	// decides.
	Retries *int `json:"retries,omitempty"`
	// RetryInterval is the number of seconds between attempts. If absent the
	// Charge Point decides.
	RetryInterval *int `json:"retryInterval,omitempty"`
	// Log says what to upload and where.
	Log LogParameters `json:"log" ocpp:"required"`
}

// Action implements ocpp.Request.
func (GetLogRequest) Action() string { return GetLogAction }

// Validate implements ocpp.Request.
func (r GetLogRequest) Validate() error {
	if err := types.First(
		types.CheckEnum("logType", r.LogType, true),
		types.CheckMinPtr("retries", r.Retries, 0),
		types.CheckMinPtr("retryInterval", r.RetryInterval, 0),
	); err != nil {
		return err
	}
	return types.Prefix("log", r.Log.Validate())
}

// GetLogConfirmation is the GetLog.conf PDU.
type GetLogConfirmation struct {
	// Status says whether the Charge Point accepted the request.
	Status LogStatus `json:"status" ocpp:"required"`
	// Filename is the name of the file that will be uploaded (at most 255
	// characters). It is absent when no log information is available.
	Filename string `json:"filename,omitempty"`
}

// Action implements ocpp.Response.
func (GetLogConfirmation) Action() string { return GetLogAction }

// Validate implements ocpp.Response.
func (c GetLogConfirmation) Validate() error {
	return types.First(
		types.CheckEnum("status", c.Status, true),
		types.CheckString("filename", c.Filename, types.CiString255, false),
	)
}
