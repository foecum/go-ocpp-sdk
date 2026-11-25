// The value lists below are copied from the OCPP 1.6 security whitepaper
// JSON schemas (docs/spec/1.6/schemas/json).

package security

// GenericStatus is a string enumeration. The outcome of a SignCertificate request.
type GenericStatus string

const (
	// GenericStatusAccepted is the "Accepted" value of GenericStatus.
	GenericStatusAccepted GenericStatus = "Accepted"
	// GenericStatusRejected is the "Rejected" value of GenericStatus.
	GenericStatusRejected GenericStatus = "Rejected"
)

// IsValid reports whether v is one of the values defined for GenericStatus.
func (v GenericStatus) IsValid() bool {
	switch v {
	case GenericStatusAccepted, GenericStatusRejected:
		return true
	}
	return false
}

// CertificateSignedStatus is a string enumeration. The outcome of a CertificateSigned request.
type CertificateSignedStatus string

const (
	// CertificateSignedStatusAccepted is the "Accepted" value of CertificateSignedStatus.
	CertificateSignedStatusAccepted CertificateSignedStatus = "Accepted"
	// CertificateSignedStatusRejected is the "Rejected" value of CertificateSignedStatus.
	CertificateSignedStatusRejected CertificateSignedStatus = "Rejected"
)

// IsValid reports whether v is one of the values defined for CertificateSignedStatus.
func (v CertificateSignedStatus) IsValid() bool {
	switch v {
	case CertificateSignedStatusAccepted, CertificateSignedStatusRejected:
		return true
	}
	return false
}

// CertificateUse is a string enumeration. The kind of certificate InstallCertificate or GetInstalledCertificateIds refers to.
type CertificateUse string

const (
	// CertificateUseCentralSystemRootCertificate is the "CentralSystemRootCertificate" value of CertificateUse.
	CertificateUseCentralSystemRootCertificate CertificateUse = "CentralSystemRootCertificate"
	// CertificateUseManufacturerRootCertificate is the "ManufacturerRootCertificate" value of CertificateUse.
	CertificateUseManufacturerRootCertificate CertificateUse = "ManufacturerRootCertificate"
)

// IsValid reports whether v is one of the values defined for CertificateUse.
func (v CertificateUse) IsValid() bool {
	switch v {
	case CertificateUseCentralSystemRootCertificate, CertificateUseManufacturerRootCertificate:
		return true
	}
	return false
}

// HashAlgorithm is a string enumeration. The algorithm used for the hashes identifying a certificate.
type HashAlgorithm string

const (
	// HashAlgorithmSHA256 is the "SHA256" value of HashAlgorithm.
	HashAlgorithmSHA256 HashAlgorithm = "SHA256"
	// HashAlgorithmSHA384 is the "SHA384" value of HashAlgorithm.
	HashAlgorithmSHA384 HashAlgorithm = "SHA384"
	// HashAlgorithmSHA512 is the "SHA512" value of HashAlgorithm.
	HashAlgorithmSHA512 HashAlgorithm = "SHA512"
)

// IsValid reports whether v is one of the values defined for HashAlgorithm.
func (v HashAlgorithm) IsValid() bool {
	switch v {
	case HashAlgorithmSHA256, HashAlgorithmSHA384, HashAlgorithmSHA512:
		return true
	}
	return false
}

// CertificateStatus is a string enumeration. The outcome of an InstallCertificate request.
type CertificateStatus string

const (
	// CertificateStatusAccepted is the "Accepted" value of CertificateStatus.
	CertificateStatusAccepted CertificateStatus = "Accepted"
	// CertificateStatusFailed is the "Failed" value of CertificateStatus.
	CertificateStatusFailed CertificateStatus = "Failed"
	// CertificateStatusRejected is the "Rejected" value of CertificateStatus.
	CertificateStatusRejected CertificateStatus = "Rejected"
)

// IsValid reports whether v is one of the values defined for CertificateStatus.
func (v CertificateStatus) IsValid() bool {
	switch v {
	case CertificateStatusAccepted, CertificateStatusFailed, CertificateStatusRejected:
		return true
	}
	return false
}

// DeleteCertificateStatus is a string enumeration. The outcome of a DeleteCertificate request.
type DeleteCertificateStatus string

