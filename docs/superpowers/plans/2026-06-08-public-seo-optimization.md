# Public SEO Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Optimize every page visible before login for search engines across both default and classic frontends, so Lizh AI is crawled as an AI model API marketplace rather than a generic admin SPA.

**Architecture:** SEO-critical output will be generated at the Go web fallback layer because both default and classic frontends are SPAs served through `router/web-router.go`. The backend will inject route-specific metadata into the embedded `index.html`, serve real `robots.txt` and `sitemap.xml`, and generate dynamic model-page SEO from the same pricing/model catalog data used by `/pricing`.

**Tech Stack:** Go 1.22+, Gin, embedded frontend assets, React 19 default frontend, React Router classic frontend, Bun/Rsbuild, GA4, Google Search Console.

---

## Current Findings

Online checks against `https://lizh.ai/` show that these URLs currently return the same generic SPA shell:

- `/`
- `/pricing`
- `/pricing/gpt-5.4`
- `/pricing/deepseek-v4-flash`
- `/robots.txt`
- `/sitemap.xml`

The returned HTML currently contains:

```html
<title>New API</title>
<meta name="description" content="Unified AI API gateway and admin dashboard.">
<div id="root"></div>
```

This means search engines see the site as a generic admin dashboard and do not receive model marketplace content, model-specific titles, structured data, or valid sitemap/robots files.

## Public Route Inventory

### Indexable Public Pages

These pages should be indexable and receive route-specific title, description, canonical URL, Open Graph, Twitter Card, and JSON-LD:

- `/`
- `/pricing`
- `/pricing/:modelId`
- `/rankings`
- `/about`

### Conditional Index Pages

These can be indexed if the operator wants legal/trust pages visible in search. Default recommendation: `index,follow` because they support trust and compliance.

- `/privacy-policy`
- `/user-agreement`

### Noindex Public Utility Pages

These are visible before login but should not appear as search results:

- `/sign-in`
- `/sign-up`
- `/forgot-password`
- `/reset`
- `/user/reset`
- `/otp`
- `/oauth`
- `/oauth/:provider`
- `/setup`
- `/console/topup`
- `/console/log`
- `/401`
- `/403`
- `/404`
- `/500`
- `/503`
- `/forbidden`
- unknown SPA fallback paths

### Authenticated Pages

Routes under default `/_authenticated/*` and classic `/console/*` admin/member surfaces must be `noindex,nofollow` if accidentally reachable by crawlers.

## File Structure

- Create `service/webseo/types.go`
  - SEO model types: `Meta`, `JSONLD`, `RouteKind`, `PublicRoute`.
  - Keeps SEO logic independent from router plumbing.

- Create `service/webseo/catalog.go`
  - Converts model pricing metadata into SEO-facing `ModelSEOItem`.
  - Computes per-1M token prices from per-token pricing.
  - Extracts model family/vendor names from model IDs.

- Create `service/webseo/routes.go`
  - Maps request paths to SEO metadata.
  - Handles `/`, `/pricing`, `/pricing/:modelId`, `/rankings`, `/about`, legal pages, auth pages, setup pages, and error pages.

- Create `service/webseo/render.go`
  - Injects route-specific title, meta tags, canonical, Open Graph/Twitter tags, robots meta, and JSON-LD into an HTML shell.
  - Escapes HTML and JSON safely.

- Create `service/webseo/sitemap.go`
  - Generates `robots.txt` and `sitemap.xml`.
  - Includes static public pages plus dynamic `/pricing/:modelId` pages for ready/indexable models.

- Create `service/webseo/webseo_test.go`
  - Tests metadata routing, HTML injection, sitemap generation, robots generation, noindex behavior, and model price formatting.

- Modify `router/web-router.go`
  - Serve `/robots.txt` and `/sitemap.xml` before static SPA fallback.
  - For fallback HTML, call `webseo.RenderIndexHTML(...)` for default and classic themes.
  - Preserve classic/default theme selection.

- Modify `web/default/index.html`
  - Add SEO marker comments only if needed by the renderer.
  - Keep default fallback metadata sane for static hosting.

