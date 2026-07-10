# PR #39 GA4 Conversion Reliability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make PR #39 preserve payment-session attribution, protect sensitive URLs, retry transient GA4 failures, and support runtime GA configuration in published Docker images.

**Architecture:** Payment API modules attach the existing first-touch snapshot only when creating an order. Controllers sanitize and store that snapshot as `TEXT` on the order, and completion handlers reconstruct a purchase attribution object whose explicit browser identifiers override webhook or administrator cookies. Frontend analytics sanitizes referrers and narrowly repairs only exact duplicated query fragments; GA4 delivery retries transient errors in the existing asynchronous goroutine; runtime Docker configuration exposes a measurement ID global that the default SPA initializes once.

**Tech Stack:** Go 1.22+, Gin, GORM, SQLite/MySQL/PostgreSQL-compatible `TEXT`, React 19, TypeScript, Bun tests, Rsbuild.

---

### Task 1: Protect Page-View URLs Without Corrupting Queries

**Files:**
- Modify: `web/default/src/lib/first-touch-attribution.ts`
- Modify: `web/default/src/lib/analytics.ts`
- Modify: `web/default/tests/analytics.test.ts`
- Create: `web/default/tests/first-touch-attribution.test.ts`

- [ ] **Step 1: Write failing URL and referrer tests**

Add tests that require the analytics layer to preserve the nested URL and repeated keys, collapse only an exact duplicated query fragment, and remove reset credentials from referrers:

```ts
expect(
  normalizeAnalyticsPagePath(
    '/callback?next=https://example.com/a?x=1&tag=a&tag=b'
  )
).toBe('/callback?next=https://example.com/a?x=1&tag=a&tag=b')

document.referrer =
  'https://lizh.ai/user/reset?email=user@example.com&token=secret'
trackPageView('/wallet')
expect(lastPageView.page_referrer).toBe('https://lizh.ai/user/reset')
```

- [ ] **Step 2: Run tests and confirm RED**

Run:

```bash
cd web/default
bun test tests/analytics.test.ts tests/first-touch-attribution.test.ts
```

Expected: FAIL because the current page view emits the raw referrer and rewrites all `?` characters while dropping duplicate keys.

- [ ] **Step 3: Export the attribution URL sanitizer and add GA cookie session refresh**

Extend the stored type and refresh missing browser identifiers:

```ts
export interface FirstTouchAttribution {
  client_id?: string
  session_id?: string
  page_location?: string
  page_referrer?: string
  source?: string
  medium?: string
  campaign?: string
  term?: string
  content?: string
  gclid?: string
  fbclid?: string
  ttclid?: string
  yclid?: string
  first_visit_at?: string
}

export function sanitizeAttributionURL(raw: string): string | undefined {
  // keep origin/path and SAFE_URL_PARAMS only
}

function readGASessionID(): string {
  // accept GS1.* numeric session segment and GS2.* $s numeric segment
}
```

`initializeFirstTouchAttribution()` must update an existing snapshot when either `client_id` or `session_id` becomes available after initial page load.

- [ ] **Step 4: Replace broad query rewriting with exact duplicate collapse**

Implement normalization without reparsing the query:

```ts
function normalizeSearchParams(search: string): string {
  if (!search) return ''
  const raw = search.slice(1)
  const duplicateSeparator = raw.indexOf('?')
  if (
    duplicateSeparator > 0 &&
    raw.slice(0, duplicateSeparator) === raw.slice(duplicateSeparator + 1)
  ) {
    return `?${raw.slice(0, duplicateSeparator)}`
  }
  return search
}
```

Use `sanitizeAttributionURL(document.referrer)` for `page_referrer`. The initial `gtag('config', ...)` call must also provide sanitized page location and referrer values so the automatic first page view cannot bypass the privacy boundary.

- [ ] **Step 5: Run frontend tests and confirm GREEN**

Run:

```bash
cd web/default
bun test tests/analytics.test.ts tests/first-touch-attribution.test.ts src/components/layout/lib/url-utils.test.ts
```

Expected: all tests pass and neither reset email nor reset token appears in serialized data-layer commands.

- [ ] **Step 6: Commit the URL/privacy change**

```bash
git add web/default/src/lib/analytics.ts web/default/src/lib/first-touch-attribution.ts web/default/tests/analytics.test.ts web/default/tests/first-touch-attribution.test.ts
git commit -m "Protect analytics URLs without rewriting valid queries"
```

### Task 2: Attach Browser Attribution To Payment Requests

