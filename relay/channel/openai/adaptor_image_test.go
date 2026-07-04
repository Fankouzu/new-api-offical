package openai

import (
	"bytes"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertImageEditJSONPassesThroughOpenAICompatFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	var request dto.ImageRequest
	require.NoError(t, common.Unmarshal([]byte(`{
		"model":"qwen-image",
		"prompt":"edit",
		"image_url":"https://example.com/ref.jpg",
		"seed":1234
	}`), &request))

	converted, err := (&Adaptor{}).ConvertImageRequest(c, &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeImagesEdits,
	}, request)

	require.NoError(t, err)
	body, err := common.Marshal(converted)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, common.Unmarshal(body, &payload))
	require.Equal(t, "qwen-image", payload["model"])
	require.Equal(t, "edit", payload["prompt"])
	require.Equal(t, "https://example.com/ref.jpg", payload["image_url"])
	require.Equal(t, float64(1234), payload["seed"])
}

func TestConvertImageEditMultipartPassesThroughFieldsWithoutImageFile(t *testing.T) {
	c := newOpenAIMultipartImageEditContext(t, map[string]string{
		"model":     "qwen-image",
		"prompt":    "edit",
		"image_url": "https://example.com/ref.jpg",
		"seed":      "1234",
	}, nil)

	converted, err := (&Adaptor{}).ConvertImageRequest(c, &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeImagesEdits,
	}, dto.ImageRequest{Model: "qwen-image", Prompt: "edit"})

	require.NoError(t, err)
	buffer, ok := converted.(*bytes.Buffer)
	require.Truef(t, ok, "converted request type = %T", converted)

	form := readConvertedMultipartForm(t, c.Request.Header.Get("Content-Type"), buffer.Bytes())
	require.Equal(t, []string{"qwen-image"}, form.Value["model"])
	require.Equal(t, []string{"edit"}, form.Value["prompt"])
	require.Equal(t, []string{"https://example.com/ref.jpg"}, form.Value["image_url"])
	require.Equal(t, []string{"1234"}, form.Value["seed"])
	require.Empty(t, form.File)
}

func TestConvertImageEditMultipartPassesThroughFileFields(t *testing.T) {
	c := newOpenAIMultipartImageEditContext(t, map[string]string{
		"model":  "qwen-image",
		"prompt": "edit",
	}, map[string]string{
		"image": "reference.jpg",
		"mask":  "mask.png",
	})

	converted, err := (&Adaptor{}).ConvertImageRequest(c, &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeImagesEdits,
	}, dto.ImageRequest{Model: "qwen-image", Prompt: "edit"})

	require.NoError(t, err)
	buffer, ok := converted.(*bytes.Buffer)
	require.Truef(t, ok, "converted request type = %T", converted)

	form := readConvertedMultipartForm(t, c.Request.Header.Get("Content-Type"), buffer.Bytes())
	require.Len(t, form.File["image"], 1)
	require.Equal(t, "reference.jpg", form.File["image"][0].Filename)
	require.Len(t, form.File["mask"], 1)
	require.Equal(t, "mask.png", form.File["mask"][0].Filename)
}

func newOpenAIMultipartImageEditContext(t *testing.T, fields map[string]string, files map[string]string) *gin.Context {
	t.Helper()

	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		require.NoError(t, writer.WriteField(key, value))
	}
	for fieldName, filename := range files {
		part, err := writer.CreateFormFile(fieldName, filename)
		require.NoError(t, err)
		_, err = part.Write([]byte("image-bytes"))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	return c
}

func readConvertedMultipartForm(t *testing.T, contentType string, body []byte) *multipart.Form {
	t.Helper()

	_, params, err := mime.ParseMediaType(contentType)
	require.NoError(t, err)
	reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	form, err := reader.ReadForm(32 << 20)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, form.RemoveAll())
	})
	return form
}
