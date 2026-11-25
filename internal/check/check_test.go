package check_test

import (
	"errors"
	"strings"
	"testing"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/check"
)

type colour string

func (c colour) IsValid() bool { return c == "red" || c == "blue" }

type item struct{ n int }

func (i item) Validate() error {
	if i.n < 0 {
		return ocpp.Invalid("n", "negative")
	}
	return nil
}

func field(t *testing.T, err error) (string, bool) {
	t.Helper()
	var ve *ocpp.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("not a validation error: %v", err)
	}
	return ve.Field, ve.Missing
}

func TestFirst(t *testing.T) {
	boom := errors.New("boom")
	if check.First() != nil || check.First(nil, nil) != nil || check.First(nil, boom, errors.New("x")) != boom {
		t.Error("First")
	}
}

func TestStrings(t *testing.T) {
	if check.String("f", "abc", 3, true) != nil || check.String("f", "", 3, false) != nil || check.String("f", "é", 1, true) != nil {
		t.Error("valid strings rejected")
	}
	if _, missing := field(t, check.String("f", "", 3, true)); !missing {
		t.Error("empty required string is missing")
	}
	if _, missing := field(t, check.String("f", "abcd", 3, false)); missing {
		t.Error("too long is not missing")
	}
	if check.Length("f", "", 0) != nil || check.Length("f", "ab", 1) == nil {
		t.Error("Length")
	}
	if check.EachString("f", []string{"a", "b"}, 1) != nil {
		t.Error("EachString valid")
	}
	if f, _ := field(t, check.EachString("f", []string{"a", "bb"}, 1)); f != "f[1]" {
		t.Errorf("EachString path %q", f)
	}
}

func TestEnums(t *testing.T) {
	if check.Enum("e", colour("red"), true) != nil || check.Enum("e", colour(""), false) != nil {
		t.Error("valid enums rejected")
	}
	if _, missing := field(t, check.Enum("e", colour(""), true)); !missing {
		t.Error("empty required enum is missing")
	}
	if _, missing := field(t, check.Enum("e", colour("green"), false)); missing {
		t.Error("bad value is not missing")
	}
	if check.EachEnum("e", []colour{"red", "blue"}) != nil {
		t.Error("EachEnum valid")
	}
	if f, _ := field(t, check.EachEnum("e", []colour{"red", ""})); f != "e[1]" {
		t.Errorf("EachEnum path %q", f)
	}
}

func TestEachInt(t *testing.T) {
	if check.EachInt("a", []int{0, 5}, check.Ptr(0), check.Ptr(9)) != nil {
		t.Error("valid ints rejected")
	}
	if f, _ := field(t, check.EachInt("a", []int{0, -1}, check.Ptr(0), nil)); f != "a[1]" {
		t.Errorf("path %q", f)
	}
}

func TestNumbers(t *testing.T) {
	if check.Int("n", 5, nil, nil) != nil || check.Int("n", 5, check.Ptr(0), check.Ptr(10)) != nil {
		t.Error("valid ints rejected")
	}
	if check.Int("n", -1, check.Ptr(0), nil) == nil || check.Int("n", 11, nil, check.Ptr(10)) == nil {
		t.Error("out of range ints accepted")
	}
	if check.IntPtr("n", nil, check.Ptr(0), nil) != nil || check.IntPtr("n", check.Ptr(-1), check.Ptr(0), nil) == nil {
		t.Error("IntPtr")
	}
	if check.Number("x", 1.5, nil, nil) != nil || check.Number("x", 1.5, check.Ptr(1.0), check.Ptr(2.0)) != nil {
		t.Error("valid numbers rejected")
	}
	if check.Number("x", 0.5, check.Ptr(1.0), nil) == nil || check.Number("x", 2.5, nil, check.Ptr(2.0)) == nil {
		t.Error("out of range numbers accepted")
	}
	if check.NumberPtr("x", nil, check.Ptr(1.0), nil) != nil || check.NumberPtr("x", check.Ptr(0.5), check.Ptr(1.0), nil) == nil {
		t.Error("NumberPtr")
	}
}

func TestTimeAndItems(t *testing.T) {
	if check.Time("t", ocpp.Now()) != nil {
		t.Error("set time rejected")
	}
	if _, missing := field(t, check.Time("t", ocpp.DateTime{})); !missing {
		t.Error("zero time is missing")
	}
	if check.Items("a", 2, 1, 3) != nil || check.Items("a", 0, 0, -1) != nil || check.Items("a", 100, -1, -1) != nil {
		t.Error("valid sizes rejected")
	}
	if check.Items("a", 0, 1, -1) == nil || check.Items("a", 4, -1, 3) == nil {
		t.Error("invalid sizes accepted")
	}
}

func TestSliceItems(t *testing.T) {
	if check.SliceItems[int]("a", nil, 1, 3) != nil || check.SliceItems("a", []int{1}, 1, 3) != nil {
		t.Error("absent or well-sized slice rejected")
	}
	if check.SliceItems("a", []int{}, 1, 3) == nil || check.SliceItems("a", []int{1, 2, 3, 4}, 1, 3) == nil {
		t.Error("badly sized slice accepted")
	}
}

func TestNestedValidation(t *testing.T) {
	if check.Nested("n", item{1}) != nil || check.Optional[item]("n", nil) != nil || check.Optional("n", &item{1}) != nil {
		t.Error("valid nested rejected")
	}
	if f, _ := field(t, check.Nested("n", item{-1})); f != "n.n" {
		t.Errorf("Nested path %q", f)
	}
	if f, _ := field(t, check.Optional("o", &item{-1})); f != "o.n" {
		t.Errorf("Optional path %q", f)
	}
	if check.Each("items", []item{{1}, {2}}) != nil {
		t.Error("Each valid")
	}
	if f, _ := field(t, check.Each("items", []item{{1}, {-1}})); f != "items[1].n" {
		t.Errorf("Each path %q", f)
	}
	plain := errors.New("plain")
	if check.Prefix("p", plain) != plain || check.Prefix("p", nil) != nil {
		t.Error("Prefix must pass other errors through")
	}
	if strings.Contains(check.Prefix("p", ocpp.Required("f")).Error(), "p.f") == false {
		t.Error("Prefix text")
	}
}
