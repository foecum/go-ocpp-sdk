package gen

// Profile describes one OCPP version the generator produces code for.
type Profile struct {
	// Name is the version as the specification writes it ("2.0.1", "2.1").
	Name string
	// Dir is the output directory and Go package name of the version (v201, v21).
	Dir string
	// SchemaRel is the directory of the official JSON schemas, relative to the
	// repository root.
	SchemaRel string
	// Const is the Go expression of the version constant ("ocpp.V201").
	Const string
	// Blocks lists the functional blocks and the messages of each. Together
	// they must cover every message that has a JSON schema.
	Blocks []Block
	// Directions gives the sender of the request of every message.
	Directions map[string]Direction
	// HandlerNames gives, for each block package, the name used in the
	// high-level APIs: the handler interface is "<Name>Handler" and the Config
	// field is "<Name>".
	HandlerNames map[string]string
	// Rules gives, by schema definition or message name, extra validation
	// expressions for constraints that the specification text states but the
	// JSON schemas do not. Each is a Go expression returning an error, in which
	// v is the value and {T} stands for the qualifier of the shared types package.
	Rules map[string][]string
	// AppendicesRel is the directory of the appendices CSV files next to the
	// schemas directory; if empty no appendices package is generated.
	AppendicesRel string
	// Extra lists value lists kept in CSV files that become constants.
	Extra []extraSet
	// OneWay lists the messages that are sent as unconfirmed SEND frames: their
	// schema is "<Name>.json" and they have no response.
	OneWay map[string]bool
}

// directionsOf builds a direction table: the messages in fromStation are sent
// by the Charging Station, DataTransfer by either side, every other message of
// the blocks by the CSMS.
func directionsOf(blocks []Block, fromStation []string) map[string]Direction {
	d := map[string]Direction{"DataTransfer": Both}
	for _, m := range fromStation {
		d[m] = FromStation
	}
	for _, b := range blocks {
		for _, m := range b.Messages {
			if _, ok := d[m]; !ok {
				d[m] = FromCSMS
			}
		}
	}
	return d
}

var blocks201 = []Block{
	{"security", "A", "Security", []string{"CertificateSigned", "SecurityEventNotification", "SignCertificate"}},
	{"provisioning", "B", "Provisioning", []string{"BootNotification", "GetBaseReport", "GetReport", "GetVariables", "NotifyReport", "Reset", "SetNetworkProfile", "SetVariables"}},
	{"authorization", "C", "Authorization", []string{"Authorize", "ClearCache"}},
	{"localauthlist", "D", "Local Authorization List", []string{"GetLocalListVersion", "SendLocalList"}},
	{"transactions", "E", "Transactions", []string{"GetTransactionStatus", "TransactionEvent"}},
	{"remotecontrol", "F", "Remote Control", []string{"RequestStartTransaction", "RequestStopTransaction", "TriggerMessage", "UnlockConnector"}},
	{"availability", "G", "Availability", []string{"ChangeAvailability", "Heartbeat", "StatusNotification"}},
	{"reservation", "H", "Reservation", []string{"CancelReservation", "ReservationStatusUpdate", "ReserveNow"}},
	{"tariffandcost", "I", "Tariff and Cost", []string{"CostUpdated"}},
	{"metervalues", "J", "Meter Values", []string{"MeterValues"}},
	{"smartcharging", "K", "Smart Charging", []string{"ClearChargingProfile", "ClearedChargingLimit", "GetChargingProfiles", "GetCompositeSchedule", "NotifyChargingLimit", "NotifyEVChargingNeeds", "NotifyEVChargingSchedule", "ReportChargingProfiles", "SetChargingProfile"}},
	{"firmware", "L", "Firmware Management", []string{"FirmwareStatusNotification", "PublishFirmware", "PublishFirmwareStatusNotification", "UnpublishFirmware", "UpdateFirmware"}},
	{"iso15118", "M", "ISO 15118 Certificate Management", []string{"DeleteCertificate", "Get15118EVCertificate", "GetCertificateStatus", "GetInstalledCertificateIds", "InstallCertificate"}},
	{"diagnostics", "N", "Diagnostics", []string{"ClearVariableMonitoring", "CustomerInformation", "GetLog", "GetMonitoringReport", "LogStatusNotification", "NotifyCustomerInformation", "NotifyEvent", "NotifyMonitoringReport", "SetMonitoringBase", "SetMonitoringLevel", "SetVariableMonitoring"}},
	{"displaymessage", "O", "Display Message", []string{"ClearDisplayMessage", "GetDisplayMessages", "NotifyDisplayMessages", "SetDisplayMessage"}},
	{"datatransfer", "P", "Data Transfer", []string{"DataTransfer"}},
}

