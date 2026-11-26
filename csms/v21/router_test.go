package csms_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
	chargepoint16 "github.com/foecum/go-ocpp-sdk/chargepoint/v16"
	chargepoint201 "github.com/foecum/go-ocpp-sdk/chargepoint/v201"
	"github.com/foecum/go-ocpp-sdk/chargepoint/v21"
	csms16 "github.com/foecum/go-ocpp-sdk/csms/v16"
	csms201 "github.com/foecum/go-ocpp-sdk/csms/v201"
	"github.com/foecum/go-ocpp-sdk/csms/v21"
	"github.com/foecum/go-ocpp-sdk/ws"
)

// One URL serves stations of all three versions, each through the CSMS of its
// version.
func TestOneEndpointForEveryVersion(t *testing.T) {
	connected := make(chan string, 3)
	s21, err := csms.New(csms.Config{OnConnect: func(cp *csms.ChargePoint) { connected <- "2.1:" + cp.ID() }})
	if err != nil {
		t.Fatal(err)
	}
	s201, err := csms201.New(csms201.Config{OnConnect: func(cp *csms201.ChargePoint) { connected <- "2.0.1:" + cp.ID() }})
	if err != nil {
		t.Fatal(err)
	}
	s16, err := csms16.New(csms16.Config{OnConnect: func(cp *csms16.ChargePoint) { connected <- "1.6:" + cp.ID() }})
	if err != nil {
		t.Fatal(err)
	}
	router, err := ws.NewVersionRouter(
		ws.VersionRoute{Version: ocpp.V21, Handler: s21},
		ws.VersionRoute{Version: ocpp.V201, Handler: s201},
		ws.VersionRoute{Version: ocpp.V16, Handler: s16},
	)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(router)
	defer ts.Close()
	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ocpp"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	a, _ := chargepoint.New(chargepoint.Config{URL: url, ID: "A"})
	b, _ := chargepoint201.New(chargepoint201.Config{URL: url, ID: "B"})
	c, _ := chargepoint16.New(chargepoint16.Config{URL: url, ID: "C"})
	for _, st := range []interface {
		Run(context.Context) error
		Close() error
	}{a, b, c} {
		go func() { _ = st.Run(ctx) }()
		defer st.Close()
	}
	got := map[string]bool{}
	for range 3 {
		select {
		case id := <-connected:
			got[id] = true
		case <-ctx.Done():
			t.Fatalf("connected so far: %v", got)
		}
	}
	if !got["2.1:A"] || !got["2.0.1:B"] || !got["1.6:C"] {
		t.Errorf("connected %v", got)
	}
	for _, sh := range []func(context.Context) error{s21.Shutdown, s201.Shutdown, s16.Shutdown} {
		_ = sh(ctx)
	}
}
