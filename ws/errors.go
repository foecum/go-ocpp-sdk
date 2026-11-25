package ws

import "errors"

// Errors reported by the transport. Errors from the ocpp package
// (ocpp.ErrNotConnected, ocpp.ErrClosed) are used where they apply.
var (
	// ErrPingTimeout is the reason a connection was closed because the peer did
	// not answer a WebSocket ping in time.
	ErrPingTimeout = errors.New("ws: ping timeout")
	// ErrReplaced is the reason a connection was closed because the same charge
	// point connected again.
	ErrReplaced = errors.New("ws: replaced by a newer connection")
	// ErrNoSubprotocol means the peer did not agree to any OCPP subprotocol we
	// offered.
	ErrNoSubprotocol = errors.New("ws: no common OCPP subprotocol")
	// ErrBinaryFrame means the peer sent a binary frame; OCPP-J uses text only.
	ErrBinaryFrame = errors.New("ws: binary frames are not allowed")
	// ErrInvalidID means a charge point identity is empty, longer than 48
	// characters, or contains ':' or '/'.
	ErrInvalidID = errors.New("ws: invalid charge point identity")
)
