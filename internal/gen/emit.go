package gen

import (
	"fmt"
	"sort"
	"strings"
)

// field is one generated struct field.
type field struct {
	goName string
	decl   string // the Go type
	tag    string
	doc    string
	check  string // a validation expression, or ""
}

func requiredSet(n *node) map[string]bool {
	set := map[string]bool{}
	for _, r := range n.Required {
		set[r] = true
	}
	return set
}

func intLit(v *float64) string {
	if v == nil {
		return "nil"
	}
	return fmt.Sprintf("check.Ptr(%d)", int(*v))
}

func floatLit(v *float64) string {
	if v == nil {
		return "nil"
	}
	return fmt.Sprintf("check.Ptr(%v)", floatText(*v))
}

func floatText(v float64) string {
	s := fmt.Sprintf("%v", v)
	if !strings.ContainsAny(s, ".e") {
		s += ".0"
	}
	return s
}

func itemBound(v *int) string {
	if v == nil {
		return "-1"
	}
	return fmt.Sprint(*v)
}

// field builds the Go field for property name of the object def (used to look
// up exceptions), with qual prefixing shared type names ("types." in message
// packages, "" inside the types package).
func (g *gen) field(def, name string, n *node, required bool, qual string) (field, error) {
	goName := goIdent(name)
	if renamed, ok := fieldRenames[def+"."+name]; ok {
		goName = renamed
	}
	fallback := goName + " is the " + name + " field."
	if name == "customData" {
		fallback = goName + " holds vendor-specific extension data."
	}
	f := field{goName: goName, doc: comment("\t", goName, n.Description, fallback)}
	omit := ""
	if !required {
		omit = ",omitempty"
	}
	f.tag = fmt.Sprintf("`json:\"%s%s\"", name, omit)
	if required {
		f.tag += ` ocpp:"required"`
	}
	f.tag += "`"
	v := "v." + goName

	switch {
	case n.Ref != "":
		d := refName(n.Ref)
		target, ok := g.defs[d]
		if !ok {
			return f, fmt.Errorf("%s.%s: unknown definition %s", def, name, d)
		}
		switch {
		case d == "CustomDataType":
			f.decl = "*" + qual + "CustomData"
			f.check = fmt.Sprintf("check.Optional(%q, %s)", name, v)
			f.tag = fmt.Sprintf("`json:\"%s,omitempty\"`", name)
		case target.Enum != nil:
			f.decl = qual + g.goNames[d]
			f.check = fmt.Sprintf("check.Enum(%q, %s, %v)", name, v, required)
		case target.Type == "object":
			if required {
				f.decl = qual + g.goNames[d]
				f.check = fmt.Sprintf("check.Nested(%q, %s)", name, v)
			} else {
				f.decl = "*" + qual + g.goNames[d]
				f.check = fmt.Sprintf("check.Optional(%q, %s)", name, v)
			}
		default:
			return f, fmt.Errorf("%s.%s: %w: reference to %s", def, name, errUnsupported, d)
		}
	case n.Type == "string" && n.Format == "date-time":
		if required {
			f.decl = "ocpp.DateTime"
			f.check = fmt.Sprintf("check.Time(%q, %s)", name, v)
		} else {
			f.decl = "*ocpp.DateTime"
		}
	case n.Type == "string":
		f.decl = "string"
		max := "check.Unbounded"
		if n.MaxLength != nil {
			max = fmt.Sprint(*n.MaxLength)
		}
		if required && !emptyAllowed[def+"."+name] {
			f.check = fmt.Sprintf("check.String(%q, %s, %s, true)", name, v, max)
		} else {
			f.check = fmt.Sprintf("check.Length(%q, %s, %s)", name, v, max)
		}
	case n.Type == "integer":
		f.decl = "int"
		if !required {
			f.decl = "*int"
		}
		if n.Minimum != nil || n.Maximum != nil {
			fn := "check.Int"
			if !required {
				fn = "check.IntPtr"
			}
			f.check = fmt.Sprintf("%s(%q, %s, %s, %s)", fn, name, v, intLit(n.Minimum), intLit(n.Maximum))
		}
	case n.Type == "number":
		f.decl = "float64"
		if !required {
			f.decl = "*float64"
		}
		if n.Minimum != nil || n.Maximum != nil {
			fn := "check.Number"
			if !required {
				fn = "check.NumberPtr"
			}
			f.check = fmt.Sprintf("%s(%q, %s, %s, %s)", fn, name, v, floatLit(n.Minimum), floatLit(n.Maximum))
		}
	case n.Type == "boolean":
		f.decl = "bool"
		if !required {
			f.decl = "*bool"
		}
	case n.Type == "array":
		return g.arrayField(def, name, n, required, qual, f)
	case !n.hasType && n.Enum == nil && len(n.Props) == 0:
		f.decl = "json.RawMessage" // any JSON value
	default:
		return f, fmt.Errorf("%s.%s: %w: type %q", def, name, errUnsupported, n.Type)
	}
	return f, nil
}

