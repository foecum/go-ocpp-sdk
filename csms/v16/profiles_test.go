package csms_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/chargepoint/v16"
	"github.com/foecum/go-ocpp-sdk/csms/v16"
	"github.com/foecum/go-ocpp-sdk/v16/firmware"
	"github.com/foecum/go-ocpp-sdk/v16/localauth"
	"github.com/foecum/go-ocpp-sdk/v16/remotetrigger"
	"github.com/foecum/go-ocpp-sdk/v16/reservation"
	"github.com/foecum/go-ocpp-sdk/v16/smartcharging"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// profileStation implements every non-Core handler a charge point can have and
// records what it was asked.
type profileStation struct {
	mu   sync.Mutex
	seen []string
}

func (p *profileStation) note(format string, args ...any) {
	p.mu.Lock()
	p.seen = append(p.seen, fmt.Sprintf(format, args...))
	p.mu.Unlock()
}

func (p *profileStation) GetDiagnostics(_ context.Context, r *firmware.GetDiagnosticsRequest) (*firmware.GetDiagnosticsConfirmation, error) {
	p.note("GetDiagnostics:%s", r.Location)
	return &firmware.GetDiagnosticsConfirmation{FileName: "diag.zip"}, nil
}
func (p *profileStation) UpdateFirmware(_ context.Context, r *firmware.UpdateFirmwareRequest) (*firmware.UpdateFirmwareConfirmation, error) {
	p.note("UpdateFirmware:%s", r.Location)
	return &firmware.UpdateFirmwareConfirmation{}, nil
}
func (p *profileStation) GetLocalListVersion(context.Context, *localauth.GetLocalListVersionRequest) (*localauth.GetLocalListVersionConfirmation, error) {
	p.note("GetLocalListVersion")
	return &localauth.GetLocalListVersionConfirmation{ListVersion: 239}, nil
}
func (p *profileStation) SendLocalList(_ context.Context, r *localauth.SendLocalListRequest) (*localauth.SendLocalListConfirmation, error) {
	p.note("SendLocalList:%d:%s:%d", r.ListVersion, r.UpdateType, len(r.LocalAuthorizationList))
	return &localauth.SendLocalListConfirmation{Status: types.UpdateStatusAccepted}, nil
}
func (p *profileStation) ReserveNow(_ context.Context, r *reservation.ReserveNowRequest) (*reservation.ReserveNowConfirmation, error) {
	p.note("ReserveNow:%d:%s", r.ReservationID, r.IDTag)
	return &reservation.ReserveNowConfirmation{Status: types.ReservationStatusOccupied}, nil
}
func (p *profileStation) CancelReservation(_ context.Context, r *reservation.CancelReservationRequest) (*reservation.CancelReservationConfirmation, error) {
	p.note("CancelReservation:%d", r.ReservationID)
	return &reservation.CancelReservationConfirmation{Status: types.CancelReservationStatusAccepted}, nil
}
func (p *profileStation) SetChargingProfile(_ context.Context, r *smartcharging.SetChargingProfileRequest) (*smartcharging.SetChargingProfileConfirmation, error) {
	p.note("SetChargingProfile:%d:%d", r.ConnectorID, r.CSChargingProfiles.ChargingProfileID)
	return &smartcharging.SetChargingProfileConfirmation{Status: types.ChargingProfileStatusAccepted}, nil
}
func (p *profileStation) ClearChargingProfile(context.Context, *smartcharging.ClearChargingProfileRequest) (*smartcharging.ClearChargingProfileConfirmation, error) {
	p.note("ClearChargingProfile")
	return &smartcharging.ClearChargingProfileConfirmation{Status: types.ClearChargingProfileStatusUnknown}, nil
}
func (p *profileStation) GetCompositeSchedule(_ context.Context, r *smartcharging.GetCompositeScheduleRequest) (*smartcharging.GetCompositeScheduleConfirmation, error) {
	p.note("GetCompositeSchedule:%d:%d", r.ConnectorID, r.Duration)
	return &smartcharging.GetCompositeScheduleConfirmation{Status: types.GetCompositeScheduleStatusRejected}, nil
}
func (p *profileStation) TriggerMessage(_ context.Context, r *remotetrigger.TriggerMessageRequest) (*remotetrigger.TriggerMessageConfirmation, error) {
	p.note("TriggerMessage:%s", r.RequestedMessage)
	return &remotetrigger.TriggerMessageConfirmation{Status: types.TriggerMessageStatusAccepted}, nil
}

