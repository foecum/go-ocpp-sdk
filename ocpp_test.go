package ocpp_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

type pingReq struct{ Message string }

func (pingReq) Action() string  { return "Ping" }
func (pingReq) Validate() error { return nil }

type pingResp struct{}

func (*pingResp) Action() string  { return "Ping" }
func (*pingResp) Validate() error { return nil }

type otherResp struct{}

func (otherResp) Action() string  { return "Other" }
func (otherResp) Validate() error { return nil }

type noActionReq struct{}

func (noActionReq) Action() string  { return "" }
func (noActionReq) Validate() error { return nil }

func TestVersion(t *testing.T) {
	for _, v := range []ocpp.Version{ocpp.V16, ocpp.V201, ocpp.V21} {
		if !v.IsValid() {
			t.Errorf("%q should be valid", v)
		}
		if v.Subprotocol() != string(v) {
			t.Errorf("subprotocol mismatch for %q", v)
		}
	}
	if ocpp.Version("ocpp9").IsValid() {
		t.Error("unknown version reported valid")
	}
	if ocpp.V16 != "ocpp1.6" || ocpp.V201 != "ocpp2.0.1" || ocpp.V21 != "ocpp2.1" {
		t.Error("subprotocol names changed")
	}
}

func TestErrorCodeSets(t *testing.T) {
	tests := []struct {
		code ocpp.ErrorCode
		v    ocpp.Version
		want bool
	}{
		{ocpp.FormationViolation, ocpp.V16, true},
		{ocpp.FormatViolation, ocpp.V16, false},
		{ocpp.OccurenceConstraintViolation, ocpp.V16, true},
		{ocpp.OccurrenceConstraintViolation, ocpp.V16, false},
		{ocpp.RpcFrameworkError, ocpp.V16, false},
		{ocpp.MessageTypeNotSupported, ocpp.V16, false},
		{ocpp.NotImplemented, ocpp.V16, true},
		{ocpp.GenericError, ocpp.V16, true},
		{ocpp.FormatViolation, ocpp.V201, true},
		{ocpp.FormationViolation, ocpp.V201, false},
		{ocpp.OccurrenceConstraintViolation, ocpp.V201, true},
		{ocpp.OccurenceConstraintViolation, ocpp.V201, false},
		{ocpp.RpcFrameworkError, ocpp.V201, true},
		{ocpp.MessageTypeNotSupported, ocpp.V21, true},
		{ocpp.SecurityError, ocpp.V21, true},
		{ocpp.ErrorCode("Bogus"), ocpp.V201, false},
		{ocpp.NotSupported, ocpp.Version("x"), false},
	}
	for _, tt := range tests {
		if got := tt.code.IsValid(tt.v); got != tt.want {
			t.Errorf("%s.IsValid(%s) = %v, want %v", tt.code, tt.v, got, tt.want)
		}
	}
	all16 := []ocpp.ErrorCode{ocpp.NotImplemented, ocpp.NotSupported, ocpp.InternalError, ocpp.ProtocolError,
		ocpp.SecurityError, ocpp.FormationViolation, ocpp.PropertyConstraintViolation,
		ocpp.OccurenceConstraintViolation, ocpp.TypeConstraintViolation, ocpp.GenericError}
	for _, c := range all16 {
		if !c.IsValid(ocpp.V16) {
			t.Errorf("%s should be valid in 1.6", c)
		}
	}
}

func TestViolationCodeSpelling(t *testing.T) {
	if ocpp.FormatViolationCode(ocpp.V16) != ocpp.FormationViolation ||
		ocpp.FormatViolationCode(ocpp.V201) != ocpp.FormatViolation {
		t.Error("format violation spelling")
	}
	if ocpp.OccurrenceViolationCode(ocpp.V16) != ocpp.OccurenceConstraintViolation ||
		ocpp.OccurrenceViolationCode(ocpp.V201) != ocpp.OccurrenceConstraintViolation {
		t.Error("occurrence violation spelling")
	}
}

func TestErrors(t *testing.T) {
	ve := ocpp.Required("idTag")
	if !errors.Is(ve, ocpp.ErrValidation) || errors.Is(ve, ocpp.ErrTimeout) {
		t.Error("ValidationError should match only ErrValidation")
	}
	var v *ocpp.ValidationError
	if !errors.As(fmt.Errorf("wrap: %w", ve), &v) || !v.Missing || v.Field != "idTag" {
		t.Errorf("unexpected %+v", v)
	}
	bad := ocpp.Invalid("status", "unknown value")
	if errors.As(bad, &v); v.Missing || v.Reason != "unknown value" {
		t.Errorf("unexpected %+v", v)
	}
	if ve.Error() == "" {
		t.Error("empty message")
	}

	ce := &ocpp.CallError{Code: ocpp.NotImplemented, Description: "nope"}
	if !errors.Is(ce, ocpp.ErrCallError) || errors.Is(ce, ocpp.ErrValidation) {
		t.Error("CallError should match only ErrCallError")
	}
	if ce.Error() != "ocpp: CALLERROR NotImplemented: nope" {
		t.Errorf("got %q", ce.Error())
	}
}

