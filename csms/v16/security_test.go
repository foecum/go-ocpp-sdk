package csms_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/chargepoint/v16"
	"github.com/foecum/go-ocpp-sdk/csms/v16"
	"github.com/foecum/go-ocpp-sdk/v16/security"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// testPEMs returns a certificate and a certificate request, generated now.
func testPEMs(t testing.TB) (cert, csr string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "CS001"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{Subject: pkix.Name{CommonName: "CS001"}}, key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER}))
}

// securityStation implements the security handler of a charge point and records
// what it was asked.
type securityStation struct {
	mu   sync.Mutex
	seen []string
}

func (s *securityStation) note(format string, args ...any) {
	s.mu.Lock()
	s.seen = append(s.seen, fmt.Sprintf(format, args...))
	s.mu.Unlock()
}

func (s *securityStation) CertificateSigned(_ context.Context, r *security.CertificateSignedRequest) (*security.CertificateSignedConfirmation, error) {
	s.note("CertificateSigned:%d", len(r.CertificateChain))
	return &security.CertificateSignedConfirmation{Status: security.CertificateSignedStatusAccepted}, nil
}
func (s *securityStation) DeleteCertificate(_ context.Context, r *security.DeleteCertificateRequest) (*security.DeleteCertificateConfirmation, error) {
	s.note("DeleteCertificate:%s", r.CertificateHashData.SerialNumber)
	return &security.DeleteCertificateConfirmation{Status: security.DeleteCertificateStatusNotFound}, nil
}
func (s *securityStation) ExtendedTriggerMessage(_ context.Context, r *security.ExtendedTriggerMessageRequest) (*security.ExtendedTriggerMessageConfirmation, error) {
	s.note("ExtendedTriggerMessage:%s", r.RequestedMessage)
	return &security.ExtendedTriggerMessageConfirmation{Status: types.TriggerMessageStatusAccepted}, nil
}
func (s *securityStation) GetInstalledCertificateIds(_ context.Context, r *security.GetInstalledCertificateIdsRequest) (*security.GetInstalledCertificateIdsConfirmation, error) {
	s.note("GetInstalledCertificateIds:%s", r.CertificateType)
	return &security.GetInstalledCertificateIdsConfirmation{Status: security.GetInstalledCertificateStatusNotFound}, nil
}
func (s *securityStation) GetLog(_ context.Context, r *security.GetLogRequest) (*security.GetLogConfirmation, error) {
	s.note("GetLog:%s:%d", r.LogType, r.RequestID)
	return &security.GetLogConfirmation{Status: security.LogStatusAccepted, Filename: "log.zip"}, nil
}
func (s *securityStation) InstallCertificate(_ context.Context, r *security.InstallCertificateRequest) (*security.InstallCertificateConfirmation, error) {
	s.note("InstallCertificate:%s", r.CertificateType)
	return &security.InstallCertificateConfirmation{Status: security.CertificateStatusAccepted}, nil
}
func (s *securityStation) SignedUpdateFirmware(_ context.Context, r *security.SignedUpdateFirmwareRequest) (*security.SignedUpdateFirmwareConfirmation, error) {
	s.note("SignedUpdateFirmware:%d", r.RequestID)
	return &security.SignedUpdateFirmwareConfirmation{Status: security.SignedUpdateFirmwareStatusAccepted}, nil
}

