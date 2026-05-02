---
name: terminal
description: SSH 终端 — 创建/管理 SSH 会话，支持密码/密钥认证、keep_alive 持久化、命令历史。触发：SSH连接、远程终端、连接服务器
triggers:
  - "SSH"
  - "远程终端"
  - "连接服务器"
  - "terminal"
---

# SSH 终端 (Terminal)

通过 DevTools 后端 API 管理 SSH 会话，支持密码/密钥认证和 WebSocket 实时通信。后端默认运行在 `https://t.jaxiu.cn`。

WebSocket 地址: `wss://t.jaxiu.cn/api/terminal/{id}/ws?user_token=xxx`

## 认证

- 先通过 `/api/terminal/login` 获取或生成 `user_token`
- 所有操作需要 `user_token` 或 `creator_key`

## 1. 用户登录

```bash
curl -s -X POST https://t.jaxiu.cn/api/terminal/login \
  -H "Content-Type: application/json" \
  -d '{}'
# 返回: {"user_token":"xxx"}
```

## 2. 创建 SSH 会话

```bash
curl -s -X POST https://t.jaxiu.cn/api/terminal \
  -H "Content-Type: application/json" \
  -d '{
    "host": "192.168.1.100",
    "port": 22,
    "username": "root",
    "password": "your_password",
    "user_token": "xxx",
    "keep_alive": true
  }'
# 返回: {"id":"xxx","creator_key":"xxx","status":"connected",...}
```

也支持密钥认证:

```bash
curl -s -X POST https://t.jaxiu.cn/api/terminal \
  -H "Content-Type: application/json" \
  -d '{
    "host": "192.168.1.100",
    "port": 22,
    "username": "root",
    "private_key": "-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----",
    "user_token": "xxx",
    "keep_alive": true
  }'
```

- `keep_alive: true`: 会话会被前端优先恢复，服务端保留时间更长
- 最大会话数: 10个/用户

## 3. 查看会话列表

```bash
curl -s "https://t.jaxiu.cn/api/terminal/list?user_token=xxx"
```

## 4. 获取会话详情

```bash
curl -s "https://t.jaxiu.cn/api/terminal/{id}?user_token=xxx"
```

## 5. 获取命令历史

```bash
curl -s "https://t.jaxiu.cn/api/terminal/{id}/history?user_token=xxx"
```

## 6. 恢复/断开会话

```bash
# 恢复 WebSocket 连接
curl -s -X POST https://t.jaxiu.cn/api/terminal/{id}/resume \
  -H "Content-Type: application/json" \
  -d '{"user_token":"xxx"}'

# 断开 SSH 连接（保留会话记录）
curl -s -X POST https://t.jaxiu.cn/api/terminal/{id}/disconnect \
  -H "Content-Type: application/json" \
  -d '{"user_token":"xxx"}'
```

## 7. 更新会话

```bash
curl -s -X PUT https://t.jaxiu.cn/api/terminal/{id} \
  -H "Content-Type: application/json" \
  -d '{
    "user_token": "xxx",
    "action": "rename",
    "title": "新名称"
  }'
```

action: `rename`、`resize`（窗口大小变更）、`extend`（延长过期）

## 8. 删除会话

```bash
curl -s -X DELETE "https://t.jaxiu.cn/api/terminal/{id}?creator_key=xxx"
```

## 快速操作

当用户说"连接服务器"时，先确保有 user_token，再创建 SSH 会话。
keep_alive 的会话在页面刷新/回前台/网络恢复后会被优先恢复。

Base URL: `https://t.jaxiu.cn`
