package firmware_test

import (
	"fmt"

	"github.com/foecum/go-ocpp-sdk/v16/firmware"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// The location must be an absolute URI.
func ExampleUpdateFirmwareRequest_Validate() {
	req := firmware.UpdateFirmwareRequest{Location: "/firmware/x1.bin", RetrieveDate: types.Now()}
	fmt.Println(req.Validate())
	// Output: ocpp: invalid field "location": must be an absolute URI
}
