package v21_test

import (
	"encoding/json"
	"fmt"

	"github.com/foecum/go-ocpp-sdk/v21"
	"github.com/foecum/go-ocpp-sdk/v21/provisioning"
	"github.com/foecum/go-ocpp-sdk/v21/types"
)

// NewRegistry returns a registry holding every OCPP 2.1 message.
func ExampleNewRegistry() {
	reg, err := v21.NewRegistry()
	if err != nil {
		panic(err)
	}
	f, _ := reg.Lookup("BootNotification")
	fmt.Println(reg.Version(), reg.Len(), f.RequestType().Name(), f.ResponseType().Name())
	// Output: ocpp2.1 91 BootNotificationRequest BootNotificationResponse
}

// Messages marshal to the JSON the specification defines, and Validate checks
// them before they are sent.
func ExampleBootNotificationRequest() {
	req := provisioning.BootNotificationRequest{
		Reason: types.BootReasonPowerUp,
		ChargingStation: types.ChargingStation{
			Model:      "SingleSocketCharger",
			VendorName: "VendorX",
		},
	}
	b, _ := json.Marshal(req)
	fmt.Println(string(b))
	fmt.Println(req.Validate())

	req.ChargingStation.Model = ""
	fmt.Println(req.Validate())
	// Output:
	// {"chargingStation":{"model":"SingleSocketCharger","vendorName":"VendorX"},"reason":"PowerUp"}
	// <nil>
	// ocpp: invalid field "chargingStation.model": required
}
