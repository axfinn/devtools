# AI Gateway API

统一对外开放项目内已有 AI 能力：

- 文本模型
  - `deepseek-chat`
  - `deepseek-reasoner`
  - `deepseek-v4-flash`
  - `deepseek-v4-pro`
  - 或配置中的 `deepseek.model`
  - `abab6.5s-chat` / `MiniMax-M2.5` 或配置中的 `minimax.model`
- 图像理解模型
  - `minimax-mcp-understand-image`（`POST /api/ai-gateway/v1/image/understanding`，内部走 MCP 图像理解工具）
  - `MiniMax-M3`（`POST /api/image-understanding/qwen-vision`，内部免 API Key 端点，非 MiniMax 开头的 model 一律强制回落 `MiniMax-M3`）
- 视频生成模型
  - `MiniMax-H3` / `MiniMax-H3-Max`（`POST /api/minimax/h3/v2/video_generation`，异步 submit → task_id → 轮询）
- DashScope 文本模型（需配置 `DASHSCOPE_API_KEY`）
  - `qwen3.5-plus`、`qwen3-max-2026-01-23`、`qwen3-coder-next`、`qwen3-coder-plus`
  - `glm-5`、`glm-4.7`、`kimi-k2.5`
- 以及 `backend/config.yaml` 的 `ai_gateway.proxy` 中启用的其他代理模型

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
  "allowed_models": ["deepseek-chat", "qwen3.5-plus"],
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
  "model": "deepseek-v4-pro",
  "messages": [
    { "role": "system", "content": "你是一个专业的后端助手" },
    { "role": "user", "content": "写一个 Gin 健康检查接口示例" }
  ],
  "temperature": 0.7,
  "max_tokens": 1024,
  "reasoning_effort": "medium"
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
- `deepseek-reasoner` 额外返回 `reasoning_content`
- `MiniMax` 会被适配成统一 `choices`

## 5. 图像理解 / 视频生成接口

> ⚠️ **历史变更**：原图片 / 视频模型模块已于 2026-09-09 整块下线（commit `7d08dfd`），
> 原 `POST /api/ai-gateway/v1/media/generations`、`GET /api/ai-gateway/v1/media/tasks`、
> `GET /api/ai-gateway/v1/media/tasks/:id` 三个路由已随之下线，调用返回 404。
> 下文是当前真实存在的图像 / 视频能力。

### 5.1 图像理解

`POST /api/ai-gateway/v1/image/understanding`

鉴权：超级管理员密码或 API Key（scope 需含 `media`）。请求体：

```json
{
  "prompt": "描述这张图片",
  "image": "data:image/png;base64,...",
  "tool": "",
  "args": {}
}
```

- `image` 必填，单张 ≤ 10MB；本端点只收 **Base64**（`data:image/...;base64,...` 或裸 base64），
  传 HTTP(S) URL 会因 base64 解码失败而报 400；需要传远端图片请用下面的 `/file` 变体或 `qwen-vision`
- `tool` 可选，留空时自动挑选 MCP 图像理解工具（优先 `understand_image`）
- `prompt` 可选，留空时使用默认描述提示词

返回：
- `tool`
- `model`（`minimax-mcp-understand-image`）
- `text`
- `result`
- `args_preview`

其他变体：

- `POST /api/ai-gateway/v1/image/understanding/file` —— multipart 上传图片
- `POST /api/ai-gateway/v1/image/understanding/sse` —— SSE 流式
- `POST /api/ai-gateway/v1/image/understanding/sse/file` —— SSE + multipart
- `POST /api/image-understanding/qwen-vision` —— 内部免 API Key 端点，最多 10 张图；
  `model` 非 `MiniMax` 前缀时强制回落 `MiniMax-M3`（原 qwen / DashScope 视觉通道已停用）

### 5.2 视频生成（MiniMax-H3，异步）

`POST /api/minimax/h3/v2/video_generation`

鉴权：超级管理员密码或 API Key（scope 需含 `media`）。请求体：

