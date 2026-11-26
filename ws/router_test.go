package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

func TestNewVersionRouterValidation(t *testing.T) {
	h := http.NotFoundHandler()
	for name, routes := range map[string][]VersionRoute{
		"no routes":       nil,
		"unknown version": {{Version: "ocpp9", Handler: h}},
		"no handler":      {{Version: ocpp.V16}},
		"twice":           {{Version: ocpp.V16, Handler: h}, {Version: ocpp.V16, Handler: h}},
	} {
		if _, err := NewVersionRouter(routes...); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

// Stations of three versions connect to one URL and each reaches the server of
// its version; the server's preference decides when a station offers several.
func TestVersionRouterRoutesByOfferedSubprotocol(t *testing.T) {
	servers := map[ocpp.Version]*recorder{}
	var routes []VersionRoute
	for _, v := range []ocpp.Version{ocpp.V21, ocpp.V201, ocpp.V16} {
		rec := newRecorder()
		cfg := ServerConfig{Versions: []ocpp.Version{v}}
		rec.apply(&cfg)
		s, err := NewServer(cfg)
		if err != nil {
			t.Fatal(err)
		}
		servers[v] = rec
		routes = append(routes, VersionRoute{Version: v, Handler: s})
	}
	router, err := NewVersionRouter(routes...)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(router)
	defer ts.Close()
	base := "ws" + strings.TrimPrefix(ts.URL, "http")

	for _, tc := range []struct {
		id     string
		offers []string
		want   ocpp.Version
	}{
		{"A", []string{"ocpp1.6"}, ocpp.V16},
		{"B", []string{"ocpp2.0.1"}, ocpp.V201},
		{"C", []string{"ocpp2.0.1", "ocpp2.1"}, ocpp.V21}, // the server prefers 2.1
		{"D", []string{"ocpp1.5", "ocpp1.6"}, ocpp.V16},
	} {
		c := mustDial(t, base, tc.id, tc.offers...)
		if c.Subprotocol() != tc.want.Subprotocol() {
			t.Errorf("%s: negotiated %q, want %s", tc.id, c.Subprotocol(), tc.want)
		}
		if conn := recv(t, servers[tc.want].connected); conn.ID() != tc.id {
			t.Errorf("%s reached the %s server as %s", tc.id, tc.want, conn.ID())
		}
	}
	// No common version: the handshake completes without a subprotocol and the
	// connection is closed.
	c, _, err := rawDial(t, base, "E", nil, "ocpp1.2")
	if err != nil {
		t.Fatal(err)
	}
	if c.Subprotocol() != "" {
		t.Errorf("negotiated %q", c.Subprotocol())
	}
}
