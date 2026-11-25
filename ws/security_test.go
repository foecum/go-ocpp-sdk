package ws

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

func basicHeader(user, pass string) http.Header {
	h := http.Header{}
	r := &http.Request{Header: h}
	r.SetBasicAuth(user, pass)
	return h
}

func TestBasicAuthProfile1(t *testing.T) {
	creds := map[string]string{"CS1": "0123456789abcdef", "CS2": "another-password-16"}
	_, rec, base := startServer(t, func(c *ServerConfig) {
		c.Authenticator = BasicAuthenticator(func(id string) (string, bool) { p, ok := creds[id]; return p, ok })
	})
	tests := []struct {
		name   string
		id     string
		header http.Header
		want   int // 0 = connects
	}{
		{"valid", "CS1", basicHeader("CS1", "0123456789abcdef"), 0},
		{"wrong password", "CS1", basicHeader("CS1", "nope"), 401},
		{"empty password", "CS1", basicHeader("CS1", ""), 401},
		{"no header", "CS1", nil, 401},
		{"username is another charge point", "CS1", basicHeader("CS2", "another-password-16"), 401},
		{"unknown identity", "CS9", basicHeader("CS9", "x"), 404},
		{"not basic", "CS1", http.Header{"Authorization": {"Bearer abc"}}, 401},
	}
	for _, tt := range tests {
		c, resp, err := rawDial(t, base, tt.id, tt.header, "ocpp2.0.1")
		if tt.want == 0 {
			if err != nil || c == nil {
				t.Errorf("%s: %v", tt.name, err)
			} else {
				recv(t, rec.connected)
			}
			continue
		}
		if err == nil || resp == nil || resp.StatusCode != tt.want {
			t.Errorf("%s: got %v %v", tt.name, resp, err)
		}
	}
}

func TestChainAuthenticators(t *testing.T) {
	var calls []string
	mk := func(name string, err error) Authenticator {
		return func(*http.Request, string) error { calls = append(calls, name); return err }
	}
	r := httptest.NewRequest("GET", "/x", nil)
	if err := ChainAuthenticators(mk("a", nil), mk("b", nil))(r, "id"); err != nil || len(calls) != 2 {
		t.Errorf("all pass: %v %v", err, calls)
	}
	calls = nil
	if err := ChainAuthenticators(mk("a", nil), mk("b", ErrForbidden), mk("c", nil))(r, "id"); !errors.Is(err, ErrForbidden) || len(calls) != 2 {
		t.Errorf("stop at first error: %v %v", err, calls)
	}
	if err := ChainAuthenticators()(r, "id"); err != nil {
		t.Errorf("empty chain: %v", err)
	}
}

func TestTLSConfigs(t *testing.T) {
	ca := newTestCA(t, "CA")
	srvCert := ca.issue(t, "localhost", "CSMS", true)
	s := ServerTLSConfig(srvCert, nil)
	if s.MinVersion != tls.VersionTLS12 || s.ClientAuth != tls.NoClientCert || len(s.CipherSuites) != len(CipherSuites()) {
		t.Errorf("profile 2 server config: %+v", s)
	}
	s = ServerTLSConfig(srvCert, ca.pool)
	if s.ClientAuth != tls.RequireAndVerifyClientCert || s.ClientCAs != ca.pool {
		t.Errorf("profile 3 server config: %+v", s)
	}
	cc := ca.issue(t, "CS1", "CSO", false)
	c := ClientTLSConfig(ca.pool, &cc)
	if c.MinVersion != tls.VersionTLS12 || len(c.Certificates) != 1 || c.RootCAs != ca.pool {
		t.Errorf("client config: %+v", c)
	}
	if c := ClientTLSConfig(nil, nil); len(c.Certificates) != 0 {
		t.Error("no client certificate expected")
	}
	// Every listed suite is a real TLS 1.2 suite.
	known := map[uint16]bool{}
	for _, cs := range tls.CipherSuites() {
		known[cs.ID] = true
	}
	for _, cs := range tls.InsecureCipherSuites() {
		known[cs.ID] = true
	}
	for _, id := range CipherSuites() {
		if !known[id] {
			t.Errorf("unknown suite %#x", id)
		}
	}
}