```json
{
  "model": "MiniMax-H3",
  "prompt": "一个男孩在海边打篮球",
  "resolution": "2K",
  "duration": 5,
  "ratio": "16:9"
}
```

文生视频（`prompt`）；图生视频额外传 `first_frame_image`（HTTP(S) URL）。

接口**立即返回 `task_id`**，由后台 goroutine 轮询上游并落库（避免 Cloudflare 524 读超时），
客户端需自行轮询任务状态：

- `GET /api/minimax/h3/v2/query/video_generation` —— 列出最近 7 天任务
  （支持 `filter.status` / `filter.task_ids` / `filter.model` / `filter.task_type`）
- `GET /api/minimax/h3/v2/query/video_generation/:task_id` —— 查询单个任务，
  `succeeded` 时 `content.url` 为视频产物
- `DELETE /api/minimax/h3/v2/video_generation/:task_id` —— 取消 `queued` 任务，
  或删除 `succeeded` / `failed` 记录（`running` / `cancelled` 不可操作）

## 6. 调用限制

AI Gateway 会做以下限制：

- API Key 状态校验
- API Key 过期校验
- Key 级别模型白名单校验
- Key 级别 Scope 校验
- Key 级别每小时请求数限制

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
- 视频生成等按次计费的模型按 `request_cost` 记一次费用（图像理解当前记 0 费用）

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
    - model: "MiniMax-H3"
      provider: "minimax"
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

### cURL 图像理解接口

```bash
curl -X POST http://localhost:8080/api/ai-gateway/v1/image/understanding \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dtk_ai_xxx" \
  -d '{
    "prompt": "描述这张图片",
    "image": "data:image/png;base64,..."
  }'
```

### cURL 视频生成接口（提交后轮询 task_id）

```bash
curl -X POST http://localhost:8080/api/minimax/h3/v2/video_generation \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dtk_ai_xxx" \
  -d '{
    "model": "MiniMax-H3",
    "prompt": "一个男孩在海边打篮球",
    "resolution": "2K",
    "duration": 5,
    "ratio": "16:9"
  }'

curl "http://localhost:8080/api/minimax/h3/v2/query/video_generation/<task_id>" \
  -H "Authorization: Bearer dtk_ai_xxx"
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

async function understandImage(imageBase64) {
  const res = await fetch('http://localhost:8080/api/ai-gateway/v1/image/understanding', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${apiKey}`,
    },
    body: JSON.stringify({
      prompt: '描述这张图片',
      image: imageBase64,
    }),
  });

  const data = await res.json();
  console.log(data.text);
}

async function generateVideo() {
  const res = await fetch('http://localhost:8080/api/minimax/h3/v2/video_generation', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${apiKey}`,
    },
    body: JSON.stringify({
      model: 'MiniMax-H3',
      prompt: '极简白色产品摄影，智能手表，广告海报',
      resolution: '2K',
      duration: 5,
      ratio: '16:9'
    }),
  });

  const data = await res.json();
  console.log(data.task_id); // 后续轮询 /api/minimax/h3/v2/query/video_generation/:task_id
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

def understand_image(image_base64):
    resp = requests.post(
        f"{BASE_URL}/v1/image/understanding",
        headers={"Authorization": f"Bearer {API_KEY}"},
        json={
            "prompt": "描述这张图片",
            "image": image_base64,
        },
        timeout=120,
    )
    print(resp.json()["text"])

def generate_video():
    resp = requests.post(
        "http://localhost:8080/api/minimax/h3/v2/video_generation",
        headers={"Authorization": f"Bearer {API_KEY}"},
        json={
            "model": "MiniMax-H3",
            "prompt": "一只金毛坐在咖啡馆里，温暖晨光",
            "resolution": "2K",
            "duration": 5,
            "ratio": "16:9"
        },
        timeout=120,
    )
    task = resp.json()
    print(task["task_id"])  # 再轮询 http://localhost:8080/api/minimax/h3/v2/query/video_generation/<task_id>
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
