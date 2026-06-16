# Tencent VOD AIGC Channel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Add Tencent Cloud VOD AIGC as a new async image/video generation channel with deterministic resolution, duration, and count billing multipliers.

**Architecture:** Implement a new `relay/channel/task/tencentvod` task adaptor that signs Tencent Cloud VOD `POST /` requests with TC3-HMAC-SHA256, submits `CreateAigcImageTask` / `CreateAigcVideoTask`, polls `DescribeTaskDetail`, and maps task results into the existing new-api async task model. Billing uses existing task `OtherRatios`: operators configure lowest-tier model price, while the adaptor multiplies by resolution, duration, count, and feature ratios.

**Tech Stack:** Go 1.22+, Gin, existing new-api task relay architecture, React default/classic channel forms.

---

## Source Document

Read `docs/integrations/tencent-vod-aigc.md` before modifying code.

## Files

- Create: `relay/channel/task/tencentvod/config.go`
- Create: `relay/channel/task/tencentvod/models.go`
- Create: `relay/channel/task/tencentvod/billing.go` (implemented inside `adaptor.go` to keep request/billing extraction together)
- Create: `relay/channel/task/tencentvod/sign.go`
- Create: `relay/channel/task/tencentvod/adaptor.go`
- Create: `relay/channel/task/tencentvod/config_test.go` (covered in `adaptor_test.go`)
- Create: `relay/channel/task/tencentvod/billing_test.go` (covered in `adaptor_test.go`)
- Create: `relay/channel/task/tencentvod/sign_test.go` (covered in `adaptor_test.go`)
- Create: `relay/channel/task/tencentvod/adaptor_test.go`
- Modify: `constant/channel.go`
- Modify: `relay/relay_adaptor.go`
- Modify: `relay/relay_adaptor_test.go`
- Modify: `web/default/src/features/channels/constants.ts`
- Modify: `web/default/src/features/channels/lib/channel-utils.ts`
- Modify: `web/default/src/features/channels/lib/channel-type-config.ts`
- Modify: `web/default/src/features/channels/components/drawers/channel-mutate-drawer.tsx`
- Modify: `web/classic/src/constants/channel.constants.js`
- Modify: `web/classic/src/helpers/render.jsx`
- Modify: `web/classic/src/components/table/channels/modals/EditChannelModal.jsx`

## Tasks

### Task 1: Add Backend Channel Type

- [x] Add `ChannelTypeTencentVODAIGC = 62` before `ChannelTypeDummy`.
- [x] Append default base URL `https://vod.tencentcloudapi.com`.
- [x] Add display name `TencentVODAIGC`.
- [x] Add adaptor routing in `relay/relay_adaptor.go`.
- [x] Extend `relay/relay_adaptor_test.go` to assert the new task adaptor is returned.

### Task 2: Add Config Parsing

- [x] Write tests for `SecretId|SecretKey|SubAppId` parsing.
- [x] Write tests for JSON API key parsing.
- [x] Implement config parser with region from channel `api_version`.
- [x] Reject missing secret id, secret key, sub app id, or region.

### Task 3: Add Model and Billing Matrix

- [x] Write tests for model mapping from public model ID to Tencent `ModelName` / `ModelVersion`.
- [x] Write tests for image resolution multipliers and output count.
- [x] Write tests for video resolution multipliers and duration.
- [x] Implement model matrix from `docs/integrations/tencent-vod-aigc.md`.
- [x] Implement `EstimateBilling` to return `resolution`, `duration`, `count`, and `task` ratios.

### Task 4: Add Tencent TC3 Signing

- [x] Write deterministic signing tests using fixed timestamp and request body.
- [x] Implement TC3-HMAC-SHA256 signer for VOD service.
- [x] Ensure no secret key is logged or returned in errors.

### Task 5: Add Submit and Poll Adaptor

- [x] Write tests for image task body conversion.
- [x] Write tests for video task body conversion.
- [x] Write tests for `DoResponse` extracting Tencent `Response.TaskId`.
- [x] Write tests for `FetchTask` sending `DescribeTaskDetail`.
- [x] Write tests for `ParseTaskResult` mapping success, processing, and failure states.
- [x] Implement `TaskAdaptor`.

### Task 6: Add Frontend Channel Form Support

- [x] Add Tencent VOD AIGC to default theme channel constants and display order.
- [x] Mark the channel as region-required in default theme.
- [x] Show helper text: `SecretId|SecretKey|SubAppId`.
- [x] Add Tencent VOD AIGC to classic theme constants and helper text.

### Task 7: Verification

- [x] Run `go test ./relay/channel/task/tencentvod ./relay`.
- [x] Run focused frontend typecheck/build if available.
- [x] Run `gofmt` on Go files.
- [x] Review git diff for accidental secrets or unrelated changes.

