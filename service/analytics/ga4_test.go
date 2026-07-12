package analytics

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

type captureSender struct {
	mu       sync.Mutex
	requests []*http.Request
	bodies   []string
	status   int
	statuses []int
	err      error
	done     chan struct{}
}

func (s *captureSender) Do(req *http.Request) (*http.Response, error) {
	s.mu.Lock()
	s.requests = append(s.requests, req)
	body, _ := io.ReadAll(req.Body)
	s.bodies = append(s.bodies, string(body))
	err := s.err
	status := s.status
	if len(s.statuses) > 0 {
		status = s.statuses[0]
		s.statuses = s.statuses[1:]
	}
	s.mu.Unlock()
	if s.done != nil {
		s.done <- struct{}{}
	}
	if err != nil {
		return nil, err
	}
	if status == 0 {
		status = http.StatusNoContent
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(nil)),
	}, nil
}

func (s *captureSender) Attempts() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
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

func TestTrackPurchaseIncludesPaymentMetadataOnly(t *testing.T) {
	sender := &captureSender{done: make(chan struct{}, 1)}
	cfg := testConfig()
	restore := ConfigureForTest(cfg, sender)
	defer restore()
	restoreServerAddress := setTestServerAddress("https://lizh.ai")
	defer restoreServerAddress()

	TrackPurchase(nil, 42, PurchaseAttribution{
		TradeNo:         "order-202607060001",
		Value:           19.99,
		Currency:        "usd",
		PaymentProvider: "stripe",
		PaymentMethod:   "card",
		ItemType:        "top_up",
		QuotaAmount:     500000,
	})

	select {
	case <-sender.done:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for purchase send")
	}

	if len(sender.bodies) != 1 {
		t.Fatalf("sent %d requests, want 1", len(sender.bodies))
	}
	var decoded ga4Payload
	if err := common.Unmarshal([]byte(sender.bodies[0]), &decoded); err != nil {
		t.Fatalf("payload is not valid json: %v", err)
	}
	if len(decoded.Events) != 1 || decoded.Events[0].Name != eventPurchase {
		t.Fatalf("unexpected events: %#v", decoded.Events)
	}
	params := decoded.Events[0].Params
	if params["transaction_id"] != "order-202607060001" {
		t.Fatalf("transaction id missing: %#v", params)
	}
	if params["value"] != 19.99 || params["currency"] != "USD" {
		t.Fatalf("purchase value/currency missing: %#v", params)
	}
	if params["payment_provider"] != "stripe" || params["payment_method"] != "stripe" || params["payment_method_detail"] != "card" || params["item_type"] != "top_up" {
		t.Fatalf("payment metadata missing: %#v", params)
	}
	if params["user_id"] != float64(42) || params["hostname"] != "lizh.ai" || params["page_location"] != "https://lizh.ai/wallet" {
		t.Fatalf("purchase attribution context missing: %#v", params)
	}
	body := sender.bodies[0]
	for _, forbidden := range []string{"private@example.com", "sk-secret", "voucher-secret", "prompt", "response", "content"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("purchase payload leaked forbidden value %q: %s", forbidden, body)
		}
	}
}

func TestTrackPurchaseUsesStoredBrowserAttribution(t *testing.T) {
	sender := &captureSender{done: make(chan struct{}, 1)}
	restore := ConfigureForTest(testConfig(), sender)
	defer restore()

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Request, _ = http.NewRequest(http.MethodPost, "/webhook", nil)
	ctx.Request.AddCookie(&http.Cookie{Name: "_ga", Value: "GA1.1.999.888"})

	TrackPurchase(ctx, 42, PurchaseAttribution{
		TradeNo: "order-browser-attribution",
		Value:   19.99,
		Attribution: SignUpAttribution{
			ClientID:     "123.456",
			SessionID:    "789",
			PageLocation: "https://lizh.ai/wallet?utm_source=google",
			PageReferrer: "https://google.com/",
			Source:       "google",
			Medium:       "cpc",
			Campaign:     "launch",
			GCLID:        "click-123",
		},
	})

	select {
	case <-sender.done:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for attributed purchase send")
	}

	var decoded ga4Payload
	if err := common.Unmarshal([]byte(sender.bodies[0]), &decoded); err != nil {
		t.Fatalf("decode purchase payload: %v", err)
	}
	if decoded.ClientID != "123.456" {
		t.Fatalf("client id = %q, want stored browser id", decoded.ClientID)
	}
	params := decoded.Events[0].Params
	if params["session_id"] != float64(789) || params["engagement_time_msec"] != float64(1) || params["source"] != "google" || params["gclid"] != "click-123" {
		t.Fatalf("browser attribution missing: %#v", params)
	}
}

