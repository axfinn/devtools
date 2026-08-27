# AGENTS.md

本文件是 DevTools 项目的唯一权威文档。供 AI Agent（Claude Code / Codex）使用。所有其他文档为辅助参考，以本文件为准。

---

## 34 模块地图

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
| 23 | **NFSShare** | `handlers/nfsshare*.go` (5 文件) | `routes/nfsshare.go` | `views/collab/NFSShareTool.vue` | SMB/NFS 文件共享，WebRTC 流媒体，HLS 分片 |
| 24 | **ImageUnderstanding** | `handlers/image_understanding.go` | `routes/image_understanding.go` | `views/ai/ImageUnderstandingTool.vue` | 图像理解（VLM），Base64/URL 输入，返回描述 |
| 25 | **Bailian 百炼** | `handlers/bailian.go` | `routes/bailian.go` | — | 阿里百炼图像理解 API 封装 |
| 26 | **AIGateway** | `handlers/ai_gateway*.go` (10 文件) | `routes/ai_gateway.go` | `views/ai/AIGatewayTool.vue` | 统一 AI 网关：OpenAI/Anthropic/MiniMax/百炼，Token 计划，费用分析 |
| 27 | **Askit 同步** | `handlers/askit*.go` (3 文件) | `routes/askit.go` | `views/other/AskitInviteTool.vue` | 跨设备数据同步，邀请码，Blob 大文件，快照 |
| 28 | **Screen 屏幕共享** | `handlers/screen.go` | `routes/screen.go` | `views/collab/ScreenShareTool.vue` | WebRTC 点对点屏幕共享，TURN 中继 |
| 29 | **APIGateway** | `handlers/apigateway.go` | `routes/apigateway.go` | — | API 网关路由（代理） |
| 30 | **AutoDev** | `handlers/autodev.go` (6 分段) | `routes/autodev.go` | `views/dev/AutoDevTool.vue` | Claude Code / Codex CLI 集成，Ask/Extend/Submit/List |
| 31 | **Mermaid** | `handlers/mermaid.go` | `routes/mermaid.go` | `views/draw/MermaidTool.vue` | Mermaid 图表，实时渲染，导出 SVG/PNG |
| 32 | **Proxy 代理** | `handlers/proxy.go` (4 分段) | `routes/proxy.go` | `views/other/ProxyTool.vue` | sing-box 嵌入式代理，节点订阅/测速/选优，NPS 隧道 |
| 33 | **NPS NPS** | `handlers/nps.go` | `routes/nps.go` | `views/other/NPSTool.vue` | NPS 内网穿透管理 |
| 34 | **Hermes** | `handlers/hermes.go` | `routes/hermes.go` | `views/dev/HermesTool.vue` | Hermes 工具 |
| 35 | **Background 背景图** | `handlers/background.go` | `routes/bg.go` | `views/other/BackgroundTool.vue` | 背景图库，本地缓存，分页浏览 |
| 36 | **Monitor 监控** | `handlers/monitoring.go` | `routes/monitor.go` | `views/other/MonitorTool.vue` | AI Gateway 使用监控，费用分析，实时日志 |
| 37 | **Console 控制台** | `handlers/console.go` | `routes/console.go` | `views/other/ConsoleTool.vue` | 管理控制台 |
| 38 | **Skills+MCP** | `handlers/skills*.go` (3 文件) | `routes/skills.go` | `views/other/SkillsTool.vue` | OpenAI function-calling 风格清单，MCP 协议，5 个 skill |
| 39 | **OCR** | `handlers/ocr.go` | `routes/ocr.go` | — | 二维码识别 + OCR，Python RapidOCR |
| 40 | **Health** | 内联 | `routes/health.go` | — | `GET /api/health` 健康检查 |

---

## 技术栈

- **前端**：Vue 3 + Vite + Element Plus + TailwindCSS（72 个 View 组件）
- **后端**：Go (Gin) + SQLite + **Redis（软依赖，失败自动降级到内存存储）**
- **辅助服务**：
  - `ocr-service`：Python FastAPI + RapidOCR（二维码识别/OCR，端口 8000）
  - `asr-service`：Python FastAPI + faster-whisper（语音识别，端口 9000）
  - `tts-service`：Python FastAPI + edge-tts（语音合成，端口 8083，容器内）
  - `diarize-service`（独立）：说话人识别（CUDA GPU 加速，可选）
