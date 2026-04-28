# Token Management API Reference

> 认证方式：Session Cookie 或 Bearer Token（用户登录后的会话凭证，非 API Key）。

---

## Overview

Token 管理接口位于 `/api/token/` 路径下，采用 **用户级认证** (`middleware.UserAuth()`)，即用户只能管理自己的 Token，管理员没有独立的管理其他用户 Token 的端点。

### 通用响应格式

所有接口统一返回 HTTP 200，通过 `success` 字段区分成功/失败：

```json
// 成功（含 data）
{
  "success": true,
  "message": "",
  "data": { ... }
}

// 成功（无 data）
{
  "success": true,
  "message": ""
}

// 失败
{
  "success": false,
  "message": "错误描述文本"
}
```

### Token 状态常量

| 值 | 常量 | 含义 |
|----|------|------|
| 1 | `TokenStatusEnabled` | 启用 |
| 2 | `TokenStatusDisabled` | 手动禁用 |
| 3 | `TokenStatusExpired` | 已过期 |
| 4 | `TokenStatusExhausted` | 额度耗尽 |

### Token 数据模型

| 字段 | 类型 | JSON key | 说明 |
|------|------|----------|------|
| Id | int | `id` | 主键 |
| UserId | int | `user_id` | 所属用户 ID |
| Key | string | `key` | 48 字符 API 密钥（自动生成，列表/详情中脱敏显示） |
| Status | int | `status` | 状态：1=启用, 2=禁用, 3=过期, 4=耗尽 |
| Name | string | `name` | 显示名称，最长 50 字符 |
| CreatedTime | int64 | `created_time` | 创建时间（Unix 秒） |
| AccessedTime | int64 | `accessed_time` | 最后访问时间（Unix 秒） |
| ExpiredTime | int64 | `expired_time` | 过期时间（Unix 秒），-1 表示永不过期 |
| RemainQuota | int | `remain_quota` | 剩余额度 |
| UnlimitedQuota | bool | `unlimited_quota` | 是否无限额度 |
| ModelLimitsEnabled | bool | `model_limits_enabled` | 是否启用模型限制 |
| ModelLimits | string | `model_limits` | 允许的模型列表（逗号分隔） |
| AllowIps | *string | `allow_ips` | IP 白名单（换行分隔），`null` 或空字符串表示不限 |
| UsedQuota | int | `used_quota` | 已用额度 |
| Group | string | `group` | 分组名称 |
| CrossGroupRetry | bool | `cross_group_retry` | 跨分组重试（仅 `auto` 组有效） |

### Key 脱敏规则

- 长度 ≤ 4：全部替换为 `*`
- 长度 ≤ 8：前2 + `****` + 后2
- 长度 > 8：前4 + `**********` + 后4

---

## 1. 获取所有令牌

```
GET /api/token/
```

**认证**：UserAuth

### 查询参数

| 参数 | 位置 | 必选 | 默认值 | 说明 |
|------|------|------|--------|------|
| `p` / `page` | query | 否 | 1 | 页码，最小 1 |
| `page_size` / `ps` / `size` | query | 否 | 10 (`ItemsPerPage`) | 每页条数，最大 100 |

### 成功响应

```json
{
  "success": true,
  "message": "",
  "data": {
    "page": 1,
    "page_size": 10,
    "total": 42,
    "items": [
      {
        "id": 1,
        "user_id": 1,
        "key": "sk-a**********xyz1",
        "status": 1,
        "name": "我的令牌",
        "created_time": 1700000000,
        "accessed_time": 1700100000,
        "expired_time": -1,
        "remain_quota": 500000,
        "unlimited_quota": false,
        "model_limits_enabled": false,
        "model_limits": "",
        "allow_ips": "",
        "used_quota": 100000,
        "group": "default",
        "cross_group_retry": false
      }
    ]
  }
}
```

### 错误响应

```json
{
  "success": false,
  "message": "数据库错误描述"
}
```

---

## 2. 搜索令牌

```
GET /api/token/search
```

**认证**：UserAuth + SearchRateLimit（每用户 10 次/分钟）

### 查询参数

| 参数 | 位置 | 必选 | 默认值 | 说明 |
|------|------|------|--------|------|
| `keyword` | query | 否 | `""` | 按名称搜索，支持 `%` 通配符（最多 2 个），不含 `%` 时精确匹配 |
| `token` | query | 否 | `""` | 按 Key 搜索（可带/不带 `sk-` 前缀），同样支持 `%` 通配符 |
| `p` / `page` | query | 否 | 1 | 页码 |
| `page_size` / `ps` / `size` | query | 否 | 10 | 每页条数，最大 100 |

