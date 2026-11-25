package localauth_test

import (
	"fmt"

	"github.com/foecum/go-ocpp-sdk/v16/localauth"
	"github.com/foecum/go-ocpp-sdk/v16/types"
)

// idTags in a local list must be unique, ignoring case.
func ExampleSendLocalListRequest_Validate() {
	accepted := &types.IDTagInfo{Status: types.AuthorizationStatusAccepted}
	req := localauth.SendLocalListRequest{
		ListVersion: 2,
		UpdateType:  types.UpdateTypeFull,
		LocalAuthorizationList: []types.AuthorizationData{
			{IDTag: "04A1B2C3", IDTagInfo: accepted},
			{IDTag: "04a1b2c3", IDTagInfo: accepted},
		},
	}
	fmt.Println(req.Validate())
	// Output: ocpp: invalid field "localAuthorizationList[1].idTag": duplicate idTag in the list
}
