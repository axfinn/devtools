---
name: voice-inbox
description: 语音收件箱 — 上传录音、语音转文字、自动创建事项任务。触发：语音记录、录音备忘、语音转文字、语音收件箱
triggers:
  - "语音备忘"
  - "语音记录"
  - "录音"
  - "语音转文字"
  - "voice memo"
  - "voice-inbox"
---

# 语音收件箱 (Voice Inbox)

通过 DevTools 后端 API 管理语音备忘录，支持语音转文字并自动创建事项任务。后端默认运行在 `https://t.jaxiu.cn`。

## 1. 上传录音

```bash
curl -X POST https://t.jaxiu.cn/api/voicememo/upload \
  -F "file=@recording.mp3"
# 返回: {"id":"xxx","filename":"voicememo_xxx.mp3","duration_seconds":45,...}
```

## 2. 查看录音列表

```bash
curl -s https://t.jaxiu.cn/api/voicememo/list
```

## 3. 获取录音详情

```bash
curl -s https://t.jaxiu.cn/api/voicememo/{id}
```

## 4. 播放录音

```bash
curl -o playback.mp3 https://t.jaxiu.cn/api/voicememo/{id}/audio
```

## 5. 语音转文字

```bash
curl -s -X POST https://t.jaxiu.cn/api/voicememo/{id}/transcribe \
  -H "Content-Type: application/json" \
  -d '{}'
# 返回: {"text":"转写后的文字内容","language":"zh",...}
```

## 6. 从语音创建事项

```bash
curl -s -X POST https://t.jaxiu.cn/api/voicememo/{id}/planner-task \
  -H "Content-Type: application/json" \
  -d '{
    "creator_key": "planner_profile_creator_key",
    "profile_id": "planner_profile_id"
  }'
# 自动从语音中提取任务信息并创建到事项管理
```

## 7. 更新/删除

```bash
# 更新标题
curl -s -X PUT https://t.jaxiu.cn/api/voicememo/{id} \
  -H "Content-Type: application/json" \
  -d '{"title":"更新后的标题"}'

# 删除
curl -s -X DELETE https://t.jaxiu.cn/api/voicememo/{id}
```

## 快速操作

当用户说"帮我记录这个录音"或"这段语音转成文字"时，使用对应 API。
语音 → 文字 → 事项任务的一键流转是核心使用场景。

Base URL: `https://t.jaxiu.cn`
