# AGENTS.md

本文件是 DevTools 项目的唯一权威文档。供 AI Agent（Claude Code / Codex）使用。所有其他文档为辅助参考，以本文件为准。

---

## 45 模块地图

| # | 模块 | Handler 文件 | Route 文件 | View 组件 | 主要功能 |
|---|------|------------|----------|---------|---------|
| 1 | **Paste 粘贴板** | `handlers/paste.go` | `routes/paste.go` | `views/collab/PasteBin.vue` | 临时分享，最大 100KB，访问次数/过期时间限制 |
| 2 | **ShortURL 短链** | `handlers/shorturl.go` | `routes/shorturl.go` | `views/collab/ShortUrl.vue` | 短链接服务，支持自定义 ID，限流 10/h/IP |
| 3 | **MockAPI** | `handlers/mockapi.go` | `routes/mockapi.go` | `views/dev/MockApi.vue` | Mock API 端点，任意 HTTP 方法，支持请求日志 |
| 4 | **MDShare Markdown** | `handlers/mdshare.go` | `routes/mdshare.go` | `views/draw/MarkdownTool.vue` | 可分享 Markdown，2-10 次访问限制 |
| 5 | **Excalidraw** | `handlers/excalidraw.go` | `routes/excalidraw.go` | `views/draw/ExcalidrawTool.vue` | Excalidraw 画图，支持云端保存、密码保护、导出 PNG/SVG |
| 6 | **PhotoWall 照片墙** | `handlers/photowall.go` | `routes/photowall.go` | `views/life/PhotoWallTool.vue` | 档案照片墙，分类/时间线，打包下载 ZIP |
| 7 | **Chat 聊天室** | `handlers/chat.go` | `routes/chat.go` | `views/collab/ChatRoom.vue` | WebSocket 实时聊天，图片/视频上传，AI 机器人，TTS |
| 8 | **Terminal SSH** | `handlers/terminal.go` | `routes/terminal.go` | `views/dev/TerminalTool.vue` | Web SSH，支持密码/私钥，keep_alive，命令历史 |
| 9 | **DNS IP** | `handlers/dns.go` | `routes/dns_image.go` | `views/convert/DnsTool.vue` | DNS 解析（A/AAAA/CNAME/MX/NS/TXT），客户端 IP 查询 |
| 10 | **Analysis 分析** | `handlers/analysis.go` | `routes/analysis.go` | — | URL 内容分析，提取最新站点 URL |
| 11 | **EdgeTTS** | `handlers/edge_tts.go` | `routes/edge_tts.go` | `views/dev/EdgeTTSTool.vue` | Edge 语音合成（TTS），Python FastAPI + edge-tts |
| 12 | **Game 游戏** | `handlers/game.go` | `routes/game.go` | `views/life/GameHall.vue` | 游戏大厅，Arcade Room 实时对战 |
| 13 | **VoiceMemo 语音** | `handlers/voicememo.go` | `routes/voicememo.go` | `views/life/VoiceInboxTool.vue` | 语音录制/转写（Whisper），AI 总结，自动创建待办 |
| 14 | **JSON 工具** | `handlers/json.go` | `routes/paste.go` | `views/dev/JsonTool.vue` | JSON 格式化/压缩/校验，转 Go Struct / TypeScript Interface |
| 15 | **Diff 对比** | `handlers/diff.go` | `routes/paste.go` | `views/dev/DiffTool.vue` | 文本对比，行级/词级/字符级差异高亮 |
| 16 | **Regex 正则** | `handlers/regex.go` | `routes/paste.go` | `views/dev/RegexTool.vue` | 正则实时匹配，常用模板 |
| 17 | **Pregnancy 孕期** | `handlers/pregnancy.go` | `routes/pregnancy.go` | `views/life/PregnancyTool.vue` | 孕周计算，产检提醒，宝宝发育参考 |
| 18 | **Expense 记账** | `handlers/expense.go` | `routes/expense.go` | `views/life/ExpenseTool.vue` | 收支记录，分类统计，趋势分析，AI 总结 |
| 19 | **Glucose 血糖** | `handlers/glucose.go` | `routes/glucose.go` | `views/life/GlucoseTool.vue` | 血糖记录，达标率，趋势图表 |
| 20 | **Planner 待办** | `handlers/planner*.go` (6 文件) | `routes/planner.go` | `views/life/PlannerTool.vue` | 工作/生活分离，时间线，专注模式，AI 建议，会议纪要 |
| 21 | **Recipe 菜谱** | `handlers/recipe.go` | `routes/recipe.go` | `views/life/RecipeTool.vue` | 每日菜谱，分类浏览，收藏 |
| 22 | **Household 家庭** | `handlers/household.go` (7 分段) | `routes/household.go` | `views/life/HouseholdTool.vue` | 物品/位置/模板管理，通知，AI 智能添加/分析，OCR |
| 23 | **NFSShare** | `handlers/nfsshare*.go` (4 文件) | `routes/nfsshare.go` | `views/collab/NFSShareTool.vue` | SMB/NFS 文件共享，WebRTC 流媒体，HLS 分片 |
| 24 | **ImageUnderstanding** | `handlers/image_understanding.go` | `routes/image_understanding.go` | `views/ai/ImageUnderstandingTool.vue` | 图像理解（VLM），Base64/URL 输入，返回描述 |
| 25 | **AIGateway** | `handlers/ai_gateway*.go` (13 文件，含 1 个测试脚手架) | `routes/ai_gateway.go` | `views/ai/AIGatewayTool.vue` | 统一 AI 网关：OpenAI/Anthropic/MiniMax/DashScope，Token 计划，费用分析；`EnglishTutor` 端点 (`handlers/english_tutor.go`) 归属此网关的服务端受限代理 |
| 26 | **Askit 同步** | `handlers/askit*.go` (3 文件) | `routes/askit.go` | `views/other/AskitInviteTool.vue` | 跨设备数据同步，邀请码，Blob 大文件，快照 |
| 27 | **Screen 屏幕共享** | `handlers/screen.go` | `routes/screen.go` | `views/collab/ScreenShareTool.vue` | WebRTC 点对点屏幕共享，TURN 中继 |
| 28 | **APIGateway** | `handlers/apigateway.go` | `routes/apigateway.go` | — | API 网关路由（代理） |
| 29 | **AutoDev** | `handlers/autodev.go` (6 分段) | `routes/autodev.go` | `views/dev/AutoDevTool.vue` | Claude Code / Codex CLI 集成，Ask/Extend/Submit/List |
| 30 | **Mermaid** | `handlers/mermaid.go` | `routes/mermaid.go` | `views/draw/MermaidTool.vue` | Mermaid 图表，实时渲染，导出 SVG/PNG |
| 31 | **Proxy 代理** | `handlers/proxy.go` (4 分段) | `routes/proxy.go` | `views/other/ProxyTool.vue` | sing-box 嵌入式代理，节点订阅/测速/选优，NPS 隧道 |
| 32 | **NPS NPS** | `handlers/nps.go` | `routes/nps.go` | `views/other/NPSTool.vue` | NPS 内网穿透管理 |
| 33 | **Hermes** | `handlers/hermes.go` | `routes/hermes.go` | `views/dev/HermesTool.vue` | Hermes 工具 |
| 34 | **Background 背景图** | `handlers/background.go` | `routes/bg.go` | `views/other/BackgroundTool.vue` | 背景图库，本地缓存，分页浏览 |
| 35 | **Monitor 监控** | `handlers/monitoring.go` | `routes/monitor.go` | `views/other/MonitorTool.vue` | AI Gateway 使用监控，费用分析，实时日志 |
| 36 | **Console 控制台** | `handlers/console.go` | `routes/console.go` | `views/other/ConsoleTool.vue` | 管理控制台 |
| 37 | **Skills+MCP** | `handlers/skills*.go` (1 文件) | `routes/skills.go` | `views/other/SkillsTool.vue` | OpenAI function-calling 风格清单，MCP 协议，7 个 skill |
| 38 | **OCR** | `handlers/ocr.go` | `routes/ocr.go` | — | 二维码识别 + OCR，Python RapidOCR |
| 39 | **Health** | 内联 | `routes/health.go` | — | `GET /api/health` 健康检查 |
| 40 | **Avatar 虚拟形象** | `handlers/avatar.go` | `routes/avatar.go` | `views/dev/VirtualAvatarTool/VirtualAvatarTool.vue` (+ 13 子组件) | three.js 虚拟形象调试：SkinnedMesh + Skeleton + 姿态预设；模型库 CRUD + me_asset 标记；姿态片段（pose JSON）上传/导出/分享；glb / pose.json / 视频三选一导出（MediaRecorder + WebCodecs → 降级关键帧 JSON → 复制）；摄像头调试（getUserMedia 四态降级：granted/denied/unsupported/no_device）；R5 双因子鉴权（creator_key + bcrypt password）；R10 帧数上限 60000；分享短链 16 字符 hex；CSS 变量跟随全局主题 (`--color-accent` 全局命名） |
| 41 | **Trip 旅游行程** | `backend/trip/*.go` + `backend/handlers/trip.go` + `backend/cmd/trip/main.go` | `routes/trip.go` | `views/life/TripTool.vue` | DDD bounded context：聚合根 `Trip` → 实体 `DayPlan` → 实体 `Activity`；值对象 `Destination` + `ActivityKind`(transport/sight/food/lodging/shopping/note)；仓储 `Repository` 接口 + SQLiteRepository；服务 `Service.RenderMarkdown` 输出可读 14 天攻略；HTTP 端点 `/api/trips/*`（列表/详情/Markdown/Summary、增删改行程·日程·活动、预算、费用账本、提醒、`seed`/`process-now`）；CLI `trip create/list/show/delete/add-day/add-activity/seed/init`；第一个真实行程 fixture = 国庆 2026 粤港澳潮汕 14 日深度游（已 seed）；默认 SQLite `./data/trip.db`（`--db` 覆盖） |
| 42 | **SSRules 规则文件** | `handlers/ssrule.go` + `handlers/ssrule_rules.go` | `routes/ssrule.go` | `views/other/SsRuleTool.vue` | Shadowsocks ACL 规则文件生成与公开分发：管理端（`/api/ssrules/*`）走 header `X-Super-Admin-Password` 密码门（未配密码 = 模块关闭）；域名列表 → ACL 正文（`(^\|\.)example\.com$` 正则形式，libev/rust/android 通吃）；32 位随机 token 公开链接 `GET /ss/:token/:filename` 分发；按端点分桶限流（verify/换发 10、读 30、写 20、分发 120 次/分钟/IP，独立计数桶）；`:filename` 白名单精确相等 + 全程不碰文件系统（路径穿越面为零）；未匹配 `/ss/` 一律 404「规则文件不存在」，不回落 SPA |
| 43 | **RolePlay 角色剧场** | —（零后端，复用 `/api/chat-plus/*`） | — | `views/ai/RolePlayTool.vue` | 多角色模拟对话：2-6 个角色（名字/人设/颜色）围绕导演话题轮流圆桌发言；前端驱动循环——每条发言一次独立 `/api/chat-plus/chat/stream` 请求（同一 `X-Chat-Password` 密码门），网络故障只损失单条；轮数 1-200（1 轮 = 每角色各说一次），上下文滑动窗口（最近 30 条）+ 单条 max_tokens 700 防 token 爆炸；暂停（本条说完）/跳过/停止、失败自动重试 1 次后进暂停态可续演、localStorage 持久化剧本与记录、4 个内置剧本模板、导出 Markdown |
| 44 | **News 每日新闻** | `handlers/news.go` | `routes/news.go` | `views/life/NewsTool.vue` | 每日新闻模块：RSS/Atom 订阅源抓取 → HTML 标签剥离 → 按内容哈希去重 → 可选 AI 精编 → 存库归档；默认 6 个内置公开订阅源（可用 `config.yaml` 的 `news.feeds` 覆盖）；`/api/news/latest`、`/api/news/list`（日期索引）、`/api/news/config`、`/api/news/:date`（按日取稿）、`POST /api/news/generate`（生成当日）、`POST /api/news/publish`（智能体采编后发布/归档） |
| 45 | **Chat+ 新版聊天页** | `handlers/chat_plus.go` | `routes/ai_gateway.go`（`/api/chat-plus/*` 注册在此文件） | `views/ai/AIChatTool.vue` | Open WebUI 风格新版 AI 聊天页：密码门（密码存服务端 `config.yaml` 的 `chat_plus.password`，为空 = 模块整体禁用 fail-closed；`models`/`stream` 走 header `X-Chat-Password`，`verify` 走 body，常量时间比较）；双引擎——局域网 Ollama（OpenAI 兼容 `/v1/chat/completions`，模型列表运行时从 `/api/tags` 动态拉取）+ MiniMax（复用 AI Gateway 的 key，key 全程不出服务端）；SSE 流式 `/api/chat-plus/chat/stream` |

---

## 技术栈

- **前端**：Vue 3 + Vite + Element Plus + TailwindCSS（88 个 View 组件 + 12 个公共组件）
- **后端**：Go (Gin) + SQLite + **Redis（软依赖，失败自动降级到内存存储）**
- **辅助服务**：
  - `ocr-service`：Python FastAPI + RapidOCR（二维码识别/OCR，端口 8000）
  - `asr-service`：Python FastAPI + faster-whisper（语音识别，端口 9000）
  - `tts-service`：Python FastAPI + edge-tts（语音合成，端口 8083，容器内）
  - `diarize-service`（独立）：说话人识别（CUDA GPU 加速，可选）
- **AI**：Anthropic Claude（via MiniMax 代理）、OpenAI、MiniMax、DashScope、sing-box
- **部署**：Docker Compose（4 服务：devtools / redis / ocr-service / asr-service）

---

## 开发命令

### 前端
```bash
cd frontend
pnpm install         # 锁文件 pnpm-lock.yaml(Dockerfile / CI 都用 pnpm)
pnpm dev             # http://localhost:5173
pnpm build
```

### 后端
```bash
cd backend
go mod tidy
go run main.go      # :8080（开发），:8082（Docker）
```

### Docker
```bash
docker compose up -d
docker compose logs -f devtools
```

---

## 项目结构

### 前端（88 View 组件）
```
frontend/src/views/
├── ai/         # AI 相关（7）：AIChatTool, AIGatewayTool, EnglishTutorTool, ImageUnderstandingTool, MiniMaxStudioTool, PetTool, RolePlayTool
├── collab/     # 协作（6）：ChatRoom, NFSShareTool, PasteBin, ScreenShare, ScreenView, ShortUrl, ...
├── convert/    # 转换（7）：Base64, Dns, QrCode, Replace, Text, Timestamp, Url
├── dev/        # 开发（22）：AutoDevTool, DiffTool, EdgeTTSTool, HermesTool, JsonTool, MockApi, RegexTool, TerminalTool, VirtualAvatarTool, 另 13 个
├── draw/       # 绘图（4）：Excalidraw, Markdown, Mermaid, MindMap
├── home/       # 首页
├── household/  # 家庭（10 个子组件）
├── life/       # 生活（13）：CounterTool, ExpenseTool, GameHall, GlucoseTool, HouseholdSpace3D, HouseholdTool, NewsTool, PhotoWallTool, PlannerTool, PregnancyTool, RecipeTool, TripTool, VoiceInboxTool
├── other/      # 其他（10）：AskitInviteTool, AskitMyDataTool, BackgroundTool, ConsoleTool, ImageViewerTool, MonitorTool, NPSTool, ProxyTool, SkillsTool, SsRuleTool
└── share/      # 分享视图（8）
```

### 后端路由拆分（routes/ 43 个文件）
```
backend/routes/
├── index.go              # RouteHandlers struct + RegisterAllRoutes（入口）
├── paste.go / analysis.go / dns_image.go / chat.go / edge_tts.go / game.go
├── voicememo.go / shorturl.go / mockapi.go / mdshare.go / excalidraw.go
├── pregnancy.go / expense.go / glucose.go / planner.go / recipe.go / household.go
├── photowall.go / terminal.go / nfsshare.go / image_understanding.go
├── ai_gateway.go / askit.go / screen.go / apigateway.go / autodev.go / mermaid.go
├── proxy.go / nps.go / hermes.go / bg.go / monitor.go / console.go / skills.go
├── ocr.go / health.go / avatar.go
├── news.go / trip.go / ssrule.go
└── ssrule_test.go / ssrule_qa_test.go    # ssrule 路由测试
```

### Handler 文件分组
```
handlers/
├── 核心工具（8）：paste, shorturl, mockapi, mdshare, excalidraw, photowall, terminal, dns
├── AI 网关（19）：ai_gateway*（13，含 1 个 test helper）, image_understanding, minimax_*（4）, planner_ai
├── 订阅/代理（5）：proxy*, singbox*, gfwlist, nps, apigateway_proxy
├── 家庭/健康（5）：household, pregnancy, expense, glucose, recipe
├── 实时功能（4）：chat, voicememo, screen, nfsshare
├── 内容创作（3）：mermaid, game, hermes
├── 开发者（3）：autodev, analysis, console
├── 辅助（7）：background, monitoring, ocr, edge_tts, skills*, askit*, fileutils
└── 测试（52）：*_test.go
```

---

## API 快速索引

### 工具类
- `POST /api/paste` — 创建粘贴
- `POST /api/shorturl` — 创建短链
- `POST /api/mockapi` — 创建 Mock API
- `ANY /mock/:id` — 执行 Mock API
- `POST /api/mdshare` — 创建 Markdown 分享
- `POST /api/excalidraw` — 创建 Excalidraw 画图
- `GET /api/dns?domain=` — DNS 查询
- `GET /api/ip` — 客户端 IP

### 虚拟形象（avatar module, P1+P2）
- `POST /api/avatar/models` — 上传模型（.glb / .gltf，多部分，R5 password 可选）
- `GET /api/avatar/models?owner_id=&limit=&offset=` — 列出模型（owner_id 为空 = 全库；带值 = 我的）
- `GET /api/avatar/models/:id` — 模型详情
- `GET /api/avatar/models/:id/file` — 模型文件二进制（GLTFLoader 直接消费）
- `DELETE /api/avatar/models/:id?owner_id=&password=` — 删除模型（R1+R5）
- `POST /api/avatar/me/assets` — 把模型加入我的资产（去重 upsert）
- `GET /api/avatar/me/assets` — 我的资产列表
- `DELETE /api/avatar/me/assets/:id` — 移除我的资产
- `POST /api/avatar/clips` — 上传姿态片段（JSON，R10 ≤ 60000 帧）
- `GET /api/avatar/clips/:id?password=` — 取姿态 JSON（密码保护）
- `DELETE /api/avatar/clips/:id?owner_id=&password=` — 删除片段
- `POST /api/avatar/share` — 创建分享短链（target_type=model|clip, expires_in_days）
- `GET /api/avatar/share/:code?password=` — 按 code 取元数据（命中后异步 +1 hit_count，goroutine defer recover 兜底）
- `DELETE /api/avatar/share/:code?owner_id=` — 创建者删除分享

### 档案/社交
- `POST /api/chat/room` — 创建聊天室（+ WebSocket `/api/chat/room/:id/ws`）
- `POST /api/photowall/profile` — 创建照片墙档案
- `POST /api/terminal` — 创建 SSH 会话（+ WebSocket `/api/terminal/:id/ws`）
- `POST /api/skills/invoke` — 调用 skill（JSON-RPC）

### AI
- `POST /api/ai-gateway/v1/chat/completions` — AI 对话
- `POST /api/ai-gateway/v1/image/understanding` — 图像理解
- `POST /api/image-understanding/describe` — 图像描述
- `POST /api/edge-tts` — 语音合成
- `GET /api/skills/manifest` — Skills 清单（OpenAI function-calling 格式）
- `GET /api/skills/mcp` — MCP 发现端点
- `POST /api/minimax/h3/v2/video_generation` — MiniMax-H3 异步文/图/多模态生视频（立即返回 task_id）
- `GET /api/minimax/h3/v2/query/video_generation` — H3 任务列表（支持 filter.status/model/task_type）
- `GET /api/minimax/h3/v2/query/video_generation/:task_id` — H3 单个任务状态查询
- `DELETE /api/minimax/h3/v2/video_generation/:task_id` — H3 任务取消/删除

### 代理
- `POST /api/proxy/trigger-refresh` — 触发订阅刷新
- `GET /api/proxy/nodes` — 节点列表
- `POST /api/proxy/start` — 启动代理
- `POST /api/proxy/stop` — 停止代理
- `POST /api/proxy/speed-test` — 节点测速

### 健康
- `GET /api/health` — 健康检查

---

## 关键设计模式

### 前端
- **动态菜单**：routes 自动生成侧边栏（`meta.title` + `meta.icon`）
- **代码分割**：所有路由懒加载
- **响应式**：移动端 Drawer 导航

### 后端
- **Redis 软依赖**：Redis 不可用时自动降级到内存存储（`state.New()` → `NewMemoryStore()`）
- **路由拆分**：`routes.go` → `routes/` 包（43 文件，40 分组）
- **自动清理**：每小时清理过期数据（pastes/shorturls/mdshares/excalidraws/photowalls/terminals）
- **静态优先**：未匹配路由返回 `index.html`（SPA 支持）
- **配置**：YAML 配置文件（`config.yaml`）

---

## 添加新工具流程

1. 创建 View 组件：`frontend/src/views/category/YourTool.vue`
2. 添加路由：`frontend/src/router/index.js`（设置 `meta.title` 和 `meta.icon`）
3. 创建 Handler：`backend/handlers/yourtool.go`
4. 添加路由文件：`backend/routes/yourtool.go`（注册 `RegisterYourToolRoutes`）
5. 在 `backend/routes/index.go` 的 `RegisterAllRoutes` 中添加调用
6. 更新本文件 `45 模块地图` 表格

---

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PORT` | 8080 | 后端端口 |
| `DB_PATH` | `./data/paste.db` | SQLite 路径 |
| `CONFIG_PATH` | `./config.yaml` | 配置文件 |
| `OCR_SERVICE_URL` | `http://ocr-service:8000` | OCR 服务 |
| `ASR_SERVICE_URL` | `http://asr-service:9000` | ASR 服务 |
| `DIARIZE_SERVICE_URL` | — | 说话人识别（可选） |
| `TTS_SERVICE_URL` | `http://127.0.0.1:8083` | TTS 服务 |
| `MINIMAX_API_KEY` | — | MiniMax API（AI 机器人） |
| `DEEPSEEK_API_KEY` | — | DeepSeek API（记账 AI） |
| `TERMINAL_ENCRYPTION_KEY` | 随机生成（重启后丢失） | SSH 终端密码加密主密钥（不设置则用随机密钥,warning） |
| `DEPLOY_MASTER_KEY` | **必填,缺失则 fatal** | AI Gateway AnthropicProvider API Key AES-256-GCM 加密主密钥（与 SSH 独立） |
| `GOPROXY` | `https://goproxy.cn,direct` | Go 代理（国内） |

---

## Module Path

Go 后端使用模块路径 `devtools`：
```go
import (
    "devtools/handlers"
    "devtools/middleware"
    "devtools/models"
    "devtools/routes"
)
```
