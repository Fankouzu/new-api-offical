package controller

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/analytics"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v81"
	"gorm.io/gorm"
)

func setupStripeWebhookControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(2)

	model.DB = db
	model.LOG_DB = db

	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Log{},
		&model.SubscriptionPlan{},
		&model.UserSubscription{},
		&model.StripeSubscriptionInvoice{},
		&model.AnalyticsEventMark{},
	))

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db
}

func setupStripeWebhookAnalytics(t *testing.T) *paymentGA4CaptureSender {
	t.Helper()
	sender := &paymentGA4CaptureSender{done: make(chan struct{}, 4)}
	restore := analytics.ConfigureForTest(analytics.Config{
		Enabled:       true,
		MeasurementID: "G-TEST",
		APISecret:     "secret",
		HashSalt:      "salt",
		Timeout:       50 * time.Millisecond,
		Endpoint:      "https://example.test/mp/collect",
	}, sender)
	t.Cleanup(restore)
	return sender
}

func seedStripeWebhookRenewalData(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Create(&model.User{
		Id:             901,
		Username:       "stripe_webhook_user",
		Status:         common.UserStatusEnabled,
		Group:          "default",
		StripeCustomer: "cus_webhook",
	}).Error)
	require.NoError(t, db.Create(&model.SubscriptionPlan{
		Id:            902,
		Title:         "Webhook Plan",
		PriceAmount:   10,
		Currency:      "USD",
		DurationUnit:  model.SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TotalAmount:   9000,
		StripePriceId: "price_webhook",
	}).Error)
}

func stripeWebhookTestEvent(eventType stripe.EventType, raw string) stripe.Event {
	return stripe.Event{
		ID:   "evt_test",
		Type: eventType,
		Data: &stripe.EventData{
			Raw: []byte(raw),
		},
	}
}

