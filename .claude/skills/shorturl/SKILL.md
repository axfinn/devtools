---
name: shorturl
description: 短链生成 — 创建/管理短链接、自定义ID、点击统计。触发：短链、短网址、缩短链接、生成短链
triggers:
  - "短链"
  - "短网址"
  - "缩短链接"
  - "生成短链"
  - "shorturl"
---

# 短链生成 (Short URL)

通过 DevTools 后端 API 创建短链接。后端默认运行在 `https://t.jaxiu.cn`。

## 1. 创建短链

### 无密码（随机ID + 限流）

```bash
curl -s -X POST https://t.jaxiu.cn/api/shorturl \
  -H "Content-Type: application/json" \
  -d '{
    "original_url": "https://example.com/very/long/url/that/needs/shortening",
    "expires_in": 30,
    "max_clicks": 1000
  }'
# 返回: {"id":"abc12345","short_url":"https://t.jaxiu.cn/s/abc12345"}
```

限流: 10次/IP/小时

### 有密码（可自定义短链ID，无限流）

```bash
curl -s -X POST https://t.jaxiu.cn/api/shorturl \
  -H "Content-Type: application/json" \
  -d '{
    "original_url": "https://example.com",
    "custom_id": "my-link",
    "password": "your_shorturl_password"
  }'
# 返回: {"id":"my-link","short_url":"https://t.jaxiu.cn/s/my-link"}
```

参数说明:
- `original_url`: 必填，原始链接
- `expires_in`: 可选，过期天数（默认30天）
- `max_clicks`: 可选，最大点击次数（默认1000）
- `custom_id`: 需要密码，自定义短链ID
- `password`: 配置文件中的 `shorturl.password`

## 2. 查看短链列表

```bash
curl -s https://t.jaxiu.cn/api/shorturl/list
```

## 3. 点击统计

```bash
curl -s https://t.jaxiu.cn/api/shorturl/{id}/stats
# 返回: {"id":"xxx","original_url":"...","clicks":42,...}
```

## 4. 更新/删除

```bash
# 更新（需密码）
curl -s -X PUT https://t.jaxiu.cn/api/shorturl/{id} \
  -H "Content-Type: application/json" \
  -d '{"password":"xxx","original_url":"https://new-url.com","expires_in":60}'

# 删除（需密码）
curl -s -X DELETE "https://t.jaxiu.cn/api/shorturl/{id}?password=xxx"
```

## 5. 访问短链

浏览器访问: `https://t.jaxiu.cn/s/{id}` 自动 302 跳转到原始链接。

## 快速操作

当用户说"把这个链接缩短"时，调用创建短链 API。
短链格式: `https://t.jaxiu.cn/s/{id}`

Base URL: `https://t.jaxiu.cn`
