package ws

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

// ServerConfig configures a Server.
type ServerConfig struct {
	// Versions lists the OCPP versions the server speaks, most preferred first.
	// Required. The connection uses the first of these that the client offers.
	Versions []ocpp.Version
	// Authenticator, if set, is called for every handshake. See
	// BasicAuthenticator and ClientCertAuthenticator. Without one, anyone may
	// connect: use only on trusted networks.
	Authenticator Authenticator
	// OnConnect is called after the handshake, before frames are delivered.
	OnConnect func(c *Conn)
	// OnFrame receives every frame, one at a time per connection, in order. It
	// is required and should not block for long: it holds up reading from that
	// connection. ocppj.Endpoint.HandleFrame is suitable.
	OnFrame func(c *Conn, frame []byte)
	// OnDisconnect is called when a connection ends, with the reason (nil for a
	// normal close).
	OnDisconnect func(c *Conn, err error)
	// TLSConfig, if set, makes Serve and ListenAndServe speak TLS (wss). See
	// ServerTLSConfig for security profiles 2 and 3. Not used by ServeHTTP.
	TLSConfig *tls.Config
	// PingInterval is how often to ping each charge point. Zero selects
	// DefaultPingInterval; a negative value disables pinging.
	PingInterval time.Duration
	// PongTimeout is how long to wait for a pong. Zero selects DefaultPongTimeout.
	PongTimeout time.Duration
	// ReadLimit is the largest accepted message. Zero selects DefaultReadLimit.
	ReadLimit int64
	// OriginPatterns lists origins allowed to connect from a browser. Charge
	// points send no Origin header, so this is normally empty.
	OriginPatterns []string
	// Logger receives diagnostics. Default: discard.
	Logger *slog.Logger
}

// Server accepts charge point connections. It implements http.Handler, so it
// can be mounted on any mux and served with any http.Server, or run on its own
// with Serve and ListenAndServe. It is safe for concurrent use.
type Server struct {
	cfg   ServerConfig
	names []string
	opts  connOptions

	baseCtx    context.Context
	baseCancel context.CancelFunc
	wg         sync.WaitGroup

	mu      sync.Mutex
	conns   map[string]*Conn
	closing bool
	srv     *http.Server
}

// NewServer creates a Server.
func NewServer(cfg ServerConfig) (*Server, error) {
	if cfg.OnFrame == nil {
		return nil, errors.New("ws: ServerConfig.OnFrame is required")
	}
	names, err := subprotocolNames(cfg.Versions)
	if err != nil {
		return nil, err
	}
	if cfg.Logger == nil {
		cfg.Logger = discardLogger()
	}
	s := &Server{cfg: cfg, names: names, conns: make(map[string]*Conn), opts: connOptions{
		pingInterval: cfg.PingInterval, pongTimeout: cfg.PongTimeout, readLimit: cfg.ReadLimit, log: cfg.Logger,
	}}
	s.opts = s.opts.withDefaults()
	s.baseCtx, s.baseCancel = context.WithCancel(context.Background())
	return s, nil
}

func (o connOptions) withDefaults() connOptions {
	if o.pingInterval == 0 {
		o.pingInterval = DefaultPingInterval
	}
	if o.pongTimeout <= 0 {
		o.pongTimeout = DefaultPongTimeout
	}
	if o.readLimit <= 0 {
		o.readLimit = DefaultReadLimit
	}
	return o
}

func subprotocolNames(versions []ocpp.Version) ([]string, error) {
	if len(versions) == 0 {
		return nil, errors.New("ws: at least one OCPP version is required")
	}
	names := make([]string, 0, len(versions))
	for _, v := range versions {
		if !v.IsValid() {
			return nil, errors.New("ws: unknown OCPP version " + string(v))
		}
		names = append(names, v.Subprotocol())
	}
	return names, nil
}

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// validateID checks a charge point identity (OCPP-J 2.0.1 §3.1.1 and A00.FR.204).
func validateID(id string) error {
	if id == "" || len(id) > MaxIDLength || strings.ContainsAny(id, ":/") {
		return ErrInvalidID
	}
	return nil
}

// idFromPath returns the identity in the last segment of an escaped URL path.
func idFromPath(escapedPath string) (string, error) {
	id, err := url.PathUnescape(escapedPath[strings.LastIndex(escapedPath, "/")+1:])
	if err != nil {
		return "", ErrInvalidID
	}
	return id, validateID(id)
}

