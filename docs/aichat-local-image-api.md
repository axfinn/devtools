# 本地出图接入说明（/ai-chat × 局域网 ComfyUI）

把局域网/本机跑的 ComfyUI（扩散模型，示例为 Qwen-Image-2.1 GGUF）接进 `/ai-chat`：

- 聊天页多一个 🎨 出图模式（附一张图即变成"按指令改这张图"）
- 对外提供一套 HTTP 接口，其中包含 OpenAI 兼容的 `/v1/images/generations`

## 为什么不是"再挂一个 Ollama 模型"

Ollama 底层是 llama.cpp，只封装语言/视觉语言模型推理，**没有 diffusion transformer 与 VAE 解码链路**，
它的模型库里也没有 Qwen-Image。扩散模型的 `.gguf` 虽然同叫 gguf，但结构不是语言模型，Ollama 会直接报不支持。

两者的分工：

| 环节 | 由谁负责 |
|---|---|
| 写提示词、扩写、翻译 | Ollama（`chat_plus.ollama_base_url`） |
| 出图 / 改图 | ComfyUI（`chat_plus.comfyui.base_url`） |

## 前置条件

1. ComfyUI 必须监听所有网卡：

   ```bash
   ./start.sh          # 等价于 python main.py --listen 0.0.0.0 --port 8188 --enable-cors-header "*"
   ```

   启动脚本默认就是 `0.0.0.0`；临时要改可 `COMFYUI_HOST=127.0.0.1 COMFYUI_PORT=8189 ./start.sh`。
   只监听 `127.0.0.1` 时，容器内的后端与局域网其他设备都连不上。

   验证（在拉起的机器以外执行）：`curl http://192.168.x.x:8188/system_stats` 返回 JSON 即通。

2. 模型文件就位（示例为 Qwen-Image-2.1 三件套）：

   ```text
   ComfyUI/models/diffusion_models/qwen-image-2.1-UC-Q4_K_M.gguf
   ComfyUI/models/text_encoders/qwen3vl_8b_int8_convrot.safetensors
   ComfyUI/models/vae/qwen_image_2.1_vae_bf16.safetensors
   ```

   ComfyUI 侧还需要 `ComfyUI-GGUF` 自定义节点（leejet 版才支持 `qwen_image21` 架构）。

3. 后端能直连该地址。容器部署时注意两点：

   - `no_proxy` 里要带上 ComfyUI 的地址（compose 里已含 `host.docker.internal`）
   - 走宿主机时可用 `http://host.docker.internal:8188`；走局域网 IP 则填 `http://192.168.x.x:8188`

## 配置

`backend/config.yaml`：

```yaml
chat_plus:
  password: "123654789"
  ollama_base_url: "http://192.168.31.147:11434"
  comfyui:
    base_url: "http://192.168.31.147:8188"   # 留空 = 关闭本地出图
    model: "qwen-image-2.1-UC-Q4_K_M.gguf"
    clip: "qwen3vl_8b_int8_convrot.safetensors"
    vae: "qwen_image_2.1_vae_bf16.safetensors"
    clip_type: "qwen_image"
    steps: 25
    cfg: 1.0
    sampler: "euler"
    scheduler: "simple"
    resolution: 1024
    timeout_sec: 900
    upload_dir: "aichat"
    public_base_url: ""
```

对应环境变量（docker 部署不必挂配置文件）：
`COMFYUI_BASE_URL`、`COMFYUI_MODEL`、`COMFYUI_CLIP`、`COMFYUI_VAE`、`COMFYUI_STEPS`、
`COMFYUI_TIMEOUT_SEC`、`COMFYUI_PUBLIC_BASE_URL`。

## 聊天页接口

鉴权：`X-Chat-Password: <chat_plus.password>`（与聊天页同一套密码门）。

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/chat-plus/image/status` | 探活 + 当前参数（前端据此启用/禁用 🎨 按钮） |
| POST | `/api/chat-plus/image/generate` | 提交任务，立即返回 `task_id` |
| GET | `/api/chat-plus/image/tasks/:id` | 轮询任务状态 |
| GET | `/api/chat-plus/image/tasks/:id/stream` | SSE 推送进度，直到终态 |
| POST | `/api/chat-plus/image/prompt/optimize` | 让文本大模型把一句大白话改写成可出图的提示词（30 次/分钟/IP） |

提交任务：

```bash
curl -X POST http://127.0.0.1:8082/api/chat-plus/image/generate \
  -H 'Content-Type: application/json' \
  -H 'X-Chat-Password: 123654789' \
  -d '{"prompt":"a red apple on a wooden table, soft studio light"}'
# {"task_id":"img_3f2a...","status":"queued","mode":"t2i","seed":123,"steps":25}
```

编辑模式（附参考图，`image` 支持 data URL 或 http 链接）：

```bash
curl -X POST http://127.0.0.1:8082/api/chat-plus/image/generate \
  -H 'Content-Type: application/json' \
  -H 'X-Chat-Password: 123654789' \
  -d '{"prompt":"make the sky a sunset, keep the sea unchanged","image":"https://example.com/beach.jpg"}'
