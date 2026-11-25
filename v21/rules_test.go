package v21_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/foecum/go-ocpp-sdk/v21"
)

const (
	token    = `{"idToken":"A","type":"Local"}`
	schedule = `{"id":1,"chargingRateUnit":"A","chargingSchedulePeriod":[{"startPeriod":0,"limit":%s%s}]}`
)

func setProfile(purpose, extra, limit, periodExtra string) string {
	sched := strings.Replace(strings.Replace(schedule, "%s", limit, 1), "%s", periodExtra, 1)
	return `{"evseId":1,"chargingProfile":{"id":1,"stackLevel":0,"chargingProfilePurpose":"` + purpose + `","chargingProfileKind":"Absolute"` + extra + `,"chargingSchedule":[` + sched + `]}}`
}

// The specification text states some constraints that the JSON schemas do not;
// the messages enforce them. want is empty when the payload must be accepted,
// otherwise a text the error must contain.
func TestConstraintsOfTheSpecificationText(t *testing.T) {
	reg, err := v21.NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	notifyReport := func(dataType, extra string) string {
		return `{"requestId":1,"generatedAt":"2026-01-01T00:00:00Z","seqNo":0,"reportData":[{"component":{"name":"C"},"variable":{"name":"V"},` +
			`"variableAttribute":[{"type":"Actual"}],"variableCharacteristics":{"dataType":"` + dataType + `","supportsMonitoring":true` + extra + `}}]}`
	}
	tests := []struct {
		name, action, payload, want string
	}{
		{"full list entry without idTokenInfo", "SendLocalList", `{"versionNumber":1,"updateType":"Full","localAuthorizationList":[{"idToken":` + token + `}]}`, "idTokenInfo"},
		{"full list entry with idTokenInfo", "SendLocalList", `{"versionNumber":1,"updateType":"Full","localAuthorizationList":[{"idToken":` + token + `,"idTokenInfo":{"status":"Accepted"}}]}`, ""},
		{"differential entry without idTokenInfo deletes it", "SendLocalList", `{"versionNumber":1,"updateType":"Differential","localAuthorizationList":[{"idToken":` + token + `}]}`, ""},

		{"charging limit on evse 0", "NotifyChargingLimit", `{"chargingLimit":{"chargingLimitSource":"EMS"},"evseId":0}`, "evseId"},
		{"charging limit on evse 1", "NotifyChargingLimit", `{"chargingLimit":{"chargingLimitSource":"EMS"},"evseId":1}`, ""},
		{"charging limit without evse", "NotifyChargingLimit", `{"chargingLimit":{"chargingLimitSource":"EMS"}}`, ""},
		{"ev charging schedule on evse 0", "NotifyEVChargingSchedule", `{"timeBase":"2026-01-01T00:00:00Z","chargingSchedule":` + strings.Replace(strings.Replace(schedule, "%s", "8", 1), "%s", "", 1) + `,"evseId":0}`, "evseId"},
		{"ev charging schedule on evse 2", "NotifyEVChargingSchedule", `{"timeBase":"2026-01-01T00:00:00Z","chargingSchedule":` + strings.Replace(strings.Replace(schedule, "%s", "8", 1), "%s", "", 1) + `,"evseId":2}`, ""},

		{"remote start on evse 0", "RequestStartTransaction", `{"idToken":` + token + `,"remoteStartId":1,"evseId":0}`, "evseId"},
		{"remote start on evse 1", "RequestStartTransaction", `{"idToken":` + token + `,"remoteStartId":1,"evseId":1}`, ""},
		{"remote start with a TxDefaultProfile", "RequestStartTransaction", `{"idToken":` + token + `,"remoteStartId":1,"chargingProfile":` + setProfile("TxDefaultProfile", "", "8", "")[len(`{"evseId":1,"chargingProfile":`):], "TxProfile"},
		{"remote start with a TxProfile", "RequestStartTransaction", `{"idToken":` + token + `,"remoteStartId":1,"chargingProfile":` + setProfile("TxProfile", "", "8", "")[len(`{"evseId":1,"chargingProfile":`):], ""},

		{"transactionId on a TxDefaultProfile", "SetChargingProfile", setProfile("TxDefaultProfile", `,"transactionId":"tx1"`, "8", ""), "transactionId"},
		{"transactionId on a TxProfile", "SetChargingProfile", setProfile("TxProfile", `,"transactionId":"tx1"`, "8", ""), ""},
		{"a TxDefaultProfile without a transaction", "SetChargingProfile", setProfile("TxDefaultProfile", "", "8", ""), ""},

		{"limit with one decimal", "SetChargingProfile", setProfile("TxDefaultProfile", "", "8.1", ""), ""},
		{"limit with two decimals", "SetChargingProfile", setProfile("TxDefaultProfile", "", "8.14", ""), "limit"},
		{"phaseToUse 3", "SetChargingProfile", setProfile("TxDefaultProfile", "", "8", `,"phaseToUse":3`), ""},
		{"phaseToUse 0", "SetChargingProfile", setProfile("TxDefaultProfile", "", "8", `,"phaseToUse":0`), "phaseToUse"},
		{"phaseToUse 4", "SetChargingProfile", setProfile("TxDefaultProfile", "", "8", `,"phaseToUse":4`), "phaseToUse"},

		{"option list without values", "NotifyReport", notifyReport("OptionList", ""), "valuesList"},
		{"member list without values", "NotifyReport", notifyReport("MemberList", ""), "valuesList"},
		{"sequence list without values", "NotifyReport", notifyReport("SequenceList", ""), "valuesList"},
		{"option list with values", "NotifyReport", notifyReport("OptionList", `,"valuesList":"A,B"`), ""},
		{"string needs no values", "NotifyReport", notifyReport("string", ""), ""},

		{"published without a location", "PublishFirmwareStatusNotification", `{"status":"Published"}`, "location"},
		{"published with a location", "PublishFirmwareStatusNotification", `{"status":"Published","location":["http://x/fw"]}`, ""},
		{"downloading needs no location", "PublishFirmwareStatusNotification", `{"status":"Downloading"}`, ""},

		{"period with a setpoint and no limit", "SetChargingProfile", setProfile("TxDefaultProfile", "", "8", `,"setpoint":-3.5`), ""},
		{"local frequency without a curve", "SetChargingProfile", setProfile("TxDefaultProfile", "", "8", `,"operationMode":"LocalFrequency"`), "v2xFreqWattCurve"},
		{"local frequency with one point", "SetChargingProfile", setProfile("TxDefaultProfile", "", "8", `,"operationMode":"LocalFrequency","v2xFreqWattCurve":[{"frequency":50,"power":1}]`), "v2xFreqWattCurve"},
		{"local frequency with two points", "SetChargingProfile", setProfile("TxDefaultProfile", "", "8", `,"operationMode":"LocalFrequency","v2xFreqWattCurve":[{"frequency":49,"power":1},{"frequency":51,"power":-1}]`), ""},
		{"a frequency curve with one point in another mode", "SetChargingProfile", setProfile("TxDefaultProfile", "", "8", `,"v2xFreqWattCurve":[{"frequency":50,"power":1}]`), "v2xFreqWattCurve"},
		{"a signal curve with one point", "SetChargingProfile", setProfile("TxDefaultProfile", "", "8", `,"v2xSignalWattCurve":[{"signal":1,"power":1}]`), "v2xSignalWattCurve"},
		{"a signal curve with two points", "SetChargingProfile", setProfile("TxDefaultProfile", "", "8", `,"v2xSignalWattCurve":[{"signal":-1,"power":1},{"signal":1,"power":-1}]`), ""},
		{"randomized delay on a TxDefaultProfile", "SetChargingProfile", strings.Replace(setProfile("TxDefaultProfile", "", "8", ""), `{"id":1,"chargingRateUnit"`, `{"id":1,"randomizedDelay":30,"chargingRateUnit"`, 1), ""},
		{"randomized delay on a TxProfile", "SetChargingProfile", strings.Replace(setProfile("TxProfile", "", "8", ""), `{"id":1,"chargingRateUnit"`, `{"id":1,"randomizedDelay":30,"chargingRateUnit"`, 1), ""},
		{"randomized delay on a charging station profile", "SetChargingProfile", strings.Replace(setProfile("ChargingStationMaxProfile", "", "8", ""), `{"id":1,"chargingRateUnit"`, `{"id":1,"randomizedDelay":30,"chargingRateUnit"`, 1), "randomizedDelay"},
		{"a zero randomized delay is no delay", "SetChargingProfile", strings.Replace(setProfile("ChargingStationMaxProfile", "", "8", ""), `{"id":1,"chargingRateUnit"`, `{"id":1,"randomizedDelay":0,"chargingRateUnit"`, 1), ""},

		{"set a stand-alone PowerMonitoringMustTrip", "SetDERControl", `{"isDefault":true,"controlId":"c1","controlType":"PowerMonitoringMustTrip"}`, "controlType"},
		{"get a stand-alone PowerMonitoringMustTrip", "GetDERControl", `{"requestId":1,"controlType":"PowerMonitoringMustTrip"}`, "controlType"},
		{"clear a stand-alone PowerMonitoringMustTrip", "ClearDERControl", `{"isDefault":true,"controlType":"PowerMonitoringMustTrip"}`, "controlType"},
		{"alarm for a stand-alone PowerMonitoringMustTrip", "NotifyDERAlarm", `{"controlType":"PowerMonitoringMustTrip","timestamp":"2026-01-01T00:00:00Z"}`, "controlType"},
		{"get another control", "GetDERControl", `{"requestId":1,"controlType":"LFMustTrip"}`, ""},
		{"clear another control", "ClearDERControl", `{"isDefault":true,"controlType":"LFMustTrip"}`, ""},
		{"alarm for another control", "NotifyDERAlarm", `{"controlType":"LFMustTrip","timestamp":"2026-01-01T00:00:00Z"}`, ""},

		{"severity 0", "SetMonitoringLevel", `{"severity":0}`, ""},
		{"severity 9", "SetMonitoringLevel", `{"severity":9}`, ""},
		{"severity 10", "SetMonitoringLevel", `{"severity":10}`, "severity"},
		{"severity -1", "SetMonitoringLevel", `{"severity":-1}`, "severity"},
	}
	for _, tt := range tests {
		if !json.Valid([]byte(tt.payload)) {
			t.Fatalf("%s: the test payload is not JSON: %s", tt.name, tt.payload)
		}
		_, err := reg.DecodeRequest(tt.action, []byte(tt.payload))
		switch {
		case tt.want == "" && err != nil:
			t.Errorf("%s: unexpectedly rejected: %v", tt.name, err)
		case tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)):
			t.Errorf("%s: want an error containing %q, got %v", tt.name, tt.want, err)
		}
	}
}
