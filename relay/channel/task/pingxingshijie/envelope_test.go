package pingxingshijie

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestNormalizePingXingOpenAIShapedSyncBody_plainOpenAI(t *testing.T) {
	raw := []byte(`{"id":"x","choices":[]}`)
	inner, code, msg := NormalizePingXingOpenAIShapedSyncBody(raw)
	if code != 0 || msg != "" {
		t.Fatalf("unexpected biz err: code=%d msg=%q", code, msg)
	}
	if string(inner) != string(raw) {
		t.Fatalf("expected unchanged body, got %s", string(inner))
	}
}

func TestNormalizePingXingOpenAIShapedSyncBody_bizErrorMessageField(t *testing.T) {
	raw := []byte(`{"code":401,"message":"无效的令牌"}`)
	inner, code, msg := NormalizePingXingOpenAIShapedSyncBody(raw)
	if inner != nil {
		t.Fatalf("expected nil inner, got %s", string(inner))
	}
	if code != 401 || msg != "无效的令牌" {
		t.Fatalf("got code=%d msg=%q", code, msg)
	}
}

func TestNormalizePingXingOpenAIShapedSyncBody_unwrapData(t *testing.T) {
	innerObj := map[string]any{"choices": []any{}}
	innerBytes, err := common.Marshal(innerObj)
	if err != nil {
		t.Fatal(err)
	}
	// Marshal nested object as JSON object for data, not string
	outer2, err := common.Marshal(struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}{Code: 0, Msg: "ok", Data: innerBytes})
	if err != nil {
		t.Fatal(err)
	}
	inner, code, msg := NormalizePingXingOpenAIShapedSyncBody(outer2)
	if code != 0 || msg != "" {
		t.Fatalf("unexpected biz err: code=%d msg=%q", code, msg)
	}
	if string(inner) != string(innerBytes) {
		t.Fatalf("unwrap mismatch: got %s want %s", string(inner), string(innerBytes))
	}
}