```

SSE 事件体统一是任务 JSON，按 `status` 判断即可：

```text
data: {"task_id":"img_3f2a","status":"running","phase":"采样中 12/25","progress":48,...}
data: {"task_id":"img_3f2a","status":"succeeded","progress":100,"images":[{"url":"/api/chat/uploads/ai/20261028/ab12.png","width":1024,"height":1024}]}
```

## AI 帮写提示词

前端出图模式里的「✨ 优化提示词」走这个接口。它只是**改写建议**：结果回填到输入框，
用户可以继续改、也可以直接丢掉，不会自动开始出图。

```bash
curl -X POST http://127.0.0.1:8082/api/chat-plus/image/prompt/optimize \
  -H 'Content-Type: application/json' \
  -H 'X-Chat-Password: 123654789' \
  -d '{"prompt":"把背景换成海边","mode":"edit","model":"ollama:qwen3:14b",
       "image_model":"Qwen-Image 2.1","image":"data:image/jpeg;base64,..."}'
# {"prompt":"保持人物五官、姿态与白色连衣裙不变……","model":"ollama:qwen3:14b","mode":"edit","used_image":true}
```

| 字段 | 说明 |
|---|---|
| `prompt` | 用户原话（≤2000 字）。`describe` 模式可为空 |
| `model` | 用于改写的**文本**模型，一般传聊天页当前选中的模型；空/不可用时回落服务端默认 |
| `strict` | `true` = 用户在弹窗里明确选了模型，不可用时直接报错而不是回落到默认（避免「选了 A 拿到 B」） |
| `mode` | `t2i` 文生图 / `edit` 图生图 / `describe` 只有参考图没写要求；留空按有没有 `prompt`、`image` 推断 |
| `image_model` | 目标出图模型名，只用于让模型知道写法偏好（FLUX 系偏好英文） |
| `image` | 参考图 dataURL（≤8MB）。**只有** 局域网 Ollama 的视觉模型会真的看图，其它模型静默降级为纯文字改写 |

三种模式的写法要求不同，这是「图生图效果差」的主要修复点：

- `edit`：必须写成「编辑指令 + 保持不变清单」，否则模型会把整张图重画；
- `t2i`：按「主体与动作 → 场景 → 光线 → 镜头 → 风格」组织；
- `describe`：先看懂参考图，再写可复现该画面与风格的提示词。

系统提示词还硬性禁止「8K/杰作/masterpiece」这类对现代扩散模型无意义的堆砌词，以及负面提示词与参数。

## OpenAI 兼容接口

鉴权走 AI Gateway：`Authorization: Bearer <API Key>`（需 `media` 权限域且模型白名单含 `qwen-image-2.1`），
或带 `X-Super-Admin-Password` 用超管身份调用。

```bash
curl -X POST http://127.0.0.1:8082/api/ai-gateway/v1/images/generations \
  -H 'Authorization: Bearer sk-xxxxxxxxxxxxxxxx' \
  -H 'Content-Type: application/json' \
  -d '{"model":"qwen-image-2.1","prompt":"a white ceramic mug on marble","size":"1024x1024","n":1}'
```

返回（`response_format: "b64_json"` 时改为内联 base64）：

```json
{"created":1761628800,"model":"qwen-image-2.1","seed":1234567890,
 "data":[{"url":"/api/chat/uploads/ai/20261028/ab12.png","width":1024,"height":1024}]}
```

该端点语义是同步的：服务端会等到出图完成再返回，最长等待 `timeout_sec`。

## 已知限制

- **只出正方形**：`size` / `width,height` 会按**短边**收敛成正方形并对齐 64。
  Qwen-Image 的编码节点只接受一个 `resolution`，强行按宽高比出图会得到拉伸画面。
- **一次一张**：`n` 只支持 1（单卡串行出图）。
- **耗时 3~5 分钟/张**：25 步 @1024²（M4 Max 实测约 9 秒/步），所以全链路是异步任务，别用同步 HTTP 客户端硬等。
- **画质受量化影响**：Q4_K_M 会有细节损失，要求高可换 Q8_0。
- **任务表在内存里**：服务重启后旧 `task_id` 会查不到（结果图已落盘，URL 仍可访问）。

## 排查

| 现象 | 原因 / 处理 |
|---|---|
| `本地出图不可用：dial tcp ... connect: connection refused` | ComfyUI 没启动，或没加 `--listen 0.0.0.0` |
| 提交返回 `Value not in list: unet_name` | 配置里的 `model` / `clip` / `vae` 文件名与 ComfyUI 实际文件不一致 |
| 提交返回 `Unknown model architecture` | ComfyUI-GGUF 节点版本过旧，Qwen-Image-2.1 需 leejet 版 |
| 出图是彩色方块噪点 | 该系列模型在「编辑 + 非方形输出」下的已知问题；本接入已统一收敛为正方形 |
| 进度条一直 0% | WebSocket `/ws` 连不上（不影响出图），实际执行状态仍以 `/history` 为准 |