const (
	// DeleteCertificateStatusAccepted is the "Accepted" value of DeleteCertificateStatus.
	DeleteCertificateStatusAccepted DeleteCertificateStatus = "Accepted"
	// DeleteCertificateStatusFailed is the "Failed" value of DeleteCertificateStatus.
	DeleteCertificateStatusFailed DeleteCertificateStatus = "Failed"
	// DeleteCertificateStatusNotFound is the "NotFound" value of DeleteCertificateStatus.
	DeleteCertificateStatusNotFound DeleteCertificateStatus = "NotFound"
)

// IsValid reports whether v is one of the values defined for DeleteCertificateStatus.
func (v DeleteCertificateStatus) IsValid() bool {
	switch v {
	case DeleteCertificateStatusAccepted, DeleteCertificateStatusFailed, DeleteCertificateStatusNotFound:
		return true
	}
	return false
}

// GetInstalledCertificateStatus is a string enumeration. The outcome of a GetInstalledCertificateIds request.
type GetInstalledCertificateStatus string

const (
	// GetInstalledCertificateStatusAccepted is the "Accepted" value of GetInstalledCertificateStatus.
	GetInstalledCertificateStatusAccepted GetInstalledCertificateStatus = "Accepted"
	// GetInstalledCertificateStatusNotFound is the "NotFound" value of GetInstalledCertificateStatus.
	GetInstalledCertificateStatusNotFound GetInstalledCertificateStatus = "NotFound"
)

// IsValid reports whether v is one of the values defined for GetInstalledCertificateStatus.
func (v GetInstalledCertificateStatus) IsValid() bool {
	switch v {
	case GetInstalledCertificateStatusAccepted, GetInstalledCertificateStatusNotFound:
		return true
	}
	return false
}

// LogType is a string enumeration. The kind of log a GetLog request asks for.
type LogType string

const (
	// LogTypeDiagnosticsLog is the "DiagnosticsLog" value of LogType.
	LogTypeDiagnosticsLog LogType = "DiagnosticsLog"
	// LogTypeSecurityLog is the "SecurityLog" value of LogType.
	LogTypeSecurityLog LogType = "SecurityLog"
)

// IsValid reports whether v is one of the values defined for LogType.
func (v LogType) IsValid() bool {
	switch v {
	case LogTypeDiagnosticsLog, LogTypeSecurityLog:
		return true
	}
	return false
}

// LogStatus is a string enumeration. The outcome of a GetLog request.
type LogStatus string

const (
	// LogStatusAccepted is the "Accepted" value of LogStatus.
	LogStatusAccepted LogStatus = "Accepted"
	// LogStatusRejected is the "Rejected" value of LogStatus.
	LogStatusRejected LogStatus = "Rejected"
	// LogStatusAcceptedCanceled is the "AcceptedCanceled" value of LogStatus.
	LogStatusAcceptedCanceled LogStatus = "AcceptedCanceled"
)

// IsValid reports whether v is one of the values defined for LogStatus.
func (v LogStatus) IsValid() bool {
	switch v {
	case LogStatusAccepted, LogStatusRejected, LogStatusAcceptedCanceled:
		return true
	}
	return false
}

// UploadLogStatus is a string enumeration. The progress of a log upload.
type UploadLogStatus string

const (
	// UploadLogStatusBadMessage is the "BadMessage" value of UploadLogStatus.
	UploadLogStatusBadMessage UploadLogStatus = "BadMessage"
	// UploadLogStatusIdle is the "Idle" value of UploadLogStatus.
	UploadLogStatusIdle UploadLogStatus = "Idle"
	// UploadLogStatusNotSupportedOperation is the "NotSupportedOperation" value of UploadLogStatus.
	UploadLogStatusNotSupportedOperation UploadLogStatus = "NotSupportedOperation"
	// UploadLogStatusPermissionDenied is the "PermissionDenied" value of UploadLogStatus.
	UploadLogStatusPermissionDenied UploadLogStatus = "PermissionDenied"
	// UploadLogStatusUploaded is the "Uploaded" value of UploadLogStatus.
	UploadLogStatusUploaded UploadLogStatus = "Uploaded"
	// UploadLogStatusUploadFailure is the "UploadFailure" value of UploadLogStatus.
	UploadLogStatusUploadFailure UploadLogStatus = "UploadFailure"
	// UploadLogStatusUploading is the "Uploading" value of UploadLogStatus.
	UploadLogStatusUploading UploadLogStatus = "Uploading"
)

