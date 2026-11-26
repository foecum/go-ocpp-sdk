// Package csmscore is the version-independent part of the high-level Central
// System APIs (csms/v16, csms/v201): it accepts charge point connections,
// creates an ocppj endpoint for each, dispatches incoming calls to a handler
// function and keeps track of the connected charge points.
package csmscore

import (
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/rpc"
	"github.com/foecum/go-ocpp-sdk/ocppj"
	"github.com/foecum/go-ocpp-sdk/ws"
)

// Seams for tests. These calls cannot fail with the arguments used here, so
// tests replace them to exercise the error paths.
var (
	newEndpoint = ocppj.NewEndpoint
	newWSServer = ws.NewServer
)

// Session is one live connection to a charge point. The public ChargePoint types
// of the high-level packages embed it.
type Session struct {
	conn *ws.Conn
	ep   *ocppj.Endpoint
}

// ID returns the charge point identity from the connection URL.
func (s *Session) ID() string { return s.conn.ID() }

// Version returns the OCPP version of the connection.
func (s *Session) Version() ocpp.Version { return s.conn.Version() }

// RemoteAddr returns the charge point's network address.
func (s *Session) RemoteAddr() string { return s.conn.RemoteAddr() }

// Done is closed when the connection has ended.
func (s *Session) Done() <-chan struct{} { return s.conn.Done() }

// Close closes the connection gracefully. The charge point will normally
// reconnect.
func (s *Session) Close() error { return s.conn.Close() }

// Endpoint returns the connection's ocppj endpoint, for sending calls.
func (s *Session) Endpoint() *ocppj.Endpoint { return s.ep }

// Call sends req to the charge point and returns the typed response.
func Call[Resp any](ctx context.Context, s *Session, req ocpp.Request) (*Resp, error) {
	return rpc.Call[Resp](ctx, s.ep, req)
}

// Send sends msg to the charge point as an unconfirmed (SEND) message, which
// OCPP 2.1 has.
func Send(ctx context.Context, s *Session, msg ocpp.Request) error {
	return rpc.Send(ctx, s.ep, msg)
}

// Config configures a Server. H is the public handle type of a connected charge
// point (for example csms.ChargePoint).
type Config[H any] struct {
	// Version is the OCPP version to speak.
	Version ocpp.Version
	// Registry holds the messages of that version.
	Registry *ocppj.Registry
	// NewHandle builds the public handle for a new connection.
	NewHandle func(s *Session) *H
	// Dispatch answers an incoming call.
	Dispatch func(ctx context.Context, h *H, req ocpp.Request) (ocpp.Response, error)
	// DispatchSend handles an incoming unconfirmed (SEND) message, which OCPP 2.1
	// has. Optional: without it such messages are dropped.
	DispatchSend func(ctx context.Context, h *H, msg ocpp.Request) error
	// OnConnect and OnDisconnect are optional hooks.
	OnConnect    func(h *H)
	OnDisconnect func(h *H, err error)

	Authenticator ws.Authenticator
	TLSConfig     *tls.Config
	CallTimeout   time.Duration
	PingInterval  time.Duration
	PongTimeout   time.Duration
	ReadLimit     int64
	Logger        *slog.Logger
}

type entry[H any] struct {
	handle  *H
	session *Session
}

// Server accepts charge point connections. It is safe for concurrent use.
type Server[H any] struct {
	cfg     Config[H]
	log     *slog.Logger
	ws      *ws.Server
	entries sync.Map // *ws.Conn -> *entry[H]
}

// New creates a Server.
func New[H any](cfg Config[H]) (*Server[H], error) {
	log := cfg.Logger
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	s := &Server[H]{cfg: cfg, log: log}
	var err error
	s.ws, err = newWSServer(ws.ServerConfig{
		Versions:      []ocpp.Version{cfg.Version},
		Authenticator: cfg.Authenticator,
		TLSConfig:     cfg.TLSConfig,
		OnConnect:     s.onConnect,
		OnFrame:       s.onFrame,
		OnDisconnect:  s.onDisconnect,
		PingInterval:  cfg.PingInterval,
		PongTimeout:   cfg.PongTimeout,
		ReadLimit:     cfg.ReadLimit,
		Logger:        log,
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Server[H]) onConnect(c *ws.Conn) {
	session := &Session{conn: c}
	handle := s.cfg.NewHandle(session)
	cfg := ocppj.Config{
		Registry: s.cfg.Registry,
		Send:     c.Write,
		Timeout:  s.cfg.CallTimeout,
		Logger:   s.log.With("charge_point", c.ID()),
		Handler: func(ctx context.Context, req ocpp.Request) (ocpp.Response, error) {
			return s.cfg.Dispatch(ctx, handle, req)
		},
	}
	if s.cfg.DispatchSend != nil {
		cfg.SendHandler = func(ctx context.Context, msg ocpp.Request) error {
			return s.cfg.DispatchSend(ctx, handle, msg)
		}
	}
	ep, err := newEndpoint(cfg)
	if err != nil { // cannot happen: Registry and Send are set
		s.log.Error("cannot create endpoint", "charge_point", c.ID(), "err", err)
		_ = c.Close()
		return
	}
	session.ep = ep
	s.entries.Store(c, &entry[H]{handle: handle, session: session})
	if s.cfg.OnConnect != nil {
		s.cfg.OnConnect(handle)
	}
}

func (s *Server[H]) onFrame(c *ws.Conn, frame []byte) {
	if v, ok := s.entries.Load(c); ok {
		v.(*entry[H]).session.ep.HandleFrame(frame)
	}
}

func (s *Server[H]) onDisconnect(c *ws.Conn, err error) {
	v, ok := s.entries.LoadAndDelete(c)
	if !ok {
		return
	}
	e := v.(*entry[H])
	e.session.ep.Abort() // fails calls still waiting; handlers finish on their own
	if s.cfg.OnDisconnect != nil {
		s.cfg.OnDisconnect(e.handle, err)
	}
}

// ServeHTTP performs the WebSocket handshake for a charge point. The identity
// is the last path segment of the URL.
func (s *Server[H]) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.ws.ServeHTTP(w, r) }

// ListenAndServe listens on addr and serves until Shutdown.
func (s *Server[H]) ListenAndServe(addr string) error { return s.ws.ListenAndServe(addr) }

// Serve accepts connections on l until Shutdown.
func (s *Server[H]) Serve(l net.Listener) error { return s.ws.Serve(l) }

// Shutdown stops accepting connections and closes every connection gracefully.
func (s *Server[H]) Shutdown(ctx context.Context) error { return s.ws.Shutdown(ctx) }

// Lookup returns the handle of the connected charge point with the given identity.
func (s *Server[H]) Lookup(id string) (*H, bool) {
	c, ok := s.ws.Conn(id)
	if !ok {
		return nil, false
	}
	v, ok := s.entries.Load(c)
	if !ok {
		return nil, false
	}
	return v.(*entry[H]).handle, true
}

// IDs returns the identities of all connected charge points, sorted.
func (s *Server[H]) IDs() []string { return s.ws.IDs() }

// ForEach calls fn for every connected charge point, in identity order, until fn
// returns false. A charge point that disconnects meanwhile is skipped.
func (s *Server[H]) ForEach(fn func(*H) bool) {
	for _, id := range s.ws.IDs() {
		if h, ok := s.Lookup(id); ok && !fn(h) {
			return
		}
	}
}
