// Code generated from the OCPP 1.6 JSON schemas (docs/spec/1.6/schemas/json); the
// value lists are copied from them, plus the "Celsius" spelling accepted by the
// OCPP-J 1.6 errata. Edit with care.

package types

// AuthorizationStatus is a string enumeration. Whether an idTag has been accepted by the Central System.
type AuthorizationStatus string

const (
	// AuthorizationStatusAccepted is the "Accepted" value of AuthorizationStatus.
	AuthorizationStatusAccepted AuthorizationStatus = "Accepted"
	// AuthorizationStatusBlocked is the "Blocked" value of AuthorizationStatus.
	AuthorizationStatusBlocked AuthorizationStatus = "Blocked"
	// AuthorizationStatusExpired is the "Expired" value of AuthorizationStatus.
	AuthorizationStatusExpired AuthorizationStatus = "Expired"
	// AuthorizationStatusInvalid is the "Invalid" value of AuthorizationStatus.
	AuthorizationStatusInvalid AuthorizationStatus = "Invalid"
	// AuthorizationStatusConcurrentTx is the "ConcurrentTx" value of AuthorizationStatus.
	AuthorizationStatusConcurrentTx AuthorizationStatus = "ConcurrentTx"
)

// IsValid reports whether v is one of the values defined for AuthorizationStatus.
func (v AuthorizationStatus) IsValid() bool {
	switch v {
	case AuthorizationStatusAccepted, AuthorizationStatusBlocked, AuthorizationStatusExpired, AuthorizationStatusInvalid, AuthorizationStatusConcurrentTx:
		return true
	}
	return false
}

// RegistrationStatus is a string enumeration. Whether the Charge Point has been registered with the Central System.
type RegistrationStatus string

const (
	// RegistrationStatusAccepted is the "Accepted" value of RegistrationStatus.
	RegistrationStatusAccepted RegistrationStatus = "Accepted"
	// RegistrationStatusPending is the "Pending" value of RegistrationStatus.
	RegistrationStatusPending RegistrationStatus = "Pending"
	// RegistrationStatusRejected is the "Rejected" value of RegistrationStatus.
	RegistrationStatusRejected RegistrationStatus = "Rejected"
)

// IsValid reports whether v is one of the values defined for RegistrationStatus.
func (v RegistrationStatus) IsValid() bool {
	switch v {
	case RegistrationStatusAccepted, RegistrationStatusPending, RegistrationStatusRejected:
		return true
	}
	return false
}

// AvailabilityType is a string enumeration. The availability change requested by ChangeAvailability.
type AvailabilityType string

const (
	// AvailabilityTypeInoperative is the "Inoperative" value of AvailabilityType.
	AvailabilityTypeInoperative AvailabilityType = "Inoperative"
	// AvailabilityTypeOperative is the "Operative" value of AvailabilityType.
	AvailabilityTypeOperative AvailabilityType = "Operative"
)

// IsValid reports whether v is one of the values defined for AvailabilityType.
func (v AvailabilityType) IsValid() bool {
	switch v {
	case AvailabilityTypeInoperative, AvailabilityTypeOperative:
		return true
	}
	return false
}

// AvailabilityStatus is a string enumeration. Whether the Charge Point can perform the availability change.
type AvailabilityStatus string

const (
	// AvailabilityStatusAccepted is the "Accepted" value of AvailabilityStatus.
	AvailabilityStatusAccepted AvailabilityStatus = "Accepted"
	// AvailabilityStatusRejected is the "Rejected" value of AvailabilityStatus.
	AvailabilityStatusRejected AvailabilityStatus = "Rejected"
	// AvailabilityStatusScheduled is the "Scheduled" value of AvailabilityStatus.
	AvailabilityStatusScheduled AvailabilityStatus = "Scheduled"
)

// IsValid reports whether v is one of the values defined for AvailabilityStatus.
func (v AvailabilityStatus) IsValid() bool {
	switch v {
	case AvailabilityStatusAccepted, AvailabilityStatusRejected, AvailabilityStatusScheduled:
		return true
	}
	return false
}

// ConfigurationStatus is a string enumeration. The outcome of a ChangeConfiguration request.
type ConfigurationStatus string

