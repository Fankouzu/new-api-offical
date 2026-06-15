# Tencent VOD AIGC Channel Integration

This document records the Tencent Cloud VOD AIGC integration contract for new-api.
It is the source of truth for the first implementation pass.

## Goal

Add Tencent Cloud VOD AIGC as a new async task channel. The channel must support
AI image and AI video generation through Tencent VOD APIs, and must charge users
with a deterministic resolution/duration/count multiplier model.

## Channel Configuration

Create a new channel type:

| Field | Value |
| --- | --- |
| Channel type | `TencentVODAIGC` |
| Default base URL | `https://vod.tencentcloudapi.com` |
| Required base URL input | Yes |
| Required region input | Yes, sent as `X-TC-Region` |
| Required API key input | Yes |
| Task style | Async submit + polling |

### API Key Format

The channel creation page must allow operators to enter:

- Base URL, normally `https://vod.tencentcloudapi.com`
- `X-TC-Region`, normally `ap-guangzhou`
- API key

API key supports two formats:

```text
SecretId|SecretKey|SubAppId
```

or JSON:

```json
{
  "secret_id": "AKID...",
  "secret_key": "...",
  "sub_app_id": 1500044236
}
```

The region is configured through the normal channel region field, not embedded
in the API key. This keeps the UI aligned with the requested base URL + region +
api-key model.

## How Operators Enter Models

On the channel creation page, select channel type `Tencent VOD AIGC`.

Recommended flow:

1. Set Base URL to `https://vod.tencentcloudapi.com`.
2. Set `X-TC-Region`, for example `ap-guangzhou`.
3. Set API key as `SecretId|SecretKey|SubAppId`.
4. Click `Fill Related Models` / `填入相关模型`.

The button fills all built-in Tencent VOD AIGC public model IDs documented in
the model matrix below. Operators may also manually enter a comma-separated
subset, for example:

```text
gv-3.1,vidu-q3-turbo,kling-image-3.0
```

These public model IDs are the names users call through new-api. The adaptor
maps them to Tencent's `ModelName` and `ModelVersion` internally.

## Tencent API Mapping

| new-api operation | Tencent action | Tencent API host |
| --- | --- | --- |
| Async image generation | `CreateAigcImageTask` | `vod.tencentcloudapi.com` |
| Async video generation | `CreateAigcVideoTask` | `vod.tencentcloudapi.com` |
| Poll task result | `DescribeTaskDetail` | `vod.tencentcloudapi.com` |

All requests are `POST /` with Tencent Cloud TC3-HMAC-SHA256 authentication.

Common Tencent headers:

```text
Content-Type: application/json; charset=utf-8
Host: vod.tencentcloudapi.com
X-TC-Action: CreateAigcVideoTask | CreateAigcImageTask | DescribeTaskDetail
X-TC-Version: 2018-07-17
X-TC-Timestamp: <unix seconds>
X-TC-Region: <configured region>
Authorization: TC3-HMAC-SHA256 ...
```

## Request Mapping

### Shared Request Fields

| new-api field | Tencent field | Notes |
| --- | --- | --- |
| `model` | `ModelName` + `ModelVersion` | Resolved by model matrix |
| `prompt` | `Prompt` | Required unless the specific mode allows media-only |
| `images` / `image` | `FileInfos` | URL mode only in first pass |
| `duration` / `seconds` | `OutputConfig.Duration` | Video only |
| `size` / `resolution` | `OutputConfig.Resolution` and optional `AspectRatio` | Normalized to Tencent tiers |
| metadata `aspect_ratio` | `OutputConfig.AspectRatio` | Examples: `16:9`, `9:16`, `1:1` |
| metadata `storage_mode` | `OutputConfig.StorageMode` | `Temporary` or `Permanent`, default `Temporary` |
| metadata `scene_type` | `SceneType` | For Kling/Vidu/Hunyuan scene features |
| metadata `ext_info` | `ExtInfo` | String or object marshalled to JSON string |
| metadata `audio_generation` | `OutputConfig.AudioGeneration` | `Enabled` / `Disabled` |
| metadata `input_region` | `InputRegion` | Use `oversea` when source media is overseas |
| metadata `last_frame_url` | `LastFrameUrl` | First/tail-frame video |
| metadata `negative_prompt` | `NegativePrompt` | If Tencent model supports it |
| metadata `enhance_prompt` | `EnhancePrompt` | `Enabled` / `Disabled` |
| metadata `seed` | `Seed` | If Tencent model supports it |
| metadata `output_image_count` / `n` | Local billing multiplier only | Tencent VOD currently rejects `OutputConfig.Count`; SI multi-image generation must be handled through documented `ExtInfo.AdditionalParameters` and prompt instructions. |

