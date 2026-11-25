package types

import (
	"fmt"

	ocpp "github.com/foecum/go-ocpp-sdk"
)

// IDTagInfo says whether an identifier (idTag) is authorised. It is returned in
// Authorize, StartTransaction and StopTransaction confirmations.
type IDTagInfo struct {
	// ExpiryDate is when the idTag should be removed from the authorisation
	// cache. If absent the status has no end date.
	ExpiryDate *DateTime `json:"expiryDate,omitempty"`
	// ParentIDTag is the parent identifier, if any (at most 20 characters).
	ParentIDTag string `json:"parentIdTag,omitempty"`
	// Status says whether the idTag has been accepted.
	Status AuthorizationStatus `json:"status" ocpp:"required"`
}

// Validate checks the structure.
func (i IDTagInfo) Validate() error {
	return First(
		CheckString("parentIdTag", i.ParentIDTag, CiString20, false),
		CheckEnum("status", i.Status, true),
	)
}

// SampledValue is a single measured value in a MeterValue.
//
// When fields are absent the specification's defaults apply: Context is
// "Sample.Periodic", Format is "Raw", Measurand is
// "Energy.Active.Import.Register", Location is "Outlet", and Unit is "Wh" for
// the default measurand.
type SampledValue struct {
	// Value is a decimal number ("Raw") or signed data as a string.
	Value string `json:"value" ocpp:"required"`
	// Context is why the value was taken.
	Context ReadingContext `json:"context,omitempty"`
	// Format is Raw or SignedData.
	Format ValueFormat `json:"format,omitempty"`
	// Measurand is the quantity measured.
	Measurand Measurand `json:"measurand,omitempty"`
	// Phase says which phase the value applies to; absent means overall.
	Phase Phase `json:"phase,omitempty"`
	// Location is where the value was measured.
	Location Location `json:"location,omitempty"`
	// Unit is the unit of the value.
	Unit UnitOfMeasure `json:"unit,omitempty"`
}

// Validate checks the structure.
func (s SampledValue) Validate() error {
	return First(
		CheckString("value", s.Value, Unbounded, true),
		CheckEnum("context", s.Context, false),
		CheckEnum("format", s.Format, false),
		CheckEnum("measurand", s.Measurand, false),
		CheckEnum("phase", s.Phase, false),
		CheckEnum("location", s.Location, false),
		CheckEnum("unit", s.Unit, false),
	)
}

// MeterValue is a set of sampled values taken at the same moment, in
// MeterValues and StopTransaction.
type MeterValue struct {
	// Timestamp is when the values were sampled.
	Timestamp DateTime `json:"timestamp" ocpp:"required"`
	// SampledValue holds one or more measured values.
	SampledValue []SampledValue `json:"sampledValue" ocpp:"required"`
}

// Validate checks the structure.
func (m MeterValue) Validate() error {
	if err := CheckDateTime("timestamp", m.Timestamp); err != nil {
		return err
	}
	if len(m.SampledValue) == 0 {
		return ocpp.Required("sampledValue")
	}
	for i, sv := range m.SampledValue {
		if err := sv.Validate(); err != nil {
			return Prefix(fmt.Sprintf("sampledValue[%d]", i), err)
		}
	}
	return nil
}

// ValidateMeterValues checks a list of meter values.
func ValidateMeterValues(field string, values []MeterValue) error {
	for i, mv := range values {
		if err := mv.Validate(); err != nil {
			return Prefix(fmt.Sprintf("%s[%d]", field, i), err)
		}
	}
	return nil
}

// KeyValue is one configuration key in a GetConfiguration confirmation.
type KeyValue struct {
	// Key is the name of the configuration key (at most 50 characters).
	Key string `json:"key" ocpp:"required"`
	// Readonly is false if ChangeConfiguration may set the value.
	Readonly bool `json:"readonly" ocpp:"required"`
	// Value is the current value (at most 500 characters). It may be absent if
	// the key is known but not set.
	Value string `json:"value,omitempty"`
}

// Validate checks the structure.
func (k KeyValue) Validate() error {
	return First(
		CheckString("key", k.Key, CiString50, true),
		CheckString("value", k.Value, CiString500, false),
	)
}

// ChargingSchedulePeriod is one period of a charging schedule.
type ChargingSchedulePeriod struct {
	// StartPeriod is the start of the period in seconds from the start of the
	// schedule. It also ends the previous period.
	StartPeriod int `json:"startPeriod" ocpp:"required"`
	// Limit is the charging rate limit, in the schedule's unit, with at most
	// one fractional digit.
	Limit float64 `json:"limit" ocpp:"required"`
	// NumberPhases is the number of phases usable for charging. If absent, 3 is
	// assumed.
	NumberPhases *int `json:"numberPhases,omitempty"`
}