func (g *gen) arrayField(def, name string, n *node, required bool, qual string, f field) (field, error) {
	v := "v." + f.goName
	if n.Items == nil {
		return f, fmt.Errorf("%s.%s: %w: array without items", def, name, errUnsupported)
	}
	if required && n.MinItems == nil {
		return f, fmt.Errorf("%s.%s: %w: required array without minItems", def, name, errUnsupported)
	}
	f.tag = strings.Replace(f.tag, `"`+name+`"`, `"`+name+`,omitempty"`, 1) // arrays are never written as null
	var elemCheck string
	switch it := n.Items; {
	case it.Ref != "":
		d := refName(it.Ref)
		target, ok := g.defs[d]
		if !ok {
			return f, fmt.Errorf("%s.%s: unknown definition %s", def, name, d)
		}
		switch {
		case target.Enum != nil:
			f.decl = "[]" + qual + g.goNames[d]
			elemCheck = fmt.Sprintf("check.EachEnum(%q, %s)", name, v)
		case target.Type == "object" && d != "CustomDataType":
			f.decl = "[]" + qual + g.goNames[d]
			elemCheck = fmt.Sprintf("check.Each(%q, %s)", name, v)
		default:
			return f, fmt.Errorf("%s.%s: %w: array of %s", def, name, errUnsupported, d)
		}
	case it.Type == "string" && it.Format == "":
		f.decl = "[]string"
		max := "check.Unbounded"
		if it.MaxLength != nil {
			max = fmt.Sprint(*it.MaxLength)
		}
		elemCheck = fmt.Sprintf("check.EachString(%q, %s, %s)", name, v, max)
	case it.Type == "integer":
		f.decl = "[]int"
		if it.Minimum != nil || it.Maximum != nil {
			elemCheck = fmt.Sprintf("check.EachInt(%q, %s, %s, %s)", name, v, intLit(it.Minimum), intLit(it.Maximum))
		}
	default:
		return f, fmt.Errorf("%s.%s: %w: array of %s", def, name, errUnsupported, it.Type)
	}
	var checks []string
	if n.MinItems != nil || n.MaxItems != nil {
		// An absent optional array is fine; a present one must be the right size.
		fn := "check.Items"
		if !required {
			fn = "check.SliceItems"
			checks = append(checks, fmt.Sprintf("%s(%q, %s, %s, %s)", fn, name, v, itemBound(n.MinItems), itemBound(n.MaxItems)))
		} else {
			checks = append(checks, fmt.Sprintf("%s(%q, len(%s), %s, %s)", fn, name, v, itemBound(n.MinItems), itemBound(n.MaxItems)))
		}
	}
	if elemCheck != "" {
		checks = append(checks, elemCheck)
	}
	f.check = strings.Join(checks, ",\n\t\t")
	return f, nil
}