### FileInfos Mapping

First pass supports URL inputs only. Tencent uses different `FileInfos`
schemas for image and video tasks.

Image task file input:

```json
{
  "Type": "Url",
  "Url": "https://example.com/input.png"
}
```

Video task file input:

```json
{
  "Type": "Url",
  "Url": "https://example.com/input.png",
  "Category": "Image",
  "Usage": "Reference"
}
```

Usage rules:

- Image tasks must not send video-only fields such as `Category` or `Usage`.
- One image in image-to-video defaults to `FirstFrame`.
- Two images in video generation map to first-frame + `LastFrameUrl`, unless
  metadata explicitly sets per-file usage.
- Multiple images default to `Reference`.
- Metadata `file_infos` may override generated FileInfos for advanced Tencent
  parameters such as `ObjectId`, `Text`, `ReferenceType`, or `VoiceId`.

## Model Matrix

The public model ID is the new-api model. The Tencent fields are sent upstream.
Operators configure the lowest-resolution per-image or per-second selling price
for the public model in the normal model pricing backend.

### Image Models

Sources: Tencent Cloud `CreateAigcImageTask` API parameter documentation, the local VOD AIGC guide PDF, and Tencent Cloud AIGC pricing page. Some rows are listed in the PDF/pricing page but are inconsistent in the API `ModelName` enum; those rows are kept because the model marketplace and pricing documents list them. If Tencent support has not enabled a vendor/model for the account, the upstream API may still reject the task.

| Public model ID | Vendor | Tencent `ModelName` | Tencent `ModelVersion` | Resolution tiers | Billing unit | Source / notes |
| --- | --- | --- | --- | --- | --- | --- |
| `og-image2-low` | OG / Image2 | `OG` | `image2_low` | `1K`, `2K`, `4K` | per image | Official API + pricing. |
| `og-image2-medium` | OG / Image2 | `OG` | `image2_medium` | `1K`, `2K`, `4K` | per image | Official API + pricing. |
| `og-image2-high` | OG / Image2 | `OG` | `image2_high` | `1K`, `2K`, `4K` | per image | Official API + pricing. |
| `gg-2.5-image` | GG / Gemini image | `GG` | `2.5` | `1K`, `2K`, `4K` | per image | Official API + pricing; PDF says 2.5 maps to Nano Banana. |
| `gg-3.0-image` | GG / Gemini image | `GG` | `3.0` | `1K`, `2K`, `4K` | per image | Official API + pricing; PDF says 3.0 maps to Nano Banana Pro. |
| `gg-3.1-image` | GG / Gemini image | `GG` | `3.1` | `512P`, `1K`, `2K`, `4K` | per image | Official API + pricing; PDF says 3.1 maps to nano2. |
| `si-4.0-image` | SI | `SI` | `4.0` | `1K`, `2K`, `4K` | per image | Official API; PDF notes SI parameters may require business confirmation. |
| `si-4.5-image` | SI | `SI` | `4.5` | `1K`, `2K`, `4K` | per image | Official API; PDF notes SI parameters may require business confirmation. |
| `si-5.0-lite-image` | SI | `SI` | `5.0-lite` | `1K`, `2K`, `4K` | per image | Official API; PDF explicitly lists SI 5.0-lite. |
| `qwen-0925-image` | Qwen | `Qwen` | `0925` | `1K`, `2K`, `4K` | per image | Official API + pricing. |
| `hunyuan-3.0-image` | Hunyuan | `Hunyuan` | `3.0` | `1K`, `2K`, `4K` | per image | Official API + pricing. |
| `hunyuan-3d-panorama` | Hunyuan | `Hunyuan` | `3d_2.0` | `1K`, `2K`, `4K` | per image/task | PDF guide; requires `SceneType=3d_panorama`; guide notes 5 yuan/task. |
| `vidu-q2-image` | Vidu | `Vidu` | `q2` | `1K`, `2K`, `4K` | per image | Official API + pricing. |
| `kling-2.1-image` | Kling | `Kling` | `2.1` | `1K`, `2K`, `4K` | per image | Official API + pricing. |
| `kling-image-3.0` | Kling | `Kling` | `3.0` | `1K`, `2K`, `4K` | per image | Official API + pricing. |
| `kling-image-3.0-omni` | Kling | `Kling` | `3.0-Omni` | `1K`, `2K`, `4K` | per image | Official API + pricing. |
| `kling-o1-image` | Kling | `Kling` | `O1` | `1K`, `2K`, `4K` | per image | Official API + pricing. |
| `kling-scene-image` | Kling | `Kling` | `scene` | `1K`, `2K`, `4K` | per image | Official API; scene image workflow. |
| `mj-v7-image` | MJ | `MJ` | `v7` | prompt-controlled | per image | PDF + pricing list MJ v7; API parameter notes list MJ v7 reference-image limit although `ModelName` enum is inconsistent. |
| `jimeng-4.0-image` | Jimeng | `Jimeng` | `4.0` | model-specific | per image | API parameter notes list Jimeng 4.0, but `ModelName` enum is inconsistent; keep as PDF/API-note candidate. |