const (
	// ConfigurationStatusAccepted is the "Accepted" value of ConfigurationStatus.
	ConfigurationStatusAccepted ConfigurationStatus = "Accepted"
	// ConfigurationStatusRejected is the "Rejected" value of ConfigurationStatus.
	ConfigurationStatusRejected ConfigurationStatus = "Rejected"
	// ConfigurationStatusRebootRequired is the "RebootRequired" value of ConfigurationStatus.
	ConfigurationStatusRebootRequired ConfigurationStatus = "RebootRequired"
	// ConfigurationStatusNotSupported is the "NotSupported" value of ConfigurationStatus.
	ConfigurationStatusNotSupported ConfigurationStatus = "NotSupported"
)

// IsValid reports whether v is one of the values defined for ConfigurationStatus.
func (v ConfigurationStatus) IsValid() bool {
	switch v {
	case ConfigurationStatusAccepted, ConfigurationStatusRejected, ConfigurationStatusRebootRequired, ConfigurationStatusNotSupported:
		return true
	}
	return false
}

// ClearCacheStatus is a string enumeration. The outcome of a ClearCache request.
type ClearCacheStatus string

const (
	// ClearCacheStatusAccepted is the "Accepted" value of ClearCacheStatus.
	ClearCacheStatusAccepted ClearCacheStatus = "Accepted"
	// ClearCacheStatusRejected is the "Rejected" value of ClearCacheStatus.
	ClearCacheStatusRejected ClearCacheStatus = "Rejected"
)

// IsValid reports whether v is one of the values defined for ClearCacheStatus.
func (v ClearCacheStatus) IsValid() bool {
	switch v {
	case ClearCacheStatusAccepted, ClearCacheStatusRejected:
		return true
	}
	return false
}

// DataTransferStatus is a string enumeration. The outcome of a DataTransfer request.
type DataTransferStatus string

const (
	// DataTransferStatusAccepted is the "Accepted" value of DataTransferStatus.
	DataTransferStatusAccepted DataTransferStatus = "Accepted"
	// DataTransferStatusRejected is the "Rejected" value of DataTransferStatus.
	DataTransferStatusRejected DataTransferStatus = "Rejected"
	// DataTransferStatusUnknownMessageId is the "UnknownMessageId" value of DataTransferStatus.
	DataTransferStatusUnknownMessageId DataTransferStatus = "UnknownMessageId"
	// DataTransferStatusUnknownVendorId is the "UnknownVendorId" value of DataTransferStatus.
	DataTransferStatusUnknownVendorId DataTransferStatus = "UnknownVendorId"
)

// IsValid reports whether v is one of the values defined for DataTransferStatus.
func (v DataTransferStatus) IsValid() bool {
	switch v {
	case DataTransferStatusAccepted, DataTransferStatusRejected, DataTransferStatusUnknownMessageId, DataTransferStatusUnknownVendorId:
		return true
	}
	return false
}

// RemoteStartStopStatus is a string enumeration. The outcome of a RemoteStartTransaction or RemoteStopTransaction request.
type RemoteStartStopStatus string

const (
	// RemoteStartStopStatusAccepted is the "Accepted" value of RemoteStartStopStatus.
	RemoteStartStopStatusAccepted RemoteStartStopStatus = "Accepted"
	// RemoteStartStopStatusRejected is the "Rejected" value of RemoteStartStopStatus.
	RemoteStartStopStatusRejected RemoteStartStopStatus = "Rejected"
)

// IsValid reports whether v is one of the values defined for RemoteStartStopStatus.
func (v RemoteStartStopStatus) IsValid() bool {
	switch v {
	case RemoteStartStopStatusAccepted, RemoteStartStopStatusRejected:
		return true
	}
	return false
}

// ResetType is a string enumeration. The kind of reset requested.
type ResetType string

const (
	// ResetTypeHard is the "Hard" value of ResetType.
	ResetTypeHard ResetType = "Hard"
	// ResetTypeSoft is the "Soft" value of ResetType.
	ResetTypeSoft ResetType = "Soft"
)

// IsValid reports whether v is one of the values defined for ResetType.
func (v ResetType) IsValid() bool {
	switch v {
	case ResetTypeHard, ResetTypeSoft:
		return true
	}
	return false
}

// ResetStatus is a string enumeration. The outcome of a Reset request.
type ResetStatus string

const (
	// ResetStatusAccepted is the "Accepted" value of ResetStatus.
	ResetStatusAccepted ResetStatus = "Accepted"
	// ResetStatusRejected is the "Rejected" value of ResetStatus.
	ResetStatusRejected ResetStatus = "Rejected"
)

