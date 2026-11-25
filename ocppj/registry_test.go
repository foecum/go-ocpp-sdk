package ocppj

import (
	"errors"
	"reflect"
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

type reqNested struct {
	Count int             `json:"count" ocpp:"required"`
	Mode  string          `json:"mode,omitempty"`
	Items []reqItem       `json:"items,omitempty"`
	One   *reqItem        `json:"one,omitempty"`
	Skip  int             `json:"-" ocpp:"required"`
	Plain int             // no json tag: key is the field name
	Embed struct{ X int } `json:"embed"`
}

type reqItem struct {
	At    int `json:"at" ocpp:"required"`
	Value int `json:"value,omitempty"`
}

func (reqNested) Action() string  { return "Nested" }
func (reqNested) Validate() error { return nil }

type respNested struct{}

func (respNested) Action() string  { return "Nested" }
func (respNested) Validate() error { return nil }

func TestRequiredKeysAtAnyDepth(t *testing.T) {
	f, err := ocpp.NewFeature(reqNested{}, respNested{})
	if err != nil {
		t.Fatal(err)
	}
	r := NewRegistry(ocpp.V201)
	if err := r.Register(f); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, payload string
		wantField     string // empty: success
	}{
		{"ok", `{"count":0}`, ""},
		{"ok nested", `{"count":1,"items":[{"at":0},{"at":5,"value":2}],"one":{"at":1}}`, ""},
		{"missing count", `{}`, "count"},
		{"null count", `{"count":null}`, "count"},
		{"missing in slice item", `{"count":1,"items":[{"at":0},{"value":2}]}`, "items[1].at"},
		{"missing in pointer", `{"count":1,"one":{}}`, "one.at"},
		{"null optional is fine", `{"count":1,"one":null,"items":null}`, ""},
		{"non-object slice ignored here", `{"count":1,"items":5}`, "~type"},
	}
	for _, tt := range tests {
		_, err := r.DecodeRequest("Nested", []byte(tt.payload))
		switch {
		case tt.wantField == "":
			if err != nil {
				t.Errorf("%s: %v", tt.name, err)
			}
		case tt.wantField == "~type":
			var de *DecodeError
			if !errors.As(err, &de) || de.Code != ocpp.TypeConstraintViolation {
				t.Errorf("%s: %v", tt.name, err)
			}
		default:
			var de *DecodeError
			var ve *ocpp.ValidationError
			if !errors.As(err, &de) || !errors.As(err, &ve) || !ve.Missing ||
				de.Code != ocpp.OccurrenceConstraintViolation || ve.Field != tt.wantField {
				t.Errorf("%s: got %v", tt.name, err)
			}
		}
	}
	// A payload that is not an object is left to the decoder to reject.
	if err := checkRequired(reflect.TypeOf(reqNested{}), []byte(`[1]`), ""); err != nil {
		t.Errorf("non-object: %v", err)
	}
	if err := checkRequired(reflect.TypeOf(&reqNested{}), []byte(`{"count":1}`), ""); err != nil {
		t.Errorf("pointer type: %v", err)
	}
}