// IsValid reports whether v is one of the values defined for UploadLogStatus.
func (v UploadLogStatus) IsValid() bool {
	switch v {
	case UploadLogStatusBadMessage, UploadLogStatusIdle, UploadLogStatusNotSupportedOperation, UploadLogStatusPermissionDenied, UploadLogStatusUploaded, UploadLogStatusUploadFailure, UploadLogStatusUploading:
		return true
	}
	return false
}

// SignedFirmwareStatus is a string enumeration. The progress of a signed firmware update. It has more values than the Core firmware status.
type SignedFirmwareStatus string

const (
	// SignedFirmwareStatusDownloaded is the "Downloaded" value of SignedFirmwareStatus.
	SignedFirmwareStatusDownloaded SignedFirmwareStatus = "Downloaded"
	// SignedFirmwareStatusDownloadFailed is the "DownloadFailed" value of SignedFirmwareStatus.
	SignedFirmwareStatusDownloadFailed SignedFirmwareStatus = "DownloadFailed"
	// SignedFirmwareStatusDownloading is the "Downloading" value of SignedFirmwareStatus.
	SignedFirmwareStatusDownloading SignedFirmwareStatus = "Downloading"
	// SignedFirmwareStatusDownloadScheduled is the "DownloadScheduled" value of SignedFirmwareStatus.
	SignedFirmwareStatusDownloadScheduled SignedFirmwareStatus = "DownloadScheduled"
	// SignedFirmwareStatusDownloadPaused is the "DownloadPaused" value of SignedFirmwareStatus.
	SignedFirmwareStatusDownloadPaused SignedFirmwareStatus = "DownloadPaused"
	// SignedFirmwareStatusIdle is the "Idle" value of SignedFirmwareStatus.
	SignedFirmwareStatusIdle SignedFirmwareStatus = "Idle"
	// SignedFirmwareStatusInstallationFailed is the "InstallationFailed" value of SignedFirmwareStatus.
	SignedFirmwareStatusInstallationFailed SignedFirmwareStatus = "InstallationFailed"
	// SignedFirmwareStatusInstalling is the "Installing" value of SignedFirmwareStatus.
	SignedFirmwareStatusInstalling SignedFirmwareStatus = "Installing"
	// SignedFirmwareStatusInstalled is the "Installed" value of SignedFirmwareStatus.
	SignedFirmwareStatusInstalled SignedFirmwareStatus = "Installed"
	// SignedFirmwareStatusInstallRebooting is the "InstallRebooting" value of SignedFirmwareStatus.
	SignedFirmwareStatusInstallRebooting SignedFirmwareStatus = "InstallRebooting"
	// SignedFirmwareStatusInstallScheduled is the "InstallScheduled" value of SignedFirmwareStatus.
	SignedFirmwareStatusInstallScheduled SignedFirmwareStatus = "InstallScheduled"
	// SignedFirmwareStatusInstallVerificationFailed is the "InstallVerificationFailed" value of SignedFirmwareStatus.
	SignedFirmwareStatusInstallVerificationFailed SignedFirmwareStatus = "InstallVerificationFailed"
	// SignedFirmwareStatusInvalidSignature is the "InvalidSignature" value of SignedFirmwareStatus.
	SignedFirmwareStatusInvalidSignature SignedFirmwareStatus = "InvalidSignature"
	// SignedFirmwareStatusSignatureVerified is the "SignatureVerified" value of SignedFirmwareStatus.
	SignedFirmwareStatusSignatureVerified SignedFirmwareStatus = "SignatureVerified"
)

// IsValid reports whether v is one of the values defined for SignedFirmwareStatus.
func (v SignedFirmwareStatus) IsValid() bool {
	switch v {
	case SignedFirmwareStatusDownloaded, SignedFirmwareStatusDownloadFailed, SignedFirmwareStatusDownloading, SignedFirmwareStatusDownloadScheduled, SignedFirmwareStatusDownloadPaused, SignedFirmwareStatusIdle, SignedFirmwareStatusInstallationFailed, SignedFirmwareStatusInstalling, SignedFirmwareStatusInstalled, SignedFirmwareStatusInstallRebooting, SignedFirmwareStatusInstallScheduled, SignedFirmwareStatusInstallVerificationFailed, SignedFirmwareStatusInvalidSignature, SignedFirmwareStatusSignatureVerified:
		return true
	}
	return false
}

