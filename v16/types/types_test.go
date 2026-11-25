package types_test

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

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
		got, err := json.Marshal(types.NewDateTime(tt.in))
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
		`"2026-01-02T02:34:05-00:30"`:   want,
		`"2026-01-02T03:04:05"`:         want, // no zone: UTC
		`"2026-01-02T03:04:05.250"`:     want.Add(250 * time.Millisecond),
	}
	for in, w := range good {
		var d types.DateTime
		if err := json.Unmarshal([]byte(in), &d); err != nil || !d.Equal(w) {
			t.Errorf("%s: got %v (%v), want %v", in, d.Time, err, w)
		}
	}
	for _, in := range []string{`"yesterday"`, `""`, `"2026-01-02"`, `"2026-13-02T03:04:05Z"`, `12345`, `true`, `{}`} {
		var d types.DateTime
		if err := json.Unmarshal([]byte(in), &d); err == nil {
			t.Errorf("%s: expected an error", in)
		}
	}
	// null leaves the value alone.
	d := types.NewDateTime(want)
	if err := json.Unmarshal([]byte(`null`), &d); err != nil || !d.Equal(want) {
		t.Errorf("null: %v %v", d, err)
	}
	var s struct {
		At *types.DateTime `json:"at"`
	}
	if err := json.Unmarshal([]byte(`{"at":"2026-01-02T03:04:05Z"}`), &s); err != nil || !s.At.Equal(want) {
		t.Errorf("pointer field: %v %v", s.At, err)
	}
	if p := types.NewDateTime(want).Ptr(); !p.Equal(want) {
		t.Error("Ptr")
	}
	if time.Since(types.Now().Time) > time.Minute {
		t.Error("Now is not now")
	}
}

