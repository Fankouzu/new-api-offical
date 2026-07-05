package service

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service/analytics"
)

type ga4TestSender struct {
	statuses []int
	requests int
	bodies   []string
	done     chan struct{}
}

func (s *ga4TestSender) Do(req *http.Request) (*http.Response, error) {
	s.requests++
	body, _ := io.ReadAll(req.Body)
	s.bodies = append(s.bodies, string(body))
	status := http.StatusNoContent
	if len(s.statuses) > 0 {
		status = s.statuses[0]
		s.statuses = s.statuses[1:]
	}
	if s.done != nil {
		s.done <- struct{}{}
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(nil)),
	}, nil
}

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
		RequestURLPath:  "/v1/chat/completions?trace=1",
	}, 100)

	var count int64
	if err := model.DB.Model(&model.AnalyticsEventMark{}).Count(&count).Error; err != nil {
		t.Fatalf("count analytics event marks: %v", err)
	}
	if count != 0 {
		t.Fatalf("disabled GA4 should not create first_api_call mark, got %d", count)
	}
}

func TestTrackFirstAPICallRetriesFailedSendAndSuppressesSentDuplicate(t *testing.T) {
	truncate(t)
	sender := &ga4TestSender{
		statuses: []int{http.StatusInternalServerError, http.StatusNoContent},
		done:     make(chan struct{}, 2),
	}
	restore := analytics.ConfigureForTest(analytics.Config{
		Enabled:       true,
		MeasurementID: "G-TEST",
		APISecret:     "secret",
		HashSalt:      "salt",
		Timeout:       50 * time.Millisecond,
		Endpoint:      "https://example.test/mp/collect",
	}, sender)
	defer restore()

	info := &relaycommon.RelayInfo{
		UserId:          42,
		TokenId:         7,
		TokenKey:        "token-key",
		OriginModelName: "gpt-test",
		RequestURLPath:  "/v1/chat/completions?trace=1",
	}

	trackFirstAPICallIfNeeded(info, 100)
	waitForGA4Send(t, sender.done)
	mark := waitForAnalyticsMarkStatus(t, 7, model.AnalyticsEventStatusFailed)
	if mark.Status != model.AnalyticsEventStatusFailed {
		t.Fatalf("status after failed send = %q, want failed", mark.Status)
	}
	if sender.requests != 1 {
		t.Fatalf("requests after failed send = %d, want 1", sender.requests)
	}

	trackFirstAPICallIfNeeded(info, 100)
	waitForGA4Send(t, sender.done)
	mark = waitForAnalyticsMarkStatus(t, 7, model.AnalyticsEventStatusSent)
	if mark.Status != model.AnalyticsEventStatusSent {
		t.Fatalf("status after retry success = %q, want sent", mark.Status)
	}
	if sender.requests != 2 {
		t.Fatalf("requests after retry = %d, want 2", sender.requests)
	}
	if !strings.Contains(sender.bodies[1], `"name":"first_api_call"`) {
		t.Fatalf("first API payload should use new event name: %s", sender.bodies[1])
	}
	if !strings.Contains(sender.bodies[1], `"endpoint":"/v1/chat/completions"`) ||
		!strings.Contains(sender.bodies[1], `"status_code":200`) ||
		!strings.Contains(sender.bodies[1], `"model":"gpt-test"`) {
		t.Fatalf("first API payload missing endpoint/status/model: %s", sender.bodies[1])
	}
	if strings.Contains(sender.bodies[1], "token-key") {
		t.Fatalf("first API payload leaked raw token key: %s", sender.bodies[1])
	}

	trackFirstAPICallIfNeeded(info, 100)
	if sender.requests != 2 {
		t.Fatalf("sent event should suppress duplicate send, got %d requests", sender.requests)
	}
}

func requireAnalyticsMark(t *testing.T, tokenID int) *model.AnalyticsEventMark {
	t.Helper()
	mark, err := model.GetAnalyticsEventMark("token", tokenID, "first_api_call")
	if err != nil {
		t.Fatalf("get analytics mark: %v", err)
	}
	return mark
}

func waitForGA4Send(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for GA4 send")
	}
}

func waitForAnalyticsMarkStatus(t *testing.T, tokenID int, status string) *model.AnalyticsEventMark {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		mark := requireAnalyticsMark(t, tokenID)
		if mark.Status == status {
			return mark
		}
		time.Sleep(10 * time.Millisecond)
	}
	return requireAnalyticsMark(t, tokenID)
}