// TestEverySecurityCallFromTheCSMS exercises the typed calls of the security
// extension, from the Central System to a station.
func TestEverySecurityCallFromTheCSMS(t *testing.T) {
	station := &securityStation{}
	srv, _, url := startCSMS(t, nil)
	startStation(t, url, "CS1", func(c *chargepoint.Config) { c.Security = station })
	cp := lookup(t, srv, "CS1")
	ctx := context.Background()
	cert, _ := testPEMs(t)
	hash := strings.Repeat("ab", 32)

	if r, err := cp.CertificateSigned(ctx, security.CertificateSignedRequest{CertificateChain: cert}); err != nil || r.Status != security.CertificateSignedStatusAccepted {
		t.Errorf("CertificateSigned: %v %v", r, err)
	}
	if r, err := cp.DeleteCertificate(ctx, security.DeleteCertificateRequest{CertificateHashData: security.CertificateHashData{
		HashAlgorithm: security.HashAlgorithmSHA256, IssuerNameHash: hash, IssuerKeyHash: hash, SerialNumber: "FFF"}}); err != nil || r.Status != security.DeleteCertificateStatusNotFound {
		t.Errorf("DeleteCertificate: %v %v", r, err)
	}
	if r, err := cp.ExtendedTriggerMessage(ctx, security.ExtendedTriggerMessageRequest{RequestedMessage: security.ExtendedMessageTriggerSignChargePointCertificate}); err != nil || r.Status != types.TriggerMessageStatusAccepted {
		t.Errorf("ExtendedTriggerMessage: %v %v", r, err)
	}
	if r, err := cp.GetInstalledCertificateIds(ctx, security.GetInstalledCertificateIdsRequest{CertificateType: security.CertificateUseCentralSystemRootCertificate}); err != nil || r.Status != security.GetInstalledCertificateStatusNotFound {
		t.Errorf("GetInstalledCertificateIds: %v %v", r, err)
	}
	if r, err := cp.GetLog(ctx, security.GetLogRequest{LogType: security.LogTypeSecurityLog, RequestID: 9, Log: security.LogParameters{RemoteLocation: "ftp://h/up"}}); err != nil || r.Filename != "log.zip" {
		t.Errorf("GetLog: %v %v", r, err)
	}
	if r, err := cp.InstallCertificate(ctx, security.InstallCertificateRequest{CertificateType: security.CertificateUseManufacturerRootCertificate, Certificate: cert}); err != nil || r.Status != security.CertificateStatusAccepted {
		t.Errorf("InstallCertificate: %v %v", r, err)
	}
	if r, err := cp.SignedUpdateFirmware(ctx, security.SignedUpdateFirmwareRequest{RequestID: 5, Firmware: security.Firmware{
		Location: "https://fw/x.bin", RetrieveDateTime: types.Now(), SigningCertificate: cert, Signature: "c2ln"}}); err != nil || r.Status != security.SignedUpdateFirmwareStatusAccepted {
		t.Errorf("SignedUpdateFirmware: %v %v", r, err)
	}
	want := fmt.Sprintf("[CertificateSigned:%d DeleteCertificate:FFF ExtendedTriggerMessage:SignChargePointCertificate "+
		"GetInstalledCertificateIds:CentralSystemRootCertificate GetLog:SecurityLog:9 InstallCertificate:ManufacturerRootCertificate SignedUpdateFirmware:5]", len(cert))
	station.mu.Lock()
	defer station.mu.Unlock()
	if fmt.Sprint(station.seen) != want {
		t.Errorf("station saw\n%v\nwant\n%s", station.seen, want)
	}
	// Invalid requests (here: not PEM) are stopped before sending.
	if _, err := cp.InstallCertificate(ctx, security.InstallCertificateRequest{CertificateType: security.CertificateUseManufacturerRootCertificate, Certificate: "nope"}); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("got %v", err)
	}
}

// securityHandler records the notifications a station sends and signs CSRs.
type securityHandler struct {
	csms.UnimplementedSecurityHandler
	mu   sync.Mutex
	seen []string
}

func (h *securityHandler) note(format string, args ...any) {
	h.mu.Lock()
	h.seen = append(h.seen, fmt.Sprintf(format, args...))
	h.mu.Unlock()
}