func TestNormalizeSignUpAttributionDropsInvalidIdentifiersAndOversizedValues(t *testing.T) {
	got := NormalizeSignUpAttribution(SignUpAttribution{
		ClientID:  "not-a-client-id",
		SessionID: "not-a-session-id",
		Source:    strings.Repeat("x", 300),
		Medium:    " cpc ",
	})
	if got.ClientID != "" || got.SessionID != "" || got.Source != "" {
		t.Fatalf("invalid attribution was retained: %#v", got)
	}
	if got.Medium != "cpc" {
		t.Fatalf("medium was not normalized: %q", got.Medium)
	}
}

func TestTrackTopUpUsesPurchaseEventNameAndConversionFields(t *testing.T) {
	sender := &captureSender{done: make(chan struct{}, 1)}
	cfg := testConfig()
	restore := ConfigureForTest(cfg, sender)
	defer restore()
	restoreServerAddress := setTestServerAddress("https://lizh.ai")
	defer restoreServerAddress()

	TrackTopUp(nil, 42, PurchaseAttribution{
		TradeNo:         "topup-202607060001",
		Value:           10,
		Currency:        "usd",
		PaymentProvider: "epay",
		PaymentMethod:   "alipay",
		ItemType:        "top_up",
		QuotaAmount:     100000,
	})

	select {
	case <-sender.done:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for top_up send")
	}

	var decoded ga4Payload
	if err := common.Unmarshal([]byte(sender.bodies[0]), &decoded); err != nil {
		t.Fatalf("payload is not valid json: %v", err)
	}
	if len(decoded.Events) != 1 || decoded.Events[0].Name != "purchase" {
		t.Fatalf("unexpected events: %#v", decoded.Events)
	}
	params := decoded.Events[0].Params
	if params["transaction_id"] != "topup-202607060001" || params["value"] != float64(10) || params["currency"] != "USD" {
		t.Fatalf("purchase conversion fields missing: %#v", params)
	}
	if params["payment_method"] != "epay" || params["payment_method_detail"] != "alipay" || params["payment_provider"] != "epay" {
		t.Fatalf("payment method missing: %#v", params)
	}
	items, ok := params["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("purchase items missing: %#v", params["items"])
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("purchase item has unexpected type: %#v", items[0])
	}
	if item["item_id"] != "top_up" || item["item_name"] != "Balance top-up" || item["price"] != float64(10) || item["quantity"] != float64(1) {
		t.Fatalf("purchase item fields missing: %#v", item)
	}
	if params["user_id"] != float64(42) || params["hostname"] != "lizh.ai" || params["page_location"] != "https://lizh.ai/wallet" {
		t.Fatalf("purchase context missing: %#v", params)
	}
}

