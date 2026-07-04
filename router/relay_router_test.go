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

func TestRelayRouterRegistersOpenRouterImageRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetRelayRouter(r)

	cases := []struct {
		method string
		path   string
		body   string
	}{
		{
			method: http.MethodPost,
			path:   "/api/v1/images",
			body:   `{"model":"z-image-turbo","prompt":"test"}`,
		},
		{
			method: http.MethodGet,
			path:   "/api/v1/images/models",
		},
		{
			method: http.MethodGet,
			path:   "/api/v1/images/models/z-image-turbo/endpoints",
		},
		{
			method: http.MethodGet,
			path:   "/api/v1/images/models/provider/model-name/endpoints",
		},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		r.ServeHTTP(recorder, req)

		if recorder.Code == http.StatusNotFound {
			t.Fatalf("expected %s %s to be registered, got 404: %s", tc.method, tc.path, recorder.Body.String())
		}
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected %s %s to reach auth middleware and return 401 without token, got %d: %s", tc.method, tc.path, recorder.Code, recorder.Body.String())
		}
	}
}
