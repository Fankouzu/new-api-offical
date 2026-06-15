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
| metadata `output_image_count` / `n` | `OutputConfig.OutputImageCount` | Image only |

### FileInfos Mapping

First pass supports URL inputs only.

```json
{
  "Type": "Url",
  "Category": "Image",
  "Url": "https://example.com/input.png",
  "Usage": "Reference"
}
```

Usage rules:

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

| Public model ID | Vendor | Tencent `ModelName` | Tencent `ModelVersion` | Supported modes | Resolution tiers | Billing unit | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `hunyuan-3d-panorama` | Hunyuan | `Hunyuan` | `3d_2.0` | text/image to panorama | `1K`, `2K`, `4K` | per image | Requires `SceneType=3d_panorama`; document notes 5 yuan/task for panorama. |
| `og-image2-low` | OG / Image2 | `OG` | `image2_low` | text/image to image | `1K`, `2K`, `4K` | per image | Up to 16 input images in Tencent guide. |
| `og-image2-medium` | OG / Image2 | `OG` | `image2_medium` | text/image to image | `1K`, `2K`, `4K` | per image | Same API shape as low. |
| `og-image2-high` | OG / Image2 | `OG` | `image2_high` | text/image to image | `1K`, `2K`, `4K` | per image | Same API shape as low. |
| `kling-image-3.0` | Kling | `Kling` | `3.0` | text/image/reference image | `1K`, `2K`, `4K` | per image | Supports scene features through `SceneType`. |
| `kling-image-3.0-omni` | Kling | `Kling` | `3.0-Omni` | text/image/reference image | `1K`, `2K`, `4K` | per image | Higher tier Kling image model. |
| `vidu-image` | Vidu | `Vidu` | provider-specific | text/image/reference image | `1K`, `2K`, `4K` | per image | Vidu 1K uses short-edge semantics in Tencent pricing notes. |

### Video Models

| Public model ID | Vendor | Tencent `ModelName` | Tencent `ModelVersion` | Supported modes | Resolution tiers | Duration | Billing unit | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `gv-3.1` | Google Veo / GV | `GV` | `3.1` | text/image/first-tail/reference video | `720P`, `1080P`, `2K`, `4K` | model-specific | per second | `ReferenceType=asset/style` supported. |
| `gv-3.1-fast` | Google Veo / GV | `GV` | `3.1-fast` | text/image | `720P`, `1080P`, `2K`, `4K` | model-specific | per second | Faster lower-cost variant. |
| `kling-3.0-std` | Kling | `Kling` | `3.0` | text/image/reference/scene | `720P` | 3-15s | per second | Standard 720P lane. |
| `kling-3.0-pro` | Kling | `Kling` | `3.0` | text/image/reference/scene | `1080P` | 3-15s | per second | Pro 1080P lane. |
| `kling-3.0-omni` | Kling | `Kling` | `3.0-Omni` | text/image/reference/scene | `4K` | 3-15s | per second | 4K lane, supports audio parameters in guide. |
| `kling-2.6` | Kling | `Kling` | `2.6` | text/image/action/lip-sync/avatar | `720P`, `1080P` | model-specific | per second | Voice ID note: Kling 2.6 only supports voice ID at 1080P. |
| `vidu-q3-turbo` | Vidu | `Vidu` | `q3-turbo` | text/image/reference/subject | `480P`, `720P`, `1080P`, `2K`, `4K` | 3-15s | per second | Supports off-peak, interpolation, logo, subject modes. |
| `vidu-q3-pro` | Vidu | `Vidu` | `q3-pro` | text/image/reference/subject | `720P`, `1080P`, `2K`, `4K` | 3-15s | per second | Higher-quality q3 lane. |
| `vidu-q3-mix` | Vidu | `Vidu` | `q3-mix` | text/image/reference | `720P`, `1080P`, `2K`, `4K` | 3-15s | per second | Guide notes stronger dynamics; no subject library in current guide. |
| `pixverse-v5.6` | PixVerse | `PixVerse` | `v5.6` | text/image/reference/video edit | `360P`, `540P`, `720P`, `1080P` | 1-15s | per second | 1080P does not support 10s in guide note. |
| `pixverse-v6` | PixVerse | `PixVerse` | `v6` | text/image/reference/video edit | `360P`, `540P`, `720P`, `1080P` | 1-15s | per second | Native audio/video generation. |
| `pixverse-c1` | PixVerse | `PixVerse` | `c1` | cinematic reference/video edit | `360P`, `540P`, `720P`, `1080P` | 1-15s | per second | Film/action/effects-oriented model. |
| `hailuo-02` | Hailuo | `Hailuo` | `02` | text/image video | `768P`, `1080P` | model-specific | per second | Guide lists Hailuo versions `02`, `2.3`, `2.3-fast`. |
| `hailuo-2.3` | Hailuo | `Hailuo` | `2.3` | text/image video | `768P`, `1080P` | model-specific | per second | Higher-quality lane. |
| `hailuo-2.3-fast` | Hailuo | `Hailuo` | `2.3-fast` | text/image video | `768P`, `1080P` | model-specific | per second | Faster lane. |
| `h2-1.0` | H2 | `H2` | `1.0` | text/first-frame/reference images | `720P`, `1080P`, `2K`, `4K` | 3-15s | per second | Supports audio; reference images 1-9. |
| `hunyuan-3d-scene` | Hunyuan | `Hunyuan` | `3d_2.0` | 3D scene video | custom | task-based | per task | Requires `SceneType=3d_scene`; guide notes 200 yuan/task. |

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
