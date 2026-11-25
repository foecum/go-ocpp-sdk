package ocppj

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

// DecodeError reports a payload that could not be decoded or validated. It
// carries the CALLERROR code to answer with.
type DecodeError struct {
	// Code is the CALLERROR code to reply with.
	Code ocpp.ErrorCode
	// Err is the underlying cause.
	Err error
}

// Error implements error.
func (e *DecodeError) Error() string {
	return fmt.Sprintf("ocppj: decode payload (%s): %v", e.Code, e.Err)
}

// Unwrap returns the underlying cause.
func (e *DecodeError) Unwrap() error { return e.Err }

// Registry maps action names to Features for one OCPP version. It is safe for
// concurrent use.
type Registry struct {
	version  ocpp.Version
	mu       sync.RWMutex
	features map[string]ocpp.Feature
}

// NewRegistry creates an empty registry for version v.
func NewRegistry(v ocpp.Version) *Registry {
	return &Registry{version: v, features: make(map[string]ocpp.Feature)}
}

// Version returns the registry's OCPP version.
func (r *Registry) Version() ocpp.Version { return r.version }

// Register adds a feature. It fails on an empty or duplicate action.
func (r *Registry) Register(f ocpp.Feature) error {
	if f == nil || f.Action() == "" {
		return errors.New("ocppj: feature needs an action")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, dup := r.features[f.Action()]; dup {
		return fmt.Errorf("ocppj: action %q already registered", f.Action())
	}
	r.features[f.Action()] = f
	return nil
}

// Lookup returns the feature registered for action.
func (r *Registry) Lookup(action string) (ocpp.Feature, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.features[action]
	return f, ok
}

// Len returns the number of registered actions.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.features)
}

// DecodeRequest decodes and validates the payload of a CALL. An unregistered
// action yields an error matching ocpp.ErrUnknownAction; other failures are
// *DecodeError.
func (r *Registry) DecodeRequest(action string, payload json.RawMessage) (ocpp.Request, error) {
	f, ok := r.Lookup(action)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ocpp.ErrUnknownAction, action)
	}
	v, err := r.decode(f.RequestType(), payload)
	if err != nil {
		return nil, err
	}
	return v.(ocpp.Request), nil
}

// DecodeResponse decodes and validates the payload of a CALLRESULT to a CALL
// with the given action.
func (r *Registry) DecodeResponse(action string, payload json.RawMessage) (ocpp.Response, error) {
	f, ok := r.Lookup(action)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ocpp.ErrUnknownAction, action)
	}
	v, err := r.decode(f.ResponseType(), payload)
	if err != nil {
		return nil, err
	}
	return v.(ocpp.Response), nil
}

type validator interface{ Validate() error }

func (r *Registry) decode(t reflect.Type, payload json.RawMessage) (any, error) {
	ptr := reflect.New(t).Interface()
	if err := json.Unmarshal(payload, ptr); err != nil {
		code := ocpp.FormatViolationCode(r.version)
		var ute *json.UnmarshalTypeError
		if errors.As(err, &ute) {
			code = ocpp.TypeConstraintViolation
		}
		return nil, &DecodeError{Code: code, Err: err}
	}
	if err := checkRequired(t, payload, ""); err != nil {
		return nil, &DecodeError{Code: r.validationCode(err), Err: err}
	}
	if err := ptr.(validator).Validate(); err != nil {
		return nil, &DecodeError{Code: r.validationCode(err), Err: err}
	}
	return ptr, nil
}

func (r *Registry) validationCode(err error) ocpp.ErrorCode {
	var ve *ocpp.ValidationError
	if errors.As(err, &ve) && ve.Missing {
		return ocpp.OccurrenceViolationCode(r.version)
	}
	return ocpp.PropertyConstraintViolation
}
