package webseo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

var testCatalog = []model.Pricing{
	{
		ModelName:       "deepseek-v4-flash",
		Description:     "Fast DeepSeek reasoning model.",
		ModelRatio:      0.5,
		CompletionRatio: 2,
		EnableGroup:     []string{"default"},
		SupportedEndpointTypes: []constant.EndpointType{
			constant.EndpointTypeOpenAI,
		},
	},
	{
		ModelName:       "gpt-5.4",
		Description:     "Frontier GPT model for complex coding and reasoning.",
		ModelRatio:      3,
		CompletionRatio: 4,
		EnableGroup:     []string{"default"},
		SupportedEndpointTypes: []constant.EndpointType{
			constant.EndpointTypeOpenAI,
			constant.EndpointTypeImageGeneration,
		},
	},
	{
		ModelName:       "openai/gpt-4o-mini",
		Description:     "OpenAI compact multimodal model.",
		ModelRatio:      0.2,
		CompletionRatio: 4,
		EnableGroup:     []string{"default"},
		SupportedEndpointTypes: []constant.EndpointType{
			constant.EndpointTypeOpenAI,
		},
	},
}

func readDefaultIndexHTML(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "web", "default", "index.html"))
	if err != nil {
		t.Fatalf("failed to read default index html: %v", err)
	}
	return string(data)
}

func TestResolveMetaForIndexablePublicRoutes(t *testing.T) {
	meta := ResolveMeta("/", "https://lizh.ai", testCatalog)
	if meta.Robots != "index,follow" {
		t.Fatalf("expected homepage to be indexable, got %q", meta.Robots)
	}
	if meta.Title != "Lizh AI" {
		t.Fatalf("homepage title should match OAuth brand name, got %q", meta.Title)
	}
	if !strings.Contains(meta.Description, "GPT") {
		t.Fatalf("homepage metadata is not marketplace-specific: %+v", meta)
	}
	if meta.CanonicalURL != "https://lizh.ai/" {
		t.Fatalf("unexpected canonical URL: %q", meta.CanonicalURL)
	}

	pricing := ResolveMeta("/pricing?currency=USD", "https://lizh.ai/", testCatalog)
	if pricing.CanonicalURL != "https://lizh.ai/pricing" {
		t.Fatalf("pricing canonical should drop query strings, got %q", pricing.CanonicalURL)
	}
	if !strings.Contains(pricing.Title, "AI Model API Pricing Marketplace") {
		t.Fatalf("unexpected pricing title: %q", pricing.Title)
	}
	if len(pricing.JSONLD) == 0 {
		t.Fatalf("pricing page should include JSON-LD")
	}
}

func TestResolveMetaForModelDetail(t *testing.T) {
	meta := ResolveMeta("/pricing/deepseek-v4-flash", "https://lizh.ai", testCatalog)
	if meta.Robots != "index,follow" {
		t.Fatalf("expected known model page to be indexable, got %q", meta.Robots)
	}
	if !strings.Contains(meta.Title, "DeepSeek V4 Flash API pricing") {
		t.Fatalf("unexpected model title: %q", meta.Title)
	}
	if !strings.Contains(meta.Description, "$1.0000 / 1M tokens") {
		t.Fatalf("model description should include input price, got %q", meta.Description)
	}
	if !strings.Contains(meta.Description, "$2.0000 / 1M tokens") {
		t.Fatalf("model description should include output price, got %q", meta.Description)
	}
	if meta.CanonicalURL != "https://lizh.ai/pricing/deepseek-v4-flash" {
		t.Fatalf("unexpected canonical URL: %q", meta.CanonicalURL)
	}
}

func TestResolveMetaForEscapedModelDetail(t *testing.T) {
	meta := ResolveMeta("/pricing/openai%2Fgpt-4o-mini", "https://lizh.ai", testCatalog)
	if meta.Robots != "index,follow" {
		t.Fatalf("expected escaped model page to be indexable, got %q", meta.Robots)
	}
	if !strings.Contains(meta.Title, "Openai GPT 4o Mini API pricing") {
		t.Fatalf("unexpected model title: %q", meta.Title)
	}
	if meta.CanonicalURL != "https://lizh.ai/pricing/openai%2Fgpt-4o-mini" {
		t.Fatalf("unexpected canonical URL: %q", meta.CanonicalURL)
	}
}

