package chargepoint

import (
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/cpcore"
	"github.com/foecum/go-ocpp-sdk/v201"
	"github.com/foecum/go-ocpp-sdk/ws"
)

// newRegistry is a seam for tests: it cannot fail, so tests replace it to
// exercise the error path.
var newRegistry = v201.NewRegistry

// Config configures a ChargePoint.
//
// Each handler field serves one functional block of the specification. A block
// with no handler, or a method left to the embedded Unimplemented...Handler, is
// answered with NotSupported; an action the SDK does not know is answered with
// NotImplemented.
type Config struct {
	// URL is the CSMS's OCPP-J endpoint URL with a ws or wss scheme, for example
	// "wss://csms.example.com/ocpp". Required. The ID is appended.
	URL string
	// ID is the charging station identity. Required: at most 48 characters,
	// without ':' or '/'.
	ID string
	// Password, if not empty, enables HTTP Basic authentication with ID as the
	// username (security profiles 1 and 2).
	Password string
	// TLSConfig configures wss connections; see ws.ClientTLSConfig for profiles
	// 2 and 3.
	TLSConfig *tls.Config
	// Reconnect configures the back-off between connection attempts. The zero
	// value selects ws.DefaultReconnect.
	Reconnect ws.Reconnect
	// Security handles the calls of the Security functional block the CSMS sends. If nil,
	// they are answered with NotSupported.
	Security SecurityHandler
	// Provisioning handles the calls of the Provisioning functional block the CSMS sends. If nil,
	// they are answered with NotSupported.
	Provisioning ProvisioningHandler
	// Authorization handles the calls of the Authorization functional block the CSMS
	// sends. If nil, they are answered with NotSupported.
	Authorization AuthorizationHandler
	// LocalAuthList handles the calls of the LocalAuthList functional block the CSMS sends. If nil,
	// they are answered with NotSupported.
	LocalAuthList LocalAuthListHandler
	// Transactions handles the calls of the Transactions functional block the CSMS sends. If nil,
	// they are answered with NotSupported.
	Transactions TransactionsHandler
	// RemoteControl handles the calls of the RemoteControl functional block the CSMS sends. If nil,
	// they are answered with NotSupported.
	RemoteControl RemoteControlHandler
	// Availability handles the calls of the Availability functional block the CSMS sends. If nil,
	// they are answered with NotSupported.
	Availability AvailabilityHandler
	// Reservation handles the calls of the Reservation functional block the CSMS sends. If nil,
	// they are answered with NotSupported.
	Reservation ReservationHandler
	// TariffAndCost handles the calls of the TariffAndCost functional block the CSMS sends. If nil,
	// they are answered with NotSupported.
	TariffAndCost TariffAndCostHandler
	// SmartCharging handles the calls of the SmartCharging functional block the CSMS sends. If nil,
	// they are answered with NotSupported.
	SmartCharging SmartChargingHandler
	// Firmware handles the calls of the Firmware functional block the CSMS sends. If nil,
	// they are answered with NotSupported.
	Firmware FirmwareHandler
	// ISO15118 handles the calls of the ISO15118 functional block the CSMS sends. If nil,
	// they are answered with NotSupported.
	ISO15118 ISO15118Handler
	// Diagnostics handles the calls of the Diagnostics functional block the CSMS sends. If nil,
	// they are answered with NotSupported.
	Diagnostics DiagnosticsHandler
	// DisplayMessage handles the calls of the DisplayMessage functional block the CSMS sends. If nil,
	// they are answered with NotSupported.
	DisplayMessage DisplayMessageHandler
	// DataTransfer handles the calls of the DataTransfer functional block the CSMS sends. If nil,
	// they are answered with NotSupported.
	DataTransfer DataTransferHandler
	// CallTimeout is how long a call waits for the answer. Zero selects
	// ocppj.DefaultTimeout (30 seconds).
	CallTimeout time.Duration
	// PingInterval, PongTimeout and ReadLimit tune the WebSocket connection;
	// see ws.ClientConfig.
	PingInterval time.Duration
	PongTimeout  time.Duration
	ReadLimit    int64
	// OnConnect is called after each successful (re)connection.
	OnConnect func()
	// OnDisconnect is called when the connection ends, with the reason (nil for
	// a normal close). The ChargePoint then reconnects.
	OnDisconnect func(err error)
	// Logger receives diagnostics. Default: discard.
	Logger *slog.Logger
}

// ChargePoint is an OCPP 2.0.1 charging station connected to a CSMS. It is safe
// for concurrent use.
//
// Run connects and keeps the connection up (reconnecting with back-off) until
// its context ends or Close is called; Connected and WaitConnected report the
// state of the connection. The methods that call the CSMS, one per message the
// station sends, are in calls_gen.go.
type ChargePoint struct {
	*cpcore.Station
	cfg Config
	log *slog.Logger
}

// New validates cfg and creates a ChargePoint. It does not connect: call Run.
func New(cfg Config) (*ChargePoint, error) {
	reg, err := newRegistry()
	if err != nil {
		return nil, err
	}
	log := cfg.Logger
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	c := &ChargePoint{cfg: cfg, log: log}
	c.Station, err = cpcore.New(cpcore.Config{
		Version:      ocpp.V201,
		Registry:     reg,
		Dispatch:     c.dispatch,
		URL:          cfg.URL,
		ID:           cfg.ID,
		Password:     cfg.Password,
		TLSConfig:    cfg.TLSConfig,
		Reconnect:    cfg.Reconnect,
		CallTimeout:  cfg.CallTimeout,
		PingInterval: cfg.PingInterval,
		PongTimeout:  cfg.PongTimeout,
		ReadLimit:    cfg.ReadLimit,
		OnConnect:    cfg.OnConnect,
		OnDisconnect: cfg.OnDisconnect,
		Logger:       log,
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

func call[Resp any](ctx context.Context, c *ChargePoint, req ocpp.Request) (*Resp, error) {
	return cpcore.Call[Resp](ctx, c.Station, req)
}
