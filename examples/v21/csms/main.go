// Command csms is an example OCPP 2.1 Central System.
//
// It accepts charging stations over WebSocket, logs everything they send,
// authorizes idTokens, and after each boot reads and changes the station's
// heartbeat interval with GetVariables and SetVariables and sets a default
// tariff (new in OCPP 2.1). It also receives the periodic event streams that
// stations open and the unconfirmed NotifyPeriodicEventStream messages that
// carry their values. With -remote-start it
// also asks the station to start a charging session and stops it a second later.
//
//	go run ./examples/v21/csms -addr :8080
//
// Then, in another terminal, start the example charging station:
//
//	go run ./examples/v21/chargingstation -url ws://localhost:8080/ocpp -id CS001
//
// Stations connect to ws://host:8080/ocpp/<identity>. Use -password to require
// HTTP Basic authentication (the username is the identity) and -tags to accept
// only some idTokens.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/csms/v21"
	"github.com/foecum/go-ocpp-sdk/v21/authorization"
	"github.com/foecum/go-ocpp-sdk/v21/availability"
	"github.com/foecum/go-ocpp-sdk/v21/diagnostics"
	"github.com/foecum/go-ocpp-sdk/v21/metervalues"
	"github.com/foecum/go-ocpp-sdk/v21/provisioning"
	"github.com/foecum/go-ocpp-sdk/v21/remotecontrol"
	"github.com/foecum/go-ocpp-sdk/v21/tariffandcost"
	"github.com/foecum/go-ocpp-sdk/v21/transactions"
	"github.com/foecum/go-ocpp-sdk/v21/types"
	"github.com/foecum/go-ocpp-sdk/ws"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	password := flag.String("password", "", "if set, stations must authenticate with this password (HTTP Basic, username = identity)")
	tags := flag.String("tags", "", "comma-separated idTokens to accept; empty accepts every idToken")
	interval := flag.Int("interval", 30, "heartbeat interval in seconds sent to stations")
	remoteStart := flag.Bool("remote-start", false, "ask each station to start a session after it boots")
	flag.Parse()

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	opts := options{password: *password, interval: *interval, remoteStart: *remoteStart}
	if *tags != "" {
		opts.tags = strings.Split(*tags, ",")
	}
	if err := run(ctx, ln, opts, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

type options struct {
	password    string
	tags        []string
	interval    int
	remoteStart bool
}

// run serves until ctx ends.
func run(ctx context.Context, ln net.Listener, opts options, out io.Writer) error {
	logger := slog.New(slog.NewTextHandler(out, nil))
	h := &handler{log: logger, interval: opts.interval, remoteStart: opts.remoteStart}
	if len(opts.tags) > 0 {
		h.tags = make(map[string]bool)
		for _, t := range opts.tags {
			h.tags[strings.ToLower(strings.TrimSpace(t))] = true
		}
	}
	cfg := csms.Config{
		Provisioning:  h,
		Authorization: h,
		Availability:  h,
		Transactions:  h,
		MeterValues:   h,
		Diagnostics:   h,
		Logger:        logger,
		OnConnect: func(cp *csms.ChargePoint) {
			logger.Info("charging station connected", "id", cp.ID(), "addr", cp.RemoteAddr())
		},
		OnDisconnect: func(cp *csms.ChargePoint, err error) {
			logger.Info("charging station disconnected", "id", cp.ID(), "reason", err)
		},
	}
	if opts.password != "" {
		cfg.Authenticator = ws.BasicAuthenticator(func(string) (string, bool) { return opts.password, true })
	}
	srv, err := csms.New(cfg)
	if err != nil {
		return err
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	fmt.Fprintf(out, "listening on %s\n", ln.Addr())

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdown); err != nil {
		return err
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// handler answers the calls stations send. Each functional block has its own
// handler interface; the Unimplemented... structs answer NotSupported for the
// calls this example does not handle.
type handler struct {
	csms.UnimplementedProvisioningHandler
	csms.UnimplementedAuthorizationHandler
	csms.UnimplementedAvailabilityHandler
	csms.UnimplementedTransactionsHandler
	csms.UnimplementedMeterValuesHandler
	csms.UnimplementedDiagnosticsHandler

	log         *slog.Logger
	tags        map[string]bool // nil accepts every idToken
	interval    int
	remoteStart bool
}

func (h *handler) status(token string) types.AuthorizationStatus {
	if h.tags == nil || h.tags[strings.ToLower(token)] {
		return types.AuthorizationStatusAccepted
	}
	return types.AuthorizationStatusInvalid
}

func (h *handler) BootNotification(ctx context.Context, cp *csms.ChargePoint, req *provisioning.BootNotificationRequest) (*provisioning.BootNotificationResponse, error) {
	h.log.Info("boot notification", "id", cp.ID(), "reason", req.Reason,
		"vendor", req.ChargingStation.VendorName, "model", req.ChargingStation.Model, "firmware", req.ChargingStation.FirmwareVersion)
	go h.inspect(cp) // call the station back, but not while handling its request
	return &provisioning.BootNotificationResponse{
		Status:      types.RegistrationStatusAccepted,
		CurrentTime: ocpp.Now(),
		Interval:    h.interval,
	}, nil
}

var heartbeatInterval = struct {
	component types.Component
	variable  types.Variable
}{types.Component{Name: "OCPPCommCtrlr"}, types.Variable{Name: "HeartbeatInterval"}}

// inspect reads and changes a station's variables: Central System to station
// calls, made with the *csms.ChargePoint the handler received.
func (h *handler) inspect(cp *csms.ChargePoint) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	got, err := cp.GetVariables(ctx, provisioning.GetVariablesRequest{GetVariableData: []types.GetVariableData{
		{Component: heartbeatInterval.component, Variable: heartbeatInterval.variable},
	}})
	if err != nil {
		h.log.Warn("get variables failed", "id", cp.ID(), "err", err)
		return
	}
	for _, r := range got.GetVariableResult {
		h.log.Info("variable", "id", cp.ID(), "name", r.Variable.Name, "value", r.AttributeValue, "status", r.AttributeStatus)
	}
	set, err := cp.SetVariables(ctx, provisioning.SetVariablesRequest{SetVariableData: []types.SetVariableData{
		{Component: heartbeatInterval.component, Variable: heartbeatInterval.variable, AttributeValue: fmt.Sprint(h.interval)},
	}})
	if err != nil {
		h.log.Warn("set variables failed", "id", cp.ID(), "err", err)
		return
	}
	for _, r := range set.SetVariableResult {
		h.log.Info("variable set", "id", cp.ID(), "name", r.Variable.Name, "status", r.AttributeStatus)
	}
	tariff, err := cp.SetDefaultTariff(ctx, tariffandcost.SetDefaultTariffRequest{
		EVSEID: 0, // every EVSE
		Tariff: types.Tariff{
			TariffID: "default", Currency: "EUR",
			Energy: &types.TariffEnergy{Prices: []types.TariffEnergyPrice{{PriceKwh: 0.30}}},
		},
	})
	if err != nil {
		h.log.Warn("set default tariff failed", "id", cp.ID(), "err", err)
	} else {
		h.log.Info("default tariff set", "id", cp.ID(), "status", tariff.Status)
	}
	if h.remoteStart {
		start, err := cp.RequestStartTransaction(ctx, remotecontrol.RequestStartTransactionRequest{
			IDToken:       types.IDToken{IDToken: "REMOTE01", Type: "Central"},
			RemoteStartID: 1,
		})
		if err != nil {
			h.log.Warn("request start failed", "id", cp.ID(), "err", err)
			return
		}
		h.log.Info("request start transaction", "id", cp.ID(), "status", start.Status)
	}
}

func (h *handler) NotifyReport(_ context.Context, cp *csms.ChargePoint, req *provisioning.NotifyReportRequest) (*provisioning.NotifyReportResponse, error) {
	h.log.Info("report", "id", cp.ID(), "request", req.RequestID, "items", len(req.ReportData))
	return &provisioning.NotifyReportResponse{}, nil
}

func (h *handler) Heartbeat(context.Context, *csms.ChargePoint, *availability.HeartbeatRequest) (*availability.HeartbeatResponse, error) {
	return &availability.HeartbeatResponse{CurrentTime: ocpp.Now()}, nil
}

func (h *handler) StatusNotification(_ context.Context, cp *csms.ChargePoint, req *availability.StatusNotificationRequest) (*availability.StatusNotificationResponse, error) {
	h.log.Info("status", "id", cp.ID(), "evse", req.EVSEID, "connector", req.ConnectorID, "status", req.ConnectorStatus)
	return &availability.StatusNotificationResponse{}, nil
}

func (h *handler) Authorize(_ context.Context, cp *csms.ChargePoint, req *authorization.AuthorizeRequest) (*authorization.AuthorizeResponse, error) {
	status := h.status(req.IDToken.IDToken)
	h.log.Info("authorize", "id", cp.ID(), "idToken", req.IDToken.IDToken, "status", status)
	return &authorization.AuthorizeResponse{IDTokenInfo: types.IDTokenInfo{Status: status}}, nil
}

// TransactionEvent receives every step of a transaction: Started, Updated, Ended.
func (h *handler) TransactionEvent(_ context.Context, cp *csms.ChargePoint, req *transactions.TransactionEventRequest) (*transactions.TransactionEventResponse, error) {
	h.log.Info("transaction event", "id", cp.ID(), "event", req.EventType, "transaction", req.TransactionInfo.TransactionID,
		"seq", req.SeqNo, "trigger", req.TriggerReason, "state", req.TransactionInfo.ChargingState)
	for _, mv := range req.MeterValue {
		for _, sv := range mv.SampledValue {
			h.log.Info("meter value", "id", cp.ID(), "transaction", req.TransactionInfo.TransactionID, "value", sv.Value, "measurand", sv.Measurand)
		}
	}
	if h.remoteStart && req.EventType == types.TransactionEventStarted && req.TriggerReason == types.TriggerReasonRemoteStart {
		go h.stopLater(cp, req.TransactionInfo.TransactionID)
	}
	resp := &transactions.TransactionEventResponse{}
	// Tell the station whether the idToken may charge: the answer to a
	// Started event that carried one.
	if req.IDToken != nil {
		resp.IDTokenInfo = &types.IDTokenInfo{Status: h.status(req.IDToken.IDToken)}
	}
	return resp, nil
}

// stopLater ends a transaction the Central System started, after a moment.
func (h *handler) stopLater(cp *csms.ChargePoint, transaction string) {
	time.Sleep(time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stop, err := cp.RequestStopTransaction(ctx, remotecontrol.RequestStopTransactionRequest{TransactionID: transaction})
	if err != nil {
		h.log.Warn("request stop failed", "id", cp.ID(), "err", err)
		return
	}
	h.log.Info("request stop transaction", "id", cp.ID(), "transaction", transaction, "status", stop.Status)
}

func (h *handler) MeterValues(_ context.Context, cp *csms.ChargePoint, req *metervalues.MeterValuesRequest) (*metervalues.MeterValuesResponse, error) {
	h.log.Info("meter values", "id", cp.ID(), "evse", req.EVSEID, "readings", len(req.MeterValue))
	return &metervalues.MeterValuesResponse{}, nil
}

// OpenPeriodicEventStream accepts the event streams stations open.
func (h *handler) OpenPeriodicEventStream(_ context.Context, cp *csms.ChargePoint, req *diagnostics.OpenPeriodicEventStreamRequest) (*diagnostics.OpenPeriodicEventStreamResponse, error) {
	h.log.Info("event stream opened", "id", cp.ID(), "stream", req.ConstantStreamData.ID)
	return &diagnostics.OpenPeriodicEventStreamResponse{Status: types.GenericStatusAccepted}, nil
}

// NotifyPeriodicEventStream receives the values of a stream. It is an
// unconfirmed message: nothing is sent back, so the handler returns only an error.
func (h *handler) NotifyPeriodicEventStream(_ context.Context, cp *csms.ChargePoint, msg *diagnostics.NotifyPeriodicEventStream) error {
	h.log.Info("event stream values", "id", cp.ID(), "stream", msg.ID, "values", len(msg.Data), "pending", msg.Pending)
	return nil
}
