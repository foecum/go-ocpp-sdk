package gen201

import (
	"fmt"
	"strings"
)

// side describes one of the two high-level APIs generated from the blocks.
type side struct {
	dir     string // output directory
	pkg     string // Go package name
	recv    string // receiver variable of the public handle type
	handle  string // public handle type name
	peer    string // what the other side is called in comments
	handles func(Direction) bool
	calls   func(Direction) bool
	// cp is true when handler methods take the connected charge point.
	cp bool
}

var csmsSide = side{
	dir: "csms/v201", pkg: "csms", recv: "cp", handle: "ChargePoint", peer: "charging station",
	handles: func(d Direction) bool { return d == FromStation || d == Both },
	calls:   func(d Direction) bool { return d == FromCSMS || d == Both },
	cp:      true,
}

var stationSide = side{
	dir: "chargepoint/v201", pkg: "chargepoint", recv: "c", handle: "ChargePoint", peer: "CSMS",
	handles: func(d Direction) bool { return d == FromCSMS || d == Both },
	calls:   func(d Direction) bool { return d == FromStation || d == Both },
}

func (g *gen) emitHighLevel() error {
	for _, s := range []side{csmsSide, stationSide} {
		for name, build := range map[string]func(side) string{
			"handlers_gen.go": g.handlersSource,
			"dispatch_gen.go": g.dispatchSource,
			"calls_gen.go":    g.callsSource,
		} {
			out, err := render(build(s))
			if err != nil {
				return err
			}
			g.files[s.dir+"/"+name] = out
		}
	}
	return nil
}

func blockImport(b Block) string {
	return fmt.Sprintf("\t\"github.com/foecum/go-ocpp-sdk/v201/%s\"\n", b.Pkg)
}

// messagesWhere returns the messages of b whose direction satisfies ok.
func messagesWhere(b Block, ok func(Direction) bool) []string {
	var out []string
	for _, m := range b.Messages {
		if ok(Directions[m]) {
			out = append(out, m)
		}
	}
	return out
}

func (s side) handlerSig(b Block, m string, named bool) string {
	args := "ctx context.Context, "
	if !named {
		args = "context.Context, "
	}
	if s.cp {
		if named {
			args += "cp *ChargePoint, "
		} else {
			args += "*ChargePoint, "
		}
	}
	req := fmt.Sprintf("*%s.%sRequest", b.Pkg, m)
	if named {
		req = "req " + req
	}
	return fmt.Sprintf("(%s%s) (*%s.%sResponse, error)", args, req, b.Pkg, m)
}

func (g *gen) handlersSource(s side) string {
	var body, imp strings.Builder
	imp.WriteString("import (\n\t\"context\"\n\n\t\"github.com/foecum/go-ocpp-sdk/internal/rpc\"\n")
	for _, b := range g.blocks {
		msgs := messagesWhere(b, s.handles)
		if len(msgs) == 0 {
			continue
		}
		imp.WriteString(blockImport(b))
		name := handlerName(b.Pkg) + "Handler"
		fmt.Fprintf(&body, "// %s handles the %s calls (OCPP 2.0.1 block %s) that the %s sends. Embed\n// Unimplemented%s to implement only some of them.\ntype %s interface {\n",
			name, b.Title, b.Letter, s.peer, name, name)
		for _, m := range msgs {
			fmt.Fprintf(&body, "\t// %s answers a %s request.\n\t%s%s\n", m, m, m, s.handlerSig(b, m, true))
		}
		body.WriteString("}\n\n")
		fmt.Fprintf(&body, "// Unimplemented%[1]s implements %[1]s by answering every call with a NotSupported\n// CALLERROR. Embed it in your handler and override what you support.\ntype Unimplemented%[1]s struct{}\n\nvar _ %[1]s = Unimplemented%[1]s{}\n\n", name)
		for _, m := range msgs {
			fmt.Fprintf(&body, "// %s returns NotSupported.\nfunc (Unimplemented%s) %s%s {\n\treturn nil, rpc.NotSupported(%s.%sAction)\n}\n\n",
				m, name, m, s.handlerSig(b, m, false), b.Pkg, m)
		}
	}
	imp.WriteString(")\n\n")
	return header + "package " + s.pkg + "\n\n" + imp.String() + body.String()
}

func (g *gen) dispatchSource(s side) string {
	var body, imp strings.Builder
	imp.WriteString("import (\n\t\"context\"\n\n\tocpp \"github.com/foecum/go-ocpp-sdk\"\n\t\"github.com/foecum/go-ocpp-sdk/internal/rpc\"\n")
	args, params := "ctx, ", "ctx context.Context, req ocpp.Request"
	if s.cp {
		args, params = "ctx, cp, ", "ctx context.Context, cp *ChargePoint, req ocpp.Request"
	}
	recv := "s"
	typ := "CSMS"
	if !s.cp {
		recv, typ = "c", "ChargePoint"
	}
	fmt.Fprintf(&body, "// dispatch routes an incoming request to the handler of its functional block. A\n// request the SDK knows but nobody handles is answered with NotSupported.\nfunc (%s *%s) dispatch(%s) (ocpp.Response, error) {\n\treturn rpc.Safe(%s.log, req.Action(), func() (ocpp.Response, error) {\n\t\tswitch r := req.(type) {\n",
		recv, typ, params, recv)
	for _, b := range g.blocks {
		msgs := messagesWhere(b, s.handles)
		if len(msgs) == 0 {
			continue
		}
		imp.WriteString(blockImport(b))
		for _, m := range msgs {
			fmt.Fprintf(&body, "\t\tcase *%s.%sRequest:\n\t\t\tif h := %s.cfg.%s; h != nil {\n\t\t\t\treturn rpc.Return(h.%s(%sr))\n\t\t\t}\n",
				b.Pkg, m, recv, handlerName(b.Pkg), m, args)
		}
	}
	body.WriteString("\t\t}\n\t\treturn nil, rpc.NotSupported(req.Action())\n\t})\n}\n")
	imp.WriteString(")\n\n")
	return header + "package " + s.pkg + "\n\n" + imp.String() + body.String()
}

func (g *gen) callsSource(s side) string {
	var body, imp strings.Builder
	imp.WriteString("import (\n\t\"context\"\n\n")
	for _, b := range g.blocks {
		msgs := messagesWhere(b, s.calls)
		if len(msgs) == 0 {
			continue
		}
		imp.WriteString(blockImport(b))
		for _, m := range msgs {
			fmt.Fprintf(&body, "// %[1]s sends a %[1]s request (OCPP 2.0.1 block %[2]s) to the %[3]s and returns its answer.\nfunc (%[4]s *%[5]s) %[1]s(ctx context.Context, req %[6]s.%[1]sRequest) (*%[6]s.%[1]sResponse, error) {\n\treturn call[%[6]s.%[1]sResponse](ctx, %[4]s, req)\n}\n\n",
				m, b.Letter, s.peer, s.recv, s.handle, b.Pkg)
		}
	}
	imp.WriteString(")\n\n")
	return header + "package " + s.pkg + "\n\n" + imp.String() + body.String()
}

// handlerName returns the name used in the high-level APIs for a block package:
// the entry of HandlerNames, or the capitalised package name for a block without one.
func handlerName(pkg string) string {
	if n, ok := HandlerNames[pkg]; ok {
		return n
	}
	return strings.ToUpper(pkg[:1]) + pkg[1:]
}
