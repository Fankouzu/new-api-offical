package helper

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetAndValidOpenAIImageEditJSONRequiresPrompt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/images/edits",
		strings.NewReader(`{"model":"qwen-image","image_url":"https://example.com/ref.jpg"}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	_, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)

	require.EqualError(t, err, "prompt is required")
}

func TestGetAndValidOpenAIImageEditJSONDoesNotDefaultOptionalFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/images/edits",
		strings.NewReader(`{"model":"qwen-image","prompt":"edit","image_url":"https://example.com/ref.jpg"}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	request, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)

	require.NoError(t, err)
	require.Nil(t, request.N)
	require.Empty(t, request.Size)
	require.Empty(t, request.Quality)
}

func TestGetAndValidOpenAIImageEditMultipartRequiresModelAndPrompt(t *testing.T) {
	t.Run("missing model", func(t *testing.T) {
		c := newMultipartImageEditContext(t, map[string]string{"prompt": "edit"}, nil)

		_, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)

		require.EqualError(t, err, "model is required")
	})

	t.Run("missing prompt", func(t *testing.T) {
		c := newMultipartImageEditContext(t, map[string]string{"model": "qwen-image"}, nil)

		_, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)

		require.EqualError(t, err, "prompt is required")
	})
}

func newMultipartImageEditContext(t *testing.T, fields map[string]string, files map[string]string) *gin.Context {
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