- Modify `web/classic/index.html`
  - Add the same SEO marker comments/fallback metadata as default.
  - Preserve existing Umami and Google Analytics placeholder comments.

- Modify `web/default/src/routes/__root.tsx`
  - Optionally keep client-side title updates for runtime UX, but do not rely on them for SEO.
  - Ensure GA page view tracking still works with canonical route paths.

- Modify `web/classic/src/App.jsx`
  - Add classic frontend runtime title fallback for users navigating client-side.
  - Optional: add GA route tracking if classic is expected to be deployed with GA too.

- Create `docs/seo/public-seo-audit.md`
  - Documents public route policy, target keywords, JSON-LD strategy, and measurement plan.

## Route Metadata Matrix

### `/`

Title:

```text
Lizh AI | GPT、Gemini、DeepSeek、Qwen 多模型 API 聚合平台
```

Description:

```text
Lizh AI 提供 OpenAI 兼容的大模型 API 聚合服务，支持 GPT、Gemini、DeepSeek、Qwen、豆包、GLM、MiniMax、Kimi 等模型，统一计费、统一接口、快速接入。
```

JSON-LD:

- `Organization`
- `WebSite`
- `SoftwareApplication`

### `/pricing`

Title:

```text
AI 大模型 API 价格广场 | GPT、Gemini、DeepSeek、Qwen、豆包模型价格 - Lizh AI
```

Description:

```text
查看 Lizh AI 在售大模型 API 价格，覆盖 GPT、Gemini、DeepSeek、Qwen、GLM、豆包、MiniMax、Kimi 等 50+ 模型，支持文本、图像、工具调用和结构化输出。
```

JSON-LD:

- `ItemList` containing top/indexable model pages.
- `BreadcrumbList`.

### `/pricing/:modelId`

Example title:

```text
DeepSeek V4 Flash API 价格 | Lizh AI
```

Example description:

```text
DeepSeek V4 Flash API 支持工具调用、JSON 模式和结构化输出，输入价格约 $0.1556 / 1M tokens，输出价格约 $0.3111 / 1M tokens。
```

JSON-LD:

- `Product` or `SoftwareApplication`.
- `BreadcrumbList`.

### `/rankings`

Title:

```text
热门 AI 大模型排行榜 | Lizh AI
```

Description:

```text
查看 Lizh AI 大模型调用排行榜，了解 GPT、Gemini、DeepSeek、Qwen、豆包、GLM 等模型的热门程度和使用趋势。
```

### `/about`

Title:

```text
关于 Lizh AI | 多模型 API 聚合与 OpenAI 兼容网关
```

Description:

```text
了解 Lizh AI 的多模型 API 聚合服务、OpenAI 兼容接口、统一计费能力和面向开发者的模型接入体验。
```

### Legal Pages

`/privacy-policy`:

```text
隐私政策 | Lizh AI
```

`/user-agreement`:

```text
用户协议 | Lizh AI
```

### Noindex Utility Pages

All auth/setup/error pages should use:

```html
<meta name="robots" content="noindex,nofollow">
```

They should still have user-friendly titles, for example:

```text
登录 | Lizh AI
注册 | Lizh AI
找回密码 | Lizh AI
页面未找到 | Lizh AI
```

## Task 1: Service-Level SEO Metadata Routing

**Files:**
- Create: `service/webseo/types.go`
- Create: `service/webseo/routes.go`
- Create: `service/webseo/webseo_test.go`

- [ ] **Step 1: Write failing route metadata tests**

Add tests:

