package gen201

// Block is one OCPP 2.0.1 functional block, which becomes one Go package.
type Block struct {
	// Pkg is the Go package name under v201/.
	Pkg string
	// Letter is the block's letter in the specification.
	Letter string
	// Title is the block's name in the specification.
	Title string
	// Messages lists the block's messages (actions).
	Messages []string
}

// Blocks lists the functional blocks of OCPP 2.0.1 part 2 and the messages of
// each. Together they cover every message that has a JSON schema.
var Blocks = []Block{
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

// Direction says who sends a message's request.
type Direction int

// Directions.
const (
	// FromStation means the Charging Station sends the request to the CSMS.
	FromStation Direction = iota
	// FromCSMS means the CSMS sends the request to the Charging Station.
	FromCSMS
	// Both means either side may send the request.
	Both
)

// Directions gives the sender of the request of every message, as stated in
// OCPP 2.0.1 part 2.
var Directions = func() map[string]Direction {
	d := map[string]Direction{"DataTransfer": Both}
	for _, m := range []string{
		"Authorize", "BootNotification", "ClearedChargingLimit", "FirmwareStatusNotification", "Get15118EVCertificate",
		"GetCertificateStatus", "Heartbeat", "LogStatusNotification", "MeterValues", "NotifyChargingLimit",
		"NotifyCustomerInformation", "NotifyDisplayMessages", "NotifyEVChargingNeeds", "NotifyEVChargingSchedule",
		"NotifyEvent", "NotifyMonitoringReport", "NotifyReport", "PublishFirmwareStatusNotification",
		"ReportChargingProfiles", "ReservationStatusUpdate", "SecurityEventNotification", "SignCertificate",
		"StatusNotification", "TransactionEvent",
	} {
		d[m] = FromStation
	}
	for _, b := range Blocks {
		for _, m := range b.Messages {
			if _, ok := d[m]; !ok {
				d[m] = FromCSMS
			}
		}
	}
	return d
}()

// initialisms are the words kept in capitals in Go identifiers.
var initialisms = map[string]string{
	"id": "ID", "evse": "EVSE", "ev": "EV", "url": "URL", "iso": "ISO", "ocsp": "OCSP", "ocpp": "OCPP", "csms": "CSMS",
	"apn": "APN", "vpn": "VPN", "iccid": "ICCID", "imsi": "IMSI", "pin": "PIN", "csr": "CSR", "exi": "EXI", "ac": "AC",
	"dc": "DC", "soc": "SoC", "uid": "UID", "tls": "TLS", "dn": "DN", "pdu": "PDU", "ip": "IP", "json": "JSON",
}

// emptyAllowed lists required string fields whose value may be empty ("Type.field").
// For these only the length is checked; presence is checked from the JSON.
var emptyAllowed = map[string]bool{
	"SetVariableDataType.attributeValue": true,
}

// fieldRenames gives Go names to fields whose natural name clashes with a
// method of the generated type ("Type.field").
var fieldRenames = map[string]string{
	// A field named Action would clash with the Action method every message has.
	"Get15118EVCertificateRequest.action": "CertificateAction",
}

// HandlerNames gives, for each block package, the name used in the high-level
// APIs: the handler interface is "<Name>Handler" and the Config field is "<Name>".
var HandlerNames = map[string]string{
	"security": "Security", "provisioning": "Provisioning", "authorization": "Authorization",
	"localauthlist": "LocalAuthList", "transactions": "Transactions", "remotecontrol": "RemoteControl",
	"availability": "Availability", "reservation": "Reservation", "tariffandcost": "TariffAndCost",
	"metervalues": "MeterValues", "smartcharging": "SmartCharging", "firmware": "Firmware",
	"iso15118": "ISO15118", "diagnostics": "Diagnostics", "displaymessage": "DisplayMessage",
	"datatransfer": "DataTransfer",
}
