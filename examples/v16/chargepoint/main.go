// Command chargepoint is an example OCPP 1.6 charging station simulator.
//
// It connects to a Central System, boots, sends heartbeats, and runs charging
// sessions in a loop: authorize, start a transaction, report meter values, stop.
// It also answers the Central System's calls (RemoteStartTransaction, Reset,
// GetConfiguration and so on) and keeps working across connection drops.
//
//	go run ./examples/v16/csms -addr :8080                  # terminal 1
//	go run ./examples/v16/chargepoint -id CS001             # terminal 2
//
// Use -password if the Central System requires HTTP Basic authentication, and
// -sessions N to stop after N sessions.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/chargepoint/v16"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/remotetrigger"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

func main() {
	var opts options
	flag.StringVar(&opts.url, "url", "ws://localhost:8080/ocpp", "Central System endpoint URL (the identity is appended)")
	flag.StringVar(&opts.id, "id", "CS001", "charge point identity")
	flag.StringVar(&opts.password, "password", "", "password for HTTP Basic authentication")
	flag.StringVar(&opts.tag, "tag", "04A1B2C3D4", "idTag used for simulated sessions")
	flag.IntVar(&opts.sessions, "sessions", 0, "stop after this many sessions (0 runs until interrupted)")
	flag.DurationVar(&opts.sessionLength, "session-length", 20*time.Second, "how long each session lasts")
	flag.DurationVar(&opts.meterEvery, "meter-every", 5*time.Second, "interval between meter values during a session")
	flag.DurationVar(&opts.idle, "idle", 10*time.Second, "pause between sessions")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, opts, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

type options struct {
	url, id, password, tag string
	sessions               int
	sessionLength          time.Duration
	meterEvery, idle       time.Duration
}

// run boots the station and drives sessions until they are done or ctx ends.
func run(ctx context.Context, opts options, out io.Writer) error {
	logger := slog.New(slog.NewTextHandler(out, nil))
	st := &station{
		log:         logger,
		opts:        opts,
		config:      map[string]string{"HeartbeatInterval": "30", "MeterValueSampleInterval": "5"},
		remoteStart: make(chan string, 1),
		stopTx:      make(chan int, 1),
	}
	cp, err := chargepoint.New(chargepoint.Config{
		URL:           opts.url,
		ID:            opts.id,
		Password:      opts.password,
		Core:          st,
		RemoteTrigger: st,
		Logger:        logger,
		OnConnect:     func() { logger.Info("connected") },
		OnDisconnect:  func(err error) { logger.Warn("disconnected, will reconnect", "reason", err) },
	})
	if err != nil {
		return err
	}
	st.cp = cp

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() { _ = cp.Run(ctx) }()
	defer cp.Close()

	if err := st.boot(ctx); err != nil {
		return ignoreCancel(err)
	}
	go st.heartbeats(ctx)
	return ignoreCancel(st.sessions(ctx))
}

func ignoreCancel(err error) error {
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

type station struct {
	// The station answers the Central System's Core calls and TriggerMessage.
	// Everything else is NotSupported.
	chargepoint.UnimplementedCoreHandler
	chargepoint.UnimplementedRemoteTriggerHandler

	cp   *chargepoint.ChargePoint
	log  *slog.Logger
	opts options

	mu     sync.Mutex
	config map[string]string

	active      atomic.Bool
	transaction atomic.Int64
	remoteStart chan string
	stopTx      chan int
}

// retry repeats f while the connection is down or the call timed out, waiting
// for the station to reconnect in between. It never retries a CALLERROR or an
// invalid request. Only repeat calls like these when the other side can cope
// with a duplicate: OCPP 1.6 asks stations to retry transaction messages.
func retry[T any](ctx context.Context, st *station, what string, f func() (T, error)) (T, error) {
	var zero T
	for attempt := 1; attempt <= 5; attempt++ {
		v, err := f()
		if err == nil {
			return v, nil
		}
		if !errors.Is(err, ocpp.ErrNotConnected) && !errors.Is(err, ocpp.ErrTimeout) {
			return zero, err
		}
		st.log.Warn("call failed, waiting for the connection", "call", what, "attempt", attempt, "err", err)
		if err := st.cp.WaitConnected(ctx); err != nil {
			return zero, err
		}
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return zero, fmt.Errorf("%s: giving up after 5 attempts", what)
}

// boot registers with the Central System, repeating until it is accepted.
func (st *station) boot(ctx context.Context) error {
	for {
		if err := st.cp.WaitConnected(ctx); err != nil {
			return err
		}
		conf, err := retry(ctx, st, "BootNotification", func() (*core.BootNotificationConfirmation, error) {
			return st.cp.BootNotification(ctx, core.BootNotificationRequest{
				ChargePointVendor: "Example",
				ChargePointModel:  "Simulator",
				FirmwareVersion:   "1.0.0",
			})
		})
		if err != nil {
			return err
		}
		st.log.Info("boot answered", "status", conf.Status, "interval", conf.Interval)
		if conf.Status == types.RegistrationStatusAccepted {
			st.mu.Lock()
			st.config["HeartbeatInterval"] = strconv.Itoa(conf.Interval)
			st.mu.Unlock()
			st.status(ctx, 0, types.ChargePointStatusAvailable)
			st.status(ctx, 1, types.ChargePointStatusAvailable)
			return nil
		}
		// Pending or Rejected: wait as long as the Central System says.
		wait := time.Duration(max(conf.Interval, 1)) * time.Second
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

func (st *station) status(ctx context.Context, connector int, status types.ChargePointStatus) {
	_, err := retry(ctx, st, "StatusNotification", func() (*core.StatusNotificationConfirmation, error) {
		return st.cp.StatusNotification(ctx, core.StatusNotificationRequest{
			ConnectorID: connector, ErrorCode: types.ChargePointErrorCodeNoError, Status: status, Timestamp: types.Now().Ptr(),
		})
	})
	if err != nil && ctx.Err() == nil {
		st.log.Warn("status notification failed", "err", err)
	}
}

func (st *station) heartbeats(ctx context.Context) {
	for {
		st.mu.Lock()
		secs, _ := strconv.Atoi(st.config["HeartbeatInterval"])
		st.mu.Unlock()
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(max(secs, 1)) * time.Second):
		}
		st.heartbeat(ctx)
	}
}

func (st *station) heartbeat(ctx context.Context) {
	// Not retried: the next heartbeat is due soon anyway.
	if conf, err := st.cp.Heartbeat(ctx, core.HeartbeatRequest{}); err != nil {
		st.log.Debug("heartbeat failed", "err", err)
	} else {
		st.log.Info("heartbeat", "centralSystemTime", conf.CurrentTime.Format(time.RFC3339))
	}
}

// sessions runs charging sessions until opts.sessions is reached.
func (st *station) sessions(ctx context.Context) error {
	for n := 0; st.opts.sessions == 0 || n < st.opts.sessions; n++ {
		tag := st.opts.tag
		select {
		case <-ctx.Done():
			return ctx.Err()
		case tag = <-st.remoteStart: // the Central System asked for a session
		case <-time.After(st.opts.idle):
		}
		if err := st.session(ctx, tag); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			st.log.Error("session failed", "err", err)
		}
	}
	return nil
}

// session is one charge: authorize, start, meter values, stop.
func (st *station) session(ctx context.Context, tag string) error {
	st.active.Store(true)
	defer st.active.Store(false)

	auth, err := retry(ctx, st, "Authorize", func() (*core.AuthorizeConfirmation, error) {
		return st.cp.Authorize(ctx, core.AuthorizeRequest{IDTag: tag})
	})
	if err != nil {
		return err
	}
	if auth.IDTagInfo.Status != types.AuthorizationStatusAccepted {
		st.log.Warn("idTag refused", "idTag", tag, "status", auth.IDTagInfo.Status)
		return nil
	}

	energy := 0
	st.status(ctx, 1, types.ChargePointStatusPreparing)
	start, err := retry(ctx, st, "StartTransaction", func() (*core.StartTransactionConfirmation, error) {
		return st.cp.StartTransaction(ctx, core.StartTransactionRequest{ConnectorID: 1, IDTag: tag, MeterStart: energy, Timestamp: types.Now()})
	})
	if err != nil {
		return err
	}
	tx := start.TransactionID
	st.transaction.Store(int64(tx))
	st.log.Info("charging", "transaction", tx)
	st.status(ctx, 1, types.ChargePointStatusCharging)

	reason := types.ReasonLocal
	deadline := time.After(st.opts.sessionLength)
	meter := time.NewTicker(st.opts.meterEvery)
	defer meter.Stop()
loop:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			break loop
		case id := <-st.stopTx:
			if id == tx {
				reason = types.ReasonRemote
				break loop
			}
		case <-meter.C:
			energy += 500
			_, err := st.cp.MeterValues(ctx, core.MeterValuesRequest{
				ConnectorID: 1, TransactionID: &tx,
				MeterValue: []types.MeterValue{{Timestamp: types.Now(), SampledValue: []types.SampledValue{{
					Value: strconv.Itoa(energy), Measurand: types.MeasurandEnergyActiveImportRegister, Unit: types.UnitOfMeasureWh,
				}}}},
			})
			if err != nil {
				st.log.Warn("meter values not delivered", "err", err) // not retried: a later reading follows
			}
		}
	}

	st.status(ctx, 1, types.ChargePointStatusFinishing)
	_, err = retry(ctx, st, "StopTransaction", func() (*core.StopTransactionConfirmation, error) {
		return st.cp.StopTransaction(ctx, core.StopTransactionRequest{
			IDTag: tag, MeterStop: energy, Timestamp: types.Now(), TransactionID: tx, Reason: reason,
		})
	})
	st.status(ctx, 1, types.ChargePointStatusAvailable)
	st.log.Info("session finished", "transaction", tx, "energyWh", energy, "reason", reason)
	return err
}

