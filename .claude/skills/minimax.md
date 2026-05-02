---
name: minimax
description: MiniMax AI 全能力聚合 — 音乐生成、视频生成、图片生成、语音合成、音色克隆、AI 对话、图片理解、结果分享
triggers:
  - "minimax"
  - "生成音乐"
  - "生成视频"
  - "语音合成"
  - "音色克隆"
  - "文生图"
  - "图生视频"
  - "翻唱"
  - "歌词生成"
  - "图片理解"
  - "声音设计"
  - "AI 对话"
  - "声音复刻"
---

# MiniMax AI 全能力平台

通过 DevTools 后端代理 MiniMax 官方 API，支持所有媒体生成能力。后端默认运行在 `https://t.jaxiu.cn`（Docker）或 `http://localhost:8080`（本地开发）。

## 鉴权

所有请求需要 API Key：
- **Header**: `Authorization: Bearer dtk_ai_xxx`（AI Gateway 分配的 API Key）
- **或**: `X-Super-Admin-Password: your_admin_password`（超级管理员密码，跳过权限检查）

---

## 1. 音乐生成 (Music Generation)

### 1.1 生成音乐 (music-2.5 / music-2.6)

根据描述生成原创音乐。

```bash
curl -s -X POST https://t.jaxiu.cn/api/minimax/token-plan/v1/generations \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "music-2.6",
    "prompt": "轻快温暖的钢琴曲，适合午后咖啡厅",
    "duration": 180,
    "output_format": "url"
  }'
```

返回 `{"task_id":"mmt_xxx","status":"pending"}`，随后轮询：

```bash
# 轮询任务状态
curl -s https://t.jaxiu.cn/api/minimax/token-plan/tasks/{task_id} \
  -H "Authorization: Bearer YOUR_API_KEY"
# 完成时 result_urls 包含音频下载链接
```

支持的 duration：30-300 秒（music-2.5: 最长120s, music-2.6: 最长300s）。

### 1.2 翻唱改编 (music-cover)

两步流程：先预处理原曲获取特征，再生成翻唱。

**Step 1 — 获取 cover_feature_id：**
```bash
curl -s -X POST https://t.jaxiu.cn/api/minimax/music/v1/cover_preprocess \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "music-cover",
    "audio_url": "https://example.com/original_song.mp3"
  }'
```

**Step 2 — 生成翻唱：**
```bash
curl -s -X POST https://t.jaxiu.cn/api/minimax/token-plan/v1/generations \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "music-cover",
    "prompt": "摇滚风格，电吉他，强烈鼓点",
    "cover_feature_id": "从上一步获取的ID",
    "lyrics": "[Verse]\n重新填写的歌词...",
    "audio_duration": 240,
    "output_format": "url"
  }'
```

### 1.3 歌词生成

```bash
curl -s -X POST https://t.jaxiu.cn/api/minimax/music/v1/lyrics_generation \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "mode": "write_full_song",
    "prompt": "关于青春和梦想的流行歌曲，中文"
  }'
```

mode 可选：`write_full_song`（完整歌词）、`write_verse_only`（仅主歌）、`write_chorus_only`（仅副歌）。

---

## 2. 视频生成 (Video Generation)

### 2.1 文生视频 (T2V)

支持模型：`MiniMax-Hailuo-2.3-Fast`（6s快速）、`MiniMax-Hailuo-2.3`（标准）、`MiniMax-Hailuo-02`、`T2V-01-Director`、`T2V-01`

```bash
curl -s -X POST https://t.jaxiu.cn/api/minimax/token-plan/v1/generations \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MiniMax-Hailuo-2.3-Fast",
    "prompt": "一只橘猫在阳光下的草地上追逐蝴蝶，电影质感，浅景深",
    "duration": 6,
    "resolution": "768P"
  }'
```

### 2.2 图生视频 (I2V)

使用 `MiniMax-Hailuo-2.3-Fast` 模型，传入首帧图片：

```bash
curl -s -X POST https://t.jaxiu.cn/api/minimax/token-plan/v1/generations \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MiniMax-Hailuo-2.3-Fast",
    "prompt": "画面缓慢移动，风吹过树叶",
    "first_frame_image": "https://example.com/first_frame.png",
    "duration": 6
  }'
```