// stationMessages201 are the 2.0.1 messages whose request the Charging Station
// sends, as stated in OCPP 2.0.1 part 2.
var stationMessages201 = []string{
	"Authorize", "BootNotification", "ClearedChargingLimit", "FirmwareStatusNotification", "Get15118EVCertificate",
	"GetCertificateStatus", "Heartbeat", "LogStatusNotification", "MeterValues", "NotifyChargingLimit",
	"NotifyCustomerInformation", "NotifyDisplayMessages", "NotifyEVChargingNeeds", "NotifyEVChargingSchedule",
	"NotifyEvent", "NotifyMonitoringReport", "NotifyReport", "PublishFirmwareStatusNotification",
	"ReportChargingProfiles", "ReservationStatusUpdate", "SecurityEventNotification", "SignCertificate",
	"StatusNotification", "TransactionEvent",
}

var handlerNames201 = map[string]string{
	"security": "Security", "provisioning": "Provisioning", "authorization": "Authorization",
	"localauthlist": "LocalAuthList", "transactions": "Transactions", "remotecontrol": "RemoteControl",
	"availability": "Availability", "reservation": "Reservation", "tariffandcost": "TariffAndCost",
	"metervalues": "MeterValues", "smartcharging": "SmartCharging", "firmware": "Firmware",
	"iso15118": "ISO15118", "diagnostics": "Diagnostics", "displaymessage": "DisplayMessage",
	"datatransfer": "DataTransfer",
}

// V201 is the OCPP 2.0.1 profile.
var V201 = Profile{
	Name: "2.0.1", Dir: "v201", SchemaRel: "docs/spec/2.0.1/schemas", Const: "ocpp.V201",
	Blocks:       blocks201,
	Directions:   directionsOf(blocks201, stationMessages201),
	HandlerNames: handlerNames201,
	Rules:        rules201,

	AppendicesRel: "Appendices_CSV_v1.5",
}

// blocks21 are the 2.0.1 blocks plus the messages OCPP 2.1 adds to them (in
// the order the messages are listed in the block's functional requirements) and
// the three blocks it adds: Q Bidirectional Power Transfer, R DER Control and
// S Battery Swapping.
var blocks21 = func() []Block {
	extra := map[string][]string{
		"authorization": {"NotifySettlement", "NotifyWebPaymentStarted", "VatNumberValidation"},
		"tariffandcost": {"ChangeTransactionTariff", "ClearTariffs", "GetTariffs", "SetDefaultTariff"},
		"smartcharging": {"NotifyPriorityCharging", "PullDynamicScheduleUpdate", "UpdateDynamicSchedule", "UsePriorityCharging"},
		"iso15118":      {"GetCertificateChainStatus"},
		"diagnostics":   {"AdjustPeriodicEventStream", "ClosePeriodicEventStream", "GetPeriodicEventStream", "NotifyPeriodicEventStream", "OpenPeriodicEventStream"},
	}
	var out []Block
	for _, b := range blocks201 {
		b.Messages = append(append([]string(nil), b.Messages...), extra[b.Pkg]...)
		if b.Pkg == "iso15118" {
			b.Title = "Certificate Management"
		}
		out = append(out, b)
	}
	return append(out,
		Block{"bidirectional", "Q", "Bidirectional Power Transfer", []string{"AFRRSignal", "NotifyAllowedEnergyTransfer"}},
		Block{"dercontrol", "R", "DER Control", []string{"ClearDERControl", "GetDERControl", "NotifyDERAlarm", "NotifyDERStartStop", "ReportDERControl", "SetDERControl"}},
		Block{"batteryswap", "S", "Battery Swapping", []string{"BatterySwap", "RequestBatterySwap"}},
	)
}()