func TestTrackVoucherRedeemSuccessUsesRedemptionIdAndOmitsRawCode(t *testing.T) {
	sender := &captureSender{done: make(chan struct{}, 1)}
	cfg := testConfig()
	restore := ConfigureForTest(cfg, sender)
	defer restore()
	restoreServerAddress := setTestServerAddress("https://lizh.ai")
	defer restoreServerAddress()

	TrackVoucherRedeemSuccess(nil, 42, "raw-voucher-code", int(10*common.QuotaPerUnit), RedemptionAttribution{
		TransactionID: "redemption:987",
		Source:        "voucher",
	})

	select {
	case <-sender.done:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for voucher_redeem_success send")
	}

	var decoded ga4Payload
	if err := common.Unmarshal([]byte(sender.bodies[0]), &decoded); err != nil {
		t.Fatalf("payload is not valid json: %v", err)
	}
	if len(decoded.Events) != 1 || decoded.Events[0].Name != "voucher_redeem_success" {
		t.Fatalf("unexpected events: %#v", decoded.Events)
	}
	params := decoded.Events[0].Params
	if params["transaction_id"] != "redemption:987" || params["value"] != float64(10) || params["currency"] != "USD" {
		t.Fatalf("voucher_redeem_success conversion fields missing: %#v", params)
	}
	if params["source"] != "voucher" || params["user_id"] != float64(42) || params["hostname"] != "lizh.ai" || params["page_location"] != "https://lizh.ai/wallet" {
		t.Fatalf("voucher_redeem_success context missing: %#v", params)
	}
	if strings.Contains(sender.bodies[0], "raw-voucher-code") {
		t.Fatalf("voucher_redeem_success payload leaked raw voucher code: %s", sender.bodies[0])
	}
}

func TestTrackVoucherRedeemSuccessUsesRequestCookieSessionContext(t *testing.T) {
	sender := &captureSender{done: make(chan struct{}, 1)}
	restore := ConfigureForTest(testConfig(), sender)
	defer restore()

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Request, _ = http.NewRequest(http.MethodPost, "/api/user/topup", nil)
	ctx.Request.AddCookie(&http.Cookie{Name: "_ga", Value: "GA1.1.123.456"})
	ctx.Request.AddCookie(&http.Cookie{Name: "_ga_TEST", Value: "GS1.1.1740000000.1.1.1740000000.0.0.0"})

	TrackVoucherRedeemSuccess(ctx, 42, "raw-voucher-code", int(10*common.QuotaPerUnit), RedemptionAttribution{
		TransactionID: "redemption:987",
	})

	select {
	case <-sender.done:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for voucher_redeem_success send")
	}

	var decoded ga4Payload
	if err := common.Unmarshal([]byte(sender.bodies[0]), &decoded); err != nil {
		t.Fatalf("payload is not valid json: %v", err)
	}
	if decoded.ClientID != "123.456" {
		t.Fatalf("client id = %q, want request cookie client id", decoded.ClientID)
	}
	params := decoded.Events[0].Params
	if params["session_id"] != float64(1740000000) || params["engagement_time_msec"] != float64(1) {
		t.Fatalf("voucher_redeem_success request cookie session context missing: %#v", params)
	}
}

func TestTrackAPIKeyCreatedIncludesKeyTypeAndContextOnly(t *testing.T) {
	sender := &captureSender{done: make(chan struct{}, 1)}
	cfg := testConfig()
	restore := ConfigureForTest(cfg, sender)
	defer restore()
	restoreServerAddress := setTestServerAddress("https://lizh.ai")
	defer restoreServerAddress()

	TrackAPIKeyCreated(nil, 42, 7, "sk-secret-api-key", UserAttribution{
		KeyType: "api_key",
	})

	select {
	case <-sender.done:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for api_key_created send")
	}

	var decoded ga4Payload
	if err := common.Unmarshal([]byte(sender.bodies[0]), &decoded); err != nil {
		t.Fatalf("payload is not valid json: %v", err)
	}
	if len(decoded.Events) != 1 || decoded.Events[0].Name != eventAPIKeyCreated {
		t.Fatalf("unexpected events: %#v", decoded.Events)
	}
	params := decoded.Events[0].Params
	if params["key_type"] != "api_key" || params["user_id"] != float64(42) || params["hostname"] != "lizh.ai" || params["page_location"] != "https://lizh.ai/keys" {
		t.Fatalf("api_key_created context missing: %#v", params)
	}
	if strings.Contains(sender.bodies[0], "sk-secret-api-key") {
		t.Fatalf("api_key_created payload leaked raw API key: %s", sender.bodies[0])
	}
}

