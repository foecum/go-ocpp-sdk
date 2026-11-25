package core_test

import "github.com/foecum/go-ocpp-sdk/internal/schematest"

const (
	ts  = `"2026-01-02T03:04:05Z"`
	tsM = `"2026-01-02T03:04:05.123Z"`
)

const fullIDTagInfo = `{"expiryDate":"2026-12-31T23:59:59Z","parentIdTag":"PARENT01","status":"Accepted"}`

const sampledValueFull = `{"value":"1234.5","context":"Sample.Periodic","format":"Raw","measurand":"Energy.Active.Import.Register","phase":"L1","location":"Outlet","unit":"Wh"}`

// samples are valid payloads for every Core message, with only the required
// fields (Min) and with every field (Full).
var samples = map[string]schematest.Sample{
	"Authorize": {
		ReqMin: `{"idTag":"04A1B2C3"}`, ReqFull: `{"idTag":"04A1B2C3"}`,
		ConfMin: `{"idTagInfo":{"status":"Accepted"}}`, ConfFull: `{"idTagInfo":` + fullIDTagInfo + `}`,
	},
	"BootNotification": {
		ReqMin: `{"chargePointVendor":"Acme","chargePointModel":"X1"}`,
		ReqFull: `{"chargePointVendor":"Acme","chargePointModel":"X1","chargePointSerialNumber":"SN-0001","chargeBoxSerialNumber":"CB-0001",` +
			`"firmwareVersion":"1.2.3","iccid":"8944500102198304826","imsi":"310150123456789","meterType":"EM-3","meterSerialNumber":"MSN-77"}`,
		ConfMin:  `{"status":"Accepted","currentTime":` + ts + `,"interval":300}`,
		ConfFull: `{"status":"Pending","currentTime":` + tsM + `,"interval":60}`,
	},
	"ChangeAvailability": {
		ReqMin: `{"connectorId":0,"type":"Inoperative"}`, ReqFull: `{"connectorId":2,"type":"Operative"}`,
		ConfMin: `{"status":"Accepted"}`, ConfFull: `{"status":"Scheduled"}`,
	},
	"ChangeConfiguration": {
		ReqMin: `{"key":"HeartbeatInterval","value":"300"}`, ReqFull: `{"key":"HeartbeatInterval","value":""}`,
		ConfMin: `{"status":"Accepted"}`, ConfFull: `{"status":"RebootRequired"}`,
	},
	"ClearCache": {
		ReqMin: `{}`, ReqFull: `{}`,
		ConfMin: `{"status":"Accepted"}`, ConfFull: `{"status":"Rejected"}`,
	},
	"DataTransfer": {
		ReqMin: `{"vendorId":"com.acme"}`, ReqFull: `{"vendorId":"com.acme","messageId":"ping","data":"{\"a\":1}"}`,
		ConfMin: `{"status":"Accepted"}`, ConfFull: `{"status":"UnknownMessageId","data":"reply"}`,
	},
	"GetConfiguration": {
		ReqMin: `{}`, ReqFull: `{"key":["HeartbeatInterval","MeterValueSampleInterval"]}`,
		ConfMin:  `{"configurationKey":[{"key":"HeartbeatInterval","readonly":true}]}`,
		ConfFull: `{"configurationKey":[{"key":"HeartbeatInterval","readonly":false,"value":"300"}],"unknownKey":["Nope"]}`,
	},
	"Heartbeat": {
		ReqMin: `{}`, ReqFull: `{}`,
		ConfMin: `{"currentTime":` + ts + `}`, ConfFull: `{"currentTime":` + tsM + `}`,
	},
	"MeterValues": {
		ReqMin:  `{"connectorId":1,"meterValue":[{"timestamp":` + ts + `,"sampledValue":[{"value":"1234"}]}]}`,
		ReqFull: `{"connectorId":1,"transactionId":42,"meterValue":[{"timestamp":` + tsM + `,"sampledValue":[` + sampledValueFull + `]}]}`,
		ConfMin: `{}`, ConfFull: `{}`,
	},
	"RemoteStartTransaction": {
		ReqMin: `{"idTag":"04A1B2C3"}`,
		ReqFull: `{"connectorId":1,"idTag":"04A1B2C3","chargingProfile":{"chargingProfileId":1,"transactionId":7,"stackLevel":0,` +
			`"chargingProfilePurpose":"TxProfile","chargingProfileKind":"Absolute","recurrencyKind":"Daily",` +
			`"validFrom":"2026-01-01T00:00:00Z","validTo":"2026-12-31T23:59:59Z","chargingSchedule":{"duration":3600,` +
			`"startSchedule":"2026-01-01T00:00:00Z","chargingRateUnit":"A","chargingSchedulePeriod":[{"startPeriod":0,"limit":16.5,"numberPhases":3}],"minChargingRate":6.5}}}`,
		ConfMin: `{"status":"Accepted"}`, ConfFull: `{"status":"Rejected"}`,
	},
	"RemoteStopTransaction": {
		ReqMin: `{"transactionId":42}`, ReqFull: `{"transactionId":42}`,
		ConfMin: `{"status":"Accepted"}`, ConfFull: `{"status":"Rejected"}`,
	},
	"Reset": {
		ReqMin: `{"type":"Soft"}`, ReqFull: `{"type":"Hard"}`,
		ConfMin: `{"status":"Accepted"}`, ConfFull: `{"status":"Rejected"}`,
	},
	"StartTransaction": {
		ReqMin:   `{"connectorId":1,"idTag":"04A1B2C3","meterStart":0,"timestamp":` + ts + `}`,
		ReqFull:  `{"connectorId":1,"idTag":"04A1B2C3","meterStart":100,"reservationId":5,"timestamp":` + tsM + `}`,
		ConfMin:  `{"idTagInfo":{"status":"Accepted"},"transactionId":42}`,
		ConfFull: `{"idTagInfo":` + fullIDTagInfo + `,"transactionId":43}`,
	},
	"StatusNotification": {
		ReqMin: `{"connectorId":1,"errorCode":"NoError","status":"Available"}`,
		ReqFull: `{"connectorId":0,"errorCode":"ConnectorLockFailure","info":"lock stuck","status":"Faulted","timestamp":` + tsM +
			`,"vendorId":"com.acme","vendorErrorCode":"E42"}`,
		ConfMin: `{}`, ConfFull: `{}`,
	},
	"StopTransaction": {
		ReqMin: `{"meterStop":1500,"timestamp":` + ts + `,"transactionId":42}`,
		ReqFull: `{"idTag":"04A1B2C3","meterStop":1500,"timestamp":` + tsM + `,"transactionId":42,"reason":"Local",` +
			`"transactionData":[{"timestamp":` + ts + `,"sampledValue":[` + sampledValueFull + `]}]}`,
		ConfMin: `{}`, ConfFull: `{"idTagInfo":{"status":"Accepted"}}`,
	},
	"UnlockConnector": {
		ReqMin: `{"connectorId":1}`, ReqFull: `{"connectorId":1}`,
		ConfMin: `{"status":"Unlocked"}`, ConfFull: `{"status":"UnlockFailed"}`,
	},
}
