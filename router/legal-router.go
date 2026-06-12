package router

import (
	"html"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/service/webseo"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

func SetLegalDocumentRouter(router *gin.Engine, assets ThemeAssets) {
	router.GET("/user-agreement", func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		renderLegalDocumentPage(c, assets, "User Agreement", system_setting.GetLegalSettings().UserAgreement)
	})
	router.GET("/privacy-policy", func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		renderLegalDocumentPage(c, assets, "Privacy Policy", system_setting.GetLegalSettings().PrivacyPolicy)
	})
}

func renderLegalDocumentPage(c *gin.Context, assets ThemeAssets, title string, content string) {
	theme := common.GetTheme()
	meta := webseo.ResolveMetaForTheme(c.Request.RequestURI, system_setting.ServerAddress, nil, theme)
	indexHTML := assets.DefaultIndexPage
	if theme == "classic" {
		indexHTML = assets.ClassicIndexPage
	}
	rendered := webseo.RenderIndexHTML(indexHTML, meta)
	rendered = injectLegalDocumentRoot(rendered, title, content, theme)

	c.Header("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "text/html; charset=utf-8", rendered)
}

func injectLegalDocumentRoot(indexHTML []byte, title string, content string, theme string) []byte {
	rootMarkup := buildLegalDocumentRoot(title, content, theme)
	output := string(indexHTML)
	if strings.Contains(output, `<div id="root"></div>`) {
		return []byte(strings.Replace(output, `<div id="root"></div>`, `<div id="root">`+rootMarkup+`</div>`, 1))
	}
	if strings.Contains(output, `<div id="root" ></div>`) {
		return []byte(strings.Replace(output, `<div id="root" ></div>`, `<div id="root">`+rootMarkup+`</div>`, 1))
	}
	return indexHTML
}

func buildLegalDocumentRoot(title string, content string, theme string) string {
	bodyContent := legalDocumentBodyHTML(content)
	if theme == "classic" {
		return `<main style="max-width: 960px; margin: 0 auto; padding: 48px 24px; line-height: 1.75;">
  <h1 style="font-size: 32px; line-height: 1.2; margin: 0 0 24px;">` + html.EscapeString(title) + `</h1>
  <article>` + bodyContent + `</article>
</main>`
	}

	return `<div class="bg-background text-foreground relative min-h-svh overflow-x-clip">
  <main class="container px-4 py-6 pt-20 md:px-4">
    <div class="mx-auto max-w-4xl space-y-6 py-12">
      <div class="space-y-2">
        <h1 class="text-3xl font-semibold tracking-tight">` + html.EscapeString(title) + `</h1>
      </div>
      <div class="prose prose-neutral dark:prose-invert max-w-none">` + bodyContent + `</div>
    </div>
  </main>
</div>`
}

func legalDocumentBodyHTML(content string) string {
	trimmedContent := strings.TrimSpace(content)
	if trimmedContent == "" {
		return `<p>This document has not been configured yet.</p>`
	}
	if isLegalDocumentExternalURL(trimmedContent) {
		escapedURL := html.EscapeString(trimmedContent)
		return `<p>This document is available at <a href="` + escapedURL + `" rel="noopener noreferrer">` + escapedURL + `</a>.</p>`
	}
	if isLegalDocumentHTML(trimmedContent) {
		return trimmedContent
	}
	return `<pre style="white-space: pre-wrap; overflow-wrap: anywhere; font: inherit; line-height: inherit;">` + html.EscapeString(trimmedContent) + `</pre>`
}

func isLegalDocumentExternalURL(value string) bool {
	return strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://")
}

func isLegalDocumentHTML(value string) bool {
	return strings.Contains(value, "<") && strings.Contains(value, ">")
}
