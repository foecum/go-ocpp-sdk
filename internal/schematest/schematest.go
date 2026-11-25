// Package schematest checks message types against the official OCPP JSON
// schemas. It is used by tests only.
//
// For every message it takes hand-written sample payloads and checks that:
//
//   - the sample is valid against the official schema (sanity check);
//   - decoding it through the registry and encoding it again gives the same JSON;
//   - the re-encoded JSON is valid against the schema;
//   - every way of breaking the sample that the schema rejects (a missing
//     required key, a string over maxLength, a value outside an enum, a wrong
//     JSON type, a malformed date-time or URI) is also rejected by the registry.
//
// The registry is deliberately more lenient than the schemas in one respect:
// it ignores unknown properties, as real charge points send them.
package schematest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	ocpp "github.com/foecum/go-ocpp-sdk"

	"github.com/foecum/go-ocpp-sdk/ocppj"
)

// Sample holds sample payloads for one action. Use JSON that the SDK would
// itself produce (timestamps as "2006-01-02T15:04:05Z" or with milliseconds).
type Sample struct {
	// ReqMin and ReqFull are request payloads with only the required fields and
	// with every field present. ConfMin and ConfFull are the same for the
	// confirmation.
	ReqMin, ReqFull, ConfMin, ConfFull string
}

// Run checks every action in samples against the schemas in dir, decoding
// through reg. Schema files are <Action>.json and <Action>Response.json.
func Run(t *testing.T, dir string, reg *ocppj.Registry, samples map[string]Sample) {
	decodeReq := func(action string, payload []byte) (any, error) { return reg.DecodeRequest(action, payload) }
	decodeResp := func(action string, payload []byte) (any, error) { return reg.DecodeResponse(action, payload) }
	t.Helper()
	for action, s := range samples {
		t.Run(action, func(t *testing.T) {
			req := loadSchema(t, dir, action)
			resp := loadSchema(t, dir, action+"Response")
			check(t, "request", req, decodeReq, s.ReqMin, s.ReqFull, action)
			check(t, "confirmation", resp, decodeResp, s.ConfMin, s.ConfFull, action)
		})
	}
}

type schemaDoc struct {
	compiled *jsonschema.Schema
	raw      map[string]any
}

func loadSchema(t *testing.T, dir, name string) schemaDoc {
	t.Helper()
	path := filepath.Join(dir, name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	raw = inlineRefs(raw, raw).(map[string]any) // the mutation walker reads plain nested schemas
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft4)
	c.AssertFormat()
	url := "mem:///" + name + ".json"
	if err := c.AddResource(url, doc); err != nil {
		t.Fatal(err)
	}
	s, err := c.Compile(url)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return schemaDoc{compiled: s, raw: raw}
}

// valid reports whether payload satisfies the schema.
func (s schemaDoc) valid(payload []byte) error {
	v, err := jsonschema.UnmarshalJSON(bytes.NewReader(payload))
	if err != nil {
		return err
	}
	return s.compiled.Validate(v)
}

// decodeFunc decodes a payload through the registry and returns the value to
// re-encode.
type decodeFunc func(action string, payload []byte) (any, error)

func check(t *testing.T, kind string, schema schemaDoc, decode decodeFunc, min, full, action string) {
	t.Helper()
	for name, sample := range map[string]string{"min": min, "full": full} {
		label := fmt.Sprintf("%s %s", kind, name)
		if err := schema.valid([]byte(sample)); err != nil {
			t.Fatalf("%s: sample is not valid against the official schema: %v\n%s", label, err, sample)
		}
		v, err := decode(action, []byte(sample))
		if err != nil {
			t.Fatalf("%s: decode: %v\n%s", label, err, sample)
		}
		out, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("%s: marshal: %v", label, err)
		}
		if err := schema.valid(out); err != nil {
			t.Errorf("%s: re-encoded payload is not valid against the schema: %v\n%s", label, err, out)
		}
		if !sameJSON(t, []byte(sample), out) {
			t.Errorf("%s: round trip changed the payload\n in: %s\nout: %s", label, sample, out)
		}
		var tree any
		if err := json.Unmarshal([]byte(sample), &tree); err != nil {
			t.Fatal(err)
		}
		for _, m := range mutations(schema.raw, tree) {
			mutated, err := json.Marshal(m.value)
			if err != nil {
				t.Fatal(err)
			}
			schemaErr := schema.valid(mutated)
			_, oursErr := decode(action, mutated)
			switch {
			case m.lenient || allowed(m.name):
				if oursErr != nil {
					t.Errorf("%s: %s: unknown properties must be ignored, got %v\n%s", label, m.name, oursErr, mutated)
				}
			case schemaErr == nil:
				t.Errorf("%s: %s: mutation is unexpectedly valid against the schema\n%s", label, m.name, mutated)
			case oursErr == nil:
				t.Errorf("%s: %s: schema rejects but the registry accepts\n%s", label, m.name, mutated)
			}
		}
	}
}

