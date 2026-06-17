package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type optionsAPIResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    []model.Option `json:"data"`
}

func TestGetOptionsReturnsConfiguredPlaceholderForBinancePaySecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)

	common.OptionMapRWMutex.Lock()
	originalOptionMap := common.OptionMap
	common.OptionMap = map[string]string{
		"BinancePayApiKey":     "real-binance-key",
		"BinancePayApiSecret":  "real-binance-secret",
		"StripeApiSecret":      "real-stripe-secret",
		"BinancePayEnabled":    "true",
		"BinancePayReturnURL":  "https://example.com/return",
		"CompletionRatio":      "{}",
		"ModelRatio":           "{}",
		"ModelPrice":           "{}",
		"CacheRatio":           "{}",
		"CreateCacheRatio":     "{}",
		"ImageRatio":           "{}",
		"AudioRatio":           "{}",
		"AudioCompletionRatio": "{}",
	}
	common.OptionMapRWMutex.Unlock()

	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/option/", nil)

	GetOptions(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response optionsAPIResponse
	if err := common.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("expected success response, got message %q", response.Message)
	}

	values := make(map[string]string, len(response.Data))
	for _, option := range response.Data {
		values[option.Key] = option.Value
	}

	if values["BinancePayApiKey"] != configuredSensitiveOptionPlaceholder {
		t.Fatalf("expected BinancePayApiKey placeholder, got %q", values["BinancePayApiKey"])
	}
	if values["BinancePayApiSecret"] != configuredSensitiveOptionPlaceholder {
		t.Fatalf("expected BinancePayApiSecret placeholder, got %q", values["BinancePayApiSecret"])
	}
	if _, ok := values["StripeApiSecret"]; ok {
		t.Fatal("expected StripeApiSecret to remain hidden")
	}
	for key, value := range values {
		switch value {
		case "real-binance-key", "real-binance-secret", "real-stripe-secret":
			t.Fatalf("option %s leaked sensitive value %q", key, value)
		}
	}
}
