package v16

import "testing"

func TestRegistryOfRejectsDuplicates(t *testing.T) {
	f := Features()
	if _, err := registryOf(append(f, f[0])); err == nil {
		t.Error("registering an action twice must fail")
	}
}