// IsValid reports whether v is one of the values defined for ResetStatus.
func (v ResetStatus) IsValid() bool {
	switch v {
	case ResetStatusAccepted, ResetStatusRejected:
		return true
	}
	return false
}

// UnlockStatus is a string enumeration. The outcome of an UnlockConnector request.
type UnlockStatus string

const (
	// UnlockStatusUnlocked is the "Unlocked" value of UnlockStatus.
	UnlockStatusUnlocked UnlockStatus = "Unlocked"
	// UnlockStatusUnlockFailed is the "UnlockFailed" value of UnlockStatus.
	UnlockStatusUnlockFailed UnlockStatus = "UnlockFailed"
	// UnlockStatusNotSupported is the "NotSupported" value of UnlockStatus.
	UnlockStatusNotSupported UnlockStatus = "NotSupported"
)

// IsValid reports whether v is one of the values defined for UnlockStatus.
func (v UnlockStatus) IsValid() bool {
	switch v {
	case UnlockStatusUnlocked, UnlockStatusUnlockFailed, UnlockStatusNotSupported:
		return true
	}
	return false
}

// ChargePointErrorCode is a string enumeration. The error code reported in a StatusNotification.
type ChargePointErrorCode string

const (
	// ChargePointErrorCodeConnectorLockFailure is the "ConnectorLockFailure" value of ChargePointErrorCode.
	ChargePointErrorCodeConnectorLockFailure ChargePointErrorCode = "ConnectorLockFailure"
	// ChargePointErrorCodeEVCommunicationError is the "EVCommunicationError" value of ChargePointErrorCode.
	ChargePointErrorCodeEVCommunicationError ChargePointErrorCode = "EVCommunicationError"
	// ChargePointErrorCodeGroundFailure is the "GroundFailure" value of ChargePointErrorCode.
	ChargePointErrorCodeGroundFailure ChargePointErrorCode = "GroundFailure"
	// ChargePointErrorCodeHighTemperature is the "HighTemperature" value of ChargePointErrorCode.
	ChargePointErrorCodeHighTemperature ChargePointErrorCode = "HighTemperature"
	// ChargePointErrorCodeInternalError is the "InternalError" value of ChargePointErrorCode.
	ChargePointErrorCodeInternalError ChargePointErrorCode = "InternalError"
	// ChargePointErrorCodeLocalListConflict is the "LocalListConflict" value of ChargePointErrorCode.
	ChargePointErrorCodeLocalListConflict ChargePointErrorCode = "LocalListConflict"
	// ChargePointErrorCodeNoError is the "NoError" value of ChargePointErrorCode.
	ChargePointErrorCodeNoError ChargePointErrorCode = "NoError"
	// ChargePointErrorCodeOtherError is the "OtherError" value of ChargePointErrorCode.
	ChargePointErrorCodeOtherError ChargePointErrorCode = "OtherError"
	// ChargePointErrorCodeOverCurrentFailure is the "OverCurrentFailure" value of ChargePointErrorCode.
	ChargePointErrorCodeOverCurrentFailure ChargePointErrorCode = "OverCurrentFailure"
	// ChargePointErrorCodePowerMeterFailure is the "PowerMeterFailure" value of ChargePointErrorCode.
	ChargePointErrorCodePowerMeterFailure ChargePointErrorCode = "PowerMeterFailure"
	// ChargePointErrorCodePowerSwitchFailure is the "PowerSwitchFailure" value of ChargePointErrorCode.
	ChargePointErrorCodePowerSwitchFailure ChargePointErrorCode = "PowerSwitchFailure"
	// ChargePointErrorCodeReaderFailure is the "ReaderFailure" value of ChargePointErrorCode.
	ChargePointErrorCodeReaderFailure ChargePointErrorCode = "ReaderFailure"
	// ChargePointErrorCodeResetFailure is the "ResetFailure" value of ChargePointErrorCode.
	ChargePointErrorCodeResetFailure ChargePointErrorCode = "ResetFailure"
	// ChargePointErrorCodeUnderVoltage is the "UnderVoltage" value of ChargePointErrorCode.
	ChargePointErrorCodeUnderVoltage ChargePointErrorCode = "UnderVoltage"
	// ChargePointErrorCodeOverVoltage is the "OverVoltage" value of ChargePointErrorCode.
	ChargePointErrorCodeOverVoltage ChargePointErrorCode = "OverVoltage"
	// ChargePointErrorCodeWeakSignal is the "WeakSignal" value of ChargePointErrorCode.
	ChargePointErrorCodeWeakSignal ChargePointErrorCode = "WeakSignal"
)

