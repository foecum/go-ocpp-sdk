package core_test

import (
	"testing"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/schematest"
	"github.com/foecum/go-ocpp-sdk/ocppj"
	"github.com/foecum/go-ocpp-sdk/v16/core"
)

const schemaDir = "../../docs/spec/1.6/schemas/json"

func newRegistry(t testing.TB) *ocppj.Registry {
	t.Helper()
	reg := ocppj.NewRegistry(ocpp.V16)
	for _, f := range core.Features() {
		if err := reg.Register(f); err != nil {
			t.Fatal(err)
		}
	}
	return reg
}

// TestAgainstOfficialSchemas round-trips sample payloads and checks the
// registry rejects whatever the official JSON schemas reject.
func TestAgainstOfficialSchemas(t *testing.T) {
	schematest.Run(t, schemaDir, newRegistry(t), samples)
}

func TestEveryCoreActionIsCovered(t *testing.T) {
	features := core.Features()
	if len(features) != 16 || len(samples) != 16 {
		t.Fatalf("%d features, %d samples, want 16", len(features), len(samples))
	}
	for _, f := range features {
		if _, ok := samples[f.Action()]; !ok {
			t.Errorf("no samples for %s", f.Action())
		}
		req := newRequest(f)
		if req.Action() != f.Action() {
			t.Errorf("%s: request reports action %q", f.Action(), req.Action())
		}
	}
}

func newRequest(f ocpp.Feature) ocpp.Request {
	return reflectNew(f.RequestType()).(ocpp.Request)
}
