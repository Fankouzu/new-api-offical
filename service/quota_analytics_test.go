package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service/analytics"
)

func TestTrackFirstAPICallDoesNotMarkWhenGA4Disabled(t *testing.T) {
	truncate(t)
	restore := analytics.ConfigureForTest(analytics.Config{
		Enabled:       true,
		MeasurementID: "G-TEST",
		APISecret:     "",
		HashSalt:      "salt",
		Timeout:       50 * time.Millisecond,
		Endpoint:      "https://example.test/mp/collect",
	}, nil)
	defer restore()

	trackFirstAPICallIfNeeded(&relaycommon.RelayInfo{
		UserId:          42,
		TokenId:         7,
		TokenKey:        "token-key",
		OriginModelName: "gpt-test",
	}, 100)

	var count int64
	if err := model.DB.Model(&model.AnalyticsEventMark{}).Count(&count).Error; err != nil {
		t.Fatalf("count analytics event marks: %v", err)
	}
	if count != 0 {
		t.Fatalf("disabled GA4 should not create first_api_call mark, got %d", count)
	}
}
