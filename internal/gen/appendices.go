package gen

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// extraSet is one of the value lists that OCPP 2.1 keeps in its appendices
// because the schemas carry them as plain strings (idToken types, connector
// types and so on).
type extraSet struct {
	// File is the CSV file name, Name the Go identifier prefix of its constants
	// and of the list they are collected in (plural).
	File, Name, List string
	// Doc says what the values are.
	Doc string
}

// readCSV reads a semicolon separated appendix file without its header row.
func readCSV(path string) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = ';'
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	var out [][]string
	for i, row := range rows {
		if i == 0 {
			if len(row) > 0 {
				row[0] = strings.TrimPrefix(row[0], "\xef\xbb\xbf")
			}
			continue // header
		}
		for j := range row {
			row[j] = strings.TrimSpace(row[j])
		}
		if len(row) == 0 || (len(row) == 1 && row[0] == "") {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

func cell(row []string, i int) string {
	if i < len(row) {
		return row[i]
	}
	return ""
}

// firstSentence shortens a CSV description to a one-line doc comment text.
func firstSentence(s string) string {
	s = strings.Join(strings.Fields(strings.ReplaceAll(s, "\\'", "'")), " ")
	if i := strings.Index(s, ". "); i > 0 {
		s = s[:i+1]
	}
	if len(s) > 220 {
		s = s[:217] + "..."
	}
	return s
}

// constName builds a unique constant name, failing when two values collide.
func constName(used map[string]string, prefix, value string) (string, error) {
	id := enumIdent(value)
	if id == "" {
		return "", fmt.Errorf("value %q has no letters or digits", value)
	}
	name := prefix + id
	if other, clash := used[name]; clash {
		return "", fmt.Errorf("values %q and %q both become %s", other, value, name)
	}
	used[name] = value
	return name, nil
}

func quote(s string) string { return fmt.Sprintf("%q", s) }

// emitAppendices renders package appendices of the profile from its CSV files.
func (g *gen) emitAppendices(csvDir string) error {
	files := map[string]func(dir string) (string, error){
		"components.go":     g.componentsSource,
		"variables.go":      g.variablesSource,
		"devicemodel.go":    g.deviceModelSource,
		"units.go":          g.unitsSource,
		"reasoncodes.go":    g.reasonCodesSource,
		"securityevents.go": g.securityEventsSource,
	}
	for _, set := range g.p.Extra {
		set := set
		files[strings.ToLower(set.Name)+"s.go"] = func(dir string) (string, error) { return g.extraSource(dir, set) }
	}
	var names []string
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		body, err := files[name](csvDir)
		if err != nil {
			return fmt.Errorf("appendices %s: %w", name, err)
		}
		out, err := render(g.header() + "package appendices\n\n" + body)
		if err != nil {
			return err
		}
		g.files[g.p.Dir+"/appendices/"+name] = out
	}
	doc := fmt.Sprintf(`// Package appendices holds the standardized names that OCPP %[1]s part 2 lists in its
// appendices, as constants and tables: components and variables of the device model,
// units of measure, reason codes and security events%[2]s.
//
// The OCPP messages carry these names as plain strings, because a vendor may add its own.
// The constants save typing them and the tables let an application check what is standard.
// Use them like any string:
//
//	provisioning.GetVariablesRequest{GetVariableData: []types.GetVariableData{{
//		Component: types.Component{Name: appendices.ComponentOCPPCommCtrlr},
//		Variable:  types.Variable{Name: appendices.VariableHeartbeatInterval},
//	}}}
//
// The files are generated from the CSV files that accompany the specification.
package appendices
`, g.p.Name, map[bool]string{true: ", and the value lists that 2.1 keeps outside its schemas", false: ""}[len(g.p.Extra) > 0])
	out, err := render(g.header() + doc)
	if err != nil {
		return err
	}
	g.files[g.p.Dir+"/appendices/doc.go"] = out
	return nil
}

func (g *gen) componentsSource(dir string) (string, error) {
	rows, err := readCSV(filepath.Join(dir, "components.csv"))
	if err != nil {
		return "", err
	}
	used := map[string]string{}
	var consts, list strings.Builder
	for _, row := range rows {
		name, desc := cell(row, 0), cell(row, 1)
		if name == "" {
			continue
		}
		id, err := constName(used, "Component", name)
		if err != nil {
			return "", err
		}
		consts.WriteString(wrap("\t// ", id+" is the standardized component "+name+". "+firstSentence(desc)))
		fmt.Fprintf(&consts, "\t%s = %s\n", id, quote(name))
		fmt.Fprintf(&list, "\t%s,\n", id)
	}
	return "// Components of the device model (part 2 appendix 3.1).\nconst (\n" + consts.String() + ")\n\n// Components lists every standardized component, in the order of the appendix.\nvar Components = []string{\n" + list.String() + "}\n", nil
}

func (g *gen) variablesSource(dir string) (string, error) {
	rows, err := readCSV(filepath.Join(dir, "variables.csv"))
	if err != nil {
		return "", err
	}
	used := map[string]string{}
	seen := map[string]bool{}
	var consts, table strings.Builder
	for _, row := range rows {
		name := cell(row, 0)
		if name == "" || seen[name] {
			continue // the appendix lists one variable twice
		}
		seen[name] = true
		id, err := constName(used, "Variable", name)
		if err != nil {
			return "", err
		}
		consts.WriteString(wrap("\t// ", id+" is the standardized variable "+name+". "+firstSentence(cell(row, 3))))
		fmt.Fprintf(&consts, "\t%s = %s\n", id, quote(name))
		fmt.Fprintf(&table, "\t{Name: %s, DataType: %s, Unit: %s},\n", id, quote(cell(row, 1)), quote(cell(row, 2)))
	}
	return "// Variables of the device model (part 2 appendix 3.2).\nconst (\n" + consts.String() + ")\n\n" +
		"// VariableInfo describes a standardized variable: its name, the data type of its values\n// and the unit they are in, if any.\ntype VariableInfo struct {\n\tName     string\n\tDataType string\n\tUnit     string\n}\n\n" +
		"// Variables lists every standardized variable, in the order of the appendix.\nvar Variables = []VariableInfo{\n" + table.String() + "}\n", nil
}

func (g *gen) deviceModelSource(dir string) (string, error) {
	rows, err := readCSV(filepath.Join(dir, "dm_components_vars.csv"))
	if err != nil {
		return "", err
	}
	var table strings.Builder
	for _, row := range rows {
		comp, variable := cell(row, 0), cell(row, 1)
		if comp == "" || variable == "" {
			continue
		}
		required := strings.EqualFold(cell(row, 3), "yes")
		fmt.Fprintf(&table, "\t{Component: %s, Variable: %s, Instance: %s, Required: %v, DataType: %s, Unit: %s},\n",
			quote(comp), quote(variable), quote(cell(row, 2)), required, quote(cell(row, 4)), quote(cell(row, 5)))
	}
	return "// Generic is the Component of the DeviceModel entries that apply to every component.\nconst Generic = \"<generic>\"\n\n" +
		"// DeviceModelEntry says that a component has a variable, and whether implementing it is\n// required. Component is Generic for variables any component may have.\ntype DeviceModelEntry struct {\n\tComponent string\n\tVariable  string\n\tInstance  string\n\tRequired  bool\n\tDataType  string\n\tUnit      string\n}\n\n" +
		"// DeviceModel lists the variables of each standardized component (part 2 appendix 3.3).\nvar DeviceModel = []DeviceModelEntry{\n" + table.String() + "}\n", nil
}

func (g *gen) unitsSource(dir string) (string, error) {
	rows, err := readCSV(filepath.Join(dir, "units_of_measure.csv"))
	if err != nil {
		return "", err
	}
	used := map[string]string{}
	var consts, list strings.Builder
	for _, row := range rows {
		value := cell(row, 0)
		if value == "" {
			continue
		}
		// Units are case sensitive and several differ only in case (for
		// example "var" and "VA"), so a lower-case value keeps its case.
		id, err := constName(used, "Unit", value)
		if err != nil {
			id, err = constName(used, "Unit", value+"Lower")
			if err != nil {
				return "", err
			}
		}
		consts.WriteString(wrap("\t// ", id+" is the unit "+value+": "+firstSentence(cell(row, 1))))
		fmt.Fprintf(&consts, "\t%s = %s\n", id, quote(value))
		fmt.Fprintf(&list, "\t%s,\n", id)
	}
	return "// Units of measure (part 2 appendix 3.4), for the unit of a measured value.\nconst (\n" + consts.String() + ")\n\n// Units lists every standardized unit of measure.\nvar Units = []string{\n" + list.String() + "}\n", nil
}

func (g *gen) reasonCodesSource(dir string) (string, error) {
	rows, err := readCSV(filepath.Join(dir, "reason_codes.csv"))
	if err != nil {
		return "", err
	}
	used := map[string]string{}
	var consts, table strings.Builder
	group := ""
	for _, row := range rows {
		code := cell(row, 1)
		if code == "" {
			if cell(row, 0) != "" {
				group = cell(row, 0) // a row that only names the group of the codes below it
			}
			continue
		}
		id, err := constName(used, "ReasonCode", code)
		if err != nil {
			return "", err
		}
		consts.WriteString(wrap("\t// ", id+" ("+group+"): "+firstSentence(cell(row, 2))))
		fmt.Fprintf(&consts, "\t%s = %s\n", id, quote(code))
		fmt.Fprintf(&table, "\t{Code: %s, Group: %s, UsedFor: %s},\n", id, quote(group), quote(cell(row, 3)))
	}
	return "// Reason codes for the reasonCode of a statusInfo (part 2 appendix 5).\nconst (\n" + consts.String() + ")\n\n" +
		"// ReasonCodeInfo describes a standardized reason code: the group it belongs to and the\n// messages it is typically used for.\ntype ReasonCodeInfo struct {\n\tCode    string\n\tGroup   string\n\tUsedFor string\n}\n\n" +
		"// ReasonCodes lists every standardized reason code, in the order of the appendix.\nvar ReasonCodes = []ReasonCodeInfo{\n" + table.String() + "}\n", nil
}

func (g *gen) securityEventsSource(dir string) (string, error) {
	rows, err := readCSV(filepath.Join(dir, "security_events.csv"))
	if err != nil {
		return "", err
	}
	used := map[string]string{}
	var consts, table strings.Builder
	for _, row := range rows {
		name := cell(row, 0)
		if name == "" {
			continue
		}
		id, err := constName(used, "SecurityEvent", name)
		if err != nil {
			return "", err
		}
		consts.WriteString(wrap("\t// ", id+": "+firstSentence(cell(row, 1))))
		fmt.Fprintf(&consts, "\t%s = %s\n", id, quote(name))
		fmt.Fprintf(&table, "\t{Name: %s, Critical: %v},\n", id, strings.EqualFold(cell(row, 2), "yes"))
	}
	return "// Security events for a SecurityEventNotification (part 2 appendix 1).\nconst (\n" + consts.String() + ")\n\n" +
		"// SecurityEventInfo describes a standardized security event. A critical event is also\n// to be reported to the CSMS even when it is offline for a while: it must be stored and sent.\ntype SecurityEventInfo struct {\n\tName     string\n\tCritical bool\n}\n\n" +
		"// SecurityEvents lists every standardized security event, in the order of the appendix.\nvar SecurityEvents = []SecurityEventInfo{\n" + table.String() + "}\n\n" +
		"// IsKnownSecurityEvent reports whether name is a standardized security event.\nfunc IsKnownSecurityEvent(name string) bool {\n\tfor _, e := range SecurityEvents {\n\t\tif e.Name == name {\n\t\t\treturn true\n\t\t}\n\t}\n\treturn false\n}\n\n" +
		"// IsCriticalSecurityEvent reports whether name is a standardized security event that is\n// marked critical.\nfunc IsCriticalSecurityEvent(name string) bool {\n\tfor _, e := range SecurityEvents {\n\t\tif e.Name == name {\n\t\t\treturn e.Critical\n\t\t}\n\t}\n\treturn false\n}\n", nil
}

func (g *gen) extraSource(dir string, set extraSet) (string, error) {
	rows, err := readCSV(filepath.Join(dir, set.File))
	if err != nil {
		return "", err
	}
	used := map[string]string{}
	var consts, list strings.Builder
	for _, row := range rows {
		value := cell(row, 0)
		if value == "" {
			continue
		}
		id, err := constName(used, set.Name, value)
		if err != nil {
			return "", err
		}
		text := id + " is the " + set.Doc + " " + value + "."
		if d := firstSentence(cell(row, 1)); d != "" && !strings.EqualFold(d, "yes") {
			text += " " + d
		}
		consts.WriteString(wrap("\t// ", text))
		fmt.Fprintf(&consts, "\t%s = %s\n", id, quote(value))
		fmt.Fprintf(&list, "\t%s,\n", id)
	}
	return "// The " + set.Doc + " values of OCPP 2.1 part 2 (appendices).\nconst (\n" + consts.String() + ")\n\n// " + set.List + " lists the standardized " + set.Doc + " values.\nvar " + set.List + " = []string{\n" + list.String() + "}\n", nil
}