// enumValues reads the allowed values of an enum from the official schema.
func enumValues(t *testing.T, schemaFile string, path ...string) []string {
	t.Helper()
	data, err := os.ReadFile("../../docs/spec/1.6/schemas/json/" + schemaFile + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var node map[string]any
	if err := json.Unmarshal(data, &node); err != nil {
		t.Fatal(err)
	}
	for _, p := range path {
		if items, ok := node["items"].(map[string]any); ok {
			node = items
		}
		node = node["properties"].(map[string]any)[p].(map[string]any)
	}
	var out []string
	for _, v := range node["enum"].([]any) {
		out = append(out, v.(string))
	}
	return out
}

func TestEnumsMatchSchemas(t *testing.T) {
	tests := []struct {
		name   string
		valid  func(string) bool
		schema string
		path   []string
	}{
		{"AuthorizationStatus", func(s string) bool { return types.AuthorizationStatus(s).IsValid() }, "AuthorizeResponse", []string{"idTagInfo", "status"}},
		{"RegistrationStatus", func(s string) bool { return types.RegistrationStatus(s).IsValid() }, "BootNotificationResponse", []string{"status"}},
		{"AvailabilityType", func(s string) bool { return types.AvailabilityType(s).IsValid() }, "ChangeAvailability", []string{"type"}},
		{"AvailabilityStatus", func(s string) bool { return types.AvailabilityStatus(s).IsValid() }, "ChangeAvailabilityResponse", []string{"status"}},
		{"ConfigurationStatus", func(s string) bool { return types.ConfigurationStatus(s).IsValid() }, "ChangeConfigurationResponse", []string{"status"}},
		{"ClearCacheStatus", func(s string) bool { return types.ClearCacheStatus(s).IsValid() }, "ClearCacheResponse", []string{"status"}},
		{"DataTransferStatus", func(s string) bool { return types.DataTransferStatus(s).IsValid() }, "DataTransferResponse", []string{"status"}},
		{"RemoteStartStopStatus", func(s string) bool { return types.RemoteStartStopStatus(s).IsValid() }, "RemoteStopTransactionResponse", []string{"status"}},
		{"ResetType", func(s string) bool { return types.ResetType(s).IsValid() }, "Reset", []string{"type"}},
		{"ResetStatus", func(s string) bool { return types.ResetStatus(s).IsValid() }, "ResetResponse", []string{"status"}},
		{"UnlockStatus", func(s string) bool { return types.UnlockStatus(s).IsValid() }, "UnlockConnectorResponse", []string{"status"}},
		{"ChargePointErrorCode", func(s string) bool { return types.ChargePointErrorCode(s).IsValid() }, "StatusNotification", []string{"errorCode"}},
		{"ChargePointStatus", func(s string) bool { return types.ChargePointStatus(s).IsValid() }, "StatusNotification", []string{"status"}},
		{"Reason", func(s string) bool { return types.Reason(s).IsValid() }, "StopTransaction", []string{"reason"}},
		{"ReadingContext", func(s string) bool { return types.ReadingContext(s).IsValid() }, "StopTransaction", []string{"transactionData", "sampledValue", "context"}},
		{"ValueFormat", func(s string) bool { return types.ValueFormat(s).IsValid() }, "StopTransaction", []string{"transactionData", "sampledValue", "format"}},
		{"Measurand", func(s string) bool { return types.Measurand(s).IsValid() }, "StopTransaction", []string{"transactionData", "sampledValue", "measurand"}},
		{"Phase", func(s string) bool { return types.Phase(s).IsValid() }, "StopTransaction", []string{"transactionData", "sampledValue", "phase"}},
		{"Location", func(s string) bool { return types.Location(s).IsValid() }, "StopTransaction", []string{"transactionData", "sampledValue", "location"}},
		{"UnitOfMeasure", func(s string) bool { return types.UnitOfMeasure(s).IsValid() }, "StopTransaction", []string{"transactionData", "sampledValue", "unit"}},
		{"ChargingProfilePurposeType", func(s string) bool { return types.ChargingProfilePurposeType(s).IsValid() }, "RemoteStartTransaction", []string{"chargingProfile", "chargingProfilePurpose"}},
		{"ChargingProfileKindType", func(s string) bool { return types.ChargingProfileKindType(s).IsValid() }, "RemoteStartTransaction", []string{"chargingProfile", "chargingProfileKind"}},
		{"RecurrencyKindType", func(s string) bool { return types.RecurrencyKindType(s).IsValid() }, "RemoteStartTransaction", []string{"chargingProfile", "recurrencyKind"}},
		{"ChargingRateUnitType", func(s string) bool { return types.ChargingRateUnitType(s).IsValid() }, "RemoteStartTransaction", []string{"chargingProfile", "chargingSchedule", "chargingRateUnit"}},
	}
	for _, tt := range tests {
		values := enumValues(t, tt.schema, tt.path...)
		if len(values) == 0 {
			t.Fatalf("%s: no values read from the schema", tt.name)
		}
		for _, v := range values {
			if !tt.valid(v) {
				t.Errorf("%s: %q should be valid", tt.name, v)
			}
		}
		for _, bad := range []string{"", "Bogus", strings.ToLower(values[0]) + "x"} {
			if tt.valid(bad) {
				t.Errorf("%s: %q should be invalid", tt.name, bad)
			}
		}
	}
	// Both spellings from the OCPP-J 1.6 errata are accepted.
	if !types.UnitOfMeasureCelcius.IsValid() || !types.UnitOfMeasureCelsius.IsValid() {
		t.Error("Celcius and Celsius must both be valid")
	}
}

func validationField(t *testing.T, err error) (string, bool) {
	t.Helper()
	var ve *ocpp.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("not a validation error: %v", err)
	}
	return ve.Field, ve.Missing
}

