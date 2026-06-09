package model

import "testing"

func TestTryMarkAnalyticsEventOnlyMarksOnce(t *testing.T) {
	truncateTables(t)

	if !TryMarkAnalyticsEvent("token", 123, "first_api_call") {
		t.Fatalf("first mark should succeed")
	}
	if TryMarkAnalyticsEvent("token", 123, "first_api_call") {
		t.Fatalf("duplicate mark should be skipped")
	}
	if !TryMarkAnalyticsEvent("token", 124, "first_api_call") {
		t.Fatalf("different subject should be marked independently")
	}
}

func TestTryMarkAnalyticsEventRejectsInvalidInput(t *testing.T) {
	truncateTables(t)

	if TryMarkAnalyticsEvent("", 123, "first_api_call") {
		t.Fatalf("empty subject type should be rejected")
	}
	if TryMarkAnalyticsEvent("token", 0, "first_api_call") {
		t.Fatalf("empty subject id should be rejected")
	}
	if TryMarkAnalyticsEvent("token", 123, "") {
		t.Fatalf("empty event name should be rejected")
	}
}
