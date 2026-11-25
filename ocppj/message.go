package ocppj

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

// MaxIDLength is the maximum length of a message ID.
const MaxIDLength = 36

// MaxDescriptionLength is the maximum length of a CALLERROR description.
const MaxDescriptionLength = 255

// UnknownID is the message ID used in a CALLERROR when the ID of the
// offending message could not be read.
const UnknownID = "-1"

// MessageType is the message type number, the first element of a frame.
type MessageType int

// Message type numbers.
const (
	// CallType is a request.
	CallType MessageType = 2
	// CallResultType is a successful response.
	CallResultType MessageType = 3
	// CallErrorType is an error response.
	CallErrorType MessageType = 4
	// CallResultErrorType is an error answering a CALLRESULT (OCPP 2.1 only).
	CallResultErrorType MessageType = 5
	// SendType is an unconfirmed message without a response (OCPP 2.1 only).
	SendType MessageType = 6
)

// Message is one OCPP-J frame: a Call, CallResult, CallErrorFrame and, in OCPP
// 2.1, a Send or CallResultErrorFrame.
type Message interface {
	// Type returns the message type number.
	Type() MessageType
	// ID returns the unique message ID.
	ID() string
}

// Call is a request frame: [2, id, action, payload].
type Call struct {
	UniqueID string
	Action   string
	Payload  json.RawMessage
}

// CallResult is a response frame: [3, id, payload].
type CallResult struct {
	UniqueID string
	Payload  json.RawMessage
}

// CallErrorFrame is an error frame: [4, id, code, description, details].
type CallErrorFrame struct {
	UniqueID    string
	Code        ocpp.ErrorCode
	Description string
	Details     json.RawMessage
}

// Send is an unconfirmed message frame, which OCPP 2.1 adds: [6, id, action,
// payload]. The receiver never answers it.
type Send struct {
	UniqueID string
	Action   string
	Payload  json.RawMessage
}

// CallResultErrorFrame is the frame a receiver of a CALLRESULT sends when it
// cannot process it, which OCPP 2.1 adds: [5, id, code, description, details].
// Its ID is that of the CALLRESULT it answers.
type CallResultErrorFrame struct {
	UniqueID    string
	Code        ocpp.ErrorCode
	Description string
	Details     json.RawMessage
}

// Type implements Message.
func (Send) Type() MessageType { return SendType }

// Type implements Message.
func (CallResultErrorFrame) Type() MessageType { return CallResultErrorType }

// ID implements Message.
func (m Send) ID() string { return m.UniqueID }

// ID implements Message.
func (m CallResultErrorFrame) ID() string { return m.UniqueID }

// Type implements Message.
func (Call) Type() MessageType { return CallType }

// Type implements Message.
func (CallResult) Type() MessageType { return CallResultType }

// Type implements Message.
func (CallErrorFrame) Type() MessageType { return CallErrorType }

// ID implements Message.
func (m Call) ID() string { return m.UniqueID }

// ID implements Message.
func (m CallResult) ID() string { return m.UniqueID }

// ID implements Message.
func (m CallErrorFrame) ID() string { return m.UniqueID }

// ParseError reports a frame that could not be parsed. It carries what is
// needed to answer the sender with a CALLERROR.
type ParseError struct {
	// Type is the message type if it could be read, otherwise zero.
	Type MessageType
	// ID is the message ID to answer with, or UnknownID if it was unreadable.
	ID string
	// Code is the CALLERROR code to reply with. It is empty when the spec says
	// the frame must simply be ignored.
	Code ocpp.ErrorCode
	// Err is the underlying cause.
	Err error

	// silent is set for malformed frames that must never be answered: responses,
	// and in OCPP 2.1 also SEND.
	silent bool
}

// Error implements error.
func (e *ParseError) Error() string { return "ocppj: invalid frame: " + e.Err.Error() }

// Unwrap returns the underlying cause.
func (e *ParseError) Unwrap() error { return e.Err }

