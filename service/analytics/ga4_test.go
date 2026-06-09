package analytics

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
)

type captureSender struct {
	requests []*http.Request
	bodies   []string
	status   int
	err      error
	done     chan struct{}
}

func (s *captureSender) Do(req *http.Request) (*http.Response, error) {
	defer func() {
		if s.done != nil {
			s.done <- struct{}{}
		}
	}()
	s.requests = append(s.requests, req)
	body, _ := io.ReadAll(req.Body)
	s.bodies = append(s.bodies, string(body))
	if s.err != nil {
		return nil, s.err
	}
	status := s.status
	if status == 0 {
		status = http.StatusNoContent
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(nil)),
	}, nil
}

func testConfig() Config {
	return Config{
		Enabled:       true,
		MeasurementID: "G-TEST",
		APISecret:     "secret",
		HashSalt:      "salt",
		Timeout:       50 * time.Millisecond,
		Endpoint:      "https://example.test/mp/collect",
	}
}

func TestParseGAClientID(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "valid ga cookie", value: "GA1.1.123456789.987654321", want: "123456789.987654321"},
		{name: "valid cookie with prefix variant", value: "GA1.2.111.222", want: "111.222"},
		{name: "malformed", value: "GA1.1.onlyone", want: ""},
		{name: "non numeric", value: "GA1.1.abc.222", want: ""},
		{name: "empty", value: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseGAClientID(tt.value); got != tt.want {
				t.Fatalf("ParseGAClientID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHashIdentifierUsesSalt(t *testing.T) {
	a := hashIdentifierWithSalt("voucher-secret", "salt-a")
	b := hashIdentifierWithSalt("voucher-secret", "salt-a")
	c := hashIdentifierWithSalt("voucher-secret", "salt-b")
	if a != b {
		t.Fatalf("same salt should be deterministic")
	}
	if a == c {
		t.Fatalf("different salts should produce different hashes")
	}
	if strings.Contains(a, "voucher-secret") {
		t.Fatalf("hash leaked raw input")
	}
}

func TestResolveGAClientIDUsesCookieWhenPresent(t *testing.T) {
	restore := ConfigureForTest(testConfig(), nil)
	defer restore()

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	ctx.Request.AddCookie(&http.Cookie{Name: "_ga", Value: "GA1.1.123.456"})

	if got := ResolveGAClientID(ctx, 42, 7); got != "123.456" {
		t.Fatalf("ResolveGAClientID() = %q, want cookie client id", got)
	}
}

func TestBuildPayloadIncludesExpectedFields(t *testing.T) {
	restore := ConfigureForTest(testConfig(), nil)
	defer restore()

	payload := buildPayload(nil, 42, 7, eventFirstAPICall, EventParams{
		"api_key_id_hash": "api-hash",
		"model_id":        "gpt-test",
		"quota_spent":     123,
	})

	if !strings.HasPrefix(payload.ClientID, "server.") {
		t.Fatalf("fallback client id = %q, want server prefix", payload.ClientID)
	}
	if payload.UserID == "" || payload.UserID == "42" {
		t.Fatalf("user id should be hashed, got %q", payload.UserID)
	}
	if len(payload.Events) != 1 || payload.Events[0].Name != eventFirstAPICall {
		t.Fatalf("unexpected events: %#v", payload.Events)
	}
	if payload.Events[0].Params["model_id"] != "gpt-test" {
		t.Fatalf("model_id missing from payload")
	}
}

func TestSendPayloadNoopsWhenDisabled(t *testing.T) {
	sender := &captureSender{}
	cfg := testConfig()
	cfg.Enabled = false
	restore := ConfigureForTest(cfg, sender)
	defer restore()

	err := sendPayload(context.Background(), cfg, ga4Payload{})
	if err != nil {
		t.Fatalf("sendPayload disabled returned error: %v", err)
	}
	if len(sender.requests) != 0 {
		t.Fatalf("disabled config sent %d requests", len(sender.requests))
	}
}

func TestSendPayloadPostsMeasurementProtocolPayload(t *testing.T) {
	sender := &captureSender{}
	cfg := testConfig()
	restore := ConfigureForTest(cfg, sender)
	defer restore()

	payload := ga4Payload{
		ClientID:           "123.456",
		UserID:             "user-hash",
		NonPersonalizedAds: true,
		Events: []ga4Event{
			{
				Name: eventVoucherRedeemSuccess,
				Params: EventParams{
					"voucher_code_hash":  "voucher-hash",
					"voucher_amount_usd": 10.5,
					"voucher_source":     "lizh_ai",
					"redeem_result":      "success",
				},
			},
		},
	}

	err := sendPayload(context.Background(), cfg, payload)
	if err != nil {
		t.Fatalf("sendPayload returned error: %v", err)
	}
	if len(sender.requests) != 1 {
		t.Fatalf("sent %d requests, want 1", len(sender.requests))
	}
	req := sender.requests[0]
	if req.Method != http.MethodPost {
		t.Fatalf("method = %s, want POST", req.Method)
	}
	if req.URL.Query().Get("measurement_id") != "G-TEST" {
		t.Fatalf("measurement_id missing from URL: %s", req.URL.String())
	}
	if req.URL.Query().Get("api_secret") != "secret" {
		t.Fatalf("api_secret missing from URL: %s", req.URL.String())
	}
	if strings.Contains(sender.bodies[0], "raw-voucher") {
		t.Fatalf("payload leaked raw voucher")
	}
	var decoded ga4Payload
	if err := common.Unmarshal([]byte(sender.bodies[0]), &decoded); err != nil {
		t.Fatalf("payload is not valid json: %v", err)
	}
	if decoded.ClientID != "123.456" || decoded.Events[0].Name != eventVoucherRedeemSuccess {
		t.Fatalf("unexpected decoded payload: %#v", decoded)
	}
}

func TestSendPayloadReturnsStatusErrorWithoutPanic(t *testing.T) {
	sender := &captureSender{status: http.StatusInternalServerError}
	cfg := testConfig()
	restore := ConfigureForTest(cfg, sender)
	defer restore()

	err := sendPayload(context.Background(), cfg, ga4Payload{ClientID: "1.2"})
	if err == nil || !strings.Contains(err.Error(), "status=500") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestEnabledReflectsUsableConfig(t *testing.T) {
	cfg := testConfig()
	restore := ConfigureForTest(cfg, nil)
	if !Enabled() {
		t.Fatalf("complete GA4 config should be enabled")
	}
	restore()

	cfg.APISecret = ""
	restore = ConfigureForTest(cfg, nil)
	defer restore()
	if Enabled() {
		t.Fatalf("missing API secret should disable GA4 tracking")
	}
}

func TestSanitizeGA4SecretsRedactsAPISecret(t *testing.T) {
	raw := "Post \"https://www.google-analytics.com/mp/collect?measurement_id=G-TEST&api_secret=leaked-secret\": dial tcp timeout"
	got := sanitizeGA4Secrets(raw)
	if strings.Contains(got, "leaked-secret") {
		t.Fatalf("secret leaked after sanitization: %s", got)
	}
	if !strings.Contains(got, "api_secret=[redacted]") {
		t.Fatalf("redacted marker missing: %s", got)
	}
	if !strings.Contains(got, "measurement_id=G-TEST") {
		t.Fatalf("non-secret query params should remain: %s", got)
	}
}
