# Tencent VOD AIGC Pricing Expression Plan

This document defines the next-generation billing design for Tencent VOD AIGC
image/video models. It replaces the current coarse "lowest price * generic
resolution multiplier" approach with model-specific pricing tables and
expression-based settlement.

Source baseline:

- Tencent Cloud VOD pay-as-you-go pricing page:
  `https://cloud.tencent.com/document/product/266/95125`
- Local integration contract:
  `docs/integrations/tencent-vod-aigc.md`
- Current Tencent VOD model matrix:
  `relay/channel/task/tencentvod/models.go`
- Billing expression contract:
  `pkg/billingexpr/expr.md`

Important: Tencent states that AIGC prices can change with model/provider
updates. The values below should be treated as an importable price snapshot, not
as permanent constants.

## Problems With The Current Multiplier Model

The current implementation stores the lowest-tier selling price in the backend
and multiplies it by generic resolution ratios.

That is no longer accurate enough because Tencent VOD prices vary by:

- model vendor and model version;
- output kind: image, video, scene/task;
- resolution tier;
- generation mode, for example text-to-video, reference generation, first/last
  frame, no-reference video, reference video;
- audio mode for Kling/GV/PixVerse video models;
- off-peak mode for Vidu;
- special per-task scene capabilities;
- special duration rounding, for example Kling Identifyface bills every started
  5-second block as 5 seconds.

Generic ratios such as `720P=1.5` happen to match some Vidu examples but do not
cover Kling 3.0, GV 3.1, PixVerse, Vidu reference lanes, or special task
models.

## Normalized Billing Dimensions

All Tencent VOD AIGC models can be represented by one of three billing units.

| Unit | Formula shape | Examples |
| --- | --- | --- |
| `image` | `unit_price(model, mode, resolution) * image_count` | Hunyuan image, Qwen image, Kling image, OG image |
| `second` | `unit_price(model, mode, resolution) * bill_seconds` | Vidu, Kling video, GV, Hailuo, PixVerse |
| `task` | `unit_price(model, task_type)` | Hunyuan 3D scene/panorama, Vidu subject recognition |

Settlement should use actual upstream output facts when available. Pre-consume
should use requested/default values and write a pricing snapshot to the task so
historical logs remain auditable after price edits.

## Recommended Expression Variables

Extend `pkg/billingexpr` for task/image/video billing with these variables.

| Variable | Type | Meaning |
| --- | --- | --- |
| `kind` | string | `image`, `video`, or `task` |
| `model` | string | public model ID, for example `vidu-q3-turbo` |
| `vendor` | string | Tencent vendor name, for example `Vidu` |
| `mode` | string | normalized mode, for example `text`, `reference`, `i2v`, `with_audio` |
| `resolution` | string | normalized billing tier, for example `720P`, `1080P`, `4K` |
| `duration` | number | requested duration in seconds |
| `actual_duration` | number | upstream result duration if known |
| `bill_sec` | number | seconds used for billing after rounding/defaulting |
| `image_count` | number | requested/actual output image count |
| `has_audio` | bool | whether audio output is enabled |
| `has_reference` | bool | whether request has reference media |
| `off_peak` | bool | Vidu off-peak mode |
| `price_key` | string | resolved key for the selected pricing row |

Add task/image/video functions:

| Function | Signature | Purpose |
| --- | --- | --- |
| `price` | `price(key) -> float64` | Lookup a unit price from the frozen model price table |
| `bill_seconds` | `bill_seconds(duration, step, min) -> float64` | Apply duration rounding rules |
| `price_key` | `price_key(parts...) -> string` | Build a stable pricing row key |

The expression engine should still follow the existing "one expression, one
truth" rule. The price table is the expression's data source and must be
versioned/snapshotted with the expression.

## Expression Patterns

### Image Per Output

```text
tier(price_key(model, mode, resolution), price(price_key(model, mode, resolution)) * image_count)
```

Example for Hunyuan 3.0 image, 4K, 2 outputs:

```text
price("hunyuan-3.0-image/default/4K") * 2
```

### Video Per Second

```text
tier(price_key(model, mode, resolution), price(price_key(model, mode, resolution)) * bill_sec)
```

Example for Vidu q3-turbo, 720/768P, 5 seconds:

```text
price("vidu-q3-turbo/i2v/720P") * 5
```