// structBody renders the fields and Validate method of an object schema.
func (g *gen) structBody(def, goName string, n *node, qual string) (string, error) {
	req := requiredSet(n)
	var fields strings.Builder
	var checks []string
	for _, p := range n.Props {
		f, err := g.field(def, p.Name, p.Node, req[p.Name], qual)
		if err != nil {
			return "", err
		}
		fields.WriteString(f.doc)
		fmt.Fprintf(&fields, "\t%s %s %s\n", f.goName, f.decl, f.tag)
		if f.check != "" {
			checks = append(checks, f.check)
		}
	}
	var sb strings.Builder
	sb.WriteString("{\n" + fields.String() + "}\n\n")
	return sb.String() + validateMethod(goName, checks), nil
}

func validateMethod(recv string, checks []string) string {
	if len(checks) == 0 {
		return fmt.Sprintf("// Validate checks the structure. It currently has nothing to check.\nfunc (v %s) Validate() error { return nil }\n\n", recv)
	}
	return fmt.Sprintf("// Validate checks required fields, lengths, ranges, sizes and enumeration values.\nfunc (v %s) Validate() error {\n\treturn check.First(\n\t\t%s,\n\t)\n}\n\n",
		recv, strings.Join(checks, ",\n\t\t"))
}

// codeOnly drops comment lines, so words in comments do not count as uses.
func codeOnly(src string) string {
	var sb strings.Builder
	for _, line := range strings.Split(src, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "//") {
			sb.WriteString(line + "\n")
		}
	}
	return sb.String()
}

func imports(body string, extra ...string) string {
	src := codeOnly(body)
	var list []string
	if strings.Contains(src, "json.RawMessage") {
		list = append(list, `"encoding/json"`)
	}
	list = append(list, extra...)
	if strings.Contains(src, "ocpp.") {
		list = append(list, `ocpp "github.com/foecum/go-ocpp-sdk"`)
	}
	if strings.Contains(src, "check.") {
		list = append(list, `"github.com/foecum/go-ocpp-sdk/internal/check"`)
	}
	if strings.Contains(src, "types.") {
		list = append(list, `"github.com/foecum/go-ocpp-sdk/v201/types"`)
	}
	if len(list) == 0 {
		return ""
	}
	return "import (\n\t" + strings.Join(list, "\n\t") + "\n)\n\n"
}

func (g *gen) emitTypes() error {
	var enums, structs strings.Builder
	for _, def := range g.order {
		if def == "CustomDataType" {
			continue
		}
		n, goName := g.defs[def], g.goNames[def]
		switch {
		case n.Enum != nil:
			if err := g.enumSource(&enums, def, goName, n); err != nil {
				return err
			}
		case n.Type == "object":
			body, err := g.structBody(def, goName, n, "")
			if err != nil {
				return err
			}
			structs.WriteString(comment("", goName, n.Description, goName+" is the "+def+" structure of OCPP 2.0.1."))
			structs.WriteString("type " + goName + " struct " + body)
		default:
			return fmt.Errorf("definition %s: %w: type %q", def, errUnsupported, n.Type)
		}
	}
	for file, body := range map[string]string{"enums.go": enums.String(), "structs.go": structs.String()} {
		src := header + "package types\n\n" + imports(body) + body
		out, err := render(src)
		if err != nil {
			return err
		}
		g.files["v201/types/"+file] = out
	}
	return nil
}

func (g *gen) enumSource(sb *strings.Builder, def, goName string, n *node) error {
	sb.WriteString(comment("", goName, n.Description, goName+" is the "+def+" enumeration of OCPP 2.0.1."))
	fmt.Fprintf(sb, "type %s string\n\nconst (\n", goName)
	seen := map[string]string{}
	var names []string
	for _, v := range n.Enum {
		id := goName + enumIdent(v)
		if other, clash := seen[id]; clash {
			return fmt.Errorf("%s: values %q and %q both become %s", def, other, v, id)
		}
		seen[id] = v
		names = append(names, id)
		fmt.Fprintf(sb, "\t// %s is the %q value of %s.\n\t%s %s = %q\n", id, v, goName, id, goName, v)
	}
	fmt.Fprintf(sb, ")\n\n// IsValid reports whether v is one of the values defined for %s.\nfunc (v %s) IsValid() bool {\n\tswitch v {\n\tcase %s:\n\t\treturn true\n\t}\n\treturn false\n}\n\n",
		goName, goName, strings.Join(names, ", "))
	return nil
}

