package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

	r := gin.New()
	SetLegalDocumentRouter(r)

	tests := []struct {
		path string
		want string
	}{
		{path: "/user-agreement", want: "Configured user agreement body."},
		{path: "/privacy-policy", want: "Configured privacy policy body."},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
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
		})
	}
}
