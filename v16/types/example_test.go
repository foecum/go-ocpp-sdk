package types_test

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// DateTime is sent in UTC and accepts the variants charge points send.
func ExampleDateTime() {
	t := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	b, _ := json.Marshal(types.NewDateTime(t))
	fmt.Println(string(b))

	b, _ = json.Marshal(types.NewDateTime(t.Add(123 * time.Millisecond)))
	fmt.Println(string(b))

	var d types.DateTime
	_ = json.Unmarshal([]byte(`"2026-01-02T04:04:05+01:00"`), &d) // with an offset
	fmt.Println(d.UTC().Format(time.RFC3339))
	_ = json.Unmarshal([]byte(`"2026-01-02T03:04:05"`), &d) // no zone: UTC
	fmt.Println(d.UTC().Format(time.RFC3339))
	// Output:
	// "2026-01-02T03:04:05Z"
	// "2026-01-02T03:04:05.123Z"
	// 2026-01-02T03:04:05Z
	// 2026-01-02T03:04:05Z
}

// Every enumeration has an IsValid method.
func ExampleMeasurand_IsValid() {
	fmt.Println(types.MeasurandEnergyActiveImportRegister.IsValid(), types.Measurand("Energy.Wrong").IsValid())
	// Output: true false
}

// Both spellings of the temperature unit are accepted (OCPP-J 1.6 errata).
func ExampleUnitOfMeasure_IsValid() {
	fmt.Println(types.UnitOfMeasureCelsius.IsValid(), types.UnitOfMeasureCelcius.IsValid())
	// Output: true true
}