func withProfiles(p *profileStation) func(*chargepoint.Config) {
	return func(c *chargepoint.Config) {
		c.Firmware, c.LocalAuth, c.Reservation, c.SmartCharging, c.RemoteTrigger = p, p, p, p, p
	}
}

// TestEveryProfileCallFromTheCSMS exercises the typed calls for the profiles
// beyond Core, from the Central System to a station.
func TestEveryProfileCallFromTheCSMS(t *testing.T) {
	station := &profileStation{}
	srv, _, url := startCSMS(t, nil)
	startStation(t, url, "CS1", withProfiles(station))
	cp := lookup(t, srv, "CS1")
	ctx := context.Background()
	now := types.Now()
	profile := types.ChargingProfile{ChargingProfileID: 5, ChargingProfilePurpose: types.ChargingProfilePurposeTypeTxDefaultProfile,
		ChargingProfileKind: types.ChargingProfileKindTypeAbsolute, ChargingSchedule: types.ChargingSchedule{
			ChargingRateUnit: types.ChargingRateUnitTypeA, ChargingSchedulePeriod: []types.ChargingSchedulePeriod{{Limit: 10}}}}

	if r, err := cp.GetDiagnostics(ctx, firmware.GetDiagnosticsRequest{Location: "ftp://h/d"}); err != nil || r.FileName != "diag.zip" {
		t.Errorf("GetDiagnostics: %v %v", r, err)
	}
	if _, err := cp.UpdateFirmware(ctx, firmware.UpdateFirmwareRequest{Location: "https://h/f.bin", RetrieveDate: now}); err != nil {
		t.Errorf("UpdateFirmware: %v", err)
	}
	if r, err := cp.GetLocalListVersion(ctx, localauth.GetLocalListVersionRequest{}); err != nil || r.ListVersion != 239 {
		t.Errorf("GetLocalListVersion: %v %v", r, err)
	}
	if r, err := cp.SendLocalList(ctx, localauth.SendLocalListRequest{ListVersion: 3, UpdateType: types.UpdateTypeFull,
		LocalAuthorizationList: []types.AuthorizationData{{IDTag: "A", IDTagInfo: &types.IDTagInfo{Status: types.AuthorizationStatusAccepted}}}}); err != nil || r.Status != types.UpdateStatusAccepted {
		t.Errorf("SendLocalList: %v %v", r, err)
	}
	if r, err := cp.ReserveNow(ctx, reservation.ReserveNowRequest{ConnectorID: 1, ExpiryDate: now, IDTag: "A", ReservationID: 7}); err != nil || r.Status != types.ReservationStatusOccupied {
		t.Errorf("ReserveNow: %v %v", r, err)
	}
	if r, err := cp.CancelReservation(ctx, reservation.CancelReservationRequest{ReservationID: 7}); err != nil || r.Status != types.CancelReservationStatusAccepted {
		t.Errorf("CancelReservation: %v %v", r, err)
	}
	if r, err := cp.SetChargingProfile(ctx, smartcharging.SetChargingProfileRequest{ConnectorID: 1, CSChargingProfiles: profile}); err != nil || r.Status != types.ChargingProfileStatusAccepted {
		t.Errorf("SetChargingProfile: %v %v", r, err)
	}
	if r, err := cp.ClearChargingProfile(ctx, smartcharging.ClearChargingProfileRequest{}); err != nil || r.Status != types.ClearChargingProfileStatusUnknown {
		t.Errorf("ClearChargingProfile: %v %v", r, err)
	}
	if r, err := cp.GetCompositeSchedule(ctx, smartcharging.GetCompositeScheduleRequest{ConnectorID: 1, Duration: 600}); err != nil || r.Status != types.GetCompositeScheduleStatusRejected {
		t.Errorf("GetCompositeSchedule: %v %v", r, err)
	}
	if r, err := cp.TriggerMessage(ctx, remotetrigger.TriggerMessageRequest{RequestedMessage: types.MessageTriggerHeartbeat}); err != nil || r.Status != types.TriggerMessageStatusAccepted {
		t.Errorf("TriggerMessage: %v %v", r, err)
	}

	want := "[GetDiagnostics:ftp://h/d UpdateFirmware:https://h/f.bin GetLocalListVersion SendLocalList:3:Full:1 ReserveNow:7:A " +
		"CancelReservation:7 SetChargingProfile:1:5 ClearChargingProfile GetCompositeSchedule:1:600 TriggerMessage:Heartbeat]"
	station.mu.Lock()
	defer station.mu.Unlock()
	if fmt.Sprint(station.seen) != want {
		t.Errorf("station saw\n%v\nwant\n%s", station.seen, want)
	}
	// Invalid requests are stopped before sending.
	if _, err := cp.TriggerMessage(ctx, remotetrigger.TriggerMessageRequest{}); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("got %v", err)
	}
}

