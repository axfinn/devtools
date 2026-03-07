# 百炼图片工具 API

所有接口都以 `/api/bailian` 为前缀。

鉴权方式：
- Header: `X-Admin-Password: <管理密码>`
- Query: `?admin_password=<管理密码>`
- Body: `{ "admin_password": "<管理密码>" }`

## 1. 获取模型列表

`GET /api/bailian/models`

返回每个模型的：
- `enabled`
- `total_quota`
- `used_quota`
- `remaining_quota`
- `expires_at`
- `is_expired`

## 2. 创建调试任务

`POST /api/bailian/tasks`

请求体示例：

```json
{
  "admin_password": "your-password",
  "model": "qwen-image-2.0-pro",
  "prompt": "一只橘猫穿着宇航服，电影海报风格",
  "negative_prompt": "模糊、低清晰度、手部变形",
  "images": ["data:image/png;base64,..."],
  "size": "1328*1328",
  "count": 1,
  "seed": 20260307,
  "auto_poll": true,
  "wait_seconds": 30,
  "client_name": "marketing-site",
  "client_request_id": "order-12345",
  "parameters": {
    "watermark": false
  }
}
```

说明：
- `images` 支持 data URL 和公网 URL
- `multimodal` 模型最多 3 张图
- `image2video` 模型至少 1 张图
- `auto_poll=true` 时，后端会在 `wait_seconds` 内自动轮询异步任务

## 3. 开放给其他业务的通用 API

`POST /api/bailian/generate`

和调试接口使用相同请求体。推荐其他业务直接调用这个接口。

## 4. 任务列表

`GET /api/bailian/tasks?model=qwen-image-2.0-pro&status=succeeded&limit=20&offset=0`

返回任务历史列表，方便回看生成记录。

## 5. 任务详情

`GET /api/bailian/tasks/:id`

返回：
- 任务基本信息
- 请求快照
- 响应快照
- 当前结果
- 全流程流水

## 6. 任务流水

`GET /api/bailian/tasks/:id/events`

典型阶段：
- `task.created`
- `vendor.response`
- `vendor.poll`
- `vendor.request_failed`
- `vendor.poll_failed`

## 7. 手动轮询异步任务

`POST /api/bailian/tasks/:id/poll`

请求体：

```json
{
  "admin_password": "your-password",
  "wait_seconds": 1
}
```

## 本地额度保护

模型额度和截止时间在 `backend/config.yaml` 或环境变量中配置。

关键行为：
- 到期后拒绝继续调用
- 本地累计次数达到 `total_quota` 后拒绝继续调用
- 所有请求会写入 SQLite，防止跨业务重复消耗额度后无法追踪

## 配置项

新增配置段：

```yaml
bailian:
  api_key: ""
  admin_password: ""
  base_url: "https://dashscope.aliyuncs.com"
  default_wait_seconds: 45
  task_retention_days: 180
  models:
    - name: "qwen-image-2.0-pro"
      type: "multimodal"
      enabled: true
      total_quota: 100
      expires_at: "2026-06-01"
      default_size: "1328x1328"
```