### 2.3 轮询视频任务 & 下载

```bash
# 查看任务（包含 result_urls / video_url）
curl -s https://t.jaxiu.cn/api/minimax/token-plan/tasks/{task_id} \
  -H "Authorization: Bearer YOUR_API_KEY"

# 直接下载产物
curl -o output.mp4 https://t.jaxiu.cn/api/minimax/token-plan/tasks/{task_id}/download \
  -H "Authorization: Bearer YOUR_API_KEY"

# 列出所有任务
curl -s https://t.jaxiu.cn/api/minimax/token-plan/tasks?limit=20 \
  -H "Authorization: Bearer YOUR_API_KEY"
```

视频任务最长等待 15 分钟。

---

## 3. 图片生成 (Image Generation)

### 3.1 文生图 (image-01)

```bash
curl -s -X POST https://t.jaxiu.cn/api/minimax/token-plan/v1/generations \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "image-01",
    "prompt": "一位穿汉服的少女站在樱花树下，日系动漫风格，柔和光线，4K精细",
    "aspect_ratio": "16:9",
    "n": 1,
    "response_format": "url"
  }'
```

支持的 aspect_ratio：`1:1`、`16:9`、`9:16`、`4:3`、`3:4`、`3:2`、`2:3`、`21:9`、`9:21`

对应传统尺寸：`1024x1024`(1:1)、`1280x720`(16:9)、`720x1280`(9:16)、`1152x864`(4:3)、`864x1152`(3:4)、`1248x832`(3:2)、`832x1248`(2:3)、`1344x576`(21:9)、`576x1344`(9:21)

### 3.2 图生图 (image-01-live)

```bash
curl -s -X POST https://t.jaxiu.cn/api/minimax/token-plan/v1/generations \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "image-01-live",
    "prompt": "将这张照片变成水彩画风格",
    "image": "https://example.com/photo.jpg",
    "response_format": "url"
  }'
```

图片任务最长等待 5 分钟。

---

## 4. 语音合成 (TTS)

### 4.1 同步 TTS（短文本）

模型：`speech-2.8-hd`（推荐）、`speech-2.8-turbo`、`speech-2.6-hd`、`speech-2.6-turbo`、`speech-02-hd`、`speech-02-turbo`、`speech-01-hd`、`speech-01-turbo`

```bash
# 方式1 — TTS 端点（简化参数）
curl -o output.mp3 https://t.jaxiu.cn/api/minimax/tts/v1/generations \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "speech-2.8-hd",
    "text": "你好，这是 MiniMax 语音合成测试。声音自然流畅。",
    "voice": "shanghai",
    "speed": 1.0,
    "audio_format": "mp3"
  }'

# 方式2 — Speech 端点（官方完整参数）
curl -s -X POST https://t.jaxiu.cn/api/minimax/speech/v1/t2a_v2 \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "speech-2.8-hd",
    "text": "你好，这是语音合成测试",
    "voice_setting": {
      "voice_id": "male-qn-qingse",
      "speed": 1.0
    },
    "audio_setting": {
      "sample_rate": 32000,
      "format": "mp3"
    }
  }'
```

常用系统音色 ID：
- `male-qn-qingse` — 青涩男声
- `female-shaonv` — 少女
- `male-qn-jingying` — 精英男声
- `shanghai` — 上海话
- `cantonese` — 粤语
- `woman` — 通用女声
- `man` — 通用男声

### 4.2 异步长文本 TTS

用于长文本语音合成：

```bash
# 提交异步任务
curl -s -X POST https://t.jaxiu.cn/api/minimax/speech/v1/t2a_async_v2 \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "speech-2.8-hd",
    "text": "这是一段很长的文本...",
    "voice_id": "male-qn-qingse"
  }'

# 查询异步任务
curl -s "https://t.jaxiu.cn/api/minimax/speech/v1/query/t2a_async_query_v2?task_id={task_id}" \
  -H "Authorization: Bearer YOUR_API_KEY"

# 列出所有异步语音任务
curl -s https://t.jaxiu.cn/api/minimax/speech/tasks?limit=20 \
  -H "Authorization: Bearer YOUR_API_KEY"
```

