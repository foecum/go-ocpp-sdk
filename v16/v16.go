// Package v16 is the entry point for OCPP 1.6 (JSON). Its NewRegistry returns an
// ocppj.Registry with every implemented 1.6 message registered, ready to hand to
// an ocppj.Endpoint.
//
//	reg, err := v16.NewRegistry()
//	if err != nil { /* ... */ }
//	ep, err := ocppj.NewEndpoint(ocppj.Config{Registry: reg, Send: conn.Write})
//
// The messages themselves live in subpackages by profile (core, firmware,
// localauth, remotetrigger, reservation and smartcharging); shared types and
// enumerations live in package types.
package v16

import (
	"fmt"

	ocpp "github.com/foecum/go-ocpp-sdk"
	"github.com/foecum/go-ocpp-sdk/ocppj"
	"github.com/foecum/go-ocpp-sdk/v16/core"
	"github.com/foecum/go-ocpp-sdk/v16/firmware"
	"github.com/foecum/go-ocpp-sdk/v16/localauth"
	"github.com/foecum/go-ocpp-sdk/v16/remotetrigger"
	"github.com/foecum/go-ocpp-sdk/v16/reservation"
	"github.com/foecum/go-ocpp-sdk/v16/smartcharging"
)

// Features returns the features of every implemented OCPP 1.6 message.
func Features() []ocpp.Feature {
	var all []ocpp.Feature
	for _, profile := range [][]ocpp.Feature{
		core.Features(),
		firmware.Features(),
		localauth.Features(),
		remotetrigger.Features(),
		reservation.Features(),
		smartcharging.Features(),
	} {
		all = append(all, profile...)
	}
	return all
}

// NewRegistry returns a registry for ocpp.V16 containing every implemented
// message.
func NewRegistry() (*ocppj.Registry, error) {
	return registryOf(Features())
}

func registryOf(features []ocpp.Feature) (*ocppj.Registry, error) {
	reg := ocppj.NewRegistry(ocpp.V16)
	for _, f := range features {
		if err := reg.Register(f); err != nil {
			return nil, fmt.Errorf("v16: %w", err)
		}
	}
	return reg, nil
}