// Validate checks the structure.
func (p ChargingSchedulePeriod) Validate() error {
	return First(
		CheckMin("startPeriod", p.StartPeriod, 0),
		CheckOneDecimal("limit", p.Limit),
		CheckMinPtr("numberPhases", p.NumberPhases, 1),
	)
}

// ChargingSchedule describes charging limits over time.
type ChargingSchedule struct {
	// Duration of the schedule in seconds. If absent the last period continues
	// indefinitely, or until the end of the transaction.
	Duration *int `json:"duration,omitempty"`
	// StartSchedule is the starting point of an absolute schedule. If absent
	// the schedule is relative to the start of charging.
	StartSchedule *DateTime `json:"startSchedule,omitempty"`
	// ChargingRateUnit is the unit of the limits: "A" or "W".
	ChargingRateUnit ChargingRateUnitType `json:"chargingRateUnit" ocpp:"required"`
	// ChargingSchedulePeriod lists the periods. The first must start at 0.
	ChargingSchedulePeriod []ChargingSchedulePeriod `json:"chargingSchedulePeriod" ocpp:"required"`
	// MinChargingRate is the minimum rate the vehicle supports, with at most
	// one fractional digit.
	MinChargingRate *float64 `json:"minChargingRate,omitempty"`
}

// Validate checks the structure.
func (s ChargingSchedule) Validate() error {
	if err := First(
		CheckMinPtr("duration", s.Duration, 0),
		CheckEnum("chargingRateUnit", s.ChargingRateUnit, true),
	); err != nil {
		return err
	}
	if len(s.ChargingSchedulePeriod) == 0 {
		return ocpp.Required("chargingSchedulePeriod")
	}
	if s.ChargingSchedulePeriod[0].StartPeriod != 0 {
		return ocpp.Invalid("chargingSchedulePeriod[0].startPeriod", "the first period must start at 0")
	}
	for i, p := range s.ChargingSchedulePeriod {
		if err := p.Validate(); err != nil {
			return Prefix(fmt.Sprintf("chargingSchedulePeriod[%d]", i), err)
		}
	}
	if s.MinChargingRate != nil {
		return CheckOneDecimal("minChargingRate", *s.MinChargingRate)
	}
	return nil
}

// ChargingProfile is a charging schedule with the rules for when it applies.
type ChargingProfile struct {
	// ChargingProfileID uniquely identifies the profile.
	ChargingProfileID int `json:"chargingProfileId" ocpp:"required"`
	// TransactionID ties a TxProfile to one transaction.
	TransactionID *int `json:"transactionId,omitempty"`
	// StackLevel orders profiles: higher values take precedence. Lowest is 0.
	StackLevel int `json:"stackLevel" ocpp:"required"`
	// ChargingProfilePurpose says what the profile is for.
	ChargingProfilePurpose ChargingProfilePurposeType `json:"chargingProfilePurpose" ocpp:"required"`
	// ChargingProfileKind says whether the schedule is absolute, recurring or relative.
	ChargingProfileKind ChargingProfileKindType `json:"chargingProfileKind" ocpp:"required"`
	// RecurrencyKind says when a recurring profile repeats.
	RecurrencyKind RecurrencyKindType `json:"recurrencyKind,omitempty"`
	// ValidFrom is when the profile becomes valid; absent means immediately.
	ValidFrom *DateTime `json:"validFrom,omitempty"`
	// ValidTo is when the profile stops being valid; absent means until replaced.
	ValidTo *DateTime `json:"validTo,omitempty"`
	// ChargingSchedule holds the limits.
	ChargingSchedule ChargingSchedule `json:"chargingSchedule" ocpp:"required"`
}

// Validate checks the structure.
func (p ChargingProfile) Validate() error {
	if err := First(
		CheckMin("stackLevel", p.StackLevel, 0),
		CheckEnum("chargingProfilePurpose", p.ChargingProfilePurpose, true),
		CheckEnum("chargingProfileKind", p.ChargingProfileKind, true),
		CheckEnum("recurrencyKind", p.RecurrencyKind, false),
	); err != nil {
		return err
	}
	if err := p.ChargingSchedule.Validate(); err != nil {
		return Prefix("chargingSchedule", err)
	}
	return nil
}

// AuthorizationData is one entry of a local authorisation list update, in
// SendLocalList.
type AuthorizationData struct {
	// IDTag is the identifier the entry applies to (at most 20 characters).
	IDTag string `json:"idTag" ocpp:"required"`
	// IDTagInfo is the authorisation information. It is required in a Full
	// update. In a Differential update, a present value adds or updates the
	// entry and an absent one deletes it.
	IDTagInfo *IDTagInfo `json:"idTagInfo,omitempty"`
}

// Validate checks the structure.
func (a AuthorizationData) Validate() error {
	if err := CheckString("idTag", a.IDTag, CiString20, true); err != nil {
		return err
	}
	if a.IDTagInfo != nil {
		return Prefix("idTagInfo", a.IDTagInfo.Validate())
	}
	return nil
}
