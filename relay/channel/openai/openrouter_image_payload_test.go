package openai_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	openai "github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenRouterQwenImageEditTranslatesReferenceToUpstreamImageURL(t *testing.T) {
	req := service.OpenRouterImageRequest{
		Model:       "Qwen-Image-Edit",
		Prompt:      "make it red",
		AspectRatio: "1:1",
		InputReferences: []service.OpenRouterImageInputReference{{
			ImageURL: &service.OpenRouterImageURL{URL: "https://example.com/input.png"},
		}},
	}

	payload := convertOpenRouterImageToOpenAIUpstreamPayload(t, req)

	require.Equal(t, "Qwen-Image-Edit", stringValue(t, payload, "model"))
	require.Equal(t, "make it red", stringValue(t, payload, "prompt"))
	require.Equal(t, "https://example.com/input.png", stringValue(t, payload, "image_url"))
	require.Equal(t, "1024x1024", stringValue(t, payload, "size"))
	require.NotContains(t, payload, "image")
	require.NotContains(t, payload, "extra_fields")
	require.NotContains(t, payload, "n")
}

func TestOpenRouterQwen2512RejectsUnsupportedTwoKSquareSize(t *testing.T) {
	req := service.OpenRouterImageRequest{
		Model:       "Qwen-Image-2512-2k",
		Prompt:      "lychee icon",
		AspectRatio: "1:1",
		Resolution:  "2k",
	}

	_, _, err := service.ConvertOpenRouterImageRequest(req)

	require.Error(t, err)
	require.Contains(t, err.Error(), "Qwen-Image-2512")
	require.Contains(t, err.Error(), "2048x2048")
}

func TestOpenRouterQwen2512KeepsSupportedTwoKPortraitSize(t *testing.T) {
	req := service.OpenRouterImageRequest{
		Model:       "Qwen-Image-2512-2k",
		Prompt:      "vertical lychee poster",
		AspectRatio: "9:16",
		Resolution:  "2k",
	}

	imageReq, mode, err := service.ConvertOpenRouterImageRequest(req)

	require.NoError(t, err)
	require.Equal(t, service.OpenRouterImageGenerationMode, mode)
	require.Equal(t, "1152x2048", imageReq.Size)
}

func convertOpenRouterImageToOpenAIUpstreamPayload(t *testing.T, req service.OpenRouterImageRequest) map[string]json.RawMessage {
	t.Helper()

	imageReq, mode, err := service.ConvertOpenRouterImageRequest(req)
	require.NoError(t, err)
	require.Equal(t, service.OpenRouterImageEditMode, mode)

	controllerBody, err := common.Marshal(imageReq)
	require.NoError(t, err)

	var relayReq dto.ImageRequest
	require.NoError(t, common.Unmarshal(controllerBody, &relayReq))

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(controllerBody))
	c.Request.Header.Set("Content-Type", "application/json")

	converted, err := (&openai.Adaptor{}).ConvertImageRequest(c, &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeImagesEdits,
	}, relayReq)
	require.NoError(t, err)

	raw, err := common.Marshal(converted)
	require.NoError(t, err)

	var payload map[string]json.RawMessage
	require.NoError(t, common.Unmarshal(raw, &payload))
	return payload
}

func stringValue(t *testing.T, payload map[string]json.RawMessage, key string) string {
	t.Helper()

	raw, ok := payload[key]
	require.Truef(t, ok, "missing key %q in payload", key)
	var value string
	require.NoError(t, common.Unmarshal(raw, &value))
	return value
}