---

## 5. 音色设计 & 复刻 (Voice Design & Cloning)

### 5.1 音色设计（用文字描述生成音色）

用文字描述想要的音色特征，AI 自动设计：

```bash
curl -s -X POST https://t.jaxiu.cn/api/minimax/speech/v1/voice_design \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "voice_id": "custom-designer-001",
    "prompt": "温柔知性的中文女声，适合朗读散文和有声书，语速稍慢",
    "preview_text": "你好，这是我为你设计的声音，希望你喜欢。"
  }'
```

### 5.2 音色克隆 / 复刻

上传一段音频样本，克隆出相似的声音：

**复制文件方式：**
```bash
curl -s -X POST https://t.jaxiu.cn/api/minimax/speech/v1/voice_clone \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "voice_id": "my-cloned-voice",
    "file_id": "从文件上传获取的file_id",
    "need_noise_reduction": true,
    "voice_name": "我的克隆音色"
  }'
```

**直接上传音频（Voice Cloning 端点）：**
```bash
curl -X POST https://t.jaxiu.cn/api/minimax/voice-cloning/upload \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -F "name=我的音色" \
  -F "audio_file=@voice_sample.wav"
```

音频样本要求：10-60 秒，清晰单人声，无背景噪音，最大 10MB。

### 5.3 使用克隆音色 TTS

```bash
curl -X POST https://t.jaxiu.cn/api/minimax/voice-cloning/tts \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "speech-2.8-hd",
    "text": "你好，这是我的克隆声音",
    "voice_id": "my-cloned-voice",
    "speed": 1.0,
    "audio_format": "mp3"
  }' -o output.mp3
```

### 5.4 音色管理

```bash
# 查询系统音色列表
curl -s -X POST https://t.jaxiu.cn/api/minimax/speech/v1/get_voice \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{}'

# 查询单个音色详情
curl -s -X POST https://t.jaxiu.cn/api/minimax/speech/v1/get_voice \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"voice_id":"male-qn-qingse"}'

# 查看我的音色列表
curl -s https://t.jaxiu.cn/api/minimax/voice-cloning/voices \
  -H "Authorization: Bearer YOUR_API_KEY"

# 删除自定义音色
curl -X DELETE https://t.jaxiu.cn/api/minimax/voice-cloning/voices/{id} \
  -H "Authorization: Bearer YOUR_API_KEY"
```

---

## 6. 语音文件管理 (Speech File Management)

```bash
# 上传文件（用途: voice_clone / prompt_audio / t2a_async_input）
curl -s -X POST https://t.jaxiu.cn/api/minimax/speech/v1/files/upload \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -F "purpose=voice_clone" \
  -F "file=@sample.wav"

# 列出文件
curl -s https://t.jaxiu.cn/api/minimax/speech/v1/files/list \
  -H "Authorization: Bearer YOUR_API_KEY"

# 获取文件元信息
curl -s "https://t.jaxiu.cn/api/minimax/speech/v1/files/retrieve?file_id={file_id}" \
  -H "Authorization: Bearer YOUR_API_KEY"

# 下载文件内容
curl -o file.mp3 "https://t.jaxiu.cn/api/minimax/speech/v1/files/retrieve_content?file_id={file_id}" \
  -H "Authorization: Bearer YOUR_API_KEY"

# 删除文件
curl -s -X POST https://t.jaxiu.cn/api/minimax/speech/v1/files/delete \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"file_id":"{file_id}"}'
```

---

## 7. AI 对话 (Chat Completions)

### 7.1 MiniMax LLM 对话

```bash
curl -s -X POST https://t.jaxiu.cn/api/ai-gateway/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MiniMax-LLM",
    "messages": [
      {"role": "user", "content": "请写一首关于秋天的七言绝句"}
    ],
    "temperature": 0.7
  }'
```

### 7.2 Anthropic 协议代理

通过 MiniMax Anthropic 兼容协议调用：