- **AI**：Anthropic Claude（via MiniMax 代理）、OpenAI、MiniMax、百炼、sing-box
- **部署**：Docker Compose（4 服务：devtools / redis / ocr-service / asr-service）

---

## 开发命令

### 前端
```bash
cd frontend
npm install
npm run dev          # http://localhost:5173
npm run build
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

### 前端（72 View 组件）
```
frontend/src/views/
├── ai/         # AI 相关（6）：AIGatewayTool, AIChatTool, BailianImage, EnglishTutor, ImageUnderstanding, MiniMaxStudio
├── collab/     # 协作（8）：ChatRoom, NFSShareTool, PasteBin, ScreenShare, ScreenView, ShortUrl, ...
├── convert/    # 转换（8）：Base64, Dns, QrCode, Replace, Text, Timestamp, Url
├── dev/        # 开发（9）：AutoDev, Diff, EdgeTTS, Hermes, Json, MockApi, Regex, Terminal
├── draw/       # 绘图（4）：Excalidraw, Markdown, Mermaid, MindMap
├── home/       # 首页
├── household/  # 家庭（9 个子组件）
├── life/       # 生活（11）：Counter, Expense, GameHall, Glucose, HouseholdSpace3D, HouseholdTool, PhotoWall, Planner, Pregnancy, Recipe, VoiceInbox
├── other/      # 其他（9）：Askit, Background, Console, ImageViewer, Monitor, Neon, NPS, Proxy, Skills, VibeMotion
└── share/      # 分享视图（9）
```

### 后端路由拆分（routes/ 38 个文件）
```
backend/routes/
├── index.go              # RouteHandlers struct + RegisterAllRoutes（入口）
├── paste.go / analysis.go / dns_image.go / chat.go / edge_tts.go / game.go
├── voicememo.go / shorturl.go / mockapi.go / mdshare.go / excalidraw.go
├── pregnancy.go / expense.go / glucose.go / planner.go / recipe.go / household.go
├── photowall.go / terminal.go / nfsshare.go / image_understanding.go / bailian.go
├── ai_gateway.go / askit.go / screen.go / apigateway.go / autodev.go / mermaid.go
├── proxy.go / nps.go / hermes.go / bg.go / monitor.go / console.go / skills.go
├── ocr.go / health.go
```

### Handler 文件分组
```
handlers/
├── 核心工具（8）：paste, shorturl, mockapi, mdshare, excalidraw, photowall, terminal, dns
├── AI 网关（14）：ai_gateway*, bailian, image_understanding, minimax_*, planner_ai
├── 订阅/代理（5）：proxy*, singbox*, gfwlist, nps, apigateway_proxy
├── 家庭/健康（5）：household, pregnancy, expense, glucose, recipe
├── 实时功能（4）：chat, voicememo, screen, nfsshare
├── 内容创作（3）：mermaid, game, hermes
├── 开发者（3）：autodev, analysis, console
├── 辅助（7）：background, monitoring, ocr, edge_tts, skills*, askit*, fileutils, security_enhance, content_parser
└── 测试（20+）：*_test.go
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

### 档案/社交
- `POST /api/chat/room` — 创建聊天室（+ WebSocket `/api/chat/room/:id/ws`）
- `POST /api/photowall/profile` — 创建照片墙档案
- `POST /api/terminal` — 创建 SSH 会话（+ WebSocket `/api/terminal/:id/ws`）
- `POST /api/skills/invoke` — 调用 skill（JSON-RPC）

### AI
- `POST /api/ai-gateway/chat` — AI 对话
- `POST /api/ai-gateway/image` — 图像理解
- `POST /api/image-understanding` — 图像描述
- `POST /api/bailian/image` — 百炼图像
- `POST /api/edge-tts` — 语音合成
- `GET /api/skills/manifest` — Skills 清单（OpenAI function-calling 格式）
- `GET /api/skills/mcp` — MCP 发现端点

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
- **路由拆分**：`routes.go` → `routes/` 包（38 文件，34 分组）
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
6. 更新本文件 `34 模块地图` 表格

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
