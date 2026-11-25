package types

import (
	"encoding/json"
	"fmt"
	"time"
)

// DateTime is the OCPP dateTime type: an RFC 3339 timestamp.
//
// It marshals in UTC, without fractional seconds when the time has none and
// with milliseconds otherwise ("2026-01-02T03:04:05Z", "2026-01-02T03:04:05.123Z").
// It accepts the variants real charge points send: with or without fractional
// seconds, with "Z" or a numeric offset, and without any zone (taken as UTC).
// A required DateTime that is the zero value counts as missing.
type DateTime struct{ time.Time }

// NewDateTime wraps t.
func NewDateTime(t time.Time) DateTime { return DateTime{t} }

// Now returns the current time as a DateTime.
func Now() DateTime { return DateTime{time.Now()} }

// Ptr returns a pointer to d, for optional timestamp fields.
func (d DateTime) Ptr() *DateTime { return &d }

const (
	layoutSeconds = "2006-01-02T15:04:05Z"
	layoutMillis  = "2006-01-02T15:04:05.000Z"
	layoutNoZone  = "2006-01-02T15:04:05"
)

// MarshalJSON implements json.Marshaler.
func (d DateTime) MarshalJSON() ([]byte, error) {
	t := d.Time.UTC()
	layout := layoutSeconds
	if t.Nanosecond() != 0 {
		layout = layoutMillis
	}
	return json.Marshal(t.Format(layout))
}

// UnmarshalJSON implements json.Unmarshaler. A JSON null leaves d unchanged.
func (d *DateTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("dateTime must be a string: %w", err)
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		d.Time = t
		return nil
	}
	if t, err := time.ParseInLocation(layoutNoZone+".999999999", s, time.UTC); err == nil {
		d.Time = t
		return nil
	}
	return fmt.Errorf("invalid dateTime %q", s)
}
