package core

import "github.com/foecum/go-ocpp-sdk/v16/types"

// ClearCacheAction is the action name of the ClearCache message.
const ClearCacheAction = "ClearCache"

// ClearCacheRequest is the ClearCache.req PDU, sent by the Central System to
// clear the Charge Point's authorisation cache. It has no fields.
type ClearCacheRequest struct{}

// Action implements ocpp.Request.
func (ClearCacheRequest) Action() string { return ClearCacheAction }

// Validate implements ocpp.Request.
func (ClearCacheRequest) Validate() error { return nil }

// ClearCacheConfirmation is the ClearCache.conf PDU.
type ClearCacheConfirmation struct {
	// Status is the outcome.
	Status types.ClearCacheStatus `json:"status" ocpp:"required"`
}

// Action implements ocpp.Response.
func (ClearCacheConfirmation) Action() string { return ClearCacheAction }

// Validate implements ocpp.Response.
func (c ClearCacheConfirmation) Validate() error { return types.CheckEnum("status", c.Status, true) }
