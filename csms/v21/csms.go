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
	"github.com/foecum/go-ocpp-sdk/v21"
	"github.com/foecum/go-ocpp-sdk/ws"
)

// newRegistry is a seam for tests: it cannot fail, so tests replace it to
// exercise the error path.
var newRegistry = v21.NewRegistry

// Config configures a CSMS. Every field is optional.
//
// Each handler field serves one functional block of the specification. A block
// with no handler, or a method left to the embedded Unimplemented...Handler, is
// answered with NotSupported; an action the SDK does not know is answered with
// NotImplemented. The blocks Local Authorization List, Remote Control, Tariff
// and Cost and Bidirectional Power Transfer only contain calls the CSMS makes,
// so they need no handler.
type Config struct {
	// Security handles the calls of the Security functional block a charging station sends. If
	// nil, they are answered with NotSupported.
	Security SecurityHandler
	// Provisioning handles the calls of the Provisioning functional block a charging station sends. If
	// nil, they are answered with NotSupported.
	Provisioning ProvisioningHandler
	// Authorization handles the calls of the Authorization functional block a charging station sends. If
	// nil, they are answered with NotSupported.
	Authorization AuthorizationHandler
	// Transactions handles the calls of the Transactions functional block a charging station sends. If
	// nil, they are answered with NotSupported.
	Transactions TransactionsHandler
	// Availability handles the calls of the Availability functional block a charging station sends. If
	// nil, they are answered with NotSupported.
	Availability AvailabilityHandler
	// Reservation handles the calls of the Reservation functional block a charging station sends. If
	// nil, they are answered with NotSupported.
	Reservation ReservationHandler
	// MeterValues handles the calls of the MeterValues functional block a charging station sends. If
	// nil, they are answered with NotSupported.
	MeterValues MeterValuesHandler
	// SmartCharging handles the calls of the SmartCharging functional block a charging station sends. If
	// nil, they are answered with NotSupported.
	SmartCharging SmartChargingHandler
	// Firmware handles the calls of the Firmware functional block a charging station sends. If
	// nil, they are answered with NotSupported.
	Firmware FirmwareHandler
	// ISO15118 handles the calls of the ISO15118 functional block a charging station sends. If
	// nil, they are answered with NotSupported.
	ISO15118 ISO15118Handler
	// Diagnostics handles the calls of the Diagnostics functional block a charging station sends. If
	// nil, they are answered with NotSupported.
	Diagnostics DiagnosticsHandler
	// DisplayMessage handles the calls of the DisplayMessage functional block a charging station sends. If
	// nil, they are answered with NotSupported.
	DisplayMessage DisplayMessageHandler
	// DataTransfer handles the calls of the DataTransfer functional block a charging station sends. If
	// nil, they are answered with NotSupported.
	DataTransfer DataTransferHandler
	// DERControl handles the calls of the DER Control functional block a charging station
	// sends. If nil, they are answered with NotSupported.
	DERControl DERControlHandler
	// BatterySwap handles the calls of the Battery Swapping functional block a charging
	// station sends. If nil, they are answered with NotSupported.
	BatterySwap BatterySwapHandler
	// Authenticator checks every connection request. See ws.BasicAuthenticator
	// and ws.ClientCertAuthenticator. Without one anyone may connect: use only
	// on trusted networks.
	Authenticator ws.Authenticator
	// TLSConfig makes Serve and ListenAndServe speak TLS (wss); see
	// ws.ServerTLSConfig. Not used when the CSMS is mounted as an http.Handler.
	TLSConfig *tls.Config
	// CallTimeout is how long a call to a charging station waits for the answer.
	// Zero selects ocppj.DefaultTimeout (30 seconds).
	CallTimeout time.Duration
	// PingInterval, PongTimeout and ReadLimit tune the WebSocket connection;
	// see ws.ServerConfig.
	PingInterval time.Duration
	PongTimeout  time.Duration
	ReadLimit    int64
	// OnConnect is called when a charging station connects, before its first
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

// CSMS is an OCPP 2.1 Central System. It implements http.Handler and is safe
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
	s.srv, err = csmscore.New(csmscore.Config[ChargePoint]{
		Version:   ocpp.V21,
		Registry:  reg,
		NewHandle: func(sess *csmscore.Session) *ChargePoint { return &ChargePoint{session: sess} },
		Dispatch: func(ctx context.Context, cp *ChargePoint, req ocpp.Request) (ocpp.Response, error) {
			return s.dispatch(ctx, cp, req)
		},
		DispatchSend: func(ctx context.Context, cp *ChargePoint, msg ocpp.Request) error {
			return s.dispatchSend(ctx, cp, msg)
		},
		OnConnect:     cfg.OnConnect,
		OnDisconnect:  cfg.OnDisconnect,
		Authenticator: cfg.Authenticator,
		TLSConfig:     cfg.TLSConfig,
		CallTimeout:   cfg.CallTimeout,
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

// ServeHTTP performs the WebSocket handshake for a charging station. The
// identity is the last path segment of the URL.
func (s *CSMS) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.srv.ServeHTTP(w, r) }

// ListenAndServe listens on addr and serves until Shutdown. It returns
// http.ErrServerClosed after a graceful Shutdown.
func (s *CSMS) ListenAndServe(addr string) error { return s.srv.ListenAndServe(addr) }

// Serve accepts connections on l until Shutdown. It returns http.ErrServerClosed
// after a graceful Shutdown.
func (s *CSMS) Serve(l net.Listener) error { return s.srv.Serve(l) }

// Shutdown stops accepting connections and closes every connection gracefully.
// If ctx ends first the rest are cut.
func (s *CSMS) Shutdown(ctx context.Context) error { return s.srv.Shutdown(ctx) }

// ChargePoint returns the connected charging station with the given identity.
func (s *CSMS) ChargePoint(id string) (*ChargePoint, bool) { return s.srv.Lookup(id) }

// ChargePointIDs returns the identities of all connected charging stations, sorted.
func (s *CSMS) ChargePointIDs() []string { return s.srv.IDs() }

// ForEach calls fn for every connected charging station, in identity order, and stops when fn
// returns false. A charging station that disconnects meanwhile is skipped. fn may call the
// charging station, for example to broadcast a message, but should not block for long.
func (s *CSMS) ForEach(fn func(*ChargePoint) bool) { s.srv.ForEach(fn) }