// IsValid reports whether v is one of the values defined for ChargePointErrorCode.
func (v ChargePointErrorCode) IsValid() bool {
	switch v {
	case ChargePointErrorCodeConnectorLockFailure, ChargePointErrorCodeEVCommunicationError, ChargePointErrorCodeGroundFailure, ChargePointErrorCodeHighTemperature, ChargePointErrorCodeInternalError, ChargePointErrorCodeLocalListConflict, ChargePointErrorCodeNoError, ChargePointErrorCodeOtherError, ChargePointErrorCodeOverCurrentFailure, ChargePointErrorCodePowerMeterFailure, ChargePointErrorCodePowerSwitchFailure, ChargePointErrorCodeReaderFailure, ChargePointErrorCodeResetFailure, ChargePointErrorCodeUnderVoltage, ChargePointErrorCodeOverVoltage, ChargePointErrorCodeWeakSignal:
		return true
	}
	return false
}

// ChargePointStatus is a string enumeration. The status of a Charge Point or connector.
type ChargePointStatus string

const (
	// ChargePointStatusAvailable is the "Available" value of ChargePointStatus.
	ChargePointStatusAvailable ChargePointStatus = "Available"
	// ChargePointStatusPreparing is the "Preparing" value of ChargePointStatus.
	ChargePointStatusPreparing ChargePointStatus = "Preparing"
	// ChargePointStatusCharging is the "Charging" value of ChargePointStatus.
	ChargePointStatusCharging ChargePointStatus = "Charging"
	// ChargePointStatusSuspendedEVSE is the "SuspendedEVSE" value of ChargePointStatus.
	ChargePointStatusSuspendedEVSE ChargePointStatus = "SuspendedEVSE"
	// ChargePointStatusSuspendedEV is the "SuspendedEV" value of ChargePointStatus.
	ChargePointStatusSuspendedEV ChargePointStatus = "SuspendedEV"
	// ChargePointStatusFinishing is the "Finishing" value of ChargePointStatus.
	ChargePointStatusFinishing ChargePointStatus = "Finishing"
	// ChargePointStatusReserved is the "Reserved" value of ChargePointStatus.
	ChargePointStatusReserved ChargePointStatus = "Reserved"
	// ChargePointStatusUnavailable is the "Unavailable" value of ChargePointStatus.
	ChargePointStatusUnavailable ChargePointStatus = "Unavailable"
	// ChargePointStatusFaulted is the "Faulted" value of ChargePointStatus.
	ChargePointStatusFaulted ChargePointStatus = "Faulted"
)

// IsValid reports whether v is one of the values defined for ChargePointStatus.
func (v ChargePointStatus) IsValid() bool {
	switch v {
	case ChargePointStatusAvailable, ChargePointStatusPreparing, ChargePointStatusCharging, ChargePointStatusSuspendedEVSE, ChargePointStatusSuspendedEV, ChargePointStatusFinishing, ChargePointStatusReserved, ChargePointStatusUnavailable, ChargePointStatusFaulted:
		return true
	}
	return false
}

// Reason is a string enumeration. Why a transaction was stopped.
type Reason string

const (
	// ReasonEmergencyStop is the "EmergencyStop" value of Reason.
	ReasonEmergencyStop Reason = "EmergencyStop"
	// ReasonEVDisconnected is the "EVDisconnected" value of Reason.
	ReasonEVDisconnected Reason = "EVDisconnected"
	// ReasonHardReset is the "HardReset" value of Reason.
	ReasonHardReset Reason = "HardReset"
	// ReasonLocal is the "Local" value of Reason.
	ReasonLocal Reason = "Local"
	// ReasonOther is the "Other" value of Reason.
	ReasonOther Reason = "Other"
	// ReasonPowerLoss is the "PowerLoss" value of Reason.
	ReasonPowerLoss Reason = "PowerLoss"
	// ReasonReboot is the "Reboot" value of Reason.
	ReasonReboot Reason = "Reboot"
	// ReasonRemote is the "Remote" value of Reason.
	ReasonRemote Reason = "Remote"
	// ReasonSoftReset is the "SoftReset" value of Reason.
	ReasonSoftReset Reason = "SoftReset"
	// ReasonUnlockCommand is the "UnlockCommand" value of Reason.
	ReasonUnlockCommand Reason = "UnlockCommand"
	// ReasonDeAuthorized is the "DeAuthorized" value of Reason.
	ReasonDeAuthorized Reason = "DeAuthorized"
)

