package ocppj

import (
	"errors"
	"testing"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

func TestRegistryRegister(t *testing.T) {
	r := newRegistry(t, ocpp.V16)
	if r.Version() != ocpp.V16 || r.Len() != 2 {
		t.Errorf("version=%s len=%d", r.Version(), r.Len())
	}
	f, _ := r.Lookup("Ping")
	if f == nil || f.Action() != "Ping" {
		t.Fatal("Ping not found")
	}
	if _, ok := r.Lookup("Nope"); ok {
		t.Error("unexpected feature")
	}
	if err := r.Register(f); err == nil {
		t.Error("duplicate registration should fail")
	}
	if err := r.Register(nil); err == nil {
		t.Error("nil feature should fail")
	}
	if err := r.Register(emptyAction{}); err == nil {
		t.Error("empty action should fail")
	}
}

type emptyAction struct{ ocpp.Feature }

func (emptyAction) Action() string { return "" }

func TestDecodeRequest(t *testing.T) {
	tests := []struct {
		name     string
		v        ocpp.Version
		action   string
		payload  string
		wantCode ocpp.ErrorCode // empty means success
		unknown  bool
	}{
		{"ok", ocpp.V201, "Ping", `{"message":"hi","count":2}`, "", false},
		{"unknown fields ignored", ocpp.V201, "Ping", `{"message":"hi","extra":1}`, "", false},
		{"unknown action", ocpp.V201, "Nope", `{}`, "", true},
		{"missing required 2.0.1", ocpp.V201, "Ping", `{}`, ocpp.OccurrenceConstraintViolation, false},
		{"missing required 1.6", ocpp.V16, "Ping", `{}`, ocpp.OccurenceConstraintViolation, false},
		{"invalid value", ocpp.V201, "Ping", `{"message":"way too long for this"}`, ocpp.PropertyConstraintViolation, false},
		{"wrong type", ocpp.V201, "Ping", `{"message":12}`, ocpp.TypeConstraintViolation, false},
		{"syntax 2.0.1", ocpp.V201, "Ping", `{"message":`, ocpp.FormatViolation, false},
		{"syntax 1.6", ocpp.V16, "Ping", `{"message":`, ocpp.FormationViolation, false},
	}
	for _, tt := range tests {
		r := newRegistry(t, tt.v)
		req, err := r.DecodeRequest(tt.action, []byte(tt.payload))
		switch {
		case tt.unknown:
			if !errors.Is(err, ocpp.ErrUnknownAction) {
				t.Errorf("%s: %v", tt.name, err)
			}
		case tt.wantCode == "":
			if err != nil || req.(*pingReq).Message != "hi" {
				t.Errorf("%s: %v %v", tt.name, req, err)
			}
		default:
			var de *DecodeError
			if !errors.As(err, &de) || de.Code != tt.wantCode || errors.Unwrap(de) != de.Err || de.Error() == "" {
				t.Errorf("%s: got %v", tt.name, err)
			}
		}
	}
	// Validation failures also match ocpp.ErrValidation.
	r := newRegistry(t, ocpp.V201)
	if _, err := r.DecodeRequest("Ping", []byte(`{}`)); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("want ErrValidation, got %v", err)
	}
}

func TestDecodeResponse(t *testing.T) {
	r := newRegistry(t, ocpp.V201)
	resp, err := r.DecodeResponse("Ping", []byte(`{"reply":"pong"}`))
	if err != nil || resp.(*pingResp).Reply != "pong" {
		t.Fatalf("%v %v", resp, err)
	}
	if _, err := r.DecodeResponse("Nope", []byte(`{}`)); !errors.Is(err, ocpp.ErrUnknownAction) {
		t.Errorf("unknown: %v", err)
	}
	var de *DecodeError
	if _, err := r.DecodeResponse("Ping", []byte(`{"reply":"bad"}`)); !errors.As(err, &de) || de.Code != ocpp.PropertyConstraintViolation {
		t.Errorf("invalid: %v", err)
	}
}