// V21 is the OCPP 2.1 profile. The new messages' senders are stated in the
// scenarios of OCPP 2.1 part 2.
var V21 = Profile{
	Name: "2.1", Dir: "v21", SchemaRel: "docs/spec/2.1/schemas", Const: "ocpp.V21",
	Blocks: blocks21,
	Directions: directionsOf(blocks21, append(append([]string(nil), stationMessages201...),
		"BatterySwap", "ClosePeriodicEventStream", "GetCertificateChainStatus", "NotifyDERAlarm", "NotifyDERStartStop",
		"NotifyPeriodicEventStream", "NotifyPriorityCharging", "NotifySettlement", "OpenPeriodicEventStream",
		"PullDynamicScheduleUpdate", "ReportDERControl", "VatNumberValidation")),
	HandlerNames: func() map[string]string {
		m := map[string]string{"bidirectional": "Bidirectional", "dercontrol": "DERControl", "batteryswap": "BatterySwap"}
		for k, v := range handlerNames201 {
			m[k] = v
		}
		return m
	}(),
	Rules:  rules21,
	OneWay: map[string]bool{"NotifyPeriodicEventStream": true},

	AppendicesRel: "Appendices_CSV_v2.1",
	Extra: []extraSet{
		{"idtokenenumtype.csv", "IDTokenType", "IDTokenTypes", "idToken type"},
		{"connectorenumtype.csv", "ConnectorType", "ConnectorTypes", "connector type"},
		{"charginglimitsourceenumtype.csv", "ChargingLimitSource", "ChargingLimitSources", "charging limit source"},
		{"paymentbrand.csv", "PaymentBrand", "PaymentBrands", "payment brand"},
		{"paymentrecognition.csv", "PaymentRecognition", "PaymentRecognitions", "payment recognition"},
		{"additional_info_types.csv", "AdditionalInfoType", "AdditionalInfoTypes", "additionalInfo type"},
		{"additional_info_types_adhoc.csv", "AdditionalInfoAdHocType", "AdditionalInfoAdHocTypes", "additionalInfo type for ad hoc payments"},
		{"signingmethod.csv", "SigningMethod", "SigningMethods", "signing method"},
	},
}

// rules201 are the constraints OCPP 2.0.1 part 2 states in its field tables and
// requirements but that the JSON schemas do not, where the text is unambiguous.
var rules201 = map[string][]string{
	"SendLocalListRequest": {
		// "Optional. Required when UpdateType is Full."
		`check.EachFn("localAuthorizationList", v.LocalAuthorizationList, func(a {T}AuthorizationData) error {
			return check.Needed(v.UpdateType == {T}UpdateFull && a.IDTokenInfo == nil, "idTokenInfo")
		})`,
	},
	"NotifyChargingLimitRequest":      {`check.Rule(v.EVSEID != nil && *v.EVSEID <= 0, "evseId", "must be greater than 0")`},
	"NotifyEVChargingScheduleRequest": {`check.Rule(v.EVSEID <= 0, "evseId", "must be greater than 0")`},
	"RequestStartTransactionRequest": {
		`check.Rule(v.EVSEID != nil && *v.EVSEID <= 0, "evseId", "must be greater than 0")`,
		`check.Rule(v.ChargingProfile != nil && v.ChargingProfile.ChargingProfilePurpose != {T}ChargingProfilePurposeTxProfile, "chargingProfile.chargingProfilePurpose", "must be TxProfile")`,
	},
	"ChargingProfileType": {
		`check.Rule(v.TransactionID != "" && v.ChargingProfilePurpose != {T}ChargingProfilePurposeTxProfile, "transactionId", "only allowed when chargingProfilePurpose is TxProfile")`,
	},
	"VariableCharacteristicsType": {
		// "Mandatory when dataType = OptionList, MemberList or SequenceList."
		`check.Needed(v.ValuesList == "" && (v.DataType == {T}DataOptionList || v.DataType == {T}DataMemberList || v.DataType == {T}DataSequenceList), "valuesList")`,
	},
	"PublishFirmwareStatusNotificationRequest": {
		`check.Needed(v.Status == {T}PublishFirmwareStatusPublished && len(v.Location) == 0, "location")`,
	},
	"ChargingSchedulePeriodType": {
		// "Accepts at most one digit fraction (e.g. 8.1)." and "Values: 1..3".
		`check.Decimals("limit", v.Limit, 1)`,
		`check.IntPtr("phaseToUse", v.PhaseToUse, check.Ptr(1), check.Ptr(3))`,
	},
	"SetMonitoringLevelRequest": {`check.Int("severity", v.Severity, check.Ptr(0), check.Ptr(9))`},
}

