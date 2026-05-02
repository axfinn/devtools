---
name: chat
description: 聊天室 — 创建房间、实时聊天、AI 机器人（10种预设角色+TTS语音）、图片/视频上传。触发：创建聊天室、聊天、加AI机器人、聊天室
triggers:
  - "聊天室"
  - "创建房间"
  - "加机器人"
  - "chat"
  - "AI聊天"
---

# 聊天室 (Chat Room)

通过 DevTools 后端 API 创建聊天房间、添加 AI 机器人和实时通信。后端默认运行在 `https://t.jaxiu.cn`。

WebSocket 地址: `wss://t.jaxiu.cn/api/chat/room/{id}/ws?nickname=xxx`

## 1. 房间管理

### 1.1 创建房间

```bash
curl -s -X POST https://t.jaxiu.cn/api/chat/room \
  -H "Content-Type: application/json" \
  -d '{
    "name": "技术交流",
    "password": "optional_password"
  }'
# 返回: {"id":"abc12345","name":"技术交流",...}
```

### 1.2 查看房间列表

```bash
curl -s https://t.jaxiu.cn/api/chat/rooms
```

### 1.3 获取房间信息

```bash
curl -s https://t.jaxiu.cn/api/chat/room/{id}
```

### 1.4 加入房间

```bash
curl -s -X POST https://t.jaxiu.cn/api/chat/room/{id}/join \
  -H "Content-Type: application/json" \
  -d '{"nickname":"小明","password":"optional_password"}'
```

### 1.5 获取历史消息

```bash
curl -s "https://t.jaxiu.cn/api/chat/room/{id}/messages?limit=50"
```

## 2. 文件上传

```bash
# 图片（最大 5MB）/ 视频（最大 50MB）
curl -X POST https://t.jaxiu.cn/api/chat/upload \
  -F "file=@image.png"
```

## 3. AI 机器人

### 3.1 10 种预设角色

| Key | 角色 | 默认音色 |
|-----|------|---------|
| `general` | 🤖 小助手 | XiaoxiaoNeural |
| `code` | 🤖 码农 | YunxiNeural |
| `translate` | 🤖 译者 | XiaoyiNeural |
| `customer` | 🤖 客服 | XiaoxiaoNeural |
| `psychology` | 🤖 心理咨询师 | XiaomouNeural |
| `student_girl` | 🤖 学生妹 | XiaoyiNeural |
| `college` | 🤖 大学生 | YunxiNeural |
| `girlfriend` | 🤖 电子女朋友 | XiaoxiaoNeural |
| `uncle` | 🤖 大叔 | YunyangNeural |
| `kid` | 🤖 小屁孩 | YunxiNeural |

### 3.2 添加机器人

```bash
curl -s -X POST https://t.jaxiu.cn/api/chat/room/{id}/bot \
  -H "Content-Type: application/json" \
  -d '{
    "role": "girlfriend",
    "nickname": "小美",
    "system_prompt": "你是一个可爱的女朋友，说话温柔甜蜜",
    "enable_tts": true
  }'
```

- `role`: 必填，从上述 10 个 key 中选择
- `nickname`: 可选，覆盖默认昵称
- `system_prompt`: 可选，自定义系统提示
- `enable_tts`: 可选，是否启用语音合成（TTS）

### 3.3 查看 bot 配置

```bash
curl -s https://t.jaxiu.cn/api/chat/room/{id}/bot
```

### 3.4 删除 bot

```bash
curl -s -X DELETE https://t.jaxiu.cn/api/chat/room/{id}/bot
```

### 3.5 停止 bot 回复

```bash
curl -s -X POST https://t.jaxiu.cn/api/chat/room/{id}/bot/stop
```

## 快速操作

当用户说"创建一个聊天室"时，调用创建房间 API。
当用户说"给房间加个 AI 助手"时，调用添加 bot API。
TTS 语音在 bot 回复时会自动生成 mp3，前端自动播放。

Base URL: `https://t.jaxiu.cn`