**Files:**
- Modify: `web/default/src/lib/first-touch-attribution.ts`
- Modify: `web/default/src/features/wallet/api.ts`
- Modify: `web/default/src/features/wallet/types.ts`
- Modify: `web/default/src/features/subscriptions/api.ts`
- Modify: `web/default/src/features/subscriptions/types.ts`
- Modify: `web/default/tests/first-touch-attribution.test.ts`

- [ ] **Step 1: Write a failing request-enrichment test**

Add a pure helper test:

```ts
expect(withFirstTouchAttribution({ amount: 10 })).toEqual({
  amount: 10,
  attribution: expect.objectContaining({
    client_id: '123.456',
    session_id: '789',
  }),
})
```

- [ ] **Step 2: Run the test and confirm RED**

Run:

```bash
cd web/default
bun test tests/first-touch-attribution.test.ts
```

Expected: FAIL because `withFirstTouchAttribution` does not exist.

- [ ] **Step 3: Implement scoped payment enrichment**

Add the helper:

```ts
export function withFirstTouchAttribution<T extends object>(
  request: T
): T & { attribution?: FirstTouchAttribution } {
  const attribution = getFirstTouchAttribution()
  return attribution ? { ...request, attribution } : request
}
```

Add `attribution?: FirstTouchAttribution` to wallet and subscription payment request types. Wrap only these POST bodies:

```ts
api.post('/api/user/pay', withFirstTouchAttribution(request))
api.post('/api/user/stripe/pay', withFirstTouchAttribution(request))
api.post('/api/user/creem/pay', withFirstTouchAttribution(request))
api.post('/api/user/waffo/pay', withFirstTouchAttribution(request))
api.post('/api/user/waffo-pancake/pay', withFirstTouchAttribution(request))
api.post('/api/user/binance-pay/pay', withFirstTouchAttribution(request))
api.post('/api/subscription/stripe/pay', withFirstTouchAttribution(data))
api.post('/api/subscription/creem/pay', withFirstTouchAttribution(data))
api.post('/api/subscription/epay/pay', withFirstTouchAttribution(data))
```

- [ ] **Step 4: Run tests and typecheck**

Run:

```bash
cd web/default
bun test tests/first-touch-attribution.test.ts
bun run typecheck
```

Expected: PASS.

- [ ] **Step 5: Commit the request enrichment**

```bash
git add web/default/src/lib/first-touch-attribution.ts web/default/src/features/wallet/api.ts web/default/src/features/wallet/types.ts web/default/src/features/subscriptions/api.ts web/default/src/features/subscriptions/types.ts web/default/tests/first-touch-attribution.test.ts
git commit -m "Carry browser attribution into payment orders"
```

### Task 3: Persist And Reuse Order Attribution

**Files:**
- Modify: `model/topup.go`
- Modify: `model/subscription.go`
- Modify: `controller/ga4_payment_events.go`
- Modify: `controller/ga4_payment_events_test.go`
- Modify: `controller/topup.go`
- Modify: `controller/topup_stripe.go`
- Modify: `controller/topup_creem.go`
- Modify: `controller/topup_waffo.go`
- Modify: `controller/topup_waffo_pancake.go`
- Modify: `controller/topup_binance_pay.go`
- Modify: `controller/subscription_payment_epay.go`
- Modify: `controller/subscription_payment_stripe.go`
- Modify: `controller/subscription_payment_creem.go`
- Modify: `service/analytics/ga4.go`
- Modify: `service/analytics/ga4_test.go`

- [ ] **Step 1: Write failing attribution serialization and purchase tests**

Add tests requiring `encodeGA4Attribution` to sanitize URLs, `decodeGA4Attribution` to round-trip browser IDs, and a purchase payload to prefer the stored client ID over a context cookie:

```go
stored := analytics.SignUpAttribution{
	ClientID: "123.456",
	SessionID: "789",
	PageReferrer: "https://lizh.ai/user/reset?email=a@example.com&token=secret",
}
encoded := encodeGA4Attribution(stored)
decoded := decodeGA4Attribution(encoded)
require.Equal(t, "123.456", decoded.ClientID)
require.Equal(t, "789", decoded.SessionID)
require.Equal(t, "https://lizh.ai/user/reset", decoded.PageReferrer)
```

- [ ] **Step 2: Run targeted tests and confirm RED**

Run:

```bash
go test ./controller -run 'TestGA4OrderAttribution' -count=1
go test ./service/analytics -run 'TestTrackPurchaseUsesStoredBrowserAttribution' -count=1
```

Expected: FAIL because order attribution fields and helpers do not exist.

- [ ] **Step 3: Add cross-database order fields**

Add the same internal field to both models:

```go
AnalyticsAttribution string `json:"-" gorm:"type:text"`
```

Rely on the existing GORM `AutoMigrate` registration for `TopUp` and `SubscriptionOrder`; do not add database-specific SQL.