// IsValid reports whether v is one of the values defined for Reason.
func (v Reason) IsValid() bool {
	switch v {
	case ReasonEmergencyStop, ReasonEVDisconnected, ReasonHardReset, ReasonLocal, ReasonOther, ReasonPowerLoss, ReasonReboot, ReasonRemote, ReasonSoftReset, ReasonUnlockCommand, ReasonDeAuthorized:
		return true
	}
	return false
}

// ReadingContext is a string enumeration. The reason a meter value was taken.
type ReadingContext string

const (
	// ReadingContextInterruptionBegin is the "Interruption.Begin" value of ReadingContext.
	ReadingContextInterruptionBegin ReadingContext = "Interruption.Begin"
	// ReadingContextInterruptionEnd is the "Interruption.End" value of ReadingContext.
	ReadingContextInterruptionEnd ReadingContext = "Interruption.End"
	// ReadingContextSampleClock is the "Sample.Clock" value of ReadingContext.
	ReadingContextSampleClock ReadingContext = "Sample.Clock"
	// ReadingContextSamplePeriodic is the "Sample.Periodic" value of ReadingContext.
	ReadingContextSamplePeriodic ReadingContext = "Sample.Periodic"
	// ReadingContextTransactionBegin is the "Transaction.Begin" value of ReadingContext.
	ReadingContextTransactionBegin ReadingContext = "Transaction.Begin"
	// ReadingContextTransactionEnd is the "Transaction.End" value of ReadingContext.
	ReadingContextTransactionEnd ReadingContext = "Transaction.End"
	// ReadingContextTrigger is the "Trigger" value of ReadingContext.
	ReadingContextTrigger ReadingContext = "Trigger"
	// ReadingContextOther is the "Other" value of ReadingContext.
	ReadingContextOther ReadingContext = "Other"
)

// IsValid reports whether v is one of the values defined for ReadingContext.
func (v ReadingContext) IsValid() bool {
	switch v {
	case ReadingContextInterruptionBegin, ReadingContextInterruptionEnd, ReadingContextSampleClock, ReadingContextSamplePeriodic, ReadingContextTransactionBegin, ReadingContextTransactionEnd, ReadingContextTrigger, ReadingContextOther:
		return true
	}
	return false
}

// ValueFormat is a string enumeration. Whether a sampled value is a raw number or signed data.
type ValueFormat string

const (
	// ValueFormatRaw is the "Raw" value of ValueFormat.
	ValueFormatRaw ValueFormat = "Raw"
	// ValueFormatSignedData is the "SignedData" value of ValueFormat.
	ValueFormatSignedData ValueFormat = "SignedData"
)

// IsValid reports whether v is one of the values defined for ValueFormat.
func (v ValueFormat) IsValid() bool {
	switch v {
	case ValueFormatRaw, ValueFormatSignedData:
		return true
	}
	return false
}

// Measurand is a string enumeration. The quantity a sampled value measures.
type Measurand string

