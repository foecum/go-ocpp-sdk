// Package check holds the validation helpers used by the generated OCPP 2.0.1
// message types. Each helper returns nil or an *ocpp.ValidationError, so the
// generated Validate methods can be one expression:
//
//	return check.First(
//		check.String("name", v.Name, 50, true),
//		check.Optional("evse", v.EVSE),
//	)
package check

import (
	"fmt"
	"math"
	"unicode/utf8"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

// Unbounded is the maximum length for strings the schemas leave open.
const Unbounded = math.MaxInt

// First returns the first non-nil error.
func First(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// String validates a string field of at most max characters. An empty string
// counts as missing when required.
func String(field, v string, max int, required bool) error {
	if v == "" && required {
		return ocpp.Required(field)
	}
	return Length(field, v, max)
}

// Length validates the length of a string that may be empty.
func Length(field, v string, max int) error {
	if n := utf8.RuneCountInString(v); n > max {
		return ocpp.Invalid(field, fmt.Sprintf("longer than %d characters (%d)", max, n))
	}
	return nil
}

// Enum validates an enumeration field. An empty value counts as missing when
// the field is required, and is fine otherwise.
func Enum[E interface {
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

// Int validates an integer against optional bounds: pass nil for no bound.
func Int(field string, v int, min, max *int) error {
	if min != nil && v < *min {
		return ocpp.Invalid(field, fmt.Sprintf("must be at least %d, got %d", *min, v))
	}
	if max != nil && v > *max {
		return ocpp.Invalid(field, fmt.Sprintf("must be at most %d, got %d", *max, v))
	}
	return nil
}

// IntPtr validates an optional integer.
func IntPtr(field string, v *int, min, max *int) error {
	if v == nil {
		return nil
	}
	return Int(field, *v, min, max)
}

// Number validates a decimal against optional bounds.
func Number(field string, v float64, min, max *float64) error {
	if min != nil && v < *min {
		return ocpp.Invalid(field, fmt.Sprintf("must be at least %v, got %v", *min, v))
	}
	if max != nil && v > *max {
		return ocpp.Invalid(field, fmt.Sprintf("must be at most %v, got %v", *max, v))
	}
	return nil
}

// NumberPtr validates an optional decimal.
func NumberPtr(field string, v *float64, min, max *float64) error {
	if v == nil {
		return nil
	}
	return Number(field, *v, min, max)
}

// Time validates a required timestamp: it must be set.
func Time(field string, v ocpp.DateTime) error {
	if v.IsZero() {
		return ocpp.Required(field)
	}
	return nil
}

// Items validates the number of elements of an array. Pass a negative bound
// for none.
func Items(field string, n, min, max int) error {
	if min >= 0 && n < min {
		return ocpp.Invalid(field, fmt.Sprintf("needs at least %d items, got %d", min, n))
	}
	if max >= 0 && n > max {
		return ocpp.Invalid(field, fmt.Sprintf("allows at most %d items, got %d", max, n))
	}
	return nil
}

// SliceItems validates the size of an optional array: nil (absent) is fine.
func SliceItems[T any](field string, s []T, min, max int) error {
	if s == nil {
		return nil
	}
	return Items(field, len(s), min, max)
}

// Validator is implemented by every generated struct.
type Validator interface{ Validate() error }

// Prefix prepends a parent path to the field named in a validation error, so a
// nested failure reads "idToken.type". Other errors are returned unchanged.
func Prefix(parent string, err error) error {
	if err == nil {
		return nil
	}
	if ve, ok := err.(*ocpp.ValidationError); ok {
		return &ocpp.ValidationError{Field: parent + "." + ve.Field, Reason: ve.Reason, Missing: ve.Missing}
	}
	return err
}

// Nested validates a required nested struct.
func Nested(field string, v Validator) error { return Prefix(field, v.Validate()) }

// Optional validates an optional nested struct: nil is fine.
func Optional[T Validator](field string, p *T) error {
	if p == nil {
		return nil
	}
	return Prefix(field, (*p).Validate())
}

// Each validates every element of an array of structs.
func Each[T Validator](field string, items []T) error {
	for i, item := range items {
		if err := Prefix(fmt.Sprintf("%s[%d]", field, i), item.Validate()); err != nil {
			return err
		}
	}
	return nil
}

// EachString validates every element of an array of strings.
func EachString(field string, items []string, max int) error {
	for i, item := range items {
		if err := String(fmt.Sprintf("%s[%d]", field, i), item, max, false); err != nil {
			return err
		}
	}
	return nil
}

// EachInt validates every element of an array of integers.
func EachInt(field string, items []int, min, max *int) error {
	for i, item := range items {
		if err := Int(fmt.Sprintf("%s[%d]", field, i), item, min, max); err != nil {
			return err
		}
	}
	return nil
}

// EachEnum validates every element of an array of enumeration values.
func EachEnum[E interface {
	~string
	IsValid() bool
}](field string, items []E) error {
	for i, item := range items {
		if err := Enum(fmt.Sprintf("%s[%d]", field, i), item, true); err != nil {
			return err
		}
	}
	return nil
}

// Ptr returns a pointer to v, for the optional bounds of Int and Number.
func Ptr[T any](v T) *T { return &v }
