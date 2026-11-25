package csms

import (
	"errors"
	"testing"

	"github.com/foecum/go-ocpp-sdk/ocppj"
)

// These tests replace the construction seams to reach error paths that cannot
// happen with the fixed arguments the package uses.

func TestNewReportsRegistryFailure(t *testing.T) {
	boom := errors.New("boom")

	old := newRegistry
	newRegistry = func() (*ocppj.Registry, error) { return nil, boom }
	_, err := New(Config{})
	newRegistry = old
	if !errors.Is(err, boom) {
		t.Errorf("registry failure: %v", err)
	}
}
