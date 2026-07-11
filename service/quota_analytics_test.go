package service

import (
	"bytes"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service/analytics"
	"gorm.io/gorm"
)

type tokenQueryCapture struct {
	count   int
	selects [][]string
}

func captureTokenQueries(t *testing.T) *tokenQueryCapture {
	t.Helper()
	capture := &tokenQueryCapture{}
	callbackName := "test:capture_token_attribution_queries:" + t.Name()
	err := model.DB.Callback().Query().After("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		table := tx.Statement.Table
		if table == "" && tx.Statement.Schema != nil {
			table = tx.Statement.Schema.Table
		}
		if table != "tokens" {
			return
		}
		capture.count++
		capture.selects = append(capture.selects, append([]string(nil), tx.Statement.Selects...))
	})
	if err != nil {
		t.Fatalf("register token query callback: %v", err)
	}
	t.Cleanup(func() {
		_ = model.DB.Callback().Query().Remove(callbackName)
	})
	return capture
}

func seedTokenAttribution(t *testing.T, tokenID int, raw string) {
	t.Helper()
	token := model.Token{
		Id:                   tokenID,
		UserId:               42,
		Key:                  "stored-token-key",
		Name:                 "attributed-token",
		Status:               1,
		UnlimitedQuota:       true,
		AnalyticsAttribution: raw,
	}
	if err := model.DB.Create(&token).Error; err != nil {
		t.Fatalf("create token: %v", err)
	}
}

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
		statuses: []int{
			http.StatusInternalServerError,
			http.StatusInternalServerError,
			http.StatusInternalServerError,
			http.StatusNoContent,
		},
		done: make(chan struct{}, 4),
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
	if sender.requests != 3 {
		t.Fatalf("requests after failed send = %d, want 3", sender.requests)
	}

	trackFirstAPICallIfNeeded(info, 100)
	waitForGA4Send(t, sender.done)
	mark = waitForAnalyticsMarkStatus(t, 7, model.AnalyticsEventStatusSent)
	if mark.Status != model.AnalyticsEventStatusSent {
		t.Fatalf("status after retry success = %q, want sent", mark.Status)
	}
	if sender.requests != 4 {
		t.Fatalf("requests after retry = %d, want 4", sender.requests)
	}
	if !strings.Contains(sender.bodies[3], `"name":"first_api_call"`) {
		t.Fatalf("first API payload should use new event name: %s", sender.bodies[3])
	}
	if !strings.Contains(sender.bodies[3], `"endpoint":"/v1/chat/completions"`) ||
		!strings.Contains(sender.bodies[3], `"status_code":200`) ||
		!strings.Contains(sender.bodies[3], `"model":"gpt-test"`) {
		t.Fatalf("first API payload missing endpoint/status/model: %s", sender.bodies[3])
	}
	if strings.Contains(sender.bodies[3], "token-key") {
		t.Fatalf("first API payload leaked raw token key: %s", sender.bodies[3])
	}

	trackFirstAPICallIfNeeded(info, 100)
	if sender.requests != 4 {
		t.Fatalf("sent event should suppress duplicate send, got %d requests", sender.requests)
	}
}

func TestTrackFirstAPICallDoesNotReadTokenAfterSentMark(t *testing.T) {
	truncate(t)
	if err := model.DB.Create(&model.AnalyticsEventMark{
		SubjectType: "token",
		SubjectId:   17,
		EventName:   "first_api_call",
		Status:      model.AnalyticsEventStatusSent,
		CreatedAt:   1,
		UpdatedAt:   1,
	}).Error; err != nil {
		t.Fatalf("create sent analytics mark: %v", err)
	}
	capture := captureTokenQueries(t)
	restore := analytics.ConfigureForTest(analytics.Config{
		Enabled:       true,
		MeasurementID: "G-TEST",
		APISecret:     "secret",
		HashSalt:      "salt",
		Timeout:       50 * time.Millisecond,
		Endpoint:      "https://example.test/mp/collect",
	}, &ga4TestSender{})
	defer restore()

	trackFirstAPICallIfNeeded(&relaycommon.RelayInfo{
		UserId:          42,
		TokenId:         17,
		TokenKey:        "token-key",
		OriginModelName: "gpt-test",
		RequestURLPath:  "/v1/chat/completions",
	}, 100)

	if capture.count != 0 {
		t.Fatalf("sent mark triggered %d token attribution queries", capture.count)
	}
}

