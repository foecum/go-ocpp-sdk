// Command chargingstation is an example OCPP 2.0.1 charging station simulator.
//
// It connects to a Central System, boots, sends heartbeats, and runs charging
// sessions in a loop: authorize, then a transaction reported with
// TransactionEvent (Started, Updated with meter values, Ended). It answers the
// Central System's calls (GetVariables, SetVariables, RequestStartTransaction,
// RequestStopTransaction, Reset, TriggerMessage) and keeps working across
// connection drops.
//
//	go run ./examples/v201/csms -addr :8080                   # terminal 1
//	go run ./examples/v201/chargingstation -id CS001          # terminal 2
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
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/chargepoint/v201"
	"github.com/foecum/go-ocpp-sdk/v201/authorization"
	"github.com/foecum/go-ocpp-sdk/v201/availability"
	"github.com/foecum/go-ocpp-sdk/v201/provisioning"
	"github.com/foecum/go-ocpp-sdk/v201/remotecontrol"
	"github.com/foecum/go-ocpp-sdk/v201/transactions"
	"github.com/foecum/go-ocpp-sdk/v201/types"
)

func main() {
	var opts options
	flag.StringVar(&opts.url, "url", "ws://localhost:8080/ocpp", "Central System endpoint URL (the identity is appended)")
	flag.StringVar(&opts.id, "id", "CS001", "charging station identity")
	flag.StringVar(&opts.password, "password", "", "password for HTTP Basic authentication")
	flag.StringVar(&opts.tag, "tag", "04A1B2C3D4", "idToken used for simulated sessions")
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
		log:  logger,
		opts: opts,
		// A tiny device model: variable name to value.
		variables:   map[string]string{"HeartbeatInterval": "30", "MeterValueSampleInterval": "5"},
		remoteStart: make(chan string, 1),
		stopTx:      make(chan string, 1),
	}
	cp, err := chargepoint.New(chargepoint.Config{
		URL:           opts.url,
		ID:            opts.id,
		Password:      opts.password,
		Provisioning:  st,
		RemoteControl: st,
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
	// The station answers the Central System's Provisioning and Remote Control
	// calls. Everything else is NotSupported.
	chargepoint.UnimplementedProvisioningHandler
	chargepoint.UnimplementedRemoteControlHandler

	cp   *chargepoint.ChargePoint
	log  *slog.Logger
	opts options

	mu        sync.Mutex
	variables map[string]string

	active      atomic.Bool
	txCount     atomic.Int64
	current     atomic.Value // string: the id of the running transaction
	remoteStart chan string
	stopTx      chan string
}

// retry repeats f while the connection is down or the call timed out, waiting
// for the station to reconnect in between. It never retries a CALLERROR or an
// invalid request. Only repeat calls like these when the other side can cope
// with a duplicate.
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
		resp, err := retry(ctx, st, "BootNotification", func() (*provisioning.BootNotificationResponse, error) {
			return st.cp.BootNotification(ctx, provisioning.BootNotificationRequest{
				Reason: types.BootReasonPowerUp,
				ChargingStation: types.ChargingStation{
					VendorName: "Example", Model: "Simulator", FirmwareVersion: "1.0.0",
				},
			})
		})
		if err != nil {
			return err
		}
		st.log.Info("boot answered", "status", resp.Status, "interval", resp.Interval)
		if resp.Status == types.RegistrationStatusAccepted {
			st.mu.Lock()
			st.variables["HeartbeatInterval"] = strconv.Itoa(resp.Interval)
			st.mu.Unlock()
			st.status(ctx, types.ConnectorStatusAvailable)
			return nil
		}
		// Pending or Rejected: wait as long as the Central System says.
		wait := time.Duration(max(resp.Interval, 1)) * time.Second
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

func (st *station) status(ctx context.Context, status types.ConnectorStatus) {
	_, err := retry(ctx, st, "StatusNotification", func() (*availability.StatusNotificationResponse, error) {
		return st.cp.StatusNotification(ctx, availability.StatusNotificationRequest{
			Timestamp: ocpp.Now(), ConnectorStatus: status, EVSEID: 1, ConnectorID: 1,
		})
	})
	if err != nil && ctx.Err() == nil {
		st.log.Warn("status notification failed", "err", err)
	}
}

func (st *station) heartbeats(ctx context.Context) {
	for {
		st.mu.Lock()
		secs, _ := strconv.Atoi(st.variables["HeartbeatInterval"])
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
	if resp, err := st.cp.Heartbeat(ctx, availability.HeartbeatRequest{}); err != nil {
		st.log.Debug("heartbeat failed", "err", err)
	} else {
		st.log.Info("heartbeat", "centralSystemTime", resp.CurrentTime.Format(time.RFC3339))
	}
}

// sessions runs charging sessions until opts.sessions is reached.
func (st *station) sessions(ctx context.Context) error {
	for n := 0; st.opts.sessions == 0 || n < st.opts.sessions; n++ {
		tag, remote := st.opts.tag, false
		select {
		case <-ctx.Done():
			return ctx.Err()
		case tag = <-st.remoteStart: // the Central System asked for a session
			remote = true
		case <-time.After(st.opts.idle):
		}
		if err := st.session(ctx, tag, remote); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			st.log.Error("session failed", "err", err)
		}
	}
	return nil
}

// session is one charge: authorize, then Started, Updated... and Ended
// TransactionEvents that share a transaction id and count up in seqNo.
func (st *station) session(ctx context.Context, tag string, remote bool) error {
	st.active.Store(true)
	defer st.active.Store(false)

	token := types.IDToken{IDToken: tag, Type: types.IDTokenEnumISO14443}
	if remote {
		token.Type = types.IDTokenEnumCentral
	} else {
		auth, err := retry(ctx, st, "Authorize", func() (*authorization.AuthorizeResponse, error) {
			return st.cp.Authorize(ctx, authorization.AuthorizeRequest{IDToken: token})
		})
		if err != nil {
			return err
		}
		if auth.IDTokenInfo.Status != types.AuthorizationStatusAccepted {
			st.log.Warn("idToken refused", "idToken", tag, "status", auth.IDTokenInfo.Status)
			return nil
		}
	}

	txID := fmt.Sprintf("%s-%d", st.opts.id, st.txCount.Add(1))
	st.current.Store(txID)
	seq := 0
	connector := 1
	send := func(event types.TransactionEvent, trigger types.TriggerReason, state types.ChargingState,
		meter []types.MeterValue, stopped types.Reason, retryIt bool) (*transactions.TransactionEventResponse, error) {
		req := transactions.TransactionEventRequest{
			EventType: event, Timestamp: ocpp.Now(), TriggerReason: trigger, SeqNo: seq, MeterValue: meter,
			TransactionInfo: types.Transaction{TransactionID: txID, ChargingState: state, StoppedReason: stopped},
			EVSE:            &types.EVSE{ID: 1, ConnectorID: &connector},
		}
		if event == types.TransactionEventStarted {
			req.IDToken = &token
		}
		seq++
		call := func() (*transactions.TransactionEventResponse, error) { return st.cp.TransactionEvent(ctx, req) }
		if retryIt {
			return retry(ctx, st, "TransactionEvent", call)
		}
		return call()
	}

	st.status(ctx, types.ConnectorStatusOccupied)
	trigger := types.TriggerReasonAuthorized
	if remote {
		trigger = types.TriggerReasonRemoteStart
	}
	started, err := send(types.TransactionEventStarted, trigger, types.ChargingStateCharging, nil, "", true)
	if err != nil {
		return err
	}
	if started.IDTokenInfo != nil && started.IDTokenInfo.Status != types.AuthorizationStatusAccepted {
		st.log.Warn("Central System refused the idToken", "status", started.IDTokenInfo.Status)
	}
	st.log.Info("charging", "transaction", txID)

	energy := 0.0
	reason, endTrigger := types.ReasonLocal, types.TriggerReasonStopAuthorized
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
			if id == txID {
				reason, endTrigger = types.ReasonRemote, types.TriggerReasonRemoteStop
				break loop
			}
		case <-meter.C:
			energy += 500
			mv := []types.MeterValue{{Timestamp: ocpp.Now(), SampledValue: []types.SampledValue{{
				Value: energy, Measurand: types.MeasurandEnergyActiveImportRegister, Context: types.ReadingContextSamplePeriodic,
				UnitOfMeasure: &types.UnitOfMeasure{Unit: "Wh"},
			}}}}
			if _, err := send(types.TransactionEventUpdated, types.TriggerReasonMeterValuePeriodic, types.ChargingStateCharging, mv, "", false); err != nil {
				st.log.Warn("meter values not delivered", "err", err) // not retried: a later reading follows
			}
		}
	}

	_, err = send(types.TransactionEventEnded, endTrigger, types.ChargingStateIdle, nil, reason, true)
	st.status(ctx, types.ConnectorStatusAvailable)
	st.log.Info("session finished", "transaction", txID, "energyWh", energy, "reason", reason)
	return err
}

