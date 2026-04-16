package pingxingshijie

import (
	"encoding/json"
	"fmt"

	"github.com/QuantumNous/new-api/common"

	"github.com/pkg/errors"
)

// APIEnvelope is the common PingXingShiJie response wrapper: {"code":0,"msg":"ok","data":...}
type APIEnvelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// UnmarshalEnvelope parses the outer wrapper and returns the raw inner data JSON.
func UnmarshalEnvelope(body []byte) (data json.RawMessage, err error) {
	var env APIEnvelope
	if err := common.Unmarshal(body, &env); err != nil {
		return nil, errors.Wrap(err, "unmarshal envelope failed")
	}
	if env.Code != 0 {
		return nil, fmt.Errorf("upstream error code=%d msg=%s", env.Code, env.Msg)
	}
	return env.Data, nil
}

// UnmarshalEnvelopeData unmarshals envelope and decodes data into v.
func UnmarshalEnvelopeData(body []byte, v any) error {
	raw, err := UnmarshalEnvelope(body)
	if err != nil {
		return err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return fmt.Errorf("empty envelope data")
	}
	return common.Unmarshal(raw, v)
}

// UnmarshalDataOrEnvelope unmarshals PingXingShiJie envelope data into v, or raw body if not wrapped.
func UnmarshalDataOrEnvelope(body []byte, v any) error {
	if err := UnmarshalEnvelopeData(body, v); err == nil {
		return nil
	}
	return common.Unmarshal(body, v)
}
