package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

// TestSmallRun runs a short load test with a few stations for both versions and
// checks that nothing fails.
func TestSmallRun(t *testing.T) {
	if testing.Short() {
		t.Skip("runs a CSMS and simulated stations")
	}
	for _, version := range []string{"1.6", "2.0.1"} {
		t.Run(version, func(t *testing.T) {
			var out bytes.Buffer
			err := run(context.Background(), options{
				stations: 20, duration: 1500 * time.Millisecond, version: version,
				heartbeat: 200 * time.Millisecond, session: 400 * time.Millisecond,
			}, &out)
			if err != nil {
				t.Fatalf("%v\n%s", err, out.String())
			}
			if !strings.Contains(out.String(), "0 errors") {
				t.Errorf("unexpected output:\n%s", out.String())
			}
		})
	}
	if err := run(context.Background(), options{stations: 1, version: "9"}, &bytes.Buffer{}); err == nil {
		t.Error("an unknown version must be rejected")
	}
	if err := run(context.Background(), options{stations: 0, version: "1.6"}, &bytes.Buffer{}); err == nil {
		t.Error("zero stations must be rejected")
	}
}
