package ocpp

// Version identifies an OCPP protocol version. Its value is the WebSocket
// subprotocol name used to negotiate that version.
type Version string

// Supported protocol versions.
const (
	// V16 is OCPP 1.6 (JSON).
	V16 Version = "ocpp1.6"
	// V201 is OCPP 2.0.1.
	V201 Version = "ocpp2.0.1"
	// V21 is OCPP 2.1.
	V21 Version = "ocpp2.1"
)

// IsValid reports whether v is a known protocol version.
func (v Version) IsValid() bool {
	switch v {
	case V16, V201, V21:
		return true
	}
	return false
}

// Subprotocol returns the WebSocket subprotocol name for v.
func (v Version) Subprotocol() string { return string(v) }
