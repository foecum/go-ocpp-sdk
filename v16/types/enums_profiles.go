// The value lists below are copied from the OCPP 1.6 JSON schemas
// (docs/spec/1.6/schemas/json).

package types

// DiagnosticsStatus is a string enumeration. The progress of a diagnostics upload.
type DiagnosticsStatus string

const (
	// DiagnosticsStatusIdle is the "Idle" value of DiagnosticsStatus.
	DiagnosticsStatusIdle DiagnosticsStatus = "Idle"
	// DiagnosticsStatusUploaded is the "Uploaded" value of DiagnosticsStatus.
	DiagnosticsStatusUploaded DiagnosticsStatus = "Uploaded"
	// DiagnosticsStatusUploadFailed is the "UploadFailed" value of DiagnosticsStatus.
	DiagnosticsStatusUploadFailed DiagnosticsStatus = "UploadFailed"
	// DiagnosticsStatusUploading is the "Uploading" value of DiagnosticsStatus.
	DiagnosticsStatusUploading DiagnosticsStatus = "Uploading"
)

// IsValid reports whether v is one of the values defined for DiagnosticsStatus.
func (v DiagnosticsStatus) IsValid() bool {
	switch v {
	case DiagnosticsStatusIdle, DiagnosticsStatusUploaded, DiagnosticsStatusUploadFailed, DiagnosticsStatusUploading:
		return true
	}
	return false
}

// FirmwareStatus is a string enumeration. The progress of a firmware download and installation.
type FirmwareStatus string

const (
	// FirmwareStatusDownloaded is the "Downloaded" value of FirmwareStatus.
	FirmwareStatusDownloaded FirmwareStatus = "Downloaded"
	// FirmwareStatusDownloadFailed is the "DownloadFailed" value of FirmwareStatus.
	FirmwareStatusDownloadFailed FirmwareStatus = "DownloadFailed"
	// FirmwareStatusDownloading is the "Downloading" value of FirmwareStatus.
	FirmwareStatusDownloading FirmwareStatus = "Downloading"
	// FirmwareStatusIdle is the "Idle" value of FirmwareStatus.
	FirmwareStatusIdle FirmwareStatus = "Idle"
	// FirmwareStatusInstallationFailed is the "InstallationFailed" value of FirmwareStatus.
	FirmwareStatusInstallationFailed FirmwareStatus = "InstallationFailed"
	// FirmwareStatusInstalling is the "Installing" value of FirmwareStatus.
	FirmwareStatusInstalling FirmwareStatus = "Installing"
	// FirmwareStatusInstalled is the "Installed" value of FirmwareStatus.
	FirmwareStatusInstalled FirmwareStatus = "Installed"
)

// IsValid reports whether v is one of the values defined for FirmwareStatus.
func (v FirmwareStatus) IsValid() bool {
	switch v {
	case FirmwareStatusDownloaded, FirmwareStatusDownloadFailed, FirmwareStatusDownloading, FirmwareStatusIdle, FirmwareStatusInstallationFailed, FirmwareStatusInstalling, FirmwareStatusInstalled:
		return true
	}
	return false
}

// UpdateType is a string enumeration. Whether a SendLocalList request replaces or amends the local authorisation list.
type UpdateType string

const (
	// UpdateTypeDifferential is the "Differential" value of UpdateType.
	UpdateTypeDifferential UpdateType = "Differential"
	// UpdateTypeFull is the "Full" value of UpdateType.
	UpdateTypeFull UpdateType = "Full"
)

// IsValid reports whether v is one of the values defined for UpdateType.
func (v UpdateType) IsValid() bool {
	switch v {
	case UpdateTypeDifferential, UpdateTypeFull:
		return true
	}
	return false
}

// UpdateStatus is a string enumeration. The outcome of a SendLocalList request.
type UpdateStatus string

