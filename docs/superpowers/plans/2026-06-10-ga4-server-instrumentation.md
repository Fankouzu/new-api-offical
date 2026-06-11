# GA4 Server Instrumentation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Emit server-confirmed GA4 Measurement Protocol events for voucher redemption, API key creation, and first successful billable API use.

**Architecture:** Add a small `service/analytics` package that is safe-by-default, disabled when config is missing, hashes sensitive values with HMAC-SHA256, resolves GA client IDs from `_ga` cookies or pseudonymous fallbacks, and sends asynchronously. Add a durable `model.AnalyticsEventMark` table with a unique event marker to make `first_api_call` idempotent per token.

**Tech Stack:** Go, Gin, GORM, GA4 Measurement Protocol, existing `common` JSON/env/logging helpers, existing `gopool` async pattern.

---

### Task 1: Analytics Service

**Files:**
- Create: `service/analytics/ga4.go`
- Create: `service/analytics/ga4_test.go`

- [ ] Add GA4 config loading from `GA4_EVENT_ENABLED`, `GA4_MEASUREMENT_ID`, `GA4_API_SECRET`, `GA4_EVENT_HASH_SALT`, `GA4_EVENT_DEBUG`, and `GA4_EVENT_TIMEOUT_MS`.
- [ ] Implement `HashIdentifier`, `_ga` cookie parsing, `ResolveGAClientID`, payload construction, and async HTTP sending.
- [ ] Add public helpers `TrackVoucherRedeemSuccess`, `TrackAPIKeyCreated`, and `TrackFirstAPICall`.
- [ ] Add tests for disabled config, hashing, `_ga` parsing, payload fields, and mocked HTTP sending.

### Task 2: Durable Event Mark

**Files:**
- Create: `model/analytics_event_mark.go`
- Modify: `model/main.go`
- Test: `model/analytics_event_mark_test.go`

- [ ] Add `AnalyticsEventMark` with unique composite index on `subject_type`, `subject_id`, and `event_name`.
- [ ] Add `TryMarkAnalyticsEvent(subjectType string, subjectID int, eventName string) bool`.
- [ ] Include the model in both normal and fast AutoMigrate paths.
- [ ] Test duplicate marks return `false`.

### Task 3: Business Event Hooks

**Files:**
- Modify: `controller/user.go`
- Modify: `controller/token.go`
- Modify: `service/quota.go`

- [ ] Call `analytics.TrackVoucherRedeemSuccess` only after `model.Redeem` succeeds.
- [ ] Call `analytics.TrackAPIKeyCreated` only after `cleanToken.Insert()` succeeds.
- [ ] Call `model.TryMarkAnalyticsEvent("token", tokenID, "first_api_call")` after successful non-zero quota settlement in the main text/audio quota paths, and call `analytics.TrackFirstAPICall` only when the marker insert wins.

### Task 4: Env Documentation

**Files:**
- Modify: `.env.example`

- [ ] Document GA4 backend env vars.
- [ ] Keep `GA4_API_SECRET` and `GA4_EVENT_HASH_SALT` empty in the example.

### Task 5: Verification

- [ ] Run `go test ./service/analytics ./model`.
- [ ] Run targeted controller/service package tests if import changes require it.
- [ ] Run broader `go test ./...` if feasible; record any unrelated failures.
- [ ] Commit with Lore trailers and push `feat/ga4-server-instrumentation`.

### Coverage Notes

This first implementation covers voucher redemption, API key creation, and main successful text/audio relay quota settlement. Midjourney/video/task-specific first-use hooks can be added later if those paths need separate first-use attribution.
