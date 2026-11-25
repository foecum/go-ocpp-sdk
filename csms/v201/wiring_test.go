package csms

import (
	"context"
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/internal/gen"
	"github.com/foecum/go-ocpp-sdk/v201"
)

func allFeatures(t *testing.T) []ocpp.Feature {
	t.Helper()
	return v201.Features()
}

func handledByCSMS(action string) bool {
	d := gen.Directions[action]
	return d == gen.FromStation || d == gen.Both
}

// allUnimplemented is a Config with the Unimplemented handler of every block.
func allUnimplemented() Config {
	return Config{
		Security: UnimplementedSecurityHandler{}, Provisioning: UnimplementedProvisioningHandler{},
		Authorization: UnimplementedAuthorizationHandler{}, Transactions: UnimplementedTransactionsHandler{},
		Availability: UnimplementedAvailabilityHandler{}, Reservation: UnimplementedReservationHandler{},
		MeterValues: UnimplementedMeterValuesHandler{}, SmartCharging: UnimplementedSmartChargingHandler{},
		Firmware: UnimplementedFirmwareHandler{}, ISO15118: UnimplementedISO15118Handler{},
		Diagnostics: UnimplementedDiagnosticsHandler{}, DisplayMessage: UnimplementedDisplayMessageHandler{},
		DataTransfer: UnimplementedDataTransferHandler{},
	}
}

func isNotSupported(err error) bool {
	var ce *ocpp.CallError
	return errors.As(err, &ce) && ce.Code == ocpp.NotSupported
}

// Every message a charging station may send is routed by dispatch: with no
// handler, or with a handler that implements nothing, it is NotSupported; a
// message the CSMS only sends is NotSupported too.
func TestDispatchAnswersNotSupportedWithoutAnImplementation(t *testing.T) {
	for name, cfg := range map[string]Config{"no handlers": {}, "unimplemented handlers": allUnimplemented()} {
		s, err := New(cfg)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, f := range allFeatures(t) {
			req := reflect.New(f.RequestType()).Interface().(ocpp.Request)
			resp, err := s.dispatch(context.Background(), &ChargePoint{}, req)
			if resp != nil || !isNotSupported(err) {
				t.Errorf("%s: %s: got %v, %v", name, f.Action(), resp, err)
			}
			if handledByCSMS(f.Action()) {
				n++
			}
		}
		if n != 25 { // 24 sent by the station, plus DataTransfer
			t.Errorf("%d messages handled by the CSMS, want 25", n)
		}
	}
}

// Every handler interface covers exactly the messages the station sends.
func TestHandlerInterfacesMatchTheDirections(t *testing.T) {
	cfg := reflect.TypeOf(Config{})
	seen := map[string]bool{}
	for _, b := range gen.Blocks {
		var want []string
		for _, m := range b.Messages {
			if handledByCSMS(m) {
				want = append(want, m)
			}
		}
		field, ok := cfg.FieldByName(gen.HandlerNames[b.Pkg])
		if len(want) == 0 {
			if ok {
				t.Errorf("block %s has no messages for the CSMS to handle but Config has a field for it", b.Pkg)
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
			if meth.Type.NumIn() != 3 || meth.Type.NumOut() != 2 {
				t.Errorf("%s.%s has an unexpected signature %v", field.Type, m, meth.Type)
			}
		}
	}
	if len(seen) != 25 {
		t.Errorf("%d messages covered, want 25", len(seen))
	}
}

// Calling every typed method on a live connection reaches the endpoint: the
// zero requests are rejected by validation or answered by nobody, but the method
// is wired to its message.
func TestEveryCallMethodReachesTheEndpoint(t *testing.T) {
	connected := make(chan *ChargePoint, 1)
	s, err := New(Config{OnConnect: func(cp *ChargePoint) { connected <- cp }, CallTimeout: 30 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s)
	defer ts.Close()
	defer func() { _ = s.Shutdown(context.Background()) }()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(ts.URL, "http")+"/ocpp/CS1", &websocket.DialOptions{Subprotocols: []string{"ocpp2.0.1"}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	cp := <-connected

	// The station answers every call it receives with a CALLERROR carrying the action.
	go func() {
		for {
			_, data, err := c.Read(ctx)
			if err != nil {
				return
			}
			parts := strings.Split(string(data), `"`)
			_ = c.Write(ctx, websocket.MessageText, []byte(`[4,"`+parts[1]+`","NotSupported","`+parts[3]+`",{}]`))
		}
	}()

	called := 0
	cpv := reflect.ValueOf(cp)
	for _, f := range allFeatures(t) {
		if handledByCSMS(f.Action()) && gen.Directions[f.Action()] != gen.Both {
			continue
		}
		m := cpv.MethodByName(f.Action())
		if !m.IsValid() {
			t.Errorf("ChargePoint has no method %s", f.Action())
			continue
		}
		out := m.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.New(f.RequestType()).Elem()})
		err, _ := out[1].Interface().(error)
		if err == nil {
			t.Errorf("%s: a zero request cannot succeed", f.Action())
		}
		if !errors.Is(err, ocpp.ErrValidation) && !isNotSupported(err) {
			t.Errorf("%s: unexpected error %v", f.Action(), err)
		}
		called++
	}
	if called != 40 { // 39 sent by the CSMS, plus DataTransfer
		t.Errorf("%d call methods, want 40", called)
	}
}
