package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/analytics"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

const (
	ga4SubjectTypeTopUp       = "top_up"
	ga4SubjectTypePurchase    = "purchase"
	ga4EventTopUp             = "top_up"
	ga4EventPurchase          = "purchase"
	ga4ItemTypeTopUp          = "top_up"
	ga4ItemTypeSubscription   = "subscription"
	defaultPaymentCurrencyUSD = "USD"
)

func trackGA4TopUpSuccess(c *gin.Context, tradeNo string) {
	trackGA4TopUpSuccessWithCurrency(c, tradeNo, "")
}

func trackGA4TopUpSuccessWithCurrency(c *gin.Context, tradeNo string, currency string) {
	if !analytics.Enabled() || tradeNo == "" {
		return
	}
	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil || topUp.Status != common.TopUpStatusSuccess {
		return
	}
	markID := model.BeginAnalyticsEventDelivery(ga4SubjectTypeTopUp, topUp.Id, ga4EventTopUp)
	if markID <= 0 {
		return
	}
	analytics.TrackTopUpWithResult(c, topUp.UserId, analytics.PurchaseAttribution{
		TradeNo:         topUp.TradeNo,
		Value:           topUp.Money,
		Currency:        resolveGA4PaymentCurrency(topUp.PaymentProvider, currency),
		PaymentProvider: topUp.PaymentProvider,
		PaymentMethod:   topUp.PaymentMethod,
		ItemType:        ga4ItemTypeTopUp,
		QuotaAmount:     topUp.Amount,
	}, trackAnalyticsMarkResult(markID))
}

func trackGA4PurchaseSuccess(c *gin.Context, tradeNo string) {
	trackGA4PurchaseSuccessWithCurrency(c, tradeNo, "")
}

func trackGA4PurchaseSuccessWithCurrency(c *gin.Context, tradeNo string, currency string) {
	if !analytics.Enabled() || tradeNo == "" {
		return
	}
	order := model.GetSubscriptionOrderByTradeNo(tradeNo)
	if order == nil || order.Status != common.TopUpStatusSuccess {
		return
	}
	markID := model.BeginAnalyticsEventDelivery(ga4SubjectTypePurchase, order.Id, ga4EventPurchase)
	if markID <= 0 {
		return
	}
	analytics.TrackPurchaseWithResult(c, order.UserId, analytics.PurchaseAttribution{
		TradeNo:         order.TradeNo,
		Value:           order.Money,
		Currency:        resolveGA4PaymentCurrency(order.PaymentProvider, currency),
		PaymentProvider: order.PaymentProvider,
		PaymentMethod:   order.PaymentMethod,
		ItemType:        ga4ItemTypeSubscription,
	}, trackAnalyticsMarkResult(markID))
}

func trackAnalyticsMarkResult(markID int) func(error) {
	return func(err error) {
		if err != nil {
			model.MarkAnalyticsEventFailed(markID)
			return
		}
		model.MarkAnalyticsEventSent(markID)
	}
}

func resolveGA4PaymentCurrency(paymentProvider string, currency string) string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency != "" {
		return currency
	}
	switch paymentProvider {
	case model.PaymentProviderBinancePay:
		return strings.ToUpper(strings.TrimSpace(setting.BinancePayCurrency))
	case model.PaymentProviderWaffo:
		return strings.ToUpper(strings.TrimSpace(setting.WaffoCurrency))
	case model.PaymentProviderWaffoPancake:
		return strings.ToUpper(strings.TrimSpace(setting.WaffoPancakeCurrency))
	default:
		return resolveGA4DisplayCurrency()
	}
}

func resolveGA4DisplayCurrency() string {
	switch operation_setting.GetQuotaDisplayType() {
	case operation_setting.QuotaDisplayTypeCNY:
		return "CNY"
	default:
		return defaultPaymentCurrencyUSD
	}
}
