package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stripe/stripe-go/v81"
)

func assertStringPtr(t *testing.T, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("expected %q, got nil", want)
	}
	if *got != want {
		t.Fatalf("expected %q, got %q", want, *got)
	}
}

func assertNilStringPtr(t *testing.T, got *string, name string) {
	t.Helper()
	if got != nil {
		t.Fatalf("expected %s to be nil, got %q", name, *got)
	}
}

func TestBuildStripeSubscriptionCheckoutParamsWithoutCustomer(t *testing.T) {
	system_setting.ServerAddress = "https://lizh.ai"

	params := buildStripeSubscriptionCheckoutParams("sub_ref_1", "", "user@example.com", "price_123")

	assertStringPtr(t, params.Mode, string(stripe.CheckoutSessionModeSubscription))
	assertStringPtr(t, params.CustomerEmail, "user@example.com")
	assertNilStringPtr(t, params.Customer, "Customer")
	assertNilStringPtr(t, params.CustomerCreation, "CustomerCreation")
	assertStringPtr(t, params.LineItems[0].Price, "price_123")
}

func TestBuildStripeSubscriptionCheckoutParamsWithCustomer(t *testing.T) {
	system_setting.ServerAddress = "https://lizh.ai"

	params := buildStripeSubscriptionCheckoutParams("sub_ref_2", "cus_123", "user@example.com", "price_456")

	assertStringPtr(t, params.Mode, string(stripe.CheckoutSessionModeSubscription))
	assertStringPtr(t, params.Customer, "cus_123")
	assertNilStringPtr(t, params.CustomerEmail, "CustomerEmail")
	assertNilStringPtr(t, params.CustomerCreation, "CustomerCreation")
	assertStringPtr(t, params.LineItems[0].Price, "price_456")
}

func TestBuildStripePaymentCheckoutParamsKeepsCustomerCreation(t *testing.T) {
	setting.StripePriceId = "price_topup"
	setting.StripePromotionCodesEnabled = true

	params := buildStripePaymentCheckoutParams("topup_ref_1", "", "user@example.com", 3, "https://lizh.ai/success", "https://lizh.ai/cancel")

	assertStringPtr(t, params.Mode, string(stripe.CheckoutSessionModePayment))
	assertStringPtr(t, params.CustomerEmail, "user@example.com")
	assertStringPtr(t, params.CustomerCreation, string(stripe.CheckoutSessionCustomerCreationAlways))
	assertNilStringPtr(t, params.Customer, "Customer")
	assertStringPtr(t, params.LineItems[0].Price, "price_topup")
	if params.LineItems[0].Quantity == nil || *params.LineItems[0].Quantity != 3 {
		t.Fatalf("expected quantity 3, got %#v", params.LineItems[0].Quantity)
	}
}
