package pingxingshijie

import (
	"bytes"

	"github.com/QuantumNous/new-api/common"
)

// jsonAnyFromBytes parses JSON into a generic value for downstream metadata (full upstream payload).
func jsonAnyFromBytes(b []byte) any {
	if len(bytes.TrimSpace(b)) == 0 {
		return nil
	}
	var v any
	if err := common.Unmarshal(b, &v); err != nil {
		return nil
	}
	return v
}