// TestProfilesWithoutAHandlerAreNotSupported: a station that sets none of the
// profile handlers (or embeds the Unimplemented ones) answers NotSupported.
func TestProfilesWithoutAHandlerAreNotSupported(t *testing.T) {
	srv, _, url := startCSMS(t, nil)
	startStation(t, url, "CS1", nil) // no profile handlers
	startStation(t, url, "CS2", func(c *chargepoint.Config) {
		c.Firmware, c.LocalAuth, c.Reservation, c.SmartCharging, c.RemoteTrigger =
			chargepoint.UnimplementedFirmwareHandler{}, chargepoint.UnimplementedLocalAuthHandler{},
			chargepoint.UnimplementedReservationHandler{}, chargepoint.UnimplementedSmartChargingHandler{},
			chargepoint.UnimplementedRemoteTriggerHandler{}
	})
	ctx := context.Background()
	now := types.Now()
	profile := types.ChargingProfile{ChargingProfileID: 1, ChargingProfilePurpose: types.ChargingProfilePurposeTypeTxProfile,
		ChargingProfileKind: types.ChargingProfileKindTypeAbsolute, ChargingSchedule: types.ChargingSchedule{
			ChargingRateUnit: types.ChargingRateUnitTypeA, ChargingSchedulePeriod: []types.ChargingSchedulePeriod{{Limit: 10}}}}
	for _, id := range []string{"CS1", "CS2"} {
		cp := lookup(t, srv, id)
		for name, f := range map[string]func() error{
			"GetDiagnostics": func() error {
				_, e := cp.GetDiagnostics(ctx, firmware.GetDiagnosticsRequest{Location: "ftp://h/d"})
				return e
			},
			"UpdateFirmware": func() error {
				_, e := cp.UpdateFirmware(ctx, firmware.UpdateFirmwareRequest{Location: "ftp://h/f", RetrieveDate: now})
				return e
			},
			"GetLocalListVersion": func() error { _, e := cp.GetLocalListVersion(ctx, localauth.GetLocalListVersionRequest{}); return e },
			"SendLocalList": func() error {
				_, e := cp.SendLocalList(ctx, localauth.SendLocalListRequest{ListVersion: 1, UpdateType: types.UpdateTypeFull})
				return e
			},
			"ReserveNow": func() error {
				_, e := cp.ReserveNow(ctx, reservation.ReserveNowRequest{ExpiryDate: now, IDTag: "A", ReservationID: 1})
				return e
			},
			"CancelReservation": func() error { _, e := cp.CancelReservation(ctx, reservation.CancelReservationRequest{}); return e },
			"SetChargingProfile": func() error {
				_, e := cp.SetChargingProfile(ctx, smartcharging.SetChargingProfileRequest{ConnectorID: 1, CSChargingProfiles: profile})
				return e
			},
			"ClearChargingProfile": func() error {
				_, e := cp.ClearChargingProfile(ctx, smartcharging.ClearChargingProfileRequest{})
				return e
			},
			"GetCompositeSchedule": func() error {
				_, e := cp.GetCompositeSchedule(ctx, smartcharging.GetCompositeScheduleRequest{ConnectorID: 1, Duration: 60})
				return e
			},
			"TriggerMessage": func() error {
				_, e := cp.TriggerMessage(ctx, remotetrigger.TriggerMessageRequest{RequestedMessage: types.MessageTriggerHeartbeat})
				return e
			},
		} {
			if ce := callError(t, f()); ce.Code != ocpp.NotSupported {
				t.Errorf("%s / %s: %s", id, name, ce.Code)
			}
		}
	}
}

