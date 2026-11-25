package security_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/foecum/go-ocpp-sdk/internal/schematest"
)

// pems holds a real certificate and certificate request, generated once per
// test run, so no key material lives in the repository.
type pems struct{ cert, cert2, csr string }

func makePEMs(t testing.TB) pems {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(4095), Subject: pkix.Name{CommonName: "CS001", Organization: []string{"Test CPO"}},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{Subject: pkix.Name{CommonName: "CS001"}}, key)
	if err != nil {
		t.Fatal(err)
	}
	return pems{cert: cert, cert2: cert, csr: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER}))}
}

func js(t testing.TB, v string) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

const hash64 = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func hashData(alg string) string {
	return `{"hashAlgorithm":"` + alg + `","issuerNameHash":"` + hash64 + `","issuerKeyHash":"` + hash64 + `","serialNumber":"FFF"}`
}

func buildSamples(t testing.TB) map[string]schematest.Sample {
	p := makePEMs(t)
	cert, chain, csr := js(t, p.cert), js(t, p.cert+p.cert2), js(t, p.csr)
	const ts = `"2026-01-02T03:04:05Z"`
	return map[string]schematest.Sample{
		"CertificateSigned": {
			ReqMin: `{"certificateChain":` + cert + `}`, ReqFull: `{"certificateChain":` + chain + `}`,
			ConfMin: `{"status":"Accepted"}`, ConfFull: `{"status":"Rejected"}`,
		},
		"DeleteCertificate": {
			ReqMin: `{"certificateHashData":` + hashData("SHA256") + `}`, ReqFull: `{"certificateHashData":` + hashData("SHA512") + `}`,
			ConfMin: `{"status":"Accepted"}`, ConfFull: `{"status":"NotFound"}`,
		},
		"ExtendedTriggerMessage": {
			ReqMin: `{"requestedMessage":"SignChargePointCertificate"}`, ReqFull: `{"requestedMessage":"LogStatusNotification","connectorId":1}`,
			ConfMin: `{"status":"Accepted"}`, ConfFull: `{"status":"NotImplemented"}`,
		},
		"GetInstalledCertificateIds": {
			ReqMin: `{"certificateType":"CentralSystemRootCertificate"}`, ReqFull: `{"certificateType":"ManufacturerRootCertificate"}`,
			ConfMin: `{"status":"NotFound"}`, ConfFull: `{"status":"Accepted","certificateHashData":[` + hashData("SHA384") + `]}`,
		},
		"GetLog": {
			ReqMin: `{"logType":"SecurityLog","requestId":1,"log":{"remoteLocation":"ftp://logs.example.com/up"}}`,
			ReqFull: `{"logType":"DiagnosticsLog","requestId":2,"retries":3,"retryInterval":60,"log":{"remoteLocation":"https://logs.example.com/up",` +
				`"oldestTimestamp":"2026-01-01T00:00:00Z","latestTimestamp":"2026-01-02T00:00:00.500Z"}}`,
			ConfMin: `{"status":"Accepted"}`, ConfFull: `{"status":"AcceptedCanceled","filename":"log-2026-01-02.zip"}`,
		},
		"InstallCertificate": {
			ReqMin:  `{"certificateType":"CentralSystemRootCertificate","certificate":` + cert + `}`,
			ReqFull: `{"certificateType":"ManufacturerRootCertificate","certificate":` + cert + `}`,
			ConfMin: `{"status":"Accepted"}`, ConfFull: `{"status":"Failed"}`,
		},
		"LogStatusNotification": {
			ReqMin: `{"status":"Uploading"}`, ReqFull: `{"status":"Uploaded","requestId":1}`,
			ConfMin: `{}`, ConfFull: `{}`,
		},
		"SecurityEventNotification": {
			ReqMin:  `{"type":"StartupOfTheDevice","timestamp":` + ts + `}`,
			ReqFull: `{"type":"TamperDetectionActivated","timestamp":"2026-01-02T03:04:05.250Z","techInfo":"cover opened"}`,
			ConfMin: `{}`, ConfFull: `{}`,
		},
		"SignCertificate": {
			ReqMin: `{"csr":` + csr + `}`, ReqFull: `{"csr":` + csr + `}`,
			ConfMin: `{"status":"Accepted"}`, ConfFull: `{"status":"Rejected"}`,
		},
		"SignedFirmwareStatusNotification": {
			ReqMin: `{"status":"Downloading"}`, ReqFull: `{"status":"InvalidSignature","requestId":3}`,
			ConfMin: `{}`, ConfFull: `{}`,
		},
		"SignedUpdateFirmware": {
			ReqMin: `{"requestId":3,"firmware":{"location":"https://fw.example.com/x1.bin","retrieveDateTime":` + ts +
				`,"signingCertificate":` + cert + `,"signature":"c2lnbmF0dXJl"}}`,
			ReqFull: `{"retries":2,"retryInterval":300,"requestId":4,"firmware":{"location":"https://fw.example.com/x1.bin",` +
				`"retrieveDateTime":` + ts + `,"installDateTime":"2026-03-01T04:00:00Z","signingCertificate":` + cert + `,"signature":"c2lnbmF0dXJl"}}`,
			ConfMin: `{"status":"Accepted"}`, ConfFull: `{"status":"RevokedCertificate"}`,
		},
	}
}

func TestSampleCertificatesAreWellFormed(t *testing.T) {
	p := makePEMs(t)
	if !strings.HasPrefix(p.cert, "-----BEGIN CERTIFICATE-----") || !strings.HasPrefix(p.csr, "-----BEGIN CERTIFICATE REQUEST-----") {
		t.Fatal("generated PEM looks wrong")
	}
}