```go
func TestResolveMetaForPricingPage(t *testing.T) {
	meta := webseo.ResolveMeta(webseo.ResolveInput{
		BaseURL: "https://lizh.ai",
		Path: "/pricing",
	})
	require.Equal(t, "AI 大模型 API 价格广场 | GPT、Gemini、DeepSeek、Qwen、豆包模型价格 - Lizh AI", meta.Title)
	require.Contains(t, meta.Description, "50+ 模型")
	require.Equal(t, "https://lizh.ai/pricing", meta.CanonicalURL)
	require.Equal(t, "index,follow", meta.Robots)
}

func TestResolveMetaMarksAuthPagesNoindex(t *testing.T) {
	meta := webseo.ResolveMeta(webseo.ResolveInput{
		BaseURL: "https://lizh.ai",
		Path: "/sign-in",
	})
	require.Equal(t, "登录 | Lizh AI", meta.Title)
	require.Equal(t, "noindex,nofollow", meta.Robots)
}
```

- [ ] **Step 2: Run tests to verify red**

Run:

```bash
go test ./service/webseo -run TestResolveMeta -count=1
```

Expected: fail because package does not exist.

- [ ] **Step 3: Implement minimal metadata types and static route mapping**

Implement:

```go
type ResolveInput struct {
	BaseURL string
	Path    string
}

type Meta struct {
	Title        string
	Description  string
	CanonicalURL string
	Robots       string
	OGType       string
}
```

Implement `ResolveMeta(input ResolveInput) Meta` with static routes for `/`, `/pricing`, `/rankings`, `/about`, legal pages, auth pages, setup pages, console public redirects, and errors.

- [ ] **Step 4: Run route metadata tests**

Run:

```bash
go test ./service/webseo -run TestResolveMeta -count=1
```

Expected: pass.

## Task 2: Model-Aware SEO Catalog

**Files:**
- Create: `service/webseo/catalog.go`
- Modify: `service/webseo/routes.go`
- Modify: `service/webseo/webseo_test.go`

- [ ] **Step 1: Write failing model SEO tests**

Add tests:

```go
func TestResolveModelMetaUsesPricingAndDescription(t *testing.T) {
	catalog := []webseo.ModelSEOItem{{
		ID: "deepseek-v4-flash",
		Name: "DeepSeek V4 Flash",
		Description: "Fast DeepSeek model for coding and agents.",
		PromptPerMillionUSD: 0.155556,
		CompletionPerMillionUSD: 0.311111,
		Features: []string{"tools", "json_mode", "structured_outputs"},
	}}
	meta := webseo.ResolveMeta(webseo.ResolveInput{
		BaseURL: "https://lizh.ai",
		Path: "/pricing/deepseek-v4-flash",
		Models: catalog,
	})
	require.Equal(t, "DeepSeek V4 Flash API 价格 | Lizh AI", meta.Title)
	require.Contains(t, meta.Description, "$0.1556 / 1M tokens")
	require.Contains(t, meta.Description, "$0.3111 / 1M tokens")
	require.Equal(t, "https://lizh.ai/pricing/deepseek-v4-flash", meta.CanonicalURL)
}
```

- [ ] **Step 2: Run model tests to verify red**

Run:

```bash
go test ./service/webseo -run TestResolveModelMeta -count=1
```

Expected: fail because model-aware routing does not exist.

- [ ] **Step 3: Implement model catalog conversion**

Implement conversion from existing `model.Pricing` data:

```go
func BuildModelSEOItems(pricings []model.Pricing) []ModelSEOItem
```

Rules:

- model ID becomes slug.
- display name defaults to model ID with separators normalized.
- `ModelRatio / (1000 * ratio_setting.USD) * 1_000_000` becomes prompt price per 1M tokens.
- completion price per 1M = prompt per 1M * completion ratio.
- feature words come from supported endpoint/features where available.

- [ ] **Step 4: Run model SEO tests**

Run:

```bash
go test ./service/webseo -run TestResolveModelMeta -count=1
```

Expected: pass.

## Task 3: HTML Metadata Injection

**Files:**
- Create: `service/webseo/render.go`
- Modify: `service/webseo/webseo_test.go`

- [ ] **Step 1: Write failing renderer tests**

Add tests:

```go
func TestRenderIndexHTMLInjectsMetaAndJSONLD(t *testing.T) {
	html := []byte(`<html><head><title>New API</title><meta name="description" content="Unified AI API gateway and admin dashboard."></head><body><div id="root"></div></body></html>`)
	meta := webseo.Meta{
		Title: "AI 大模型 API 价格广场 | Lizh AI",
		Description: "查看 Lizh AI 在售大模型 API 价格。",
		CanonicalURL: "https://lizh.ai/pricing",
		Robots: "index,follow",
		OGType: "website",
		JSONLD: []map[string]any{{"@context": "https://schema.org", "@type": "WebPage", "name": "AI 大模型 API 价格广场 | Lizh AI"}},
	}
	out := string(webseo.RenderIndexHTML(html, meta))
	require.Contains(t, out, `<title>AI 大模型 API 价格广场 | Lizh AI</title>`)
	require.Contains(t, out, `<meta name="description" content="查看 Lizh AI 在售大模型 API 价格。">`)
	require.Contains(t, out, `<link rel="canonical" href="https://lizh.ai/pricing">`)
	require.Contains(t, out, `<meta property="og:title" content="AI 大模型 API 价格广场 | Lizh AI">`)
	require.Contains(t, out, `application/ld+json`)
}
```

- [ ] **Step 2: Run renderer tests to verify red**

Run:

```bash
go test ./service/webseo -run TestRenderIndexHTML -count=1
```

Expected: fail because renderer does not exist.

- [ ] **Step 3: Implement safe metadata injection**

Renderer requirements:

- Replace existing `<title>...</title>`.
- Replace existing `<meta name="description"...>`.
- Insert robots/canonical/Open Graph/Twitter/JSON-LD before `</head>`.
- Use `html/template.HTMLEscapeString` or equivalent escaping for attribute values.
- Use project `common.Marshal` for JSON-LD.
- Do not mutate body markup.

- [ ] **Step 4: Run renderer tests**

Run:

```bash
go test ./service/webseo -run TestRenderIndexHTML -count=1
```

Expected: pass.

## Task 4: Robots and Sitemap

**Files:**
- Create: `service/webseo/sitemap.go`
- Modify: `service/webseo/webseo_test.go`
- Modify: `router/web-router.go`

- [ ] **Step 1: Write failing robots/sitemap tests**

Add tests:

```go
func TestRobotsTextReferencesSitemap(t *testing.T) {
	body := webseo.RenderRobotsTxt("https://lizh.ai")
	require.Contains(t, string(body), "User-agent: *")
	require.Contains(t, string(body), "Allow: /")
	require.Contains(t, string(body), "Sitemap: https://lizh.ai/sitemap.xml")
}

func TestSitemapIncludesPublicAndModelPages(t *testing.T) {
	body := webseo.RenderSitemapXML("https://lizh.ai", []webseo.ModelSEOItem{{ID: "gpt-5.4"}, {ID: "deepseek-v4-flash"}})
	xml := string(body)
	require.Contains(t, xml, "<loc>https://lizh.ai/</loc>")
	require.Contains(t, xml, "<loc>https://lizh.ai/pricing</loc>")
	require.Contains(t, xml, "<loc>https://lizh.ai/pricing/gpt-5.4</loc>")
	require.Contains(t, xml, "<loc>https://lizh.ai/pricing/deepseek-v4-flash</loc>")
}
```

- [ ] **Step 2: Run tests to verify red**

Run:

```bash
go test ./service/webseo -run 'TestRobotsText|TestSitemap' -count=1
```

Expected: fail because functions do not exist.

- [ ] **Step 3: Implement robots/sitemap generation**

Sitemap should include:

- `/`
- `/pricing`
- `/rankings`
- `/about`
- `/privacy-policy`
- `/user-agreement`
- all indexable `/pricing/:modelId` pages.

Do not include:

- auth pages
- setup pages
- console pages
- error pages.

- [ ] **Step 4: Wire router routes**

In `router/web-router.go`, before `static.Serve("/", themeFS)`:

```go
router.GET("/robots.txt", func(c *gin.Context) {
	c.Data(http.StatusOK, "text/plain; charset=utf-8", webseo.RenderRobotsTxt(webseo.BaseURLFromRequest(c.Request)))
})
router.GET("/sitemap.xml", func(c *gin.Context) {
	models := webseo.BuildModelSEOItems(model.GetPricing())
	c.Data(http.StatusOK, "application/xml; charset=utf-8", webseo.RenderSitemapXML(webseo.BaseURLFromRequest(c.Request), models))
})
```

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./service/webseo ./router -run 'TestRobotsText|TestSitemap|TestWebRouter' -count=1
```

Expected: pass.

## Task 5: Default and Classic HTML Fallback SEO

**Files:**
- Modify: `router/web-router.go`
- Modify: `web/default/index.html`
- Modify: `web/classic/index.html`
- Create or modify: `router/web_seo_test.go`

- [ ] **Step 1: Write failing web router tests**

Add tests that build a Gin router with default/classic embedded test HTML and assert:

```go
func TestWebRouterInjectsPricingSEOForDefaultTheme(t *testing.T) {
	// request GET /pricing
	// expect HTML title contains "AI 大模型 API 价格广场"
	// expect canonical href "https://example.com/pricing"
}

func TestWebRouterInjectsPricingSEOForClassicTheme(t *testing.T) {
	// set common theme classic
	// request GET /pricing
	// expect same SEO metadata injected into classic HTML shell
}

func TestWebRouterMarksSignInNoindex(t *testing.T) {
	// request GET /sign-in
	// expect <meta name="robots" content="noindex,nofollow">
}
```

- [ ] **Step 2: Run router tests to verify red**

Run:

```bash
go test ./router -run TestWebRouter.*SEO -count=1
```

Expected: fail because fallback still serves raw index.

- [ ] **Step 3: Inject SEO in fallback**

Update `NoRoute` fallback:

```go
baseURL := webseo.BaseURLFromRequest(c.Request)
models := webseo.BuildModelSEOItems(model.GetPricing())
meta := webseo.ResolveMeta(webseo.ResolveInput{BaseURL: baseURL, Path: c.Request.URL.Path, Models: models})
if common.GetTheme() == "classic" {
	c.Data(http.StatusOK, "text/html; charset=utf-8", webseo.RenderIndexHTML(assets.ClassicIndexPage, meta))
	return
}
c.Data(http.StatusOK, "text/html; charset=utf-8", webseo.RenderIndexHTML(assets.DefaultIndexPage, meta))
```

- [ ] **Step 4: Update frontend HTML fallbacks**

Set default/classic static fallback metadata to Lizh AI, so static hosting is not completely generic:

Default and classic title:

```html
<title>Lizh AI | GPT、Gemini、DeepSeek、Qwen 多模型 API 聚合平台</title>
```

Default and classic description:

```html
<meta name="description" content="Lizh AI 提供 OpenAI 兼容的大模型 API 聚合服务，支持 GPT、Gemini、DeepSeek、Qwen、豆包、GLM、MiniMax、Kimi 等模型。">
```

- [ ] **Step 5: Run router tests**

Run:

```bash
go test ./router -run TestWebRouter.*SEO -count=1
```

Expected: pass.

## Task 6: Frontend Runtime SEO Consistency

**Files:**
- Create: `web/default/src/lib/seo.ts`
- Modify: `web/default/src/routes/__root.tsx`
- Create: `web/classic/src/helpers/seo.js`
- Modify: `web/classic/src/App.jsx`
- Create: `web/default/tests/seo.test.ts`

- [ ] **Step 1: Write default frontend tests**

Add tests for a pure function:

```ts
test('returns pricing route metadata', () => {
  expect(getClientSEOMeta('/pricing').title).toContain('AI 大模型 API 价格广场')
})