// ---- calls from the Central System ----

func (st *station) GetVariables(_ context.Context, req *provisioning.GetVariablesRequest) (*provisioning.GetVariablesResponse, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	resp := &provisioning.GetVariablesResponse{}
	for _, d := range req.GetVariableData {
		r := types.GetVariableResult{Component: d.Component, Variable: d.Variable, AttributeType: d.AttributeType}
		if v, ok := st.variables[d.Variable.Name]; ok {
			r.AttributeStatus, r.AttributeValue = types.GetVariableStatusAccepted, v
		} else {
			r.AttributeStatus = types.GetVariableStatusUnknownVariable
		}
		resp.GetVariableResult = append(resp.GetVariableResult, r)
	}
	return resp, nil
}

func (st *station) SetVariables(_ context.Context, req *provisioning.SetVariablesRequest) (*provisioning.SetVariablesResponse, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	resp := &provisioning.SetVariablesResponse{}
	for _, d := range req.SetVariableData {
		r := types.SetVariableResult{Component: d.Component, Variable: d.Variable, AttributeType: d.AttributeType}
		_, known := st.variables[d.Variable.Name]
		if n, err := strconv.Atoi(d.AttributeValue); !known {
			r.AttributeStatus = types.SetVariableStatusUnknownVariable
		} else if err != nil || n < 1 {
			r.AttributeStatus = types.SetVariableStatusRejected
		} else {
			st.variables[d.Variable.Name] = d.AttributeValue
			r.AttributeStatus = types.SetVariableStatusAccepted
			st.log.Info("variable changed", "name", d.Variable.Name, "value", d.AttributeValue)
		}
		resp.SetVariableResult = append(resp.SetVariableResult, r)
	}
	return resp, nil
}

