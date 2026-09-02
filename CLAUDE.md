# CLAUDE.md

DevTools 项目快速参考。详细架构和 API 文档见 `AGENTS.md`（34 模块地图、完整 handler/route 索引、详细 API 索引）。

---

## 概览

- **前端**：Vue 3 + Vite + Element Plus + TailwindCSS（72 个 View 组件）
- **后端**：Go (Gin) + SQLite + **Redis（软依赖，失败自动降级内存存储）**
- **辅助服务**：ocr-service (8000), asr-service (9000), tts-service (8083), diarize-service (9100, 可选)
- **部署**：Docker Compose（4 服务）

---

## 开发命令

```bash
# 前端
cd frontend && npm install && npm run dev   # :5173

# 后端
cd backend && go mod tidy && go run main.go   # :8080

# Docker
docker compose up -d && docker compose logs -f devtools
```

---

## 项目结构速查

```
frontend/src/views/       # 72 Vue 组件，按 category/ 分组
backend/
├── main.go              # 入口：DB 初始化、中间件、每小时清理 goroutine
├── routes/              # 38 个路由文件（routes.go 已拆分）
│   └── index.go         # RouteHandlers struct + RegisterAllRoutes
├── handlers/            # ~95 个 handler 文件（按域分组）
├── models/              # SQLite 操作（自动建表）
├── middleware/          # RateLimiter、ContentSizeLimiter 等
└── state/               # Redis/内存状态（Redis 软依赖）
```

---

## 关键设计模式

| 模式 | 说明 |
|------|------|
| **Redis 软依赖** | `state.New()` → Redis 失败返回 `MemoryStore`，进程不退出 |
| **路由拆分** | `routes.go` → `routes/` 包（38 文件，34 分组） |
| **自动清理** | 每小时清理过期数据（pastes/shorturls/mdshares/excalidraws/photowalls/terminals） |
| **SPA fallback** | 未匹配路由返回 `index.html` |
| **配置** | YAML（`config.yaml`），环境变量覆盖 |
| **Handler 分组** | 大文件分段注释（proxy/household/autodev 等） |

---

## 添加新工具

1. `frontend/src/views/category/YourTool.vue`
2. `frontend/src/router/index.js`（`meta.title` + `meta.icon`）
3. `backend/handlers/yourtool.go`
4. `backend/routes/yourtool.go`（`RegisterYourToolRoutes`）
5. `backend/routes/index.go` → `RegisterAllRoutes` 中添加调用
6. 更新 `AGENTS.md` 34 模块地图

---

## 环境变量

| 变量 | 默认 | 说明 |
|------|------|------|
| `PORT` | 8080 | 端口 |
| `DB_PATH` | `./data/paste.db` | SQLite |
| `CONFIG_PATH` | `./config.yaml` | 配置文件 |
| `OCR_SERVICE_URL` | `http://ocr-service:8000` | |
| `ASR_SERVICE_URL` | `http://asr-service:9000` | |
| `TTS_SERVICE_URL` | `http://127.0.0.1:8083` | |
| `DIARIZE_SERVICE_URL` | — | 说话人识别（可选） |
| `MINIMAX_API_KEY` | — | AI 机器人 |
| `DEEPSEEK_API_KEY` | — | 记账 AI |
| `TERMINAL_ENCRYPTION_KEY` | 随机(warning) | SSH 终端密码加密 |
| `DEPLOY_MASTER_KEY` | **必填,fatal** | AI Gateway API Key AES-GCM 加密 |
| `GOPROXY` | `https://goproxy.cn,direct` | Go 代理（国内） |

---

## Module Path

```go
import (
    "devtools/handlers"
    "devtools/middleware"
    "devtools/models"
    "devtools/routes"
)
```
