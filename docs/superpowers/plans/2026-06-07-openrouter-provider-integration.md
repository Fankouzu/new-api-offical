# OpenRouter Provider Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make this gateway eligible for OpenRouter provider onboarding by exposing a provider-grade model catalog and tightening provider-facing runtime behavior.

**Architecture:** Add a dedicated OpenRouter provider catalog surface instead of changing the existing OpenAI-compatible `/v1/models` response for all clients. Reuse existing model, pricing, and endpoint metadata where possible, and keep missing OpenRouter-only metadata represented by explicit defaults/config parsing so the response is stable and testable.

**Tech Stack:** Go 1.22+, Gin, GORM, existing `common` JSON wrappers, existing model pricing cache, `go test`.

---

## File Structure

- Create `dto/openrouter_provider.go`
  - Defines OpenRouter provider model response DTOs: model, pricing, datacenter, slug wrapper.
  - Keeps pricing fields as strings to avoid floating point precision issues, matching OpenRouter's contract.

- Create `service/openrouter_provider_catalog.go`
  - Converts existing `model.Pricing` records into OpenRouter provider models.
  - Derives default modalities/features from `SupportedEndpointTypes`.
  - Converts local ratio pricing back to OpenRouter per-token USD strings.
  - Parses optional model metadata from `model.Model.Tags` using conservative `or:*` tags.

- Create `service/openrouter_provider_catalog_test.go`
  - Locks price conversion, endpoint-to-modality mapping, optional metadata parsing, and free-model behavior.

- Modify `controller/model.go`
  - Adds `ListOpenRouterProviderModels(c *gin.Context)` handler using the service catalog builder.

- Modify `router/relay-router.go`
  - Adds `GET /openrouter/v1/models` behind existing `TokenAuth`.
  - Leaves `/v1/models` unchanged for normal OpenAI clients.

- Create `controller/openrouter_provider_model_test.go`
  - Verifies the new handler returns OpenRouter fields, filters unavailable/unpriced models consistently, and does not return the legacy `success` wrapper.

- Modify `middleware/model-rate-limit.go`
  - Ensures memory-based model request rate limits return an OpenAI-style error JSON body instead of an empty 429.
  - Adds `Retry-After` for provider-facing 429 responses.

- Create `middleware/model-rate-limit_test.go`
  - Verifies 429 body and `Retry-After` behavior.

## OpenRouter Contract Coverage

- Required model catalog fields:
  - `id`: existing model name.
  - `hugging_face_id`: parsed from tag `or:hf=<id>`, default `""`.
  - `name`: parsed from tag `or:name=<display>`, default model id.
  - `created`: existing static fallback unless model metadata has `CreatedTime`.
  - `input_modalities` / `output_modalities`: derived from endpoint types and optional tags.
  - `quantization`: parsed from tag `or:quantization=<value>`, default `""`.
  - `context_length`: parsed from tag `or:context=<int>`, default `0`.
  - `max_output_length`: parsed from tag `or:max_output=<int>`, default `0`.
  - `pricing`: local ratio/model price converted to USD strings.
  - `supported_sampling_parameters`: conservative default list for text chat models.
  - `supported_features`: derived from endpoint type plus optional tags.

- Optional model catalog fields:
  - `description`: existing `model.Model.Description`.
  - `deprecation_date`: parsed from tag `or:deprecation=<date>`.
  - `is_ready`: parsed from tag `or:ready=false`, default omitted/true behavior represented as `true`.
  - `is_free`: true when all relevant pricing fields are zero.
  - `openrouter.slug`: parsed from tag `or:slug=<slug>`, default model id.
  - `datacenters`: parsed from tag `or:dc=<ISO2>`, repeatable via comma-separated values.

## Task 1: Provider DTO and Catalog Mapping

**Files:**
- Create: `dto/openrouter_provider.go`
- Create: `service/openrouter_provider_catalog.go`
- Create: `service/openrouter_provider_catalog_test.go`

- [x] **Step 1: Write failing service tests**

Add tests that call `service.BuildOpenRouterProviderModels` with synthetic `model.Pricing` values:

```go
func TestBuildOpenRouterProviderModelsConvertsTextPricingAndMetadata(t *testing.T) {
	items := []model.Pricing{{
		ModelName: "acme/test-chat",
		Description: "A test model",
		Tags: "or:hf=acme/test-chat-hf,or:name=Acme Test Chat,or:context=128000,or:max_output=8192,or:quantization=fp16,or:dc=US,or:feature=tools",
		QuotaType: 0,
		ModelRatio: 0.001,
		CompletionRatio: 2,
		SupportedEndpointTypes: []constant.EndpointType{constant.EndpointTypeChat},
		EnableGroup: []string{"default"},
	}}

	result := BuildOpenRouterProviderModels(items)

	require.Len(t, result, 1)
	got := result[0]
	require.Equal(t, "acme/test-chat", got.ID)
	require.Equal(t, "acme/test-chat-hf", got.HuggingFaceID)
	require.Equal(t, "Acme Test Chat", got.Name)
	require.Equal(t, int64(128000), got.ContextLength)
	require.Equal(t, int64(8192), got.MaxOutputLength)
	require.Equal(t, "fp16", got.Quantization)
	require.Equal(t, []string{"text"}, got.InputModalities)
	require.Equal(t, []string{"text"}, got.OutputModalities)
	require.Contains(t, got.SupportedFeatures, "tools")
	require.Equal(t, "0.000000002", got.Pricing.Prompt)
	require.Equal(t, "0.000000004", got.Pricing.Completion)
	require.Equal(t, "acme/test-chat", got.OpenRouter.Slug)
	require.Equal(t, "US", got.Datacenters[0].CountryCode)
}
```