const (
	// UpdateStatusAccepted is the "Accepted" value of UpdateStatus.
	UpdateStatusAccepted UpdateStatus = "Accepted"
	// UpdateStatusFailed is the "Failed" value of UpdateStatus.
	UpdateStatusFailed UpdateStatus = "Failed"
	// UpdateStatusNotSupported is the "NotSupported" value of UpdateStatus.
	UpdateStatusNotSupported UpdateStatus = "NotSupported"
	// UpdateStatusVersionMismatch is the "VersionMismatch" value of UpdateStatus.
	UpdateStatusVersionMismatch UpdateStatus = "VersionMismatch"
)

// IsValid reports whether v is one of the values defined for UpdateStatus.
func (v UpdateStatus) IsValid() bool {
	switch v {
	case UpdateStatusAccepted, UpdateStatusFailed, UpdateStatusNotSupported, UpdateStatusVersionMismatch:
		return true
	}
	return false
}

// ReservationStatus is a string enumeration. The outcome of a ReserveNow request.
type ReservationStatus string

const (
	// ReservationStatusAccepted is the "Accepted" value of ReservationStatus.
	ReservationStatusAccepted ReservationStatus = "Accepted"
	// ReservationStatusFaulted is the "Faulted" value of ReservationStatus.
	ReservationStatusFaulted ReservationStatus = "Faulted"
	// ReservationStatusOccupied is the "Occupied" value of ReservationStatus.
	ReservationStatusOccupied ReservationStatus = "Occupied"
	// ReservationStatusRejected is the "Rejected" value of ReservationStatus.
	ReservationStatusRejected ReservationStatus = "Rejected"
	// ReservationStatusUnavailable is the "Unavailable" value of ReservationStatus.
	ReservationStatusUnavailable ReservationStatus = "Unavailable"
)

// IsValid reports whether v is one of the values defined for ReservationStatus.
func (v ReservationStatus) IsValid() bool {
	switch v {
	case ReservationStatusAccepted, ReservationStatusFaulted, ReservationStatusOccupied, ReservationStatusRejected, ReservationStatusUnavailable:
		return true
	}
	return false
}

// CancelReservationStatus is a string enumeration. The outcome of a CancelReservation request.
type CancelReservationStatus string

const (
	// CancelReservationStatusAccepted is the "Accepted" value of CancelReservationStatus.
	CancelReservationStatusAccepted CancelReservationStatus = "Accepted"
	// CancelReservationStatusRejected is the "Rejected" value of CancelReservationStatus.
	CancelReservationStatusRejected CancelReservationStatus = "Rejected"
)

// IsValid reports whether v is one of the values defined for CancelReservationStatus.
func (v CancelReservationStatus) IsValid() bool {
	switch v {
	case CancelReservationStatusAccepted, CancelReservationStatusRejected:
		return true
	}
	return false
}

// ChargingProfileStatus is a string enumeration. The outcome of a SetChargingProfile request.
type ChargingProfileStatus string

const (
	// ChargingProfileStatusAccepted is the "Accepted" value of ChargingProfileStatus.
	ChargingProfileStatusAccepted ChargingProfileStatus = "Accepted"
	// ChargingProfileStatusRejected is the "Rejected" value of ChargingProfileStatus.
	ChargingProfileStatusRejected ChargingProfileStatus = "Rejected"
	// ChargingProfileStatusNotSupported is the "NotSupported" value of ChargingProfileStatus.
	ChargingProfileStatusNotSupported ChargingProfileStatus = "NotSupported"
)

// IsValid reports whether v is one of the values defined for ChargingProfileStatus.
func (v ChargingProfileStatus) IsValid() bool {
	switch v {
	case ChargingProfileStatusAccepted, ChargingProfileStatusRejected, ChargingProfileStatusNotSupported:
		return true
	}
	return false
}

// ClearChargingProfileStatus is a string enumeration. The outcome of a ClearChargingProfile request.
type ClearChargingProfileStatus string

const (
	// ClearChargingProfileStatusAccepted is the "Accepted" value of ClearChargingProfileStatus.
	ClearChargingProfileStatusAccepted ClearChargingProfileStatus = "Accepted"
	// ClearChargingProfileStatusUnknown is the "Unknown" value of ClearChargingProfileStatus.
	ClearChargingProfileStatusUnknown ClearChargingProfileStatus = "Unknown"
)