```bash
curl -s -X POST https://t.jaxiu.cn/api/minimax/anthropic/v1/messages \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MiniMax-M2.5",
    "max_tokens": 1024,
    "messages": [
      {"role": "user", "content": "解释量子计算的基本原理"}
    ]
  }'
```

---

## 8. 图片理解 (Image Understanding)

上传图片进行 AI 分析和理解：

```bash
# 通过 URL / Base64
curl -s -X POST https://t.jaxiu.cn/api/ai-gateway/v1/image/understanding \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "image": "data:image/png;base64,iVBORw0KGgo...",
    "prompt": "请详细描述这张图片的内容，包括人物、场景、动作和氛围",
    "tool": "understand_image"
  }'

# 通过文件上传
curl -s -X POST https://t.jaxiu.cn/api/ai-gateway/v1/image/understanding/file \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -F "file=@photo.jpg" \
  -F "prompt=描述这张图片"
```

---

## 9. 结果分享 (Result Shares)

将生成的内容（音乐、视频、图片、文本等）保存为可分享的页面：

```bash
# 创建分享
curl -s -X POST https://t.jaxiu.cn/api/minimax/result-shares \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "我的 AI 生成作品",
    "summary": "MiniMax 生成的音乐和视频",
    "result_type": "media",
    "model": "music-2.6",
    "payload": {
      "prompt": "轻快的钢琴曲",
      "task_id": "mmt_xxx"
    },
    "assets": [
      {
        "url": "https://example.com/output.mp3",
        "kind": "audio",
        "content_type": "audio/mpeg",
        "filename": "music.mp3"
      }
    ]
  }'

# 查看分享（返回分享页面 URL: /minimax/share/{id}）
curl -s https://t.jaxiu.cn/api/minimax/result-shares/{id}

# 访问分享中的媒体资产
curl -o asset.mp3 https://t.jaxiu.cn/api/minimax/result-shares/{id}/assets/asset_01
```

result_type 可选：`text`、`lyrics`、`image`、`audio`、`video`、`media`（混合）、`other`

---

## 10. 聊天室 AI Bot

在聊天室中添加 AI 机器人（10 种预设角色）：

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

```bash
# 在聊天室添加 Bot
curl -s -X POST https://t.jaxiu.cn/api/chat/room/{room_id}/bot \
  -H "Content-Type: application/json" \
  -d '{"role":"girlfriend","enable_tts":true}'

# 获取 Bot 配置
curl -s https://t.jaxiu.cn/api/chat/room/{room_id}/bot

# 删除 Bot
curl -s -X DELETE https://t.jaxiu.cn/api/chat/room/{room_id}/bot
```

---

## 使用示例

你可以直接向我提出请求，我来调用对应的 API：

- 🎵 **音乐**: "生成一首关于夏天的轻快钢琴曲" / "把这首歌翻唱成爵士风格"
- 🎤 **歌词**: "写一段关于友情的流行歌词"
- 🎬 **视频**: "生成一个海浪拍打礁石的慢动作视频" / "用这张图生成一个延时摄影视频"
- 🖼️ **图片**: "画一幅赛博朋克风格的未来城市" / "把这张照片改成梵高星空风格"
- 🔊 **语音**: "用温柔女声朗读这首诗" / "把这段新闻转成语音"
- 🎙️ **音色**: "设计一个适合讲故事的温暖声音" / "克隆我的声音"
- 💬 **对话**: "用 MiniMax M2.5 解答量子力学入门问题"
- 👁️ **识图**: "分析这张图片里有什么"
- 📤 **分享**: "帮我分享这段 AI 生成的音乐"
- 🤖 **Bot**: "在聊天室加一个 AI 助手"

## 注意事项

- **异步轮询**: 音乐/视频/图片生成为异步任务，提交后返回 `task_id`，需要轮询等待结果
- **超时时间**: 视频最长 15 分钟，音乐 8 分钟，图片 5 分钟
- **输出格式**: 音乐和图片默认使用 `output_format: url` / `response_format: url`，返回可直接下载的 URL
- **文件限制**: 音色克隆音频 ≤ 10MB，图片理解 ≤ 10MB，结果分享资产 ≤ 120MB
- **API Key**: 需要在 DevTools 后台先创建 AI Gateway API Key，或使用超管密码