- [ ] **Step 4: Add shared request, encoding, and decoding helpers**

In `controller/ga4_payment_events.go`, define:

```go
type ga4AttributionRequest struct {
	Attribution analytics.SignUpAttribution `json:"attribution"`
}

func encodeGA4Attribution(attrs analytics.SignUpAttribution) string
func decodeGA4Attribution(raw string) analytics.SignUpAttribution
```

Both helpers must call the exported analytics normalizer; encoding uses `common.Marshal`, decoding uses `common.Unmarshal`, and malformed input returns an empty attribution.

- [ ] **Step 5: Store the snapshot in every order-creation path**

Embed `ga4AttributionRequest` in `EpayRequest`, `StripePayRequest`, `CreemPayRequest`, `WaffoPayRequest`, `WaffoPancakePayRequest`, `BinancePayRequest`, `SubscriptionEpayPayRequest`, `SubscriptionStripePayRequest`, and `SubscriptionCreemPayRequest`.

Each `TopUp` or `SubscriptionOrder` literal must set:

```go
AnalyticsAttribution: encodeGA4Attribution(req.Attribution),
```

Do not add attribution to amount-preview endpoints. Remove the raw Creem request-body log because the body now contains analytics identifiers.

- [ ] **Step 6: Build purchase attribution from the stored snapshot**

Extend `PurchaseAttribution`:

```go
Attribution SignUpAttribution
```

Normalize it, add valid `session_id`, source/campaign/click fields to event params, and call `trackWithClientID` using `Attribution.ClientID`. `trackGA4TopUpSuccessWithCurrency` and `trackGA4PurchaseSuccessWithCurrency` pass the decoded order snapshot.

- [ ] **Step 7: Run targeted tests and confirm GREEN**

Run:

```bash
go test ./controller -run 'Test(GA4OrderAttribution|ResolveGA4|BeginGA4)' -count=1
go test ./service/analytics -run 'Test(TrackPurchase|NormalizeSignUpAttribution)' -count=1
go test ./model -run 'Test(PaymentMethod|AnalyticsEvent)' -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit the persisted attribution change**

```bash
git add model/topup.go model/subscription.go controller/ga4_payment_events.go controller/ga4_payment_events_test.go controller/topup.go controller/topup_stripe.go controller/topup_creem.go controller/topup_waffo.go controller/topup_waffo_pancake.go controller/topup_binance_pay.go controller/subscription_payment_epay.go controller/subscription_payment_stripe.go controller/subscription_payment_creem.go service/analytics/ga4.go service/analytics/ga4_test.go
git commit -m "Preserve browser attribution through payment completion"
```

### Task 4: Retry Transient GA4 Failures

**Files:**
- Modify: `service/analytics/ga4.go`
- Modify: `service/analytics/ga4_test.go`
- Modify: `controller/token_test.go`

- [ ] **Step 1: Write failing retry tests**

Add sender sequences for `500, 500, 204` and `400, 204`. Require three attempts for the transient sequence, one attempt for the permanent 4xx sequence, and exactly one result callback.

```go
err := sendPayloadWithRetry(context.Background(), cfg, payload, []time.Duration{0, 0})
require.NoError(t, err)
require.Equal(t, 3, sender.Attempts())
```

- [ ] **Step 2: Run retry tests and confirm RED**

Run:

```bash
go test ./service/analytics -run 'TestSendPayloadRetries' -count=1
```

Expected: FAIL because retry classification and the retry loop do not exist.

- [ ] **Step 3: Implement bounded retry classification**

Introduce a typed status error and retry only transport errors, timeouts, 408, 429, and 5xx:

```go
type ga4StatusError struct{ StatusCode int }

func isRetryableDeliveryError(err error) bool

