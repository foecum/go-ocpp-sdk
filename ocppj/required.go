package ocppj

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

// requiredTag marks a struct field that must be present in the JSON payload:
//
//	Count int `json:"count" ocpp:"required"`
//
// Go cannot tell a missing integer or enum from its zero value after decoding,
// so the registry checks the raw JSON for the keys of tagged fields, at any
// depth through nested structs, pointers and slices. A key that is absent or
// null counts as missing.
const requiredTag = "ocpp"

// checkRequired reports the first required key missing from raw, which must be
// the JSON for a value of type t. Payloads that are not objects where one is
// expected are ignored here; decoding reports those.
func checkRequired(t reflect.Type, raw []byte, path string) error {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	raw = bytes.TrimSpace(raw)
	switch t.Kind() {
	case reflect.Slice:
		var items []json.RawMessage
		if json.Unmarshal(raw, &items) != nil {
			return nil
		}
		for i, item := range items {
			if err := checkRequired(t.Elem(), item, path+"["+itoa(i)+"]"); err != nil {
				return err
			}
		}
	case reflect.Struct:
		var obj map[string]json.RawMessage
		if json.Unmarshal(raw, &obj) != nil {
			return nil
		}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			name, ok := jsonName(f)
			if !ok {
				continue
			}
			val, present := obj[name]
			isNull := bytes.Equal(bytes.TrimSpace(val), []byte("null"))
			full := name
			if path != "" {
				full = path + "." + name
			}
			if hasRequired(f) && (!present || isNull) {
				return ocpp.Required(full)
			}
			if present && !isNull {
				if err := checkRequired(f.Type, val, full); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func jsonName(f reflect.StructField) (string, bool) {
	if !f.IsExported() || f.Anonymous {
		return "", false
	}
	tag := f.Tag.Get("json")
	if tag == "-" {
		return "", false
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" {
		name = f.Name
	}
	return name, true
}

func hasRequired(f reflect.StructField) bool {
	for _, opt := range strings.Split(f.Tag.Get(requiredTag), ",") {
		if opt == "required" {
			return true
		}
	}
	return false
}

func itoa(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}
