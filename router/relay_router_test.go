package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRelayRouterRegistersAsyncImageGenerationRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetRelayRouter(r)

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/images/generations/async",
		strings.NewReader(`{"model":"gpt-image-2-text-to-image","prompt":"test"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	r.ServeHTTP(recorder, req)

	if recorder.Code == http.StatusNotFound {
		t.Fatalf("expected async image generation route to be registered, got 404: %s", recorder.Body.String())
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected request to reach auth middleware and return 401 without a token, got %d: %s", recorder.Code, recorder.Body.String())
	}
}