const (
	// MeasurandEnergyActiveExportRegister is the "Energy.Active.Export.Register" value of Measurand.
	MeasurandEnergyActiveExportRegister Measurand = "Energy.Active.Export.Register"
	// MeasurandEnergyActiveImportRegister is the "Energy.Active.Import.Register" value of Measurand.
	MeasurandEnergyActiveImportRegister Measurand = "Energy.Active.Import.Register"
	// MeasurandEnergyReactiveExportRegister is the "Energy.Reactive.Export.Register" value of Measurand.
	MeasurandEnergyReactiveExportRegister Measurand = "Energy.Reactive.Export.Register"
	// MeasurandEnergyReactiveImportRegister is the "Energy.Reactive.Import.Register" value of Measurand.
	MeasurandEnergyReactiveImportRegister Measurand = "Energy.Reactive.Import.Register"
	// MeasurandEnergyActiveExportInterval is the "Energy.Active.Export.Interval" value of Measurand.
	MeasurandEnergyActiveExportInterval Measurand = "Energy.Active.Export.Interval"
	// MeasurandEnergyActiveImportInterval is the "Energy.Active.Import.Interval" value of Measurand.
	MeasurandEnergyActiveImportInterval Measurand = "Energy.Active.Import.Interval"
	// MeasurandEnergyReactiveExportInterval is the "Energy.Reactive.Export.Interval" value of Measurand.
	MeasurandEnergyReactiveExportInterval Measurand = "Energy.Reactive.Export.Interval"
	// MeasurandEnergyReactiveImportInterval is the "Energy.Reactive.Import.Interval" value of Measurand.
	MeasurandEnergyReactiveImportInterval Measurand = "Energy.Reactive.Import.Interval"
	// MeasurandPowerActiveExport is the "Power.Active.Export" value of Measurand.
	MeasurandPowerActiveExport Measurand = "Power.Active.Export"
	// MeasurandPowerActiveImport is the "Power.Active.Import" value of Measurand.
	MeasurandPowerActiveImport Measurand = "Power.Active.Import"
	// MeasurandPowerOffered is the "Power.Offered" value of Measurand.
	MeasurandPowerOffered Measurand = "Power.Offered"
	// MeasurandPowerReactiveExport is the "Power.Reactive.Export" value of Measurand.
	MeasurandPowerReactiveExport Measurand = "Power.Reactive.Export"
	// MeasurandPowerReactiveImport is the "Power.Reactive.Import" value of Measurand.
	MeasurandPowerReactiveImport Measurand = "Power.Reactive.Import"
	// MeasurandPowerFactor is the "Power.Factor" value of Measurand.
	MeasurandPowerFactor Measurand = "Power.Factor"
	// MeasurandCurrentImport is the "Current.Import" value of Measurand.
	MeasurandCurrentImport Measurand = "Current.Import"
	// MeasurandCurrentExport is the "Current.Export" value of Measurand.
	MeasurandCurrentExport Measurand = "Current.Export"
	// MeasurandCurrentOffered is the "Current.Offered" value of Measurand.
	MeasurandCurrentOffered Measurand = "Current.Offered"
	// MeasurandVoltage is the "Voltage" value of Measurand.
	MeasurandVoltage Measurand = "Voltage"
	// MeasurandFrequency is the "Frequency" value of Measurand.
	MeasurandFrequency Measurand = "Frequency"
	// MeasurandTemperature is the "Temperature" value of Measurand.
	MeasurandTemperature Measurand = "Temperature"
	// MeasurandSoC is the "SoC" value of Measurand.
	MeasurandSoC Measurand = "SoC"
	// MeasurandRPM is the "RPM" value of Measurand.
	MeasurandRPM Measurand = "RPM"
)

// IsValid reports whether v is one of the values defined for Measurand.
func (v Measurand) IsValid() bool {
	switch v {
	case MeasurandEnergyActiveExportRegister, MeasurandEnergyActiveImportRegister, MeasurandEnergyReactiveExportRegister, MeasurandEnergyReactiveImportRegister, MeasurandEnergyActiveExportInterval, MeasurandEnergyActiveImportInterval, MeasurandEnergyReactiveExportInterval, MeasurandEnergyReactiveImportInterval, MeasurandPowerActiveExport, MeasurandPowerActiveImport, MeasurandPowerOffered, MeasurandPowerReactiveExport, MeasurandPowerReactiveImport, MeasurandPowerFactor, MeasurandCurrentImport, MeasurandCurrentExport, MeasurandCurrentOffered, MeasurandVoltage, MeasurandFrequency, MeasurandTemperature, MeasurandSoC, MeasurandRPM:
		return true
	}
	return false
}

// Phase is a string enumeration. The phase a sampled value applies to.
type Phase string

const (
	// PhaseL1 is the "L1" value of Phase.
	PhaseL1 Phase = "L1"
	// PhaseL2 is the "L2" value of Phase.
	PhaseL2 Phase = "L2"
	// PhaseL3 is the "L3" value of Phase.
	PhaseL3 Phase = "L3"
	// PhaseN is the "N" value of Phase.
	PhaseN Phase = "N"
	// PhaseL1N is the "L1-N" value of Phase.
	PhaseL1N Phase = "L1-N"
	// PhaseL2N is the "L2-N" value of Phase.
	PhaseL2N Phase = "L2-N"
	// PhaseL3N is the "L3-N" value of Phase.
	PhaseL3N Phase = "L3-N"
	// PhaseL1L2 is the "L1-L2" value of Phase.
	PhaseL1L2 Phase = "L1-L2"
	// PhaseL2L3 is the "L2-L3" value of Phase.
	PhaseL2L3 Phase = "L2-L3"
	// PhaseL3L1 is the "L3-L1" value of Phase.
	PhaseL3L1 Phase = "L3-L1"
)