func TestTrackAPIKeyCreatedUsesBrowserAttributionContext(t *testing.T) {
	sender := &captureSender{done: make(chan struct{}, 1)}
	restore := ConfigureForTest(testConfig(), sender)
	defer restore()

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Request, _ = http.NewRequest(http.MethodPost, "/api/token/", nil)
	ctx.Request.AddCookie(&http.Cookie{Name: "_ga", Value: "GA1.1.999.888"})
	ctx.Request.AddCookie(&http.Cookie{Name: "_ga_TEST", Value: "GS1.1.999.1.1.999.0.0.0"})

	attrs := UserAttribution{
		KeyType: "api_key",
		Attribution: SignUpAttribution{
			ClientID:     "123.456",
			SessionID:    "789",
			PageLocation: "https://lizh.ai/console/token?utm_source=google",
			Source:       "google",
			Campaign:     "launch",
			GCLID:        "click",
		},
	}
	TrackAPIKeyCreated(ctx, 42, 7, "sk-secret-api-key", attrs)

	select {
	case <-sender.done:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for api_key_created send")
	}

	var decoded ga4Payload
	if err := common.Unmarshal([]byte(sender.bodies[0]), &decoded); err != nil {
		t.Fatalf("payload is not valid json: %v", err)
	}
	if decoded.ClientID != "123.456" {
		t.Fatalf("client id = %q, want browser attribution client id", decoded.ClientID)
	}
	params := decoded.Events[0].Params
	if params["session_id"] != float64(789) || params["engagement_time_msec"] != float64(1) || params["source"] != "google" || params["campaign"] != "launch" || params["gclid"] != "click" {
		t.Fatalf("api_key_created browser attribution missing: %#v", params)
	}
	if params["page_location"] != "https://lizh.ai/console/token?utm_source=google" {
		t.Fatalf("page location = %#v, want browser attribution page", params["page_location"])
	}
	if strings.Contains(sender.bodies[0], "sk-secret-api-key") {
		t.Fatalf("api_key_created payload leaked raw API key: %s", sender.bodies[0])
	}
}

func TestTrackAPIKeyCreatedFallsBackToRequestCookiesWithoutAttribution(t *testing.T) {
	sender := &captureSender{done: make(chan struct{}, 1)}
	restore := ConfigureForTest(testConfig(), sender)
	defer restore()

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Request, _ = http.NewRequest(http.MethodPost, "/api/token/", nil)
	ctx.Request.AddCookie(&http.Cookie{Name: "_ga", Value: "GA1.1.123.456"})
	ctx.Request.AddCookie(&http.Cookie{Name: "_ga_TEST", Value: "GS1.1.789.1.1.999.0.0.0"})

	TrackAPIKeyCreated(ctx, 42, 7, "sk-secret-api-key", UserAttribution{KeyType: "api_key"})

	select {
	case <-sender.done:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for api_key_created send")
	}

	var decoded ga4Payload
	if err := common.Unmarshal([]byte(sender.bodies[0]), &decoded); err != nil {
		t.Fatalf("payload is not valid json: %v", err)
	}
	params := decoded.Events[0].Params
	if decoded.ClientID != "123.456" || params["session_id"] != float64(789) || params["engagement_time_msec"] != float64(1) {
		t.Fatalf("api_key_created request cookie fallback missing: %#v", decoded)
	}
}