func (st *station) Reset(_ context.Context, req *provisioning.ResetRequest) (*provisioning.ResetResponse, error) {
	// A real station would reboot after answering. This one just says so.
	st.log.Info("reset requested", "type", req.Type)
	return &provisioning.ResetResponse{Status: types.ResetStatusAccepted}, nil
}

func (st *station) RequestStartTransaction(_ context.Context, req *remotecontrol.RequestStartTransactionRequest) (*remotecontrol.RequestStartTransactionResponse, error) {
	if st.active.Load() {
		return &remotecontrol.RequestStartTransactionResponse{Status: types.RequestStartStopStatusRejected}, nil
	}
	select {
	case st.remoteStart <- req.IDToken.IDToken:
		st.log.Info("remote start accepted", "idToken", req.IDToken.IDToken)
		return &remotecontrol.RequestStartTransactionResponse{Status: types.RequestStartStopStatusAccepted}, nil
	default:
		return &remotecontrol.RequestStartTransactionResponse{Status: types.RequestStartStopStatusRejected}, nil
	}
}

func (st *station) RequestStopTransaction(_ context.Context, req *remotecontrol.RequestStopTransactionRequest) (*remotecontrol.RequestStopTransactionResponse, error) {
	cur, _ := st.current.Load().(string)
	if !st.active.Load() || req.TransactionID != cur {
		return &remotecontrol.RequestStopTransactionResponse{Status: types.RequestStartStopStatusRejected}, nil
	}
	select {
	case st.stopTx <- req.TransactionID:
	default:
	}
	return &remotecontrol.RequestStopTransactionResponse{Status: types.RequestStartStopStatusAccepted}, nil
}

// TriggerMessage answers a request to send a message right now. The work
// happens in a goroutine, after the answer has gone out.
func (st *station) TriggerMessage(ctx context.Context, req *remotecontrol.TriggerMessageRequest) (*remotecontrol.TriggerMessageResponse, error) {
	switch req.RequestedMessage {
	case types.MessageTriggerHeartbeat:
		go st.heartbeat(context.WithoutCancel(ctx))
	case types.MessageTriggerStatusNotification:
		go st.status(context.WithoutCancel(ctx), types.ConnectorStatusAvailable)
	default:
		return &remotecontrol.TriggerMessageResponse{Status: types.TriggerMessageStatusNotImplemented}, nil
	}
	return &remotecontrol.TriggerMessageResponse{Status: types.TriggerMessageStatusAccepted}, nil
}