### Rounded Per 5 Seconds

```text
tier("kling-identifyface/5s", price("kling-identifyface/default/720P") * bill_seconds(duration, 5, 5))
```

### Fixed Task

```text
tier(price_key(model, "task"), price(price_key(model, "task")))
```

## Official Price Matrix Snapshot

Currency is CNY. Image unit is CNY/image. Video unit is CNY/second. Task unit is
CNY/task.

### Image Models With Public Pricing

| Public model | Price key mode | 512P | 1K | 2K | 4K | Notes |
| --- | --- | ---: | ---: | ---: | ---: | --- |
| `hunyuan-3.0-image` | `default` | - | 0.200 | 0.280 | 0.360 | Tencent table: Hunyuan 3.0 |
| `qwen-0925-image` | `default` | - | 0.300 | 0.380 | 0.460 | Tencent table: Qwen 0925 |
| `og-image2-low` | `default` | - | 0.300 | 0.338 | 0.398 | Tencent table: OG image2 low |
| `og-image2-medium` | `default` | - | 0.638 | 1.050 | 1.583 | Tencent table: OG image2 medium |
| `og-image2-high` | `default` | - | 1.838 | 3.450 | 5.588 | Tencent table: OG image2 high |
| `gg-3.1-image` | `default` | 0.333 | 0.500 | 0.750 | 1.120 | Tencent table: GG 3.1 |
| `gg-3.0-image` | `default` | - | 1.000 | 1.000 | 1.800 | Tencent notes GG 2.5 2K/4K trial behavior separately |
| `gg-2.5-image` | `default` | - | 0.300 | 0.380 | 0.460 | Tencent table: GG 2.5 |
| `vidu-q2-image` | `text_to_image` | - | 0.188 | 0.250 | 0.313 | Vidu q2 text image |
| `vidu-q2-image` | `reference_1_3` | - | 0.250 | 0.375 | 0.500 | Vidu q2 reference 1-3 images |
| `vidu-q2-image` | `reference_4_7` | - | 0.313 | 0.625 | 0.938 | Vidu q2 reference 4-7 images |
| `mj-v7-image` | `default` | - | 0.300 | 0.380 | 0.460 | Tencent table: MJ v7 |
| `kling-image-3.0` | `default` | - | 0.200 | 0.200 | 0.400 | Tencent table: Kling 3.0 |
| `kling-image-3.0-omni` | `default` | - | 0.200 | 0.200 | 0.400 | Tencent table: Kling 3.0-omni |
| `kling-o1-image` | `default` | - | 0.200 | 0.200 | 0.400 | Tencent table: Kling O1 |
| `kling-2.1-image` | `text_to_image` | - | 0.100 | 0.100 | 0.260 | Tencent table: Kling 2.1 text image |
| `kling-2.1-image` | `multi_reference` | - | 0.400 | 0.480 | 0.560 | Tencent table: Kling 2.1 multi-reference |

### Image Models Missing Public Pricing Rows

These are present in the current adapter/model matrix but were not found as
public rows in the Tencent pricing page snapshot. Exact billing requires a
Tencent quote, a business contract price, or disabling the model until pricing is
confirmed.

| Public model | Current Tencent mapping | Recommended handling |
| --- | --- | --- |
| `si-4.0-image` | `SI` / `4.0` | Require admin price table entry before enabling exact billing |
| `si-4.5-image` | `SI` / `4.5` | Require admin price table entry before enabling exact billing |
| `si-5.0-lite-image` | `SI` / `5.0-lite` | Require admin price table entry before enabling exact billing |
| `kling-scene-image` | `Kling` / `scene` | Require admin price table entry; mode likely scene-specific |
| `jimeng-4.0-image` | `Jimeng` / `4.0` | Require admin price table entry |

### Fixed Task Prices

| Public model / capability | Unit price | Unit | Notes |
| --- | ---: | --- | --- |
| `hunyuan-3d-panorama` | 5.000 | task | HY World 2 Panorama |
| `hunyuan-3d-scene` | 200.000 | task | HY World 2 Scene |
| Vidu subject recognition | 0.470 | task | Extra capability, not a public generation model |
| Vidu audio direct output | 0.469 | task | Extra capability when enabled |
| Kling custom voice | 0.050 | task | Extra capability when enabled |
| Kling prompt sound effect | 0.250 | task | Extra capability when enabled |
| Kling video sound effect | 0.250 | task | Extra capability when enabled |