### Video Models

Sources: Tencent Cloud `CreateAigcVideoTask` API parameter documentation, the local VOD AIGC guide PDF, and Tencent Cloud AIGC pricing page. Some provider/model names are from the PDF model marketplace and pricing page and may require Tencent business enablement before live use.

| Public model ID | Vendor | Tencent `ModelName` | Tencent `ModelVersion` | Resolution tiers | Duration | Billing unit | Source / notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `kling-1.6` | Kling | `Kling` | `1.6` | `720P`, `1080P`, `2K`, `4K` | model-specific | per second | Pricing page lists 1.6/2.0/2.1. |
| `kling-2.0` | Kling | `Kling` | `2.0` | `720P`, `1080P`, `2K`, `4K` | model-specific | per second | Pricing page lists 1.6/2.0/2.1. |
| `kling-2.1` | Kling | `Kling` | `2.1` | `720P`, `1080P`, `2K`, `4K` | model-specific | per second | Pricing page + API notes. |
| `kling-2.5-pro` | Kling | `Kling` | `2.5-pro` | `720P`, `1080P`, `2K`, `4K` | model-specific | per second | Pricing page. |
| `kling-2.6` | Kling | `Kling` | `2.6` | `720P`, `1080P`, `2K`, `4K` | model-specific | per second | PDF + pricing; supports first/tail frame, action/lip/avatar modes. |
| `kling-2.6-motion-control` | Kling | `Kling` | `2.6` | `720P`, `1080P`, `2K`, `4K` | model-specific | per second | Uses `SceneType=motion_control`. |
| `kling-3.0` | Kling | `Kling` | `3.0` | `720P`, `1080P`, `2K`, `4K` | 3-15s | per second | Official/PDF; std/pro are resolution/price lanes, not Tencent `ModelVersion`. |
| `kling-3.0-omni` | Kling | `Kling` | `3.0-Omni` | `720P`, `1080P`, `2K`, `4K` | 3-15s | per second | Official/PDF; supports audio parameters. |
| `kling-avatar` | Kling | `Kling` | `avater` | `720P`, `1080P`, `2K`, `4K` | model-specific | per second | Pricing page spelling is `avater`; keep upstream value as documented. |
| `kling-identifyface` | Kling | `Kling` | `Identifyface` | 720P lane | per 5 seconds | per second | Pricing page; lip-sync/face identification mode. |
| `jimeng-3.0-pro` | Jimeng | `Jimeng` | `3.0pro` | model-specific | model-specific | per second | PDF model marketplace. |
| `jimeng-4.0` | Jimeng | `Jimeng` | `4.0` | model-specific | model-specific | per second | PDF model marketplace. |
| `sv-1.0-pro` | SV | `SV` | `1.0-pro` | model-specific | model-specific | per second | PDF model marketplace; SV params may require business confirmation. |
| `sv-1.0-lite-i2v` | SV | `SV` | `1.0-lite-i2v` | model-specific | model-specific | per second | PDF model marketplace; image-to-video lane. |
| `vidu-q2` | Vidu | `Vidu` | `q2` | `540P`, `720P`, `1080P`, `2K`, `4K` | model-specific | per second | PDF + Vidu guide. |
| `vidu-q2-turbo` | Vidu | `Vidu` | `q2-turbo` | `540P`, `720P`, `1080P`, `2K`, `4K` | model-specific | per second | PDF + Vidu guide. |
| `vidu-q2-pro` | Vidu | `Vidu` | `q2-pro` | `540P`, `720P`, `1080P`, `2K`, `4K` | model-specific | per second | PDF + Vidu guide. |
| `vidu-q2-pro-fast` | Vidu | `Vidu` | `q2-pro-fast` | `540P`, `720P`, `1080P`, `2K`, `4K` | model-specific | per second | Vidu guide supported-mode list. |
| `vidu-q3` | Vidu | `Vidu` | `q3` | `540P`, `720P`, `1080P`, `2K`, `4K` | 3-15s | per second | PDF + pricing. |
| `vidu-q3-turbo` | Vidu | `Vidu` | `q3-turbo` | `540P`, `720P`, `1080P`, `2K`, `4K` | 3-15s | per second | Official/PDF + pricing. |
| `vidu-q3-pro` | Vidu | `Vidu` | `q3-pro` | `540P`, `720P`, `1080P`, `2K`, `4K` | 3-15s | per second | PDF + pricing. |
| `vidu-q3-mix` | Vidu | `Vidu` | `q3-mix` | `540P`, `720P`, `1080P`, `2K`, `4K` | 3-15s | per second | PDF + pricing; reference-generation only in current guide. |
| `hunyuan-1.5` | Hunyuan | `Hunyuan` | `1.5` | `720P`, `1080P`, `2K`, `4K` | model-specific | per second | Pricing page. |
| `hunyuan-3d-scene` | Hunyuan | `Hunyuan` | `3d_2.0` | custom | task-based | per task | PDF guide; requires `SceneType=3d_scene`; guide notes 200 yuan/task. |
| `h2-1.0` | H2 | `H2` | `1.0` | `720P`, `1080P`, `2K`, `4K` | 3-15s | per second | PDF guide. |
| `hailuo-02` | Hailuo | `Hailuo` | `02` | `768P`, `1080P`, `2K`, `4K` | model-specific | per second | Official/PDF + pricing. |
| `hailuo-2.3` | Hailuo | `Hailuo` | `2.3` | `768P`, `1080P`, `2K`, `4K` | model-specific | per second | Official/PDF + pricing. |
| `hailuo-2.3-fast` | Hailuo | `Hailuo` | `2.3-fast` | `768P`, `1080P`, `2K`, `4K` | model-specific | per second | Pricing page + PDF changelog. |
| `gv-3.1` | Google Veo / GV | `GV` | `3.1` | `720P`, `1080P`, `2K`, `4K` | model-specific | per second | Official/PDF + pricing; supports audio/no-audio lanes. |
| `gv-3.1-fast` | Google Veo / GV | `GV` | `3.1-fast` | `720P`, `1080P`, `2K`, `4K` | model-specific | per second | PDF + pricing; faster lower-cost lane. |
| `gv-3.1-lite` | Google Veo / GV | `GV` | `3.1-lite` | `720P`, `1080P`, `2K`, `4K` | model-specific | per second | Pricing page. |
| `os-2.0` | OpenAI Sora / OS | `OS` | `2.0` | `720P`, `1080P`, `2K`, `4K` | model-specific | per second | Official/PDF + pricing. |
| `pixverse-v5.6` | PixVerse | `PixVerse` | `v5.6` | `360P`, `540P`, `720P`, `1080P`, `2K`, `4K` | 1-15s | per second | Official/PDF + pricing. |
| `pixverse-v6` | PixVerse | `PixVerse` | `v6` | `360P`, `540P`, `720P`, `1080P`, `2K`, `4K` | 1-15s | per second | Official/PDF + pricing. |
| `pixverse-c1` | PixVerse | `PixVerse` | `c1` | `360P`, `540P`, `720P`, `1080P`, `2K`, `4K` | 1-15s | per second | Official/PDF + pricing. |
| `mingmou-1.0` | Mingmou | `Mingmou` | `1.0` | `720P`, `1080P`, `2K`, `4K` | model-specific | per second | Official API/pricing. |

