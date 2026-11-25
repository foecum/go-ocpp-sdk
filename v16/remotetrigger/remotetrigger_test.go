package remotetrigger_test

import (
	"errors"
	"testing"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/schematest"
	"github.com/foecum/go-ocpp-sdk/v16/remotetrigger"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

var samples = map[string]schematest.Sample{
	"TriggerMessage": {
		ReqMin: `{"requestedMessage":"Heartbeat"}`, ReqFull: `{"requestedMessage":"StatusNotification","connectorId":1}`,
		ConfMin: `{"status":"Accepted"}`, ConfFull: `{"status":"NotImplemented"}`,
	},
}

func TestAgainstOfficialSchemas(t *testing.T) {
	features := remotetrigger.Features()
	schematest.Run(t, "../../docs/spec/1.6/schemas/json", schematest.NewRegistry(t, features), samples)
	schematest.CheckActions(t, features)
}

func TestValidationRules(t *testing.T) {
	zero, one := 0, 1
	tests := []struct {
		name  string
		v     interface{ Validate() error }
		field string
	}{
		{"ok", remotetrigger.TriggerMessageRequest{RequestedMessage: types.MessageTriggerHeartbeat}, ""},
		{"ok with connector", remotetrigger.TriggerMessageRequest{RequestedMessage: types.MessageTriggerMeterValues, ConnectorID: &one}, ""},
		{"missing message", remotetrigger.TriggerMessageRequest{}, "requestedMessage"},
		{"unknown message", remotetrigger.TriggerMessageRequest{RequestedMessage: "Authorize"}, "requestedMessage"},
		{"connector 0", remotetrigger.TriggerMessageRequest{RequestedMessage: types.MessageTriggerMeterValues, ConnectorID: &zero}, "connectorId"},
		{"conf", remotetrigger.TriggerMessageConfirmation{Status: types.TriggerMessageStatusRejected}, ""},
		{"conf missing", remotetrigger.TriggerMessageConfirmation{}, "status"},
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
}