### Video Models With Public Pricing

| Public model | Price key mode | 480/540P | 720/768P | 1080P | 2K | 4K | Notes |
| --- | --- | ---: | ---: | ---: | ---: | ---: | --- |
| `hunyuan-1.5` | `default` | - | 0.300 | 0.500 | 0.750 | 1.120 | Tencent table: Hunyuan 1.5 |
| `vidu-q3-mix` | `reference` | - | 0.782 | 0.938 | 1.126 | 1.352 | Reference generation |
| `vidu-q3` | `reference` | 0.313 | 0.625 | 0.782 | 0.939 | 1.127 | Reference generation |
| `vidu-q3` | `reference_off_peak` | 0.157 | 0.313 | 0.391 | 0.470 | 0.564 | Off-peak |
| `vidu-q3-pro` | `default` | 0.313 | 0.782 | 0.938 | 1.126 | 1.351 | Image/text/first-last frame |
| `vidu-q3-pro` | `default_off_peak` | 0.157 | 0.391 | 0.469 | 0.563 | 0.676 | Off-peak |
| `vidu-q3-turbo` | `default` | 0.250 | 0.375 | 0.438 | 0.526 | 0.631 | Image/text/first-last frame |
| `vidu-q3-turbo` | `default_off_peak` | 0.125 | 0.188 | 0.219 | 0.263 | 0.316 | Off-peak |
| `vidu-q2` | `text` | - | 0.320 | 0.470 | 0.700 | 1.050 | Text generation |
| `vidu-q2` | `text_off_peak` | - | 0.160 | 0.235 | 0.350 | 0.525 | Off-peak |
| `vidu-q2` | `reference` | 0.240 | 0.320 | 0.820 | 1.230 | 1.845 | Reference generation |
| `vidu-q2` | `reference_off_peak` | 0.120 | 0.160 | 0.410 | 0.615 | 0.923 | Off-peak |
| `vidu-q2-pro` | `i2v_first_last` | - | 0.350 | 0.700 | 1.000 | 1.500 | Image/first-last frame |
| `vidu-q2-pro` | `i2v_first_last_off_peak` | - | 0.175 | 0.350 | 0.500 | 0.750 | Off-peak |
| `vidu-q2-pro` | `reference` | 0.270 | 0.350 | 0.900 | 1.350 | 2.025 | Reference generation |
| `vidu-q2-pro` | `reference_off_peak` | 0.135 | 0.175 | 0.450 | 0.675 | 1.013 | Off-peak |
| `vidu-q2-turbo` | `i2v_first_last` | - | 0.250 | 0.470 | 0.700 | 1.050 | Image/first-last frame |
| `vidu-q2-turbo` | `i2v_first_last_off_peak` | - | 0.125 | 0.235 | 0.350 | 0.525 | Off-peak |
| `kling-3.0-omni` | `no_reference_no_audio` | - | 0.600 | 0.800 | 1.000 | 3.000 | No reference, no audio |
| `kling-3.0-omni` | `no_reference_audio` | - | 0.800 | 1.000 | 1.200 | 3.000 | No reference, audio |
| `kling-3.0-omni` | `reference_no_audio` | - | 0.900 | 1.200 | 1.500 | 2.000 | Reference, no audio |
| `kling-3.0-omni` | `reference_audio` | - | 1.100 | 1.400 | 1.800 | 2.400 | Reference, audio |
| `kling-3.0` | `silent` | - | 0.600 | 0.800 | 1.000 | 3.000 | No audio |
| `kling-3.0` | `audio_no_voice` | - | 0.900 | 1.200 | 1.500 | 3.000 | Audio, no custom voice |
| `kling-3.0` | `audio_voice` | - | 1.100 | 1.400 | 1.800 | 2.400 | Audio with voice |
| `kling-3.0` | `motion_control` | - | 0.900 | 1.200 | 1.800 | 2.700 | Tencent table labels this as `3.0-montion-control` |
| `kling-2.6` | `silent` | - | 0.300 | 0.500 | 0.750 | 1.120 | No audio |
| `kling-2.6` | `audio` | - | - | 1.000 | 1.500 | 2.250 | Audio lane lacks 720P public row |
| `kling-2.6-motion-control` | `motion_control` | - | 0.500 | 0.800 | 1.200 | 1.800 | Tencent table labels this as `2.6-montion-control` |
| `kling-2.5-pro` | `default` | - | 0.300 | 0.500 | 0.750 | 1.120 | Tencent table |
| `kling-1.6` | `default` | - | 0.400 | 0.700 | 1.000 | 1.500 | Shared row with 2.0/2.1 |
| `kling-2.0` | `default` | - | 0.400 | 0.700 | 1.000 | 1.500 | Shared row with 1.6/2.1 |
| `kling-2.1` | `default` | - | 0.400 | 0.700 | 1.000 | 1.500 | Shared row with 1.6/2.0 |
| `kling-avatar` | `default` | - | 0.400 | 0.800 | 1.200 | 1.800 | Tencent spelling is `avater` |
| `kling-identifyface` | `default` | - | 0.100 | - | - | - | Every started 5 seconds bills as 5 seconds |
| `h2-1.0` | `default` | - | 0.900 | 1.600 | 1.920 | 2.304 | Tencent table: H2 1.0 |
| `hailuo-02` | `default` | - | 0.330 | 0.580 | 0.930 | 1.490 | Shared row with Hailuo 2.3 |
| `hailuo-2.3` | `default` | - | 0.330 | 0.580 | 0.930 | 1.490 | Shared row with Hailuo 02 |
| `hailuo-2.3-fast` | `default` | - | 0.225 | 0.385 | 0.580 | 0.870 | Tencent table |
| `gv-3.1` | `audio` | - | 3.000 | 3.000 | 3.750 | 4.500 | GV with audio |
| `gv-3.1` | `silent` | - | 1.500 | 1.500 | 2.250 | 3.000 | GV no audio |
| `gv-3.1-fast` | `audio` | - | 1.125 | 1.125 | 1.875 | 2.625 | GV fast with audio |
| `gv-3.1-fast` | `silent` | - | 0.750 | 0.750 | 1.500 | 2.250 | GV fast no audio |
| `gv-3.1-lite` | `audio` | - | 0.375 | 0.600 | 0.900 | 1.125 | GV lite with audio |
| `gv-3.1-lite` | `silent` | - | 0.225 | 0.375 | 0.600 | 0.750 | GV lite no audio |
| `os-2.0` | `default` | - | 0.750 | 1.125 | 1.688 | 2.531 | Tencent table: OS 2.0 |
| `pixverse-v5.6` | `silent` | 0.245 | 0.315 | 0.525 | 0.735 | 1.029 | Tencent table: V5.6 |
| `pixverse-v6` | `silent` | 0.205 | 0.264 | 0.528 | 0.634 | 0.760 | Tencent table: V6.0 silent |
| `pixverse-v6` | `audio` | 0.264 | 0.352 | 0.675 | 0.810 | 0.971 | Tencent table: V6.0 audio |
| `pixverse-c1` | `silent` | 0.235 | 0.293 | 0.557 | 0.669 | 0.803 | Tencent table: C1 silent |
| `pixverse-c1` | `audio` | 0.293 | 0.381 | 0.704 | 0.845 | 1.014 | Tencent table: C1 audio |
| `mingmou-1.0` | `default` | - | 0.300 | 0.500 | 0.750 | 1.120 | Tencent table |