// ServeHTTP performs the OCPP-J handshake and then serves the connection until
// it ends. The charge point identity is the last path segment of the URL.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.EscapedPath())
	if err != nil {
		http.Error(w, "invalid charge point identity", http.StatusBadRequest)
		return
	}
	if s.cfg.Authenticator != nil {
		if err := s.cfg.Authenticator(r, id); err != nil {
			rejectHandshake(w, err)
			return
		}
	}

	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		http.Error(w, "server shutting down", http.StatusServiceUnavailable)
		return
	}
	s.wg.Add(1)
	s.mu.Unlock()
	defer s.wg.Done()

	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols:   s.names,
		OriginPatterns: s.cfg.OriginPatterns,
	})
	if err != nil {
		s.cfg.Logger.Warn("ws: handshake failed", "id", id, "err", err)
		return
	}
	if ws.Subprotocol() == "" {
		// OCPP-J §3.2: complete the handshake without a subprotocol, then close.
		_ = ws.Close(websocket.StatusProtocolError, "no supported OCPP subprotocol")
		return
	}
	c := newConn(s.baseCtx, ws, id, ocpp.Version(ws.Subprotocol()), r.RemoteAddr)
	s.register(c)
	if s.cfg.OnConnect != nil {
		s.cfg.OnConnect(c)
	}
	err = c.run(s.opts, func(frame []byte) { s.cfg.OnFrame(c, frame) })
	s.unregister(c)
	if s.cfg.OnDisconnect != nil {
		s.cfg.OnDisconnect(c, err)
	}
}

func rejectHandshake(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrUnknownID):
		http.Error(w, "unknown charge point", http.StatusNotFound)
	case errors.Is(err, ErrForbidden):
		http.Error(w, "forbidden", http.StatusForbidden)
	default:
		w.Header().Set("WWW-Authenticate", `Basic realm="OCPP", charset="UTF-8"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}
}

// register records c, replacing (and closing) any earlier connection with the
// same identity: after a network drop a charge point usually reconnects before
// the server notices the old connection is dead.
func (s *Server) register(c *Conn) {
	s.mu.Lock()
	old := s.conns[c.id]
	s.conns[c.id] = c
	s.mu.Unlock()
	if old != nil {
		go func() {
			_ = old.closeWith(ErrReplaced, websocket.StatusPolicyViolation, "replaced by a newer connection")
		}()
	}
}

func (s *Server) unregister(c *Conn) {
	s.mu.Lock()
	if s.conns[c.id] == c {
		delete(s.conns, c.id)
	}
	s.mu.Unlock()
}

// Conn returns the live connection for a charge point identity.
func (s *Server) Conn(id string) (*Conn, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.conns[id]
	return c, ok
}

// IDs returns the identities of all connected charge points, sorted.
func (s *Server) IDs() []string {
	s.mu.Lock()
	ids := make([]string, 0, len(s.conns))
	for id := range s.conns {
		ids = append(ids, id)
	}
	s.mu.Unlock()
	sort.Strings(ids)
	return ids
}

// ListenAndServe listens on addr and serves until Shutdown. It returns
// http.ErrServerClosed after a graceful Shutdown.
func (s *Server) ListenAndServe(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.Serve(l)
}

// Serve accepts connections on l (wrapped in TLS if ServerConfig.TLSConfig is
// set) until Shutdown. It returns http.ErrServerClosed after a graceful Shutdown.
func (s *Server) Serve(l net.Listener) error {
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		_ = l.Close()
		return http.ErrServerClosed
	}
	srv := &http.Server{
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
		ErrorLog:          slog.NewLogLogger(s.cfg.Logger.Handler(), slog.LevelWarn),
	}
	s.srv = srv
	s.mu.Unlock()
	if s.cfg.TLSConfig != nil {
		l = tls.NewListener(l, s.cfg.TLSConfig)
	}
	return srv.Serve(l)
}

// Shutdown stops accepting connections, closes every connection with close
// code 1001 (going away) and waits for them to finish. If ctx ends first the
// remaining connections are cut and ctx's error is returned.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.closing = true
	srv := s.srv
	conns := make([]*Conn, 0, len(s.conns))
	for _, c := range s.conns {
		conns = append(conns, c)
	}
	s.mu.Unlock()

	var err error
	if srv != nil {
		err = srv.Shutdown(ctx)
	}
	for _, c := range conns {
		go func() { _ = c.closeWith(nil, websocket.StatusGoingAway, "server shutting down") }()
	}
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
		s.baseCancel()
		return err
	case <-ctx.Done():
		s.baseCancel()
		<-done
		return ctx.Err()
	}
}
