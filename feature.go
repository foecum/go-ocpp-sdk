package ocpp

import (
	"errors"
	"reflect"
)

// Request is an OCPP request payload (the payload of a CALL).
type Request interface {
	// Action returns the OCPP action name, without the "Request" suffix.
	Action() string
	// Validate checks required fields, lengths and enum values. It returns a
	// *ValidationError on failure.
	Validate() error
}

// Response is an OCPP response payload (the payload of a CALLRESULT).
type Response interface {
	// Action returns the action name of the request this answers.
	Action() string
	// Validate checks required fields, lengths and enum values.
	Validate() error
}

// Feature binds an action name to its request and response types.
type Feature interface {
	// Action returns the action name.
	Action() string
	// RequestType returns the (non-pointer) request struct type.
	RequestType() reflect.Type
	// ResponseType returns the (non-pointer) response struct type.
	ResponseType() reflect.Type
}

type feature struct {
	action string
	req    reflect.Type
	resp   reflect.Type
}

func (f feature) Action() string             { return f.action }
func (f feature) RequestType() reflect.Type  { return f.req }
func (f feature) ResponseType() reflect.Type { return f.resp }

// NewFeature builds a Feature from sample request and response values, which
// may be zero values or pointers to them. Both must report the same action.
func NewFeature(req Request, resp Response) (Feature, error) {
	if req == nil || resp == nil {
		return nil, errors.New("ocpp: NewFeature needs a request and a response")
	}
	if req.Action() == "" {
		return nil, errors.New("ocpp: empty action")
	}
	if req.Action() != resp.Action() {
		return nil, errors.New("ocpp: request and response actions differ")
	}
	return feature{action: req.Action(), req: structType(req), resp: structType(resp)}, nil
}

func structType(v any) reflect.Type {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}
