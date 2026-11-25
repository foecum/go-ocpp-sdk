package types

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"unicode/utf8"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

// Unbounded is the maximum length for strings the specification leaves open.
const Unbounded = math.MaxInt

// Maximum lengths of the OCPP string types.
const (
	CiString20  = 20
	CiString25  = 25
	CiString40  = 40
	CiString50  = 50
	CiString255 = 255
	CiString500 = 500
)

// CheckString validates a string field: at most max characters and, if
// required, not empty. An empty required string counts as missing.
func CheckString(field, v string, max int, required bool) error {
	if v == "" {
		if required {
			return ocpp.Required(field)
		}
		return nil
	}
	if n := utf8.RuneCountInString(v); n > max {
		return ocpp.Invalid(field, fmt.Sprintf("longer than %d characters (%d)", max, n))
	}
	return nil
}

// CheckEnum validates an enumeration field. An empty value counts as missing
// when the field is required, and is fine otherwise.
func CheckEnum[E interface {
	~string
	IsValid() bool
}](field string, v E, required bool) error {
	if v == "" {
		if required {
			return ocpp.Required(field)
		}
		return nil
	}
	if !v.IsValid() {
		return ocpp.Invalid(field, fmt.Sprintf("unknown value %q", string(v)))
	}
	return nil
}

// CheckMin validates that v >= min.
func CheckMin(field string, v, min int) error {
	if v < min {
		return ocpp.Invalid(field, fmt.Sprintf("must be at least %d, got %d", min, v))
	}
	return nil
}

// CheckMinPtr validates an optional integer.
func CheckMinPtr(field string, v *int, min int) error {
	if v == nil {
		return nil
	}
	return CheckMin(field, *v, min)
}

// CheckDateTime validates a required timestamp: it must be set.
func CheckDateTime(field string, v DateTime) error {
	if v.IsZero() {
		return ocpp.Required(field)
	}
	return nil
}

// CheckOneDecimal validates a decimal that accepts at most one fractional
// digit, as the specification requires for charging limits.
func CheckOneDecimal(field string, v float64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return ocpp.Invalid(field, "must be a finite number")
	}
	if scaled := v * 10; math.Abs(scaled-math.Round(scaled)) > 1e-6 {
		return ocpp.Invalid(field, "accepts at most one digit fraction")
	}
	return nil
}

// Prefix prepends a parent field path to the field named in a validation
// error, so a nested failure reads "idTagInfo.status". Other errors are
// returned unchanged.
func Prefix(parent string, err error) error {
	var ve *ocpp.ValidationError
	if errors.As(err, &ve) {
		return &ocpp.ValidationError{Field: parent + "." + ve.Field, Reason: ve.Reason, Missing: ve.Missing}
	}
	return err
}

// First returns the first non-nil error, for chaining checks.
func First(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// CheckURI validates a URI field: if present it must be an
// absolute URI (one with a scheme such as "ftp:" or "https:"), as the schemas'
// "uri" format requires.
func CheckURI(field, v string, required bool) error {
	if v == "" {
		if required {
			return ocpp.Required(field)
		}
		return nil
	}
	u, err := url.Parse(v)
	if err != nil || u.Scheme == "" {
		return ocpp.Invalid(field, "must be an absolute URI")
	}
	return nil
}