func TestNewFeature(t *testing.T) {
	f, err := ocpp.NewFeature(pingReq{}, &pingResp{})
	if err != nil {
		t.Fatal(err)
	}
	if f.Action() != "Ping" ||
		f.RequestType() != reflect.TypeOf(pingReq{}) ||
		f.ResponseType() != reflect.TypeOf(pingResp{}) {
		t.Errorf("unexpected feature %v %v %v", f.Action(), f.RequestType(), f.ResponseType())
	}
	f, err = ocpp.NewFeature(&pingReq{}, pingRespValue{})
	if err != nil || f.RequestType() != reflect.TypeOf(pingReq{}) {
		t.Errorf("pointer sample: %v %v", f, err)
	}

	for name, args := range map[string][2]any{
		"nil request":  {nil, &pingResp{}},
		"nil response": {pingReq{}, nil},
		"empty action": {noActionReq{}, noActionReq{}},
		"mismatch":     {pingReq{}, otherResp{}},
	} {
		var req ocpp.Request
		var resp ocpp.Response
		if args[0] != nil {
			req = args[0].(ocpp.Request)
		}
		if args[1] != nil {
			resp = args[1].(ocpp.Response)
		}
		if _, err := ocpp.NewFeature(req, resp); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

type pingRespValue struct{}

func (pingRespValue) Action() string  { return "Ping" }
func (pingRespValue) Validate() error { return nil }

func TestDateTimeMarshal(t *testing.T) {
	base := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	tests := []struct {
		name string
		in   time.Time
		want string
	}{
		{"whole seconds", base, `"2026-01-02T03:04:05Z"`},
		{"milliseconds", base.Add(123 * time.Millisecond), `"2026-01-02T03:04:05.123Z"`},
		{"sub-millisecond is truncated", base.Add(123456789 * time.Nanosecond), `"2026-01-02T03:04:05.123Z"`},
		{"converted to UTC", base.In(time.FixedZone("CET", 3600)), `"2026-01-02T03:04:05Z"`},
	}
	for _, tt := range tests {
		got, err := json.Marshal(ocpp.NewDateTime(tt.in))
		if err != nil || string(got) != tt.want {
			t.Errorf("%s: got %s (%v), want %s", tt.name, got, err, tt.want)
		}
	}
}

func TestDateTimeUnmarshal(t *testing.T) {
	want := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	good := map[string]time.Time{
		`"2026-01-02T03:04:05Z"`:        want,
		`"2026-01-02T03:04:05.5Z"`:      want.Add(500 * time.Millisecond),
		`"2026-01-02T03:04:05.123456Z"`: want.Add(123456 * time.Microsecond),
		`"2026-01-02T04:04:05+01:00"`:   want,
		`"2026-01-02T03:04:05"`:         want, // no zone: UTC
		`"2026-01-02T03:04:05.250"`:     want.Add(250 * time.Millisecond),
	}
	for in, w := range good {
		var d ocpp.DateTime
		if err := json.Unmarshal([]byte(in), &d); err != nil || !d.Equal(w) {
			t.Errorf("%s: got %v (%v), want %v", in, d.Time, err, w)
		}
	}
	for _, in := range []string{`"yesterday"`, `""`, `"2026-01-02"`, `12345`, `true`, `{}`} {
		var d ocpp.DateTime
		if err := json.Unmarshal([]byte(in), &d); err == nil {
			t.Errorf("%s: expected an error", in)
		}
	}
	d := ocpp.NewDateTime(want)
	if err := json.Unmarshal([]byte(`null`), &d); err != nil || !d.Equal(want) {
		t.Errorf("null: %v %v", d, err)
	}
	if p := ocpp.NewDateTime(want).Ptr(); !p.Equal(want) {
		t.Error("Ptr")
	}
	if time.Since(ocpp.Now().Time) > time.Minute {
		t.Error("Now is not now")
	}
}

func TestNewSendFeature(t *testing.T) {
	f, err := ocpp.NewSendFeature(&pingReq{})
	if err != nil {
		t.Fatal(err)
	}
	if f.Action() != "Ping" || f.RequestType() != reflect.TypeOf(pingReq{}) || f.ResponseType() != nil || !ocpp.IsOneWay(f) {
		t.Errorf("unexpected one-way feature %v %v %v", f.Action(), f.RequestType(), f.ResponseType())
	}
	if _, err := ocpp.NewSendFeature(nil); err == nil {
		t.Error("nil request: expected an error")
	}
	if _, err := ocpp.NewSendFeature(noActionReq{}); err == nil {
		t.Error("empty action: expected an error")
	}
	two, _ := ocpp.NewFeature(pingReq{}, &pingResp{})
	if ocpp.IsOneWay(two) {
		t.Error("a request/response feature is not one-way")
	}
}

type foreignFeature struct{ ocpp.Feature }

func TestIsOneWayIgnoresForeignFeatures(t *testing.T) {
	if ocpp.IsOneWay(foreignFeature{}) {
		t.Error("a feature without the OneWay method is not one-way")
	}
}
