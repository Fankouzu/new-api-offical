package relay

import (
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func TestApplyOtherRatiosToQuotaMultipliesBeforeTruncating(t *testing.T) {
	ratios := map[string]float64{
		"resolution":  51.0 / 46.0,
		"video_input": 31.0 / 51.0,
	}

	if got := applyOtherRatiosToQuota(46, ratios); got != 31 {
		t.Fatalf("quota: got %d want 31", got)
	}
}

func TestRecalcQuotaFromRatiosMultipliesBeforeTruncating(t *testing.T) {
	info := &relaycommon.RelayInfo{
		PriceData: types.PriceData{
			Quota: 46,
		},
	}
	ratios := map[string]float64{
		"resolution":  51.0 / 46.0,
		"video_input": 31.0 / 51.0,
	}

	if got := recalcQuotaFromRatios(info, ratios); got != 31 {
		t.Fatalf("quota: got %d want 31", got)
	}
}

func TestRewriteLocalTaskContentURLUsesRequestHost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Request = httptest.NewRequest("GET", "http://127.0.0.1:3001/v1/images/generations/task_123", nil)

	got := rewriteLocalTaskContentURL(c, "http://localhost:3000/v1/videos/task_123/content")
	if got != "http://127.0.0.1:3001/v1/videos/task_123/content" {
		t.Fatalf("url = %q", got)
	}
}