func TestTrackAPIKeyCreatedFallsBackToServerClientWithoutContext(t *testing.T) {
	sender := &captureSender{done: make(chan struct{}, 1)}
	restore := ConfigureForTest(testConfig(), sender)
	defer restore()

	TrackAPIKeyCreated(nil, 42, 7, "sk-secret-api-key", UserAttribution{KeyType: "api_key"})

	select {
	case <-sender.done:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for api_key_created send")
	}

	var decoded ga4Payload
	if err := common.Unmarshal([]byte(sender.bodies[0]), &decoded); err != nil {
		t.Fatalf("payload is not valid json: %v", err)
	}
	if !strings.HasPrefix(decoded.ClientID, "server.") {
		t.Fatalf("client id = %q, want server fallback", decoded.ClientID)
	}
	params := decoded.Events[0].Params
	if _, ok := params["session_id"]; ok {
		t.Fatalf("nil context unexpectedly produced session id: %#v", params)
	}
	if _, ok := params["engagement_time_msec"]; ok {
		t.Fatalf("nil context unexpectedly produced engagement context: %#v", params)
	}
	if strings.Contains(sender.bodies[0], "sk-secret-api-key") {
		t.Fatalf("api_key_created payload leaked raw API key: %s", sender.bodies[0])
	}
}

func TestTrackFirstAPICallIncludesEndpointStatusAndContext(t *testing.T) {
	sender := &captureSender{done: make(chan struct{}, 1)}
	cfg := testConfig()
	restore := ConfigureForTest(cfg, sender)
	defer restore()
	restoreServerAddress := setTestServerAddress("https://lizh.ai")
	defer restoreServerAddress()

	TrackFirstAPIRequestSuccessWithResult(nil, 42, 7, "sk-secret-api-key", FirstAPIRequestAttribution{
		Model:      "glm-5.2",
		Endpoint:   "/v1/chat/completions",
		StatusCode: http.StatusOK,
		QuotaSpent: 100,
	}, nil)

	select {
	case <-sender.done:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for first_api_call send")
	}

	var decoded ga4Payload
	if err := common.Unmarshal([]byte(sender.bodies[0]), &decoded); err != nil {
		t.Fatalf("payload is not valid json: %v", err)
	}
	if len(decoded.Events) != 1 || decoded.Events[0].Name != "first_api_call" {
		t.Fatalf("unexpected events: %#v", decoded.Events)
	}
	params := decoded.Events[0].Params
	if params["model"] != "glm-5.2" || params["endpoint"] != "/v1/chat/completions" || params["status_code"] != float64(200) {
		t.Fatalf("first_api_call metadata missing: %#v", params)
	}
	if params["user_id"] != float64(42) || params["hostname"] != "lizh.ai" || params["page_location"] != "https://lizh.ai/v1/chat/completions" {
		t.Fatalf("first_api_call context missing: %#v", params)
	}
	if !strings.HasPrefix(decoded.ClientID, "server.") {
		t.Fatalf("client id = %q, want server fallback", decoded.ClientID)
	}
	if _, ok := params["session_id"]; ok {
		t.Fatalf("nil context unexpectedly produced session id: %#v", params)
	}
	if _, ok := params["engagement_time_msec"]; ok {
		t.Fatalf("nil context unexpectedly produced engagement context: %#v", params)
	}
	if strings.Contains(sender.bodies[0], "sk-secret-api-key") {
		t.Fatalf("first_api_call payload leaked raw API key: %s", sender.bodies[0])
	}
}

func TestTrackFirstAPICallUsesBrowserAttributionContext(t *testing.T) {
	sender := &captureSender{done: make(chan struct{}, 1)}
	restore := ConfigureForTest(testConfig(), sender)
	defer restore()

	attrs := FirstAPIRequestAttribution{
		Model:      "glm-5.2",
		Endpoint:   "/v1/chat/completions",
		StatusCode: http.StatusOK,
		QuotaSpent: 100,
		Attribution: SignUpAttribution{
			ClientID:     "123.456",
			SessionID:    "789",
			PageLocation: "https://lizh.ai/console/token?utm_source=google",
			Source:       "google",
			Campaign:     "launch",
			GCLID:        "click",
		},
	}
	TrackFirstAPIRequestSuccessWithResult(nil, 42, 7, "sk-secret-api-key", attrs, nil)

	select {
	case <-sender.done:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for first_api_call send")
	}

	var decoded ga4Payload
	if err := common.Unmarshal([]byte(sender.bodies[0]), &decoded); err != nil {
		t.Fatalf("payload is not valid json: %v", err)
	}
	if decoded.ClientID != "123.456" {
		t.Fatalf("client id = %q, want browser attribution client id", decoded.ClientID)
	}
	params := decoded.Events[0].Params
	if params["session_id"] != float64(789) || params["engagement_time_msec"] != float64(1) || params["source"] != "google" || params["campaign"] != "launch" || params["gclid"] != "click" {
		t.Fatalf("first_api_call browser attribution missing: %#v", params)
	}
	if strings.Contains(sender.bodies[0], "sk-secret-api-key") {
		t.Fatalf("first_api_call payload leaked raw API key: %s", sender.bodies[0])
	}
}

