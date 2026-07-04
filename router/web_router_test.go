package router

import (
	"embed"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

func TestWebRouterServesLlmsTxtAsPlainText(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldServerAddress := system_setting.ServerAddress
	defer func() {
		system_setting.ServerAddress = oldServerAddress
	}()
	system_setting.ServerAddress = "https://lizh.ai"

	r := gin.New()
	SetWebRouter(r, ThemeAssets{
		DefaultBuildFS:   emptyWebDefaultBuildFS,
		DefaultIndexPage: []byte(`<!doctype html><html><head><title>Old</title></head><body><div id="root"></div></body></html>`),
		ClassicBuildFS:   emptyWebClassicBuildFS,
		ClassicIndexPage: []byte(`<!doctype html><html><head><title>Old</title></head><body><div id="root"></div></body></html>`),
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/llms.txt", nil)
	req.Host = "lizh.ai"

	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	contentType := recorder.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") && !strings.Contains(contentType, "text/markdown") {
		t.Fatalf("content type = %q, want text/plain or text/markdown", contentType)
	}
	body := recorder.Body.String()
	required := []string{
		"# Lizh AI",
		"Canonical site: https://lizh.ai",
		"API Base URL: https://lizh.ai/v1",
		"AI Crawler Policy",
	}
	for _, needle := range required {
		if !strings.Contains(body, needle) {
			t.Fatalf("llms.txt body missing %q:\n%s", needle, body)
		}
	}
	forbidden := []string{
		"<!doctype html>",
		"noindex",
	}
	for _, needle := range forbidden {
		if strings.Contains(strings.ToLower(body), strings.ToLower(needle)) {
			t.Fatalf("llms.txt body should not contain %q:\n%s", needle, body)
		}
	}
}

//go:embed web/default/dist
var emptyWebDefaultBuildFS embed.FS

//go:embed web/classic/dist
var emptyWebClassicBuildFS embed.FS
