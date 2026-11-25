// Package featureset builds the Features of a profile from request and
// confirmation pairs.
package featureset

import ocpp "github.com/foecum/go-ocpp-sdk"

// Pair is a request and its confirmation. Resp is nil for an unconfirmed
// (SEND) message.
type Pair struct {
	Req  ocpp.Request
	Resp ocpp.Response
}

// Build returns one Feature per pair. The pairs are fixed tables in each
// profile package, so a mismatch is a programming error: Build panics, and
// every profile's tests call Features to catch it.
func Build(pairs ...Pair) []ocpp.Feature {
	features := make([]ocpp.Feature, 0, len(pairs))
	for _, p := range pairs {
		var f ocpp.Feature
		var err error
		if p.Resp == nil {
			f, err = ocpp.NewSendFeature(p.Req)
		} else {
			f, err = ocpp.NewFeature(p.Req, p.Resp)
		}
		if err != nil {
			panic("featureset: " + err.Error())
		}
		features = append(features, f)
	}
	return features
}
