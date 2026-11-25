package chargepoint

import (
	"context"
	"errors"
	"reflect"
	"testing"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/gen"
	"github.com/foecum/go-ocpp-sdk/ocppj"
	"github.com/foecum/go-ocpp-sdk/v201"
)

func handledByStation(action string) bool {
	d := gen.Directions[action]
	return d == gen.FromCSMS || d == gen.Both
}

func allUnimplemented() Config {
	return Config{
		Security: UnimplementedSecurityHandler{}, Provisioning: UnimplementedProvisioningHandler{},
		Authorization: UnimplementedAuthorizationHandler{}, LocalAuthList: UnimplementedLocalAuthListHandler{},
		Transactions: UnimplementedTransactionsHandler{}, RemoteControl: UnimplementedRemoteControlHandler{},
		Availability: UnimplementedAvailabilityHandler{}, Reservation: UnimplementedReservationHandler{},
		TariffAndCost: UnimplementedTariffAndCostHandler{}, SmartCharging: UnimplementedSmartChargingHandler{},
		Firmware: UnimplementedFirmwareHandler{}, ISO15118: UnimplementedISO15118Handler{},
		Diagnostics: UnimplementedDiagnosticsHandler{}, DisplayMessage: UnimplementedDisplayMessageHandler{},
		DataTransfer: UnimplementedDataTransferHandler{},
	}
}

func isNotSupported(err error) bool {
	var ce *ocpp.CallError
	return errors.As(err, &ce) && ce.Code == ocpp.NotSupported
}

func newStation(t *testing.T, cfg Config) *ChargePoint {
	t.Helper()
	cfg.URL, cfg.ID = "ws://localhost:1/ocpp", "CS1"
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestDispatchAnswersNotSupportedWithoutAnImplementation(t *testing.T) {
	for name, cfg := range map[string]Config{"no handlers": {}, "unimplemented handlers": allUnimplemented()} {
		c := newStation(t, cfg)
		n := 0
		for _, f := range v201.Features() {
			req := reflect.New(f.RequestType()).Interface().(ocpp.Request)
			resp, err := c.dispatch(context.Background(), req)
			if resp != nil || !isNotSupported(err) {
				t.Errorf("%s: %s: got %v, %v", name, f.Action(), resp, err)
			}
			if handledByStation(f.Action()) {
				n++
			}
		}
		if n != 40 { // 39 sent by the CSMS, plus DataTransfer
			t.Errorf("%d messages handled by the station, want 40", n)
		}
	}
}

func TestHandlerInterfacesMatchTheDirections(t *testing.T) {
	cfg := reflect.TypeOf(Config{})
	seen := map[string]bool{}
	for _, b := range gen.Blocks {
		var want []string
		for _, m := range b.Messages {
			if handledByStation(m) {
				want = append(want, m)
			}
		}
		field, ok := cfg.FieldByName(gen.HandlerNames[b.Pkg])
		if len(want) == 0 {
			if ok {
				t.Errorf("block %s has no messages for the station to handle but Config has a field for it", b.Pkg)
			}
			continue
		}
		if !ok {
			t.Errorf("Config has no field for block %s", b.Pkg)
			continue
		}
		if field.Type.NumMethod() != len(want) {
			t.Errorf("%s has %d methods, want %d", field.Type, field.Type.NumMethod(), len(want))
		}
		for _, m := range want {
			meth, ok := field.Type.MethodByName(m)
			if !ok {
				t.Errorf("%s lacks %s", field.Type, m)
				continue
			}
			seen[m] = true
			if meth.Type.NumIn() != 2 || meth.Type.NumOut() != 2 {
				t.Errorf("%s.%s has an unexpected signature %v", field.Type, m, meth.Type)
			}
		}
	}
	if len(seen) != 40 {
		t.Errorf("%d messages covered, want 40", len(seen))
	}
}

// Every typed call method exists and, while disconnected, fails with ErrNotConnected.
func TestEveryCallMethodFailsWhileDisconnected(t *testing.T) {
	c := newStation(t, Config{})
	called := 0
	for _, f := range v201.Features() {
		d := gen.Directions[f.Action()]
		if d != gen.FromStation && d != gen.Both {
			continue
		}
		m := reflect.ValueOf(c).MethodByName(f.Action())
		if !m.IsValid() {
			t.Errorf("ChargePoint has no method %s", f.Action())
			continue
		}
		out := m.Call([]reflect.Value{reflect.ValueOf(context.Background()), reflect.New(f.RequestType()).Elem()})
		if err, _ := out[1].Interface().(error); !errors.Is(err, ocpp.ErrNotConnected) {
			t.Errorf("%s: %v", f.Action(), err)
		}
		called++
	}
	if called != 25 { // 24 sent by the station, plus DataTransfer
		t.Errorf("%d call methods, want 25", called)
	}
}

func TestNewReportsInternalFailure(t *testing.T) {
	boom := errors.New("boom")
	old := newRegistry
	newRegistry = func() (*ocppj.Registry, error) { return nil, boom }
	defer func() { newRegistry = old }()
	if _, err := New(Config{URL: "ws://localhost/ocpp", ID: "CS1"}); !errors.Is(err, boom) {
		t.Errorf("got %v", err)
	}
}

func TestNewRejectsAnInvalidConfig(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Error("an empty config must be rejected")
	}
}
