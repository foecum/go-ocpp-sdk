package ws

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

// VersionRoute sends the handshakes of one OCPP version to a handler, normally
// the CSMS of that version (csms/v16, csms/v201 or csms/v21).
type VersionRoute struct {
	Version ocpp.Version
	Handler http.Handler
}

// NewVersionRouter returns an http.Handler that serves charge points of several
// OCPP versions on one URL, which is what OCPP-J version negotiation is for: a
// fleet that mixes 1.6, 2.0.1 and 2.1 stations can use a single endpoint.
//
// Each handshake goes to the first route, in the order given (the server's
// preference), whose version the charge point offers in Sec-WebSocket-Protocol.
// If it offers none of them, the first route answers, which completes the
// handshake without a subprotocol and closes the connection as OCPP-J requires.
//
//	router, err := ws.NewVersionRouter(
//		ws.VersionRoute{Version: ocpp.V21, Handler: csms21},
//		ws.VersionRoute{Version: ocpp.V201, Handler: csms201},
//		ws.VersionRoute{Version: ocpp.V16, Handler: csms16},
//	)
//	http.Handle("/ocpp/", router)
//
// The CSMS of each version keeps its own connections: a station that reconnects
// with another version is a new connection in that version's CSMS, and its old
// connection ends on its own (by its close, or by the ping timeout).
func NewVersionRouter(routes ...VersionRoute) (http.Handler, error) {
	if len(routes) == 0 {
		return nil, errors.New("ws: a version router needs at least one route")
	}
	seen := map[ocpp.Version]bool{}
	for _, r := range routes {
		if !r.Version.IsValid() {
			return nil, fmt.Errorf("ws: unknown OCPP version %q", r.Version)
		}
		if r.Handler == nil {
			return nil, fmt.Errorf("ws: no handler for %s", r.Version)
		}
		if seen[r.Version] {
			return nil, fmt.Errorf("ws: %s is routed twice", r.Version)
		}
		seen[r.Version] = true
	}
	return versionRouter(append([]VersionRoute(nil), routes...)), nil
}

type versionRouter []VersionRoute

func (vr versionRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	offered := map[string]bool{}
	for _, header := range r.Header.Values("Sec-WebSocket-Protocol") {
		for _, p := range strings.Split(header, ",") {
			offered[strings.TrimSpace(p)] = true
		}
	}
	for _, route := range vr {
		if offered[route.Version.Subprotocol()] {
			route.Handler.ServeHTTP(w, r)
			return
		}
	}
	vr[0].Handler.ServeHTTP(w, r)
}