// IsValid reports whether v is one of the values defined for Phase.
func (v Phase) IsValid() bool {
	switch v {
	case PhaseL1, PhaseL2, PhaseL3, PhaseN, PhaseL1N, PhaseL2N, PhaseL3N, PhaseL1L2, PhaseL2L3, PhaseL3L1:
		return true
	}
	return false
}

// Location is a string enumeration. Where a sampled value was measured.
type Location string

const (
	// LocationCable is the "Cable" value of Location.
	LocationCable Location = "Cable"
	// LocationEV is the "EV" value of Location.
	LocationEV Location = "EV"
	// LocationInlet is the "Inlet" value of Location.
	LocationInlet Location = "Inlet"
	// LocationOutlet is the "Outlet" value of Location.
	LocationOutlet Location = "Outlet"
	// LocationBody is the "Body" value of Location.
	LocationBody Location = "Body"
)

// IsValid reports whether v is one of the values defined for Location.
func (v Location) IsValid() bool {
	switch v {
	case LocationCable, LocationEV, LocationInlet, LocationOutlet, LocationBody:
		return true
	}
	return false
}

// UnitOfMeasure is a string enumeration. The unit of a sampled value.
type UnitOfMeasure string

const (
	// UnitOfMeasureWh is the "Wh" value of UnitOfMeasure.
	UnitOfMeasureWh UnitOfMeasure = "Wh"
	// UnitOfMeasureKWh is the "kWh" value of UnitOfMeasure.
	UnitOfMeasureKWh UnitOfMeasure = "kWh"
	// UnitOfMeasureVarh is the "varh" value of UnitOfMeasure.
	UnitOfMeasureVarh UnitOfMeasure = "varh"
	// UnitOfMeasureKvarh is the "kvarh" value of UnitOfMeasure.
	UnitOfMeasureKvarh UnitOfMeasure = "kvarh"
	// UnitOfMeasureW is the "W" value of UnitOfMeasure.
	UnitOfMeasureW UnitOfMeasure = "W"
	// UnitOfMeasureKW is the "kW" value of UnitOfMeasure.
	UnitOfMeasureKW UnitOfMeasure = "kW"
	// UnitOfMeasureVA is the "VA" value of UnitOfMeasure.
	UnitOfMeasureVA UnitOfMeasure = "VA"
	// UnitOfMeasureKVA is the "kVA" value of UnitOfMeasure.
	UnitOfMeasureKVA UnitOfMeasure = "kVA"
	// UnitOfMeasureVar is the "var" value of UnitOfMeasure.
	UnitOfMeasureVar UnitOfMeasure = "var"
	// UnitOfMeasureKvar is the "kvar" value of UnitOfMeasure.
	UnitOfMeasureKvar UnitOfMeasure = "kvar"
	// UnitOfMeasureA is the "A" value of UnitOfMeasure.
	UnitOfMeasureA UnitOfMeasure = "A"
	// UnitOfMeasureV is the "V" value of UnitOfMeasure.
	UnitOfMeasureV UnitOfMeasure = "V"
	// UnitOfMeasureK is the "K" value of UnitOfMeasure.
	UnitOfMeasureK UnitOfMeasure = "K"
	// UnitOfMeasureCelcius is the "Celcius" value of UnitOfMeasure.
	UnitOfMeasureCelcius UnitOfMeasure = "Celcius"
	// UnitOfMeasureCelsius is the "Celsius" value of UnitOfMeasure.
	UnitOfMeasureCelsius UnitOfMeasure = "Celsius"
	// UnitOfMeasureFahrenheit is the "Fahrenheit" value of UnitOfMeasure.
	UnitOfMeasureFahrenheit UnitOfMeasure = "Fahrenheit"
	// UnitOfMeasurePercent is the "Percent" value of UnitOfMeasure.
	UnitOfMeasurePercent UnitOfMeasure = "Percent"
)

