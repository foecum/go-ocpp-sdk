package firmware_test

import (
	"errors"
	"strings"
	"testing"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/schematest"
	"github.com/foecum/go-ocpp-sdk/ocppj"
	"github.com/foecum/go-ocpp-sdk/v16/firmware"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

func registry(t testing.TB) *ocppj.Registry {
	t.Helper()
	reg := ocppj.NewRegistry(ocpp.V16)
	for _, f := range firmware.Features() {
		if err := reg.Register(f); err != nil {
			t.Fatal(err)
		}
	}
	return reg
}

func TestAgainstOfficialSchemas(t *testing.T) {
	schematest.Run(t, "../../docs/spec/1.6/schemas/json", registry(t), samples)
	if len(firmware.Features()) != len(samples) || len(samples) != 4 {
		t.Errorf("%d features, %d samples", len(firmware.Features()), len(samples))
	}
}

type validator interface{ Validate() error }

func TestValidationRules(t *testing.T) {
	now := types.Now()
	neg := -1
	tests := []struct {
		name  string
		v     validator
		field string
	}{
		{"GetDiagnostics ok", firmware.GetDiagnosticsRequest{Location: "ftp://h/d"}, ""},
		{"GetDiagnostics no location", firmware.GetDiagnosticsRequest{}, "location"},
		{"GetDiagnostics relative location", firmware.GetDiagnosticsRequest{Location: "/tmp/x"}, "location"},
		{"GetDiagnostics retries", firmware.GetDiagnosticsRequest{Location: "ftp://h/d", Retries: &neg}, "retries"},
		{"GetDiagnostics retryInterval", firmware.GetDiagnosticsRequest{Location: "ftp://h/d", RetryInterval: &neg}, "retryInterval"},
		{"GetDiagnostics.conf ok", firmware.GetDiagnosticsConfirmation{}, ""},
		{"GetDiagnostics.conf long", firmware.GetDiagnosticsConfirmation{FileName: strings.Repeat("a", 256)}, "fileName"},
		{"DiagnosticsStatus ok", firmware.DiagnosticsStatusNotificationRequest{Status: types.DiagnosticsStatusIdle}, ""},
		{"DiagnosticsStatus missing", firmware.DiagnosticsStatusNotificationRequest{}, "status"},
		{"DiagnosticsStatus bad", firmware.DiagnosticsStatusNotificationRequest{Status: "Nope"}, "status"},
		{"FirmwareStatus ok", firmware.FirmwareStatusNotificationRequest{Status: types.FirmwareStatusInstalled}, ""},
		{"FirmwareStatus missing", firmware.FirmwareStatusNotificationRequest{}, "status"},
		{"UpdateFirmware ok", firmware.UpdateFirmwareRequest{Location: "https://h/f", RetrieveDate: now}, ""},
		{"UpdateFirmware location", firmware.UpdateFirmwareRequest{RetrieveDate: now}, "location"},
		{"UpdateFirmware retries", firmware.UpdateFirmwareRequest{Location: "https://h/f", RetrieveDate: now, Retries: &neg}, "retries"},
		{"UpdateFirmware date", firmware.UpdateFirmwareRequest{Location: "https://h/f"}, "retrieveDate"},
		{"UpdateFirmware retryInterval", firmware.UpdateFirmwareRequest{Location: "https://h/f", RetrieveDate: now, RetryInterval: &neg}, "retryInterval"},
		{"empty confirmations", firmware.UpdateFirmwareConfirmation{}, ""},
	}
	for _, tt := range tests {
		err := tt.v.Validate()
		if tt.field == "" {
			if err != nil {
				t.Errorf("%s: %v", tt.name, err)
			}
			continue
		}
		var ve *ocpp.ValidationError
		if !errors.As(err, &ve) || ve.Field != tt.field {
			t.Errorf("%s: got %v, want an error on %q", tt.name, err, tt.field)
		}
	}
	for _, c := range []validator{firmware.DiagnosticsStatusNotificationConfirmation{}, firmware.FirmwareStatusNotificationConfirmation{}} {
		if c.Validate() != nil {
			t.Errorf("%T must always validate", c)
		}
	}
}

func TestActions(t *testing.T) {
	for _, f := range firmware.Features() {
		req := reflectNew(f.RequestType()).(ocpp.Request)
		resp := reflectNew(f.ResponseType()).(ocpp.Response)
		if req.Action() != f.Action() || resp.Action() != f.Action() {
			t.Errorf("%s: %q / %q", f.Action(), req.Action(), resp.Action())
		}
	}
}
