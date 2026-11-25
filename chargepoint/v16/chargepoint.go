package chargepoint

import (
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"sync"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/rpc"
	"github.com/foecum/go-ocpp-sdk/ocppj"
	"github.com/foecum/go-ocpp-sdk/v16"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/firmware"
	"github.com/foecum/go-ocpp-sdk/v16/security"
	"github.com/foecum/go-ocpp-sdk/ws"
)

// Seams for tests. These calls cannot fail with the fixed arguments used here,
// so tests replace them to exercise the error paths.
var (
	newRegistry = v16.NewRegistry
	newEndpoint = ocppj.NewEndpoint
)

// Config configures a ChargePoint.
type Config struct {
	// URL is the Central System's OCPP-J endpoint URL with a ws or wss scheme,
	// for example "wss://csms.example.com/ocpp". Required. The ID is appended.
	URL string
	// ID is the charge point identity. Required: at most 48 characters, without
	// ':' or '/'.
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
	// Core handles the Core profile calls the Central System sends. If nil,
	// they are answered with NotSupported.
	Core CoreHandler
	// Firmware, LocalAuth, Reservation, SmartCharging and RemoteTrigger handle
	// the calls of the other OCPP 1.6 profiles. Each is optional: calls of a
	// profile without a handler are answered with NotSupported.
	Firmware      FirmwareHandler
	LocalAuth     LocalAuthHandler
	Reservation   ReservationHandler
	SmartCharging SmartChargingHandler
	RemoteTrigger RemoteTriggerHandler
	// Security handles the security extension calls (certificates, logs, signed
	// firmware). If nil, they are answered with NotSupported.
	Security SecurityHandler
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

// ChargePoint is an OCPP 1.6 charging station connected to a Central System.
// It is safe for concurrent use.
type ChargePoint struct {
	cfg    Config
	log    *slog.Logger
	reg    *ocppj.Registry
	client *ws.Client

	mu sync.Mutex
	ep *ocppj.Endpoint // current connection's endpoint, nil while disconnected
	up chan struct{}   // closed while connected; replaced on disconnect
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
	c := &ChargePoint{cfg: cfg, log: log, reg: reg, up: make(chan struct{})}
	c.client, err = ws.NewClient(ws.ClientConfig{
		URL:          cfg.URL,
		ID:           cfg.ID,
		Versions:     []ocpp.Version{ocpp.V16},
		Password:     cfg.Password,
		TLSConfig:    cfg.TLSConfig,
		Reconnect:    cfg.Reconnect,
		OnConnect:    c.onConnect,
		OnFrame:      c.onFrame,
		OnDisconnect: c.onDisconnect,
		PingInterval: cfg.PingInterval,
		PongTimeout:  cfg.PongTimeout,
		ReadLimit:    cfg.ReadLimit,
		Logger:       log,
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *ChargePoint) onConnect(conn *ws.Conn) {
	ep, err := newEndpoint(ocppj.Config{
		Registry: c.reg,
		Send:     conn.Write,
		Timeout:  c.cfg.CallTimeout,
		Logger:   c.log,
		Handler:  c.dispatch,
	})
	if err != nil { // cannot happen: Registry and Send are set
		c.log.Error("cannot create endpoint", "err", err)
		_ = conn.Close()
		return
	}
	c.mu.Lock()
	c.ep = ep
	close(c.up)
	c.mu.Unlock()
	if c.cfg.OnConnect != nil {
		c.cfg.OnConnect()
	}
}

func (c *ChargePoint) onFrame(_ *ws.Conn, frame []byte) {
	if ep := c.endpoint(); ep != nil {
		ep.HandleFrame(frame)
	}
}

func (c *ChargePoint) onDisconnect(_ *ws.Conn, err error) {
	c.mu.Lock()
	ep := c.ep
	c.ep = nil
	c.up = make(chan struct{})
	c.mu.Unlock()
	if ep != nil {
		ep.Close() // fails calls still waiting for the Central System
	}
	if c.cfg.OnDisconnect != nil {
		c.cfg.OnDisconnect(err)
	}
}

func (c *ChargePoint) endpoint() *ocppj.Endpoint {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ep
}

// Run connects to the Central System and keeps the connection up, reconnecting
// with back-off after any failure, until ctx ends or Close is called. It
// returns ocpp.ErrClosed after Close, or ctx's error. Call it once, usually in
// its own goroutine.
func (c *ChargePoint) Run(ctx context.Context) error { return c.client.Run(ctx) }

// Close disconnects gracefully, stops reconnecting and waits for Run to return.
func (c *ChargePoint) Close() error { return c.client.Close() }

// Connected reports whether the charge point is connected right now.
func (c *ChargePoint) Connected() bool { return c.endpoint() != nil }

// WaitConnected blocks until the charge point is connected or ctx ends.
func (c *ChargePoint) WaitConnected(ctx context.Context) error {
	c.mu.Lock()
	up := c.up
	c.mu.Unlock()
	select {
	case <-up:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func call[Resp any](ctx context.Context, c *ChargePoint, req ocpp.Request) (*Resp, error) {
	ep := c.endpoint()
	if ep == nil {
		return nil, ocpp.ErrNotConnected
	}
	return rpc.Call[Resp](ctx, ep, req)
}

// Authorize asks the Central System whether an identifier may charge.
func (c *ChargePoint) Authorize(ctx context.Context, req core.AuthorizeRequest) (*core.AuthorizeConfirmation, error) {
	return call[core.AuthorizeConfirmation](ctx, c, req)
}

// BootNotification registers the charge point with the Central System.
func (c *ChargePoint) BootNotification(ctx context.Context, req core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
	return call[core.BootNotificationConfirmation](ctx, c, req)
}

// DataTransfer sends vendor-specific data to the Central System.
func (c *ChargePoint) DataTransfer(ctx context.Context, req core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	return call[core.DataTransferConfirmation](ctx, c, req)
}

// Heartbeat tells the Central System the charge point is alive and returns its
// time.
func (c *ChargePoint) Heartbeat(ctx context.Context, req core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
	return call[core.HeartbeatConfirmation](ctx, c, req)
}

// MeterValues reports meter readings.
func (c *ChargePoint) MeterValues(ctx context.Context, req core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	return call[core.MeterValuesConfirmation](ctx, c, req)
}

// StartTransaction reports that a transaction started.
func (c *ChargePoint) StartTransaction(ctx context.Context, req core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	return call[core.StartTransactionConfirmation](ctx, c, req)
}

// StatusNotification reports a connector status change.
func (c *ChargePoint) StatusNotification(ctx context.Context, req core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
	return call[core.StatusNotificationConfirmation](ctx, c, req)
}

// StopTransaction reports that a transaction ended.
func (c *ChargePoint) StopTransaction(ctx context.Context, req core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	return call[core.StopTransactionConfirmation](ctx, c, req)
}

// DiagnosticsStatusNotification reports the progress of a diagnostics upload.
func (c *ChargePoint) DiagnosticsStatusNotification(ctx context.Context, req firmware.DiagnosticsStatusNotificationRequest) (*firmware.DiagnosticsStatusNotificationConfirmation, error) {
	return call[firmware.DiagnosticsStatusNotificationConfirmation](ctx, c, req)
}

// FirmwareStatusNotification reports the progress of a firmware update.
func (c *ChargePoint) FirmwareStatusNotification(ctx context.Context, req firmware.FirmwareStatusNotificationRequest) (*firmware.FirmwareStatusNotificationConfirmation, error) {
	return call[firmware.FirmwareStatusNotificationConfirmation](ctx, c, req)
}

// LogStatusNotification reports the progress of a log upload.
func (c *ChargePoint) LogStatusNotification(ctx context.Context, req security.LogStatusNotificationRequest) (*security.LogStatusNotificationConfirmation, error) {
	return call[security.LogStatusNotificationConfirmation](ctx, c, req)
}

// SecurityEventNotification reports a security event; critical ones (see security.IsCriticalEvent) should be pushed to the Central System.
func (c *ChargePoint) SecurityEventNotification(ctx context.Context, req security.SecurityEventNotificationRequest) (*security.SecurityEventNotificationConfirmation, error) {
	return call[security.SecurityEventNotificationConfirmation](ctx, c, req)
}

// SignCertificate asks the Central System to sign the charge point certificate request (CSR).
func (c *ChargePoint) SignCertificate(ctx context.Context, req security.SignCertificateRequest) (*security.SignCertificateConfirmation, error) {
	return call[security.SignCertificateConfirmation](ctx, c, req)
}

// SignedFirmwareStatusNotification reports the progress of a signed firmware update.
func (c *ChargePoint) SignedFirmwareStatusNotification(ctx context.Context, req security.SignedFirmwareStatusNotificationRequest) (*security.SignedFirmwareStatusNotificationConfirmation, error) {
	return call[security.SignedFirmwareStatusNotificationConfirmation](ctx, c, req)
}
