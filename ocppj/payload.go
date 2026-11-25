package ocppj

import (
	"encoding/json"
	"fmt"
)

func marshalPayload(v any) (json.RawMessage, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("ocppj: marshal payload: %w", err)
	}
	return b, nil
}
