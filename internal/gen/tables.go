package gen

// Block is one OCPP functional block, which becomes one Go package.
type Block struct {
	// Pkg is the Go package name under the profile's directory (v201/, v21/).
	Pkg string
	// Letter is the block's letter in the specification.
	Letter string
	// Title is the block's name in the specification.
	Title string
	// Messages lists the block's messages (actions).
	Messages []string
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
