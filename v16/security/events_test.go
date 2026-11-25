package security_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/foecum/go-ocpp-sdk/v16/security"
)

func TestSecurityEvents(t *testing.T) {
	critical := []string{security.EventFirmwareUpdated, security.EventSettingSystemTime, security.EventStartupOfTheDevice,
		security.EventResetOrReboot, security.EventSecurityLogWasCleared, security.EventMemoryExhaustion, security.EventTamperDetectionActivated}
	notCritical := []string{security.EventFailedToAuthenticateAtCentral, security.EventCentralSystemFailedToAuthenticate,
		security.EventReconfigurationOfSecurityParams, security.EventInvalidMessages, security.EventAttemptedReplayAttacks,
		security.EventInvalidFirmwareSignature, security.EventInvalidFirmwareSigningCertificate, security.EventInvalidCentralSystemCertificate,
		security.EventInvalidChargePointCertificate, security.EventInvalidTLSVersion, security.EventInvalidTLSCipherSuite}
	for _, e := range critical {
		if !security.IsKnownEvent(e) || !security.IsCriticalEvent(e) {
			t.Errorf("%s should be known and critical", e)
		}
	}
	for _, e := range notCritical {
		if !security.IsKnownEvent(e) || security.IsCriticalEvent(e) {
			t.Errorf("%s should be known and not critical", e)
		}
	}
	if len(critical)+len(notCritical) != 18 {
		t.Errorf("the whitepaper lists 18 events, have %d", len(critical)+len(notCritical))
	}
	if security.IsKnownEvent("VendorSpecificThing") || security.IsCriticalEvent("VendorSpecificThing") {
		t.Error("unknown events are neither known nor critical")
	}
}

func TestAuthorizationKey(t *testing.T) {
	for _, ok := range []string{strings.Repeat("a", 32), strings.Repeat("F0", 20), "0123456789abcdefABCDEF0123456789", strings.Repeat("1", 40)} {
		if err := security.CheckAuthorizationKey(ok); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
	for name, bad := range map[string]string{
		"empty": "", "too short": strings.Repeat("a", 30), "too long": strings.Repeat("a", 42),
		"odd digits": strings.Repeat("a", 33), "not hex": strings.Repeat("g", 32), "binary key": "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f",
	} {
		if err := security.CheckAuthorizationKey(bad); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	hexRE := regexp.MustCompile(`^[0-9a-f]{40}$`)
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		k := security.GenerateAuthorizationKey()
		if !hexRE.MatchString(k) || security.CheckAuthorizationKey(k) != nil || seen[k] {
			t.Fatalf("bad or repeated key %q", k)
		}
		seen[k] = true
	}
}
