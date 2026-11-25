// Package cpcore is the version-independent part of the high-level Charge Point
// APIs (chargepoint/v16, chargepoint/v201): it connects to the Central System,
// reconnects, creates an ocppj endpoint for each connection, and dispatches
// incoming calls to a handler function.
package cpcore

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
	"github.com/foecum/go-ocpp-sdk/ws"
)

// newEndpoint is a seam for tests: it cannot fail with the arguments used here.
var newEndpoint = ocppj.NewEndpoint

// Config configures a Station.
type Config struct {
	// Version is the OCPP version to speak.
	Version ocpp.Version
	// Registry holds the messages of that version.
	Registry *ocppj.Registry
	// Dispatch answers an incoming call.
	Dispatch ocppj.RequestHandler

	URL          string
	ID           string
	Password     string
	TLSConfig    *tls.Config
	Reconnect    ws.Reconnect
	CallTimeout  time.Duration
	PingInterval time.Duration
	PongTimeout  time.Duration
	ReadLimit    int64
	// OnConnect and OnDisconnect are optional hooks.
	OnConnect    func()
	OnDisconnect func(err error)
	Logger       *slog.Logger
}

// Station is a charging station connected to a Central System. The public
// ChargePoint types of the high-level packages embed it. It is safe for
// concurrent use.
type Station struct {
	cfg    Config
	log    *slog.Logger
	client *ws.Client

	mu sync.Mutex
	ep *ocppj.Endpoint // current connection's endpoint, nil while disconnected
	up chan struct{}   // closed while connected; replaced on disconnect
}

// New validates cfg and creates a Station. It does not connect: call Run.
func New(cfg Config) (*Station, error) {
	log := cfg.Logger
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	s := &Station{cfg: cfg, log: log, up: make(chan struct{})}
	var err error
	s.client, err = ws.NewClient(ws.ClientConfig{
		URL:          cfg.URL,
		ID:           cfg.ID,
		Versions:     []ocpp.Version{cfg.Version},
		Password:     cfg.Password,
		TLSConfig:    cfg.TLSConfig,
		Reconnect:    cfg.Reconnect,
		OnConnect:    s.onConnect,
		OnFrame:      s.onFrame,
		OnDisconnect: s.onDisconnect,
		PingInterval: cfg.PingInterval,
		PongTimeout:  cfg.PongTimeout,
		ReadLimit:    cfg.ReadLimit,
		Logger:       log,
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Station) onConnect(conn *ws.Conn) {
	ep, err := newEndpoint(ocppj.Config{
		Registry: s.cfg.Registry,
		Send:     conn.Write,
		Timeout:  s.cfg.CallTimeout,
		Logger:   s.log,
		Handler:  s.cfg.Dispatch,
	})
	if err != nil { // cannot happen: Registry and Send are set
		s.log.Error("cannot create endpoint", "err", err)
		_ = conn.Close()
		return
	}
	s.mu.Lock()
	s.ep = ep
	close(s.up)
	s.mu.Unlock()
	if s.cfg.OnConnect != nil {
		s.cfg.OnConnect()
	}
}

func (s *Station) onFrame(_ *ws.Conn, frame []byte) {
	if ep := s.Endpoint(); ep != nil {
		ep.HandleFrame(frame)
	}
}

func (s *Station) onDisconnect(_ *ws.Conn, err error) {
	s.mu.Lock()
	ep := s.ep
	s.ep = nil
	s.up = make(chan struct{})
	s.mu.Unlock()
	if ep != nil {
		ep.Close() // fails calls still waiting for the Central System
	}
	if s.cfg.OnDisconnect != nil {
		s.cfg.OnDisconnect(err)
	}
}

// Endpoint returns the current connection's ocppj endpoint, or nil while
// disconnected.
func (s *Station) Endpoint() *ocppj.Endpoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ep
}

// Run connects to the Central System and keeps the connection up, reconnecting
// with back-off after any failure, until ctx ends or Close is called. It
// returns ocpp.ErrClosed after Close, or ctx's error. Call it once, usually in
// its own goroutine.
func (s *Station) Run(ctx context.Context) error { return s.client.Run(ctx) }

// Close disconnects gracefully, stops reconnecting and waits for Run to return.
func (s *Station) Close() error { return s.client.Close() }

// Connected reports whether the station is connected right now.
func (s *Station) Connected() bool { return s.Endpoint() != nil }

// WaitConnected blocks until the station is connected or ctx ends.
func (s *Station) WaitConnected(ctx context.Context) error {
	s.mu.Lock()
	up := s.up
	s.mu.Unlock()
	select {
	case <-up:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Call sends req to the Central System and returns the typed response. While
// disconnected it fails at once with an error matching ocpp.ErrNotConnected.
func Call[Resp any](ctx context.Context, s *Station, req ocpp.Request) (*Resp, error) {
	ep := s.Endpoint()
	if ep == nil {
		return nil, ocpp.ErrNotConnected
	}
	return rpc.Call[Resp](ctx, ep, req)
}
