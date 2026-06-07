# Public SEO Audit and Measurement Plan

## Scope

This SEO pass covers pages visible before login in both frontend themes:

- `/`
- `/pricing`
- `/pricing/:modelId`
- `/rankings` in the default theme
- `/about`
- `/privacy-policy`
- `/user-agreement`
- auth, setup, console utility, OAuth, and error pages

The Go web fallback injects route-specific metadata because both frontends are SPAs. Client-side metadata sync is only a navigation fallback; crawlers should receive useful HTML before JavaScript runs.

## Indexing Policy

Index:

- `/`
- `/pricing`
- `/pricing/:modelId`
- `/rankings` when the default theme is active
- `/about`
- `/privacy-policy`
- `/user-agreement`

Noindex:

- login, register, password reset, OTP, OAuth callback, setup
- `/console/*`
- `/api/*`, `/v1/*`, `/openrouter/*`
- error pages and unknown SPA fallback paths

## Target Search Intents

- OpenAI compatible API gateway
- AI model API pricing
- GPT API price
- Gemini API price
- DeepSeek API price
- Qwen API price
- Doubao API price
- GLM API price
- MiniMax API price
- Kimi API price
- multi-model API aggregation

## Technical SEO Output

The server now generates:

- route-specific `<title>`
- `description`
- `robots`
- canonical URL
- Open Graph and Twitter summary metadata
- JSON-LD for homepage, pricing list, model detail pages, and breadcrumbs
- `robots.txt`
- `sitemap.xml`

## Google Analytics Feedback Loop

Google Analytics 4 can help evaluate SEO outcomes, but it is not sufficient alone.

GA4 can show:

- organic landing pages
- page views and engaged sessions
- engagement rate and average engagement time
- pricing page and model detail page journeys
- sign-up, token creation, top-up, and other configured conversion events

GA4 cannot fully show:

- exact Google query terms for every visit
- impressions, CTR, and average position for each query/page
- indexing coverage and crawl errors

Use Google Search Console together with GA4:

- Search Console reports queries, impressions, clicks, CTR, average position, indexing status, and sitemap ingestion.
- Link Search Console to GA4 so landing-page engagement can be compared with query and ranking data.
- Use URL Inspection for important pages such as `/pricing`, `/pricing/gpt-5.4`, and `/pricing/deepseek-v4-flash`.

Recommended review cadence:

- Weekly for the first month after deployment.
- Monthly after indexing stabilizes.

Optimization loop:

1. In Search Console, find pages with high impressions and low CTR.
2. Improve titles/descriptions for those exact query intents.
3. In GA4, check whether organic visitors engage with pricing/model pages.
4. Add or refine visible page copy for pages with weak engagement.
5. Resubmit sitemap and inspect key URLs after large metadata or content changes.

References:

- https://support.google.com/analytics/answer/10737381
- https://developers.google.com/search/docs/monitor-debug/google-analytics-search-console
- https://support.google.com/webmasters/answer/10268906
- https://support.google.com/webmasters/answer/7042828
