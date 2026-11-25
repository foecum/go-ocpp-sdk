package rpc_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/rpc"
	"github.com/foecum/go-ocpp-sdk/ocppj"
	"github.com/foecum/go-ocpp-sdk/v16"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v21"
	"github.com/foecum/go-ocpp-sdk/v21/diagnostics"
	"github.com/foecum/go-ocpp-sdk/v21/types"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

func endpoint(t *testing.T, send func(context.Context, []byte) error) *ocppj.Endpoint {
	t.Helper()
	reg, err := v16.NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	ep, err := ocppj.NewEndpoint(ocppj.Config{Registry: reg, Send: send})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ep.Close)
	return ep
}

func TestCall(t *testing.T) {
	var peer *ocppj.Endpoint
	station := endpoint(t, func(_ context.Context, f []byte) error { peer.HandleFrame(f); return nil })
	reg, _ := v16.NewRegistry()
	peer, _ = ocppj.NewEndpoint(ocppj.Config{Registry: reg,
		Send: func(_ context.Context, f []byte) error { station.HandleFrame(f); return nil },
		Handler: func(context.Context, ocpp.Request) (ocpp.Response, error) {
			return &core.ClearCacheConfirmation{Status: "Accepted"}, nil
		}})
	t.Cleanup(peer.Close)

	got, err := rpc.Call[core.ClearCacheConfirmation](context.Background(), station, core.ClearCacheRequest{})
	if err != nil || got.Status != "Accepted" {
		t.Fatalf("%v %v", got, err)
	}
	// Asking for the wrong confirmation type is reported, not panicked on.
	if _, err := rpc.Call[core.ResetConfirmation](context.Background(), station, core.ClearCacheRequest{}); err == nil {
		t.Error("expected a type mismatch error")
	}
	// Validation errors pass through.
	if _, err := rpc.Call[core.ResetConfirmation](context.Background(), station, core.ResetRequest{}); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("got %v", err)
	}
}

func TestCallMapsClosedToNotConnected(t *testing.T) {
	ep := endpoint(t, func(context.Context, []byte) error { return nil })
	ep.Close()
	_, err := rpc.Call[core.ClearCacheConfirmation](context.Background(), ep, core.ClearCacheRequest{})
	if !errors.Is(err, ocpp.ErrNotConnected) {
		t.Errorf("got %v", err)
	}
}

func TestReturn(t *testing.T) {
	conf := &core.ClearCacheConfirmation{Status: "Accepted"}
	if resp, err := rpc.Return(conf, nil); err != nil || resp != ocpp.Response(conf) {
		t.Errorf("%v %v", resp, err)
	}
	boom := errors.New("boom")
	if resp, err := rpc.Return[core.ClearCacheConfirmation](conf, boom); resp != nil || err != boom {
		t.Errorf("%v %v", resp, err)
	}
	// A nil pointer must not become a non-nil interface.
	resp, err := rpc.Return[core.ClearCacheConfirmation](nil, nil)
	if resp != nil || err == nil {
		t.Errorf("nil confirmation: %v %v", resp, err)
	}
}

func TestNotSupported(t *testing.T) {
	var ce *ocpp.CallError
	if err := rpc.NotSupported("Reset"); !errors.As(err, &ce) || ce.Code != ocpp.NotSupported || ce.Description == "" {
		t.Errorf("got %v", err)
	}
}

func TestSafe(t *testing.T) {
	ok := &core.ClearCacheConfirmation{}
	if resp, err := rpc.Safe(quiet, "X", func() (ocpp.Response, error) { return ok, nil }); err != nil || resp != ocpp.Response(ok) {
		t.Errorf("%v %v", resp, err)
	}
	resp, err := rpc.Safe(quiet, "X", func() (ocpp.Response, error) { panic("kaboom") })
	var ce *ocpp.CallError
	if resp != nil || !errors.As(err, &ce) || ce.Code != ocpp.InternalError {
		t.Errorf("panic: %v %v", resp, err)
	}
	if ce.Description == "" || ce.Description == "kaboom" {
		t.Errorf("the panic value must not leak to the peer: %q", ce.Description)
	}
}

func endpoint21(t *testing.T, send func(context.Context, []byte) error) *ocppj.Endpoint {
	t.Helper()
	reg, err := v21.NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	ep, err := ocppj.NewEndpoint(ocppj.Config{Registry: reg, Send: send})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ep.Close)
	return ep
}

func TestSend(t *testing.T) {
	var got []byte
	ep := endpoint21(t, func(_ context.Context, f []byte) error { got = f; return nil })
	msg := diagnostics.NotifyPeriodicEventStream{ID: 1, Basetime: ocpp.Now(), Data: []types.StreamDataElement{{V: "1"}}}
	if err := rpc.Send(context.Background(), ep, msg); err != nil || len(got) == 0 || got[1] != '6' {
		t.Fatalf("%s %v", got, err)
	}
	// Validation errors pass through.
	if err := rpc.Send(context.Background(), ep, diagnostics.NotifyPeriodicEventStream{}); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("got %v", err)
	}
	// A closed endpoint means there is no connection.
	ep.Close()
	if err := rpc.Send(context.Background(), ep, msg); !errors.Is(err, ocpp.ErrNotConnected) {
		t.Errorf("got %v", err)
	}
}

func TestSafeSend(t *testing.T) {
	boom := errors.New("boom")
	if err := rpc.SafeSend(quiet, "X", func() error { return boom }); err != boom {
		t.Errorf("got %v", err)
	}
	if err := rpc.SafeSend(quiet, "X", func() error { return nil }); err != nil {
		t.Errorf("got %v", err)
	}
	if err := rpc.SafeSend(quiet, "X", func() error { panic("oops") }); err == nil {
		t.Error("a panic must become an error")
	}
}
