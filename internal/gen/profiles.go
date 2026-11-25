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
	OneWay: map[string]bool{"NotifyPeriodicEventStream": true},
}
