package relay

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type nilUsageAdaptor struct{}

func (a *nilUsageAdaptor) Init(info *relaycommon.RelayInfo) {}

func (a *nilUsageAdaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return "", nil
}

func (a *nilUsageAdaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	return nil
}

func (a *nilUsageAdaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	return request, nil
}

func (a *nilUsageAdaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, nil
}

func (a *nilUsageAdaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return nil, nil
}

func (a *nilUsageAdaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	return nil, nil
}

func (a *nilUsageAdaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	return nil, nil
}

func (a *nilUsageAdaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return nil, nil
}

func (a *nilUsageAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"id":"ok"}`)),
	}, nil
}

func (a *nilUsageAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (any, *types.NewAPIError) {
	return nil, nil
}

func (a *nilUsageAdaptor) GetModelList() []string { return nil }

func (a *nilUsageAdaptor) GetChannelName() string { return "nil-usage-test" }

func (a *nilUsageAdaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	return nil, nil
}

func (a *nilUsageAdaptor) ConvertGeminiRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeminiChatRequest) (any, error) {
	return nil, nil
}

func TestTextHelperReturnsBadResponseWhenAdaptorUsageIsNil(t *testing.T) {
	gin.SetMode(gin.TestMode)

	previous := getAdaptor
	getAdaptor = func(apiType int) channel.Adaptor {
		return &nilUsageAdaptor{}
	}
	t.Cleanup(func() {
		getAdaptor = previous
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"doubao-seedance-1-5-pro-251215","messages":[{"role":"user","content":"metadata only"}]}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	common.SetContextKey(ctx, constant.ContextKeyChannelType, constant.ChannelTypePingXingShiJie)
	common.SetContextKey(ctx, constant.ContextKeyChannelId, 5)
	common.SetContextKey(ctx, constant.ContextKeyChannelBaseUrl, "https://api.pingxingshijie.cn")
	common.SetContextKey(ctx, constant.ContextKeyOriginalModel, "doubao-seedance-1-5-pro-251215")
	common.SetContextKey(ctx, constant.ContextKeyRequestStartTime, time.Now())

	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeChatCompletions,
		RelayFormat:     types.RelayFormatOpenAI,
		OriginModelName: "doubao-seedance-1-5-pro-251215",
		RequestURLPath:  "/v1/chat/completions",
		Request: &dto.GeneralOpenAIRequest{
			Model: "doubao-seedance-1-5-pro-251215",
			Messages: []dto.Message{
				{Role: "user"},
			},
		},
	}

	err := TextHelper(ctx, info)
	if err == nil {
		t.Fatal("TextHelper returned nil error for nil usage")
	}
	if err.GetErrorCode() != types.ErrorCodeBadResponse {
		t.Fatalf("error code = %s, want %s", err.GetErrorCode(), types.ErrorCodeBadResponse)
	}
}

func TestValidateTextUsageReturnsBadResponseWhenRelayInfoMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	usage, err := validateTextUsage(ctx, nil, nil)
	if usage != nil {
		t.Fatalf("usage = %#v, want nil", usage)
	}
	if err == nil {
		t.Fatal("validateTextUsage returned nil error")
	}
	if err.GetErrorCode() != types.ErrorCodeBadResponse {
		t.Fatalf("error code = %s, want %s", err.GetErrorCode(), types.ErrorCodeBadResponse)
	}
}
