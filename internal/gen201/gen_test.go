package gen201

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the generated files")

const (
	schemaDir = "../../docs/spec/2.0.1/schemas"
	repoRoot  = "../.."
)

// TestGeneratedCodeIsUpToDate checks that the committed v201 sources are exactly
// what the generator produces from the schemas. Run it with -update to rewrite
// them after changing the generator or the schemas.
func TestGeneratedCodeIsUpToDate(t *testing.T) {
	files, err := Generate(schemaDir)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		dst := filepath.Join(repoRoot, p)
		if *update {
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(dst, files[p], 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		have, err := os.ReadFile(dst)
		if err != nil {
			t.Errorf("%s is missing; run: go test ./internal/gen201 -run TestGeneratedCodeIsUpToDate -update", p)
			continue
		}
		if !bytes.Equal(have, files[p]) {
			t.Errorf("%s is out of date; run: go test ./internal/gen201 -run TestGeneratedCodeIsUpToDate -update", p)
		}
	}
}

// writeSchemas writes the given schema files (name to JSON) to a new directory.
func writeSchemas(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// pair returns a request and response schema for a message whose request has
// the given properties and whose response is empty.
func pair(name, definitions, props, required string) map[string]string {
	req := `{"definitions":{` + definitions + `},"type":"object","properties":{` + props + `},"required":[` + required + `]}`
	resp := `{"type":"object","properties":{}}`
	return map[string]string{name + "Request.json": req, name + "Response.json": resp}
}

var oneBlock = []Block{{Pkg: "demo", Letter: "X", Title: "Demo", Messages: []string{"Demo"}}}

func mustFail(t *testing.T, files map[string]string, blocks []Block, want string) {
	t.Helper()
	_, err := generate(writeSchemas(t, files), blocks)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("want an error containing %q, got %v", want, err)
	}
}

func TestGenerateRejectsBadInput(t *testing.T) {
	if _, err := Generate(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("a missing directory must fail")
	}
	mustFail(t, map[string]string{"DemoRequest.json": `{not json`}, oneBlock, "DemoRequest.json")
	mustFail(t, map[string]string{"DemoRequest.json": `{"properties":[1]}`}, oneBlock, "properties")
	mustFail(t, map[string]string{"DemoRequest.json": `{"properties":{"a":5}}`}, oneBlock, "DemoRequest.json")
	mustFail(t, map[string]string{"Other.json": `{}`}, oneBlock, "not a Request or Response")
	mustFail(t, map[string]string{"DemoRequest.json": `{}`}, oneBlock, "lacks its request or its response")
	mustFail(t, map[string]string{"Other.txt": `x`, "DemoRequest.json": `{}`, "DemoResponse.json": `{}`}, []Block{{Pkg: "demo", Messages: []string{"Nope"}}}, "has no schema")
	mustFail(t, map[string]string{"DemoRequest.json": `{}`, "DemoResponse.json": `{}`, "ExtraRequest.json": `{}`, "ExtraResponse.json": `{}`}, oneBlock, "not assigned to a block")

	// The same definition with different shapes in two schemas.
	a := pair("Demo", `"ThingEnumType":{"type":"string","enum":["A"]}`, `"t":{"$ref":"#/definitions/ThingEnumType"}`, `"t"`)
	b := pair("Other", `"ThingEnumType":{"type":"string","enum":["B"]}`, `"t":{"$ref":"#/definitions/ThingEnumType"}`, `"t"`)
	files := map[string]string{}
	for k, v := range a {
		files[k] = v
	}
	for k, v := range b {
		files[k] = v
	}
	mustFail(t, files, []Block{{Pkg: "demo", Messages: []string{"Demo", "Other"}}}, "differs")
}

func TestGenerateRejectsUnsupportedSchemas(t *testing.T) {
	tests := map[string]map[string]string{
		"unknown definition":        pair("Demo", ``, `"a":{"$ref":"#/definitions/Nope"}`, ``),
		"reference to a string def": pair("Demo", `"SType":{"type":"string"}`, `"a":{"$ref":"#/definitions/SType"}`, ``),
		"inline object":             pair("Demo", ``, `"a":{"type":"object","properties":{"b":{"type":"string"}}}`, ``),
		"array without items":       pair("Demo", ``, `"a":{"type":"array"}`, ``),
		"required array no min":     pair("Demo", ``, `"a":{"type":"array","items":{"type":"string"}}`, `"a"`),
		"array of unknown ref":      pair("Demo", ``, `"a":{"type":"array","items":{"$ref":"#/definitions/Nope"}}`, ``),
		"array of custom data":      pair("Demo", `"CustomDataType":{"type":"object","properties":{"vendorId":{"type":"string"}}}`, `"a":{"type":"array","items":{"$ref":"#/definitions/CustomDataType"}}`, ``),
		"array of numbers":          pair("Demo", ``, `"a":{"type":"array","items":{"type":"number"}}`, ``),
		"array of dates":            pair("Demo", ``, `"a":{"type":"array","items":{"type":"string","format":"date-time"}}`, ``),
		"definition of a string":    pair("Demo", `"SType":{"type":"string"}`, `"a":{"type":"string"}`, ``),
		"definition of an array":    pair("Demo", `"AType":{"type":"array","items":{"type":"string"}}`, `"a":{"type":"string"}`, ``),
		"bad property name":         pair("Demo", ``, `"1x":{"type":"string"}`, ``),
	}
	for name, files := range tests {
		if _, err := generate(writeSchemas(t, files), oneBlock); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	// Two definitions that become the same Go name, and two enum values that do.
	mustFail(t, pair("Demo", `"FooType":{"type":"object","properties":{}},"Foo":{"type":"object","properties":{}}`, `"a":{"type":"string"}`, ``), oneBlock, "both become")
	mustFail(t, pair("Demo", `"Foo":{"type":"object","properties":{}},"FooEnum":{"type":"string","enum":["x"]},"FooEnumType":{"type":"string","enum":["y"]}`, `"a":{"type":"string"}`, ``), oneBlock, "both become")
	mustFail(t, pair("Demo", `"KindEnumType":{"type":"string","enum":["a-b","a_b"]}`, `"a":{"type":"string"}`, ``), oneBlock, "both become")
}

// TestGenerateCoversEveryConstruct runs the generator on a schema that uses
// each construct once and checks the output compiles and has the expected shape.
func TestGenerateCoversEveryConstruct(t *testing.T) {
	defs := `"CustomDataType":{"type":"object","properties":{"vendorId":{"type":"string","maxLength":255}},"required":["vendorId"]},` +
		`"ModeEnumType":{"description":"The mode.","type":"string","enum":["Fast","Slow-ish"]},` +
		`"InnerType":{"description":"Inner_ Thing\r\nurn:x-oca:ocpp:uid:2:1\r\nAn inner &lt;thing&gt; with <<anchor,a link>>.","type":"object","properties":{"n":{"type":"integer","minimum":0,"maximum":9}},"required":["n"]},` +
		`"EmptyType":{"type":"object","properties":{}}`
	props := `"customData":{"$ref":"#/definitions/CustomDataType"},` +
		`"mode":{"$ref":"#/definitions/ModeEnumType"},` +
		`"modes":{"type":"array","items":{"$ref":"#/definitions/ModeEnumType"},"minItems":1,"maxItems":3},` +
		`"inner":{"$ref":"#/definitions/InnerType"},` +
		`"inners":{"type":"array","items":{"$ref":"#/definitions/InnerType"},"minItems":1},` +
		`"empty":{"$ref":"#/definitions/EmptyType"},` +
		`"name":{"type":"string","maxLength":10},` +
		`"free":{"type":"string"},` +
		`"when":{"type":"string","format":"date-time"},` +
		`"whenToo":{"type":"string","format":"date-time"},` +
		`"count":{"type":"integer","minimum":1},` +
		`"countToo":{"type":"integer","maximum":5},` +
		`"free2":{"type":"integer"},` +
		`"ratio":{"type":"number","minimum":0.5,"maximum":2},` +
		`"ratioToo":{"type":"number","minimum":1},` +
		`"free3":{"type":"number"},` +
		`"flag":{"type":"boolean"},` +
		`"flagToo":{"type":"boolean"},` +
		`"tags":{"type":"array","items":{"type":"string","maxLength":5},"maxItems":4},` +
		`"words":{"type":"array","items":{"type":"string"}},` +
		`"ids":{"type":"array","items":{"type":"integer","minimum":0},"minItems":1},` +
		`"plainIds":{"type":"array","items":{"type":"integer"}},` +
		`"anything":{"description":"Any value."},` +
		`"action":{"type":"string"},` +
		`"evseId":{"type":"integer"}`
	files := pair("Demo", defs, props, `"mode","modes","inner","name","when","count","ratio","flag","anything","ids","action","free2","free3","flagToo","evseId"`)
	files["DemoResponse.json"] = `{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"]}`
	out, err := generate(writeSchemas(t, files), oneBlock)
	if err != nil {
		t.Fatal(err)
	}
	structs := string(out["v201/types/structs.go"])
	for _, want := range []string{"type Inner struct", "type Empty struct", "// Inner is an inner <thing> with a link.", "func (v Empty) Validate() error { return nil }"} {
		if !strings.Contains(structs, want) {
			t.Errorf("structs.go lacks %q:\n%s", want, structs)
		}
	}
	enums := string(out["v201/types/enums.go"])
	if !strings.Contains(enums, `ModeSlowIsh Mode = "Slow-ish"`) {
		t.Errorf("enums.go:\n%s", enums)
	}
	msg := string(out["v201/demo/demo.go"])
	for _, want := range []string{
		"Count int", "free", "json.RawMessage", "check.Number(\"ratio\"", "check.Ptr(0.5)", "check.Ptr(2.0)", "check.NumberPtr(\"ratioToo\"",
		"check.EachInt(\"ids\"", "check.SliceItems(\"tags\"", "check.Items(\"modes\", len(v.Modes), 1, 3)", "check.Each(\"inners\"",
		"*ocpp.DateTime", "check.Time(\"when\"", "check.IntPtr(\"countToo\"", "check.Int(\"count\"",
	} {
		if !strings.Contains(strings.ReplaceAll(msg, "Free ", "free "), want) {
			t.Errorf("demo.go lacks %q:\n%s", want, msg)
		}
	}
	for _, name := range []string{"v201/demo/features.go", "v201/demo/doc.go", "v201/demo/schema_test.go", "v201/v201.go", "v201/types/schema_test.go"} {
		if len(out[name]) == 0 {
			t.Errorf("%s was not generated", name)
		}
	}
}

func TestNaming(t *testing.T) {
	idents := map[string]string{
		"evseId": "EVSEID", "idToken": "IDToken", "ocppCsmsUrl": "OCPPCSMSURL", "iso15118SchemaVersion": "ISO15118SchemaVersion",
		"simPin": "SimPIN", "exiRequest": "EXIRequest", "stateOfCharge": "StateOfCharge", "fullSoC": "FullSoC", "customData": "CustomData",
		"a_b-c": "ABC", "NotifyEVChargingNeeds": "NotifyEVChargingNeeds",
	}
	for in, want := range idents {
		if got := goIdent(in); got != want {
			t.Errorf("goIdent(%q) = %q, want %q", in, got, want)
		}
	}
	types := map[string]string{"IdTokenType": "IDToken", "AuthorizationStatusEnumType": "AuthorizationStatus", "EVSEType": "EVSE", "AuthorizationData": "AuthorizationData", "Type": "Type"}
	for in, want := range types {
		if got := typeName(in); got != want {
			t.Errorf("typeName(%q) = %q, want %q", in, got, want)
		}
	}
	if got := snake("NotifyEVChargingNeeds"); got != "notify_ev_charging_needs" {
		t.Errorf("snake: %q", got)
	}
	if got := snake("Get15118EVCertificate"); got != "get_15118_ev_certificate" {
		t.Errorf("snake: %q", got)
	}
	if got := enumIdent("Energy.Active.Import-Register"); got != "EnergyActiveImportRegister" {
		t.Errorf("enumIdent: %q", got)
	}
	if got := floatText(2); got != "2.0" {
		t.Errorf("floatText: %q", got)
	}
	if got := floatText(0.5); got != "0.5" {
		t.Errorf("floatText: %q", got)
	}
}

func TestCommentFormatting(t *testing.T) {
	long := strings.Repeat("word ", 60)
	got := comment("\t", "Thing", "The "+long, "fallback")
	if !strings.HasPrefix(got, "\t// Thing is the word") || strings.Count(got, "\n") < 2 {
		t.Errorf("long comment not wrapped:\n%s", got)
	}
	if got := comment("", "Thing", "", "Thing is the fallback."); got != "// Thing is the fallback.\n" {
		t.Errorf("%q", got)
	}
	if got := comment("", "Thing", "Thing already starts right.", "x"); got != "// Thing already starts right.\n" {
		t.Errorf("%q", got)
	}
	if got := comment("", "Count", "Number of things.", "x"); got != "// Count: Number of things.\n" {
		t.Errorf("%q", got)
	}
	if got := comment("", "Thing", "Label_ Words\r\nurn:x-oca:ocpp:uid:2:5\r\nA real sentence.", "x"); got != "// Thing is a real sentence.\n" {
		t.Errorf("%q", got)
	}
	if got := lowerFirst("OCPP stays"); got != "OCPP stays" {
		t.Errorf("lowerFirst: %q", got)
	}
}

func TestDirectionsCoverEveryMessage(t *testing.T) {
	total, fromStation, fromCSMS := 0, 0, 0
	for _, b := range Blocks {
		for _, m := range b.Messages {
			total++
			switch Directions[m] {
			case FromStation:
				fromStation++
			case FromCSMS:
				fromCSMS++
			}
		}
	}
	if total != 64 || fromStation != 24 || fromCSMS != 39 || Directions["DataTransfer"] != Both {
		t.Errorf("total=%d fromStation=%d fromCSMS=%d", total, fromStation, fromCSMS)
	}
}
