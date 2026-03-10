# 图像理解 API 文档

本项目提供两套图像理解接口：

1. **直接接口（无需 API Key）**：`/api/image-understanding/*`
2. **API Gateway 接口（需要 API Key）**：`/api/api-gateway/v1/image/understanding*`

两套接口最终调用同一套 MiniMax MCP 图像理解能力，但鉴权、输入格式和日志有所不同。

---

## 基础信息

- **服务前缀**：`/api`
- **图片大小限制**：10MB
- **默认提示词**：`请简洁描述图片内容，提取关键对象、场景和文字信息。`
- **超时**：由 `config.yaml` 的 `minimax_mcp.timeout_seconds` 控制（默认 300 秒）
- **内容类型**：
  - JSON 请求：`application/json`
  - 文件上传：`multipart/form-data`

---

## 1) 直接接口（无需 API Key）

### 1.1 获取可用工具

**GET** `/api/image-understanding/tools`

**响应示例**
```json
{
  "tools": [
    {
      "name": "understand_image",
      "description": "…",
      "inputSchema": { "properties": { "image_source": { "type": "string" } } }
    }
  ]
}
```

### 1.2 JSON 图像理解

**POST** `/api/image-understanding/describe`

**请求体**
```json
{
  "image": "data:image/png;base64,....",
  "prompt": "请描述图片内容",
  "tool": "understand_image",
  "args": {
    "detail": "high"
  }
}
```

**字段说明**
- `image`：必填，支持 `data:image/*;base64,` 或纯 base64
- `prompt`：可选，默认使用系统提示词
- `tool`：可选，指定工具名称；不传则自动选择
- `args`：可选，透传给工具的自定义参数

**响应示例**
```json
{
  "tool": "understand_image",
  "text": "图片内容描述…",
  "result": { "content": "..." },
  "args_preview": { "image_source": "/tmp/minimax-image-xxx.png", "prompt": "..." }
}
```

### 1.3 文件上传图像理解

**POST** `/api/image-understanding/describe-file`

**表单字段**
- `file`：必填，图片文件
- `prompt`：可选
- `tool`：可选
- `args`：可选，JSON 字符串

**curl 示例**
```bash
curl -X POST http://localhost:8082/api/image-understanding/describe-file \
  -F "file=@/path/to/image.png" \
  -F "prompt=请描述图片内容"
```

**响应格式**同 1.2。

---

## 2) API Gateway 接口（需要 API Key）

这组接口会校验 API Key 权限并记录调用日志。

**鉴权**
- Header：`Authorization: Bearer <API Key>` 或 `X-API-Key: <API Key>`
- API Key 需具备 `media` scope，并允许模型 `minimax-mcp-understand-image`

### 2.1 JSON 图像理解

**POST** `/api/api-gateway/v1/image/understanding`

**请求体**
```json
{
  "image": "data:image/png;base64,....",
  "prompt": "请描述图片内容",
  "tool": "understand_image",
  "args": {
    "detail": "high"
  }
}
```

**curl 示例**
```bash
curl -X POST http://localhost:8082/api/api-gateway/v1/image/understanding \
  -H "Authorization: Bearer dtk_ai_xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "请描述图片内容",
    "image": "data:image/png;base64,...."
  }'
```

**响应示例**
```json
{
  "tool": "understand_image",
  "model": "minimax-mcp-understand-image",
  "text": "图片内容描述…",
  "result": { "content": "..." },
  "args_preview": { "image_source": "/tmp/minimax-image-xxx.png", "prompt": "..." }
}
```

### 2.2 文件上传图像理解

**POST** `/api/api-gateway/v1/image/understanding/file`

**表单字段**
- `file`：必填，图片文件
- `prompt`：可选
- `tool`：可选
- `args`：可选，JSON 字符串

**curl 示例**
```bash
curl -X POST http://localhost:8082/api/api-gateway/v1/image/understanding/file \
  -H "Authorization: Bearer dtk_ai_xxx" \
  -F "file=@/path/to/image.png" \
  -F "prompt=请描述图片内容"
```

**响应格式**同 2.1。

---

## 常见错误

- `400 缺少 image/file`：请求参数不完整
- `400 图片大小不能超过 10MB`
- `401/403 API Key 无效或无权限`（仅 API Gateway）
- `502/503 上游服务错误或未配置 API Key`

---

## 配置说明（MiniMax MCP）

见 `backend/config.example.yaml`：
```yaml
minimax_mcp:
  api_key: ""
  api_host: "https://api.minimaxi.com"
  command: "uvx"
  args:
    - "minimax-coding-plan-mcp"
  timeout_seconds: 300
  transport: "line"
```
