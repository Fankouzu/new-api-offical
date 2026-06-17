package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestFormatWaffoPancakeAmount_UsesDisplayPriceString(t *testing.T) {
	testCases := []struct {
		name     string
		amount   float64
		expected string
	}{
		{name: "whole amount", amount: 29, expected: "29.00"},
		{name: "decimal amount", amount: 29.9, expected: "29.90"},
		{name: "round half up to cents", amount: 29.999, expected: "30.00"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, formatWaffoPancakeAmount(tc.amount))
		})
	}
}

func TestWaffoPancakeWebhookDoesNotLogRawPayloadOrSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalEnabled := setting.WaffoPancakeEnabled
	originalPublicKey := setting.WaffoPancakeWebhookPublicKey
	originalSandbox := setting.WaffoPancakeSandbox
	originalTestKey := setting.WaffoPancakeWebhookTestKey
	originalMerchantID := setting.WaffoPancakeMerchantID
	originalPrivateKey := setting.WaffoPancakePrivateKey
	originalStoreID := setting.WaffoPancakeStoreID
	originalProductID := setting.WaffoPancakeProductID
	t.Cleanup(func() {
		setting.WaffoPancakeEnabled = originalEnabled
		setting.WaffoPancakeWebhookPublicKey = originalPublicKey
		setting.WaffoPancakeSandbox = originalSandbox
		setting.WaffoPancakeWebhookTestKey = originalTestKey
		setting.WaffoPancakeMerchantID = originalMerchantID
		setting.WaffoPancakePrivateKey = originalPrivateKey
		setting.WaffoPancakeStoreID = originalStoreID
		setting.WaffoPancakeProductID = originalProductID
	})

	setting.WaffoPancakeEnabled = true
	setting.WaffoPancakeSandbox = false
	setting.WaffoPancakeMerchantID = "merchant"
	setting.WaffoPancakePrivateKey = "private"
	setting.WaffoPancakeStoreID = "store"
	setting.WaffoPancakeProductID = "product"
	setting.WaffoPancakeWebhookPublicKey = "invalid-public-key"
	setting.WaffoPancakeWebhookTestKey = ""

	var logBuffer bytes.Buffer
	oldWriter := gin.DefaultWriter
	oldErrorWriter := gin.DefaultErrorWriter
	gin.DefaultWriter = &logBuffer
	gin.DefaultErrorWriter = &logBuffer
	t.Cleanup(func() {
		gin.DefaultWriter = oldWriter
		gin.DefaultErrorWriter = oldErrorWriter
	})

	payload := `{"id":"evt_test","eventType":"order.completed","mode":"prod","data":{"orderId":"WAFFO_PANCAKE-1","buyerEmail":"buyer@example.com"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/waffo-pancake/webhook", strings.NewReader(payload))
	req.Header.Set("X-Waffo-Signature", "sensitive-signature")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req

	WaffoPancakeWebhook(c)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	logOutput := logBuffer.String()
	require.NotContains(t, logOutput, "buyer@example.com")
	require.NotContains(t, logOutput, payload)
	require.NotContains(t, logOutput, "sensitive-signature")
	require.Contains(t, logOutput, "body_bytes=")
}

func TestGetWaffoPancakePayMoney(t *testing.T) {
	originalUnitPrice := setting.WaffoPancakeUnitPrice
	originalQuotaDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	originalDiscounts := make(map[int]float64, len(operation_setting.GetPaymentSetting().AmountDiscount))
	for k, v := range operation_setting.GetPaymentSetting().AmountDiscount {
		originalDiscounts[k] = v
	}
	originalTopupGroupRatio := common.TopupGroupRatio2JSONString()

	t.Cleanup(func() {
		setting.WaffoPancakeUnitPrice = originalUnitPrice
		operation_setting.GetGeneralSetting().QuotaDisplayType = originalQuotaDisplayType
		operation_setting.GetPaymentSetting().AmountDiscount = originalDiscounts
		require.NoError(t, common.UpdateTopupGroupRatioByJSONString(originalTopupGroupRatio))
	})

	setting.WaffoPancakeUnitPrice = 2.5
	operation_setting.GetPaymentSetting().AmountDiscount = map[int]float64{
		10:                           0.8,
		int(common.QuotaPerUnit * 3): 0.5,
		20:                           0,
	}
	require.NoError(t, common.UpdateTopupGroupRatioByJSONString(`{"default":1,"vip":1.2}`))

	testCases := []struct {
		name             string
		amount           int64
		group            string
		quotaDisplayType string
		expected         float64
	}{
		{
			name:             "currency display applies unit price group ratio and discount",
			amount:           10,
			group:            "vip",
			quotaDisplayType: operation_setting.QuotaDisplayTypeUSD,
			expected:         24,
		},
		{
			name:             "tokens display converts quota to display units before pricing",
			amount:           int64(common.QuotaPerUnit * 3),
			group:            "vip",
			quotaDisplayType: operation_setting.QuotaDisplayTypeTokens,
			expected:         4.5,
		},
		{
			name:             "non-positive discount falls back to no discount",
			amount:           20,
			group:            "default",
			quotaDisplayType: operation_setting.QuotaDisplayTypeUSD,
			expected:         50,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			operation_setting.GetGeneralSetting().QuotaDisplayType = tc.quotaDisplayType
			actual := getWaffoPancakePayMoney(tc.amount, tc.group)
			require.InDelta(t, tc.expected, actual, 0.000001)
		})
	}
}