## Billing Model

### Principle

The backend model price stores the lowest-resolution selling price:

- Image models: lowest-tier price per output image.
- Video models: lowest-tier price per output second.
- Task-based special models: price per task.

The adaptor computes `OtherRatios` from request parameters. Existing task
billing then multiplies the backend model price by all ratios.

### Formulas

Image:

```text
quota = base_model_price_per_image * resolution_multiplier * output_image_count
```

Video:

```text
quota = base_model_price_per_second * resolution_multiplier * duration_seconds * feature_multiplier
```

Task-based:

```text
quota = base_task_price * feature_multiplier
```

### Resolution Multiplier

The multiplier is based on Tencent's relative pricing:

```text
resolution_multiplier = official_price[current_tier] / official_price[base_tier]
```

The implementation must not charge users directly from Tencent CNY prices.
Tencent CNY prices are only used to derive stable relative multipliers. If the
operator changes the backend lowest-tier model price, all higher-resolution
prices scale automatically.

### Built-In Multiplier Matrix

The first implementation includes a conservative built-in matrix. Values can be
updated when Tencent updates the official price table.

Generic image tiers:

| Tier | Multiplier |
| --- | ---: |
| `512P` | 1.0 |
| `1K` | 1.0 |
| `2K` | 1.4 |
| `4K` | 1.8 |