func TestCheckHelpers(t *testing.T) {
	if err := types.CheckString("f", "abc", 3, true); err != nil {
		t.Error(err)
	}
	if err := types.CheckString("f", "é", 1, true); err != nil {
		t.Errorf("length counts characters, not bytes: %v", err)
	}
	if err := types.CheckString("f", "", 3, false); err != nil {
		t.Error(err)
	}
	if f, missing := validationField(t, types.CheckString("f", "", 3, true)); f != "f" || !missing {
		t.Error("empty required string is missing")
	}
	if f, missing := validationField(t, types.CheckString("f", "abcd", 3, false)); f != "f" || missing {
		t.Error("too long")
	}

	if err := types.CheckEnum("e", types.ResetTypeHard, true); err != nil {
		t.Error(err)
	}
	if err := types.CheckEnum("e", types.ResetType(""), false); err != nil {
		t.Error(err)
	}
	if _, missing := validationField(t, types.CheckEnum("e", types.ResetType(""), true)); !missing {
		t.Error("empty required enum is missing")
	}
	if _, missing := validationField(t, types.CheckEnum("e", types.ResetType("Warm"), false)); missing {
		t.Error("bad enum value is not missing")
	}

	zero := 0
	if types.CheckMin("n", 0, 0) != nil || types.CheckMin("n", -1, 0) == nil {
		t.Error("CheckMin")
	}
	if types.CheckMinPtr("n", nil, 5) != nil || types.CheckMinPtr("n", &zero, 0) != nil || types.CheckMinPtr("n", &zero, 1) == nil {
		t.Error("CheckMinPtr")
	}
	if types.CheckDateTime("t", types.Now()) != nil {
		t.Error("set DateTime")
	}
	if _, missing := validationField(t, types.CheckDateTime("t", types.DateTime{})); !missing {
		t.Error("zero DateTime is missing")
	}

	for _, ok := range []float64{0, 8.1, 16, 6.5, -3.2, 0.1 + 0.2 + 0.7} {
		if err := types.CheckOneDecimal("l", ok); err != nil {
			t.Errorf("%v: %v", ok, err)
		}
	}
	for _, bad := range []float64{8.15, 0.01, math.NaN(), math.Inf(1)} {
		if err := types.CheckOneDecimal("l", bad); err == nil {
			t.Errorf("%v should be rejected", bad)
		}
	}

	if got, _ := validationField(t, types.Prefix("parent", ocpp.Required("child"))); got != "parent.child" {
		t.Errorf("Prefix: %q", got)
	}
	plain := errors.New("plain")
	if types.Prefix("parent", plain) != plain || types.Prefix("parent", nil) != nil {
		t.Error("Prefix must pass other errors through")
	}
	if types.First(nil, nil) != nil || types.First(nil, plain, errors.New("x")) != plain {
		t.Error("First")
	}
}