func (h *securityHandler) LogStatusNotification(_ context.Context, cp *csms.ChargePoint, r *security.LogStatusNotificationRequest) (*security.LogStatusNotificationConfirmation, error) {
	h.note("log:%s", r.Status)
	return &security.LogStatusNotificationConfirmation{}, nil
}
func (h *securityHandler) SecurityEventNotification(_ context.Context, cp *csms.ChargePoint, r *security.SecurityEventNotificationRequest) (*security.SecurityEventNotificationConfirmation, error) {
	h.note("event:%s:critical=%v", r.Type, security.IsCriticalEvent(r.Type))
	return &security.SecurityEventNotificationConfirmation{}, nil
}
func (h *securityHandler) SignCertificate(_ context.Context, cp *csms.ChargePoint, r *security.SignCertificateRequest) (*security.SignCertificateConfirmation, error) {
	h.note("sign:%d", len(r.CSR))
	return &security.SignCertificateConfirmation{Status: security.GenericStatusAccepted}, nil
}
func (h *securityHandler) SignedFirmwareStatusNotification(_ context.Context, cp *csms.ChargePoint, r *security.SignedFirmwareStatusNotificationRequest) (*security.SignedFirmwareStatusNotificationConfirmation, error) {
	h.note("firmware:%s", r.Status)
	return &security.SignedFirmwareStatusNotificationConfirmation{}, nil
}

func TestSecurityNotificationsReachTheHandler(t *testing.T) {
	h := &securityHandler{}
	_, _, url := startCSMS(t, func(c *csms.Config) { c.Security = h })
	cp := startStation(t, url, "CS1", nil)
	ctx := context.Background()
	_, csr := testPEMs(t)
	if _, err := cp.SecurityEventNotification(ctx, security.SecurityEventNotificationRequest{Type: security.EventStartupOfTheDevice, Timestamp: types.Now()}); err != nil {
		t.Fatal(err)
	}
	if r, err := cp.SignCertificate(ctx, security.SignCertificateRequest{CSR: csr}); err != nil || r.Status != security.GenericStatusAccepted {
		t.Fatalf("%v %v", r, err)
	}
	if _, err := cp.LogStatusNotification(ctx, security.LogStatusNotificationRequest{Status: security.UploadLogStatusUploading}); err != nil {
		t.Fatal(err)
	}
	if _, err := cp.SignedFirmwareStatusNotification(ctx, security.SignedFirmwareStatusNotificationRequest{Status: security.SignedFirmwareStatusInstalled}); err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("[event:StartupOfTheDevice:critical=true sign:%d log:Uploading firmware:Installed]", len(csr))
	h.mu.Lock()
	defer h.mu.Unlock()
	if fmt.Sprint(h.seen) != want {
		t.Errorf("handler saw\n%v\nwant\n%s", h.seen, want)
	}
	if _, err := cp.SignCertificate(ctx, security.SignCertificateRequest{CSR: "not pem"}); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("got %v", err)
	}
}

func TestSecurityNotSupportedWithoutAHandler(t *testing.T) {
	cert, csr := testPEMs(t)
	_ = cert
	for name, sec := range map[string]csms.SecurityHandler{"nil handler": nil, "unimplemented embed": csms.UnimplementedSecurityHandler{}} {
		_, _, url := startCSMS(t, func(c *csms.Config) { c.Security = sec })
		cp := startStation(t, url, "CS1", nil)
		ctx := context.Background()
		for call, f := range map[string]func() error{
			"SecurityEventNotification": func() error {
				_, e := cp.SecurityEventNotification(ctx, security.SecurityEventNotificationRequest{Type: "x", Timestamp: types.Now()})
				return e
			},
			"SignCertificate": func() error { _, e := cp.SignCertificate(ctx, security.SignCertificateRequest{CSR: csr}); return e },
			"LogStatusNotification": func() error {
				_, e := cp.LogStatusNotification(ctx, security.LogStatusNotificationRequest{Status: security.UploadLogStatusIdle})
				return e
			},
			"SignedFirmwareStatusNotification": func() error {
				_, e := cp.SignedFirmwareStatusNotification(ctx, security.SignedFirmwareStatusNotificationRequest{Status: security.SignedFirmwareStatusIdle})
				return e
			},
		} {
			if ce := callError(t, f()); ce.Code != ocpp.NotSupported {
				t.Errorf("%s / %s: %s", name, call, ce.Code)
			}
		}
	}
}