### Video Models Missing Public Pricing Rows

| Public model | Current Tencent mapping | Recommended handling |
| --- | --- | --- |
| `jimeng-3.0-pro` | `Jimeng` / `3.0pro` | Require admin price table entry before enabling exact billing |
| `jimeng-4.0` | `Jimeng` / `4.0` | Require admin price table entry before enabling exact billing |
| `sv-1.0-pro` | `SV` / `1.0-pro` | Require admin price table entry before enabling exact billing |
| `sv-1.0-lite-i2v` | `SV` / `1.0-lite-i2v` | Require admin price table entry before enabling exact billing |

## Pricing Rule Resolution

The adapter must normalize request parameters into a deterministic price row.

### Resolution Normalization

Use Tencent's billing tier definitions:

- Images except Vidu: `512P` if short side <= 512, `1K` if <= 1024,
  `2K` if <= 2048, otherwise `4K`.
- Vidu images: `1K` if short side <= 1080, `2K` if <= 2048, otherwise `4K`.
- Videos: use output short-side billing tiers:
  `480P`, `540P`, `720P`, `768P`, `1080P`, `2K`, `4K`.
- If request sends an unsupported exact value such as `721P`, use request
  default for pre-consume and actual output tier for settlement if Tencent
  returns dimensions.

### Mode Normalization