func sameJSON(t *testing.T, a, b []byte) bool {
	t.Helper()
	var x, y any
	if err := json.Unmarshal(a, &x); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &y); err != nil {
		t.Fatal(err)
	}
	return reflect.DeepEqual(x, y)
}

type mutation struct {
	name    string
	value   any
	lenient bool // the registry is expected to accept it although the schema rejects it
}

// mutations derives broken variants of value from schema.
func mutations(schema map[string]any, value any) []mutation {
	var out []mutation
	walk(schema, value, nil, func(path []any, name string, apply func(any) any, lenient bool) {
		out = append(out, mutation{name: pathString(path) + ": " + name, value: apply(value), lenient: lenient})
	})
	return out
}

type emit func(path []any, name string, apply func(root any) any, lenient bool)

func walk(schema map[string]any, value any, path []any, out emit) {
	switch schema["type"] {
	case "object":
		obj, ok := value.(map[string]any)
		if !ok {
			return
		}
		props, _ := schema["properties"].(map[string]any)
		required := map[string]bool{}
		if rs, ok := schema["required"].([]any); ok {
			for _, r := range rs {
				required[r.(string)] = true
			}
		}
		for _, key := range sortedKeys(obj) {
			p := append(append([]any{}, path...), key)
			if required[key] {
				out(p, "missing required key", func(root any) any { return edit(root, p, nil, true) }, false)
			}
			if ps, ok := props[key].(map[string]any); ok {
				propertyMutations(ps, obj[key], p, out)
				walk(ps, obj[key], p, out)
			}
		}
		out(path, "unknown property", func(root any) any {
			return edit(root, append(append([]any{}, path...), "bogusProperty"), 1.0, false)
		}, true)
	case "array":
		arr, ok := value.([]any)
		items, _ := schema["items"].(map[string]any)
		if ok && len(arr) > 0 && items != nil {
			p := append(append([]any{}, path...), 0)
			propertyMutations(items, arr[0], p, out)
			walk(items, arr[0], p, out)
		}
	}
}

// propertyMutations emits mutations of a single value.
func propertyMutations(schema map[string]any, value any, path []any, out emit) {
	set := func(name string, v any) {
		out(path, name, func(root any) any { return edit(root, path, v, false) }, false)
	}
	switch schema["type"] {
	case "string":
		set("wrong type (number instead of string)", 12345.0)
		if max, ok := schema["maxLength"].(float64); ok {
			set("string over maxLength", strings.Repeat("a", int(max)+1))
		}
		if enum, ok := schema["enum"].([]any); ok && len(enum) > 0 {
			set("value outside enum", "NotInTheEnum")
		}
		switch schema["format"] {
		case "date-time":
			set("malformed date-time", "yesterday-ish")
		case "uri":
			set("malformed uri", "not a uri")
		}
	case "integer":
		set("wrong type (string instead of integer)", "abc")
		set("fraction instead of integer", 1.5)
		rangeMutations(schema, set)
	case "number":
		set("wrong type (string instead of number)", "abc")
		rangeMutations(schema, set)
	case "boolean":
		set("wrong type (string instead of boolean)", "abc")
	case "array":
		set("wrong type (string instead of array)", "abc")
		if min, ok := schema["minItems"].(float64); ok && min >= 1 {
			set("array below minItems", []any{})
		}
		if max, ok := schema["maxItems"].(float64); ok {
			if arr, ok := value.([]any); ok && len(arr) > 0 {
				over := make([]any, int(max)+1)
				for i := range over {
					over[i] = arr[0]
				}
				set("array above maxItems", over)
			}
		}
	case "object":
		set("wrong type (string instead of object)", "abc")
	}
}