// startTLSServer serves s over TLS using cfg and returns the wss:// base URL.
func startTLSServer(t testing.TB, mutate func(*ServerConfig), cfg *tls.Config) (*recorder, string) {
	t.Helper()
	rec := newRecorder()
	sc := ServerConfig{Versions: []ocpp.Version{ocpp.V201}}
	rec.apply(&sc)
	if mutate != nil {
		mutate(&sc)
	}
	s, err := NewServer(sc)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewUnstartedServer(s)
	ts.TLS = cfg
	ts.StartTLS()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
		ts.Close()
	})
	return rec, "wss" + strings.TrimPrefix(ts.URL, "https")
}

func tlsDial(t testing.TB, base, id string, header http.Header, cfg *tls.Config) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	hc := &http.Client{Transport: &http.Transport{TLSClientConfig: cfg}}
	c, resp, err := websocket.Dial(ctx, base+"/ocpp/"+id, &websocket.DialOptions{
		HTTPClient: hc, HTTPHeader: header, Subprotocols: []string{"ocpp2.0.1"},
	})
	if c != nil {
		t.Cleanup(func() { _ = c.CloseNow() })
	}
	return c, resp, err
}

func TestTLSBasicAuthProfile2(t *testing.T) {
	ca := newTestCA(t, "CA")
	rec, base := startTLSServer(t, func(c *ServerConfig) {
		c.Authenticator = BasicAuthenticator(func(id string) (string, bool) { return "0123456789abcdef", true })
	}, ServerTLSConfig(ca.issue(t, "localhost", "CSMS", true), nil))
	auth := basicHeader("CS1", "0123456789abcdef")

	if _, _, err := tlsDial(t, base, "CS1", auth, ClientTLSConfig(ca.pool, nil)); err != nil {
		t.Fatalf("trusted CA: %v", err)
	}
	recv(t, rec.connected)

	if _, _, err := tlsDial(t, base, "CS1", auth, ClientTLSConfig(newTestCA(t, "other").pool, nil)); err == nil {
		t.Error("an untrusted server certificate must be rejected")
	}
	if _, resp, err := tlsDial(t, base, "CS1", basicHeader("CS1", "bad"), ClientTLSConfig(ca.pool, nil)); err == nil || resp.StatusCode != 401 {
		t.Errorf("bad password over TLS: %v %v", resp, err)
	}
}

func TestTLSRejectsOldVersionsAndWeakSuites(t *testing.T) {
	ca := newTestCA(t, "CA")
	_, base := startTLSServer(t, nil, ServerTLSConfig(ca.issue(t, "localhost", "CSMS", true), nil))

	old := ClientTLSConfig(ca.pool, nil)
	old.MinVersion, old.MaxVersion = tls.VersionTLS10, tls.VersionTLS11
	if _, _, err := tlsDial(t, base, "CS1", nil, old); err == nil {
		t.Error("TLS 1.1 must be rejected")
	}

	weak := ClientTLSConfig(ca.pool, nil)
	weak.MaxVersion = tls.VersionTLS12
	weak.CipherSuites = []uint16{tls.TLS_RSA_WITH_AES_128_CBC_SHA}
	if _, _, err := tlsDial(t, base, "CS1", nil, weak); err == nil {
		t.Error("a CBC suite must be rejected")
	}

	tls12 := ClientTLSConfig(ca.pool, nil)
	tls12.MaxVersion = tls.VersionTLS12
	if _, _, err := tlsDial(t, base, "CS1", nil, tls12); err != nil {
		t.Errorf("TLS 1.2 with an allowed suite must work: %v", err)
	}
}

func TestClientCertProfile3(t *testing.T) {
	ca := newTestCA(t, "CA")
	rec, base := startTLSServer(t, func(c *ServerConfig) {
		c.Authenticator = ClientCertAuthenticator(ClientCertOptions{Organization: "Test CSO"})
	}, ServerTLSConfig(ca.issue(t, "localhost", "CSMS", true), ca.pool))

	good := ca.issue(t, "CS1", "Test CSO", false)
	if _, _, err := tlsDial(t, base, "CS1", nil, ClientTLSConfig(ca.pool, &good)); err != nil {
		t.Fatalf("valid certificate: %v", err)
	}
	recv(t, rec.connected)

	// Certificate for another identity.
	if _, resp, err := tlsDial(t, base, "CS2", nil, ClientTLSConfig(ca.pool, &good)); err == nil || resp.StatusCode != 403 {
		t.Errorf("CN mismatch: %v %v", resp, err)
	}
	// Right CN, wrong operator organization.
	wrongOrg := ca.issue(t, "CS1", "Someone Else", false)
	if _, resp, err := tlsDial(t, base, "CS1", nil, ClientTLSConfig(ca.pool, &wrongOrg)); err == nil || resp.StatusCode != 403 {
		t.Errorf("organization mismatch: %v %v", resp, err)
	}
	// Certificate from an untrusted CA, and no certificate at all, fail in TLS.
	rogue := newTestCA(t, "rogue").issue(t, "CS1", "Test CSO", false)
	if _, _, err := tlsDial(t, base, "CS1", nil, ClientTLSConfig(ca.pool, &rogue)); err == nil {
		t.Error("certificate from an untrusted CA must be rejected")
	}
	if _, _, err := tlsDial(t, base, "CS1", nil, ClientTLSConfig(ca.pool, nil)); err == nil {
		t.Error("a missing client certificate must be rejected")
	}
}

