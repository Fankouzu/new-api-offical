package oauth

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

func TestLinuxDOExchangeTokenUsesFrontendRedirectURI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var tokenForm url.Values
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse token request form: %v", err)
		}
		tokenForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"linuxdo-access-token"}`))
	}))
	defer tokenServer.Close()

	t.Setenv("LINUX_DO_TOKEN_ENDPOINT", tokenServer.URL)
	previousServerAddress := system_setting.ServerAddress
	previousClientID := common.LinuxDOClientId
	previousClientSecret := common.LinuxDOClientSecret
	system_setting.ServerAddress = "https://lizh.ai"
	common.LinuxDOClientId = "linuxdo-client-id"
	common.LinuxDOClientSecret = "linuxdo-client-secret"
	defer func() {
		system_setting.ServerAddress = previousServerAddress
		common.LinuxDOClientId = previousClientID
		common.LinuxDOClientSecret = previousClientSecret
	}()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "https://lizh.ai/api/oauth/linuxdo?code=code", strings.NewReader(""))

	_, err := (&LinuxDOProvider{}).ExchangeToken(c.Request.Context(), "code", c)
	if err != nil {
		t.Fatalf("ExchangeToken returned error: %v", err)
	}

	if got := tokenForm.Get("redirect_uri"); got != "https://lizh.ai/oauth/linuxdo" {
		t.Fatalf("redirect_uri = %q, want frontend callback URL", got)
	}
}
