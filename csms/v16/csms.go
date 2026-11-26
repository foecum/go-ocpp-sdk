package csms

import (
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/csmscore"
	"github.com/foecum/go-ocpp-sdk/v16"
	"github.com/foecum/go-ocpp-sdk/ws"
)

// newRegistry is a seam for tests: it cannot fail, so tests replace it to
// exercise the error path.
var newRegistry = v16.NewRegistry

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
	//
	// When a station reconnects before its old connection has ended, the new
	// connection replaces the old one: OnConnect for the new connection can come
	// before OnDisconnect for the old one, whose err then matches ws.ErrReplaced.
	// If you track stations by identity, ignore that disconnect (or compare cp
	// with the one CSMS.ChargePoint returns), or a connected station will look
	// offline.
	OnDisconnect func(cp *ChargePoint, err error)
	// Logger receives diagnostics. Default: discard.
	Logger *slog.Logger
}

// CSMS is an OCPP 1.6 Central System. It implements http.Handler and is safe
// for concurrent use.
type CSMS struct {
	cfg Config
	log *slog.Logger
	srv *csmscore.Server[ChargePoint]
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
	s := &CSMS{cfg: cfg, log: log}
	core := csmscore.Config[ChargePoint]{
		Version:   ocpp.V16,
		Registry:  reg,
		NewHandle: func(sess *csmscore.Session) *ChargePoint { return &ChargePoint{session: sess} },
		Dispatch: func(ctx context.Context, cp *ChargePoint, req ocpp.Request) (ocpp.Response, error) {
			return s.dispatch(ctx, cp, req)
		},
		Authenticator: cfg.Authenticator,
		TLSConfig:     cfg.TLSConfig,
		CallTimeout:   cfg.CallTimeout,
		PingInterval:  cfg.PingInterval,
		PongTimeout:   cfg.PongTimeout,
		ReadLimit:     cfg.ReadLimit,
		OnConnect:     cfg.OnConnect,
		Logger:        log,
	}
	if cfg.OnDisconnect != nil {
		core.OnDisconnect = cfg.OnDisconnect
	}
	s.srv, err = csmscore.New(core)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// ServeHTTP performs the WebSocket handshake for a charge point. The identity
// is the last path segment of the URL.
func (s *CSMS) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.srv.ServeHTTP(w, r) }

// ListenAndServe listens on addr and serves until Shutdown. It returns
// http.ErrServerClosed after a graceful Shutdown.
func (s *CSMS) ListenAndServe(addr string) error { return s.srv.ListenAndServe(addr) }

// Serve accepts connections on l until Shutdown. It returns http.ErrServerClosed
// after a graceful Shutdown.
func (s *CSMS) Serve(l net.Listener) error { return s.srv.Serve(l) }

// Shutdown stops accepting connections and closes every charge point
// connection gracefully. If ctx ends first the rest are cut.
func (s *CSMS) Shutdown(ctx context.Context) error { return s.srv.Shutdown(ctx) }

// ChargePoint returns the connected charge point with the given identity.
func (s *CSMS) ChargePoint(id string) (*ChargePoint, bool) { return s.srv.Lookup(id) }

// ChargePointIDs returns the identities of all connected charge points, sorted.
func (s *CSMS) ChargePointIDs() []string { return s.srv.IDs() }

// ForEach calls fn for every connected charge point, in identity order, and stops when fn
// returns false. A charge point that disconnects meanwhile is skipped. fn may call the
// charge point, for example to broadcast a message, but should not block for long.
func (s *CSMS) ForEach(fn func(*ChargePoint) bool) { s.srv.ForEach(fn) }