// edit returns a deep copy of root with the value at path replaced (or, with
// remove set, deleted).
func edit(root any, path []any, v any, remove bool) any {
	root = deepCopy(root)
	cur := root
	for i, step := range path {
		last := i == len(path)-1
		switch node := cur.(type) {
		case map[string]any:
			key := step.(string)
			switch {
			case last && remove:
				delete(node, key)
			case last:
				node[key] = v
			default:
				cur = node[key]
			}
		case []any:
			idx := step.(int)
			if last {
				node[idx] = v
			} else {
				cur = node[idx]
			}
		}
	}
	return root
}

func deepCopy(v any) any {
	b, _ := json.Marshal(v)
	var out any
	_ = json.Unmarshal(b, &out)
	return out
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Small maps; insertion sort keeps the output deterministic.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

func pathString(path []any) string {
	var sb strings.Builder
	for _, p := range path {
		switch p := p.(type) {
		case string:
			if sb.Len() > 0 {
				sb.WriteByte('.')
			}
			sb.WriteString(p)
		case int:
			fmt.Fprintf(&sb, "[%d]", p)
		}
	}
	if sb.Len() == 0 {
		return "(root)"
	}
	return sb.String()
}

// NewRegistry returns a registry for OCPP 1.6 holding the given features.
func NewRegistry(t testing.TB, features []ocpp.Feature) *ocppj.Registry {
	t.Helper()
	return NewRegistryFor(t, ocpp.V16, features)
}

// NewRegistryFor returns a registry for the given version holding the features.
func NewRegistryFor(t testing.TB, version ocpp.Version, features []ocpp.Feature) *ocppj.Registry {
	t.Helper()
	reg := ocppj.NewRegistry(version)
	for _, f := range features {
		if err := reg.Register(f); err != nil {
			t.Fatal(err)
		}
	}
	return reg
}

// CheckActions verifies that every feature's request and confirmation types
// report the feature's action, and that the type names follow the specification
// ("<Action>Request" and "<Action>Confirmation").
func CheckActions(t testing.TB, features []ocpp.Feature) {
	t.Helper()
	CheckActionsSuffix(t, features, "Confirmation")
}

// CheckActionsSuffix is CheckActions for a version whose response types end in
// the given suffix ("Confirmation" in OCPP 1.6, "Response" in 2.0.1).
func CheckActionsSuffix(t testing.TB, features []ocpp.Feature, suffix string) {
	t.Helper()
	for _, f := range features {
		if ocpp.IsOneWay(f) {
			msg := reflect.New(f.RequestType()).Interface().(ocpp.Request)
			if msg.Action() != f.Action() || f.RequestType().Name() != f.Action() || f.ResponseType() != nil {
				t.Errorf("%s: unexpected one-way type %s reporting %q", f.Action(), f.RequestType().Name(), msg.Action())
			}
			continue
		}
		req := reflect.New(f.RequestType()).Interface().(ocpp.Request)
		resp := reflect.New(f.ResponseType()).Interface().(ocpp.Response)
		if req.Action() != f.Action() || resp.Action() != f.Action() {
			t.Errorf("%s: request reports %q, confirmation %q", f.Action(), req.Action(), resp.Action())
		}
		if f.RequestType().Name() != f.Action()+"Request" || f.ResponseType().Name() != f.Action()+suffix {
			t.Errorf("%s: unexpected type names %s / %s", f.Action(), f.RequestType().Name(), f.ResponseType().Name())
		}
	}
}

// inlineRefs returns node with every {"$ref": "#/definitions/X"} replaced by a
// copy of that definition, so the mutation walker needs no reference handling.
// The schemas have no recursive definitions.
func inlineRefs(node any, root map[string]any) any {
	switch n := node.(type) {
	case map[string]any:
		if ref, ok := n["$ref"].(string); ok {
			name := strings.TrimPrefix(ref, "#/definitions/")
			defs, _ := root["definitions"].(map[string]any)
			if def, found := defs[name]; found {
				return inlineRefs(def, root)
			}
		}
		out := make(map[string]any, len(n))
		for k, v := range n {
			if k == "definitions" {
				continue
			}
			out[k] = inlineRefs(v, root)
		}
		return out
	case []any:
		out := make([]any, len(n))
		for i, v := range n {
			out[i] = inlineRefs(v, root)
		}
		return out
	}
	return node
}

// RunGenerated is like Run, but derives the sample payloads from the schemas
// themselves: for each action a minimal one (required properties only) and a
// full one (every property). It is meant for the generated OCPP 2.0.1 messages,
// where writing 128 samples by hand would be error-prone.
func RunGenerated(t *testing.T, dir string, reg *ocppj.Registry, actions []string) {
	t.Helper()
	decodeReq := func(action string, payload []byte) (any, error) { return reg.DecodeRequest(action, payload) }
	decodeResp := func(action string, payload []byte) (any, error) { return reg.DecodeResponse(action, payload) }
	for _, action := range actions {
		t.Run(action, func(t *testing.T) {
			if f, ok := reg.Lookup(action); ok && ocpp.IsOneWay(f) {
				// An unconfirmed message has one schema, named after the action.
				msg := loadSchema(t, dir, action)
				check(t, "message", msg, decodeReq, sampleJSON(t, msg.raw, false), sampleJSON(t, msg.raw, true), action)
				return
			}
			req := loadSchema(t, dir, action+"Request")
			resp := loadSchema(t, dir, action+"Response")
			check(t, "request", req, decodeReq, sampleJSON(t, req.raw, false), sampleJSON(t, req.raw, true), action)
			check(t, "response", resp, decodeResp, sampleJSON(t, resp.raw, false), sampleJSON(t, resp.raw, true), action)
		})
	}
}

func sampleJSON(t *testing.T, schema map[string]any, full bool) string {
	t.Helper()
	b, err := json.Marshal(sampleValue(schema, full, ""))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// sampleValue builds a value satisfying schema. With full set, optional
// properties are included too.
func sampleValue(schema map[string]any, full bool, name string) any {
	if enum, ok := schema["enum"].([]any); ok && len(enum) > 0 {
		if name == "chargingProfilePurpose" && full {
			return "TxProfile" // the purpose the specification demands where a transactionId is given
		}
		if full {
			return enum[len(enum)-1]
		}
		return enum[0]
	}
	switch schema["type"] {
	case "string":
		if schema["format"] == "date-time" {
			if full {
				return "2026-01-02T03:04:05.500Z"
			}
			return "2026-01-02T03:04:05Z"
		}
		switch name { // strings whose syntax the specification text fixes
		case "startTimeOfDay", "endTimeOfDay":
			return "00:00"
		case "validFromDate", "validToDate":
			return "2026-01-02"
		}
		n := 3
		if max, ok := schema["maxLength"].(float64); ok && int(max) < n {
			n = int(max)
		}
		return strings.Repeat("a", n)
	case "integer":
		return numberWithin(schema, 1)
	case "number":
		return numberWithin(schema, 1.5)
	case "boolean":
		return true
	case "array":
		items, _ := schema["items"].(map[string]any)
		count := 1
		if name == "v2xFreqWattCurve" || name == "v2xSignalWattCurve" {
			count = 2 // the specification asks for at least two coordinates
		}
		if min, ok := schema["minItems"].(float64); ok && int(min) > count {
			count = int(min)
		}
		out := make([]any, 0, count)
		for i := 0; i < count; i++ {
			out = append(out, sampleValue(items, full, name))
		}
		return out
	case "object":
		props, _ := schema["properties"].(map[string]any)
		required := map[string]bool{}
		if rs, ok := schema["required"].([]any); ok {
			for _, r := range rs {
				required[r.(string)] = true
			}
		}
		obj := map[string]any{}
		for key, p := range props {
			if required[key] || full {
				obj[key] = sampleValue(p.(map[string]any), full, key)
			}
		}
		if name == "customData" && full {
			obj["extension"] = 1.0 // customData accepts arbitrary extra properties
		}
		return obj
	}
	return map[string]any{"any": "json"} // a property without a type holds any JSON value
}

func numberWithin(schema map[string]any, preferred float64) float64 {
	v := preferred
	if min, ok := schema["minimum"].(float64); ok && v < min {
		v = min
	}
	if max, ok := schema["maximum"].(float64); ok && v > max {
		v = max
	}
	return v
}

// rangeMutations emits values just outside minimum and maximum.
func rangeMutations(schema map[string]any, set func(name string, v any)) {
	if min, ok := schema["minimum"].(float64); ok {
		set("below minimum", min-1)
	}
	if max, ok := schema["maximum"].(float64); ok {
		set("above maximum", max+1)
	}
}

var allowedMutations []string

// AllowMutation declares that the registry deliberately accepts the mutation
// whose name ends with suffix although the schema rejects it, because the
// specification text has precedence over the schema. Such mutations must be
// accepted. Call it from a test before Run.
func AllowMutation(suffix string) { allowedMutations = append(allowedMutations, suffix) }

func allowed(name string) bool {
	for _, suffix := range allowedMutations {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}
