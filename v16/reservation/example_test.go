package reservation_test

import (
	"fmt"

	"github.com/foecum/go-ocpp-sdk/v16/reservation"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

func ExampleReserveNowRequest_Validate() {
	req := reservation.ReserveNowRequest{ConnectorID: -1, ExpiryDate: types.Now(), IDTag: "04A1B2C3", ReservationID: 7}
	fmt.Println(req.Validate())
	// Output: ocpp: invalid field "connectorId": must be at least 0, got -1
}