func snake(name string) string { return strings.ToLower(strings.Join(words(name), "_")) }

func (g *gen) emitBlocks() error {
	covered := map[string]bool{}
	for _, b := range g.blocks {
		var pairs []string
		for _, m := range b.Messages {
			msg := g.messages[m]
			if msg == nil {
				return fmt.Errorf("block %s lists %s, which has no schema", b.Pkg, m)
			}
			covered[m] = true
			src, err := g.messageSource(b, msg)
			if err != nil {
				return err
			}
			out, err := render(src)
			if err != nil {
				return err
			}
			g.files["v201/"+b.Pkg+"/"+snake(m)+".go"] = out
			pairs = append(pairs, fmt.Sprintf("\t\tfeatureset.Pair{Req: %sRequest{}, Resp: %sResponse{}},\n", m, m))
		}
		feat := header + "package " + b.Pkg + "\n\nimport (\n\tocpp \"github.com/foecum/go-ocpp-sdk\"\n\t\"github.com/foecum/go-ocpp-sdk/internal/featureset\"\n)\n\n" +
			"// Features returns the feature of every message of this block, ready to register in an\n// ocppj.Registry.\nfunc Features() []ocpp.Feature {\n\treturn featureset.Build(\n" + strings.Join(pairs, "") + "\t)\n}\n"
		out, err := render(feat)
		if err != nil {
			return err
		}
		g.files["v201/"+b.Pkg+"/features.go"] = out
		out, err = render(header + blockDoc(b))
		if err != nil {
			return err
		}
		g.files["v201/"+b.Pkg+"/doc.go"] = out
		out, err = render(blockTest(b))
		if err != nil {
			return err
		}
		g.files["v201/"+b.Pkg+"/schema_test.go"] = out
	}
	if err := g.emitRoot(); err != nil {
		return err
	}
	var missing []string
	for m := range g.messages {
		if !covered[m] {
			missing = append(missing, m)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("messages not assigned to a block: %v", missing)
	}
	return nil
}

// blockTest renders the test that checks a block's messages against the schemas.
func blockTest(b Block) string {
	var actions []string
	for _, m := range b.Messages {
		actions = append(actions, fmt.Sprintf("%q", m))
	}
	return header + fmt.Sprintf(`package %[1]s_test

import (
	"testing"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/schematest"
	"github.com/foecum/go-ocpp-sdk/v201/%[1]s"
)

// TestAgainstOfficialSchemas round-trips sample payloads derived from the
// official JSON schemas and checks that the registry rejects whatever the
// schemas reject.
func TestAgainstOfficialSchemas(t *testing.T) {
	features := %[1]s.Features()
	schematest.RunGenerated(t, "../../docs/spec/2.0.1/schemas", schematest.NewRegistryFor(t, ocpp.V201, features),
		[]string{%[2]s})
	schematest.CheckActionsSuffix(t, features, "Response")
	if len(features) != %[3]d {
		t.Errorf("%%d features, want %[3]d", len(features))
	}
}
`, b.Pkg, strings.Join(actions, ", "), len(b.Messages))
}

func blockDoc(b Block) string {
	var sb strings.Builder
	var lines []string
	for _, m := range b.Messages {
		dir := map[Direction]string{FromStation: "sent by the Charging Station", FromCSMS: "sent by the CSMS", Both: "sent by either side"}[Directions[m]]
		lines = append(lines, "//   - "+m+" ("+dir+")")
	}
	fmt.Fprintf(&sb, "// Package %s implements the OCPP 2.0.1 functional block %s, %s.\n//\n// Messages:\n//\n%s\n//\n", b.Pkg, b.Letter, b.Title, strings.Join(lines, "\n"))
	sb.WriteString("// Each message is a Request and a Response type named after the specification. They\n// have Validate methods that check required fields, lengths, ranges, array sizes and\n// enumeration values, and Features registers them in an ocppj.Registry. Shared types are\n// in package v201/types.\n")
	fmt.Fprintf(&sb, "package %s\n", b.Pkg)
	return sb.String()
}

func (g *gen) messageSource(b Block, m *message) (string, error) {
	sender := map[Direction]string{FromStation: "the Charging Station", FromCSMS: "the CSMS", Both: "either side"}[Directions[m.Name]]
	var sb strings.Builder
	fmt.Fprintf(&sb, "// %sAction is the action name of the %s message.\nconst %sAction = %q\n\n", m.Name, m.Name, m.Name, m.Name)
	for _, kind := range []struct {
		suffix string
		n      *node
		who    string
	}{
		{"Request", m.Request, "sent by " + sender},
		{"Response", m.Response, "the answer to it"},
	} {
		name := m.Name + kind.suffix
		def := name
		body, err := g.structBody(def, name, kind.n, "types.")
		if err != nil {
			return "", err
		}
		doc := fmt.Sprintf("%s is the %s of the %s message (OCPP 2.0.1 block %s, %s), %s.",
			name, strings.ToLower(kind.suffix), m.Name, b.Letter, b.Title, kind.who)
		sb.WriteString(wrap("// ", doc))
		fmt.Fprintf(&sb, "type %s struct %s", name, body)
		fmt.Fprintf(&sb, "// Action returns the action name of the message.\nfunc (%s) Action() string { return %sAction }\n\n", name, m.Name)
	}
	body := sb.String()
	return header + "package " + b.Pkg + "\n\n" + imports(body) + body, nil
}

// emitRoot renders package v201 (the registry of every message) and the test of
// package types, which can only exercise the shared types through the messages.
func (g *gen) emitRoot() error {
	var imports, calls, actions []string
	for _, b := range g.blocks {
		imports = append(imports, fmt.Sprintf("\t\"github.com/foecum/go-ocpp-sdk/v201/%s\"", b.Pkg))
		calls = append(calls, fmt.Sprintf("\t\t%s.Features(),", b.Pkg))
		for _, m := range b.Messages {
			actions = append(actions, fmt.Sprintf("%q", m))
		}
	}
	sort.Strings(actions)
	root := header + `// Package v201 is the entry point for OCPP 2.0.1 (JSON). Its NewRegistry returns an
// ocppj.Registry with every OCPP 2.0.1 message registered, ready to hand to an
// ocppj.Endpoint.
//
// The messages live in one package per functional block of the specification
// (security, provisioning, authorization and so on); the types they share live in
// package types.
package v201

import (
	"fmt"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/ocppj"
` + strings.Join(imports, "\n") + `
)

// Features returns the features of every OCPP 2.0.1 message.
func Features() []ocpp.Feature {
	var all []ocpp.Feature
	for _, block := range [][]ocpp.Feature{
` + strings.Join(calls, "\n") + `
	} {
		all = append(all, block...)
	}
	return all
}

// NewRegistry returns a registry for ocpp.V201 containing every message.
func NewRegistry() (*ocppj.Registry, error) { return registryOf(Features()) }

func registryOf(features []ocpp.Feature) (*ocppj.Registry, error) {
	reg := ocppj.NewRegistry(ocpp.V201)
	for _, f := range features {
		if err := reg.Register(f); err != nil {
			return nil, fmt.Errorf("v201: %w", err)
		}
	}
	return reg, nil
}
`
	out, err := render(root)
	if err != nil {
		return err
	}
	g.files["v201/v201.go"] = out
	typesTest := header + `package types_test

import (
	"testing"

	"github.com/foecum/go-ocpp-sdk/internal/schematest"
	"github.com/foecum/go-ocpp-sdk/v201"
)

// TestSharedTypesAgainstOfficialSchemas runs every message through the schema
// tests. The shared types are exercised by all of them, which is the only way
// to reach every generated structure and enumeration.
func TestSharedTypesAgainstOfficialSchemas(t *testing.T) {
	reg, err := v201.NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	schematest.RunGenerated(t, "../../docs/spec/2.0.1/schemas", reg, []string{` + strings.Join(actions, ", ") + `})
}
`
	out, err = render(typesTest)
	if err != nil {
		return err
	}
	g.files["v201/types/schema_test.go"] = out
	return nil
}
