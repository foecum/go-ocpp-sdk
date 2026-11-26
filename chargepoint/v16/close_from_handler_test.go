package chargepoint_test

import (
	"context"
	"testing"
	"time"

	"github.com/foecum/go-ocpp-sdk/chargepoint/v16"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// A station may shut itself down from inside a handler, for example on Reset.
// Close used to wait for the handler that called it and never returned.
func TestCloseInsideAHandlerReturns(t *testing.T) {
	srv, _, url := startCSMS(t, nil)
	var cp *chargepoint.ChargePoint
	done := make(chan error, 1)
	st := okStation()
	st.reset = func(context.Context, *core.ResetRequest) (*core.ResetConfirmation, error) {
		done <- cp.Close()
		return &core.ResetConfirmation{Status: types.ResetStatusAccepted}, nil
	}
	cp = startStation(t, url, "CS-RESET", func(c *chargepoint.Config) { c.Core = st })
	remote := lookup(t, srv, "CS-RESET")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() { _, _ = remote.Reset(ctx, core.ResetRequest{Type: types.ResetTypeHard}) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Close called inside a handler did not return")
	}
	if cp.Connected() {
		t.Error("still connected after Close")
	}
}

// A handler that blocks does not keep the station from reconnecting.
func TestABlockedHandlerDoesNotPreventReconnecting(t *testing.T) {
	srv, ev, url := startCSMS(t, nil)
	release := make(chan struct{})
	defer close(release)
	started := make(chan struct{}, 1)
	st := okStation()
	st.reset = func(context.Context, *core.ResetRequest) (*core.ResetConfirmation, error) {
		started <- struct{}{}
		<-release // ignores its context
		return &core.ResetConfirmation{Status: types.ResetStatusAccepted}, nil
	}
	cp := startStation(t, url, "CS-BLOCK", func(c *chargepoint.Config) { c.Core = st })
	remote := lookup(t, srv, "CS-BLOCK")
	<-ev.connected
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() { _, _ = remote.Reset(ctx, core.ResetRequest{Type: types.ResetTypeSoft}) }()
	<-started
	_ = remote.Close() // the Central System drops the connection
	select {
	case <-ev.connected:
	case <-time.After(5 * time.Second):
		t.Fatal("the station did not reconnect while a handler was blocked")
	}
	if err := cp.WaitConnected(ctx); err != nil {
		t.Fatal(err)
	}
}