Generic video tiers:

| Tier | Multiplier |
| --- | ---: |
| `360P` | 1.0 |
| `480P` | 1.0 |
| `540P` | 1.1 |
| `720P` | 1.5 |
| `768P` | 1.5 |
| `1080P` | 1.75 |
| `2K` | 2.1 |
| `4K` | 2.5 |

Known Vidu q3-turbo ratio example from the requested billing model:

| Tier | Official example CNY/s | Multiplier from 480P |
| --- | ---: | ---: |
| `480P` | 0.250 | 1.000 |
| `720P` | 0.375 | 1.500 |
| `1080P` | 0.438 | 1.752 |
| `2K` | 0.526 | 2.104 |
| `4K` | 0.631 | 2.524 |

Known Hunyuan 3.0 image ratio example from the requested billing model:

| Tier | Official example CNY/image | Multiplier from 1K |
| --- | ---: | ---: |
| `1K` | 0.200 | 1.000 |
| `2K` | 0.280 | 1.400 |
| `4K` | 0.360 | 1.800 |

### Other Ratios

| Ratio key | Meaning | Applies to |
| --- | --- | --- |
| `resolution` | Resolution multiplier | Image/video |
| `duration` | Duration in seconds | Video |
| `count` | Output image count | Image |
| `task` | Fixed task multiplier | Special task models |
| `feature` | Feature/mode surcharge | Optional scene features |

### Validation Rules

- Missing image count defaults to 1.
- Missing video duration defaults to 5 seconds.
- Missing resolution defaults to the model's base tier.
- Unsupported resolution returns a `400` task error before pre-consume.
- Output image count must be positive.
- Video duration must be positive and within model-specific min/max when known.
- If Tencent returns failure, the existing task failure path refunds pre-consume.
- If Tencent later returns actual duration/count/resolution, `AdjustBillingOnComplete`
  may return an adjusted quota.

## Implementation Files

Backend:

- `constant/channel.go`: add channel type, base URL, display name.
- `common/api_type.go`: route task channel type as task-capable if required.
- `relay/relay_adaptor.go`: return Tencent VOD task adaptor.
- `relay/channel/task/tencentvod/`: new adaptor, signing, model matrix, billing.
- `relay/channel/tencent/relay-tencent.go`: use as signature reference only.

Frontend:

- `web/default/src/features/channels/constants.ts`: add type display name and order.
- `web/default/src/features/channels/lib/channel-utils.ts`: add icon mapping.
- `web/default/src/features/channels/lib/channel-type-config.ts`: require region.
- `web/default/src/features/channels/components/drawers/channel-mutate-drawer.tsx`:
  display key helper text for `SecretId|SecretKey|SubAppId`.
- `web/classic/src/constants/channel.constants.js`: add type.
- `web/classic/src/helpers/render.jsx`: add icon fallback.
- `web/classic/src/components/table/channels/modals/EditChannelModal.jsx`:
  display key helper text and region input.

Tests:

- `relay/channel/task/tencentvod/config_test.go`
- `relay/channel/task/tencentvod/billing_test.go`
- `relay/channel/task/tencentvod/sign_test.go`
- `relay/channel/task/tencentvod/adaptor_test.go`
- `relay/relay_adaptor_test.go`

## Delivery Scope

First PR scope:

- URL input only.
- `CreateAigcImageTask`.
- `CreateAigcVideoTask`.
- `DescribeTaskDetail` polling.
- TC3 signing.
- Resolution/duration/count billing multipliers.
- Channel UI supports base URL, region, and API key.

Out of scope for first PR:

- VOD upload to FileId.
- Reliable callback consumption.
- SceneAigcImageTask / SceneAigcVideoTask wrappers.
- Audio-only `CreateAigcAudioTask`.
- Tencent account balance query.