func TestStructValidation(t *testing.T) {
	one := 1
	ok := func(name string, err error) {
		t.Helper()
		if err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	bad := func(name string, err error, field string) {
		t.Helper()
		if err == nil {
			t.Errorf("%s: expected an error", name)
			return
		}
		if got, _ := validationField(t, err); got != field {
			t.Errorf("%s: field %q, want %q", name, got, field)
		}
	}

	ok("IDTagInfo", types.IDTagInfo{Status: types.AuthorizationStatusAccepted, ParentIDTag: "p"}.Validate())
	bad("IDTagInfo status", types.IDTagInfo{}.Validate(), "status")
	bad("IDTagInfo parent", types.IDTagInfo{Status: types.AuthorizationStatusAccepted, ParentIDTag: strings.Repeat("p", 21)}.Validate(), "parentIdTag")

	sv := types.SampledValue{Value: "1"}
	ok("SampledValue", sv.Validate())
	bad("SampledValue value", types.SampledValue{}.Validate(), "value")
	for field, mutate := range map[string]func(*types.SampledValue){
		"context":   func(s *types.SampledValue) { s.Context = "x" },
		"format":    func(s *types.SampledValue) { s.Format = "x" },
		"measurand": func(s *types.SampledValue) { s.Measurand = "x" },
		"phase":     func(s *types.SampledValue) { s.Phase = "x" },
		"location":  func(s *types.SampledValue) { s.Location = "x" },
		"unit":      func(s *types.SampledValue) { s.Unit = "x" },
	} {
		s := sv
		mutate(&s)
		bad("SampledValue "+field, s.Validate(), field)
	}

	mv := types.MeterValue{Timestamp: types.Now(), SampledValue: []types.SampledValue{sv}}
	ok("MeterValue", mv.Validate())
	bad("MeterValue timestamp", types.MeterValue{SampledValue: []types.SampledValue{sv}}.Validate(), "timestamp")
	bad("MeterValue no samples", types.MeterValue{Timestamp: types.Now()}.Validate(), "sampledValue")
	bad("MeterValue bad sample", types.MeterValue{Timestamp: types.Now(), SampledValue: []types.SampledValue{sv, {}}}.Validate(), "sampledValue[1].value")
	ok("ValidateMeterValues", types.ValidateMeterValues("m", []types.MeterValue{mv}))
	ok("ValidateMeterValues empty", types.ValidateMeterValues("m", nil))
	bad("ValidateMeterValues bad", types.ValidateMeterValues("m", []types.MeterValue{mv, {}}), "m[1].timestamp")

	ok("KeyValue", types.KeyValue{Key: "k", Value: "v"}.Validate())
	bad("KeyValue key", types.KeyValue{}.Validate(), "key")
	bad("KeyValue value", types.KeyValue{Key: "k", Value: strings.Repeat("v", 501)}.Validate(), "value")

	ok("period", types.ChargingSchedulePeriod{StartPeriod: 0, Limit: 16, NumberPhases: &one}.Validate())
	bad("period start", types.ChargingSchedulePeriod{StartPeriod: -1}.Validate(), "startPeriod")
	bad("period limit", types.ChargingSchedulePeriod{Limit: 1.25}.Validate(), "limit")
	zero := 0
	bad("period phases", types.ChargingSchedulePeriod{NumberPhases: &zero}.Validate(), "numberPhases")

	sched := types.ChargingSchedule{ChargingRateUnit: types.ChargingRateUnitTypeA,
		ChargingSchedulePeriod: []types.ChargingSchedulePeriod{{StartPeriod: 0, Limit: 10}, {StartPeriod: 600, Limit: 6}}}
	ok("schedule", sched.Validate())
	min := 6.5
	withMin := sched
	withMin.MinChargingRate = &min
	ok("schedule min rate", withMin.Validate())
	badMin := 6.55
	withMin.MinChargingRate = &badMin
	bad("schedule min rate", withMin.Validate(), "minChargingRate")
	neg := -1
	negDur := sched
	negDur.Duration = &neg
	bad("schedule duration", negDur.Validate(), "duration")
	noUnit := sched
	noUnit.ChargingRateUnit = ""
	bad("schedule unit", noUnit.Validate(), "chargingRateUnit")
	noPeriods := sched
	noPeriods.ChargingSchedulePeriod = nil
	bad("schedule periods", noPeriods.Validate(), "chargingSchedulePeriod")
	late := sched
	late.ChargingSchedulePeriod = []types.ChargingSchedulePeriod{{StartPeriod: 60, Limit: 10}}
	bad("schedule first period", late.Validate(), "chargingSchedulePeriod[0].startPeriod")
	badPeriod := sched
	badPeriod.ChargingSchedulePeriod = []types.ChargingSchedulePeriod{{StartPeriod: 0, Limit: 10}, {StartPeriod: 5, Limit: 1.234}}
	bad("schedule bad period", badPeriod.Validate(), "chargingSchedulePeriod[1].limit")

	prof := types.ChargingProfile{ChargingProfileID: 1, StackLevel: 0,
		ChargingProfilePurpose: types.ChargingProfilePurposeTypeTxProfile,
		ChargingProfileKind:    types.ChargingProfileKindTypeAbsolute, ChargingSchedule: sched}
	ok("profile", prof.Validate())
	for field, mutate := range map[string]func(*types.ChargingProfile){
		"stackLevel":                        func(p *types.ChargingProfile) { p.StackLevel = -1 },
		"chargingProfilePurpose":            func(p *types.ChargingProfile) { p.ChargingProfilePurpose = "" },
		"chargingProfileKind":               func(p *types.ChargingProfile) { p.ChargingProfileKind = "x" },
		"recurrencyKind":                    func(p *types.ChargingProfile) { p.RecurrencyKind = "Monthly" },
		"chargingSchedule.chargingRateUnit": func(p *types.ChargingProfile) { p.ChargingSchedule.ChargingRateUnit = "" },
	} {
		p := prof
		mutate(&p)
		bad("profile "+field, p.Validate(), field)
	}
}

func TestProfileEnumsMatchSchemas(t *testing.T) {
	tests := []struct {
		name   string
		valid  func(string) bool
		schema string
		path   []string
	}{
		{"DiagnosticsStatus", func(s string) bool { return types.DiagnosticsStatus(s).IsValid() }, "DiagnosticsStatusNotification", []string{"status"}},
		{"FirmwareStatus", func(s string) bool { return types.FirmwareStatus(s).IsValid() }, "FirmwareStatusNotification", []string{"status"}},
		{"UpdateType", func(s string) bool { return types.UpdateType(s).IsValid() }, "SendLocalList", []string{"updateType"}},
		{"UpdateStatus", func(s string) bool { return types.UpdateStatus(s).IsValid() }, "SendLocalListResponse", []string{"status"}},
		{"ReservationStatus", func(s string) bool { return types.ReservationStatus(s).IsValid() }, "ReserveNowResponse", []string{"status"}},
		{"CancelReservationStatus", func(s string) bool { return types.CancelReservationStatus(s).IsValid() }, "CancelReservationResponse", []string{"status"}},
		{"ChargingProfileStatus", func(s string) bool { return types.ChargingProfileStatus(s).IsValid() }, "SetChargingProfileResponse", []string{"status"}},
		{"ClearChargingProfileStatus", func(s string) bool { return types.ClearChargingProfileStatus(s).IsValid() }, "ClearChargingProfileResponse", []string{"status"}},
		{"GetCompositeScheduleStatus", func(s string) bool { return types.GetCompositeScheduleStatus(s).IsValid() }, "GetCompositeScheduleResponse", []string{"status"}},
		{"MessageTrigger", func(s string) bool { return types.MessageTrigger(s).IsValid() }, "TriggerMessage", []string{"requestedMessage"}},
		{"TriggerMessageStatus", func(s string) bool { return types.TriggerMessageStatus(s).IsValid() }, "TriggerMessageResponse", []string{"status"}},
	}
	for _, tt := range tests {
		values := enumValues(t, tt.schema, tt.path...)
		if len(values) == 0 {
			t.Fatalf("%s: no values read from the schema", tt.name)
		}
		for _, v := range values {
			if !tt.valid(v) {
				t.Errorf("%s: %q should be valid", tt.name, v)
			}
		}
		for _, bad := range []string{"", "Bogus", strings.ToLower(values[0]) + "x"} {
			if tt.valid(bad) {
				t.Errorf("%s: %q should be invalid", tt.name, bad)
			}
		}
	}
}

func TestCheckURIAndAuthorizationData(t *testing.T) {
	for _, ok := range []string{"ftp://host/dir", "https://fw.example.com/x.bin", "file:///tmp/x", "urn:isbn:1"} {
		if err := types.CheckURI("u", ok, true); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
	if err := types.CheckURI("u", "", false); err != nil {
		t.Error(err)
	}
	if _, missing := validationField(t, types.CheckURI("u", "", true)); !missing {
		t.Error("empty required URI is missing")
	}
	for _, bad := range []string{"not a uri", "/just/a/path", "host/path", "http://[::1", "%zz"} {
		if _, missing := validationField(t, types.CheckURI("u", bad, true)); missing {
			t.Errorf("%q: invalid, not missing", bad)
		}
	}

	if err := (types.AuthorizationData{IDTag: "A"}).Validate(); err != nil {
		t.Error(err)
	}
	info := &types.IDTagInfo{Status: types.AuthorizationStatusAccepted}
	if err := (types.AuthorizationData{IDTag: "A", IDTagInfo: info}).Validate(); err != nil {
		t.Error(err)
	}
	if f, _ := validationField(t, (types.AuthorizationData{}).Validate()); f != "idTag" {
		t.Errorf("field %q", f)
	}
	if f, _ := validationField(t, (types.AuthorizationData{IDTag: "A", IDTagInfo: &types.IDTagInfo{}}).Validate()); f != "idTagInfo.status" {
		t.Errorf("field %q", f)
	}
}
