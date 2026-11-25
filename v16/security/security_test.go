package security_test

import (
	"testing"

	"github.com/foecum/go-ocpp-sdk/internal/schematest"
	"github.com/foecum/go-ocpp-sdk/v16/security"
)

// The security schemas are draft-06 with definitions; the harness inlines the
// references before breaking sample payloads.
func TestAgainstOfficialSchemas(t *testing.T) {
	// The whitepaper allows zero or more certificate hashes (0..*); the schema
	// says at least one. The specification text wins.
	schematest.AllowMutation("certificateHashData: array below minItems")
	features := security.Features()
	samples := buildSamples(t)
	schematest.Run(t, "../../docs/spec/1.6/schemas/json", schematest.NewRegistry(t, features), samples)
	schematest.CheckActions(t, features)
	if len(features) != 11 || len(samples) != 11 {
		t.Errorf("%d features, %d samples, want 11", len(features), len(samples))
	}
}
