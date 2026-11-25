package gen

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

func csmsSide(p *Profile) side {
	return side{
		dir: "csms/" + p.Dir, pkg: "csms", recv: "cp", handle: "ChargePoint", peer: "charging station",
		handles: func(d Direction) bool { return d == FromStation || d == Both },
		calls:   func(d Direction) bool { return d == FromCSMS || d == Both },
		cp:      true,
	}
}

func stationSide(p *Profile) side {
	return side{
		dir: "chargepoint/" + p.Dir, pkg: "chargepoint", recv: "c", handle: "ChargePoint", peer: "CSMS",
		handles: func(d Direction) bool { return d == FromCSMS || d == Both },
		calls:   func(d Direction) bool { return d == FromStation || d == Both },
	}
}

func (g *gen) emitHighLevel() error {
	for _, s := range []side{csmsSide(g.p), stationSide(g.p)} {
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

func (g *gen) blockImport(b Block) string {
	return fmt.Sprintf("\t\"github.com/foecum/go-ocpp-sdk/%s/%s\"\n", g.p.Dir, b.Pkg)
}

// messagesWhere returns the messages of b whose direction satisfies ok.
func (g *gen) messagesWhere(b Block, ok func(Direction) bool) []string {
	var out []string
	for _, m := range b.Messages {
		if ok(g.p.Directions[m]) {
			out = append(out, m)
		}
	}
	return out
}

func (g *gen) handlerSig(s side, b Block, m string, named bool) string {
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
	if g.p.OneWay[m] {
		msg := fmt.Sprintf("*%s.%s", b.Pkg, m)
		if named {
			msg = "msg " + msg
		}
		return fmt.Sprintf("(%s%s) error", args, msg)
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
		msgs := g.messagesWhere(b, s.handles)
		if len(msgs) == 0 {
			continue
		}
		imp.WriteString(g.blockImport(b))
		name := g.handlerName(b.Pkg) + "Handler"
		fmt.Fprintf(&body, "// %s handles the %s calls (OCPP %s block %s) that the %s sends. Embed\n// Unimplemented%s to implement only some of them.\ntype %s interface {\n",
			name, b.Title, g.p.Name, b.Letter, s.peer, name, name)
		for _, m := range msgs {
			fmt.Fprintf(&body, "\t// %s answers a %s request.\n\t%s%s\n", m, m, m, g.handlerSig(s, b, m, true))
		}
		body.WriteString("}\n\n")
		fmt.Fprintf(&body, "// Unimplemented%[1]s implements %[1]s by answering every call with a NotSupported\n// CALLERROR. Embed it in your handler and override what you support.\ntype Unimplemented%[1]s struct{}\n\nvar _ %[1]s = Unimplemented%[1]s{}\n\n", name)
		for _, m := range msgs {
			ret := "nil, "
			if g.p.OneWay[m] {
				ret = ""
			}
			fmt.Fprintf(&body, "// %s returns NotSupported.\nfunc (Unimplemented%s) %s%s {\n\treturn %srpc.NotSupported(%s.%sAction)\n}\n\n",
				m, name, m, g.handlerSig(s, b, m, false), ret, b.Pkg, m)
		}
	}
	imp.WriteString(")\n\n")
	return g.header() + "package " + s.pkg + "\n\n" + imp.String() + body.String()
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
		msgs := g.messagesWhere(b, s.handles)
		if len(msgs) == 0 {
			continue
		}
		imp.WriteString(g.blockImport(b))
		for _, m := range msgs {
			if g.p.OneWay[m] {
				continue
			}
			fmt.Fprintf(&body, "\t\tcase *%s.%sRequest:\n\t\t\tif h := %s.cfg.%s; h != nil {\n\t\t\t\treturn rpc.Return(h.%s(%sr))\n\t\t\t}\n",
				b.Pkg, m, recv, g.handlerName(b.Pkg), m, args)
		}
	}
	body.WriteString("\t\t}\n\t\treturn nil, rpc.NotSupported(req.Action())\n\t})\n}\n")
	if len(g.p.OneWay) > 0 {
		g.dispatchSendSource(s, &body, &imp, recv, typ)
	}
	imp.WriteString(")\n\n")
	return g.header() + "package " + s.pkg + "\n\n" + imp.String() + body.String()
}

// dispatchSendSource adds dispatchSend, which routes an incoming unconfirmed
// (SEND) message to the handler of its functional block.
func (g *gen) dispatchSendSource(s side, body, imp *strings.Builder, recv, typ string) {
	args, params := "ctx, ", "ctx context.Context, msg ocpp.Request"
	if s.cp {
		args, params = "ctx, cp, ", "ctx context.Context, cp *ChargePoint, msg ocpp.Request"
	}
	var cases strings.Builder
	for _, b := range g.blocks {
		for _, m := range g.messagesWhere(b, s.handles) {
			if !g.p.OneWay[m] {
				continue
			}
			fmt.Fprintf(&cases, "\t\tcase *%s.%s:\n\t\t\tif h := %s.cfg.%s; h != nil {\n\t\t\t\treturn h.%s(%sm)\n\t\t\t}\n",
				b.Pkg, m, recv, g.handlerName(b.Pkg), m, args)
		}
	}
	bind := ""
	if cases.Len() > 0 {
		bind = "m := "
	}
	fmt.Fprintf(body, "\n// dispatchSend routes an incoming unconfirmed (SEND) message to the handler of its\n// functional block. It is never answered; an error is only logged.\nfunc (%s *%s) dispatchSend(%s) error {\n\treturn rpc.SafeSend(%s.log, msg.Action(), func() error {\n\t\tswitch %smsg.(type) {\n%s",
		recv, typ, params, recv, bind, cases.String())
	body.WriteString("\t\t}\n\t\treturn rpc.NotSupported(msg.Action())\n\t})\n}\n")
}

func (g *gen) callsSource(s side) string {
	var body, imp strings.Builder
	imp.WriteString("import (\n\t\"context\"\n\n")
	for _, b := range g.blocks {
		msgs := g.messagesWhere(b, s.calls)
		if len(msgs) == 0 {
			continue
		}
		imp.WriteString(g.blockImport(b))
		for _, m := range msgs {
			if g.p.OneWay[m] {
				fmt.Fprintf(&body, "// %[1]s sends the unconfirmed %[1]s message (OCPP %[7]s block %[2]s) to the %[3]s. The %[3]s never\n// answers it, so it returns as soon as the message is written.\nfunc (%[4]s *%[5]s) %[1]s(ctx context.Context, msg %[6]s.%[1]s) error {\n\treturn send(ctx, %[4]s, msg)\n}\n\n",
					m, b.Letter, s.peer, s.recv, s.handle, b.Pkg, g.p.Name)
				continue
			}
			fmt.Fprintf(&body, "// %[1]s sends a %[1]s request (OCPP %[7]s block %[2]s) to the %[3]s and returns its answer.\nfunc (%[4]s *%[5]s) %[1]s(ctx context.Context, req %[6]s.%[1]sRequest) (*%[6]s.%[1]sResponse, error) {\n\treturn call[%[6]s.%[1]sResponse](ctx, %[4]s, req)\n}\n\n",
				m, b.Letter, s.peer, s.recv, s.handle, b.Pkg, g.p.Name)
		}
	}
	imp.WriteString(")\n\n")
	return g.header() + "package " + s.pkg + "\n\n" + imp.String() + body.String()
}

// handlerName returns the name used in the high-level APIs for a block package:
// the entry of HandlerNames, or the capitalised package name for a block without one.
func (g *gen) handlerName(pkg string) string {
	if n, ok := g.p.HandlerNames[pkg]; ok {
		return n
	}
	return strings.ToUpper(pkg[:1]) + pkg[1:]
}
