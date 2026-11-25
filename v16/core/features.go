package core

import ocpp "github.com/foecum/go-ocpp-sdk"

// Features returns the feature (action plus request and response types) of
// every Core profile message, ready to register in an ocppj.Registry.
func Features() []ocpp.Feature {
	pairs := []struct {
		req  ocpp.Request
		resp ocpp.Response
	}{
		{AuthorizeRequest{}, AuthorizeConfirmation{}},
		{BootNotificationRequest{}, BootNotificationConfirmation{}},
		{ChangeAvailabilityRequest{}, ChangeAvailabilityConfirmation{}},
		{ChangeConfigurationRequest{}, ChangeConfigurationConfirmation{}},
		{ClearCacheRequest{}, ClearCacheConfirmation{}},
		{DataTransferRequest{}, DataTransferConfirmation{}},
		{GetConfigurationRequest{}, GetConfigurationConfirmation{}},
		{HeartbeatRequest{}, HeartbeatConfirmation{}},
		{MeterValuesRequest{}, MeterValuesConfirmation{}},
		{RemoteStartTransactionRequest{}, RemoteStartTransactionConfirmation{}},
		{RemoteStopTransactionRequest{}, RemoteStopTransactionConfirmation{}},
		{ResetRequest{}, ResetConfirmation{}},
		{StartTransactionRequest{}, StartTransactionConfirmation{}},
		{StatusNotificationRequest{}, StatusNotificationConfirmation{}},
		{StopTransactionRequest{}, StopTransactionConfirmation{}},
		{UnlockConnectorRequest{}, UnlockConnectorConfirmation{}},
	}
	features := make([]ocpp.Feature, 0, len(pairs))
	for _, p := range pairs {
		f, err := ocpp.NewFeature(p.req, p.resp)
		if err != nil {
			// The table above is fixed; a mismatch is a programming error caught by tests.
			panic("v16/core: " + err.Error())
		}
		features = append(features, f)
	}
	return features
}
