package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func insertStripeInvoiceUserForTest(t *testing.T, id int, stripeCustomer string) {
	t.Helper()
	user := &User{
		Id:             id,
		Username:       "stripe_invoice_user",
		Status:         common.UserStatusEnabled,
		Group:          "default",
		StripeCustomer: stripeCustomer,
	}
	require.NoError(t, DB.Create(user).Error)
}

func insertStripeInvoicePlanForTest(t *testing.T, id int, priceId string) *SubscriptionPlan {
	t.Helper()
	plan := &SubscriptionPlan{
		Id:            id,
		Title:         "Stripe Renewal Plan",
		PriceAmount:   12.34,
		Currency:      "USD",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TotalAmount:   5000,
		StripePriceId: priceId,
	}
	require.NoError(t, DB.Create(plan).Error)
	return plan
}

func countStripeInvoiceSubscriptionsForTest(t *testing.T, userId int) int64 {
	t.Helper()
	var count int64
	require.NoError(t, DB.Model(&UserSubscription{}).Where("user_id = ?", userId).Count(&count).Error)
	return count
}

func countStripeInvoiceRecordsForTest(t *testing.T) int64 {
	t.Helper()
	var count int64
	require.NoError(t, DB.Model(&StripeSubscriptionInvoice{}).Count(&count).Error)
	return count
}

func TestCompleteStripeSubscriptionInvoiceCreatesRenewalSubscription(t *testing.T) {
	truncateTables(t)

	insertStripeInvoiceUserForTest(t, 701, "cus_renewal")
	plan := insertStripeInvoicePlanForTest(t, 801, "price_renewal")

	result, err := CompleteStripeSubscriptionInvoice(StripeSubscriptionInvoiceInput{
		EventId:        "evt_invoice_paid",
		EventType:      "invoice.paid",
		InvoiceId:      "in_renewal_1",
		SubscriptionId: "sub_renewal",
		CustomerId:     "cus_renewal",
		PriceId:        "price_renewal",
		BillingReason:  "subscription_cycle",
		AmountPaid:     1234,
		Currency:       "usd",
		Payload:        `{"id":"in_renewal_1"}`,
	})

	require.NoError(t, err)
	assert.True(t, result.Created)
	assert.Equal(t, 701, result.UserId)
	assert.Equal(t, plan.Id, result.PlanId)
	assert.Equal(t, int64(1), countStripeInvoiceSubscriptionsForTest(t, 701))

	var sub UserSubscription
	require.NoError(t, DB.Where("user_id = ?", 701).First(&sub).Error)
	assert.Equal(t, "stripe_invoice", sub.Source)
	assert.Equal(t, plan.TotalAmount, sub.AmountTotal)

	assert.Equal(t, int64(1), countStripeInvoiceRecordsForTest(t))
}

func TestCompleteStripeSubscriptionInvoiceIsIdempotentByInvoiceId(t *testing.T) {
	truncateTables(t)

	insertStripeInvoiceUserForTest(t, 702, "cus_replay")
	insertStripeInvoicePlanForTest(t, 802, "price_replay")

	input := StripeSubscriptionInvoiceInput{
		EventId:        "evt_invoice_paid_first",
		EventType:      "invoice.paid",
		InvoiceId:      "in_replay_1",
		SubscriptionId: "sub_replay",
		CustomerId:     "cus_replay",
		PriceId:        "price_replay",
		BillingReason:  "subscription_cycle",
		AmountPaid:     2345,
		Currency:       "usd",
		Payload:        `{"id":"in_replay_1"}`,
	}

	first, err := CompleteStripeSubscriptionInvoice(input)
	require.NoError(t, err)
	assert.True(t, first.Created)

	input.EventId = "evt_invoice_paid_replay"
	second, err := CompleteStripeSubscriptionInvoice(input)
	require.NoError(t, err)
	assert.False(t, second.Created)
	assert.Equal(t, int64(1), countStripeInvoiceSubscriptionsForTest(t, 702))
	assert.Equal(t, int64(1), countStripeInvoiceRecordsForTest(t))
}

func TestCompleteStripeSubscriptionInvoiceBypassesPurchaseLimitForRenewal(t *testing.T) {
	truncateTables(t)

	insertStripeInvoiceUserForTest(t, 705, "cus_limit_renewal")
	plan := insertStripeInvoicePlanForTest(t, 805, "price_limit_renewal")
	plan.MaxPurchasePerUser = 1
	require.NoError(t, DB.Save(plan).Error)
	require.NoError(t, DB.Create(&UserSubscription{
		UserId:      705,
		PlanId:      plan.Id,
		AmountTotal: plan.TotalAmount,
		StartTime:   common.GetTimestamp() - 60,
		EndTime:     common.GetTimestamp() + 3600,
		Status:      "active",
		Source:      "order",
	}).Error)

	result, err := CompleteStripeSubscriptionInvoice(StripeSubscriptionInvoiceInput{
		EventId:        "evt_limit_renewal",
		EventType:      "invoice.paid",
		InvoiceId:      "in_limit_renewal",
		SubscriptionId: "sub_limit_renewal",
		CustomerId:     "cus_limit_renewal",
		PriceId:        "price_limit_renewal",
		BillingReason:  "subscription_cycle",
		AmountPaid:     3456,
		Currency:       "usd",
		Payload:        `{"id":"in_limit_renewal"}`,
	})

	require.NoError(t, err)
	assert.True(t, result.Created)
	assert.Equal(t, int64(2), countStripeInvoiceSubscriptionsForTest(t, 705))
}

