package csms

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
	"github.com/foecum/go-ocpp-sdk/ocppj"
	"github.com/foecum/go-ocpp-sdk/v16"
	"github.com/foecum/go-ocpp-sdk/ws"
)

// Seams for tests. These calls cannot fail with the fixed arguments used here,
// so tests replace them to exercise the error paths.
var (
	newRegistry = v16.NewRegistry
	newEndpoint = ocppj.NewEndpoint
	newWSServer = ws.NewServer
)

// Config configures a CSMS. Every field is optional.
type Config struct {
	// Core handles the Core profile calls charge points send. If nil, they are
	// answered with NotSupported.
	Core CoreHandler
	// Firmware handles the Firmware Management profile calls (diagnostics and
	// firmware status notifications). If nil, they are answered with
	// NotSupported.
	Firmware FirmwareHandler
	// Security handles the security extension calls (security events, signing
	// requests and status notifications). If nil, they are answered with
	// NotSupported.
	Security SecurityHandler
	// Authenticator checks every connection request. See ws.BasicAuthenticator
	// and ws.ClientCertAuthenticator. Without one anyone may connect: use only
	// on trusted networks.
	Authenticator ws.Authenticator
	// TLSConfig makes Serve and ListenAndServe speak TLS (wss); see
	// ws.ServerTLSConfig. Not used when the CSMS is mounted as an http.Handler.
	TLSConfig *tls.Config
	// CallTimeout is how long a call to a charge point waits for the answer.
	// Zero selects ocppj.DefaultTimeout (30 seconds).
	CallTimeout time.Duration
	// PingInterval, PongTimeout and ReadLimit tune the WebSocket connection;
	// see ws.ServerConfig.
	PingInterval time.Duration
	PongTimeout  time.Duration
	ReadLimit    int64
	// OnConnect is called when a charge point connects, before its first
	// message is handled.
	OnConnect func(cp *ChargePoint)
	// OnDisconnect is called when a connection ends, with the reason (nil for a
	// normal close).
	OnDisconnect func(cp *ChargePoint, err error)
	// Logger receives diagnostics. Default: discard.
	Logger *slog.Logger
}

// CSMS is an OCPP 1.6 Central System. It implements http.Handler and is safe
// for concurrent use.
type CSMS struct {
	cfg Config
	log *slog.Logger
	reg *ocppj.Registry
	ws  *ws.Server
	cps sync.Map // *ws.Conn -> *ChargePoint
}

// New creates a CSMS.
func New(cfg Config) (*CSMS, error) {
	reg, err := newRegistry()
	if err != nil {
		return nil, err
	}
	log := cfg.Logger
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	s := &CSMS{cfg: cfg, log: log, reg: reg}
	s.ws, err = newWSServer(ws.ServerConfig{
		Versions:      []ocpp.Version{ocpp.V16},
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

func (s *CSMS) onConnect(c *ws.Conn) {
	cp := &ChargePoint{id: c.ID(), conn: c}
	ep, err := newEndpoint(ocppj.Config{
		Registry: s.reg,
		Send:     c.Write,
		Timeout:  s.cfg.CallTimeout,
		Logger:   s.log.With("charge_point", c.ID()),
		Handler: func(ctx context.Context, req ocpp.Request) (ocpp.Response, error) {
			return s.dispatch(ctx, cp, req)
		},
	})
	if err != nil { // cannot happen: Registry and Send are set
		s.log.Error("cannot create endpoint", "charge_point", c.ID(), "err", err)
		_ = c.Close()
		return
	}
	cp.ep = ep
	s.cps.Store(c, cp)
	if s.cfg.OnConnect != nil {
		s.cfg.OnConnect(cp)
	}
}

func (s *CSMS) onFrame(c *ws.Conn, frame []byte) {
	if v, ok := s.cps.Load(c); ok {
		v.(*ChargePoint).ep.HandleFrame(frame)
	}
}

func (s *CSMS) onDisconnect(c *ws.Conn, err error) {
	v, ok := s.cps.LoadAndDelete(c)
	if !ok {
		return
	}
	cp := v.(*ChargePoint)
	cp.ep.Close() // fails calls still waiting for this charge point
	if s.cfg.OnDisconnect != nil {
		s.cfg.OnDisconnect(cp, err)
	}
}

// ServeHTTP performs the WebSocket handshake for a charge point. The identity
// is the last path segment of the URL.
func (s *CSMS) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.ws.ServeHTTP(w, r) }

// ListenAndServe listens on addr and serves until Shutdown. It returns
// http.ErrServerClosed after a graceful Shutdown.
func (s *CSMS) ListenAndServe(addr string) error { return s.ws.ListenAndServe(addr) }

// Serve accepts connections on l until Shutdown. It returns http.ErrServerClosed
// after a graceful Shutdown.
func (s *CSMS) Serve(l net.Listener) error { return s.ws.Serve(l) }

// Shutdown stops accepting connections and closes every charge point
// connection gracefully. If ctx ends first the rest are cut.
func (s *CSMS) Shutdown(ctx context.Context) error { return s.ws.Shutdown(ctx) }

// ChargePoint returns the connected charge point with the given identity.
func (s *CSMS) ChargePoint(id string) (*ChargePoint, bool) {
	c, ok := s.ws.Conn(id)
	if !ok {
		return nil, false
	}
	v, ok := s.cps.Load(c)
	if !ok {
		return nil, false
	}
	return v.(*ChargePoint), true
}

// ChargePoints returns the identities of all connected charge points, sorted.
func (s *CSMS) ChargePoints() []string { return s.ws.IDs() }