func sendPayloadWithRetry(
	ctx context.Context,
	cfg Config,
	payload ga4Payload,
	backoffs []time.Duration,
) error
```

Production uses two short backoffs, giving three total attempts. `trackWithClientID` invokes the retry wrapper inside the existing `gopool` task and calls `onResult` once after final success or failure.

- [ ] **Step 4: Run analytics and controller tests**

Run:

```bash
go test ./service/analytics -count=1
go test -race ./service/analytics -count=1
go test ./controller -run 'TestAddTokenMarksAPIKeyCreated' -count=1
```

Expected: PASS, with failed marks written only after the final attempt.

- [ ] **Step 5: Commit the retry change**

```bash
git add service/analytics/ga4.go service/analytics/ga4_test.go controller/token_test.go
git commit -m "Retry transient GA4 conversion delivery failures"
```

### Task 5: Restore Runtime GA Configuration For Default Docker Images

**Files:**
- Modify: `main.go`
- Create: `main_analytics_test.go`
- Modify: `web/default/index.html`
- Modify: `web/default/src/lib/analytics.ts`
- Modify: `web/default/tests/analytics.test.ts`
- Modify: `web/default/src/env.d.ts`

- [ ] **Step 1: Write failing runtime-precedence tests**

Require the runtime global to override the build-time value and require the Go default-theme snippet to set only the escaped global:

```ts
window.__GOOGLE_ANALYTICS_ID__ = 'G-RUNTIME'
expect(getGoogleAnalyticsMeasurementId()).toBe('G-RUNTIME')
```

```go
defaultSnippet, classicSnippet := buildGoogleAnalyticsSnippets(`G-TEST`)
require.Contains(t, defaultSnippet, `window.__GOOGLE_ANALYTICS_ID__="G-TEST"`)
require.NotContains(t, defaultSnippet, "googletagmanager.com")
require.Contains(t, classicSnippet, "googletagmanager.com/gtag/js")
```

- [ ] **Step 2: Run tests and confirm RED**

Run:

```bash
go test . -run TestBuildGoogleAnalyticsSnippets -count=1
cd web/default && bun test tests/analytics.test.ts
```

Expected: FAIL because the runtime global and split snippets do not exist.

- [ ] **Step 3: Implement runtime configuration**

Add `<!--Google Analytics-->` to `web/default/index.html`. Add the global type and runtime-first lookup:

```ts
interface Window {
  __GOOGLE_ANALYTICS_ID__?: string
}

return (
  window.__GOOGLE_ANALYTICS_ID__ ||
  import.meta.env?.VITE_GOOGLE_ANALYTICS_ID ||
  ''
).trim()
```

Refactor `InjectGoogleAnalytics` through `buildGoogleAnalyticsSnippets`. Use `common.Marshal` to encode the ID. The default snippet sets the global only; the classic snippet retains direct gtag initialization. Replace the placeholder separately for each embedded index page.

- [ ] **Step 4: Run runtime tests, typecheck, and build**

Run:

```bash
go test . -run TestBuildGoogleAnalyticsSnippets -count=1
cd web/default
bun test tests/analytics.test.ts
bun run typecheck
VITE_GOOGLE_ANALYTICS_ID=G-BUILD bun run build
```

Expected: PASS and the production bundle contains the build-time ID while runtime lookup remains available.

- [ ] **Step 5: Commit the runtime configuration change**

```bash
git add main.go main_analytics_test.go web/default/index.html web/default/src/lib/analytics.ts web/default/tests/analytics.test.ts web/default/src/env.d.ts
git commit -m "Allow default Docker images to configure GA at runtime"
```

### Task 6: Final Verification, Push, And Review

**Files:**
- Verify all files changed by Tasks 1-5

- [ ] **Step 1: Run backend verification**

```bash
go test ./controller ./model ./router ./service/analytics -count=1
go test ./service -run 'TestTrackFirstAPICall' -count=1
go test -race ./service/analytics -count=1
```

Expected: PASS.

- [ ] **Step 2: Run frontend verification**

```bash
cd web/default
bun test tests/analytics.test.ts tests/first-touch-attribution.test.ts src/components/layout/lib/url-utils.test.ts src/features/wallet/lib/payment.test.ts
bun run typecheck
bunx eslint src/lib/analytics.ts src/lib/first-touch-attribution.ts src/features/wallet/api.ts src/features/wallet/types.ts src/features/subscriptions/api.ts src/features/subscriptions/types.ts tests/analytics.test.ts tests/first-touch-attribution.test.ts
bunx prettier --check src/lib/analytics.ts src/lib/first-touch-attribution.ts src/features/wallet/api.ts src/features/wallet/types.ts src/features/subscriptions/api.ts src/features/subscriptions/types.ts tests/analytics.test.ts tests/first-touch-attribution.test.ts index.html
bun run build
```

Expected: PASS.

- [ ] **Step 3: Check the final diff and commit any verification-only fixes**

```bash
git diff --check
git status --short
git log --oneline origin/feature/lizh-ads-conversion-tracking..HEAD
```

Expected: clean worktree and only PR #39 reliability commits ahead of origin.

- [ ] **Step 4: Push the branch**

```bash
git push origin feature/lizh-ads-conversion-tracking
```

Expected: PR #39 head advances to the final local commit.

- [ ] **Step 5: Review the pushed PR head**

Review `origin/main...origin/feature/lizh-ads-conversion-tracking`, rerun the direct URL/referrer reproductions, inspect GitHub checks, and report findings first. Approve only if no CRITICAL or HIGH issues remain.