Mode should be computed in one place from request fields:

| Model family | Normalization rule |
| --- | --- |
| Vidu image | no image input -> `text_to_image`; 1-3 refs -> `reference_1_3`; 4-7 refs -> `reference_4_7` |
| Vidu q2 video | text-only -> `text`; reference inputs -> `reference`; apply `_off_peak` when requested |
| Vidu q2-pro/q2-turbo | image/first-last -> `i2v_first_last`; reference -> `reference`; apply `_off_peak` |
| Vidu q3-pro/q3-turbo | text/image/first-last -> `default`; apply `_off_peak` |
| Vidu q3/q3-mix | reference generation -> `reference`; apply `_off_peak` where available |
| Kling 3.0 Omni | reference present + audio flag -> one of four reference/audio modes |
| Kling 3.0 | audio disabled -> `silent`; audio enabled with voice -> `audio_voice`; otherwise `audio_no_voice`; motion control overrides to `motion_control` |
| Kling 2.6 | audio disabled -> `silent`; audio enabled -> `audio`; motion control overrides to `motion_control` |
| GV | audio flag -> `audio`; otherwise `silent` |
| PixVerse | audio flag -> `audio`; otherwise `silent` |
| Other public rows | `default` |

### Duration Rules

| Model | Rule |
| --- | --- |
| `kling-identifyface` | `bill_sec = ceil(duration / 5) * 5`, minimum 5 |
| Standard video models | `bill_sec = actual_duration` if known, else requested/default duration |
| Fixed task models | duration ignored |

## Admin UI Design

Add a Tencent VOD pricing editor to system settings or channel settings.

Required capabilities:

1. Import official snapshot JSON.
2. View/edit price rows by model, mode, resolution, unit, price, currency.
3. Mark rows as "official", "contract", or "manual".
4. Per-row enable/disable flag.
5. Expression editor with preset templates.
6. Calculator:
   - model;
   - mode inputs;
   - resolution;
   - duration;
   - output count;
   - result price/quota preview.
7. Publish creates a new pricing version.
8. New tasks store:
   - expression version;
   - price table version;
   - resolved price key;
   - resolved unit price;
   - pre-consume variables.

## Backend Implementation Plan

1. Add task billing environment support to `pkg/billingexpr`.
2. Add a persisted pricing table:
   - model;
   - kind;
   - mode;
   - resolution;
   - unit;
   - unit_price;
   - currency;
   - source;
   - version;
   - enabled.
3. Add resolver in `relay/channel/task/tencentvod`:
   - normalize request facts;
   - build price key;
   - compute `bill_sec` / `image_count`;
   - run expression against price table.
4. Keep fallback support for existing `OtherRatios` only during migration.
5. Add settlement hook to recompute from upstream actual output facts.
6. Store billing snapshot into task private data/log `other`.
7. Add tests:
   - Vidu q3-turbo all resolutions;
   - Hunyuan 3.0 image all resolutions;
   - Kling 3.0 audio/no-audio lanes;
   - GV audio/no-audio lanes;
   - PixVerse audio/no-audio lanes;
   - Kling Identifyface 5-second rounding;
   - missing price row rejects channel/model enablement or request with explicit error;
   - settlement uses frozen snapshot after admin price edit.

## Migration Strategy

1. Keep existing backend model price as the operator selling price fallback.
2. Add Tencent VOD pricing table and expression mode behind a feature flag.
3. For models with official rows, import the snapshot above.
4. For missing rows, require manual admin entry before exact billing mode can be
   enabled.
5. Once task expression billing is stable, remove generic Tencent VOD
   resolution multipliers for exact-priced models.

## Recommended Default Policy

For Tencent VOD AIGC, do not use a generic multiplier as the long-term billing
source. Use:

```text
official_or_contract_unit_price(model, mode, resolution) * actual_usage
```

Then apply the platform's sale margin/group ratio outside the provider-price
snapshot, the same way text model pricing separates provider price from user
quota conversion.

