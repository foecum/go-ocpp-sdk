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
	resp   reflect.Type // nil for a one-way feature
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

// OneWayFeature is implemented by the Feature of a message that is sent as an
// OCPP-J SEND frame (OCPP 2.1): it has no response, so its ResponseType is nil.
type OneWayFeature interface {
	Feature
	// OneWay reports that the message is unconfirmed.
	OneWay() bool
}

func (f feature) OneWay() bool { return f.resp == nil }

// NewSendFeature builds the Feature of an unconfirmed message (OCPP 2.1 SEND,
// for example NotifyPeriodicEventStream). req may be a zero value or a pointer
// to one. The feature's ResponseType is nil.
func NewSendFeature(req Request) (Feature, error) {
	if req == nil {
		return nil, errors.New("ocpp: NewSendFeature needs a request")
	}
	if req.Action() == "" {
		return nil, errors.New("ocpp: empty action")
	}
	return feature{action: req.Action(), req: structType(req)}, nil
}

// IsOneWay reports whether f is the feature of an unconfirmed (SEND) message.
func IsOneWay(f Feature) bool {
	o, ok := f.(OneWayFeature)
	return ok && o.OneWay()
}

func structType(v any) reflect.Type {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}
