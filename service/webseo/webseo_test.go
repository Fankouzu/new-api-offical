package webseo

import (
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

	compare := ResolveMeta("/compare/ai-api-pricing", "https://lizh.ai", testCatalog)
	if compare.Robots != "index,follow" {
		t.Fatalf("compare pricing page should be indexable, got %q", compare.Robots)
	}
	if compare.CanonicalURL != "https://lizh.ai/compare/ai-api-pricing" {
		t.Fatalf("unexpected compare canonical URL: %q", compare.CanonicalURL)
	}
	if !strings.Contains(compare.Title, "AI API Pricing Comparison") {
		t.Fatalf("unexpected compare title: %q", compare.Title)
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
	if meta.OGType != "website" {
		t.Fatalf("model detail page should not use product Open Graph type, got %q", meta.OGType)
	}
}

func TestModelDetailJSONLDAvoidsProductAndMerchantRichResults(t *testing.T) {
	html := `<!doctype html><html><head><title>New API</title></head><body><div id="root"></div></body></html>`
	meta := ResolveMeta("/pricing/deepseek-v4-flash", "https://lizh.ai", testCatalog)
	rendered := string(RenderIndexHTML([]byte(html), meta))

	required := []string{
		`"@type":"Service"`,
		`"serviceType":"AI model API access"`,
		`"provider":{"@type":"Organization"`,
	}
	for _, needle := range required {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("model JSON-LD missing %q:\n%s", needle, rendered)
		}
	}

	forbidden := []string{
		`"@type":"Product"`,
		`"@type":"Offer"`,
		`"offers"`,
		`"priceSpecification"`,
		`"aggregateRating"`,
		`"review"`,
		`"hasMerchantReturnPolicy"`,
		`"shippingDetails"`,
	}
	for _, needle := range forbidden {
		if strings.Contains(rendered, needle) {
			t.Fatalf("model JSON-LD should not include product or merchant rich-result field %q:\n%s", needle, rendered)
		}
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
		`<link rel="alternate" type="text/plain" href="https://lizh.ai/llms.txt" title="llms.txt">`,
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

func TestRenderIndexHTMLInjectsLlmsAlternateLinkOnHomepage(t *testing.T) {
	html := `<!doctype html><html><head><title>New API</title></head><body><div id="root"></div></body></html>`
	meta := ResolveMeta("/", "https://lizh.ai", testCatalog)

	rendered := string(RenderIndexHTML([]byte(html), meta))
	needle := `<link rel="alternate" type="text/plain" href="https://lizh.ai/llms.txt" title="llms.txt">`
	if !strings.Contains(rendered, needle) {
		t.Fatalf("homepage HTML missing llms.txt alternate link %q:\n%s", needle, rendered)
	}
}

func TestBuildRobotsAndSitemap(t *testing.T) {
	robots := BuildRobotsTxt("https://lizh.ai")
	if !strings.Contains(robots, "Sitemap: https://lizh.ai/sitemap.xml") {
		t.Fatalf("robots should link sitemap, got:\n%s", robots)
	}
	if !strings.Contains(robots, "Allow: /llms.txt") {
		t.Fatalf("robots should explicitly allow llms.txt, got:\n%s", robots)
	}
	if strings.Contains(robots, "Disallow: /llms.txt") {
		t.Fatalf("robots should not disallow llms.txt, got:\n%s", robots)
	}
	if !strings.Contains(robots, "Disallow: /console/") {
		t.Fatalf("robots should disallow console paths, got:\n%s", robots)
	}

	sitemap := BuildSitemapXML("https://lizh.ai", testCatalog)
	required := []string{
		"<loc>https://lizh.ai/</loc>",
		"<loc>https://lizh.ai/pricing</loc>",
		"<loc>https://lizh.ai/compare/ai-api-pricing</loc>",
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

func TestIsKnownRouteRejectsUnknownAndMissingModelPaths(t *testing.T) {
	if !IsKnownRoute("/pricing", nil, "") {
		t.Fatal("pricing should be a known public route")
	}
	if IsKnownRoute("/pricing/missing-model", nil, "") {
		t.Fatal("missing model page should not be treated as a valid route")
	}
	if IsKnownRoute("/this-page-does-not-exist", nil, "") {
		t.Fatal("unknown paths should not fall back to the SPA shell")
	}
	if !IsKnownRoute("/console/topup", nil, "") {
		t.Fatal("authenticated SPA routes should keep their fallback")
	}
}

func TestIsKnownRouteAllowsFrontendDeepLinks(t *testing.T) {
	tests := []struct {
		path  string
		theme string
	}{
		{path: "/dashboard"},
		{path: "/dashboard/overview"},
		{path: "/profile"},
		{path: "/keys"},
		{path: "/subscriptions"},
		{path: "/redemption-codes"},
		{path: "/chat/conversation-123"},
		{path: "/oauth"},
		{path: "/sign-in"},
		{path: "/sign-up"},
		{path: "/forgot-password"},
		{path: "/otp"},
		{path: "/404"},
		{path: "/login", theme: "classic"},
		{path: "/register", theme: "classic"},
		{path: "/forbidden", theme: "classic"},
	}
	for _, tt := range tests {
		t.Run(tt.theme+tt.path, func(t *testing.T) {
			if !IsKnownRoute(tt.path, nil, tt.theme) {
				t.Fatalf("expected %s to be a known %s frontend route", tt.path, tt.theme)
			}
		})
	}
}

func TestIsKnownRouteRejectsInvalidDeepLinksAndWrongThemeRoutes(t *testing.T) {
	tests := []struct {
		path  string
		theme string
	}{
		{path: "/keys/not-real"},
		{path: "/profile/not-real"},
		{path: "/subscriptions/not-real"},
		{path: "/redemption-codes/not-real"},
		{path: "/dashboard/section/extra"},
		{path: "/chat/conversation/extra"},
		{path: "/oauth/provider/extra"},
		{path: "/oauth/provider/extra", theme: "classic"},
		{path: "/dashboard", theme: "classic"},
		{path: "/keys", theme: "classic"},
		{path: "/profile", theme: "classic"},
		{path: "/chat/conversation", theme: "classic"},
		{path: "/oauth", theme: "classic"},
		{path: "/compare/ai-api-pricing", theme: "classic"},
		{path: "/sign-in", theme: "classic"},
		{path: "/sign-up", theme: "classic"},
		{path: "/forgot-password", theme: "classic"},
		{path: "/otp", theme: "classic"},
		{path: "/401", theme: "classic"},
		{path: "/403", theme: "classic"},
		{path: "/404", theme: "classic"},
		{path: "/500", theme: "classic"},
		{path: "/503", theme: "classic"},
		{path: "/login"},
		{path: "/register"},
		{path: "/forbidden"},
		{path: "/usage-logs/activity", theme: "classic"},
		{path: "/playground", theme: "classic"},
		{path: "/wallet", theme: "classic"},
		{path: "/channels", theme: "classic"},
		{path: "/system-settings/site", theme: "classic"},
		{path: "/models/catalog", theme: "classic"},
	}
	for _, tt := range tests {
		t.Run(tt.theme+tt.path, func(t *testing.T) {
			if IsKnownRoute(tt.path, nil, tt.theme) {
				t.Fatalf("expected %s to be rejected for theme %q", tt.path, tt.theme)
			}
		})
	}
}

func TestBuildSitemapOnlyIncludesIndexableURLs(t *testing.T) {
	sitemap := BuildSitemapXML("https://lizh.ai", testCatalog)
	for _, url := range sitemapURLs(t, sitemap) {
		path := strings.TrimPrefix(url, "https://lizh.ai")
		meta := ResolveMeta(path, "https://lizh.ai", testCatalog)
		if strings.Contains(meta.Robots, "noindex") {
			t.Fatalf("sitemap includes noindex URL %s with robots %q", url, meta.Robots)
		}
	}
}

func sitemapURLs(t *testing.T, sitemap string) []string {
	t.Helper()

	var urls []string
	remaining := sitemap
	for {
		start := strings.Index(remaining, "<loc>")
		if start < 0 {
			return urls
		}
		remaining = remaining[start+len("<loc>"):]
		end := strings.Index(remaining, "</loc>")
		if end < 0 {
			t.Fatalf("malformed sitemap loc: %s", remaining)
		}
		urls = append(urls, remaining[:end])
		remaining = remaining[end+len("</loc>"):]
	}
}