func TestTrackSignUpIncludesAttributionWithoutPII(t *testing.T) {
	sender := &captureSender{done: make(chan struct{}, 1)}
	cfg := testConfig()
	restore := ConfigureForTest(cfg, sender)
	defer restore()

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Request, _ = http.NewRequest(http.MethodPost, "/api/user/register", nil)

	TrackSignUp(ctx, 42, SignUpAttribution{
		ClientID:     "111.222",
		SessionID:    "333",
		PageLocation: "https://lizh.ai/?utm_source=plati",
		PageReferrer: "https://plati.market/",
		Source:       "plati",
		Medium:       "marketplace",
		Campaign:     "launch",
		Term:         "chatgpt",
		Content:      "card-a",
		GCLID:        "gclid-value",
		FBCLID:       "fbclid-value",
		TTCLID:       "ttclid-value",
		YCLID:        "yclid-value",
		FirstVisitAt: "2026-06-12T10:00:00.000Z",
		Method:       "email",
	})

	select {
	case <-sender.done:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for sign_up send")
	}

	if len(sender.bodies) != 1 {
		t.Fatalf("sent %d requests, want 1", len(sender.bodies))
	}
	var decoded ga4Payload
	if err := common.Unmarshal([]byte(sender.bodies[0]), &decoded); err != nil {
		t.Fatalf("payload is not valid json: %v", err)
	}
	if decoded.ClientID != "111.222" {
		t.Fatalf("client id = %q, want attribution client id", decoded.ClientID)
	}
	if decoded.UserID == "" || decoded.UserID == "42" {
		t.Fatalf("user id should be hashed, got %q", decoded.UserID)
	}
	if len(decoded.Events) != 1 || decoded.Events[0].Name != eventSignUp {
		t.Fatalf("unexpected events: %#v", decoded.Events)
	}
	params := decoded.Events[0].Params
	if params["method"] != "email" || params["session_id"] != float64(333) || params["engagement_time_msec"] != float64(1) || params["source"] != "plati" || params["gclid"] != "gclid-value" {
		t.Fatalf("attribution params missing: %#v", params)
	}
	if params["user_id"] != float64(42) || params["hostname"] != "lizh.ai" {
		t.Fatalf("sign_up context missing: %#v", params)
	}
	for _, forbidden := range []string{"email", "username", "password", "phone"} {
		if _, ok := params[forbidden]; ok {
			t.Fatalf("sign_up params leaked private field %q: %#v", forbidden, params)
		}
		if strings.Contains(sender.bodies[0], forbidden+"@") {
			t.Fatalf("sign_up payload appears to leak private value: %s", sender.bodies[0])
		}
	}
}

