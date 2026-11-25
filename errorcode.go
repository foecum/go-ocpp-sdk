package ocpp

// ErrorCode is the errorCode field of a CALLERROR message.
//
// The code sets differ between versions, including spelling: 1.6 uses
// FormationViolation and OccurenceConstraintViolation (sic), while 2.0.1 and
// 2.1 use FormatViolation and OccurrenceConstraintViolation.
type ErrorCode string

// CALLERROR codes.
const (
	// NotImplemented means the requested action is not known by the receiver.
	NotImplemented ErrorCode = "NotImplemented"
	// NotSupported means the action is recognized but not supported.
	NotSupported ErrorCode = "NotSupported"
	// InternalError means an internal error prevented processing the action.
	InternalError ErrorCode = "InternalError"
	// ProtocolError means the payload does not conform to the PDU structure.
	ProtocolError ErrorCode = "ProtocolError"
	// SecurityError means a security issue prevented completing the action.
	SecurityError ErrorCode = "SecurityError"
	// FormationViolation is the 1.6 spelling of a syntactically incorrect payload.
	FormationViolation ErrorCode = "FormationViolation"
	// FormatViolation is the 2.0.1 and 2.1 spelling of a syntactically incorrect payload.
	FormatViolation ErrorCode = "FormatViolation"
	// PropertyConstraintViolation means a field contains an invalid value.
	PropertyConstraintViolation ErrorCode = "PropertyConstraintViolation"
	// OccurenceConstraintViolation is the 1.6 spelling (sic) of an occurrence violation.
	OccurenceConstraintViolation ErrorCode = "OccurenceConstraintViolation"
	// OccurrenceConstraintViolation is the 2.0.1 and 2.1 spelling of an occurrence violation.
	OccurrenceConstraintViolation ErrorCode = "OccurrenceConstraintViolation"
	// TypeConstraintViolation means a field violates data type constraints.
	TypeConstraintViolation ErrorCode = "TypeConstraintViolation"
	// GenericError is any other error not covered by a more specific code.
	GenericError ErrorCode = "GenericError"
	// MessageTypeNotSupported means the message type number is not supported (2.0.1 and 2.1).
	MessageTypeNotSupported ErrorCode = "MessageTypeNotSupported"
	// RpcFrameworkError means the content is not a valid RPC request, for
	// example the message ID could not be read (2.0.1 and 2.1).
	RpcFrameworkError ErrorCode = "RpcFrameworkError"
)

var codes16 = map[ErrorCode]bool{
	NotImplemented: true, NotSupported: true, InternalError: true,
	ProtocolError: true, SecurityError: true, FormationViolation: true,
	PropertyConstraintViolation: true, OccurenceConstraintViolation: true,
	TypeConstraintViolation: true, GenericError: true,
}

var codes201 = map[ErrorCode]bool{
	FormatViolation: true, GenericError: true, InternalError: true,
	MessageTypeNotSupported: true, NotImplemented: true, NotSupported: true,
	OccurrenceConstraintViolation: true, PropertyConstraintViolation: true,
	ProtocolError: true, RpcFrameworkError: true, SecurityError: true,
	TypeConstraintViolation: true,
}

// IsValid reports whether c is an allowed CALLERROR code in version v.
// OCPP 2.1 uses the same set as 2.0.1.
func (c ErrorCode) IsValid(v Version) bool {
	switch v {
	case V16:
		return codes16[c]
	case V201, V21:
		return codes201[c]
	}
	return false
}

// FormatViolationCode returns the version's code for a syntactically
// incorrect payload.
func FormatViolationCode(v Version) ErrorCode {
	if v == V16 {
		return FormationViolation
	}
	return FormatViolation
}

// OccurrenceViolationCode returns the version's code for a violated
// occurrence (required field missing) constraint.
func OccurrenceViolationCode(v Version) ErrorCode {
	if v == V16 {
		return OccurenceConstraintViolation
	}
	return OccurrenceConstraintViolation
}