### 搜索规则

- `keyword` 和 `token` 均为空时，等同于获取全部令牌
- 使用 `%` 通配符（模糊搜索）时，去掉 `%` 后的关键词长度必须 ≥ 2
- 不允许连续 `%%`
- 超量用户（令牌数超过系统上限）禁止模糊搜索，仅允许精确匹配
- 硬上限：最多返回 100 条结果

### 成功响应

格式同 [获取所有令牌](#1-获取所有令牌)，分页结构一致。

### 错误响应

```json
{
  "success": false,
  "message": "搜索模式中不允许包含连续的 % 通配符"
}
```

```json
{
  "success": false,
  "message": "使用模糊搜索时，关键词长度至少为 2 个字符"
}
```

```json
{
  "success": false,
  "message": "令牌数量超过上限，仅允许精确搜索，请勿使用 % 通配符"
}
```

```json
{
  "success": false,
  "message": "搜索令牌失败"
}
```

---

## 3. 获取单个令牌

```
GET /api/token/:id
```

**认证**：UserAuth

### 路径参数

| 参数 | 位置 | 必选 | 说明 |
|------|------|------|------|
| `id` | path | 是 | 令牌 ID |

### 成功响应

```json
{
  "success": true,
  "message": "",
  "data": {
    "id": 1,
    "user_id": 1,
    "key": "sk-a**********xyz1",
    "status": 1,
    "name": "我的令牌",
    "created_time": 1700000000,
    "accessed_time": 1700100000,
    "expired_time": -1,
    "remain_quota": 500000,
    "unlimited_quota": false,
    "model_limits_enabled": false,
    "model_limits": "",
    "allow_ips": "",
    "used_quota": 100000,
    "group": "default",
    "cross_group_retry": false
  }
}
```

### 错误响应

```json
{
  "success": false,
  "message": "id 或 userId 为空！"
}
```

```json
{
  "success": false,
  "message": "record not found"
}
```

---

## 4. 获取令牌完整 Key

```
POST /api/token/:id/key
```

**认证**：UserAuth + CriticalRateLimit + DisableCache

**频率限制**：20 次 / 20 分钟

### 路径参数

| 参数 | 位置 | 必选 | 说明 |
|------|------|------|------|
| `id` | path | 是 | 令牌 ID |

### 请求体

无

### 成功响应

```json
{
  "success": true,
  "message": "",
  "data": {
    "key": "sk-abcdef12345678901234567890123456789012345678"
  }
}
```

### 错误响应

同 [获取单个令牌](#3-获取单个令牌) 的错误格式。

---

## 5. 创建令牌

```
POST /api/token/
```

**认证**：UserAuth

### 请求体

Content-Type: `application/json`

| 字段 | 类型 | 必选 | 默认值 | 说明 |
|------|------|------|--------|------|
| `name` | string | 否 | `""` | 令牌名称，最长 50 字符 |
| `expired_time` | int64 | 否 | -1 | 过期时间（Unix 秒），-1 表示永不过期 |
| `remain_quota` | int | 否 | 0 | 初始剩余额度 |
| `unlimited_quota` | bool | 否 | false | 是否无限额度 |
| `model_limits_enabled` | bool | 否 | false | 是否启用模型限制 |
| `model_limits` | string | 否 | `""` | 允许的模型列表（逗号分隔，如 `gpt-4,claude-3`） |
| `allow_ips` | string/null | 否 | null | IP 白名单（换行分隔），null 或空字符串表示不限 |
| `group` | string | 否 | `""` | 分组名称 |
| `cross_group_retry` | bool | 否 | false | 跨分组重试（仅 `auto` 组有效） |

### 验证规则

1. `name` 长度不得超过 50 字符
2. 非无限额度时，`remain_quota` 必须 ≥ 0 且 ≤ `1000000000 * QuotaPerUnit`
3. 用户令牌数量不能超过系统上限（`operation_setting.GetMaxUserTokens()`）

### 请求示例

```json
{
  "name": "我的令牌",
  "expired_time": 1735689600,
  "remain_quota": 500000,
  "unlimited_quota": false,
  "model_limits_enabled": true,
  "model_limits": "gpt-4,claude-3-opus",
  "allow_ips": "192.168.1.0/24\n10.0.0.1",
  "group": "default",
  "cross_group_retry": false
}
```

### 成功响应

```json
{
  "success": true,
  "message": ""
}
```

> 创建成功后响应体中不包含令牌 Key。如需获取 Key，需调用 [获取令牌完整 Key](#4-获取令牌完整-key)。

### 错误响应

```json
{
  "success": false,
  "message": "令牌名称过长"
}
```

```json
{
  "success": false,
  "message": "额度不能为负数"
}
```

```json
{
  "success": false,
  "message": "额度超出最大值: <Max>"
}
```

```json
{
  "success": false,
  "message": "已达到最大令牌数量限制 (100)"
}
```

```json
{
  "success": false,
  "message": "令牌生成失败"
}
```

---

## 6. 更新令牌

```
PUT /api/token/
```

**认证**：UserAuth

### 查询参数

| 参数 | 位置 | 必选 | 默认值 | 说明 |
|------|------|------|--------|------|
| `status_only` | query | 否 | `""`（空） | 非空时仅更新 `status` 字段，忽略其他字段 |

### 请求体

Content-Type: `application/json`

| 字段 | 类型 | 必选（status_only 为空时） | 默认值 | 说明 |
|------|------|---------------------------|--------|------|
| `id` | int | **是** | — | 要更新的令牌 ID |
| `status` | int | `status_only` 模式下必选 | — | 新状态值 |
| `name` | string | 否 | — | 令牌名称，最长 50 字符 |
| `expired_time` | int64 | 否 | — | 过期时间 |
| `remain_quota` | int | 否 | — | 剩余额度 |
| `unlimited_quota` | bool | 否 | — | 是否无限额度 |
| `model_limits_enabled` | bool | 否 | — | 是否启用模型限制 |
| `model_limits` | string | 否 | — | 允许的模型列表 |
| `allow_ips` | string/null | 否 | — | IP 白名单 |
| `group` | string | 否 | — | 分组名称 |
| `cross_group_retry` | bool | 否 | — | 跨分组重试 |

### 状态更新限制

当尝试将 `status` 设为 1（启用）时，系统会检查原始令牌状态：
- 如果令牌已过期（状态 3）且过期时间已到，**拒绝启用**
- 如果令牌已耗尽（状态 4）且无剩余额度且非无限额度，**拒绝启用**

### 请求示例

```json
// 完整更新
{
  "id": 1,
  "name": "更新后的名称",
  "expired_time": -1,
  "remain_quota": 1000000,
  "unlimited_quota": false,
  "model_limits_enabled": false,
  "model_limits": "",
  "allow_ips": null,
  "group": "vip",
  "cross_group_retry": true
}
```

```json
// 仅更新状态
// PUT /api/token/?status_only=true
{
  "id": 1,
  "status": 2
}
```

### 成功响应

```json
{
  "success": true,
  "message": "",
  "data": {
    "id": 1,
    "key": "sk-a**********xyz1",
    "status": 2,
    "name": "更新后的名称",
    "created_time": 1700000000,
    "accessed_time": 1700100000,
    "expired_time": -1,
    "remain_quota": 1000000,
    "unlimited_quota": false,
    "model_limits_enabled": false,
    "model_limits": "",
    "allow_ips": null,
    "used_quota": 100000,
    "group": "vip",
    "cross_group_retry": true
  }
}
```

### 错误响应

```json
{
  "success": false,
  "message": "id 或 userId 为空！"
}
```

```json
{
  "success": false,
  "message": "record not found"
}
```

```json
{
  "success": false,
  "message": "令牌已过期，无法启用"
}
```

```json
{
  "success": false,
  "message": "令牌额度已耗尽，无法启用"
}
```

```json
{
  "success": false,
  "message": "令牌名称过长"
}
```

---

## 7. 删除令牌

```
DELETE /api/token/:id
```

**认证**：UserAuth

### 路径参数

| 参数 | 位置 | 必选 | 说明 |
|------|------|------|------|
| `id` | path | 是 | 令牌 ID |

### 请求体

无

### 成功响应

```json
{
  "success": true,
  "message": ""
}
```

> 删除为软删除（GORM `DeletedAt`），数据保留在数据库中。

### 错误响应

```json
{
  "success": false,
  "message": "record not found"
}
```

---

## 8. 批量删除令牌

```
POST /api/token/batch
```

**认证**：UserAuth

### 请求体

Content-Type: `application/json`

| 字段 | 类型 | 必选 | 说明 |
|------|------|------|------|
| `ids` | int[] | 是 | 要删除的令牌 ID 数组 |

### 请求示例

```json
{
  "ids": [1, 2, 3, 5]
}
```

### 成功响应

```json
{
  "success": true,
  "message": "",
  "data": 4
}
```

> `data` 为实际成功删除的数量（仅统计属于当前用户的令牌）。

### 错误响应

```json
{
  "success": false,
  "message": "参数无效"
}
```

> 当 `ids` 为空或 JSON 解析失败时返回。

---

## 9. 批量获取令牌 Key

```
POST /api/token/batch/keys
```

**认证**：UserAuth + CriticalRateLimit + DisableCache

**频率限制**：20 次 / 20 分钟

### 请求体

Content-Type: `application/json`

| 字段 | 类型 | 必选 | 说明 |
|------|------|------|------|
| `ids` | int[] | 是 | 令牌 ID 数组，最多 100 个 |

### 请求示例

```json
{
  "ids": [1, 2, 3]
}
```

### 成功响应

```json
{
  "success": true,
  "message": "",
  "data": {
    "keys": {
      "1": "sk-abcdef12345678901234567890123456789012345678",
      "2": "sk-ghijklmnopqrstuvwxzy1234567890abcdefghijklmnop",
      "3": "sk-qrstuvwxyz1234567890abcdefghijklmnopqrstuvwxyz"
    }
  }
}
```

> `keys` 的 key 为令牌 ID（字符串），value 为完整 API Key。

### 错误响应

```json
{
  "success": false,
  "message": "参数无效"
}
```

```json
{
  "success": false,
  "message": "批量操作数量过多，最大允许: 100"
}
```

---

## 10. 查询令牌用量

```
GET /api/usage/token/
```

**认证**：TokenAuthReadOnly（通过 Bearer Token 认证，只读模式，不检查额度/过期）

### 请求头

| 头 | 必选 | 说明 |
|----|------|------|
| `Authorization` | 是 | `Bearer sk-<api_key>` 格式 |

### 查询参数

无

### 成功响应

```json
{
  "code": true,
  "message": "ok",
  "data": {
    "object": "token_usage",
    "name": "我的令牌",
    "total_granted": 600000,
    "total_used": 100000,
    "total_available": 500000,
    "unlimited_quota": false,
    "model_limits": {
      "gpt-4": true,
      "claude-3": true
    },
    "model_limits_enabled": true,
    "expires_at": 1735689600
  }
}
```

> 注意：此接口响应格式与标准格式不同，使用 `code` 而非 `success`。
> `expires_at` 为 0 表示永不过期（原始值 -1 转换为 0）。

### 错误响应

```json
// HTTP 401
{
  "success": false,
  "message": "No Authorization header"
}
```

```json
// HTTP 401
{
  "success": false,
  "message": "Invalid Bearer token"
}
```

```json
// HTTP 200
{
  "success": false,
  "message": "获取令牌信息失败"
}
```

---

## 11. 查询令牌调用日志

```
GET /api/log/token
```

**认证**：TokenAuthReadOnly + CORS + CriticalRateLimit

### 查询参数

具体参数取决于 `controller.GetLogByKey` 实现，通常支持分页和时间范围过滤。

### 成功响应

返回该 Token 的调用日志列表（具体格式取决于日志控制器实现）。

---

## 附：未注册的端点

以下 Controller 函数已定义但 **未在任何 Router 中注册**，属于死代码：

- `GetTokenStatus` — 返回令牌额度摘要（OpenAI 兼容格式 `credit_summary`），未使用

---

## 源码参考

| 文件 | 说明 |
|------|------|
| `controller/token.go` | 所有 Token 控制器函数 |
| `model/token.go` | Token 数据模型、数据库操作、搜索/验证逻辑 |
| `model/token_cache.go` | Token Redis 缓存层 |
| `router/api-router.go:249-261` | Token 路由注册 |
| `common/gin.go` | `ApiSuccess` / `ApiError` / `ApiErrorI18n` 响应函数 |
| `common/page_info.go` | `PageInfo` 分页结构与 `GetPageQuery` 参数解析 |
| `common/constants.go` | Token 状态常量、频率限制配置 |
