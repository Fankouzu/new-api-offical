# PingXingShiJie (channel 58) — OpenAI compatibility notes

This document describes how the gateway exposes PingXingShiJie async APIs next to OpenAI-style routes, and where behavior differs from official OpenAI APIs.

**Downstream API reference (parameters, responses, curl):** [pingxingshijie-api-reference.md](./pingxingshijie-api-reference.md).

## Shared conventions

- **Authentication**: Same as other channels — `Authorization: Bearer <token>` on gateway routes that use token auth.
- **Base URL (upstream)**: Default `https://api.pingxingshijie.cn` (overridable per channel). Upstream responses use a unified envelope: `{"code":0,"msg":"...","data":{...}}` with HTTP 200; business errors use non-zero `code`.

## Routes aligned with existing gateway / OpenAI-style usage

| Capability | Gateway routes | Upstream (PingXingShiJie) |
|------------|----------------|---------------------------|
| Text chat | `POST /v1/chat/completions` (same global relay router as other OpenAI-style channels) | `POST /api/v3/chat/completions` (Volcengine Ark path; same HTTP contract as channel **VolcEngine**) |
| Video async | `POST /v1/video/generations`, `POST /v1/videos`, `GET /v1/video/generations/:task_id`, `GET /v1/videos/:task_id` | `POST /v2/video/generations`, `GET /v2/video/generations/tasks/{id}` |
| Image async | `POST /v1/images/generations/async`, `GET /v1/images/generations/:task_id` | `POST /v2/image/generations`, `GET /v2/image/generations/tasks/{id}` |
| Asset async | `POST /v1/assets/upload`, `GET /v1/assets/:task_id` | `POST /v2/asset/upload`, `POST /v2/asset/status` (polled server-side) |

## Differences from OpenAI

- **Chat**: The gateway still exposes **`/v1/chat/completions`** to clients. For channel 58 the relay uses the **Volcengine Ark** adaptor, so the upstream HTTP path is **`/api/v3/chat/completions`**, not OpenAI’s literal `/v1/...` on the provider host. If the provider returns the PingXing envelope (`code` / `msg` / `data`), the gateway unwraps **`data`** for **non-streaming** responses before emitting an OpenAI-shaped JSON body.
- **Official `POST /v1/images/generations` (OpenAI)**: Synchronous image URL in the response body. **This gateway’s** `POST /v1/images/generations/async` is **async**: it returns a public `task_id` and requires **`GET /v1/images/generations/:task_id`** (or the unified task APIs) to poll until completion. Clients must not assume OpenAI’s synchronous semantics on the async route.
- **Video**: `generate_audio` is sent upstream; when omitted in the mapped request, it defaults to **true** (per provider contract).
- **Assets**: There is **no** OpenAI-standard equivalent. `POST /v1/assets/upload` / `GET /v1/assets/:task_id` are **gateway extensions** for PingXingShiJie asset upload and status surfaced as a single task record.

## Task storage

- Tasks store `private_data.upstream_kind` as `video` | `image` | `asset` so polling hits the correct upstream endpoint (including **POST** `/v2/asset/status` for assets).
