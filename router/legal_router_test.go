package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

func TestLegalDocumentRoutesRenderConfiguredHTML(t *testing.T) {
	gin.SetMode(gin.TestMode)

	legalSettings := system_setting.GetLegalSettings()
	oldUserAgreement := legalSettings.UserAgreement
	oldPrivacyPolicy := legalSettings.PrivacyPolicy
	defer func() {
		legalSettings.UserAgreement = oldUserAgreement
		legalSettings.PrivacyPolicy = oldPrivacyPolicy
	}()

	legalSettings.UserAgreement = "<p>Configured user agreement body.</p>"
	legalSettings.PrivacyPolicy = "<p>Configured privacy policy body.</p>"
	oldTheme := common.GetTheme()
	defer common.SetTheme(oldTheme)

	r := gin.New()
	SetLegalDocumentRouter(r, ThemeAssets{
		DefaultIndexPage: []byte(`<!doctype html><html><head><title>Old</title><link href="/static/css/index.css" rel="stylesheet"></head><body><div id="root"></div><script src="/static/js/index.js"></script></body></html>`),
		ClassicIndexPage: []byte(`<!doctype html><html><head><title>Old</title><link href="/assets/index.css" rel="stylesheet"></head><body><div id="root"></div><script src="/assets/index.js"></script></body></html>`),
	})

	tests := []struct {
		path string
		want string
	}{
		{path: "/user-agreement", want: "Configured user agreement body."},
		{path: "/privacy-policy", want: "Configured privacy policy body."},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			common.SetTheme("default")
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Host = "lizh.ai"

			r.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
			}
			contentType := recorder.Header().Get("Content-Type")
			if !strings.Contains(contentType, "text/html") {
				t.Fatalf("content type = %q, want text/html", contentType)
			}
			body := recorder.Body.String()
			if !strings.Contains(body, tt.want) {
				t.Fatalf("body does not contain configured content %q: %s", tt.want, body)
			}
			if !strings.Contains(body, "<!doctype html>") {
				t.Fatalf("body is not a full HTML document: %s", body)
			}
			if !strings.Contains(body, `href="/static/css/index.css"`) {
				t.Fatalf("body does not preserve frontend stylesheet: %s", body)
			}
			if !strings.Contains(body, `src="/static/js/index.js"`) {
				t.Fatalf("body does not preserve frontend script: %s", body)
			}
			if !strings.Contains(body, `class="prose prose-neutral dark:prose-invert max-w-none"`) {
				t.Fatalf("body does not use the default legal page content classes: %s", body)
			}
		})
	}
}

func TestLegalDocumentRoutesPreserveClassicAssets(t *testing.T) {
	gin.SetMode(gin.TestMode)

	legalSettings := system_setting.GetLegalSettings()
	oldUserAgreement := legalSettings.UserAgreement
	oldTheme := common.GetTheme()
	defer func() {
		legalSettings.UserAgreement = oldUserAgreement
		common.SetTheme(oldTheme)
	}()

	legalSettings.UserAgreement = "<p>Configured user agreement body.</p>"
	common.SetTheme("classic")

	r := gin.New()
	SetLegalDocumentRouter(r, ThemeAssets{
		DefaultIndexPage: []byte(`<!doctype html><html><head><title>Old</title><link href="/static/css/index.css" rel="stylesheet"></head><body><div id="root"></div><script src="/static/js/index.js"></script></body></html>`),
		ClassicIndexPage: []byte(`<!doctype html><html><head><title>Old</title><link href="/assets/index.css" rel="stylesheet"></head><body><div id="root"></div><script src="/assets/index.js"></script></body></html>`),
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/user-agreement", nil)
	req.Host = "lizh.ai"

	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "Configured user agreement body.") {
		t.Fatalf("body does not contain configured content: %s", body)
	}
	if !strings.Contains(body, `href="/assets/index.css"`) {
		t.Fatalf("body does not preserve classic stylesheet: %s", body)
	}
	if !strings.Contains(body, `src="/assets/index.js"`) {
		t.Fatalf("body does not preserve classic script: %s", body)
	}
}
