package types

import (
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

// DateTime is the OCPP dateTime type: an RFC 3339 timestamp. It is the same type
// as ocpp.DateTime, shared by every OCPP version: see there for the formats it
// marshals and accepts.
type DateTime = ocpp.DateTime

// NewDateTime wraps t.
func NewDateTime(t time.Time) DateTime { return ocpp.NewDateTime(t) }

// Now returns the current time as a DateTime.
func Now() DateTime { return ocpp.Now() }
