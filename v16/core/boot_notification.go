package core

import "github.com/foecum/go-ocpp-sdk/v16/types"

// BootNotificationAction is the action name of the BootNotification message.
const BootNotificationAction = "BootNotification"

// BootNotificationRequest is the BootNotification.req PDU, sent by a Charge
// Point after it starts up to register with the Central System.
type BootNotificationRequest struct {
	// ChargePointVendor identifies the vendor (at most 20 characters).
	ChargePointVendor string `json:"chargePointVendor" ocpp:"required"`
	// ChargePointModel identifies the model (at most 20 characters).
	ChargePointModel string `json:"chargePointModel" ocpp:"required"`
	// ChargePointSerialNumber is the Charge Point's serial number (at most 25 characters).
	ChargePointSerialNumber string `json:"chargePointSerialNumber,omitempty"`
	// ChargeBoxSerialNumber is the serial number of the Charge Box inside the
	// Charge Point (at most 25 characters). Deprecated by the specification.
	ChargeBoxSerialNumber string `json:"chargeBoxSerialNumber,omitempty"`
	// FirmwareVersion is the firmware version (at most 50 characters).
	FirmwareVersion string `json:"firmwareVersion,omitempty"`
	// ICCID is the ICCID of the modem's SIM card (at most 20 characters).
	ICCID string `json:"iccid,omitempty"`
	// IMSI is the IMSI of the modem's SIM card (at most 20 characters).
	IMSI string `json:"imsi,omitempty"`
	// MeterType is the type of the main electrical meter (at most 25 characters).
	MeterType string `json:"meterType,omitempty"`
	// MeterSerialNumber is the serial number of the main meter (at most 25 characters).
	MeterSerialNumber string `json:"meterSerialNumber,omitempty"`
}

// Action implements ocpp.Request.
func (BootNotificationRequest) Action() string { return BootNotificationAction }

// Validate implements ocpp.Request.
func (r BootNotificationRequest) Validate() error {
	return types.First(
		types.CheckString("chargePointVendor", r.ChargePointVendor, types.CiString20, true),
		types.CheckString("chargePointModel", r.ChargePointModel, types.CiString20, true),
		types.CheckString("chargePointSerialNumber", r.ChargePointSerialNumber, types.CiString25, false),
		types.CheckString("chargeBoxSerialNumber", r.ChargeBoxSerialNumber, types.CiString25, false),
		types.CheckString("firmwareVersion", r.FirmwareVersion, types.CiString50, false),
		types.CheckString("iccid", r.ICCID, types.CiString20, false),
		types.CheckString("imsi", r.IMSI, types.CiString20, false),
		types.CheckString("meterType", r.MeterType, types.CiString25, false),
		types.CheckString("meterSerialNumber", r.MeterSerialNumber, types.CiString25, false),
	)
}

// BootNotificationConfirmation is the BootNotification.conf PDU.
type BootNotificationConfirmation struct {
	// Status says whether the Charge Point has been registered.
	Status types.RegistrationStatus `json:"status" ocpp:"required"`
	// CurrentTime is the Central System's current time.
	CurrentTime types.DateTime `json:"currentTime" ocpp:"required"`
	// Interval is the heartbeat interval in seconds when Status is Accepted;
	// otherwise the minimum wait before the next BootNotification.
	Interval int `json:"interval" ocpp:"required"`
}

// Action implements ocpp.Response.
func (BootNotificationConfirmation) Action() string { return BootNotificationAction }

// Validate implements ocpp.Response.
func (c BootNotificationConfirmation) Validate() error {
	return types.First(
		types.CheckEnum("status", c.Status, true),
		types.CheckDateTime("currentTime", c.CurrentTime),
		types.CheckMin("interval", c.Interval, 0),
	)
}
