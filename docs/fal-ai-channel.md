# Fal.ai 渠道使用说明

## 接口地址

| 操作 | 方法 | 路径 |
|---|---|---|
| 提交任务 | `POST` | `/v1/images/generations/async` |
| 查询结果 | `GET` | `/v1/images/generations/{task_id}` |

鉴权方式：`Authorization: Bearer {token}`

---

## 模型

| 模型名 | 说明 |
|---|---|
| `openai/gpt-image-2` | 文生图 (t2i) |
| `openai/gpt-image-2/edit` | 图生图 (i2i) |

---

## 请求参数

### 文生图 (t2i)

```json
{
  "model": "openai/gpt-image-2",
  "prompt": "描述词",
  "size": "1024x1024",
  "quality": "medium"
}
```

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|
| `model` | string | 是 | — | 固定 `openai/gpt-image-2` |
| `prompt` | string | 是 | — | 英文描述词 |
| `size` | string | 否 | `1024x1024` | `WxH` 格式，见下方支持尺寸 |
| `quality` | string | 否 | `medium` | `low` / `medium` / `high` |

### 图生图 (i2i)

在一张或多张参考图基础上生成新图片。额外参数：

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `images` | string[] | 是 | 参考图公网 URL 列表，单个元素为一张图的完整 URL |

示例：

```json
{
  "model": "openai/gpt-image-2/edit",
  "prompt": "make it a watercolor painting",
  "size": "1024x768",
  "quality": "low",
  "images": ["https://example.com/photo1.jpg", "https://example.com/photo2.jpg"]
}
```

---

## 支持尺寸

| 尺寸 | 像素 |
|---|---|
| `1024x768` | 1024 × 768 |
| `1024x1024` | 1024 × 1024 |
| `1024x1536` | 1024 × 1536 |
| `1920x1080` | 1920 × 1080 |
| `2560x1440` | 2560 × 1440 |
| `3840x2160` | 3840 × 2160 |

---

## 响应

### 提交成功

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "object": "video",
  "model": "openai/gpt-image-2",
  "status": "queued",
  "progress": 0,
  "created_at": 1777990453
}
```

### 查询结果

轮询 `GET /v1/images/generations/{task_id}`，成功后返回：

```json
{
  "created": 1777990453,
  "data": [
    {
      "url": "https://v3b.fal.media/files/..."
    }
  ]
}
```

---

## 计费说明

### 定价模型

- **基价**：¥0.042（对应 fal.ai 1024×1024 低品质价格）
- **计算公式**：`扣费 = 基价 × 尺寸品质乘数`
- **计价方式**：按次计费，提交任务时预扣，任务完成后结算
- **计价货币**：人民币

### 乘数基准

所有乘数以 `1024×1024 low t2i` 为 1.0000×。不同尺寸和品质的乘数反映 fal.ai 的实际定价比例。

---

## 文生图 (t2i) 价格表

| 尺寸 | 低品质 (low) | 中等品质 (medium) | 高品质 (high) |
|---|---|---|---|
| 1024×768 | 0.8333× = **¥0.04** | 6.1667× = **¥0.26** | 24.1667× = **¥1.02** |
| 1024×1024 | 1.0000× = **¥0.04** | 8.8333× = **¥0.37** | 35.1667× = **¥1.48** |
| 1024×1536 | 0.8333× = **¥0.04** | 7.0000× = **¥0.29** | 27.5000× = **¥1.16** |
| 1920×1080 | 0.8333× = **¥0.04** | 6.6667× = **¥0.28** | 26.3333× = **¥1.11** |
| 2560×1440 | 1.1667× = **¥0.05** | 9.3333× = **¥0.39** | 37.0000× = **¥1.55** |
| 3840×2160 | 2.0000× = **¥0.08** | 16.8333× = **¥0.71** | 66.8333× = **¥2.81** |

---

## 图生图 (i2i) 价格表

| 尺寸 | 低品质 (low) | 中等品质 (medium) | 高品质 (high) |
|---|---|---|---|
| 1024×768 | 1.8333× = **¥0.08** | 7.1667× = **¥0.30** | 25.1667× = **¥1.06** |
| 1024×1024 | 2.5000× = **¥0.11** | 10.1667× = **¥0.43** | 36.5000× = **¥1.53** |
| 1024×1536 | 3.0000× = **¥0.13** | 9.0000× = **¥0.38** | 29.6667× = **¥1.25** |
| 1920×1080 | 2.8333× = **¥0.12** | 8.8333× = **¥0.37** | 26.3333× = **¥1.11** |
| 2560×1440 | 3.1667× = **¥0.13** | 11.3333× = **¥0.48** | 39.0000× = **¥1.64** |
| 3840×2160 | 4.0000× = **¥0.17** | 18.8333× = **¥0.79** | 68.8333× = **¥2.89** |

---

## 画幅矩阵

|比例 | 清晰度|  尺寸 |
|---|---|---|
| 1:1  | 1K | 1024×1024 |
|      | 2K | 2048x2048 |
|      | 4K | 2880x2880 |
| 4:3  | 1K | 1024×768 |
|      | 2K | 2048x1536 |
|      | 4K | 2880x2160 |
| 3:4  | 1K | 768x1024 |
|      | 2K | 1536x2048 |
|      | 4K | 2160x2880 |
| 2:3  | 1K | 1024×1536 |
|      | 2K | 1920x2880 |
|      | 4K | 2048x3840 |
| 3:2  | 1K | 1536x1024 |
|      | 2K | 2880x1920 |
|      | 4K | 3840x2048 |
| 16:9 | 1K | 1920×1080 |
|      | 2K | 2560×1440 |
|      | 4K | 3840×2160 |
| 9:16 | 1K | 1080x1920 |
|      | 2K | 1440x2560 |
|      | 4K | 2160x3840 |


## 调用示例

### curl - t2i 1024×1024 中等品质

```bash
curl -X POST https://your-api-host/v1/images/generations/async \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "openai/gpt-image-2",
    "prompt": "a red apple on white background",
    "size": "1024x1024",
    "quality": "medium"
  }'
```

### curl - i2i 1920×1080 低品质

```bash
curl -X POST https://your-api-host/v1/images/generations/async \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "openai/gpt-image-2/edit",
    "prompt": "make it look like an oil painting",
    "size": "1920x1080",
    "quality": "low",
    "images": ["https://example.com/input.jpg"]
  }'
```

### curl - 查询结果

```bash
curl https://your-api-host/v1/images/generations/task_xxx \
  -H "Authorization: Bearer sk-xxx"
```
