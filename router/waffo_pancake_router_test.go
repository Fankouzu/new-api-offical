package router

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestApiRouterRegistersWaffoPancakeRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetApiRouter(r)

	expected := map[string]string{
		"POST /api/waffo-pancake/webhook":     "",
		"POST /api/user/waffo-pancake/amount": "",
		"POST /api/user/waffo-pancake/pay":    "",
		"POST /api/binance-pay/webhook":       "",
		"POST /api/user/binance-pay/amount":   "",
		"POST /api/user/binance-pay/pay":      "",
	}
	for _, route := range r.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := expected[key]; ok {
			expected[key] = route.Handler
		}
	}

	for key, handler := range expected {
		if handler == "" {
			t.Fatalf("expected route %s to be registered", key)
		}
	}
}

func TestApiRouterBinancePayAmountDoesNotReturnNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetApiRouter(r)

	req, err := http.NewRequest(http.MethodPost, "/api/user/binance-pay/amount", nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, route := range r.Routes() {
		if route.Method == req.Method && route.Path == req.URL.Path {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected %s %s to be registered", req.Method, req.URL.Path)
	}
}

func TestApiRouterWaffoPancakeAmountDoesNotReturnNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetApiRouter(r)

	req, err := http.NewRequest(http.MethodPost, "/api/user/waffo-pancake/amount", nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, route := range r.Routes() {
		if route.Method == req.Method && route.Path == req.URL.Path {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected %s %s to be registered", req.Method, req.URL.Path)
	}
}
