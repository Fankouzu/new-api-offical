package controller

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
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
	))

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db
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
}

func TestStripeInvoicePaymentFailedWebhookDoesNotCreateSubscription(t *testing.T) {
	db := setupStripeWebhookControllerTestDB(t)
	seedStripeWebhookRenewalData(t, db)

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
}