// IsValid reports whether v is one of the values defined for ClearChargingProfileStatus.
func (v ClearChargingProfileStatus) IsValid() bool {
	switch v {
	case ClearChargingProfileStatusAccepted, ClearChargingProfileStatusUnknown:
		return true
	}
	return false
}

// GetCompositeScheduleStatus is a string enumeration. The outcome of a GetCompositeSchedule request.
type GetCompositeScheduleStatus string

const (
	// GetCompositeScheduleStatusAccepted is the "Accepted" value of GetCompositeScheduleStatus.
	GetCompositeScheduleStatusAccepted GetCompositeScheduleStatus = "Accepted"
	// GetCompositeScheduleStatusRejected is the "Rejected" value of GetCompositeScheduleStatus.
	GetCompositeScheduleStatusRejected GetCompositeScheduleStatus = "Rejected"
)

// IsValid reports whether v is one of the values defined for GetCompositeScheduleStatus.
func (v GetCompositeScheduleStatus) IsValid() bool {
	switch v {
	case GetCompositeScheduleStatusAccepted, GetCompositeScheduleStatusRejected:
		return true
	}
	return false
}

// MessageTrigger is a string enumeration. The message a TriggerMessage request asks the Charge Point to send.
type MessageTrigger string

const (
	// MessageTriggerBootNotification is the "BootNotification" value of MessageTrigger.
	MessageTriggerBootNotification MessageTrigger = "BootNotification"
	// MessageTriggerDiagnosticsStatusNotification is the "DiagnosticsStatusNotification" value of MessageTrigger.
	MessageTriggerDiagnosticsStatusNotification MessageTrigger = "DiagnosticsStatusNotification"
	// MessageTriggerFirmwareStatusNotification is the "FirmwareStatusNotification" value of MessageTrigger.
	MessageTriggerFirmwareStatusNotification MessageTrigger = "FirmwareStatusNotification"
	// MessageTriggerHeartbeat is the "Heartbeat" value of MessageTrigger.
	MessageTriggerHeartbeat MessageTrigger = "Heartbeat"
	// MessageTriggerMeterValues is the "MeterValues" value of MessageTrigger.
	MessageTriggerMeterValues MessageTrigger = "MeterValues"
	// MessageTriggerStatusNotification is the "StatusNotification" value of MessageTrigger.
	MessageTriggerStatusNotification MessageTrigger = "StatusNotification"
)

// IsValid reports whether v is one of the values defined for MessageTrigger.
func (v MessageTrigger) IsValid() bool {
	switch v {
	case MessageTriggerBootNotification, MessageTriggerDiagnosticsStatusNotification, MessageTriggerFirmwareStatusNotification, MessageTriggerHeartbeat, MessageTriggerMeterValues, MessageTriggerStatusNotification:
		return true
	}
	return false
}

// TriggerMessageStatus is a string enumeration. The outcome of a TriggerMessage request.
type TriggerMessageStatus string

const (
	// TriggerMessageStatusAccepted is the "Accepted" value of TriggerMessageStatus.
	TriggerMessageStatusAccepted TriggerMessageStatus = "Accepted"
	// TriggerMessageStatusRejected is the "Rejected" value of TriggerMessageStatus.
	TriggerMessageStatusRejected TriggerMessageStatus = "Rejected"
	// TriggerMessageStatusNotImplemented is the "NotImplemented" value of TriggerMessageStatus.
	TriggerMessageStatusNotImplemented TriggerMessageStatus = "NotImplemented"
)

// IsValid reports whether v is one of the values defined for TriggerMessageStatus.
func (v TriggerMessageStatus) IsValid() bool {
	switch v {
	case TriggerMessageStatusAccepted, TriggerMessageStatusRejected, TriggerMessageStatusNotImplemented:
		return true
	}
	return false
}
