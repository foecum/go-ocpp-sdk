package localauth_test

import (
	"errors"
	"testing"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/schematest"
	"github.com/foecum/go-ocpp-sdk/v16/localauth"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

var samples = map[string]schematest.Sample{
	"GetLocalListVersion": {
		ReqMin: `{}`, ReqFull: `{}`,
		ConfMin: `{"listVersion":-1}`, ConfFull: `{"listVersion":239}`,
	},
	"SendLocalList": {
		ReqMin: `{"listVersion":1,"updateType":"Full"}`,
		ReqFull: `{"listVersion":2,"localAuthorizationList":[` +
			`{"idTag":"04A1B2C3","idTagInfo":{"expiryDate":"2026-12-31T23:59:59Z","parentIdTag":"PARENT01","status":"Accepted"}},` +
			`{"idTag":"04D4E5F6"}],"updateType":"Differential"}`,
		ConfMin: `{"status":"Accepted"}`, ConfFull: `{"status":"VersionMismatch"}`,
	},
}

func TestAgainstOfficialSchemas(t *testing.T) {
	features := localauth.Features()
	schematest.Run(t, "../../docs/spec/1.6/schemas/json", schematest.NewRegistry(t, features), samples)
	schematest.CheckActions(t, features)
	if len(features) != len(samples) {
		t.Errorf("%d features, %d samples", len(features), len(samples))
	}
}

func TestValidationRules(t *testing.T) {
	info := &types.IDTagInfo{Status: types.AuthorizationStatusAccepted}
	entry := func(tag string, withInfo bool) types.AuthorizationData {
		e := types.AuthorizationData{IDTag: tag}
		if withInfo {
			e.IDTagInfo = info
		}
		return e
	}
	tests := []struct {
		name  string
		v     interface{ Validate() error }
		field string
	}{
		{"GetLocalListVersion", localauth.GetLocalListVersionRequest{}, ""},
		{"conf -1 (unsupported)", localauth.GetLocalListVersionConfirmation{ListVersion: -1}, ""},
		{"conf 0 (empty)", localauth.GetLocalListVersionConfirmation{ListVersion: 0}, ""},
		{"conf below -1", localauth.GetLocalListVersionConfirmation{ListVersion: -2}, "listVersion"},

		{"empty Full list clears everything", localauth.SendLocalListRequest{ListVersion: 1, UpdateType: types.UpdateTypeFull}, ""},
		{"empty Differential list", localauth.SendLocalListRequest{ListVersion: 1, UpdateType: types.UpdateTypeDifferential}, ""},
		{"update type missing", localauth.SendLocalListRequest{ListVersion: 1}, "updateType"},
		{"update type bad", localauth.SendLocalListRequest{ListVersion: 1, UpdateType: "Partial"}, "updateType"},
		{"Full with info", localauth.SendLocalListRequest{UpdateType: types.UpdateTypeFull,
			LocalAuthorizationList: []types.AuthorizationData{entry("A", true), entry("B", true)}}, ""},
		{"Full needs idTagInfo", localauth.SendLocalListRequest{UpdateType: types.UpdateTypeFull,
			LocalAuthorizationList: []types.AuthorizationData{entry("A", true), entry("B", false)}}, "localAuthorizationList[1].idTagInfo"},
		{"Differential may omit idTagInfo (delete)", localauth.SendLocalListRequest{UpdateType: types.UpdateTypeDifferential,
			LocalAuthorizationList: []types.AuthorizationData{entry("A", false), entry("B", true)}}, ""},
		{"duplicate idTag", localauth.SendLocalListRequest{UpdateType: types.UpdateTypeDifferential,
			LocalAuthorizationList: []types.AuthorizationData{entry("A", true), entry("B", true), entry("A", false)}}, "localAuthorizationList[2].idTag"},
		{"idTags are case insensitive", localauth.SendLocalListRequest{UpdateType: types.UpdateTypeDifferential,
			LocalAuthorizationList: []types.AuthorizationData{entry("abc", true), entry("ABC", true)}}, "localAuthorizationList[1].idTag"},
		{"bad entry", localauth.SendLocalListRequest{UpdateType: types.UpdateTypeFull,
			LocalAuthorizationList: []types.AuthorizationData{entry("", true)}}, "localAuthorizationList[0].idTag"},
		{"bad entry info", localauth.SendLocalListRequest{UpdateType: types.UpdateTypeFull,
			LocalAuthorizationList: []types.AuthorizationData{{IDTag: "A", IDTagInfo: &types.IDTagInfo{}}}}, "localAuthorizationList[0].idTagInfo.status"},
		{"SendLocalList.conf", localauth.SendLocalListConfirmation{Status: types.UpdateStatusFailed}, ""},
		{"SendLocalList.conf missing", localauth.SendLocalListConfirmation{}, "status"},
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