// ShouldReply reports whether the sender should be answered with a CALLERROR.
// Malformed responses are never answered.
func (e *ParseError) ShouldReply() bool { return e.Code != "" && !e.silent }

// quiet reports whether a malformed frame of type mt is never answered: a
// response is not, and neither is a SEND or CALLRESULTERROR in OCPP 2.1.
func quiet(v ocpp.Version, mt MessageType) bool {
	switch mt {
	case CallResultType, CallErrorType:
		return true
	case CallResultErrorType, SendType:
		return v == ocpp.V21
	}
	return false
}

// Parse decodes one frame received on a connection speaking version v. The
// message types SEND and CALLRESULTERROR exist only in OCPP 2.1; for other
// versions they are unknown message types.
//
// A payload of null is accepted and treated as an empty object. On failure the
// error is a *ParseError.
func Parse(v ocpp.Version, data []byte) (Message, error) {
	var elems []json.RawMessage
	if err := json.Unmarshal(data, &elems); err != nil {
		return nil, unreadable(v, fmt.Errorf("not a JSON array: %w", err))
	}
	if len(elems) < 2 {
		return nil, unreadable(v, errors.New("frame needs at least a message type and an ID"))
	}
	var mt MessageType
	if err := json.Unmarshal(elems[0], &mt); err != nil {
		return nil, unreadable(v, fmt.Errorf("message type: %w", err))
	}
	var id string
	if err := json.Unmarshal(elems[1], &id); err != nil || id == "" || len(id) > MaxIDLength {
		return nil, &ParseError{Type: mt, ID: UnknownID, Code: unreadableCode(v), Err: errors.New("message ID missing, not a string or too long"), silent: quiet(v, mt)}
	}
	fail := func(err error) (Message, error) {
		return nil, &ParseError{Type: mt, ID: id, Code: ocpp.ProtocolError, Err: err, silent: quiet(v, mt)}
	}
	switch mt {
	case CallType:
		if len(elems) != 4 {
			return fail(fmt.Errorf("CALL needs 4 elements, got %d", len(elems)))
		}
		var action string
		if err := json.Unmarshal(elems[2], &action); err != nil || action == "" {
			return fail(errors.New("CALL action missing or not a string"))
		}
		payload, err := object(elems[3])
		if err != nil {
			return fail(err)
		}
		return &Call{UniqueID: id, Action: action, Payload: payload}, nil
	case SendType:
		if v != ocpp.V21 {
			break
		}
		if len(elems) != 4 {
			return fail(fmt.Errorf("SEND needs 4 elements, got %d", len(elems)))
		}
		var action string
		if err := json.Unmarshal(elems[2], &action); err != nil || action == "" {
			return fail(errors.New("SEND action missing or not a string"))
		}
		payload, err := object(elems[3])
		if err != nil {
			return fail(err)
		}
		return &Send{UniqueID: id, Action: action, Payload: payload}, nil
	case CallResultType:
		if len(elems) != 3 {
			return fail(fmt.Errorf("CALLRESULT needs 3 elements, got %d", len(elems)))
		}
		payload, err := object(elems[2])
		if err != nil {
			return fail(err)
		}
		return &CallResult{UniqueID: id, Payload: payload}, nil
	case CallErrorType:
		code, desc, details, err := parseError("CALLERROR", elems)
		if err != nil {
			return fail(err)
		}
		return &CallErrorFrame{UniqueID: id, Code: code, Description: desc, Details: details}, nil
	case CallResultErrorType:
		if v != ocpp.V21 {
			break
		}
		code, desc, details, err := parseError("CALLRESULTERROR", elems)
		if err != nil {
			return fail(err)
		}
		return &CallResultErrorFrame{UniqueID: id, Code: code, Description: desc, Details: details}, nil
	}
	pe := &ParseError{Type: mt, ID: id, Err: fmt.Errorf("unknown message type %d", mt)}
	if v == ocpp.V201 {
		// OCPP 2.1 errata 2026-09 (part 4, 4.1 and 4.3): an unknown message type is
		// silently ignored and MessageTypeNotSupported is deprecated.
		pe.Code = ocpp.MessageTypeNotSupported
	}
	return nil, pe
}

