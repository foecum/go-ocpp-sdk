package types

import (
	"encoding/json"
	"fmt"

	"github.com/foecum/go-ocpp-sdk/internal/check"
)

// CustomData is the vendor extension every OCPP 2.0.1 object can carry. Besides
// the required VendorID it may hold any other JSON properties, which are kept in
// Extra and written back unchanged.
type CustomData struct {
	// VendorID identifies the vendor the extension belongs to (at most 255 characters).
	VendorID string `json:"vendorId" ocpp:"required"`
	// Extra holds the other properties, as raw JSON values by name.
	Extra map[string]json.RawMessage `json:"-"`
}

// MarshalJSON implements json.Marshaler. Extra properties are written along
// with vendorId; an extra property named "vendorId" is ignored.
func (c CustomData) MarshalJSON() ([]byte, error) {
	m := make(map[string]json.RawMessage, len(c.Extra)+1)
	for k, v := range c.Extra {
		m[k] = v
	}
	id, _ := json.Marshal(c.VendorID) // a string always marshals
	m["vendorId"] = id
	return json.Marshal(m)
}

// UnmarshalJSON implements json.Unmarshaler.
func (c *CustomData) UnmarshalJSON(data []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("customData must be an object: %w", err)
	}
	*c = CustomData{}
	if raw, ok := m["vendorId"]; ok {
		if err := json.Unmarshal(raw, &c.VendorID); err != nil {
			return fmt.Errorf("customData.vendorId: %w", err)
		}
		delete(m, "vendorId")
	}
	if len(m) > 0 {
		c.Extra = m
	}
	return nil
}

// Validate checks the structure.
func (c CustomData) Validate() error { return check.String("vendorId", c.VendorID, 255, true) }
