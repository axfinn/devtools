# AI Gateway API

统一对外开放项目内已有 AI 能力：

- 文本模型
  - `deepseek-chat` 或配置中的 `deepseek.model`
  - `abab6.5s-chat` / `MiniMax-M2.5` 或配置中的 `minimax.model`
- 图片 / 视频模型
  - `qwen-image-2.0-pro`
  - `qwen-image-2.0`
  - `qwen-image-2.0-pro-2026-03-03`
  - `qwen-image-2.0-2026-03-03`
  - `qwen-image-plus-2026-01-09`
  - `wan2.6-i2v-flash`
  - 以及 `backend/config.yaml` 中启用的其他 Bailian 模型

## 1. 超级管理员能力

超级管理员通过 `ai_gateway.super_admin_password` 或环境变量 `AI_GATEWAY_SUPER_ADMIN_PASSWORD` 控制。

鉴权方式：

- Header: `X-Super-Admin-Password: your-password`
- Query: `?super_admin_password=your-password`
- Body: `{ "super_admin_password": "your-password" }`

### 创建 API Key

`POST /api/ai-gateway/admin/keys`

```json
{
  "super_admin_password": "your-password",
  "name": "marketing-service",
  "allowed_models": ["deepseek-chat", "qwen-image-2.0-pro"],
  "allowed_scopes": ["chat", "media"],
  "expires_days": 90,
  "rate_limit_per_hour": 500,
  "budget_limit": 300,
  "alert_threshold": 0.8,
  "notes": "市场部业务"
}
```

返回：

- `plain_key`
- `key.id`
- `key.key_prefix`
- `key.allowed_models`
- `key.rate_limit_per_hour`

注意：
- `plain_key` 只返回一次
- 如果 `allowed_models` 留空，默认允许 `["*"]`
- 如果 `allowed_scopes` 留空，默认允许 `["chat","media"]`

### 查看 Key 列表

`GET /api/ai-gateway/admin/keys`

### 查看 Key 详情

`GET /api/ai-gateway/admin/keys/:id`

返回：
- Key 基本信息
- 最近 50 条请求明细

### 吊销 Key

`POST /api/ai-gateway/admin/keys/:id/revoke`

```json
{
  "super_admin_password": "your-password"
}
```

### 查看请求明细

`GET /api/ai-gateway/admin/logs?api_key_id=xxx&limit=100`

### 查看按天/月账单报表

`GET /api/ai-gateway/admin/reports?group_by=day&days=30`

### 查看预算阈值告警

`GET /api/ai-gateway/admin/alerts`

## 2. 模型目录

`GET /api/ai-gateway/catalog`

返回每个模型的：
- `model`
- `provider`
- `type`
- `endpoint`
- `description`

## 3. 业务方鉴权

业务方使用 API Key 访问。

推荐 Header：

```http
Authorization: Bearer dtk_ai_xxxxxxxxxxxxx
```

也支持：

```http
X-API-Key: dtk_ai_xxxxxxxxxxxxx
```

## 4. 文本模型接口

`POST /api/ai-gateway/v1/chat/completions`

请求体：

```json
{
  "model": "deepseek-chat",
  "messages": [
    { "role": "system", "content": "你是一个专业的后端助手" },
    { "role": "user", "content": "写一个 Gin 健康检查接口示例" }
  ],
  "temperature": 0.7,
  "max_tokens": 1024
}
```

返回是统一格式，接近 OpenAI Chat Completions：

```json
{
  "id": "chatcmpl-xxxx",
  "object": "chat.completion",
  "created": 1760000000,
  "model": "deepseek-chat",
  "provider": "deepseek",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "..."
      },
      "finish_reason": "stop"
    }
  ],
  "content": "...",
  "usage": {},
  "raw_response": {}
}
```

说明：
- `DeepSeek` 直接透传 `choices`
- `MiniMax` 会被适配成统一 `choices`

## 5. 图片 / 视频模型接口

`POST /api/ai-gateway/v1/media/generations`

请求体：

```json
{
  "model": "qwen-image-2.0-pro",
  "prompt": "一只橘猫穿宇航服，电影海报风格",
  "negative_prompt": "模糊、低清晰度、变形手",
  "images": ["data:image/png;base64,..."],
  "size": "1328x1328",
  "count": 1,
  "auto_poll": true,
  "wait_seconds": 30,
  "client_name": "marketing-site",
  "client_request_id": "order-20260307-1",
  "parameters": {
    "watermark": false
  }
}
```

返回：
- `task`
- `events`
- `assets`
- `completed`
- `waited`

### 查看当前 API Key 的媒体任务列表

`GET /api/ai-gateway/v1/media/tasks?model=qwen-image-2.0-pro&status=succeeded&limit=20&offset=0`

### 查看当前 API Key 的媒体任务详情

`GET /api/ai-gateway/v1/media/tasks/:id`

## 6. 调用限制

AI Gateway 会做以下限制：

- API Key 状态校验
- API Key 过期校验
- Key 级别模型白名单校验
- Key 级别 Scope 校验
- Key 级别每小时请求数限制
- Bailian 模型本地额度限制和截止时间限制

## 7. Token 统计与计费

网关会对每次调用记录：

- `input_tokens`
- `output_tokens`
- `total_tokens`
- `estimated_cost`
- `currency`

统计规则：

