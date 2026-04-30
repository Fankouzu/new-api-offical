package relay

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
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