func TestTrackFirstAPICallUsesPersistedTokenAttribution(t *testing.T) {
	truncate(t)
	seedTokenAttribution(t, 7, `{"client_id":"123.456","session_id":"789","page_location":"https://lizh.ai/console/token?utm_source=google","source":"google","campaign":"launch","gclid":"click","unknown_secret":"do-not-send"}`)
	capture := captureTokenQueries(t)

	sender := &ga4TestSender{done: make(chan struct{}, 1)}
	restore := analytics.ConfigureForTest(analytics.Config{
		Enabled:       true,
		MeasurementID: "G-TEST",
		APISecret:     "secret",
		HashSalt:      "salt",
		Timeout:       50 * time.Millisecond,
		Endpoint:      "https://example.test/mp/collect",
	}, sender)
	defer restore()

	trackFirstAPICallIfNeeded(&relaycommon.RelayInfo{
		UserId:          42,
		TokenId:         7,
		TokenKey:        "token-key-secret",
		OriginModelName: "gpt-test",
		RequestURLPath:  "/v1/chat/completions?trace=1",
	}, 100)
	waitForGA4Send(t, sender.done)
	mark := waitForAnalyticsMarkStatus(t, 7, model.AnalyticsEventStatusSent)
	if mark.Status != model.AnalyticsEventStatusSent {
		t.Fatalf("status after attributed send = %q, want sent", mark.Status)
	}

	var payload struct {
		ClientID string `json:"client_id"`
		Events   []struct {
			Params map[string]any `json:"params"`
		} `json:"events"`
	}
	if err := common.Unmarshal([]byte(sender.bodies[0]), &payload); err != nil {
		t.Fatalf("decode first API payload: %v", err)
	}
	if payload.ClientID != "123.456" {
		t.Fatalf("client id = %q, want persisted attribution client id", payload.ClientID)
	}
	params := payload.Events[0].Params
	if params["session_id"] != float64(789) || params["engagement_time_msec"] != float64(1) || params["source"] != "google" || params["campaign"] != "launch" || params["gclid"] != "click" {
		t.Fatalf("persisted token attribution missing from first API payload: %#v", params)
	}
	if capture.count != 1 {
		t.Fatalf("token attribution query count = %d, want 1", capture.count)
	}
	if len(capture.selects) != 1 || !reflect.DeepEqual(capture.selects[0], []string{"analytics_attribution"}) {
		t.Fatalf("token attribution query selects = %#v, want analytics_attribution only", capture.selects)
	}
	if strings.Contains(sender.bodies[0], "token-key-secret") || strings.Contains(sender.bodies[0], "stored-token-key") || strings.Contains(sender.bodies[0], "do-not-send") || strings.Contains(sender.bodies[0], "unknown_secret") {
		t.Fatalf("first API payload leaked raw token key: %s", sender.bodies[0])
	}
}

func TestTrackFirstAPICallFallsBackWhenTokenAttributionUnavailable(t *testing.T) {
	for _, tt := range []struct {
		name    string
		tokenID int
		raw     string
		seed    bool
	}{
		{name: "missing token", tokenID: 8},
		{name: "malformed attribution", tokenID: 9, raw: `{"client_id":`, seed: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			truncate(t)
			if tt.seed {
				seedTokenAttribution(t, tt.tokenID, tt.raw)
			}
			sender := &ga4TestSender{done: make(chan struct{}, 1)}
			restore := analytics.ConfigureForTest(analytics.Config{
				Enabled:       true,
				MeasurementID: "G-TEST",
				APISecret:     "secret",
				HashSalt:      "salt",
				Timeout:       50 * time.Millisecond,
				Endpoint:      "https://example.test/mp/collect",
			}, sender)
			defer restore()

			trackFirstAPICallIfNeeded(&relaycommon.RelayInfo{
				UserId:          42,
				TokenId:         tt.tokenID,
				TokenKey:        "fallback-token-key",
				OriginModelName: "gpt-test",
				RequestURLPath:  "/v1/chat/completions",
			}, 100)
			waitForGA4Send(t, sender.done)

			var payload struct {
				ClientID string `json:"client_id"`
			}
			if err := common.Unmarshal([]byte(sender.bodies[0]), &payload); err != nil {
				t.Fatalf("decode fallback payload: %v", err)
			}
			if !strings.HasPrefix(payload.ClientID, "server.") {
				t.Fatalf("client id = %q, want server fallback", payload.ClientID)
			}
			if strings.Contains(sender.bodies[0], "fallback-token-key") {
				t.Fatalf("fallback payload leaked raw token key: %s", sender.bodies[0])
			}
		})
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
