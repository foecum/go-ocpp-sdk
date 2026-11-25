package core_test

import (
	"errors"
	"strings"
	"testing"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

func intp(i int) *int { return &i }

var now = types.NewDateTime(types.Now().Time)

func validSampled() []types.MeterValue {
	return []types.MeterValue{{Timestamp: now, SampledValue: []types.SampledValue{{Value: "1"}}}}
}

func validProfile() *types.ChargingProfile {
	return &types.ChargingProfile{
		ChargingProfileID: 1, ChargingProfilePurpose: types.ChargingProfilePurposeTypeTxProfile,
		ChargingProfileKind: types.ChargingProfileKindTypeAbsolute,
		ChargingSchedule: types.ChargingSchedule{ChargingRateUnit: types.ChargingRateUnitTypeA,
			ChargingSchedulePeriod: []types.ChargingSchedulePeriod{{Limit: 16}}},
	}
}

type validator interface{ Validate() error }

// TestValidationRules covers rules the specification states beyond what the JSON
// schemas express: connector id ranges, non-empty required strings,
// cardinalities, and the TxProfile rule.
func TestValidationRules(t *testing.T) {
	long := func(n int) string { return strings.Repeat("a", n+1) }
	tests := []struct {
		name  string
		v     validator
		field string // empty: must be valid
	}{
		{"Authorize ok", core.AuthorizeRequest{IDTag: "x"}, ""},
		{"Authorize empty", core.AuthorizeRequest{}, "idTag"},
		{"Authorize long", core.AuthorizeRequest{IDTag: long(20)}, "idTag"},
		{"Authorize.conf ok", core.AuthorizeConfirmation{IDTagInfo: types.IDTagInfo{Status: types.AuthorizationStatusBlocked}}, ""},
		{"Authorize.conf bad", core.AuthorizeConfirmation{}, "idTagInfo.status"},

		{"Boot ok", core.BootNotificationRequest{ChargePointVendor: "v", ChargePointModel: "m"}, ""},
		{"Boot no vendor", core.BootNotificationRequest{ChargePointModel: "m"}, "chargePointVendor"},
		{"Boot no model", core.BootNotificationRequest{ChargePointVendor: "v"}, "chargePointModel"},
		{"Boot serial", core.BootNotificationRequest{ChargePointVendor: "v", ChargePointModel: "m", ChargePointSerialNumber: long(25)}, "chargePointSerialNumber"},
		{"Boot box serial", core.BootNotificationRequest{ChargePointVendor: "v", ChargePointModel: "m", ChargeBoxSerialNumber: long(25)}, "chargeBoxSerialNumber"},
		{"Boot firmware", core.BootNotificationRequest{ChargePointVendor: "v", ChargePointModel: "m", FirmwareVersion: long(50)}, "firmwareVersion"},
		{"Boot iccid", core.BootNotificationRequest{ChargePointVendor: "v", ChargePointModel: "m", ICCID: long(20)}, "iccid"},
		{"Boot imsi", core.BootNotificationRequest{ChargePointVendor: "v", ChargePointModel: "m", IMSI: long(20)}, "imsi"},
		{"Boot meter type", core.BootNotificationRequest{ChargePointVendor: "v", ChargePointModel: "m", MeterType: long(25)}, "meterType"},
		{"Boot meter serial", core.BootNotificationRequest{ChargePointVendor: "v", ChargePointModel: "m", MeterSerialNumber: long(25)}, "meterSerialNumber"},
		{"Boot.conf ok", core.BootNotificationConfirmation{Status: types.RegistrationStatusAccepted, CurrentTime: now, Interval: 0}, ""},
		{"Boot.conf status", core.BootNotificationConfirmation{CurrentTime: now}, "status"},
		{"Boot.conf time", core.BootNotificationConfirmation{Status: types.RegistrationStatusAccepted}, "currentTime"},
		{"Boot.conf interval", core.BootNotificationConfirmation{Status: types.RegistrationStatusAccepted, CurrentTime: now, Interval: -1}, "interval"},

		{"ChangeAvailability ok", core.ChangeAvailabilityRequest{ConnectorID: 0, Type: types.AvailabilityTypeOperative}, ""},
		{"ChangeAvailability negative", core.ChangeAvailabilityRequest{ConnectorID: -1, Type: types.AvailabilityTypeOperative}, "connectorId"},
		{"ChangeAvailability type", core.ChangeAvailabilityRequest{}, "type"},
		{"ChangeAvailability.conf", core.ChangeAvailabilityConfirmation{}, "status"},
		{"ChangeAvailability.conf ok", core.ChangeAvailabilityConfirmation{Status: types.AvailabilityStatusAccepted}, ""},

		{"ChangeConfiguration ok (empty value)", core.ChangeConfigurationRequest{Key: "k"}, ""},
		{"ChangeConfiguration key", core.ChangeConfigurationRequest{}, "key"},
		{"ChangeConfiguration value", core.ChangeConfigurationRequest{Key: "k", Value: long(500)}, "value"},
		{"ChangeConfiguration.conf", core.ChangeConfigurationConfirmation{}, "status"},

		{"ClearCache", core.ClearCacheRequest{}, ""},
		{"ClearCache.conf", core.ClearCacheConfirmation{}, "status"},
		{"ClearCache.conf ok", core.ClearCacheConfirmation{Status: types.ClearCacheStatusAccepted}, ""},

		{"DataTransfer ok", core.DataTransferRequest{VendorID: "v"}, ""},
		{"DataTransfer vendor", core.DataTransferRequest{}, "vendorId"},
		{"DataTransfer vendor long", core.DataTransferRequest{VendorID: long(255)}, "vendorId"},
		{"DataTransfer message", core.DataTransferRequest{VendorID: "v", MessageID: long(50)}, "messageId"},
		{"DataTransfer.conf", core.DataTransferConfirmation{}, "status"},
		{"DataTransfer.conf ok", core.DataTransferConfirmation{Status: types.DataTransferStatusAccepted}, ""},

		{"GetConfiguration ok", core.GetConfigurationRequest{Key: []string{"a", "b"}}, ""},
		{"GetConfiguration empty key", core.GetConfigurationRequest{Key: []string{"a", ""}}, "key[1]"},
		{"GetConfiguration long key", core.GetConfigurationRequest{Key: []string{long(50)}}, "key[0]"},
		{"GetConfiguration.conf ok", core.GetConfigurationConfirmation{ConfigurationKey: []types.KeyValue{{Key: "a"}}, UnknownKey: []string{"b"}}, ""},
		{"GetConfiguration.conf key", core.GetConfigurationConfirmation{ConfigurationKey: []types.KeyValue{{Key: "a"}, {}}}, "configurationKey[1].key"},
		{"GetConfiguration.conf unknown", core.GetConfigurationConfirmation{UnknownKey: []string{long(50)}}, "unknownKey[0]"},

		{"Heartbeat", core.HeartbeatRequest{}, ""},
		{"Heartbeat.conf ok", core.HeartbeatConfirmation{CurrentTime: now}, ""},
		{"Heartbeat.conf", core.HeartbeatConfirmation{}, "currentTime"},

		{"MeterValues ok", core.MeterValuesRequest{ConnectorID: 0, MeterValue: validSampled()}, ""},
		{"MeterValues negative connector", core.MeterValuesRequest{ConnectorID: -1, MeterValue: validSampled()}, "connectorId"},
		{"MeterValues none", core.MeterValuesRequest{ConnectorID: 1}, "meterValue"},
		{"MeterValues bad sample", core.MeterValuesRequest{ConnectorID: 1, MeterValue: []types.MeterValue{{Timestamp: now, SampledValue: []types.SampledValue{{}}}}}, "meterValue[0].sampledValue[0].value"},
		{"MeterValues.conf", core.MeterValuesConfirmation{}, ""},

		{"RemoteStart ok", core.RemoteStartTransactionRequest{IDTag: "x"}, ""},
		{"RemoteStart ok full", core.RemoteStartTransactionRequest{IDTag: "x", ConnectorID: intp(1), ChargingProfile: validProfile()}, ""},
		{"RemoteStart connector 0", core.RemoteStartTransactionRequest{IDTag: "x", ConnectorID: intp(0)}, "connectorId"},
		{"RemoteStart idTag", core.RemoteStartTransactionRequest{}, "idTag"},
		{"RemoteStart profile purpose", core.RemoteStartTransactionRequest{IDTag: "x", ChargingProfile: func() *types.ChargingProfile {
			p := validProfile()
			p.ChargingProfilePurpose = types.ChargingProfilePurposeTypeTxDefaultProfile
			return p
		}()}, "chargingProfile.chargingProfilePurpose"},
		{"RemoteStart profile invalid", core.RemoteStartTransactionRequest{IDTag: "x", ChargingProfile: &types.ChargingProfile{}}, "chargingProfile.chargingProfilePurpose"},
		{"RemoteStart.conf", core.RemoteStartTransactionConfirmation{}, "status"},
		{"RemoteStart.conf ok", core.RemoteStartTransactionConfirmation{Status: types.RemoteStartStopStatusAccepted}, ""},

		{"RemoteStop", core.RemoteStopTransactionRequest{TransactionID: 0}, ""},
		{"RemoteStop.conf", core.RemoteStopTransactionConfirmation{}, "status"},
		{"RemoteStop.conf ok", core.RemoteStopTransactionConfirmation{Status: types.RemoteStartStopStatusRejected}, ""},

		{"Reset ok", core.ResetRequest{Type: types.ResetTypeSoft}, ""},
		{"Reset type", core.ResetRequest{}, "type"},
		{"Reset.conf", core.ResetConfirmation{}, "status"},
		{"Reset.conf ok", core.ResetConfirmation{Status: types.ResetStatusAccepted}, ""},

		{"StartTransaction ok", core.StartTransactionRequest{ConnectorID: 1, IDTag: "x", Timestamp: now}, ""},
		{"StartTransaction connector 0", core.StartTransactionRequest{ConnectorID: 0, IDTag: "x", Timestamp: now}, "connectorId"},
		{"StartTransaction idTag", core.StartTransactionRequest{ConnectorID: 1, Timestamp: now}, "idTag"},
		{"StartTransaction timestamp", core.StartTransactionRequest{ConnectorID: 1, IDTag: "x"}, "timestamp"},
		{"StartTransaction.conf ok", core.StartTransactionConfirmation{IDTagInfo: types.IDTagInfo{Status: types.AuthorizationStatusAccepted}}, ""},
		{"StartTransaction.conf", core.StartTransactionConfirmation{}, "idTagInfo.status"},

		{"StatusNotification ok", core.StatusNotificationRequest{ConnectorID: 0, ErrorCode: types.ChargePointErrorCodeNoError, Status: types.ChargePointStatusAvailable}, ""},
		{"StatusNotification negative", core.StatusNotificationRequest{ConnectorID: -1, ErrorCode: types.ChargePointErrorCodeNoError, Status: types.ChargePointStatusAvailable}, "connectorId"},
		{"StatusNotification errorCode", core.StatusNotificationRequest{Status: types.ChargePointStatusAvailable}, "errorCode"},
		{"StatusNotification info", core.StatusNotificationRequest{ErrorCode: types.ChargePointErrorCodeNoError, Status: types.ChargePointStatusAvailable, Info: long(50)}, "info"},
		{"StatusNotification status", core.StatusNotificationRequest{ErrorCode: types.ChargePointErrorCodeNoError}, "status"},
		{"StatusNotification vendorId", core.StatusNotificationRequest{ErrorCode: types.ChargePointErrorCodeNoError, Status: types.ChargePointStatusAvailable, VendorID: long(255)}, "vendorId"},
		{"StatusNotification vendorErrorCode", core.StatusNotificationRequest{ErrorCode: types.ChargePointErrorCodeNoError, Status: types.ChargePointStatusAvailable, VendorErrorCode: long(50)}, "vendorErrorCode"},
		{"StatusNotification.conf", core.StatusNotificationConfirmation{}, ""},

		{"StopTransaction ok", core.StopTransactionRequest{Timestamp: now}, ""},
		{"StopTransaction idTag", core.StopTransactionRequest{Timestamp: now, IDTag: long(20)}, "idTag"},
		{"StopTransaction timestamp", core.StopTransactionRequest{}, "timestamp"},
		{"StopTransaction reason", core.StopTransactionRequest{Timestamp: now, Reason: "Because"}, "reason"},
		{"StopTransaction data", core.StopTransactionRequest{Timestamp: now, TransactionData: []types.MeterValue{{}}}, "transactionData[0].timestamp"},
		{"StopTransaction.conf none", core.StopTransactionConfirmation{}, ""},
		{"StopTransaction.conf ok", core.StopTransactionConfirmation{IDTagInfo: &types.IDTagInfo{Status: types.AuthorizationStatusAccepted}}, ""},
		{"StopTransaction.conf bad", core.StopTransactionConfirmation{IDTagInfo: &types.IDTagInfo{}}, "idTagInfo.status"},

		{"UnlockConnector ok", core.UnlockConnectorRequest{ConnectorID: 1}, ""},
		{"UnlockConnector 0", core.UnlockConnectorRequest{ConnectorID: 0}, "connectorId"},
		{"UnlockConnector.conf", core.UnlockConnectorConfirmation{}, "status"},
		{"UnlockConnector.conf ok", core.UnlockConnectorConfirmation{Status: types.UnlockStatusUnlocked}, ""},
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

func TestActionsAndNaming(t *testing.T) {
	for _, f := range core.Features() {
		req := newRequest(f)
		resp := reflectNew(f.ResponseType()).(ocpp.Response)
		if req.Action() != f.Action() || resp.Action() != f.Action() {
			t.Errorf("%s: request %q, confirmation %q", f.Action(), req.Action(), resp.Action())
		}
		if !strings.HasSuffix(f.RequestType().Name(), "Request") || !strings.HasSuffix(f.ResponseType().Name(), "Confirmation") ||
			strings.TrimSuffix(f.RequestType().Name(), "Request") != f.Action() {
			t.Errorf("%s: unexpected type names %s / %s", f.Action(), f.RequestType().Name(), f.ResponseType().Name())
		}
	}
}
