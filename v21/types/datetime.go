package types

import (
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

// DateTime is the OCPP dateTime type, the same type as ocpp.DateTime.
type DateTime = ocpp.DateTime

// NewDateTime wraps t.
func NewDateTime(t time.Time) DateTime { return ocpp.NewDateTime(t) }

// Now returns the current time as a DateTime.
func Now() DateTime { return ocpp.Now() }
