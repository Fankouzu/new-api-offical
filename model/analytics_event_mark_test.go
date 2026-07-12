package model

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

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

func TestRedeemWithAuditDetailsReturnsStableTransactionIdentity(t *testing.T) {
	truncateTables(t)

	user := &User{Id: 42, Username: "redeem_user", Quota: 0, Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(user).Error)
	redemption := &Redemption{
		Key:         "raw-redemption-code",
		Name:        "test voucher",
		Status:      common.RedemptionCodeStatusEnabled,
		Quota:       int(10 * common.QuotaPerUnit),
		CreatedTime: common.GetTimestamp(),
	}
	require.NoError(t, redemption.Insert())

	details, err := RedeemWithAuditDetails(redemption.Key, user.Id, "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, redemption.Id, details.ID)
	require.Equal(t, redemption.Quota, details.Quota)
	require.Equal(t, "redemption:"+strconv.Itoa(redemption.Id), details.TransactionID())

	var stored User
	require.NoError(t, DB.First(&stored, "id = ?", user.Id).Error)
	require.Equal(t, redemption.Quota, stored.Quota)
}
