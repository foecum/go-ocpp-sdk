package security

import (
	"crypto/rand"
	"encoding/hex"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

// The AuthorizationKey is the password of HTTP Basic authentication (security
// profiles 1 and 2): at least 16 bytes, at most 20 bytes, represented as 32 to
// 40 hexadecimal digits. The charge point sends it as that text, so
// ws.BasicAuthenticator needs no special handling.
const (
	minAuthorizationKeyDigits = 32
	maxAuthorizationKeyDigits = 40
)

// CheckAuthorizationKey validates an AuthorizationKey: 32 to 40 hexadecimal
// digits (16 to 20 bytes).
func CheckAuthorizationKey(key string) error {
	if len(key) < minAuthorizationKeyDigits || len(key) > maxAuthorizationKeyDigits || len(key)%2 != 0 {
		return ocpp.Invalid("authorizationKey", "must be 32 to 40 hexadecimal digits (16 to 20 bytes)")
	}
	if _, err := hex.DecodeString(key); err != nil {
		return ocpp.Invalid("authorizationKey", "must be hexadecimal")
	}
	return nil
}

// GenerateAuthorizationKey returns a new random AuthorizationKey: 20 random
// bytes as 40 hexadecimal digits, the longest the specification allows.
func GenerateAuthorizationKey() string {
	var b [20]byte
	// As of Go 1.24 crypto/rand.Read never returns an error.
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
