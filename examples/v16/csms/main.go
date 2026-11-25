// Command csms is an example OCPP 1.6 Central System.
//
// It accepts charge points over WebSocket, logs everything they send, hands out
// transaction ids, and asks each station for its configuration after it boots.
//
//	go run ./examples/v16/csms -addr :8080
//
// Then, in another terminal, start the example charge point:
//
//	go run ./examples/v16/chargepoint -url ws://localhost:8080/ocpp -id CS001
//
// Charge points connect to ws://host:8080/ocpp/<identity>. Use -password to
// require HTTP Basic authentication (the username is the identity) and -tags to
// accept only some idTags.
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
	"sync/atomic"
	"time"

	"github.com/foecum/go-ocpp-sdk/csms/v16"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/types"
	"github.com/foecum/go-ocpp-sdk/ws"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	password := flag.String("password", "", "if set, charge points must authenticate with this password (HTTP Basic, username = identity)")
	tags := flag.String("tags", "", "comma-separated idTags to accept; empty accepts every idTag")
	interval := flag.Int("interval", 30, "heartbeat interval in seconds sent to charge points")
	flag.Parse()

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	opts := options{password: *password, interval: *interval}
	if *tags != "" {
		opts.tags = strings.Split(*tags, ",")
	}
	if err := run(ctx, ln, opts, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

type options struct {
	password string
	tags     []string
	interval int
}

// run serves until ctx ends.
func run(ctx context.Context, ln net.Listener, opts options, out io.Writer) error {
	logger := slog.New(slog.NewTextHandler(out, nil))
	h := &handler{log: logger, interval: opts.interval}
	if len(opts.tags) > 0 {
		h.tags = make(map[string]bool)
		for _, t := range opts.tags {
			h.tags[strings.ToLower(strings.TrimSpace(t))] = true
		}
	}
	cfg := csms.Config{
		Core:   h,
		Logger: logger,
		OnConnect: func(cp *csms.ChargePoint) {
			logger.Info("charge point connected", "id", cp.ID(), "addr", cp.RemoteAddr())
		},
		OnDisconnect: func(cp *csms.ChargePoint, err error) {
			logger.Info("charge point disconnected", "id", cp.ID(), "reason", err)
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

// handler answers the Core calls charge points send. Everything else it
// inherits from UnimplementedCoreHandler, which replies NotSupported.
type handler struct {
	csms.UnimplementedCoreHandler
	log      *slog.Logger
	tags     map[string]bool // nil accepts every idTag
	interval int
	nextTx   atomic.Int64
}

func (h *handler) status(idTag string) types.AuthorizationStatus {
	if h.tags == nil || h.tags[strings.ToLower(idTag)] {
		return types.AuthorizationStatusAccepted
	}
	return types.AuthorizationStatusInvalid
}

func (h *handler) BootNotification(ctx context.Context, cp *csms.ChargePoint, req *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
	h.log.Info("boot notification", "id", cp.ID(), "vendor", req.ChargePointVendor, "model", req.ChargePointModel, "firmware", req.FirmwareVersion)
	go h.inspect(cp) // call the station back, but not while handling its request
	return &core.BootNotificationConfirmation{
		Status:      types.RegistrationStatusAccepted,
		CurrentTime: types.Now(),
		Interval:    h.interval,
	}, nil
}

// inspect reads the station's configuration: a Central System to charge point
// call, made with the *csms.ChargePoint the handler received.
func (h *handler) inspect(cp *csms.ChargePoint) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conf, err := cp.GetConfiguration(ctx, core.GetConfigurationRequest{})
	if err != nil {
		h.log.Warn("get configuration failed", "id", cp.ID(), "err", err)
		return
	}
	h.log.Info("configuration", "id", cp.ID(), "keys", len(conf.ConfigurationKey))
}

func (h *handler) Heartbeat(context.Context, *csms.ChargePoint, *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
	return &core.HeartbeatConfirmation{CurrentTime: types.Now()}, nil
}

func (h *handler) StatusNotification(_ context.Context, cp *csms.ChargePoint, req *core.StatusNotificationRequest) (*core.StatusNotificationConfirmation, error) {
	h.log.Info("status", "id", cp.ID(), "connector", req.ConnectorID, "status", req.Status, "error", req.ErrorCode)
	return &core.StatusNotificationConfirmation{}, nil
}

func (h *handler) Authorize(_ context.Context, cp *csms.ChargePoint, req *core.AuthorizeRequest) (*core.AuthorizeConfirmation, error) {
	status := h.status(req.IDTag)
	h.log.Info("authorize", "id", cp.ID(), "idTag", req.IDTag, "status", status)
	return &core.AuthorizeConfirmation{IDTagInfo: types.IDTagInfo{Status: status}}, nil
}

func (h *handler) StartTransaction(_ context.Context, cp *csms.ChargePoint, req *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	status := h.status(req.IDTag)
	tx := int(h.nextTx.Add(1))
	h.log.Info("start transaction", "id", cp.ID(), "connector", req.ConnectorID, "idTag", req.IDTag, "meterStart", req.MeterStart, "transaction", tx, "status", status)
	return &core.StartTransactionConfirmation{TransactionID: tx, IDTagInfo: types.IDTagInfo{Status: status}}, nil
}

func (h *handler) MeterValues(_ context.Context, cp *csms.ChargePoint, req *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	for _, mv := range req.MeterValue {
		for _, sv := range mv.SampledValue {
			h.log.Info("meter value", "id", cp.ID(), "connector", req.ConnectorID, "value", sv.Value, "unit", sv.Unit)
		}
	}
	return &core.MeterValuesConfirmation{}, nil
}

func (h *handler) StopTransaction(_ context.Context, cp *csms.ChargePoint, req *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	h.log.Info("stop transaction", "id", cp.ID(), "transaction", req.TransactionID, "meterStop", req.MeterStop, "reason", req.Reason)
	conf := &core.StopTransactionConfirmation{}
	if req.IDTag != "" {
		conf.IDTagInfo = &types.IDTagInfo{Status: h.status(req.IDTag)}
	}
	return conf, nil
}

// DataTransfer shows how to refuse a vendor extension politely: a CALLERROR is
// for protocol problems, a "Rejected" status is for requests you decline.
func (h *handler) DataTransfer(_ context.Context, cp *csms.ChargePoint, req *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	h.log.Info("data transfer", "id", cp.ID(), "vendor", req.VendorID, "message", req.MessageID)
	return &core.DataTransferConfirmation{Status: types.DataTransferStatusUnknownVendorId}, nil
}