test('marks auth pages noindex', () => {
  expect(getClientSEOMeta('/sign-in').robots).toBe('noindex,nofollow')
})
```

- [ ] **Step 2: Implement default frontend runtime metadata**

Implement `getClientSEOMeta(pathname)` and a root `useEffect` that updates:

- `document.title`
- `meta[name="description"]`
- `meta[name="robots"]`
- `link[rel="canonical"]`

This is for client navigation UX only. Server fallback remains the SEO authority.

- [ ] **Step 3: Implement classic runtime metadata**

In `web/classic/src/helpers/seo.js`, implement:

```js
export function applyClientSEO(pathname) {
  const meta = getClientSEOMeta(pathname)
  document.title = meta.title
  // update description, robots, canonical
}
```

Call it from `web/classic/src/App.jsx` when `location.pathname` changes.

- [ ] **Step 4: Run frontend tests and typecheck/build**

Run:

```bash
cd web/default
bun test tests/seo.test.ts
bun run typecheck
bun run build
```

For classic:

```bash
cd web/classic
npm run build
```

Expected: pass.

## Task 7: Structured Data

**Files:**
- Modify: `service/webseo/routes.go`
- Modify: `service/webseo/render.go`
- Modify: `service/webseo/webseo_test.go`

- [ ] **Step 1: Add JSON-LD tests**

Test that:

- `/` includes `Organization`, `WebSite`, `SoftwareApplication`.
- `/pricing` includes `ItemList`.
- `/pricing/:modelId` includes `Product` or `SoftwareApplication`.
- `/pricing/:modelId` includes `BreadcrumbList`.

- [ ] **Step 2: Implement JSON-LD builders**

Add helpers:

```go
func OrganizationJSONLD(baseURL string) map[string]any
func WebSiteJSONLD(baseURL string) map[string]any
func PricingItemListJSONLD(baseURL string, models []ModelSEOItem) map[string]any
func ModelProductJSONLD(baseURL string, model ModelSEOItem) map[string]any
func BreadcrumbJSONLD(baseURL string, items []BreadcrumbItem) map[string]any
```

- [ ] **Step 3: Validate JSON-LD output**

Run:

```bash
go test ./service/webseo -run Test.*JSONLD -count=1
```

Expected: pass.

## Task 8: Public Content Improvements

**Files:**
- Modify: `web/default/src/features/home/*`
- Modify: `web/default/src/features/pricing/*`
- Modify: `web/default/src/features/rankings/*`
- Modify: `web/classic/src/pages/Home/index.jsx`
- Modify: `web/classic/src/pages/Pricing/index.jsx`
- Modify: `web/classic/src/pages/About/index.jsx`

- [ ] **Step 1: Add crawlable headings**

Ensure public pages render clear H1/H2 text after JS loads:

- Home H1: `Lizh AI 多模型 API 聚合平台`
- Pricing H1: `AI 大模型 API 价格广场`
- Rankings H1: `热门 AI 大模型排行榜`
- About H1: `关于 Lizh AI`

- [ ] **Step 2: Add keyword-rich but natural public sections**

For `/pricing`, add sections:

- `热门模型 API`
- `低价模型 API`
- `高速模型 API`
- `图像生成 API`
- `支持工具调用的模型`
- `支持结构化输出的模型`
- `国产大模型 API`
- `OpenAI 兼容 API 调用说明`

- [ ] **Step 3: Keep auth utility pages minimal**

Do not add marketing blocks to auth/reset/setup pages. They remain `noindex`.

- [ ] **Step 4: Verify visual and build**

Run default and classic builds. Use browser screenshots for `/`, `/pricing`, `/rankings`, `/about` in both themes if local servers are available.

## Task 9: Measurement and Feedback Loop

**Files:**
- Modify: `web/default/src/lib/analytics.ts`
- Modify: `web/classic/src/helpers/analytics.js` if classic GA support is implemented.
- Create: `docs/seo/measurement-plan.md`

- [ ] **Step 1: Document GA4 and Search Console setup**

GA4 alone can report behavior after users arrive on the site: sessions, engaged sessions, conversions/key events, landing pages, page views, scrolls/events if configured.

GA4 alone does not reliably provide Google search query data. For SEO feedback, link GA4 with Google Search Console. Google documents that the integration adds:

- `Google Organic Search Queries`
- `Google Organic Search Traffic`

Search Console provides impressions, clicks, CTR, and average position. GA4 then helps evaluate landing page engagement and key events.

Sources:

- https://support.google.com/analytics/answer/10737381
- https://developers.google.com/search/docs/monitor-debug/google-analytics-search-console
- https://support.google.com/webmasters/answer/10268906
- https://support.google.com/webmasters/answer/7042828

- [ ] **Step 2: Add SEO key events**

Track non-sensitive public conversion events:

```ts
trackAnalyticsEvent('pricing_model_open', { model_id: modelId })
trackAnalyticsEvent('pricing_filter_use', { filter_type: filterType })
trackAnalyticsEvent('api_docs_copy_click', { source: 'pricing' })
trackAnalyticsEvent('sign_up_cta_click', { source: pathname })
```

Do not send API keys, user emails, prompts, or request payloads.

- [ ] **Step 3: Define dashboards**

GA4 reports:

- Organic sessions by landing page.
- Engagement rate by landing page.
- Sign-up CTA clicks by landing page.
- Model detail opens by model family.
- Pricing filter usage.

Search Console reports:

- Queries with high impressions and low CTR.
- Pages with average position 5-20 and meaningful impressions.
- `/pricing/:modelId` pages with impressions but low CTR.
- Missing indexed pages from sitemap.

- [ ] **Step 4: Establish optimization cadence**

Every 2 weeks:

1. Export GSC query/page data.
2. Identify pages with:
   - high impressions + low CTR: rewrite title/description.
   - rank 5-20 + high impressions: expand page content and internal links.
   - clicks + poor engagement: improve page content/CTA.
   - no impressions: verify sitemap/indexing/internal links.
3. Compare GA4 organic landing page engagement and key events.
4. Update copy, headings, JSON-LD, and internal links.

## Verification Commands

Backend:

```bash
go test ./service/webseo ./router -count=1
go test ./... -count=1
```

Default frontend:

```bash
cd web/default
bun test tests/seo.test.ts
bun run typecheck
bun run build
```

Classic frontend:

```bash
cd web/classic
npm run build
```

Live curl checks after deploy:

```bash
curl -L -sS https://lizh.ai/pricing | rg '<title>|description|canonical|application/ld\\+json'
curl -L -sS https://lizh.ai/pricing/deepseek-v4-flash | rg '<title>|description|canonical|application/ld\\+json'
curl -L -sS https://lizh.ai/robots.txt
curl -L -sS https://lizh.ai/sitemap.xml | head
```

Expected:

- `/robots.txt` returns `text/plain`, not SPA HTML.
- `/sitemap.xml` returns XML and includes model URLs.
- `/pricing` has marketplace title/description.
- `/pricing/:modelId` has model-specific title/description.
- auth/setup/error pages include `noindex,nofollow`.

## GA4 Feedback Answer

Yes, Google Analytics can be used as part of the optimization feedback loop, but not by itself.

What GA4 can tell you:

- Which public landing pages receive organic traffic.
- Whether organic visitors engage after landing.
- Which model pages lead to sign-up clicks or other key events.
- Which pricing interactions users use.
- Whether SEO traffic converts after landing.

What GA4 cannot fully tell you alone:

- Exact Google search queries for each converting user.
- Complete impressions, CTR, and average position for Google Search.
- Whether Google indexed every submitted model URL.

Required pairing:

- Link GA4 with Google Search Console.
- Use Search Console for query, impression, CTR, and position.
- Use GA4 for engagement, retention, and conversion/key events.
- Use both together by landing page to decide what to rewrite or expand.

## Self-Review

- Spec coverage: Covers all logged-out public routes, default frontend, classic frontend, server-rendered metadata, robots, sitemap, model pages, JSON-LD, noindex policy, and GA/GSC feedback loop.
- Placeholder scan: No task uses TBD or deferred placeholders; each task states files, tests, implementation, and expected commands.
- Type consistency: `webseo.ResolveMeta`, `webseo.RenderIndexHTML`, `webseo.RenderRobotsTxt`, `webseo.RenderSitemapXML`, and `webseo.BuildModelSEOItems` names are consistent across tasks.
