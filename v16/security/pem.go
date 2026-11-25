package security

import (
	"bytes"
	"encoding/pem"
	"fmt"
	"strings"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

// PEM block types.
const (
	pemCertificate = "CERTIFICATE"
	pemCSR         = "CERTIFICATE REQUEST"
	pemCSRLegacy   = "NEW CERTIFICATE REQUEST"
)

// checkPEM validates a field holding PEM text: not empty, at most max
// characters, and nothing but PEM blocks whose types are in types (at least
// one, at most maxBlocks). The contents of the blocks are not parsed.
func checkPEM(field, v string, max, maxBlocks int, types ...string) error {
	if v == "" {
		return ocpp.Required(field)
	}
	if n := len([]rune(v)); n > max {
		return ocpp.Invalid(field, fmt.Sprintf("longer than %d characters (%d)", max, n))
	}
	rest := []byte(v)
	blocks := 0
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if !contains(types, block.Type) {
			return ocpp.Invalid(field, fmt.Sprintf("unexpected PEM block %q, want %s", block.Type, strings.Join(types, " or ")))
		}
		blocks++
	}
	if blocks == 0 {
		return ocpp.Invalid(field, "must be PEM encoded")
	}
	if blocks > maxBlocks {
		return ocpp.Invalid(field, fmt.Sprintf("contains %d PEM blocks, at most %d allowed", blocks, maxBlocks))
	}
	if len(bytes.TrimSpace(rest)) != 0 {
		return ocpp.Invalid(field, "contains data outside the PEM blocks")
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