func TestCompleteStripeSubscriptionInvoiceSkipsMissingPriceOrCustomer(t *testing.T) {
	testCases := []struct {
		name       string
		input      StripeSubscriptionInvoiceInput
		seedUser   bool
		seedPlan   bool
		expectCode string
	}{
		{
			name: "missing price",
			input: StripeSubscriptionInvoiceInput{
				EventId:        "evt_missing_price",
				EventType:      "invoice.paid",
				InvoiceId:      "in_missing_price",
				SubscriptionId: "sub_missing_price",
				CustomerId:     "cus_missing_price",
				PriceId:        "price_missing",
				BillingReason:  "subscription_cycle",
				AmountPaid:     111,
				Currency:       "usd",
				Payload:        `{"id":"in_missing_price"}`,
			},
			seedUser:   true,
			expectCode: StripeInvoiceResultMissingPlan,
		},
		{
			name: "missing customer",
			input: StripeSubscriptionInvoiceInput{
				EventId:        "evt_missing_customer",
				EventType:      "invoice.paid",
				InvoiceId:      "in_missing_customer",
				SubscriptionId: "sub_missing_customer",
				CustomerId:     "cus_missing_customer",
				PriceId:        "price_missing_customer",
				BillingReason:  "subscription_cycle",
				AmountPaid:     222,
				Currency:       "usd",
				Payload:        `{"id":"in_missing_customer"}`,
			},
			seedPlan:   true,
			expectCode: StripeInvoiceResultMissingUser,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			truncateTables(t)
			if tc.seedUser {
				insertStripeInvoiceUserForTest(t, 703, tc.input.CustomerId)
			}
			if tc.seedPlan {
				insertStripeInvoicePlanForTest(t, 803, tc.input.PriceId)
			}

			result, err := CompleteStripeSubscriptionInvoice(tc.input)

			require.NoError(t, err)
			assert.False(t, result.Created)
			assert.Equal(t, tc.expectCode, result.Status)
			assert.Equal(t, int64(0), countStripeInvoiceSubscriptionsForTest(t, 703))
			assert.Equal(t, int64(1), countStripeInvoiceRecordsForTest(t))
		})
	}
}

func TestCompleteStripeSubscriptionInvoiceSkipsInitialSubscriptionCreate(t *testing.T) {
	truncateTables(t)

	insertStripeInvoiceUserForTest(t, 704, "cus_initial")
	insertStripeInvoicePlanForTest(t, 804, "price_initial")

	result, err := CompleteStripeSubscriptionInvoice(StripeSubscriptionInvoiceInput{
		EventId:        "evt_initial_invoice",
		EventType:      "invoice.paid",
		InvoiceId:      "in_initial",
		SubscriptionId: "sub_initial",
		CustomerId:     "cus_initial",
		PriceId:        "price_initial",
		BillingReason:  "subscription_create",
		AmountPaid:     333,
		Currency:       "usd",
		Payload:        `{"id":"in_initial"}`,
	})

	require.NoError(t, err)
	assert.False(t, result.Created)
	assert.Equal(t, StripeInvoiceResultInitialInvoice, result.Status)
	assert.Equal(t, int64(0), countStripeInvoiceSubscriptionsForTest(t, 704))
	assert.Equal(t, int64(1), countStripeInvoiceRecordsForTest(t))
}

func TestCompleteStripeSubscriptionInvoiceIgnoresSubscriptionUpdateInvoices(t *testing.T) {
	truncateTables(t)

	insertStripeInvoiceUserForTest(t, 706, "cus_update")
	insertStripeInvoicePlanForTest(t, 806, "price_update")

	result, err := CompleteStripeSubscriptionInvoice(StripeSubscriptionInvoiceInput{
		EventId:        "evt_update_invoice",
		EventType:      "invoice.paid",
		InvoiceId:      "in_update",
		SubscriptionId: "sub_update",
		CustomerId:     "cus_update",
		PriceId:        "price_update",
		BillingReason:  "subscription_update",
		AmountPaid:     444,
		Currency:       "usd",
		Payload:        `{"id":"in_update"}`,
	})

	require.NoError(t, err)
	assert.False(t, result.Created)
	assert.Equal(t, StripeInvoiceResultIgnored, result.Status)
	assert.Equal(t, int64(0), countStripeInvoiceSubscriptionsForTest(t, 706))
	assert.Equal(t, int64(1), countStripeInvoiceRecordsForTest(t))
}