func TestResolveMetaNoindexesUtilityAndUnknownRoutes(t *testing.T) {
	paths := []string{"/login", "/sign-in", "/setup", "/console/topup", "/oauth/github", "/unknown-path"}
	for _, path := range paths {
		meta := ResolveMeta(path, "https://lizh.ai", testCatalog)
		if meta.Robots != "noindex,nofollow" {
			t.Fatalf("%s should be noindex,nofollow, got %q", path, meta.Robots)
		}
	}
}

func TestResolveMetaForAuthenticatedRoutesUsesConsoleTitle(t *testing.T) {
	paths := []string{
		"/usage-logs/common?startTime=1781107200000&endTime=1781127101503&page=1",
		"/usage-logs/task",
		"/wallet",
		"/tokens",
		"/playground",
		"/settings",
		"/user/edit",
	}
	for _, path := range paths {
		meta := ResolveMeta(path, "https://lizh.ai", testCatalog)
		if meta.Robots != "noindex,nofollow" {
			t.Fatalf("%s should remain noindex,nofollow, got %q", path, meta.Robots)
		}
		if strings.Contains(meta.Title, "Page not found") {
			t.Fatalf("%s should not use not-found title, got %q", path, meta.Title)
		}
		if meta.Title != "Console | Lizh AI" {
			t.Fatalf("%s title = %q, want console title", path, meta.Title)
		}
	}
}

func TestRenderIndexHTMLInjectsRouteSpecificTags(t *testing.T) {
	html := `<!doctype html><html><head><title>New API</title><meta name="description" content="old"></head><body><div id="root"></div></body></html>`
	meta := ResolveMeta("/pricing/deepseek-v4-flash", "https://lizh.ai", testCatalog)

	rendered := RenderIndexHTML([]byte(html), meta)
	text := string(rendered)
	required := []string{
		"<title>DeepSeek V4 Flash API pricing | Lizh AI</title>",
		`<meta name="description" content="DeepSeek V4 Flash API`,
		`<meta name="robots" content="index,follow">`,
		`<link rel="canonical" href="https://lizh.ai/pricing/deepseek-v4-flash">`,
		`<meta property="og:title" content="DeepSeek V4 Flash API pricing | Lizh AI">`,
		`<script type="application/ld+json">`,
	}
	for _, needle := range required {
		if !strings.Contains(text, needle) {
			t.Fatalf("rendered HTML missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, `content="old"`) {
		t.Fatalf("old generic description should be removed:\n%s", text)
	}
}

func TestDefaultIndexHTMLExposesHomepageBrandWithoutJavaScript(t *testing.T) {
	html := readDefaultIndexHTML(t)
	if !strings.Contains(html, "<title>Lizh AI</title>") {
		t.Fatalf("default index title should expose short OAuth brand name:\n%s", html)
	}
	if !strings.Contains(html, "<h1") || !strings.Contains(html, "Lizh AI") {
		t.Fatalf("default index body should expose homepage brand without JavaScript:\n%s", html)
	}
}

func TestBuildRobotsAndSitemap(t *testing.T) {
	robots := BuildRobotsTxt("https://lizh.ai")
	if !strings.Contains(robots, "Sitemap: https://lizh.ai/sitemap.xml") {
		t.Fatalf("robots should link sitemap, got:\n%s", robots)
	}
	if !strings.Contains(robots, "Disallow: /console/") {
		t.Fatalf("robots should disallow console paths, got:\n%s", robots)
	}

	sitemap := BuildSitemapXML("https://lizh.ai", testCatalog)
	required := []string{
		"<loc>https://lizh.ai/</loc>",
		"<loc>https://lizh.ai/pricing</loc>",
		"<loc>https://lizh.ai/pricing/deepseek-v4-flash</loc>",
		"<loc>https://lizh.ai/pricing/gpt-5.4</loc>",
		"<loc>https://lizh.ai/pricing/openai%2Fgpt-4o-mini</loc>",
	}
	for _, needle := range required {
		if !strings.Contains(sitemap, needle) {
			t.Fatalf("sitemap missing %q:\n%s", needle, sitemap)
		}
	}
	if strings.Contains(sitemap, "/login") || strings.Contains(sitemap, "/console/") {
		t.Fatalf("sitemap should not include utility/auth pages:\n%s", sitemap)
	}
}