// parseError reads the code, description and details of a CALLERROR or
// CALLRESULTERROR frame.
func parseError(kind string, elems []json.RawMessage) (ocpp.ErrorCode, string, json.RawMessage, error) {
	if len(elems) != 5 {
		return "", "", nil, fmt.Errorf("%s needs 5 elements, got %d", kind, len(elems))
	}
	var code, desc string
	if err := json.Unmarshal(elems[2], &code); err != nil || code == "" {
		return "", "", nil, fmt.Errorf("%s code missing or not a string", kind)
	}
	if err := json.Unmarshal(elems[3], &desc); err != nil {
		return "", "", nil, fmt.Errorf("%s description is not a string", kind)
	}
	details, err := object(elems[4])
	if err != nil {
		return "", "", nil, err
	}
	return ocpp.ErrorCode(code), desc, details, nil
}

func unreadableCode(v ocpp.Version) ocpp.ErrorCode {
	if v == ocpp.V16 {
		return ocpp.FormationViolation
	}
	return ocpp.RpcFrameworkError
}

func unreadable(v ocpp.Version, err error) *ParseError {
	return &ParseError{ID: UnknownID, Code: unreadableCode(v), Err: err}
}

// object checks that raw is a JSON object, mapping null to {}.
func object(raw json.RawMessage) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) {
		return json.RawMessage(`{}`), nil
	}
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, errors.New("payload is not a JSON object")
	}
	return trimmed, nil
}

// Marshal encodes a message as a frame. An empty payload becomes {}.
func Marshal(m Message) ([]byte, error) {
	var frame []any
	switch m := m.(type) {
	case *Call:
		if err := checkID(m.UniqueID); err != nil {
			return nil, err
		}
		if m.Action == "" {
			return nil, ocpp.Required("action")
		}
		frame = []any{CallType, m.UniqueID, m.Action, payloadOrEmpty(m.Payload)}
	case *CallResult:
		if err := checkID(m.UniqueID); err != nil {
			return nil, err
		}
		frame = []any{CallResultType, m.UniqueID, payloadOrEmpty(m.Payload)}
	case *CallErrorFrame:
		var err error
		if frame, err = errorFrame(CallErrorType, m.UniqueID, m.Code, m.Description, m.Details); err != nil {
			return nil, err
		}
	case *CallResultErrorFrame:
		var err error
		if frame, err = errorFrame(CallResultErrorType, m.UniqueID, m.Code, m.Description, m.Details); err != nil {
			return nil, err
		}
	case *Send:
		if err := checkID(m.UniqueID); err != nil {
			return nil, err
		}
		if m.Action == "" {
			return nil, ocpp.Required("action")
		}
		frame = []any{SendType, m.UniqueID, m.Action, payloadOrEmpty(m.Payload)}
	default:
		return nil, fmt.Errorf("ocppj: cannot marshal %T", m)
	}
	data, err := json.Marshal(frame)
	if err != nil {
		return nil, fmt.Errorf("ocppj: marshal frame: %w", err)
	}
	return data, nil
}

func errorFrame(t MessageType, id string, code ocpp.ErrorCode, desc string, details json.RawMessage) ([]any, error) {
	if err := checkID(id); err != nil {
		return nil, err
	}
	if code == "" {
		return nil, ocpp.Required("errorCode")
	}
	if utf8.RuneCountInString(desc) > MaxDescriptionLength {
		return nil, ocpp.Invalid("errorDescription", "longer than 255 characters")
	}
	return []any{t, id, string(code), desc, payloadOrEmpty(details)}, nil
}

func checkID(id string) error {
	if id == "" {
		return ocpp.Required("messageId")
	}
	if len(id) > MaxIDLength {
		return ocpp.Invalid("messageId", "longer than 36 characters")
	}
	return nil
}

func payloadOrEmpty(p json.RawMessage) json.RawMessage {
	if len(p) == 0 {
		return json.RawMessage(`{}`)
	}
	return p
}
