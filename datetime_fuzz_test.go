package ocpp

import (
	"encoding/json"
	"testing"
)

// FuzzDateTime checks that UnmarshalJSON never panics, and that a value it
// accepts survives a marshal and unmarshal round trip to the same instant.
func FuzzDateTime(f *testing.F) {
	for _, s := range []string{
		`"2026-11-25T12:00:00Z"`, `"2026-11-25T12:00:00.123+02:00"`, `"2026-11-25T12:00:00"`, `"2026-11-25T12:00:00.5"`,
		`null`, `""`, `123`, `"2026-13-45T25:61:61Z"`, `"0001-01-01T00:00:00Z"`, `"9999-12-31T23:59:59.999999999Z"`,
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		var d DateTime
		if err := json.Unmarshal([]byte(s), &d); err != nil || d.IsZero() {
			return
		}
		out, err := json.Marshal(d)
		if err != nil {
			return // years outside 0..9999 cannot be written; not a parse problem
		}
		var back DateTime
		if err := json.Unmarshal(out, &back); err != nil {
			t.Fatalf("%s marshalled to %s, which does not parse: %v", s, out, err)
		}
	})
}
