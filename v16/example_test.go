package v16_test

import (
	"fmt"

	"github.com/foecum/go-ocpp-sdk/v16"
)

// NewRegistry returns a registry holding every implemented OCPP 1.6 message.
func ExampleNewRegistry() {
	reg, err := v16.NewRegistry()
	if err != nil {
		panic(err)
	}
	f, _ := reg.Lookup("BootNotification")
	fmt.Println(reg.Version(), reg.Len(), f.RequestType().Name())
	// Output: ocpp1.6 28 BootNotificationRequest
}
