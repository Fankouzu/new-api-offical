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

### Task 7: Close Frontend Privacy And Navigation Gaps

**Files:**
- Modify: `web/default/src/lib/analytics.ts`
- Modify: `web/default/src/lib/first-touch-attribution.ts`
- Modify: `web/default/tests/analytics.test.ts`
- Modify: `web/default/tests/first-touch-attribution.test.ts`

- [x] **Step 1: Add failing regression tests**

Add cases for encoded OAuth callbacks in `redirect`/`next`, usage-log token filters, sequential SPA referrers, malformed GA cookies, array-shaped localStorage, and the legitimate self-similar nested URL:

```ts
trackPageView(
  '/sign-in?redirect=%2Foauth%2Fgithub%3Fcode%3Dsecret%26state%3Dsecret'
)
expect(lastPageView.page_location).toBe('http://localhost/sign-in')

trackPageView('/usage-logs/common?token=production-key&page=2')
expect(lastPageView.page_path).toBe('/usage-logs/common?page=2')

trackPageView('/pricing')
trackPageView('/wallet')
expect(lastPageView.page_referrer).toBe('http://localhost/pricing')

document.cookie = '_ga_TEST=%E0%A4%A'
expect(() => getFirstTouchAttribution()).not.toThrow()

localStorage.setItem('lizh_first_touch_attribution', '[]')
expect(withFirstTouchAttribution({ amount: 10 })).toEqual({ amount: 10 })

expect(
  normalizeAnalyticsPagePath(
    '/callback?next=https://example.com/path?next=https://example.com/path'
  )
).toBe('/callback?next=https://example.com/path?next=https://example.com/path')
```

- [x] **Step 2: Run the focused tests and confirm RED**

Run:

```bash
cd web/default
bun test tests/analytics.test.ts tests/first-touch-attribution.test.ts
```

Expected: the new cases fail because nested navigation values and usage-log tokens are retained, page referrer is static, cookie decoding throws, arrays pass the storage check, and exact duplicate repair truncates a nested URL.

- [x] **Step 3: Implement the narrow frontend fixes**

Treat `redirect`, `next`, `return_url`, and `return_to` as sensitive page-view keys. Remove the usage-log token exception. Track the sanitized automatic initial page location and each later successfully queued page view, then reset that state in `resetAnalyticsForTests`. Wrap cookie decoding in a helper that returns an empty string on malformed encoding. Accept stored attribution only when it is a non-array object whose present known fields are strings, and project only the known fields into outgoing requests. Preserve all query text exactly; rely on the existing `urlToString` source fix to prevent app-owned duplicate query concatenation.

- [x] **Step 4: Run focused tests and confirm GREEN**

```bash
cd web/default
bun test tests/analytics.test.ts tests/first-touch-attribution.test.ts
```

Expected: PASS with the direct reproductions sanitized or preserved as specified.

### Task 8: Preserve API-Key Attribution Through The First Billable Call

**Files:**
- Modify: `web/default/src/features/keys/api.ts`
- Create: `web/default/tests/keys-api.test.ts`
- Modify: `controller/token.go`
- Modify: `controller/token_test.go`
- Modify: `model/token.go`
- Modify: `service/analytics/ga4.go`
- Modify: `service/analytics/ga4_test.go`
- Modify: `service/quota.go`
- Modify: `service/quota_analytics_test.go`

- [x] **Step 1: Add failing frontend and backend attribution tests**

Require `createApiKey` to enrich only the create body, require token creation to store normalized attribution without exposing it in API JSON, and require `first_api_call` to reuse the stored client/session/campaign values:

```go
if token.AnalyticsAttribution == "" {
    t.Fatal("token did not persist browser attribution")
}
if decoded.ClientID != "123.456" || decoded.SessionID != "789" {
    t.Fatalf("stored attribution = %#v", decoded)
}

if payload.ClientID != "123.456" || params["session_id"] != float64(789) {
    t.Fatalf("first_api_call lost browser attribution: %#v", payload)
}
```

- [x] **Step 2: Run the focused tests and confirm RED**

```bash
cd web/default && bun test tests/keys-api.test.ts
go test ./controller -run 'TestAddToken.*Attribution' -count=1
go test ./service -run 'TestTrackFirstAPICall.*Attribution' -count=1
go test ./service/analytics -run 'TestTrack(APIKeyCreated|FirstAPICall).*Session' -count=1
```

Expected: FAIL because token creation neither receives nor stores the snapshot and first-call tracking still builds a synthetic client ID.

- [x] **Step 3: Implement scoped token attribution**

Wrap only `createApiKey` with `withFirstTouchAttribution`. Add `AnalyticsAttribution string \`json:"-" gorm:"type:text"\`` to `model.Token`, and clear it in `Token.Clean()` before Redis caching. Bind token creation through a request DTO containing `analytics.SignUpAttribution`, normalize it through the existing controller encoding helper, and store it on the new token. Add a model selector that reads only `analytics_attribution` by token ID after the first-call delivery mark is acquired. Extend the API-key and first-call analytics attributes to use the stored browser client/session and campaign fields while keeping existing cookie/server fallbacks.

- [x] **Step 4: Run focused tests and confirm GREEN**