// firmwareHandler records the notifications a station sends.
type firmwareHandler struct {
	csms.UnimplementedFirmwareHandler
	diag chan types.DiagnosticsStatus
	fw   chan types.FirmwareStatus
}

func (h firmwareHandler) DiagnosticsStatusNotification(_ context.Context, cp *csms.ChargePoint, r *firmware.DiagnosticsStatusNotificationRequest) (*firmware.DiagnosticsStatusNotificationConfirmation, error) {
	h.diag <- r.Status
	return &firmware.DiagnosticsStatusNotificationConfirmation{}, nil
}

func (h firmwareHandler) FirmwareStatusNotification(_ context.Context, cp *csms.ChargePoint, r *firmware.FirmwareStatusNotificationRequest) (*firmware.FirmwareStatusNotificationConfirmation, error) {
	h.fw <- r.Status
	return &firmware.FirmwareStatusNotificationConfirmation{}, nil
}

func TestFirmwareNotificationsReachTheHandler(t *testing.T) {
	h := firmwareHandler{diag: make(chan types.DiagnosticsStatus, 1), fw: make(chan types.FirmwareStatus, 1)}
	_, _, url := startCSMS(t, func(c *csms.Config) { c.Firmware = h })
	cp := startStation(t, url, "CS1", nil)
	ctx := context.Background()
	if _, err := cp.DiagnosticsStatusNotification(ctx, firmware.DiagnosticsStatusNotificationRequest{Status: types.DiagnosticsStatusUploading}); err != nil {
		t.Fatal(err)
	}
	if _, err := cp.FirmwareStatusNotification(ctx, firmware.FirmwareStatusNotificationRequest{Status: types.FirmwareStatusInstalled}); err != nil {
		t.Fatal(err)
	}
	if got := recv(t, h.diag); got != types.DiagnosticsStatusUploading {
		t.Errorf("diagnostics status %q", got)
	}
	if got := recv(t, h.fw); got != types.FirmwareStatusInstalled {
		t.Errorf("firmware status %q", got)
	}
	if _, err := cp.FirmwareStatusNotification(ctx, firmware.FirmwareStatusNotificationRequest{}); !errors.Is(err, ocpp.ErrValidation) {
		t.Errorf("invalid request: %v", err)
	}
}

func TestFirmwareNotificationsNotSupportedWithoutAHandler(t *testing.T) {
	for name, fw := range map[string]csms.FirmwareHandler{"nil handler": nil, "unimplemented embed": csms.UnimplementedFirmwareHandler{}} {
		_, _, url := startCSMS(t, func(c *csms.Config) { c.Firmware = fw })
		cp := startStation(t, url, "CS1", nil)
		_, err := cp.DiagnosticsStatusNotification(context.Background(), firmware.DiagnosticsStatusNotificationRequest{Status: types.DiagnosticsStatusIdle})
		if ce := callError(t, err); ce.Code != ocpp.NotSupported {
			t.Errorf("%s diagnostics: %s", name, ce.Code)
		}
		_, err = cp.FirmwareStatusNotification(context.Background(), firmware.FirmwareStatusNotificationRequest{Status: types.FirmwareStatusIdle})
		if ce := callError(t, err); ce.Code != ocpp.NotSupported {
			t.Errorf("%s firmware: %s", name, ce.Code)
		}
	}
}

func TestCSMSRejectsStationOnlyActionsFromAStation(t *testing.T) {
	_, _, url := startCSMS(t, nil)
	c := rawStation(t, url, "RAW2")
	for i, frame := range []string{
		`[2,"d1","UpdateFirmware",{"location":"ftp://h/f","retrieveDate":"2026-01-02T03:04:05Z"}]`,
		`[2,"d2","TriggerMessage",{"requestedMessage":"Heartbeat"}]`,
		`[2,"d3","SendLocalList",{"listVersion":1,"updateType":"Full"}]`,
	} {
		if ce := exchange(t, c, frame); ce.Code != ocpp.NotSupported || ce.UniqueID != fmt.Sprintf("d%d", i+1) {
			t.Errorf("%s: %+v", frame, ce)
		}
	}
	if ce := exchange(t, c, `[2,"d4","UpdateFirmware",{"location":"not a uri","retrieveDate":"2026-01-02T03:04:05Z"}]`); ce.Code != ocpp.PropertyConstraintViolation {
		t.Errorf("bad URI: %+v", ce)
	}
}
