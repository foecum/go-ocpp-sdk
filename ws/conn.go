package ws

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

// Defaults for server and client configuration.
const (
	// DefaultPingInterval is how often a WebSocket ping is sent.
	DefaultPingInterval = 30 * time.Second
	// DefaultPongTimeout is how long to wait for the pong.
	DefaultPongTimeout = 10 * time.Second
	// DefaultReadLimit is the largest accepted message, in bytes.
	DefaultReadLimit = 1 << 20
	// DefaultWriteTimeout bounds how long writing one frame may take before the
	// connection is considered stuck and closed.
	DefaultWriteTimeout = 10 * time.Second
	// MaxIDLength is the longest allowed charge point identity (OCPP-J 2.0.1 §3.1.1).
	MaxIDLength = 48
)

// Conn is one established OCPP-J WebSocket connection, on either side. It is
// safe for concurrent use.
type Conn struct {
	id      string
	version ocpp.Version
	remote  string
	ws      *websocket.Conn

	writeTimeout time.Duration

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	mu            sync.Mutex
	err           error
	finished      bool
	closedLocally bool
}

func newConn(parent context.Context, ws *websocket.Conn, id string, v ocpp.Version, remote string, writeTimeout time.Duration) *Conn {
	ctx, cancel := context.WithCancel(parent)
	return &Conn{id: id, version: v, remote: remote, ws: ws, writeTimeout: writeTimeout, ctx: ctx, cancel: cancel, done: make(chan struct{})}
}

// ID returns the charge point identity.
func (c *Conn) ID() string { return c.id }

// Version returns the negotiated OCPP version.
func (c *Conn) Version() ocpp.Version { return c.version }

// RemoteAddr returns the peer's address, if known.
func (c *Conn) RemoteAddr() string { return c.remote }

// Done is closed when the connection has ended.
func (c *Conn) Done() <-chan struct{} { return c.done }

// Err returns why the connection ended, or nil for a normal close. It is
// meaningful once Done is closed.
func (c *Conn) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// Write sends one text frame. If the connection is closed it returns an error
// matching ocpp.ErrNotConnected, so Write can be used directly as
// ocppj.Config.Send.
//
// ctx is checked before writing: if it has ended, nothing is sent and its error
// is returned. A write that has started is not cut short by ctx, because the
// WebSocket would have to be closed for every other user of the connection;
// instead it is bounded by the write timeout (DefaultWriteTimeout unless
// configured), after which the connection is considered stuck and closed.
func (c *Conn) Write(ctx context.Context, frame []byte) error {
	select {
	case <-c.done:
		return ocpp.ErrNotConnected
	default:
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	wctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), c.writeTimeout)
	defer cancel()
	if err := c.ws.Write(wctx, websocket.MessageText, frame); err != nil {
		select {
		case <-c.done:
			return fmt.Errorf("%w: %v", ocpp.ErrNotConnected, err)
		default:
		}
		return fmt.Errorf("ws: write: %w", err)
	}
	return nil
}

// Close closes the connection gracefully (close code 1000) and waits for the
// peer to acknowledge, for up to a few seconds.
func (c *Conn) Close() error {
	return c.closeWith(nil, websocket.StatusNormalClosure, "")
}

// closeWith records cause (if it is the first) and performs the close
// handshake with the given code.
func (c *Conn) closeWith(cause error, code websocket.StatusCode, reason string) error {
	c.setErr(cause)
	c.mu.Lock()
	c.closedLocally = true
	c.mu.Unlock()
	return c.ws.Close(code, reason)
}

func (c *Conn) setErr(err error) {
	if err == nil {
		return
	}
	c.mu.Lock()
	if c.err == nil {
		c.err = err
	}
	c.mu.Unlock()
}

type connOptions struct {
	writeTimeout time.Duration
	pingInterval time.Duration
	pongTimeout  time.Duration
	readLimit    int64
	log          *slog.Logger
}

// run reads frames and passes each to onFrame, in order, until the connection
// ends. It sends WebSocket pings, closes the connection if the peer stops
// answering, and returns Err.
func (c *Conn) run(o connOptions, onFrame func([]byte)) error {
	c.ws.SetReadLimit(o.readLimit)
	if o.pingInterval > 0 {
		go c.pingLoop(o)
	}
	var cause error
	for {
		typ, data, err := c.ws.Read(c.ctx)
		if err != nil {
			cause = err
			break
		}
		if typ != websocket.MessageText {
			_ = c.closeWith(ErrBinaryFrame, websocket.StatusUnsupportedData, "OCPP-J frames must be text")
			break
		}
		onFrame(data)
	}
	c.finish(cause)
	return c.Err()
}

// finish tears the connection down and records the reason.
func (c *Conn) finish(cause error) {
	c.mu.Lock()
	local := c.closedLocally
	c.mu.Unlock()
	// A close we initiated, or one the peer completed normally, is not an error.
	if status := websocket.CloseStatus(cause); local || status == websocket.StatusNormalClosure ||
		status == websocket.StatusGoingAway {
		cause = nil
	}
	c.setErr(cause)
	c.mu.Lock()
	already := c.finished
	c.finished = true
	c.mu.Unlock()
	if already {
		return
	}
	c.cancel()
	_ = c.ws.CloseNow()
	close(c.done)
}

func (c *Conn) pingLoop(o connOptions) {
	t := time.NewTicker(o.pingInterval)
	defer t.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-t.C:
		}
		ctx, cancel := context.WithTimeout(c.ctx, o.pongTimeout)
		err := c.ws.Ping(ctx)
		cancel()
		if err != nil {
			if c.ctx.Err() == nil {
				o.log.Warn("ws: ping failed, closing connection", "id", c.id, "err", err)
				c.setErr(ErrPingTimeout)
				c.cancel() // unblocks Read, which tears the connection down
			}
			return
		}
	}
}