```bash
cd web/default && bun test tests/keys-api.test.ts tests/first-touch-attribution.test.ts
go test ./controller -run 'TestAddToken' -count=1
go test ./service -run 'TestTrackFirstAPICall' -count=1
go test ./service/analytics -run 'TestTrack(APIKeyCreated|FirstAPICall)' -count=1
```

Expected: PASS; raw token keys and stored attribution remain absent from API responses and GA payloads.

### Task 9: Complete Browser Session And Registration Events

**Files:**
- Modify: `service/analytics/ga4.go`
- Modify: `service/analytics/ga4_test.go`
- Modify: `controller/wechat.go`
- Create: `controller/wechat_test.go`

- [x] **Step 1: Add failing event tests**

Require `voucher_redeem_success` and `api_key_created` to contain numeric `session_id` plus `engagement_time_msec`, and require only a newly inserted WeChat user to emit `sign_up` with method `wechat`.

- [x] **Step 2: Run tests and confirm RED**

```bash
go test ./service/analytics -run 'TestTrack(VoucherRedeemSuccess|APIKeyCreated).*Session' -count=1
go test ./controller -run 'TestWeChatAuth.*SignUp' -count=1
```

Expected: FAIL because the key events omit session context and WeChat registration does not call `TrackSignUp`.

- [x] **Step 3: Add session context and the new-user hook**

Resolve the GA session cookie for voucher events. For API-key events, prefer the stored attribution session and fall back to the request cookie. In `WeChatAuth`, retain an `isNewUser` flag and call:

```go
analytics.TrackSignUp(c, user.Id, analytics.SignUpAttribution{Method: "wechat"})
```

only after a successful new-user insert. Use `common.DecodeJson` for the WeChat service response while touching this file.

- [x] **Step 4: Run tests and confirm GREEN**

```bash
go test ./service/analytics -run 'TestTrack(VoucherRedeemSuccess|APIKeyCreated)' -count=1
go test ./controller -run 'TestWeChatAuth' -count=1
```

Expected: PASS without emitting `sign_up` for existing WeChat users or failed registration.

### Task 10: Emit Idempotent Stripe Renewal Purchases

**Files:**
- Modify: `model/subscription.go`
- Modify: `controller/ga4_payment_events.go`
- Modify: `controller/ga4_payment_events_test.go`
- Modify: `controller/topup_stripe.go`
- Modify: `controller/stripe_webhook_subscription_test.go`

- [x] **Step 1: Add failing renewal conversion tests**

Configure a capture sender and require a processed `subscription_cycle` invoice to emit one event with:

```go
"name":"purchase"
"transaction_id":"in_webhook_renewal"
"value":10
"currency":"USD"
"payment_method":"stripe"
```

Replay the same webhook input and assert no second request. Add initial-invoice coverage asserting zero renewal purchase requests.

- [x] **Step 2: Run the focused tests and confirm RED**

```bash
go test ./controller -run 'TestStripeInvoicePaidWebhook.*Purchase' -count=1
```

Expected: FAIL because the processed invoice is logged but never handed to the analytics sender.

- [x] **Step 3: Implement invoice-row idempotency**

Return the persisted invoice row ID and renewal eligibility in `StripeSubscriptionInvoiceResult`. Rely on the invoice unique index; on conflict, roll back and load the existing row outside the transaction. Add a `stripe_invoice` analytics subject and begin a `purchase` delivery only for a persisted eligible `subscription_cycle` row. Send with the invoice ID as trade number, `float64(input.AmountPaid) / 100`, normalized currency, Stripe provider/method, and subscription item type. Sent marks suppress duplicate delivery, while failed marks can be reclaimed by a duplicate webhook without creating another invoice or subscription. Initial and ignored invoice rows remain ineligible.

- [x] **Step 4: Run tests and confirm GREEN**

```bash
go test ./model -run 'TestCompleteStripeSubscriptionInvoice' -count=1
go test ./controller -run 'TestStripeInvoice' -count=1
```

Expected: PASS with one successful renewal purchase and no duplicate or initial-invoice conversion.

### Task 11: Verify And Publish Third-Review Fixes

**Files:**
- Verify Tasks 7-10 and the complete PR diff

- [x] **Step 1: Run backend verification**

```bash
go test ./service/analytics ./controller ./model ./router -count=1
go test ./service -run 'TestTrackFirstAPICall' -count=1
go test -race ./service/analytics -count=1
go vet ./service/analytics ./controller ./model ./router
```

- [x] **Step 2: Run frontend verification**

```bash
cd web/default
bun test tests/analytics.test.ts tests/first-touch-attribution.test.ts tests/keys-api.test.ts src/components/layout/lib/url-utils.test.ts
bun run typecheck
bun run build
```

- [ ] **Step 3: Verify diff, commit with Lore trailers, and push**

```bash
git diff --check
git status --short
git push origin feature/lizh-ads-conversion-tracking
```

- [ ] **Step 4: Re-review the pushed PR head**

Review `origin/main...origin/feature/lizh-ads-conversion-tracking`, rerun the nine direct regression cases, and inspect PR checks. Report findings first; do not claim readiness while any confirmed issue remains.