// ---- calls from the Central System ----

func (st *station) ChangeAvailability(_ context.Context, req *core.ChangeAvailabilityRequest) (*core.ChangeAvailabilityConfirmation, error) {
	st.log.Info("change availability", "connector", req.ConnectorID, "type", req.Type)
	return &core.ChangeAvailabilityConfirmation{Status: types.AvailabilityStatusAccepted}, nil
}

func (st *station) ChangeConfiguration(_ context.Context, req *core.ChangeConfigurationRequest) (*core.ChangeConfigurationConfirmation, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, known := st.config[req.Key]; !known {
		return &core.ChangeConfigurationConfirmation{Status: types.ConfigurationStatusNotSupported}, nil
	}
	if n, err := strconv.Atoi(req.Value); err != nil || n < 1 {
		return &core.ChangeConfigurationConfirmation{Status: types.ConfigurationStatusRejected}, nil
	}
	st.config[req.Key] = req.Value
	st.log.Info("configuration changed", "key", req.Key, "value", req.Value)
	return &core.ChangeConfigurationConfirmation{Status: types.ConfigurationStatusAccepted}, nil
}

func (st *station) ClearCache(context.Context, *core.ClearCacheRequest) (*core.ClearCacheConfirmation, error) {
	return &core.ClearCacheConfirmation{Status: types.ClearCacheStatusAccepted}, nil
}

