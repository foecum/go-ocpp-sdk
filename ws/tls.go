package ws

import (
	"crypto/tls"
	"crypto/x509"
)

// CipherSuites are the TLS 1.2 cipher suites allowed for OCPP: the four suites
// the 2.0.1 specification requires (A00.FR.318/319) plus the equivalent ECDHE
// RSA suites, which also give forward secrecy. TLS 1.3 suites are not
// configurable in Go and are always enabled.
func CipherSuites() []uint16 {
	return []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
	}
}

// ServerTLSConfig returns a TLS configuration for a CSMS: TLS 1.2 or newer
// (A00.FR.313) and the allowed cipher suites. If clientCAs is not nil, client
// certificates are required and verified against it (security profile 3);
// otherwise none are requested (profile 2).
func ServerTLSConfig(cert tls.Certificate, clientCAs *x509.CertPool) *tls.Config {
	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: CipherSuites(),
	}
	if clientCAs != nil {
		cfg.ClientCAs = clientCAs
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return cfg
}

// ClientTLSConfig returns a TLS configuration for a charge point: TLS 1.2 or
// newer and the allowed cipher suites. rootCAs verifies the CSMS certificate
// (nil uses the system roots). clientCert, if not nil, is presented to the CSMS
// (security profile 3).
//
// The CSMS certificate is verified against the server name using its
// subjectAltName, as Go requires; certificates that only carry a commonName are
// rejected.
func ClientTLSConfig(rootCAs *x509.CertPool, clientCert *tls.Certificate) *tls.Config {
	cfg := &tls.Config{
		RootCAs:      rootCAs,
		MinVersion:   tls.VersionTLS12,
		CipherSuites: CipherSuites(),
	}
	if clientCert != nil {
		cfg.Certificates = []tls.Certificate{*clientCert}
	}
	return cfg
}
