package main

import (
	"context"
	"net"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/chargepoint/v201"
	"github.com/foecum/go-ocpp-sdk/csms/v201"
	"github.com/foecum/go-ocpp-sdk/v201/authorization"
	"github.com/foecum/go-ocpp-sdk/v201/availability"
	"github.com/foecum/go-ocpp-sdk/v201/provisioning"
	"github.com/foecum/go-ocpp-sdk/v201/transactions"
	"github.com/foecum/go-ocpp-sdk/v201/types"
)

type handler201 struct {
	csms.UnimplementedProvisioningHandler
	csms.UnimplementedAvailabilityHandler
	csms.UnimplementedAuthorizationHandler
	csms.UnimplementedTransactionsHandler
}

func (handler201) BootNotification(context.Context, *csms.ChargePoint, *provisioning.BootNotificationRequest) (*provisioning.BootNotificationResponse, error) {
	return &provisioning.BootNotificationResponse{Status: types.RegistrationStatusAccepted, CurrentTime: ocpp.Now(), Interval: 60}, nil
}
func (handler201) Heartbeat(context.Context, *csms.ChargePoint, *availability.HeartbeatRequest) (*availability.HeartbeatResponse, error) {
	return &availability.HeartbeatResponse{CurrentTime: ocpp.Now()}, nil
}
func (handler201) Authorize(context.Context, *csms.ChargePoint, *authorization.AuthorizeRequest) (*authorization.AuthorizeResponse, error) {
	return &authorization.AuthorizeResponse{IDTokenInfo: types.IDTokenInfo{Status: types.AuthorizationStatusAccepted}}, nil
}
func (handler201) TransactionEvent(context.Context, *csms.ChargePoint, *transactions.TransactionEventRequest) (*transactions.TransactionEventResponse, error) {
	return &transactions.TransactionEventResponse{}, nil
}

type central201 struct{ srv *csms.CSMS }

func (c central201) serve(ln net.Listener) error        { return c.srv.Serve(ln) }
func (c central201) shutdown(ctx context.Context) error { return c.srv.Shutdown(ctx) }

type station201 struct{ cp *chargepoint.ChargePoint }

func (s station201) run(ctx context.Context)                 { _ = s.cp.Run(ctx) }
func (s station201) waitConnected(ctx context.Context) error { return s.cp.WaitConnected(ctx) }
func (s station201) close()                                  { _ = s.cp.Close() }
func (s station201) boot(ctx context.Context) error {
	_, err := s.cp.BootNotification(ctx, provisioning.BootNotificationRequest{
		Reason: types.BootReasonPowerUp, ChargingStation: types.ChargingStation{VendorName: "Load", Model: "Sim"}})
	return err
}
func (s station201) heartbeat(ctx context.Context) error {
	_, err := s.cp.Heartbeat(ctx, availability.HeartbeatRequest{})
	return err
}

// chargingSession is Authorize and a Started, Updated and Ended TransactionEvent.
func (s station201) chargingSession(ctx context.Context) error {
	token := types.IDToken{IDToken: "TAG", Type: types.IDTokenEnumISO14443}
	if _, err := s.cp.Authorize(ctx, authorization.AuthorizeRequest{IDToken: token}); err != nil {
		return err
	}
	for seq, step := range []struct {
		event   types.TransactionEvent
		trigger types.TriggerReason
	}{
		{types.TransactionEventStarted, types.TriggerReasonAuthorized},
		{types.TransactionEventUpdated, types.TriggerReasonMeterValuePeriodic},
		{types.TransactionEventEnded, types.TriggerReasonStopAuthorized},
	} {
		req := transactions.TransactionEventRequest{
			EventType: step.event, Timestamp: ocpp.Now(), TriggerReason: step.trigger, SeqNo: seq,
			TransactionInfo: types.Transaction{TransactionID: "TX"},
		}
		if step.event == types.TransactionEventUpdated {
			req.MeterValue = []types.MeterValue{{Timestamp: ocpp.Now(), SampledValue: []types.SampledValue{{Value: 500}}}}
		}
		if _, err := s.cp.TransactionEvent(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

func factory201() factory {
	return factory{
		newCentral: func() (central, error) {
			srv, err := csms.New(csms.Config{
				Provisioning: handler201{}, Availability: handler201{}, Authorization: handler201{}, Transactions: handler201{}})
			return central201{srv}, err
		},
		newStation: func(url, id string) (simulated, error) {
			cp, err := chargepoint.New(chargepoint.Config{URL: url, ID: id})
			return station201{cp}, err
		},
	}
}
