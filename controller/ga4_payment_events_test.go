package controller

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/analytics"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stripe/stripe-go/v81"
	"gorm.io/gorm"
)

type paymentGA4CaptureSender struct {
	mu     sync.Mutex
	bodies []string
	done   chan struct{}
	status int
}

func (s *paymentGA4CaptureSender) Do(req *http.Request) (*http.Response, error) {
	body, _ := io.ReadAll(req.Body)
	s.mu.Lock()
	s.bodies = append(s.bodies, string(body))
	status := s.status
	s.mu.Unlock()
	if s.done != nil {
		s.done <- struct{}{}
	}
	if status == 0 {
		status = http.StatusNoContent
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(nil)),
	}, nil
}

func (s *paymentGA4CaptureSender) snapshotBodies() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.bodies...)
}

func (s *paymentGA4CaptureSender) setStatus(status int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
}

func TestResolveGA4PaymentCurrencyPreservesExplicitCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		want     string
	}{
		{name: "euro", currency: "eur", want: "EUR"},
		{name: "yuan", currency: " cny ", want: "CNY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveGA4PaymentCurrency(model.PaymentProviderStripe, tt.currency)
			if got != tt.want {
				t.Fatalf("resolveGA4PaymentCurrency() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveGA4PaymentCurrencyNormalizesStablecoinsToUSD(t *testing.T) {
	for _, currency := range []string{"USDT", "usdc", " BUSD "} {
		t.Run(currency, func(t *testing.T) {
			got := resolveGA4PaymentCurrency(model.PaymentProviderBinancePay, currency)
			if got != "USD" {
				t.Fatalf("resolveGA4PaymentCurrency() = %q, want USD", got)
			}
		})
	}
}

func TestResolveGA4PaymentCurrencyFallsBackToProviderConfiguration(t *testing.T) {
	originalBinancePayCurrency := setting.BinancePayCurrency
	originalWaffoCurrency := setting.WaffoCurrency
	originalWaffoPancakeCurrency := setting.WaffoPancakeCurrency
	t.Cleanup(func() {
		setting.BinancePayCurrency = originalBinancePayCurrency
		setting.WaffoCurrency = originalWaffoCurrency
		setting.WaffoPancakeCurrency = originalWaffoPancakeCurrency
	})

	setting.BinancePayCurrency = "USDT"
	setting.WaffoCurrency = "eur"
	setting.WaffoPancakeCurrency = " cny "

	tests := []struct {
		provider string
		want     string
	}{
		{provider: model.PaymentProviderBinancePay, want: "USD"},
		{provider: model.PaymentProviderWaffo, want: "EUR"},
		{provider: model.PaymentProviderWaffoPancake, want: "CNY"},
	}
	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			if got := resolveGA4PaymentCurrency(tt.provider, ""); got != tt.want {
				t.Fatalf("resolveGA4PaymentCurrency() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveGA4PaymentCurrencyFallsBackToCNYDisplay(t *testing.T) {
	generalSetting := operation_setting.GetGeneralSetting()
	originalDisplayType := generalSetting.QuotaDisplayType
	t.Cleanup(func() {
		generalSetting.QuotaDisplayType = originalDisplayType
	})
	generalSetting.QuotaDisplayType = operation_setting.QuotaDisplayTypeCNY

	got := resolveGA4PaymentCurrency(model.PaymentProviderEpay, "")
	if got != "CNY" {
		t.Fatalf("resolveGA4PaymentCurrency() = %q, want CNY", got)
	}
}

func TestBeginGA4TopUpDeliveryKeepsLegacyIdempotencyKey(t *testing.T) {
	setupGA4PaymentEventTestDB(t)

	legacyID := model.BeginAnalyticsEventDelivery(ga4SubjectTypeTopUp, 42, "top_up")
	if legacyID <= 0 || !model.MarkAnalyticsEventSent(legacyID) {
		t.Fatalf("failed to seed legacy sent delivery")
	}
	if got := beginGA4TopUpDelivery(42); got != 0 {
		t.Fatalf("legacy sent delivery should suppress replay, got mark id %d", got)
	}
	if _, err := model.GetAnalyticsEventMark(ga4SubjectTypeTopUp, 42, "purchase"); err == nil {
		t.Fatalf("legacy delivery replay created a purchase mark")
	}

	newID := beginGA4TopUpDelivery(43)
	if newID <= 0 {
		t.Fatalf("new top-up delivery should begin")
	}
	mark, err := model.GetAnalyticsEventMark(ga4SubjectTypeTopUp, 43, "top_up")
	if err != nil || mark.Id != newID {
		t.Fatalf("new delivery did not use legacy top_up key: mark=%#v err=%v", mark, err)
	}
}

func TestEpayNotifyReclaimsFailedTopUpAnalyticsMark(t *testing.T) {
	setupGA4PaymentEventTestDB(t)
	sender, topUp := setupFailedTopUpAnalyticsReplay(t, model.PaymentProviderEpay, "epay-replay")

	originalPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	originalPayMethods := operation_setting.PayMethods
	t.Cleanup(func() {
		operation_setting.PayAddress = originalPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
		operation_setting.PayMethods = originalPayMethods
	})
	operation_setting.PayAddress = "https://payments.example.test"
	operation_setting.EpayId = "merchant-test"
	operation_setting.EpayKey = "epay-test-key"
	operation_setting.PayMethods = []map[string]string{{"type": model.PaymentProviderEpay}}

	params := epay.GenerateParams(map[string]string{
		"pid":          operation_setting.EpayId,
		"type":         "alipay",
		"out_trade_no": topUp.TradeNo,
		"trade_no":     "provider-epay-replay",
		"trade_status": epay.StatusTradeSuccess,
		"money":        "10.00",
	}, operation_setting.EpayKey)
	query := url.Values{}
	for key, value := range params {
		query.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/user/pay/notify?"+query.Encode(), nil)
	EpayNotify(c)

	waitForPaymentGA4Send(t, sender.done)
	mark := waitForTopUpGA4MarkTerminal(t, topUp.Id)
	if mark.Status != model.AnalyticsEventStatusSent {
		t.Fatalf("Epay replay mark status = %q, want sent", mark.Status)
	}
}

func TestEpayNotifyDoesNotCreateAnalyticsMarkForUnverifiedCompletedOrder(t *testing.T) {
	setupGA4PaymentEventTestDB(t)
	topUp := &model.TopUp{
		UserId:          902,
		Amount:          2,
		Money:           10,
		TradeNo:         "epay-no-mark",
		PaymentMethod:   "alipay",
		PaymentProvider: model.PaymentProviderEpay,
		Status:          common.TopUpStatusSuccess,
		CreateTime:      common.GetTimestamp(),
		CompleteTime:    common.GetTimestamp(),
	}
	if err := model.DB.Create(topUp).Error; err != nil {
		t.Fatalf("create completed Epay top-up: %v", err)
	}

	originalPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	originalPayMethods := operation_setting.PayMethods
	t.Cleanup(func() {
		operation_setting.PayAddress = originalPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
		operation_setting.PayMethods = originalPayMethods
	})
	operation_setting.PayAddress = "https://payments.example.test"
	operation_setting.EpayId = "merchant-test"
	operation_setting.EpayKey = "epay-test-key"
	operation_setting.PayMethods = []map[string]string{{"type": model.PaymentProviderEpay}}

	sender := &paymentGA4CaptureSender{done: make(chan struct{}, 1)}
	restore := analytics.ConfigureForTest(analytics.Config{
		Enabled:       true,
		MeasurementID: "G-TEST",
		APISecret:     "secret",
		HashSalt:      "salt",
		Timeout:       50 * time.Millisecond,
		Endpoint:      "https://example.test/mp/collect",
	}, sender)
	t.Cleanup(restore)

	params := epay.GenerateParams(map[string]string{
		"pid":          operation_setting.EpayId,
		"type":         "alipay",
		"out_trade_no": topUp.TradeNo,
		"trade_no":     "provider-epay-no-mark",
		"trade_status": epay.StatusTradeSuccess,
		"money":        "10.00",
	}, operation_setting.EpayKey)
	query := url.Values{}
	for key, value := range params {
		query.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/user/pay/notify?"+query.Encode(), nil)

	EpayNotify(c)

	select {
	case <-sender.done:
		t.Fatal("Epay replay without a failed mark sent a GA4 event")
	case <-time.After(100 * time.Millisecond):
	}
	if _, err := model.GetAnalyticsEventMark(ga4SubjectTypeTopUp, topUp.Id, ga4DeliveryKeyTopUp); err == nil {
		t.Fatal("Epay replay without a prior delivery created an analytics mark")
	}
}

func TestStripeTopUpReplayReclaimsFailedAnalyticsMark(t *testing.T) {
	setupGA4PaymentEventTestDB(t)
	sender, topUp := setupFailedTopUpAnalyticsReplay(t, model.PaymentProviderStripe, "stripe-replay")
	event := stripeWebhookTestEvent(stripe.EventTypeCheckoutSessionCompleted, `{
		"amount_total": 1000,
		"currency": "usd"
	}`)

	fulfillOrder(context.Background(), event, topUp.TradeNo, "cus_replay", "127.0.0.1")

	waitForPaymentGA4Send(t, sender.done)
	mark := waitForTopUpGA4MarkTerminal(t, topUp.Id)
	if mark.Status != model.AnalyticsEventStatusSent {
		t.Fatalf("Stripe replay mark status = %q, want sent", mark.Status)
	}
}

func TestCreemTopUpReplayReclaimsFailedAnalyticsMark(t *testing.T) {
	setupGA4PaymentEventTestDB(t)
	sender, topUp := setupFailedTopUpAnalyticsReplay(t, model.PaymentProviderCreem, "creem-replay")

	event := &CreemWebhookEvent{}
	event.Object.RequestId = topUp.TradeNo
	event.Object.Order.Id = "creem-order-replay"
	event.Object.Order.Status = "paid"
	event.Object.Order.Type = "onetime"
	event.Object.Order.Currency = "USD"
	event.Object.Order.AmountPaid = 1000
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/user/creem/webhook", nil)

	handleCheckoutCompleted(c, event)

	waitForPaymentGA4Send(t, sender.done)
	mark := waitForTopUpGA4MarkTerminal(t, topUp.Id)
	if mark.Status != model.AnalyticsEventStatusSent {
		t.Fatalf("Creem replay mark status = %q, want sent", mark.Status)
	}
}

func setupFailedTopUpAnalyticsReplay(t *testing.T, provider string, tradeNo string) (*paymentGA4CaptureSender, *model.TopUp) {
	t.Helper()
	topUp := &model.TopUp{
		UserId:          901,
		Amount:          2,
		Money:           10,
		TradeNo:         tradeNo,
		PaymentMethod:   provider,
		PaymentProvider: provider,
		Status:          common.TopUpStatusSuccess,
		CreateTime:      common.GetTimestamp(),
		CompleteTime:    common.GetTimestamp(),
	}
	if err := model.DB.Create(topUp).Error; err != nil {
		t.Fatalf("create completed top-up: %v", err)
	}
	markID := beginGA4TopUpDelivery(topUp.Id)
	if markID <= 0 || !model.MarkAnalyticsEventFailed(markID) {
		t.Fatalf("seed failed top-up analytics mark")
	}

	sender := &paymentGA4CaptureSender{done: make(chan struct{}, 1)}
	restore := analytics.ConfigureForTest(analytics.Config{
		Enabled:       true,
		MeasurementID: "G-TEST",
		APISecret:     "secret",
		HashSalt:      "salt",
		Timeout:       50 * time.Millisecond,
		Endpoint:      "https://example.test/mp/collect",
	}, sender)
	t.Cleanup(restore)
	return sender, topUp
}

func TestGA4StripeRenewalPurchaseRequiresCompletedSubscriptionCycle(t *testing.T) {
	setupGA4PaymentEventTestDB(t)
	sender := &paymentGA4CaptureSender{done: make(chan struct{}, 1)}
	restore := analytics.ConfigureForTest(analytics.Config{
		Enabled:       true,
		MeasurementID: "G-TEST",
		APISecret:     "secret",
		HashSalt:      "salt",
		Timeout:       50 * time.Millisecond,
		Endpoint:      "https://example.test/mp/collect",
	}, sender)
	t.Cleanup(restore)

	input := model.StripeSubscriptionInvoiceInput{
		InvoiceId:     "in_helper_renewal",
		BillingReason: "subscription_cycle",
		AmountPaid:    1000,
		Currency:      "USD",
	}
	result := model.StripeSubscriptionInvoiceResult{
		Created:                 false,
		Status:                  model.StripeInvoiceResultDuplicate,
		UserId:                  901,
		InvoiceRecordId:         77,
		RenewalPurchaseEligible: true,
	}

	trackGA4StripeRenewalPurchase(input, result)
	waitForPaymentGA4Send(t, sender.done)
	mark := waitForPaymentGA4MarkTerminal(t, ga4SubjectTypeStripeInvoice, result.InvoiceRecordId)
	if mark.Status != model.AnalyticsEventStatusSent {
		t.Fatalf("stripe renewal mark status = %q, want sent", mark.Status)
	}
	if bodies := sender.snapshotBodies(); len(bodies) != 1 {
		t.Fatalf("stripe renewal sent %d requests, want 1", len(bodies))
	}
}

func TestGA4StripeRenewalPurchaseRejectsIneligibleResults(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*model.StripeSubscriptionInvoiceInput, *model.StripeSubscriptionInvoiceResult)
	}{
		{name: "not eligible", mutate: func(_ *model.StripeSubscriptionInvoiceInput, result *model.StripeSubscriptionInvoiceResult) {
			result.RenewalPurchaseEligible = false
		}},
		{name: "missing invoice record", mutate: func(_ *model.StripeSubscriptionInvoiceInput, result *model.StripeSubscriptionInvoiceResult) {
			result.InvoiceRecordId = 0
		}},
		{name: "missing user", mutate: func(_ *model.StripeSubscriptionInvoiceInput, result *model.StripeSubscriptionInvoiceResult) {
			result.UserId = 0
		}},
		{name: "missing invoice id", mutate: func(input *model.StripeSubscriptionInvoiceInput, _ *model.StripeSubscriptionInvoiceResult) {
			input.InvoiceId = ""
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupGA4PaymentEventTestDB(t)
			db := model.DB
			sender := &paymentGA4CaptureSender{done: make(chan struct{}, 1)}
			restore := analytics.ConfigureForTest(analytics.Config{
				Enabled:       true,
				MeasurementID: "G-TEST",
				APISecret:     "secret",
				HashSalt:      "salt",
				Timeout:       50 * time.Millisecond,
				Endpoint:      "https://example.test/mp/collect",
			}, sender)
			t.Cleanup(restore)

			input := model.StripeSubscriptionInvoiceInput{
				InvoiceId:     "in_ineligible",
				BillingReason: "subscription_cycle",
				AmountPaid:    1000,
				Currency:      "USD",
			}
			result := model.StripeSubscriptionInvoiceResult{
				Created:                 false,
				Status:                  model.StripeInvoiceResultDuplicate,
				UserId:                  901,
				InvoiceRecordId:         77,
				RenewalPurchaseEligible: true,
			}
			tt.mutate(&input, &result)

			trackGA4StripeRenewalPurchase(input, result)

			var markCount int64
			if err := db.Model(&model.AnalyticsEventMark{}).Count(&markCount).Error; err != nil {
				t.Fatalf("count analytics marks: %v", err)
			}
			if markCount != 0 {
				t.Fatalf("ineligible renewal created %d analytics marks", markCount)
			}
			if bodies := sender.snapshotBodies(); len(bodies) != 0 {
				t.Fatalf("ineligible renewal sent %d requests", len(bodies))
			}
		})
	}
}

func waitForPaymentGA4Send(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for payment GA4 send")
	}
}

func waitForPaymentGA4MarkTerminal(t *testing.T, subjectType string, subjectID int) *model.AnalyticsEventMark {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	var last *model.AnalyticsEventMark
	for time.Now().Before(deadline) {
		mark, err := model.GetAnalyticsEventMark(subjectType, subjectID, ga4EventPurchase)
		if err == nil {
			last = mark
			if mark.Status == model.AnalyticsEventStatusSent || mark.Status == model.AnalyticsEventStatusFailed {
				return mark
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if last != nil {
		return last
	}
	mark, err := model.GetAnalyticsEventMark(subjectType, subjectID, ga4EventPurchase)
	if err != nil {
		t.Fatalf("get payment GA4 mark: %v", err)
	}
	return mark
}

func waitForTopUpGA4MarkTerminal(t *testing.T, topUpID int) *model.AnalyticsEventMark {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	var last *model.AnalyticsEventMark
	for time.Now().Before(deadline) {
		mark, err := model.GetAnalyticsEventMark(ga4SubjectTypeTopUp, topUpID, ga4DeliveryKeyTopUp)
		if err == nil {
			last = mark
			if mark.Status == model.AnalyticsEventStatusSent || mark.Status == model.AnalyticsEventStatusFailed {
				return mark
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if last != nil {
		return last
	}
	mark, err := model.GetAnalyticsEventMark(ga4SubjectTypeTopUp, topUpID, ga4DeliveryKeyTopUp)
	if err != nil {
		t.Fatalf("get top-up GA4 mark: %v", err)
	}
	return mark
}

func TestGA4OrderAttributionRoundTripsSanitizedBrowserContext(t *testing.T) {
	encoded := encodeGA4Attribution(analytics.SignUpAttribution{
		ClientID:     "123.456",
		SessionID:    "789",
		PageLocation: "https://lizh.ai/wallet?utm_source=google&token=secret",
		PageReferrer: "https://mail.example/reset?email=user@example.com&token=secret",
		Source:       "google",
		Medium:       "cpc",
		Campaign:     "launch",
		GCLID:        "click-123",
	})
	if encoded == "" {
		t.Fatalf("encoded attribution is empty")
	}
	for _, forbidden := range []string{"secret", "user@example.com", "email=", "token="} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("encoded attribution leaked %q: %s", forbidden, encoded)
		}
	}

	decoded := decodeGA4Attribution(encoded)
	if decoded.ClientID != "123.456" || decoded.SessionID != "789" {
		t.Fatalf("browser identifiers did not round trip: %#v", decoded)
	}
	if decoded.PageLocation != "https://lizh.ai/wallet?utm_source=google" {
		t.Fatalf("unexpected page location: %q", decoded.PageLocation)
	}
	if decoded.PageReferrer != "https://mail.example/reset" {
		t.Fatalf("unexpected page referrer: %q", decoded.PageReferrer)
	}
}

func TestGA4OrderAttributionRejectsMalformedStoredJSON(t *testing.T) {
	if got := decodeGA4Attribution(`{"client_id":`); got != (analytics.SignUpAttribution{}) {
		t.Fatalf("malformed attribution decoded as %#v", got)
	}
}

func setupGA4PaymentEventTestDB(t *testing.T) {
	t.Helper()
	oldDB := model.DB
	oldSQLite := common.UsingSQLite
	oldMySQL := common.UsingMySQL
	oldPostgreSQL := common.UsingPostgreSQL

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open analytics test database: %v", err)
	}
	if err := db.AutoMigrate(&model.AnalyticsEventMark{}, &model.TopUp{}, &model.SubscriptionOrder{}); err != nil {
		t.Fatalf("migrate analytics event marks: %v", err)
	}
	model.DB = db
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	t.Cleanup(func() {
		model.DB = oldDB
		common.UsingSQLite = oldSQLite
		common.UsingMySQL = oldMySQL
		common.UsingPostgreSQL = oldPostgreSQL
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
}
