package security

// Security event types defined by the whitepaper, for
// SecurityEventNotificationRequest.Type. The field is an open string, so a
// vendor may send others.
const (
	// EventFirmwareUpdated is the "FirmwareUpdated" security event.
	EventFirmwareUpdated = "FirmwareUpdated"
	// EventFailedToAuthenticateAtCentral is the "FailedToAuthenticateAtCentralSystem" security event.
	EventFailedToAuthenticateAtCentral = "FailedToAuthenticateAtCentralSystem"
	// EventCentralSystemFailedToAuthenticate is the "CentralSystemFailedToAuthenticate" security event.
	EventCentralSystemFailedToAuthenticate = "CentralSystemFailedToAuthenticate"
	// EventSettingSystemTime is the "SettingSystemTime" security event.
	EventSettingSystemTime = "SettingSystemTime"
	// EventStartupOfTheDevice is the "StartupOfTheDevice" security event.
	EventStartupOfTheDevice = "StartupOfTheDevice"
	// EventResetOrReboot is the "ResetOrReboot" security event.
	EventResetOrReboot = "ResetOrReboot"
	// EventSecurityLogWasCleared is the "SecurityLogWasCleared" security event.
	EventSecurityLogWasCleared = "SecurityLogWasCleared"
	// EventReconfigurationOfSecurityParams is the "ReconfigurationOfSecurityParameters" security event.
	EventReconfigurationOfSecurityParams = "ReconfigurationOfSecurityParameters"
	// EventMemoryExhaustion is the "MemoryExhaustion" security event.
	EventMemoryExhaustion = "MemoryExhaustion"
	// EventInvalidMessages is the "InvalidMessages" security event.
	EventInvalidMessages = "InvalidMessages"
	// EventAttemptedReplayAttacks is the "AttemptedReplayAttacks" security event.
	EventAttemptedReplayAttacks = "AttemptedReplayAttacks"
	// EventTamperDetectionActivated is the "TamperDetectionActivated" security event.
	EventTamperDetectionActivated = "TamperDetectionActivated"
	// EventInvalidFirmwareSignature is the "InvalidFirmwareSignature" security event.
	EventInvalidFirmwareSignature = "InvalidFirmwareSignature"
	// EventInvalidFirmwareSigningCertificate is the "InvalidFirmwareSigningCertificate" security event.
	EventInvalidFirmwareSigningCertificate = "InvalidFirmwareSigningCertificate"
	// EventInvalidCentralSystemCertificate is the "InvalidCentralSystemCertificate" security event.
	EventInvalidCentralSystemCertificate = "InvalidCentralSystemCertificate"
	// EventInvalidChargePointCertificate is the "InvalidChargePointCertificate" security event.
	EventInvalidChargePointCertificate = "InvalidChargePointCertificate"
	// EventInvalidTLSVersion is the "InvalidTLSVersion" security event.
	EventInvalidTLSVersion = "InvalidTLSVersion"
	// EventInvalidTLSCipherSuite is the "InvalidTLSCipherSuite" security event.
	EventInvalidTLSCipherSuite = "InvalidTLSCipherSuite"
)

// criticalEvents maps every event type in the whitepaper's table to whether it
// is critical. The whitepaper says critical events should be pushed to the
// Central System.
var criticalEvents = map[string]bool{
	EventFirmwareUpdated:                   true,
	EventFailedToAuthenticateAtCentral:     false,
	EventCentralSystemFailedToAuthenticate: false,
	EventSettingSystemTime:                 true,
	EventStartupOfTheDevice:                true,
	EventResetOrReboot:                     true,
	EventSecurityLogWasCleared:             true,
	EventReconfigurationOfSecurityParams:   false,
	EventMemoryExhaustion:                  true,
	EventInvalidMessages:                   false,
	EventAttemptedReplayAttacks:            false,
	EventTamperDetectionActivated:          true,
	EventInvalidFirmwareSignature:          false,
	EventInvalidFirmwareSigningCertificate: false,
	EventInvalidCentralSystemCertificate:   false,
	EventInvalidChargePointCertificate:     false,
	EventInvalidTLSVersion:                 false,
	EventInvalidTLSCipherSuite:             false,
}

// IsKnownEvent reports whether eventType is one of the security events the
// whitepaper defines.
func IsKnownEvent(eventType string) bool {
	_, ok := criticalEvents[eventType]
	return ok
}

// IsCriticalEvent reports whether eventType is a critical security event: one
// the whitepaper says should be pushed to the Central System. Unknown event
// types are not critical.
func IsCriticalEvent(eventType string) bool { return criticalEvents[eventType] }