func TestClientCertAuthenticatorDirect(t *testing.T) {
	ca := newTestCA(t, "CA")
	leaf := ca.issue(t, "CS1", "Test CSO", false)
	parsed, err := x509.ParseCertificate(leaf.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	withCert := func() *http.Request {
		r := httptest.NewRequest("GET", "/ocpp/CS1", nil)
		r.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{parsed}}
		return r
	}
	plain := httptest.NewRequest("GET", "/ocpp/CS1", nil)

	def := ClientCertAuthenticator(ClientCertOptions{})
	if err := def(plain, "CS1"); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("no TLS: %v", err)
	}
	empty := httptest.NewRequest("GET", "/", nil)
	empty.TLS = &tls.ConnectionState{}
	if err := def(empty, "CS1"); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("no peer certificate: %v", err)
	}
	if err := def(withCert(), "CS1"); err != nil {
		t.Errorf("valid: %v", err)
	}
	if err := def(withCert(), "CS2"); !errors.Is(err, ErrForbidden) {
		t.Errorf("identity mismatch: %v", err)
	}
	custom := ClientCertAuthenticator(ClientCertOptions{
		Identity: func(c *x509.Certificate, id string) error {
			if !strings.HasPrefix(id, c.Subject.CommonName) {
				return errors.New("no")
			}
			return nil
		},
	})
	if err := custom(withCert(), "CS1-extra"); err != nil {
		t.Errorf("custom identity check: %v", err)
	}
	if err := ClientCertAuthenticator(ClientCertOptions{Organization: "Other"})(withCert(), "CS1"); !errors.Is(err, ErrForbidden) {
		t.Errorf("organization: %v", err)
	}
}

func TestServeWithTLSConfig(t *testing.T) {
	ca := newTestCA(t, "CA")
	rec := newRecorder()
	cfg := ServerConfig{Versions: []ocpp.Version{ocpp.V201}, TLSConfig: ServerTLSConfig(ca.issue(t, "localhost", "CSMS", true), nil)}
	rec.apply(&cfg)
	s, err := NewServer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ln := mustListen(t)
	go func() { _ = s.Serve(ln) }()
	t.Cleanup(func() { _ = s.Shutdown(context.Background()) })
	if _, _, err := tlsDial(t, "wss://"+ln.Addr().String(), "CS1", nil, ClientTLSConfig(ca.pool, nil)); err != nil {
		t.Fatal(err)
	}
	recv(t, rec.connected)
}

func TestListenAndServe(t *testing.T) {
	s, _, _ := startServer(t, nil)
	if err := s.ListenAndServe("127.0.0.1:99999"); err == nil {
		t.Error("bad address should fail")
	}
	s2, err := NewServer(ServerConfig{Versions: []ocpp.Version{ocpp.V16}, OnFrame: func(*Conn, []byte) {}})
	if err != nil {
		t.Fatal(err)
	}
	errc := make(chan error, 1)
	go func() { errc <- s2.ListenAndServe("127.0.0.1:0") }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		s2.mu.Lock()
		up := s2.srv != nil
		s2.mu.Unlock()
		if up {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("server did not start")
		}
		time.Sleep(time.Millisecond)
	}
	if err := s2.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		t.Errorf("got %v", err)
	}
}

func TestNonWebSocketRequestGetsUpgradeRequired(t *testing.T) {
	_, _, base := startServer(t, nil)
	resp, err := http.Get("http" + strings.TrimPrefix(base, "ws") + "/ocpp/CS1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUpgradeRequired {
		t.Errorf("status %d", resp.StatusCode)
	}
}