- 文本模型优先使用上游返回的 `usage`
- 如果上游没返回 `usage`，后端会做本地估算
- 图片 / 视频模型按 `request_cost` 记一次费用

预算与告警：

- `budget_limit` 是单个 API Key 的预算上限
- `alert_threshold` 是告警阈值，默认 `0.8`
- 当 `total_cost / budget_limit >= alert_threshold` 时，会出现在告警列表
- 当 `total_cost / budget_limit >= 1` 时，告警等级为 `critical`

本地估算说明：

- 文本按字符数粗略折算 token
- 估算值用于内部计费和报表，不保证与供应商账单 100% 一致
- 如果你要尽量贴近供应商账单，建议把单价和对账规则配成你自己的内部标准

计费配置示例：

```yaml
ai_gateway:
  pricing:
    - model: "deepseek-chat"
      provider: "deepseek"
      input_per_1k_tokens: 0.002
      output_per_1k_tokens: 0.008
      request_cost: 0
      currency: "CNY"
    - model: "qwen-image-2.0-pro"
      provider: "bailian"
      input_per_1k_tokens: 0
      output_per_1k_tokens: 0
      request_cost: 0.12
      currency: "CNY"
```

## 8. 配置

```yaml
ai_gateway:
  super_admin_password: ""
  default_key_expires_days: 90
  default_rate_limit_per_hour: 1000
  request_retention_days: 180
  pricing:
    - model: "deepseek-chat"
      provider: "deepseek"
      input_per_1k_tokens: 0
      output_per_1k_tokens: 0
      request_cost: 0
      currency: "CNY"
```

建议在创建 Key 时给关键业务填预算：

```json
{
  "name": "billing-service",
  "budget_limit": 500,
  "alert_threshold": 0.8
}
```

完整配置示例：

```yaml
ai_gateway:
  super_admin_password: ""
  default_key_expires_days: 90
  default_rate_limit_per_hour: 1000
  request_retention_days: 180
  pricing:
    - model: "deepseek-chat"
      provider: "deepseek"
      input_per_1k_tokens: 0
      output_per_1k_tokens: 0
      request_cost: 0
      currency: "CNY"
```

## 9. 接入示例

### cURL 文本接口

```bash
curl -X POST http://localhost:8080/api/ai-gateway/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dtk_ai_xxx" \
  -d '{
    "model": "deepseek-chat",
    "messages": [
      {"role": "system", "content": "你是一个专业助手"},
      {"role": "user", "content": "写一个 Go HTTP Server 示例"}
    ],
    "temperature": 0.7
  }'
```

### cURL 图片接口

```bash
curl -X POST http://localhost:8080/api/ai-gateway/v1/media/generations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dtk_ai_xxx" \
  -d '{
    "model": "qwen-image-2.0-pro",
    "prompt": "一只橘猫穿宇航服，电影海报风格",
    "size": "1328x1328",
    "auto_poll": true,
    "wait_seconds": 30
  }'
```

### JavaScript 示例

```js
const apiKey = process.env.AI_GATEWAY_KEY;

async function chat() {
  const res = await fetch('http://localhost:8080/api/ai-gateway/v1/chat/completions', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${apiKey}`,
    },
    body: JSON.stringify({
      model: 'deepseek-chat',
      messages: [
        { role: 'system', content: '你是一个专业助手' },
        { role: 'user', content: '给我一个 Gin 中间件示例' }
      ],
      temperature: 0.5
    }),
  });

  const data = await res.json();
  console.log(data.content);
}

async function generateImage() {
  const res = await fetch('http://localhost:8080/api/ai-gateway/v1/media/generations', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${apiKey}`,
    },
    body: JSON.stringify({
      model: 'qwen-image-2.0-pro',
      prompt: '极简白色产品摄影，智能手表，广告海报',
      size: '1328x1328',
      auto_poll: true,
      wait_seconds: 30
    }),
  });

  const data = await res.json();
  console.log(data.assets);
}
```

### Python 示例

```python
import requests

API_KEY = "dtk_ai_xxx"
BASE_URL = "http://localhost:8080/api/ai-gateway"

def chat():
    resp = requests.post(
        f"{BASE_URL}/v1/chat/completions",
        headers={"Authorization": f"Bearer {API_KEY}"},
        json={
            "model": "deepseek-chat",
            "messages": [
                {"role": "system", "content": "你是一个专业助手"},
                {"role": "user", "content": "请写一个 Python FastAPI 示例"}
            ],
            "temperature": 0.6
        },
        timeout=60,
    )
    print(resp.json()["content"])

def generate():
    resp = requests.post(
        f"{BASE_URL}/v1/media/generations",
        headers={"Authorization": f"Bearer {API_KEY}"},
        json={
            "model": "qwen-image-2.0-pro",
            "prompt": "一只金毛坐在咖啡馆里，温暖晨光",
            "size": "1328x1328",
            "auto_poll": True,
            "wait_seconds": 30
        },
        timeout=120,
    )
    print(resp.json())
```

### Go 示例

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func main() {
	apiKey := "dtk_ai_xxx"

	body := map[string]interface{}{
		"model": "deepseek-chat",
		"messages": []map[string]string{
			{"role": "system", "content": "你是一个专业助手"},
			{"role": "user", "content": "给我一个 Go Gin 路由示例"},
		},
		"temperature": 0.7,
	}

	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "http://localhost:8080/api/ai-gateway/v1/chat/completions", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	fmt.Println(result["content"])
}
```