// rules21 are the rules of OCPP 2.1 part 2 and its errata 2026-09 that the JSON
// schemas do not state. They carry over those of 2.0.1 where the text is the
// same, and add the ones of the new data types.
var rules21 = func() map[string][]string {
	out := map[string][]string{}
	for k, v := range rules201 {
		out[k] = v
	}
	// In 2.1 limit is optional (a period may carry a setpoint instead).
	out["ChargingSchedulePeriodType"] = []string{
		`check.DecimalsPtr("limit", v.Limit, 1)`,
		`check.IntPtr("phaseToUse", v.PhaseToUse, check.Ptr(1), check.Ptr(3))`,
		// "Only required when operationMode = LocalFrequency. When used it must
		// contain at least two coordinates." Likewise for the signal curve.
		`check.Rule(len(v.V2XFreqWattCurve) == 1 || (v.OperationMode == {T}OperationModeLocalFrequency && len(v.V2XFreqWattCurve) == 0), "v2xFreqWattCurve", "must contain at least two coordinates when operationMode is LocalFrequency")`,
		`check.Rule(len(v.V2XSignalWattCurve) == 1, "v2xSignalWattCurve", "must contain at least two coordinates")`,
	}
	out["ChargingProfileType"] = append(append([]string(nil), rules201["ChargingProfileType"]...),
		// randomizedDelay: "Only allowed for TxProfile and TxDefaultProfile."
		`check.EachFn("chargingSchedule", v.ChargingSchedule, func(s {T}ChargingSchedule) error {
			return check.Rule(s.RandomizedDelay != nil && *s.RandomizedDelay != 0 &&
				v.ChargingProfilePurpose != {T}ChargingProfilePurposeTxProfile && v.ChargingProfilePurpose != {T}ChargingProfilePurposeTxDefaultProfile,
				"randomizedDelay", "only allowed for TxProfile and TxDefaultProfile")
		})`)
	// Errata 2026-09, 2.28: pctMaxDischargePower is required, as a percentage.
	out["LimitMaxDischargeType"] = []string{
		`check.Needed(v.PctMaxDischargePower == nil, "pctMaxDischargePower")`,
		`check.NumberPtr("pctMaxDischargePower", v.PctMaxDischargePower, check.Ptr(0.0), check.Ptr(100.0))`,
	}
	// Errata 2026-09, 2.30: PowerMonitoringMustTrip "is not a valid DER control";
	// it only exists as part of a LimitMaxDischarge control.
	notStandalone := `check.Rule(v.ControlType == {T}DERControlPowerMonitoringMustTrip, "controlType", "PowerMonitoringMustTrip is not a stand-alone DER control")`
	for _, m := range []string{"SetDERControlRequest", "GetDERControlRequest", "ClearDERControlRequest", "NotifyDERAlarmRequest"} {
		out[m] = []string{notStandalone}
	}
	// "This value is mandatory if hv10MinMeanTripDelay is set" and the reverse.
	out["VoltageParamsType"] = []string{
		`check.Needed(v.Hv10MinMeanTripDelay != nil && v.Hv10MinMeanValue == nil, "hv10MinMeanValue")`,
		`check.Needed(v.Hv10MinMeanValue != nil && v.Hv10MinMeanTripDelay == nil, "hv10MinMeanTripDelay")`,
	}
	// "Must be in 24h format with leading zeros."
	// "Format as per RFC 3339: full-date", written YYYY-MM-DD.
	conditions := []string{
		`check.TimeOfDay("startTimeOfDay", v.StartTimeOfDay)`, `check.TimeOfDay("endTimeOfDay", v.EndTimeOfDay)`,
		`check.Date("validFromDate", v.ValidFromDate)`, `check.Date("validToDate", v.ValidToDate)`,
	}
	out["TariffConditionsType"] = conditions
	out["TariffConditionsFixedType"] = conditions
	return out
}()
