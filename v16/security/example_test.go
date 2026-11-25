package security_test

import (
	"fmt"
	"strings"

	"github.com/foecum/go-ocpp-sdk/v16/security"
)

// The AuthorizationKey is the Basic-auth password of OCPP 1.6 security profiles
// 1 and 2: 16 to 20 bytes written as 32 to 40 hexadecimal digits.
func ExampleCheckAuthorizationKey() {
	fmt.Println(security.CheckAuthorizationKey(strings.Repeat("ab", 16)))
	fmt.Println(security.CheckAuthorizationKey("hunter2"))
	// Output:
	// <nil>
	// ocpp: invalid field "authorizationKey": must be 32 to 40 hexadecimal digits (16 to 20 bytes)
}

func ExampleGenerateAuthorizationKey() {
	key := security.GenerateAuthorizationKey()
	fmt.Println(len(key), security.CheckAuthorizationKey(key))
	// Output: 40 <nil>
}

// Critical security events should be pushed to the Central System.
func ExampleIsCriticalEvent() {
	fmt.Println(security.IsCriticalEvent(security.EventTamperDetectionActivated))
	fmt.Println(security.IsCriticalEvent(security.EventInvalidTLSVersion))
	// Output:
	// true
	// false
}

// Certificates and requests must be PEM encoded.
func ExampleSignCertificateRequest_Validate() {
	req := security.SignCertificateRequest{CSR: "not a PEM block"}
	fmt.Println(req.Validate())
	// Output: ocpp: invalid field "csr": must be PEM encoded
}