func (st *station) DataTransfer(context.Context, *core.DataTransferRequest) (*core.DataTransferConfirmation, error) {
	return &core.DataTransferConfirmation{Status: types.DataTransferStatusUnknownVendorId}, nil
}

func (st *station) GetConfiguration(_ context.Context, req *core.GetConfigurationRequest) (*core.GetConfigurationConfirmation, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	keys := req.Key
	if len(keys) == 0 {
		for k := range st.config {
			keys = append(keys, k)
		}
		sort.Strings(keys)
	}
	conf := &core.GetConfigurationConfirmation{}
	for _, k := range keys {
		if v, ok := st.config[k]; ok {
			conf.ConfigurationKey = append(conf.ConfigurationKey, types.KeyValue{Key: k, Value: v})
		} else {
			conf.UnknownKey = append(conf.UnknownKey, k)
		}
	}
	return conf, nil
}

func (st *station) RemoteStartTransaction(_ context.Context, req *core.RemoteStartTransactionRequest) (*core.RemoteStartTransactionConfirmation, error) {
	if st.active.Load() {
		return &core.RemoteStartTransactionConfirmation{Status: types.RemoteStartStopStatusRejected}, nil
	}
	select {
	case st.remoteStart <- req.IDTag:
		st.log.Info("remote start accepted", "idTag", req.IDTag)
		return &core.RemoteStartTransactionConfirmation{Status: types.RemoteStartStopStatusAccepted}, nil
	default:
		return &core.RemoteStartTransactionConfirmation{Status: types.RemoteStartStopStatusRejected}, nil
	}
}

func (st *station) RemoteStopTransaction(_ context.Context, req *core.RemoteStopTransactionRequest) (*core.RemoteStopTransactionConfirmation, error) {
	if !st.active.Load() || int64(req.TransactionID) != st.transaction.Load() {
		return &core.RemoteStopTransactionConfirmation{Status: types.RemoteStartStopStatusRejected}, nil
	}
	select {
	case st.stopTx <- req.TransactionID:
	default:
	}
	return &core.RemoteStopTransactionConfirmation{Status: types.RemoteStartStopStatusAccepted}, nil
}

func (st *station) Reset(_ context.Context, req *core.ResetRequest) (*core.ResetConfirmation, error) {
	// A real station would reboot after answering. This one just says so.
	st.log.Info("reset requested", "type", req.Type)
	return &core.ResetConfirmation{Status: types.ResetStatusAccepted}, nil
}

func (st *station) UnlockConnector(_ context.Context, req *core.UnlockConnectorRequest) (*core.UnlockConnectorConfirmation, error) {
	st.log.Info("unlock connector", "connector", req.ConnectorID)
	return &core.UnlockConnectorConfirmation{Status: types.UnlockStatusUnlocked}, nil
}

// TriggerMessage answers a request to send a message right now. The work
// happens in a goroutine, after the answer has gone out.
func (st *station) TriggerMessage(ctx context.Context, req *remotetrigger.TriggerMessageRequest) (*remotetrigger.TriggerMessageConfirmation, error) {
	switch req.RequestedMessage {
	case types.MessageTriggerHeartbeat:
		go st.heartbeat(context.WithoutCancel(ctx))
	case types.MessageTriggerStatusNotification:
		go st.status(context.WithoutCancel(ctx), 1, types.ChargePointStatusAvailable)
	default:
		return &remotetrigger.TriggerMessageConfirmation{Status: types.TriggerMessageStatusNotImplemented}, nil
	}
	return &remotetrigger.TriggerMessageConfirmation{Status: types.TriggerMessageStatusAccepted}, nil
}
