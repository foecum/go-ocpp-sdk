package core_test

import (
	"encoding/json"
	"fmt"

	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// Messages marshal to exactly the JSON the specification defines. Optional
// fields left empty are omitted.
func ExampleBootNotificationRequest() {
	b, _ := json.Marshal(core.BootNotificationRequest{ChargePointVendor: "Acme", ChargePointModel: "X1"})
	fmt.Println(string(b))
	// Output: {"chargePointVendor":"Acme","chargePointModel":"X1"}
}

// Validate checks required fields, lengths, ranges and enumerations. The
// endpoints call it for you before sending and after receiving.
func ExampleStartTransactionRequest_Validate() {
	req := core.StartTransactionRequest{ConnectorID: 0, IDTag: "04A1B2C3", Timestamp: types.Now()}
	fmt.Println(req.Validate())
	// Output: ocpp: invalid field "connectorId": must be at least 1, got 0
}

func ExampleStatusNotificationRequest_Validate() {
	req := core.StatusNotificationRequest{ConnectorID: 1, ErrorCode: types.ChargePointErrorCodeNoError, Status: "Sleeping"}
	fmt.Println(req.Validate())
	// Output: ocpp: invalid field "status": unknown value "Sleeping"
}
