package model

import "testing"

func TestTryMarkAnalyticsEventOnlyMarksOnce(t *testing.T) {
	truncateTables(t)

	if !TryMarkAnalyticsEvent("token", 123, "first_api_call") {
		t.Fatalf("first mark should succeed")
	}
	if TryMarkAnalyticsEvent("token", 123, "first_api_call") {
		t.Fatalf("duplicate in-flight mark should be skipped")
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

func TestAnalyticsEventMarkStatusRetriesFailedAndSuppressesSent(t *testing.T) {
	truncateTables(t)

	id := BeginAnalyticsEventDelivery("token", 123, "first_api_call")
	if id <= 0 {
		t.Fatalf("first delivery should begin")
	}
	mark, err := GetAnalyticsEventMark("token", 123, "first_api_call")
	if err != nil {
		t.Fatalf("get mark: %v", err)
	}
	if mark.Status != AnalyticsEventStatusSending {
		t.Fatalf("status = %q, want sending", mark.Status)
	}
	if !MarkAnalyticsEventFailed(id) {
		t.Fatalf("mark failed should update status")
	}

	retryID := BeginAnalyticsEventDelivery("token", 123, "first_api_call")
	if retryID != id {
		t.Fatalf("retry id = %d, want existing id %d", retryID, id)
	}
	if !MarkAnalyticsEventSent(id) {
		t.Fatalf("mark sent should update status")
	}
	if BeginAnalyticsEventDelivery("token", 123, "first_api_call") != 0 {
		t.Fatalf("sent event must suppress duplicate delivery")
	}

	mark, err = GetAnalyticsEventMark("token", 123, "first_api_call")
	if err != nil {
		t.Fatalf("get mark after sent: %v", err)
	}
	if mark.Status != AnalyticsEventStatusSent {
		t.Fatalf("status = %q, want sent", mark.Status)
	}
}

func TestAnalyticsEventMarkSuppressesFreshSendingDelivery(t *testing.T) {
	truncateTables(t)

	id := BeginAnalyticsEventDelivery("token", 123, "first_api_call")
	if id <= 0 {
		t.Fatalf("first delivery should begin")
	}

	if BeginAnalyticsEventDelivery("token", 123, "first_api_call") != 0 {
		t.Fatalf("fresh sending delivery should suppress duplicate")
	}
}

func TestAnalyticsEventMarkRetriesStaleSendingDelivery(t *testing.T) {
	truncateTables(t)

	id := BeginAnalyticsEventDelivery("token", 123, "first_api_call")
	if id <= 0 {
		t.Fatalf("first delivery should begin")
	}
	staleUpdatedAt := currentAnalyticsEventTimestamp() - analyticsEventSendingTimeoutSeconds - 1
	if err := DB.Model(&AnalyticsEventMark{}).Where("id = ?", id).Update("updated_at", staleUpdatedAt).Error; err != nil {
		t.Fatalf("make delivery stale: %v", err)
	}

	retryID := BeginAnalyticsEventDelivery("token", 123, "first_api_call")
	if retryID != id {
		t.Fatalf("stale sending retry id = %d, want existing id %d", retryID, id)
	}
}