// SignedUpdateFirmwareStatus is a string enumeration. The outcome of a SignedUpdateFirmware request.
type SignedUpdateFirmwareStatus string

const (
	// SignedUpdateFirmwareStatusAccepted is the "Accepted" value of SignedUpdateFirmwareStatus.
	SignedUpdateFirmwareStatusAccepted SignedUpdateFirmwareStatus = "Accepted"
	// SignedUpdateFirmwareStatusRejected is the "Rejected" value of SignedUpdateFirmwareStatus.
	SignedUpdateFirmwareStatusRejected SignedUpdateFirmwareStatus = "Rejected"
	// SignedUpdateFirmwareStatusAcceptedCanceled is the "AcceptedCanceled" value of SignedUpdateFirmwareStatus.
	SignedUpdateFirmwareStatusAcceptedCanceled SignedUpdateFirmwareStatus = "AcceptedCanceled"
	// SignedUpdateFirmwareStatusInvalidCertificate is the "InvalidCertificate" value of SignedUpdateFirmwareStatus.
	SignedUpdateFirmwareStatusInvalidCertificate SignedUpdateFirmwareStatus = "InvalidCertificate"
	// SignedUpdateFirmwareStatusRevokedCertificate is the "RevokedCertificate" value of SignedUpdateFirmwareStatus.
	SignedUpdateFirmwareStatusRevokedCertificate SignedUpdateFirmwareStatus = "RevokedCertificate"
)

// IsValid reports whether v is one of the values defined for SignedUpdateFirmwareStatus.
func (v SignedUpdateFirmwareStatus) IsValid() bool {
	switch v {
	case SignedUpdateFirmwareStatusAccepted, SignedUpdateFirmwareStatusRejected, SignedUpdateFirmwareStatusAcceptedCanceled, SignedUpdateFirmwareStatusInvalidCertificate, SignedUpdateFirmwareStatusRevokedCertificate:
		return true
	}
	return false
}

// ExtendedMessageTrigger is a string enumeration. The message an ExtendedTriggerMessage request asks the Charge Point to send.
type ExtendedMessageTrigger string

const (
	// ExtendedMessageTriggerBootNotification is the "BootNotification" value of ExtendedMessageTrigger.
	ExtendedMessageTriggerBootNotification ExtendedMessageTrigger = "BootNotification"
	// ExtendedMessageTriggerLogStatusNotification is the "LogStatusNotification" value of ExtendedMessageTrigger.
	ExtendedMessageTriggerLogStatusNotification ExtendedMessageTrigger = "LogStatusNotification"
	// ExtendedMessageTriggerFirmwareStatusNotification is the "FirmwareStatusNotification" value of ExtendedMessageTrigger.
	ExtendedMessageTriggerFirmwareStatusNotification ExtendedMessageTrigger = "FirmwareStatusNotification"
	// ExtendedMessageTriggerHeartbeat is the "Heartbeat" value of ExtendedMessageTrigger.
	ExtendedMessageTriggerHeartbeat ExtendedMessageTrigger = "Heartbeat"
	// ExtendedMessageTriggerMeterValues is the "MeterValues" value of ExtendedMessageTrigger.
	ExtendedMessageTriggerMeterValues ExtendedMessageTrigger = "MeterValues"
	// ExtendedMessageTriggerSignChargePointCertificate is the "SignChargePointCertificate" value of ExtendedMessageTrigger.
	ExtendedMessageTriggerSignChargePointCertificate ExtendedMessageTrigger = "SignChargePointCertificate"
	// ExtendedMessageTriggerStatusNotification is the "StatusNotification" value of ExtendedMessageTrigger.
	ExtendedMessageTriggerStatusNotification ExtendedMessageTrigger = "StatusNotification"
)

// IsValid reports whether v is one of the values defined for ExtendedMessageTrigger.
func (v ExtendedMessageTrigger) IsValid() bool {
	switch v {
	case ExtendedMessageTriggerBootNotification, ExtendedMessageTriggerLogStatusNotification, ExtendedMessageTriggerFirmwareStatusNotification, ExtendedMessageTriggerHeartbeat, ExtendedMessageTriggerMeterValues, ExtendedMessageTriggerSignChargePointCertificate, ExtendedMessageTriggerStatusNotification:
		return true
	}
	return false
}
