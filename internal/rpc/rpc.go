// Package rpc holds the helpers shared by the high-level chargepoint and csms
// packages: typed calls, typed handler results and panic safety.
package rpc

import (
	"errors"
	"fmt"
	"log/slog"

	"context"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/ocppj"
)

// Call sends req on ep and returns the typed confirmation. An error from a
// connection that closed during the call matches ocpp.ErrNotConnected.
func Call[Resp any](ctx context.Context, ep *ocppj.Endpoint, req ocpp.Request) (*Resp, error) {
	resp, err := ep.Call(ctx, req)
	if err != nil {
		if errors.Is(err, ocpp.ErrClosed) {
			return nil, fmt.Errorf("%w: connection closed during the call", ocpp.ErrNotConnected)
		}
		return nil, err
	}
	out, ok := any(resp).(*Resp)
	if !ok {
		return nil, fmt.Errorf("ocpp: unexpected response type %T for %s", resp, req.Action())
	}
	return out, nil
}

// Return converts a handler's typed result to the interface the endpoint
// expects. A nil confirmation with a nil error is reported as an error rather
// than becoming a non-nil interface holding a nil pointer.
func Return[C any, PC interface {
	*C
	ocpp.Response
}](c PC, err error) (ocpp.Response, error) {
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, errors.New("handler returned no confirmation")
	}
	return c, nil
}

// NotSupported is the error returned for an action the SDK knows but the
// application has no handler for.
func NotSupported(action string) error {
	return &ocpp.CallError{Code: ocpp.NotSupported, Description: action + " is not supported"}
}

// Safe runs fn and turns a panic into an InternalError, so a faulty handler
// cannot take the process down.
func Safe(log *slog.Logger, action string, fn func() (ocpp.Response, error)) (resp ocpp.Response, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("handler panicked", "action", action, "panic", fmt.Sprint(r))
			resp, err = nil, &ocpp.CallError{Code: ocpp.InternalError, Description: "internal error"}
		}
	}()
	return fn()
}