func TestTrackSignUpUsesRequestSessionCookieWhenAttributionMissing(t *testing.T) {
	tests := []struct {
		name          string
		cookieValue   string
		wantSessionID float64
	}{
		{name: "GS1 cookie", cookieValue: "GS1.1.1700000000.1.1.1700000100.0.0.0", wantSessionID: 1700000000},
		{name: "GS2 cookie", cookieValue: "GS2.1.s1700000001$o1$g0$t1700000100$j60$l0$h0", wantSessionID: 1700000001},
		{name: "invalid prefix", cookieValue: "garbage.s1700000002", wantSessionID: 0},
		{name: "invalid GS1 prefix", cookieValue: "GSX.1.1700000003.1", wantSessionID: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &captureSender{done: make(chan struct{}, 1)}
			restore := ConfigureForTest(testConfig(), sender)
			defer restore()

			gin.SetMode(gin.TestMode)
			ctx, _ := gin.CreateTestContext(nil)
			ctx.Request, _ = http.NewRequest(http.MethodGet, "/api/oauth/github", nil)
			ctx.Request.AddCookie(&http.Cookie{Name: "_ga", Value: "GA1.1.111.222"})
			ctx.Request.AddCookie(&http.Cookie{Name: "_ga_TEST", Value: tt.cookieValue})

			TrackSignUp(ctx, 42, SignUpAttribution{Method: "github"})

			select {
			case <-sender.done:
			case <-time.After(time.Second):
				t.Fatalf("timed out waiting for sign_up send")
			}

			var decoded ga4Payload
			if err := common.Unmarshal([]byte(sender.bodies[0]), &decoded); err != nil {
				t.Fatalf("payload is not valid json: %v", err)
			}
			params := decoded.Events[0].Params
			if tt.wantSessionID == 0 {
				if _, ok := params["session_id"]; ok {
					t.Fatalf("invalid cookie produced session context: %#v", params)
				}
				if _, ok := params["engagement_time_msec"]; ok {
					t.Fatalf("invalid cookie produced engagement context: %#v", params)
				}
				return
			}
			if params["session_id"] != tt.wantSessionID || params["engagement_time_msec"] != float64(1) {
				t.Fatalf("request session context missing: %#v", params)
			}
		})
	}
}

func TestTrackSignUpSanitizesAttributionURLs(t *testing.T) {
	sender := &captureSender{done: make(chan struct{}, 1)}
	cfg := testConfig()
	restore := ConfigureForTest(cfg, sender)
	defer restore()

	TrackSignUp(nil, 42, SignUpAttribution{
		ClientID:     "111.222",
		PageLocation: "https://lizh.ai/user/reset?email=private@example.com&token=secret-token&utm_source=plati&gclid=gclid-value",
		PageReferrer: "https://partner.example/path?email=private@example.com&token=secret-token&utm_medium=marketplace",
		Source:       "plati",
		Medium:       "marketplace",
		Method:       "email",
	})

	select {
	case <-sender.done:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for sign_up send")
	}

	var decoded ga4Payload
	if err := common.Unmarshal([]byte(sender.bodies[0]), &decoded); err != nil {
		t.Fatalf("payload is not valid json: %v", err)
	}
	body := sender.bodies[0]
	for _, forbidden := range []string{"private@example.com", "secret-token", "email=", "token="} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("sign_up payload leaked private URL value %q: %s", forbidden, body)
		}
	}
	params := decoded.Events[0].Params
	if params["page_location"] != "https://lizh.ai/user/reset?gclid=gclid-value&utm_source=plati" {
		t.Fatalf("unexpected sanitized page_location: %#v", params["page_location"])
	}
	if params["page_referrer"] != "https://partner.example/path?utm_medium=marketplace" {
		t.Fatalf("unexpected sanitized page_referrer: %#v", params["page_referrer"])
	}
}

func TestSanitizeAttributionURLPreservesEscapedPath(t *testing.T) {
	raw := "https://lizh.ai/%E4%BB%B7%E6%A0%BC/a%20b?token=secret&utm_source=plati"
	want := "https://lizh.ai/%E4%BB%B7%E6%A0%BC/a%20b?utm_source=plati"

	if got := sanitizeAttributionURL(raw); got != want {
		t.Fatalf("sanitizeAttributionURL() = %q, want %q", got, want)
	}
}

func TestResolvePageLocationPreservesEscapedFallbackPath(t *testing.T) {
	restoreServerAddress := setTestServerAddress("https://lizh.ai")
	defer restoreServerAddress()

	fallback := "https://api.example/%E4%BB%B7%E6%A0%BC/a%20b?token=secret"
	want := "https://lizh.ai/%E4%BB%B7%E6%A0%BC/a%20b"
	if got := resolvePageLocation(nil, "", fallback); got != want {
		t.Fatalf("resolvePageLocation() = %q, want %q", got, want)
	}
}

