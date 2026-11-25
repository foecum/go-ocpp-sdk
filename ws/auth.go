package ws

import (
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Authentication errors. An Authenticator returns one of these (or an error
// wrapping one) to choose the HTTP status of the rejected handshake.
var (
	// ErrUnauthorized rejects with 401 and a Basic challenge.
	ErrUnauthorized = errors.New("ws: unauthorized")
	// ErrUnknownID rejects with 404: the charge point identity is not known
	// (OCPP-J 2.0.1 §3.2).
	ErrUnknownID = errors.New("ws: unknown charge point")
	// ErrForbidden rejects with 403.
	ErrForbidden = errors.New("ws: forbidden")
)

// Authenticator decides whether a handshake request from charge point id may
// proceed. Return nil to accept, or one of ErrUnauthorized, ErrUnknownID or
// ErrForbidden (any other error is treated as ErrUnauthorized). It is called
// before the WebSocket upgrade.
type Authenticator func(r *http.Request, id string) error

// BasicAuthenticator implements security profiles 1 and 2: HTTP Basic
// authentication where the username is the charge point identity.
//
// lookup returns the expected password for an identity, or ok=false if the
// identity is unknown (answered with 404). The password is compared in
// constant time. Per OCPP 2.0.1 (A00.FR.205) it is the UTF-8 string as sent.
func BasicAuthenticator(lookup func(id string) (password string, ok bool)) Authenticator {
	return func(r *http.Request, id string) error {
		user, pass, ok := r.BasicAuth()
		if !ok || user != id {
			return ErrUnauthorized
		}
		want, known := lookup(id)
		if !known {
			return ErrUnknownID
		}
		if !equalConstantTime(pass, want) {
			return ErrUnauthorized
		}
		return nil
	}
}

func equalConstantTime(a, b string) bool {
	ha, hb := sha256.Sum256([]byte(a)), sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(ha[:], hb[:]) == 1
}

// ClientCertOptions tune ClientCertAuthenticator.
type ClientCertOptions struct {
	// Organization, if set, must appear in the certificate's subject O
	// (organizationName), identifying the charge point operator (A00.FR.404).
	Organization string
	// Identity checks that the certificate belongs to charge point id. The
	// default requires the subject CN (commonName) to equal id (A00.FR.405,
	// A00.FR.511).
	Identity func(cert *x509.Certificate, id string) error
}

// ClientCertAuthenticator implements the identity checks of security profile 3.
// The TLS layer must already have verified the certificate chain, which
// ServerTLSConfig does when given a client CA pool.
func ClientCertAuthenticator(opts ClientCertOptions) Authenticator {
	identity := opts.Identity
	if identity == nil {
		identity = func(cert *x509.Certificate, id string) error {
			if cert.Subject.CommonName != id {
				return fmt.Errorf("certificate CN %q does not match identity", cert.Subject.CommonName)
			}
			return nil
		}
	}
	return func(r *http.Request, id string) error {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			return ErrUnauthorized
		}
		cert := r.TLS.PeerCertificates[0]
		if err := identity(cert, id); err != nil {
			return fmt.Errorf("%w: %v", ErrForbidden, err)
		}
		if opts.Organization != "" && !containsString(cert.Subject.Organization, opts.Organization) {
			return fmt.Errorf("%w: certificate organization does not match", ErrForbidden)
		}
		return nil
	}
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if strings.Contains(x, s) {
			return true
		}
	}
	return false
}

// ChainAuthenticators runs the authenticators in order and stops at the first
// error. Use it to combine checks.
func ChainAuthenticators(auths ...Authenticator) Authenticator {
	return func(r *http.Request, id string) error {
		for _, a := range auths {
			if err := a(r, id); err != nil {
				return err
			}
		}
		return nil
	}
}
