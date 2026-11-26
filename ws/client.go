package ws

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/coder/websocket"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

// DefaultHandshakeTimeout bounds one connection attempt.
const DefaultHandshakeTimeout = 30 * time.Second

// ClientConfig configures a Client (a charging station).
type ClientConfig struct {
	// URL is the CSMS's OCPP-J endpoint URL, with a ws or wss scheme, for
	// example "wss://csms.example.com/ocpp". The charge point identity is
	// appended to its path.
	URL string
	// ID is the charge point identity: at most 48 characters, without ':' or '/'.
	ID string
	// Versions lists the OCPP versions to offer, most preferred first.
	Versions []ocpp.Version
	// Password, if not empty, enables HTTP Basic authentication with ID as the
	// username (security profiles 1 and 2).
	Password string
	// TLSConfig configures wss connections. See ClientTLSConfig for profiles 2
	// and 3.
	TLSConfig *tls.Config
	// Reconnect configures the back-off between attempts. The zero value
	// selects DefaultReconnect.
	Reconnect Reconnect
	// OnConnect is called after each successful connection.
	OnConnect func(c *Conn)
	// OnFrame receives every frame, in order. Required. It should not block for
	// long. ocppj.Endpoint.HandleFrame is suitable.
	OnFrame func(c *Conn, frame []byte)
	// OnDisconnect is called when a connection ends, with the reason (nil for a
	// normal close). The client then reconnects.
	OnDisconnect func(c *Conn, err error)
	// PingInterval is how often to ping the CSMS. Zero selects
	// DefaultPingInterval; a negative value disables pinging.
	PingInterval time.Duration
	// PongTimeout is how long to wait for a pong. Zero selects DefaultPongTimeout.
	PongTimeout time.Duration
	// ReadLimit is the largest accepted message. Zero selects DefaultReadLimit.
	ReadLimit int64
	// WriteTimeout bounds writing one frame; a connection whose write takes
	// longer is closed. Zero selects DefaultWriteTimeout.
	WriteTimeout time.Duration
	// HandshakeTimeout bounds one connection attempt. Zero selects
	// DefaultHandshakeTimeout.
	HandshakeTimeout time.Duration
	// Logger receives diagnostics. Default: discard.
	Logger *slog.Logger
}

// Client connects a charging station to a CSMS and keeps it connected,
// reconnecting with back-off after a failure. It is safe for concurrent use.
type Client struct {
	cfg    ClientConfig
	names  []string
	url    string
	header http.Header
	http   *http.Client
	opts   connOptions
	bo     *backoff

	mu      sync.Mutex
	conn    *Conn
	running bool
	closed  bool
	cancel  context.CancelFunc
	stopped chan struct{}
}

// NewClient validates cfg and creates a Client. It does not connect: call Run.
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.OnFrame == nil {
		return nil, errors.New("ws: ClientConfig.OnFrame is required")
	}
	if err := validateID(cfg.ID); err != nil {
		return nil, err
	}
	names, err := subprotocolNames(cfg.Versions)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("ws: invalid URL: %w", err)
	}
	if (u.Scheme != "ws" && u.Scheme != "wss") || u.Host == "" {
		return nil, errors.New("ws: URL must be ws://host/path or wss://host/path")
	}
	if cfg.Logger == nil {
		cfg.Logger = discardLogger()
	}
	if cfg.HandshakeTimeout <= 0 {
		cfg.HandshakeTimeout = DefaultHandshakeTimeout
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = cfg.TLSConfig
	c := &Client{
		cfg: cfg, names: names, url: u.JoinPath(cfg.ID).String(), header: http.Header{},
		http: &http.Client{Transport: tr}, bo: newBackoff(cfg.Reconnect),
		opts: connOptions{pingInterval: cfg.PingInterval, pongTimeout: cfg.PongTimeout, readLimit: cfg.ReadLimit, writeTimeout: cfg.WriteTimeout, log: cfg.Logger}.withDefaults(),
	}
	if cfg.Password != "" {
		c.header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(cfg.ID+":"+cfg.Password)))
	}
	return c, nil
}

// Run connects and keeps the connection up until ctx ends or Close is called,
// waiting between attempts as configured by Reconnect. It returns
// ocpp.ErrClosed after Close, or ctx's error. Run may be called once.
func (c *Client) Run(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ocpp.ErrClosed
	}
	if c.running {
		c.mu.Unlock()
		return errors.New("ws: client is already running")
	}
	ctx, c.cancel = context.WithCancel(ctx)
	c.running = true
	c.stopped = make(chan struct{})
	c.mu.Unlock()
	defer close(c.stopped)

	for {
		conn, err := c.dial(ctx)
		if err == nil {
			c.bo.reset()
			c.serve(ctx, conn)
		} else if ctx.Err() == nil {
			c.cfg.Logger.Warn("ws: connect failed", "id", c.cfg.ID, "err", err)
		}
		if ctx.Err() == nil {
			wait := c.bo.next()
			c.cfg.Logger.Debug("ws: reconnecting", "id", c.cfg.ID, "wait", wait)
			timer := time.NewTimer(wait)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
			}
		}
		if ctx.Err() != nil {
			c.mu.Lock()
			closed := c.closed
			c.mu.Unlock()
			if closed {
				return ocpp.ErrClosed
			}
			return ctx.Err()
		}
	}
}

// dial makes one connection attempt.
func (c *Client) dial(ctx context.Context) (*Conn, error) {
	hctx, cancel := context.WithTimeout(ctx, c.cfg.HandshakeTimeout)
	defer cancel()
	ws, resp, err := websocket.Dial(hctx, c.url, &websocket.DialOptions{
		HTTPClient: c.http, HTTPHeader: c.header, Subprotocols: c.names,
	})
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("handshake rejected (HTTP %d): %w", resp.StatusCode, err)
		}
		return nil, err
	}
	if ws.Subprotocol() == "" {
		_ = ws.Close(websocket.StatusProtocolError, "no common OCPP subprotocol")
		return nil, ErrNoSubprotocol
	}
	return newConn(context.Background(), ws, c.cfg.ID, ocpp.Version(ws.Subprotocol()), c.cfg.URL, c.opts.writeTimeout), nil
}

// serve runs one established connection to its end.
func (c *Client) serve(ctx context.Context, conn *Conn) {
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	if c.cfg.OnConnect != nil {
		c.cfg.OnConnect(conn)
	}
	// Close the connection gracefully when asked to stop.
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-conn.Done():
		}
	}()
	err := conn.run(c.opts, func(frame []byte) { c.cfg.OnFrame(conn, frame) })
	c.mu.Lock()
	c.conn = nil
	c.mu.Unlock()
	if c.cfg.OnDisconnect != nil {
		c.cfg.OnDisconnect(conn, err)
	}
}

// Conn returns the current connection, or nil while disconnected.
func (c *Client) Conn() *Conn {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn
}

// Send writes a frame on the current connection. While disconnected it returns
// an error matching ocpp.ErrNotConnected. It can be used as ocppj.Config.Send.
func (c *Client) Send(ctx context.Context, frame []byte) error {
	conn := c.Conn()
	if conn == nil {
		return ocpp.ErrNotConnected
	}
	return conn.Write(ctx, frame)
}

// Close stops the client: it closes the connection gracefully (close code
// 1000), stops reconnecting and waits for Run to return. It is idempotent.
func (c *Client) Close() error {
	c.mu.Lock()
	c.closed = true
	cancel, stopped := c.cancel, c.stopped
	c.mu.Unlock()
	if cancel != nil {
		cancel()
		<-stopped
	}
	return nil
}
