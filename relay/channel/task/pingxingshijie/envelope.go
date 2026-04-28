package pingxingshijie

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/common"

	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
)

// APIEnvelope is the common PingXingShiJie response wrapper: {"code":0,"msg":"ok","data":...}
// Some routes (e.g. /v2/chat/completions) use "message" instead of "msg" for human-readable text.
type APIEnvelope struct {
	Code    int             `json:"code"`
	Msg     string          `json:"msg"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (e *APIEnvelope) text() string {
	if e.Msg != "" {
		return e.Msg
	}
	return e.Message
}

// HTTPStatusForPingXingBizCode maps upstream business code (HTTP 200 body) to HTTP status for clients.
func HTTPStatusForPingXingBizCode(code int) int {
	switch code {
	case 401:
		return http.StatusUnauthorized
	case 403:
		return http.StatusForbidden
	case 429:
		return http.StatusTooManyRequests
	default:
		return http.StatusBadRequest
	}
}

// NormalizePingXingOpenAIShapedSyncBody unwraps PingXing envelope when top-level "code" exists.
// Plain OpenAI JSON (no "code") is returned unchanged. On business failure returns bizCode != 0 and bizMsg.
func NormalizePingXingOpenAIShapedSyncBody(body []byte) (inner []byte, bizCode int, bizMsg string) {
	if len(body) == 0 || !gjson.GetBytes(body, "code").Exists() {
		return body, 0, ""
	}
	var env APIEnvelope
	if err := common.Unmarshal(body, &env); err != nil {
		return body, 0, ""
	}
	if env.Code != 0 {
		return nil, env.Code, env.text()
	}
	if len(env.Data) > 0 && string(env.Data) != "null" {
		return env.Data, 0, ""
	}
	return body, 0, ""
}

// UnmarshalEnvelope parses the outer wrapper and returns the raw inner data JSON.
func UnmarshalEnvelope(body []byte) (data json.RawMessage, err error) {
	var env APIEnvelope
	if err := common.Unmarshal(body, &env); err != nil {
		return nil, errors.Wrap(err, "unmarshal envelope failed")
	}
	if env.Code != 0 {
		return nil, fmt.Errorf("upstream error code=%d msg=%s", env.Code, env.text())
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
