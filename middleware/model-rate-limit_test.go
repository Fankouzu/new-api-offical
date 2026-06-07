package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestModelRequestRateLimitReturnsStructured429(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalDuration := setting.ModelRequestRateLimitDurationMinutes
	setting.ModelRequestRateLimitDurationMinutes = 2
	t.Cleanup(func() {
		setting.ModelRequestRateLimitDurationMinutes = originalDuration
	})

	handler := memoryRateLimitHandler(120, 1, 1)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("id", 1001)
		c.Next()
	})
	router.Use(handler)
	router.GET("/v1/chat/completions", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	first := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	router.ServeHTTP(first, req)
	require.Equal(t, http.StatusOK, first.Code)

	second := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	router.ServeHTTP(second, req)

	require.Equal(t, http.StatusTooManyRequests, second.Code)
	require.Equal(t, "120", second.Header().Get("Retry-After"))

	var payload struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, common.Unmarshal(second.Body.Bytes(), &payload))
	require.Contains(t, payload.Error.Message, "请求")
	require.Equal(t, "new_api_error", payload.Error.Type)
	require.NotEmpty(t, payload.Error.Code)
}