- [x] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./service -run TestBuildOpenRouterProviderModels -count=1
```

Expected: fail because `BuildOpenRouterProviderModels` and DTOs do not exist.

- [x] **Step 3: Implement DTOs and minimal catalog mapping**

Implement `dto.OpenRouterProviderModel`, `dto.OpenRouterProviderPricing`, `dto.OpenRouterDatacenter`, `dto.OpenRouterSlug`.

Implement `service.BuildOpenRouterProviderModels(pricings []model.Pricing) []dto.OpenRouterProviderModel`.

- [x] **Step 4: Run service tests to verify green**

Run:

```bash
go test ./service -run TestBuildOpenRouterProviderModels -count=1
```

Expected: pass.

## Task 2: Provider Models Handler and Route

**Files:**
- Modify: `controller/model.go`
- Modify: `router/relay-router.go`
- Create: `controller/openrouter_provider_model_test.go`

- [x] **Step 1: Write failing controller test**

Add a test that seeds an enabled user, ability, model metadata, and ratio settings, calls `ListOpenRouterProviderModels`, and asserts:

- HTTP 200.
- Top-level shape is `{ "data": [...] }`.
- No legacy `success` field.
- Returned item includes `pricing`, `input_modalities`, `output_modalities`, and `supported_features`.

- [x] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./controller -run TestListOpenRouterProviderModels -count=1
```

Expected: fail because handler does not exist.

- [x] **Step 3: Implement controller handler**

Add:

```go
func ListOpenRouterProviderModels(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"data": service.BuildOpenRouterProviderModels(model.GetPricing()),
	})
}
```

- [x] **Step 4: Register route**

Add route group:

```go
openRouterProviderModelsRouter := router.Group("/openrouter/v1/models")
openRouterProviderModelsRouter.Use(middleware.RouteTag("relay"))
openRouterProviderModelsRouter.Use(middleware.TokenAuth())
openRouterProviderModelsRouter.GET("", controller.ListOpenRouterProviderModels)
```

- [x] **Step 5: Run controller test to verify green**

Run:

```bash
go test ./controller -run TestListOpenRouterProviderModels -count=1
```

Expected: pass.

## Task 3: Structured 429 Runtime Behavior

**Files:**
- Modify: `middleware/model-rate-limit.go`
- Create: `middleware/model-rate-limit_test.go`

- [x] **Step 1: Write failing middleware tests**

Add tests for the memory branch that force rate limit rejection and assert:

- Status is 429.
- Response contains OpenAI-compatible `error.message`, `error.type`, and `error.code`.
- `Retry-After` header is set to the configured window duration.

- [x] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./middleware -run TestModelRequestRateLimitReturnsStructured429 -count=1
```

Expected: fail because memory branch currently returns empty 429.

- [x] **Step 3: Implement structured model-rate-limit abort helper**

Update both Redis and memory limit rejection paths to use the existing `abortWithOpenAiMessage` helper and set `Retry-After`.

- [x] **Step 4: Run middleware tests to verify green**

Run:

```bash
go test ./middleware -run TestModelRequestRateLimitReturnsStructured429 -count=1
```

Expected: pass.

## Task 4: Integration Verification

**Files:**
- Existing backend packages only.

- [x] **Step 1: Run focused package tests**

Run:

```bash
go test ./dto ./service ./controller ./middleware -count=1
```

Expected: pass.

- [x] **Step 2: Run broader backend smoke tests**

Run:

```bash
go test ./... -count=1
```

Expected: pass or identify unrelated known failures with evidence.

- [x] **Step 3: Inspect final diff**

Run:

```bash
git diff --stat
git diff -- docs/superpowers/plans/2026-06-07-openrouter-provider-integration.md dto/openrouter_provider.go service/openrouter_provider_catalog.go controller/model.go router/relay-router.go middleware/model-rate-limit.go
```

Expected: changes are limited to OpenRouter provider integration and structured rate-limit behavior.

## Self-Review

- Spec coverage: Covers OpenRouter model-list required fields, dedicated endpoint, pricing string conversion, readiness/free/deprecation metadata, modalities/features, 429 behavior, SSE keep-alive already exists and remains unchanged.
- Placeholder scan: No task uses deferred placeholders; metadata defaults are explicit.
- Type consistency: DTO and service names are consistent across controller and tests.

## Execution Notes

- Focused tests passed:
  - `go test ./service -run TestBuildOpenRouterProviderModels -count=1`
  - `go test ./controller -run TestListOpenRouterProviderModels -count=1`
  - `go test ./middleware -run TestModelRequestRateLimitReturnsStructured429 -count=1`
- Affected package tests passed:
  - `go test ./service ./controller ./middleware -count=1`
- Full suite command `go test ./... -count=1` was run. It failed in existing relay tests outside this change:
  - `github.com/QuantumNous/new-api/relay`: `TestGetTaskAdaptorReturnsSub2APIAsyncAdaptor` expected two Sub2API-async models, got 3.
  - `github.com/QuantumNous/new-api/relay/channel/claude`: `TestRequestOpenAI2ClaudeMessage_IgnoresUnsupportedFileContent` expected 1 item, got 2.
