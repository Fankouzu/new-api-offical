# PR #39 GA4 Conversion Reliability Design

## Goal

Resolve the second-review findings on PR #39 without changing payment settlement behavior or introducing a general analytics platform. The completed change must preserve the browser attribution that existed when a payment order was created, prevent sensitive URL data from reaching GA4, keep valid query strings intact, retry transient Measurement Protocol failures, and allow the default frontend in published Docker images to use a runtime GA measurement ID.

## Scope

This design covers five changes:

1. Persist browser attribution on balance top-up and subscription orders.
2. Sanitize page referrers and initial page-view context.
3. Replace broad query repair with exact duplicate-query collapse.
4. Add bounded asynchronous retries for transient GA4 delivery failures.
5. Restore runtime GA configuration for the default Docker frontend.

It does not add a durable analytics outbox, a new dependency, client-side purchase completion, historical event backfill, or unrelated payment and analytics refactors.

## Order Attribution

### Browser snapshot

The existing first-touch attribution object will gain an optional GA4 `session_id`. When a payment API call is made, the wallet and subscription API modules will add the current first-touch attribution under an `attribution` request field. Attribution enrichment is limited to payment-order creation calls; it will not be added to the global API client.

The browser snapshot may contain:

- GA `client_id` and `session_id`;
- sanitized `page_location` and `page_referrer`;
- UTM source, medium, campaign, term, and content;
- supported click identifiers such as `gclid`;
- the first-visit timestamp.

The frontend will refresh missing GA client and session identifiers when the payment request is made, because the GA cookies may not exist during the first React effect.

### Request and storage

Each balance and subscription payment request DTO will accept the same optional nested attribution object. A shared controller helper will validate and normalize it, marshal it with `common.Marshal`, and return a compact JSON string.

`TopUp` and `SubscriptionOrder` will each receive an internal `analytics_attribution` field stored as `TEXT` and omitted from JSON responses. GORM `AutoMigrate` will add the nullable-compatible column on SQLite, MySQL, and PostgreSQL. Existing rows remain valid with an empty value.

Every order-creation path will store the normalized snapshot. This includes Epay, Stripe, Creem, Waffo, Waffo Pancake, Binance Pay, and the supported subscription payment providers.

### Conversion delivery

When a top-up or subscription completes, the GA4 payment helper will decode the order snapshot with `common.Unmarshal`. The decoded browser `client_id` will take precedence over any cookie on the webhook or administrator request. A valid numeric `session_id` will be emitted as an event parameter, along with the stored campaign and click fields.

If an old order has no snapshot, existing server-generated client-ID and configured-site URL fallbacks remain in effect. Invalid or oversized attribution values are dropped rather than blocking order creation or settlement.

## URL Privacy And Normalization

The first-touch URL sanitizer will become a shared exported helper and will be used for GA page referrers. It keeps the origin and path but only retains explicitly allowed attribution query keys. Password-reset `email`, `token`, and other unapproved parameters are therefore removed before any explicit page-view event is queued.

The initial GA `config` command will also receive sanitized page location and referrer values. This prevents the automatic initial page view from bypassing the explicit page-view sanitizer.

Page-path normalization will preserve valid query values, encoding, ordering, and repeated keys. It will collapse a second query fragment only when the complete fragment after the second `?` exactly duplicates the first fragment. For example:

- `/usage-logs/common?page=2?page=2` becomes `/usage-logs/common?page=2`;
- `/callback?next=https://example.com/a?x=1&tag=a&tag=b` is preserved;
- repeated keys such as `tag=a&tag=b` are preserved.

## GA4 Retry Behavior

GA4 delivery remains asynchronous so user and webhook responses are not delayed. A delivery will make at most three attempts with short bounded backoff.

Retries apply only to transport errors, timeouts, HTTP 408, HTTP 429, and HTTP 5xx responses. Other HTTP 4xx responses fail immediately because they indicate a payload or configuration problem. The analytics event mark remains `sending` during attempts, becomes `sent` after the first success, and becomes `failed` only after the final failure.

The existing unique analytics mark continues to prevent duplicate conversions. A later duplicate business trigger may reclaim a failed mark. Process-crash recovery without another trigger is intentionally outside this change; that requires a durable outbox and worker.

## Runtime Docker Configuration

The default frontend HTML will gain the existing Google Analytics placeholder. Runtime injection will set a small escaped global measurement-ID value for the default theme instead of independently initializing gtag. The default SPA will read the runtime value first and the build-time `VITE_GOOGLE_ANALYTICS_ID` second, then perform its normal single initialization.

The classic theme will retain its existing direct gtag injection. If both runtime and build-time IDs are present for the default theme, the runtime ID wins and only one GA script/config sequence is created. Published Docker images can therefore be configured with `GOOGLE_ANALYTICS_ID` without image-specific frontend rebuild arguments.

## Error Handling

- Missing attribution is valid and never blocks payment creation.
- Malformed attribution JSON is ignored during conversion delivery and logged without sensitive values.
- Attribution URL and identifier validation is deterministic and bounded.
- GA delivery errors are sanitized before logging so the API secret is never printed.
- Payment provider acknowledgements and order settlement remain independent from GA delivery success.

## Test Strategy

Implementation will follow red-green-refactor for each behavior:

1. Frontend tests prove payment APIs attach attribution and late GA cookie identifiers are refreshed.
2. Controller/model tests prove order snapshots are stored, decoded, and preferred over webhook/admin cookies.
3. Analytics tests prove purchase payloads contain the stored client/session identifiers and sanitized page context.
4. Frontend tests prove reset-link secrets are removed and valid/repeated query parameters are preserved.
5. Analytics tests prove transient failures retry, permanent 4xx failures do not retry, and callbacks run once.
6. Runtime configuration tests prove runtime precedence and single initialization.
7. Migration checks cover SQLite directly and the repository's MySQL/PostgreSQL dry-run or schema-test patterns where available.

Final verification will include targeted Go tests, `go test -race ./service/analytics`, frontend unit tests, typecheck, scoped lint and formatting checks, the production frontend build, and `git diff --check`.

## Acceptance Criteria

- Webhook and administrator-completed purchases use the originating browser attribution when it was captured.
- No explicit GA page-view payload contains reset-link email or token values.
- Valid nested URL values and repeated query keys are not rewritten or dropped.
- Retryable GA failures receive bounded retries before the event mark is failed.
- A published Docker image can enable default-theme analytics with runtime `GOOGLE_ANALYTICS_ID` alone.
- Existing payment completion, event naming, transaction IDs, and analytics idempotency keys remain unchanged.
