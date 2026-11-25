package security_test

import (
	"errors"
	"strings"
	"testing"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/v16/security"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

type validator interface{ Validate() error }

func intp(i int) *int { return &i }

func TestValidationRules(t *testing.T) {
	p := makePEMs(t)
	now := types.Now()
	hash := strings.Repeat("ab", 32)
	goodHash := security.CertificateHashData{HashAlgorithm: security.HashAlgorithmSHA256, IssuerNameHash: hash, IssuerKeyHash: hash, SerialNumber: "FFF"}
	withHash := func(f func(*security.CertificateHashData)) security.CertificateHashData {
		h := goodHash
		f(&h)
		return h
	}
	goodFirmware := security.Firmware{Location: "https://fw.example.com/x1.bin", RetrieveDateTime: now, SigningCertificate: p.cert, Signature: "c2lnbmF0dXJl"}
	withFirmware := func(f func(*security.Firmware)) security.Firmware { fw := goodFirmware; f(&fw); return fw }
	goodLog := security.LogParameters{RemoteLocation: "ftp://logs.example.com/up"}
	long := func(n int) string { return strings.Repeat("a", n+1) }

	tests := []struct {
		name  string
		v     validator
		field string
	}{
		// SignCertificate
		{"SignCertificate ok", security.SignCertificateRequest{CSR: p.csr}, ""},
		{"SignCertificate empty", security.SignCertificateRequest{}, "csr"},
		{"SignCertificate not PEM", security.SignCertificateRequest{CSR: "not pem"}, "csr"},
		{"SignCertificate certificate instead of CSR", security.SignCertificateRequest{CSR: p.cert}, "csr"},
		{"SignCertificate too long", security.SignCertificateRequest{CSR: long(5500)}, "csr"},
		{"SignCertificate trailing data", security.SignCertificateRequest{CSR: p.csr + "garbage"}, "csr"},
		{"SignCertificate two requests", security.SignCertificateRequest{CSR: p.csr + p.csr}, "csr"},
		{"SignCertificate legacy block name", security.SignCertificateRequest{CSR: strings.ReplaceAll(p.csr, "CERTIFICATE REQUEST", "NEW CERTIFICATE REQUEST")}, ""},
		{"SignCertificate.conf", security.SignCertificateConfirmation{Status: security.GenericStatusAccepted}, ""},
		{"SignCertificate.conf missing", security.SignCertificateConfirmation{}, "status"},

		// CertificateSigned
		{"CertificateSigned ok", security.CertificateSignedRequest{CertificateChain: p.cert}, ""},
		{"CertificateSigned chain", security.CertificateSignedRequest{CertificateChain: p.cert + "\n" + p.cert2}, ""},
		{"CertificateSigned empty", security.CertificateSignedRequest{}, "certificateChain"},
		{"CertificateSigned CSR instead", security.CertificateSignedRequest{CertificateChain: p.csr}, "certificateChain"},
		{"CertificateSigned too long", security.CertificateSignedRequest{CertificateChain: long(10000)}, "certificateChain"},
		{"CertificateSigned.conf", security.CertificateSignedConfirmation{Status: security.CertificateSignedStatusRejected}, ""},
		{"CertificateSigned.conf bad", security.CertificateSignedConfirmation{Status: "Maybe"}, "status"},

		// InstallCertificate
		{"InstallCertificate ok", security.InstallCertificateRequest{CertificateType: security.CertificateUseCentralSystemRootCertificate, Certificate: p.cert}, ""},
		{"InstallCertificate type", security.InstallCertificateRequest{Certificate: p.cert}, "certificateType"},
		{"InstallCertificate certificate", security.InstallCertificateRequest{CertificateType: security.CertificateUseManufacturerRootCertificate}, "certificate"},
		{"InstallCertificate chain not allowed", security.InstallCertificateRequest{CertificateType: security.CertificateUseManufacturerRootCertificate, Certificate: p.cert + p.cert2}, "certificate"},
		{"InstallCertificate.conf", security.InstallCertificateConfirmation{Status: security.CertificateStatusFailed}, ""},
		{"InstallCertificate.conf missing", security.InstallCertificateConfirmation{}, "status"},

		// DeleteCertificate and hash data
		{"DeleteCertificate ok", security.DeleteCertificateRequest{CertificateHashData: goodHash}, ""},
		{"hash algorithm", security.DeleteCertificateRequest{CertificateHashData: withHash(func(h *security.CertificateHashData) { h.HashAlgorithm = "MD5" })}, "certificateHashData.hashAlgorithm"},
		{"issuer name hash empty", security.DeleteCertificateRequest{CertificateHashData: withHash(func(h *security.CertificateHashData) { h.IssuerNameHash = "" })}, "certificateHashData.issuerNameHash"},
		{"issuer name hash not hex", security.DeleteCertificateRequest{CertificateHashData: withHash(func(h *security.CertificateHashData) { h.IssuerNameHash = "xyz1" })}, "certificateHashData.issuerNameHash"},
		{"issuer key hash odd digits", security.DeleteCertificateRequest{CertificateHashData: withHash(func(h *security.CertificateHashData) { h.IssuerKeyHash = "abc" })}, "certificateHashData.issuerKeyHash"},
		{"issuer key hash too long", security.DeleteCertificateRequest{CertificateHashData: withHash(func(h *security.CertificateHashData) { h.IssuerKeyHash = strings.Repeat("ab", 65) })}, "certificateHashData.issuerKeyHash"},
		{"serial number not hex", security.DeleteCertificateRequest{CertificateHashData: withHash(func(h *security.CertificateHashData) { h.SerialNumber = "0xFFF" })}, "certificateHashData.serialNumber"},
		{"serial number empty", security.DeleteCertificateRequest{CertificateHashData: withHash(func(h *security.CertificateHashData) { h.SerialNumber = "" })}, "certificateHashData.serialNumber"},
		{"serial number may be odd length and lower case", security.DeleteCertificateRequest{CertificateHashData: withHash(func(h *security.CertificateHashData) { h.SerialNumber = "fff" })}, ""},
		{"serial number leading zeros are tolerated", security.DeleteCertificateRequest{CertificateHashData: withHash(func(h *security.CertificateHashData) { h.SerialNumber = "00FFF" })}, ""},
		{"DeleteCertificate.conf", security.DeleteCertificateConfirmation{Status: security.DeleteCertificateStatusNotFound}, ""},
		{"DeleteCertificate.conf missing", security.DeleteCertificateConfirmation{}, "status"},

		// GetInstalledCertificateIds
		{"GetInstalledCertificateIds ok", security.GetInstalledCertificateIdsRequest{CertificateType: security.CertificateUseManufacturerRootCertificate}, ""},
		{"GetInstalledCertificateIds type", security.GetInstalledCertificateIdsRequest{}, "certificateType"},
		{"GetInstalledCertificateIds.conf none", security.GetInstalledCertificateIdsConfirmation{Status: security.GetInstalledCertificateStatusNotFound}, ""},
		{"GetInstalledCertificateIds.conf list", security.GetInstalledCertificateIdsConfirmation{Status: security.GetInstalledCertificateStatusAccepted, CertificateHashData: []security.CertificateHashData{goodHash, goodHash}}, ""},
		{"GetInstalledCertificateIds.conf status", security.GetInstalledCertificateIdsConfirmation{}, "status"},
		{"GetInstalledCertificateIds.conf bad entry", security.GetInstalledCertificateIdsConfirmation{Status: security.GetInstalledCertificateStatusAccepted,
			CertificateHashData: []security.CertificateHashData{goodHash, {}}}, "certificateHashData[1].hashAlgorithm"},

		// GetLog and LogStatusNotification
		{"GetLog ok", security.GetLogRequest{LogType: security.LogTypeSecurityLog, RequestID: 0, Log: goodLog}, ""},
		{"GetLog type", security.GetLogRequest{Log: goodLog}, "logType"},
		{"GetLog retries", security.GetLogRequest{LogType: security.LogTypeSecurityLog, Log: goodLog, Retries: intp(-1)}, "retries"},
		{"GetLog retryInterval", security.GetLogRequest{LogType: security.LogTypeSecurityLog, Log: goodLog, RetryInterval: intp(-1)}, "retryInterval"},
		{"GetLog no location", security.GetLogRequest{LogType: security.LogTypeSecurityLog}, "log.remoteLocation"},
		{"GetLog relative location", security.GetLogRequest{LogType: security.LogTypeSecurityLog, Log: security.LogParameters{RemoteLocation: "/up"}}, "log.remoteLocation"},
		{"GetLog long location", security.GetLogRequest{LogType: security.LogTypeSecurityLog, Log: security.LogParameters{RemoteLocation: "ftp://h/" + long(512)}}, "log.remoteLocation"},
		{"GetLog.conf", security.GetLogConfirmation{Status: security.LogStatusAccepted, Filename: "log.zip"}, ""},
		{"GetLog.conf missing", security.GetLogConfirmation{}, "status"},
		{"GetLog.conf filename", security.GetLogConfirmation{Status: security.LogStatusAccepted, Filename: long(255)}, "filename"},
		{"LogStatusNotification ok", security.LogStatusNotificationRequest{Status: security.UploadLogStatusIdle, RequestID: intp(1)}, ""},
		{"LogStatusNotification missing", security.LogStatusNotificationRequest{}, "status"},
		{"LogStatusNotification.conf", security.LogStatusNotificationConfirmation{}, ""},

		// SignedUpdateFirmware and its notification
		{"SignedUpdateFirmware ok", security.SignedUpdateFirmwareRequest{RequestID: 1, Firmware: goodFirmware}, ""},
		{"SignedUpdateFirmware retries", security.SignedUpdateFirmwareRequest{Retries: intp(-1), Firmware: goodFirmware}, "retries"},
		{"SignedUpdateFirmware retryInterval", security.SignedUpdateFirmwareRequest{RetryInterval: intp(-1), Firmware: goodFirmware}, "retryInterval"},
		{"firmware location", security.SignedUpdateFirmwareRequest{Firmware: withFirmware(func(f *security.Firmware) { f.Location = "" })}, "firmware.location"},
		{"firmware relative location", security.SignedUpdateFirmwareRequest{Firmware: withFirmware(func(f *security.Firmware) { f.Location = "fw.bin" })}, "firmware.location"},
		{"firmware long location", security.SignedUpdateFirmwareRequest{Firmware: withFirmware(func(f *security.Firmware) { f.Location = "https://h/" + long(512) })}, "firmware.location"},
		{"firmware retrieve date", security.SignedUpdateFirmwareRequest{Firmware: withFirmware(func(f *security.Firmware) { f.RetrieveDateTime = types.DateTime{} })}, "firmware.retrieveDateTime"},
		{"firmware certificate", security.SignedUpdateFirmwareRequest{Firmware: withFirmware(func(f *security.Firmware) { f.SigningCertificate = "" })}, "firmware.signingCertificate"},
		{"firmware signature missing", security.SignedUpdateFirmwareRequest{Firmware: withFirmware(func(f *security.Firmware) { f.Signature = "" })}, "firmware.signature"},
		{"firmware signature not base64", security.SignedUpdateFirmwareRequest{Firmware: withFirmware(func(f *security.Firmware) { f.Signature = "not base64!" })}, "firmware.signature"},
		{"firmware signature too long", security.SignedUpdateFirmwareRequest{Firmware: withFirmware(func(f *security.Firmware) { f.Signature = strings.Repeat("QUJD", 201) })}, "firmware.signature"},
		{"SignedUpdateFirmware.conf", security.SignedUpdateFirmwareConfirmation{Status: security.SignedUpdateFirmwareStatusInvalidCertificate}, ""},
		{"SignedUpdateFirmware.conf missing", security.SignedUpdateFirmwareConfirmation{}, "status"},
		{"SignedFirmwareStatusNotification ok", security.SignedFirmwareStatusNotificationRequest{Status: security.SignedFirmwareStatusSignatureVerified}, ""},
		{"SignedFirmwareStatusNotification bad", security.SignedFirmwareStatusNotificationRequest{Status: "Done"}, "status"},
		{"SignedFirmwareStatusNotification.conf", security.SignedFirmwareStatusNotificationConfirmation{}, ""},

		// SecurityEventNotification and ExtendedTriggerMessage
		{"SecurityEventNotification ok", security.SecurityEventNotificationRequest{Type: security.EventStartupOfTheDevice, Timestamp: now}, ""},
		{"SecurityEventNotification vendor event", security.SecurityEventNotificationRequest{Type: "VendorSpecificThing", Timestamp: now, TechInfo: "x"}, ""},
		{"SecurityEventNotification type", security.SecurityEventNotificationRequest{Timestamp: now}, "type"},
		{"SecurityEventNotification long type", security.SecurityEventNotificationRequest{Type: long(50), Timestamp: now}, "type"},
		{"SecurityEventNotification timestamp", security.SecurityEventNotificationRequest{Type: "x"}, "timestamp"},
		{"SecurityEventNotification techInfo", security.SecurityEventNotificationRequest{Type: "x", Timestamp: now, TechInfo: long(255)}, "techInfo"},
		{"SecurityEventNotification.conf", security.SecurityEventNotificationConfirmation{}, ""},
		{"ExtendedTriggerMessage ok", security.ExtendedTriggerMessageRequest{RequestedMessage: security.ExtendedMessageTriggerSignChargePointCertificate}, ""},
		{"ExtendedTriggerMessage with connector", security.ExtendedTriggerMessageRequest{RequestedMessage: security.ExtendedMessageTriggerMeterValues, ConnectorID: intp(2)}, ""},
		{"ExtendedTriggerMessage missing", security.ExtendedTriggerMessageRequest{}, "requestedMessage"},
		{"ExtendedTriggerMessage connector 0", security.ExtendedTriggerMessageRequest{RequestedMessage: security.ExtendedMessageTriggerHeartbeat, ConnectorID: intp(0)}, "connectorId"},
		{"ExtendedTriggerMessage.conf", security.ExtendedTriggerMessageConfirmation{Status: types.TriggerMessageStatusNotImplemented}, ""},
		{"ExtendedTriggerMessage.conf missing", security.ExtendedTriggerMessageConfirmation{}, "status"},
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