func TestNormalizePagePathDropsQueryOnlyInput(t *testing.T) {
	path, rawPath := normalizePagePath("?token=secret")
	if path != "" || rawPath != "" {
		t.Fatalf("normalizePagePath() = (%q, %q), want empty paths", path, rawPath)
	}
}

func TestNormalizePagePathDropsAbsoluteURLWithoutPath(t *testing.T) {
	path, rawPath := normalizePagePath("https://api.example?token=secret")
	if path != "" || rawPath != "" {
		t.Fatalf("normalizePagePath() = (%q, %q), want empty paths", path, rawPath)
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

func TestTrackFirstAPICallWithResultDoesNotReportSuccessWhenDisabled(t *testing.T) {
	sender := &captureSender{}
	cfg := testConfig()
	cfg.Enabled = false
	restore := ConfigureForTest(cfg, sender)
	defer restore()

	called := false
	TrackFirstAPIRequestSuccessWithResult(nil, 42, 7, "token-key", FirstAPIRequestAttribution{
		Model:      "gpt-test",
		Endpoint:   "/v1/chat/completions",
		StatusCode: http.StatusOK,
		QuotaSpent: 100,
	}, func(err error) {
		called = true
	})

	if called {
		t.Fatalf("disabled tracking should not report successful delivery")
	}
	if len(sender.requests) != 0 {
		t.Fatalf("disabled tracking sent %d requests", len(sender.requests))
	}
}

func setTestServerAddress(address string) func() {
	old := system_setting.ServerAddress
	system_setting.ServerAddress = address
	return func() {
		system_setting.ServerAddress = old
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

func TestSendPayloadRetriesTransientStatuses(t *testing.T) {
	sender := &captureSender{statuses: []int{
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusNoContent,
	}}
	cfg := testConfig()
	restore := ConfigureForTest(cfg, sender)
	defer restore()

	err := sendPayloadWithRetry(
		context.Background(),
		cfg,
		ga4Payload{ClientID: "1.2"},
		[]time.Duration{0, 0},
	)
	if err != nil {
		t.Fatalf("sendPayloadWithRetry returned error: %v", err)
	}
	if sender.Attempts() != 3 {
		t.Fatalf("attempts = %d, want 3", sender.Attempts())
	}
}

func TestSendPayloadDoesNotRetryPermanentClientError(t *testing.T) {
	sender := &captureSender{statuses: []int{http.StatusBadRequest, http.StatusNoContent}}
	cfg := testConfig()
	restore := ConfigureForTest(cfg, sender)
	defer restore()

	err := sendPayloadWithRetry(
		context.Background(),
		cfg,
		ga4Payload{ClientID: "1.2"},
		[]time.Duration{0, 0},
	)
	if err == nil || !strings.Contains(err.Error(), "status=400") {
		t.Fatalf("expected permanent status error, got %v", err)
	}
	if sender.Attempts() != 1 {
		t.Fatalf("attempts = %d, want 1", sender.Attempts())
	}
}

func TestTrackWithResultCallsCallbackOnceAfterRetry(t *testing.T) {
	sender := &captureSender{statuses: []int{http.StatusInternalServerError, http.StatusNoContent}}
	cfg := testConfig()
	restore := ConfigureForTest(cfg, sender)
	defer restore()

	results := make(chan error, 2)
	trackWithClientID(nil, cfg, 42, 0, eventPurchase, EventParams{}, "123.456", func(err error) {
		results <- err
	})

	select {
	case err := <-results:
		if err != nil {
			t.Fatalf("callback returned error after retry: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for retry callback")
	}
	select {
	case err := <-results:
		t.Fatalf("callback ran more than once: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if sender.Attempts() != 2 {
		t.Fatalf("attempts = %d, want 2", sender.Attempts())
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
