package ocpp_test

import (
	"errors"
	"fmt"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/v16/core"
)

// A Feature binds an action name to its request and confirmation types. The
// SDK's message packages provide ready-made ones through their Features
// functions; NewFeature builds one for your own messages.
func ExampleNewFeature() {
	f, err := ocpp.NewFeature(core.HeartbeatRequest{}, core.HeartbeatConfirmation{})
	if err != nil {
		panic(err)
	}
	fmt.Println(f.Action(), f.RequestType().Name(), f.ResponseType().Name())
	// Output: Heartbeat HeartbeatRequest HeartbeatConfirmation
}

// A CALLERROR from the peer comes back as a *CallError. Match it with errors.Is
// to treat any CALLERROR alike, or with errors.As to read its code.
func ExampleCallError() {
	err := fmt.Errorf("reset failed: %w", &ocpp.CallError{Code: ocpp.NotSupported, Description: "Reset is not supported"})

	fmt.Println(errors.Is(err, ocpp.ErrCallError))
	var ce *ocpp.CallError
	if errors.As(err, &ce) {
		fmt.Println(ce.Code, "-", ce.Description)
	}
	// Output:
	// true
	// NotSupported - Reset is not supported
}

// OCPP 1.6 and 2.0.1 spell some CALLERROR codes differently.
func ExampleErrorCode_IsValid() {
	fmt.Println(ocpp.FormationViolation.IsValid(ocpp.V16), ocpp.FormationViolation.IsValid(ocpp.V201))
	fmt.Println(ocpp.FormatViolation.IsValid(ocpp.V16), ocpp.FormatViolation.IsValid(ocpp.V201))
	// Output:
	// true false
	// false true
}

// Validation errors name the offending field.
func ExampleValidationError() {
	err := ocpp.Required("idTag")
	fmt.Println(err)
	fmt.Println(errors.Is(err, ocpp.ErrValidation))
	var ve *ocpp.ValidationError
	if errors.As(err, &ve) {
		fmt.Println(ve.Field, ve.Missing)
	}
	// Output:
	// ocpp: invalid field "idTag": required
	// true
	// idTag true
}
