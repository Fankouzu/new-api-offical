package controller

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/analytics"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

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
