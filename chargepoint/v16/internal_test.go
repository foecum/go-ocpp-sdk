package chargepoint

import (
	"errors"
	"testing"

	"github.com/foecum/go-ocpp-sdk/ocppj"
)

func TestNewReportsInternalFailure(t *testing.T) {
	boom := errors.New("boom")
	old := newRegistry
	newRegistry = func() (*ocppj.Registry, error) { return nil, boom }
	defer func() { newRegistry = old }()
	if _, err := New(Config{URL: "ws://localhost/ocpp", ID: "CS1"}); !errors.Is(err, boom) {
		t.Errorf("got %v", err)
	}
}