func TestStripeInvoicePaidWebhookCreatesRenewalSubscription(t *testing.T) {
	db := setupStripeWebhookControllerTestDB(t)
	seedStripeWebhookRenewalData(t, db)
	sender := setupStripeWebhookAnalytics(t)

	event := stripeWebhookTestEvent(stripe.EventTypeInvoicePaid, `{
		"id": "in_webhook_renewal",
		"object": "invoice",
		"customer": "cus_webhook",
		"subscription": "sub_webhook",
		"billing_reason": "subscription_cycle",
		"amount_paid": 1000,
		"currency": "usd",
		"customer_email": "billing@example.com",
		"customer_name": "Billing Person",
		"lines": {
			"object": "list",
			"data": [
				{
					"id": "il_webhook",
					"object": "line_item",
					"price": {
						"id": "price_webhook",
						"object": "price"
					}
				}
			]
		}
	}`)

	handleStripeInvoicePaid(context.Background(), event, "127.0.0.1")
	waitForPaymentGA4Send(t, sender.done)

	var subCount int64
	require.NoError(t, db.Model(&model.UserSubscription{}).Where("user_id = ?", 901).Count(&subCount).Error)
	assert.Equal(t, int64(1), subCount)

	var invoice model.StripeSubscriptionInvoice
	require.NoError(t, db.Where("invoice_id = ?", "in_webhook_renewal").First(&invoice).Error)
	assert.Equal(t, model.StripeInvoiceResultProcessed, invoice.Status)
	assert.Equal(t, 901, invoice.UserId)
	assert.Equal(t, 902, invoice.PlanId)
	assert.NotContains(t, invoice.Payload, "billing@example.com")
	assert.NotContains(t, invoice.Payload, "Billing Person")
	assert.Contains(t, invoice.Payload, "in_webhook_renewal")

	mark := waitForPaymentGA4MarkTerminal(t, ga4SubjectTypeStripeInvoice, invoice.Id)
	require.Equal(t, model.AnalyticsEventStatusSent, mark.Status)

	bodies := sender.snapshotBodies()
	require.Len(t, bodies, 1)
	var payload struct {
		Events []struct {
			Name   string         `json:"name"`
			Params map[string]any `json:"params"`
		} `json:"events"`
	}
	require.NoError(t, common.Unmarshal([]byte(bodies[0]), &payload))
	require.Len(t, payload.Events, 1)
	assert.Equal(t, ga4EventPurchase, payload.Events[0].Name)
	params := payload.Events[0].Params
	assert.Equal(t, "in_webhook_renewal", params["transaction_id"])
	assert.Equal(t, float64(10), params["value"])
	assert.Equal(t, "USD", params["currency"])
	assert.Equal(t, model.PaymentProviderStripe, params["payment_provider"])
	assert.Equal(t, model.PaymentProviderStripe, params["payment_method"])
	assert.Equal(t, ga4ItemTypeSubscription, params["item_type"])
	assert.Equal(t, float64(901), params["user_id"])
	items, ok := params["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	item, ok := items[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, ga4ItemTypeSubscription, item["item_id"])
	assert.NotContains(t, bodies[0], "billing@example.com")
	assert.NotContains(t, bodies[0], "Billing Person")

	handleStripeInvoicePaid(context.Background(), event, "127.0.0.1")

	var replaySubCount int64
	require.NoError(t, db.Model(&model.UserSubscription{}).Where("user_id = ?", 901).Count(&replaySubCount).Error)
	assert.Equal(t, int64(1), replaySubCount)
	var invoiceCount int64
	require.NoError(t, db.Model(&model.StripeSubscriptionInvoice{}).Count(&invoiceCount).Error)
	assert.Equal(t, int64(1), invoiceCount)
	var markCount int64
	require.NoError(t, db.Model(&model.AnalyticsEventMark{}).Where("subject_type = ? AND event_name = ?", ga4SubjectTypeStripeInvoice, ga4EventPurchase).Count(&markCount).Error)
	assert.Equal(t, int64(1), markCount)
	require.Equal(t, model.AnalyticsEventStatusSent, waitForPaymentGA4MarkTerminal(t, ga4SubjectTypeStripeInvoice, invoice.Id).Status)
	assert.Len(t, sender.snapshotBodies(), 1)
}

func TestStripeInvoicePaidWebhookSkipsInitialInvoicePurchase(t *testing.T) {
	db := setupStripeWebhookControllerTestDB(t)
	seedStripeWebhookRenewalData(t, db)
	sender := setupStripeWebhookAnalytics(t)

	event := stripeWebhookTestEvent(stripe.EventTypeInvoicePaid, `{
		"id": "in_webhook_initial",
		"object": "invoice",
		"customer": "cus_webhook",
		"subscription": "sub_webhook",
		"billing_reason": "subscription_create",
		"amount_paid": 1000,
		"currency": "usd",
		"lines": {
			"object": "list",
			"data": [{
				"id": "il_webhook_initial",
				"object": "line_item",
				"price": {"id": "price_webhook", "object": "price"}
			}]
		}
	}`)

	handleStripeInvoicePaid(context.Background(), event, "127.0.0.1")
	handleStripeInvoicePaid(context.Background(), event, "127.0.0.1")

	var invoice model.StripeSubscriptionInvoice
	require.NoError(t, db.Where("invoice_id = ?", "in_webhook_initial").First(&invoice).Error)
	assert.Equal(t, model.StripeInvoiceResultInitialInvoice, invoice.Status)
	var subCount int64
	require.NoError(t, db.Model(&model.UserSubscription{}).Count(&subCount).Error)
	assert.Zero(t, subCount)
	var markCount int64
	require.NoError(t, db.Model(&model.AnalyticsEventMark{}).Where("subject_type = ?", ga4SubjectTypeStripeInvoice).Count(&markCount).Error)
	assert.Zero(t, markCount)
	assert.Empty(t, sender.snapshotBodies())
	var invoiceCount int64
	require.NoError(t, db.Model(&model.StripeSubscriptionInvoice{}).Count(&invoiceCount).Error)
	assert.Equal(t, int64(1), invoiceCount)
}

func TestStripeInvoicePaidWebhookSkipsIgnoredInvoicePurchase(t *testing.T) {
	db := setupStripeWebhookControllerTestDB(t)
	seedStripeWebhookRenewalData(t, db)
	sender := setupStripeWebhookAnalytics(t)

	event := stripeWebhookTestEvent(stripe.EventTypeInvoicePaid, `{
		"id": "in_webhook_ignored",
		"object": "invoice",
		"customer": "cus_webhook",
		"subscription": "sub_webhook",
		"billing_reason": "subscription_update",
		"amount_paid": 1000,
		"currency": "usd",
		"lines": {
			"object": "list",
			"data": [{
				"id": "il_webhook_ignored",
				"object": "line_item",
				"price": {"id": "price_webhook", "object": "price"}
			}]
		}
	}`)

	handleStripeInvoicePaid(context.Background(), event, "127.0.0.1")
	handleStripeInvoicePaid(context.Background(), event, "127.0.0.1")

	var invoice model.StripeSubscriptionInvoice
	require.NoError(t, db.Where("invoice_id = ?", "in_webhook_ignored").First(&invoice).Error)
	assert.Equal(t, model.StripeInvoiceResultIgnored, invoice.Status)
	var subCount int64
	require.NoError(t, db.Model(&model.UserSubscription{}).Count(&subCount).Error)
	assert.Zero(t, subCount)
	var markCount int64
	require.NoError(t, db.Model(&model.AnalyticsEventMark{}).Where("subject_type = ?", ga4SubjectTypeStripeInvoice).Count(&markCount).Error)
	assert.Zero(t, markCount)
	assert.Empty(t, sender.snapshotBodies())
	var invoiceCount int64
	require.NoError(t, db.Model(&model.StripeSubscriptionInvoice{}).Count(&invoiceCount).Error)
	assert.Equal(t, int64(1), invoiceCount)
}

func TestStripeInvoicePaidWebhookKeepsRenewalWhenGA4Fails(t *testing.T) {
	db := setupStripeWebhookControllerTestDB(t)
	seedStripeWebhookRenewalData(t, db)
	sender := setupStripeWebhookAnalytics(t)
	sender.setStatus(http.StatusBadRequest)

	event := stripeWebhookTestEvent(stripe.EventTypeInvoicePaid, `{
		"id": "in_webhook_ga4_failed",
		"object": "invoice",
		"customer": "cus_webhook",
		"subscription": "sub_webhook",
		"billing_reason": "subscription_cycle",
		"amount_paid": 1000,
		"currency": "usd",
		"lines": {
			"object": "list",
			"data": [{
				"id": "il_webhook_ga4_failed",
				"object": "line_item",
				"price": {"id": "price_webhook", "object": "price"}
			}]
		}
	}`)

	handleStripeInvoicePaid(context.Background(), event, "127.0.0.1")
	waitForPaymentGA4Send(t, sender.done)

	var invoice model.StripeSubscriptionInvoice
	require.NoError(t, db.Where("invoice_id = ?", "in_webhook_ga4_failed").First(&invoice).Error)
	assert.Equal(t, model.StripeInvoiceResultProcessed, invoice.Status)
	var subCount int64
	require.NoError(t, db.Model(&model.UserSubscription{}).Where("user_id = ?", 901).Count(&subCount).Error)
	assert.Equal(t, int64(1), subCount)
	mark := waitForPaymentGA4MarkTerminal(t, ga4SubjectTypeStripeInvoice, invoice.Id)
	assert.Equal(t, model.AnalyticsEventStatusFailed, mark.Status)
	assert.Len(t, sender.snapshotBodies(), 1)

	sender.setStatus(http.StatusNoContent)
	handleStripeInvoicePaid(context.Background(), event, "127.0.0.1")
	waitForPaymentGA4Send(t, sender.done)

	retriedMark := waitForPaymentGA4MarkTerminal(t, ga4SubjectTypeStripeInvoice, invoice.Id)
	assert.Equal(t, mark.Id, retriedMark.Id)
	assert.Equal(t, model.AnalyticsEventStatusSent, retriedMark.Status)
	assert.Len(t, sender.snapshotBodies(), 2)
	var invoiceCount int64
	require.NoError(t, db.Model(&model.StripeSubscriptionInvoice{}).Count(&invoiceCount).Error)
	assert.Equal(t, int64(1), invoiceCount)
	require.NoError(t, db.Model(&model.UserSubscription{}).Where("user_id = ?", 901).Count(&subCount).Error)
	assert.Equal(t, int64(1), subCount)
	var markCount int64
	require.NoError(t, db.Model(&model.AnalyticsEventMark{}).Where("subject_type = ?", ga4SubjectTypeStripeInvoice).Count(&markCount).Error)
	assert.Equal(t, int64(1), markCount)

	handleStripeInvoicePaid(context.Background(), event, "127.0.0.1")
	assert.Len(t, sender.snapshotBodies(), 2)
}

func TestStripeInvoicePaymentFailedWebhookDoesNotCreateSubscription(t *testing.T) {
	db := setupStripeWebhookControllerTestDB(t)
	seedStripeWebhookRenewalData(t, db)
	sender := setupStripeWebhookAnalytics(t)

	event := stripeWebhookTestEvent(stripe.EventTypeInvoicePaymentFailed, `{
		"id": "in_failed",
		"object": "invoice",
		"customer": "cus_webhook",
		"subscription": "sub_webhook",
		"billing_reason": "subscription_cycle",
		"amount_paid": 0,
		"currency": "usd",
		"attempt_count": 2
	}`)

	handleStripeInvoicePaymentFailed(context.Background(), event, "127.0.0.1")

	var subCount int64
	require.NoError(t, db.Model(&model.UserSubscription{}).Where("user_id = ?", 901).Count(&subCount).Error)
	assert.Equal(t, int64(0), subCount)

	var invoiceCount int64
	require.NoError(t, db.Model(&model.StripeSubscriptionInvoice{}).Count(&invoiceCount).Error)
	assert.Equal(t, int64(0), invoiceCount)
	var markCount int64
	require.NoError(t, db.Model(&model.AnalyticsEventMark{}).Where("subject_type = ?", ga4SubjectTypeStripeInvoice).Count(&markCount).Error)
	assert.Zero(t, markCount)
	assert.Empty(t, sender.snapshotBodies())
}
