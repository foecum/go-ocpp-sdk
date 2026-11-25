package main

import (
	"context"
	"net"

	"github.com/foecum/go-ocpp-sdk/chargepoint/v16"
	"github.com/foecum/go-ocpp-sdk/csms/v16"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

type handler16 struct{ csms.UnimplementedCoreHandler }

func (handler16) BootNotification(context.Context, *csms.ChargePoint, *core.BootNotificationRequest) (*core.BootNotificationConfirmation, error) {
	return &core.BootNotificationConfirmation{Status: types.RegistrationStatusAccepted, CurrentTime: types.Now(), Interval: 60}, nil
}
func (handler16) Heartbeat(context.Context, *csms.ChargePoint, *core.HeartbeatRequest) (*core.HeartbeatConfirmation, error) {
	return &core.HeartbeatConfirmation{CurrentTime: types.Now()}, nil
}
func (handler16) Authorize(context.Context, *csms.ChargePoint, *core.AuthorizeRequest) (*core.AuthorizeConfirmation, error) {
	return &core.AuthorizeConfirmation{IDTagInfo: types.IDTagInfo{Status: types.AuthorizationStatusAccepted}}, nil
}
func (handler16) StartTransaction(context.Context, *csms.ChargePoint, *core.StartTransactionRequest) (*core.StartTransactionConfirmation, error) {
	return &core.StartTransactionConfirmation{TransactionID: 1, IDTagInfo: types.IDTagInfo{Status: types.AuthorizationStatusAccepted}}, nil
}
func (handler16) MeterValues(context.Context, *csms.ChargePoint, *core.MeterValuesRequest) (*core.MeterValuesConfirmation, error) {
	return &core.MeterValuesConfirmation{}, nil
}
func (handler16) StopTransaction(context.Context, *csms.ChargePoint, *core.StopTransactionRequest) (*core.StopTransactionConfirmation, error) {
	return &core.StopTransactionConfirmation{}, nil
}

type central16 struct{ srv *csms.CSMS }

func (c central16) serve(ln net.Listener) error        { return c.srv.Serve(ln) }
func (c central16) shutdown(ctx context.Context) error { return c.srv.Shutdown(ctx) }

type station16 struct{ cp *chargepoint.ChargePoint }

func (s station16) run(ctx context.Context)                 { _ = s.cp.Run(ctx) }
func (s station16) waitConnected(ctx context.Context) error { return s.cp.WaitConnected(ctx) }
func (s station16) close()                                  { _ = s.cp.Close() }
func (s station16) boot(ctx context.Context) error {
	_, err := s.cp.BootNotification(ctx, core.BootNotificationRequest{ChargePointVendor: "Load", ChargePointModel: "Sim"})
	return err
}
func (s station16) heartbeat(ctx context.Context) error {
	_, err := s.cp.Heartbeat(ctx, core.HeartbeatRequest{})
	return err
}

// chargingSession is Authorize, StartTransaction, MeterValues and StopTransaction.
func (s station16) chargingSession(ctx context.Context) error {
	if _, err := s.cp.Authorize(ctx, core.AuthorizeRequest{IDTag: "TAG"}); err != nil {
		return err
	}
	start, err := s.cp.StartTransaction(ctx, core.StartTransactionRequest{ConnectorID: 1, IDTag: "TAG", MeterStart: 0, Timestamp: types.Now()})
	if err != nil {
		return err
	}
	tx := start.TransactionID
	if _, err := s.cp.MeterValues(ctx, core.MeterValuesRequest{ConnectorID: 1, TransactionID: &tx,
		MeterValue: []types.MeterValue{{Timestamp: types.Now(), SampledValue: []types.SampledValue{{Value: "500", Unit: types.UnitOfMeasureWh}}}}}); err != nil {
		return err
	}
	_, err = s.cp.StopTransaction(ctx, core.StopTransactionRequest{TransactionID: tx, MeterStop: 500, Timestamp: types.Now()})
	return err
}

func factory16() factory {
	return factory{
		newCentral: func() (central, error) {
			srv, err := csms.New(csms.Config{Core: handler16{}})
			return central16{srv}, err
		},
		newStation: func(url, id string) (simulated, error) {
			cp, err := chargepoint.New(chargepoint.Config{URL: url, ID: id})
			return station16{cp}, err
		},
	}
}
