package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type panicRefundSpy struct {
	refunded    bool
	refundCount int
}

func (s *panicRefundSpy) Settle(actualQuota int) error { return nil }

func (s *panicRefundSpy) Refund(c *gin.Context) {
	s.refunded = true
	s.refundCount++
}

func (s *panicRefundSpy) NeedsRefund() bool { return true }

func (s *panicRefundSpy) GetPreConsumedQuota() int { return 82000 }

func (s *panicRefundSpy) Reserve(targetQuota int) error { return nil }

func TestHandleRelayPanicRefundsPreConsumedBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var errorLog bytes.Buffer
	previousErrorWriter := gin.DefaultErrorWriter
	common.LogWriterMu.Lock()
	gin.DefaultErrorWriter = &errorLog
	common.LogWriterMu.Unlock()
	t.Cleanup(func() {
		common.LogWriterMu.Lock()
		gin.DefaultErrorWriter = previousErrorWriter
		common.LogWriterMu.Unlock()
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	billing := &panicRefundSpy{}
	info := &relaycommon.RelayInfo{
		RequestId:             "req-panic",
		RequestURLPath:        "/v1/chat/completions",
		OriginModelName:       "doubao-seedance-1-5-pro-251215",
		UserId:                73,
		TokenId:               44,
		ChannelMeta:           &relaycommon.ChannelMeta{ChannelId: 5, ChannelType: 58, UpstreamModelName: "doubao-seedance-1-5-pro-251215"},
		UsingGroup:            "default",
		FinalPreConsumedQuota: 82000,
		Billing:               billing,
	}

	err := handleRelayPanic(ctx, info, "req-panic", "boom")
	if err == nil {
		t.Fatal("handleRelayPanic returned nil")
	}
	if err.GetErrorCode() != types.ErrorCodeBadResponse {
		t.Fatalf("error code = %s, want %s", err.GetErrorCode(), types.ErrorCodeBadResponse)
	}
	if !billing.refunded {
		t.Fatal("expected panic handler to refund pre-consumed billing")
	}
	logLine := errorLog.String()
	for _, want := range []string{
		"relay panic recovered:",
		"request_id=req-panic",
		"path=/v1/chat/completions",
		"model=doubao-seedance-1-5-pro-251215",
		"upstream_model=doubao-seedance-1-5-pro-251215",
		"user_id=73",
		"token_id=44",
		"channel_id=5",
		"channel_type=58",
		"group=default",
		"panic_summary=redacted non-runtime panic value",
		"stack=",
	} {
		if !strings.Contains(logLine, want) {
			t.Fatalf("panic log missing %q:\n%s", want, logLine)
		}
	}
	if strings.Contains(logLine, "boom") {
		t.Fatalf("panic log leaked non-runtime panic value:\n%s", logLine)
	}
}

func TestRelayPanicRecoveryRefundsWhenInnerErrorDeferSeesNoAPIError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	billing := &panicRefundSpy{}
	info := &relaycommon.RelayInfo{
		RequestId:             "req-defer-order",
		RequestURLPath:        "/v1/chat/completions",
		OriginModelName:       "doubao-seedance-1-5-pro-251215",
		UserId:                73,
		TokenId:               44,
		ChannelMeta:           &relaycommon.ChannelMeta{ChannelId: 5, ChannelType: 58, UpstreamModelName: "doubao-seedance-1-5-pro-251215"},
		UsingGroup:            "default",
		FinalPreConsumedQuota: 82000,
		Billing:               billing,
	}

	var newAPIError *types.NewAPIError
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				newAPIError = handleRelayPanic(ctx, info, "req-defer-order", recovered)
			}
		}()
		defer func() {
			if newAPIError != nil && info.Billing != nil {
				info.Billing.Refund(ctx)
			}
		}()

		panic("downstream panic after preconsume")
	}()

	if newAPIError == nil {
		t.Fatal("expected panic recovery to create API error")
	}
	if newAPIError.StatusCode != http.StatusBadGateway {
		t.Fatalf("status code = %d, want %d", newAPIError.StatusCode, http.StatusBadGateway)
	}
	if billing.refundCount != 1 {
		t.Fatalf("refund count = %d, want 1", billing.refundCount)
	}
}
