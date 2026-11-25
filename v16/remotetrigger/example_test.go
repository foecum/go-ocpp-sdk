package remotetrigger_test

import (
	"fmt"

	"github.com/foecum/go-ocpp-sdk/v16/remotetrigger"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

func ExampleTriggerMessageRequest_Validate() {
	zero := 0
	req := remotetrigger.TriggerMessageRequest{RequestedMessage: types.MessageTriggerStatusNotification, ConnectorID: &zero}
	fmt.Println(req.Validate())
	// Output: ocpp: invalid field "connectorId": must be at least 1, got 0
}