func TestStationsWithoutSecurityHandlersAnswerNotSupported(t *testing.T) {
	cert, _ := testPEMs(t)
	hash := strings.Repeat("cd", 32)
	srv, _, url := startCSMS(t, nil)
	startStation(t, url, "CS1", nil) // no handler
	startStation(t, url, "CS2", func(c *chargepoint.Config) { c.Security = chargepoint.UnimplementedSecurityHandler{} })
	ctx := context.Background()
	for _, id := range []string{"CS1", "CS2"} {
		cp := lookup(t, srv, id)
		for name, f := range map[string]func() error{
			"CertificateSigned": func() error {
				_, e := cp.CertificateSigned(ctx, security.CertificateSignedRequest{CertificateChain: cert})
				return e
			},
			"DeleteCertificate": func() error {
				_, e := cp.DeleteCertificate(ctx, security.DeleteCertificateRequest{CertificateHashData: security.CertificateHashData{
					HashAlgorithm: security.HashAlgorithmSHA256, IssuerNameHash: hash, IssuerKeyHash: hash, SerialNumber: "1"}})
				return e
			},
			"ExtendedTriggerMessage": func() error {
				_, e := cp.ExtendedTriggerMessage(ctx, security.ExtendedTriggerMessageRequest{RequestedMessage: security.ExtendedMessageTriggerHeartbeat})
				return e
			},
			"GetInstalledCertificateIds": func() error {
				_, e := cp.GetInstalledCertificateIds(ctx, security.GetInstalledCertificateIdsRequest{CertificateType: security.CertificateUseCentralSystemRootCertificate})
				return e
			},
			"GetLog": func() error {
				_, e := cp.GetLog(ctx, security.GetLogRequest{LogType: security.LogTypeDiagnosticsLog, Log: security.LogParameters{RemoteLocation: "ftp://h/u"}})
				return e
			},
			"InstallCertificate": func() error {
				_, e := cp.InstallCertificate(ctx, security.InstallCertificateRequest{CertificateType: security.CertificateUseCentralSystemRootCertificate, Certificate: cert})
				return e
			},
			"SignedUpdateFirmware": func() error {
				_, e := cp.SignedUpdateFirmware(ctx, security.SignedUpdateFirmwareRequest{Firmware: security.Firmware{
					Location: "https://h/f", RetrieveDateTime: types.Now(), SigningCertificate: cert, Signature: "c2ln"}})
				return e
			},
		} {
			if ce := callError(t, f()); ce.Code != ocpp.NotSupported {
				t.Errorf("%s / %s: %s", id, name, ce.Code)
			}
		}
	}
}

func TestCSMSRejectsCSMSOnlySecurityActionsFromAStation(t *testing.T) {
	_, _, url := startCSMS(t, nil)
	c := rawStation(t, url, "RAW3")
	if ce := exchange(t, c, `[2,"g1","GetLog",{"logType":"SecurityLog","requestId":1,"log":{"remoteLocation":"ftp://h/u"}}]`); ce.Code != ocpp.NotSupported {
		t.Errorf("GetLog from a station: %+v", ce)
	}
	if ce := exchange(t, c, `[2,"g2","SignCertificate",{"csr":"not pem"}]`); ce.Code != ocpp.PropertyConstraintViolation {
		t.Errorf("bad CSR: %+v", ce)
	}
	bad, _ := json.Marshal(map[string]any{"type": strings.Repeat("x", 51), "timestamp": "2026-01-02T03:04:05Z"})
	if ce := exchange(t, c, `[2,"g3","SecurityEventNotification",`+string(bad)+`]`); ce.Code != ocpp.PropertyConstraintViolation {
		t.Errorf("long event type: %+v", ce)
	}
}
