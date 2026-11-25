package firmware_test

import "github.com/foecum/go-ocpp-sdk/internal/schematest"

var samples = map[string]schematest.Sample{
	"GetDiagnostics": {
		ReqMin: `{"location":"ftp://logs.example.com/diag"}`,
		ReqFull: `{"location":"ftp://logs.example.com/diag","retries":3,"retryInterval":60,` +
			`"startTime":"2026-01-01T00:00:00Z","stopTime":"2026-01-02T00:00:00.500Z"}`,
		ConfMin: `{}`, ConfFull: `{"fileName":"diag-2026-01-02.zip"}`,
	},
	"DiagnosticsStatusNotification": {
		ReqMin: `{"status":"Uploading"}`, ReqFull: `{"status":"UploadFailed"}`,
		ConfMin: `{}`, ConfFull: `{}`,
	},
	"FirmwareStatusNotification": {
		ReqMin: `{"status":"Downloaded"}`, ReqFull: `{"status":"InstallationFailed"}`,
		ConfMin: `{}`, ConfFull: `{}`,
	},
	"UpdateFirmware": {
		ReqMin:  `{"location":"https://fw.example.com/x1-1.2.3.bin","retrieveDate":"2026-03-01T02:00:00Z"}`,
		ReqFull: `{"location":"https://fw.example.com/x1-1.2.3.bin","retries":2,"retrieveDate":"2026-03-01T02:00:00.250Z","retryInterval":300}`,
		ConfMin: `{}`, ConfFull: `{}`,
	},
}