// IsValid reports whether v is one of the values defined for UnitOfMeasure.
func (v UnitOfMeasure) IsValid() bool {
	switch v {
	case UnitOfMeasureWh, UnitOfMeasureKWh, UnitOfMeasureVarh, UnitOfMeasureKvarh, UnitOfMeasureW, UnitOfMeasureKW, UnitOfMeasureVA, UnitOfMeasureKVA, UnitOfMeasureVar, UnitOfMeasureKvar, UnitOfMeasureA, UnitOfMeasureV, UnitOfMeasureK, UnitOfMeasureCelcius, UnitOfMeasureCelsius, UnitOfMeasureFahrenheit, UnitOfMeasurePercent:
		return true
	}
	return false
}

// ChargingProfilePurposeType is a string enumeration. What a charging profile is for.
type ChargingProfilePurposeType string

const (
	// ChargingProfilePurposeTypeChargePointMaxProfile is the "ChargePointMaxProfile" value of ChargingProfilePurposeType.
	ChargingProfilePurposeTypeChargePointMaxProfile ChargingProfilePurposeType = "ChargePointMaxProfile"
	// ChargingProfilePurposeTypeTxDefaultProfile is the "TxDefaultProfile" value of ChargingProfilePurposeType.
	ChargingProfilePurposeTypeTxDefaultProfile ChargingProfilePurposeType = "TxDefaultProfile"
	// ChargingProfilePurposeTypeTxProfile is the "TxProfile" value of ChargingProfilePurposeType.
	ChargingProfilePurposeTypeTxProfile ChargingProfilePurposeType = "TxProfile"
)

// IsValid reports whether v is one of the values defined for ChargingProfilePurposeType.
func (v ChargingProfilePurposeType) IsValid() bool {
	switch v {
	case ChargingProfilePurposeTypeChargePointMaxProfile, ChargingProfilePurposeTypeTxDefaultProfile, ChargingProfilePurposeTypeTxProfile:
		return true
	}
	return false
}

// ChargingProfileKindType is a string enumeration. Whether a charging profile is absolute, recurring or relative.
type ChargingProfileKindType string

const (
	// ChargingProfileKindTypeAbsolute is the "Absolute" value of ChargingProfileKindType.
	ChargingProfileKindTypeAbsolute ChargingProfileKindType = "Absolute"
	// ChargingProfileKindTypeRecurring is the "Recurring" value of ChargingProfileKindType.
	ChargingProfileKindTypeRecurring ChargingProfileKindType = "Recurring"
	// ChargingProfileKindTypeRelative is the "Relative" value of ChargingProfileKindType.
	ChargingProfileKindTypeRelative ChargingProfileKindType = "Relative"
)

// IsValid reports whether v is one of the values defined for ChargingProfileKindType.
func (v ChargingProfileKindType) IsValid() bool {
	switch v {
	case ChargingProfileKindTypeAbsolute, ChargingProfileKindTypeRecurring, ChargingProfileKindTypeRelative:
		return true
	}
	return false
}

// RecurrencyKindType is a string enumeration. When a recurring charging profile repeats.
type RecurrencyKindType string

const (
	// RecurrencyKindTypeDaily is the "Daily" value of RecurrencyKindType.
	RecurrencyKindTypeDaily RecurrencyKindType = "Daily"
	// RecurrencyKindTypeWeekly is the "Weekly" value of RecurrencyKindType.
	RecurrencyKindTypeWeekly RecurrencyKindType = "Weekly"
)

// IsValid reports whether v is one of the values defined for RecurrencyKindType.
func (v RecurrencyKindType) IsValid() bool {
	switch v {
	case RecurrencyKindTypeDaily, RecurrencyKindTypeWeekly:
		return true
	}
	return false
}

// ChargingRateUnitType is a string enumeration. The unit a charging schedule limit is expressed in.
type ChargingRateUnitType string

const (
	// ChargingRateUnitTypeA is the "A" value of ChargingRateUnitType.
	ChargingRateUnitTypeA ChargingRateUnitType = "A"
	// ChargingRateUnitTypeW is the "W" value of ChargingRateUnitType.
	ChargingRateUnitTypeW ChargingRateUnitType = "W"
)

// IsValid reports whether v is one of the values defined for ChargingRateUnitType.
func (v ChargingRateUnitType) IsValid() bool {
	switch v {
	case ChargingRateUnitTypeA, ChargingRateUnitTypeW:
		return true
	}
	return false
}
