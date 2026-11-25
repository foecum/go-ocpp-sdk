package ocpp

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Sentinel errors callers can branch on with errors.Is.
var (
	// ErrTimeout means no response arrived in time.
	ErrTimeout = errors.New("ocpp: request timed out")
	// ErrNotConnected means there is no connection to send on.
	ErrNotConnected = errors.New("ocpp: not connected")
	// ErrValidation means a message failed validation.
	ErrValidation = errors.New("ocpp: validation failed")
	// ErrCallError means the peer answered with a CALLERROR.
	ErrCallError = errors.New("ocpp: received CALLERROR")
	// ErrQueueFull means too many calls are already waiting to be sent.
	ErrQueueFull = errors.New("ocpp: call queue full")
	// ErrClosed means the endpoint or connection was closed.
	ErrClosed = errors.New("ocpp: closed")
	// ErrUnknownAction means no feature is registered for the action.
	ErrUnknownAction = errors.New("ocpp: unknown action")
)

// ValidationError describes a field that failed validation. It matches
// ErrValidation with errors.Is.
type ValidationError struct {
	// Field is the JSON name of the offending field.
	Field string
	// Reason says what is wrong with it.
	Reason string
	// Missing is true when the field is required but absent, which maps to an
	// occurrence-constraint violation rather than a property-constraint one.
	Missing bool
}

// Error implements error.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("ocpp: invalid field %q: %s", e.Field, e.Reason)
}

// Is reports whether target is ErrValidation.
func (e *ValidationError) Is(target error) bool { return target == ErrValidation }

// Required returns a ValidationError for a missing required field.
func Required(field string) error {
	return &ValidationError{Field: field, Reason: "required", Missing: true}
}

// Invalid returns a ValidationError for a field holding a bad value.
func Invalid(field, reason string) error {
	return &ValidationError{Field: field, Reason: reason}
}

// CallError is a CALLERROR received from the peer. It matches ErrCallError
// with errors.Is.
type CallError struct {
	// Code is the errorCode field.
	Code ErrorCode
	// Description is the errorDescription field.
	Description string
	// Details is the raw errorDetails object.
	Details json.RawMessage
}

// Error implements error.
func (e *CallError) Error() string {
	return fmt.Sprintf("ocpp: CALLERROR %s: %s", e.Code, e.Description)
}

// Is reports whether target is ErrCallError.
func (e *CallError) Is(target error) bool { return target == ErrCallError }
