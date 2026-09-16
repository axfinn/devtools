# /api/health 真实依赖探测 + 监控台可视化 — 技术方案

> **状态**：待评审（Stage 2 = FINN-4 devtools-design-reviewer）
> **作者**：devtools-architect · 2026-09-16
> **上游**：FINN-2「健康检查增强」（parent issue `01a0a840-e468-76b4-81ae-6cdcd68f6a06`）
> **基线**：FINN-3 原稿 commit `53303cf74eca1777b3229f103732ff4194bb0bce`
> **本次修订**：FINN-15（追加要求 1 = 探测地址矩阵；追加要求 2 = 详细模式公网暴露面）
> **勘察方式**：本文所有 `file:line` 均为本次实际 grep / 阅读确认，未采用需求描述里的任何未经核实的路径

---

## 目标与非目标

### 目标

1. `GET /api/health` 增加**详细形态**，返回版本标识、uptime、以及 5 个依赖（SQLite / Redis / OCR / ASR / TTS）的实时 up / down / warming / disabled 状态与探测耗时。
2. **默认形态严格不变**：`GET /api/health`（无参数）继续返回 `200 {"status":"ok"}`，字段一个不多一个不少，且**永不返回非 200**。
3. 监控台 `views/other/MonitorTool.vue` 一处可视化：一眼看到各依赖当前是否可用、上次探测时间、版本号。
4. 新增单测覆盖：默认形态向后兼容、**鉴权三态（未授权/已授权/未配置密码）**、**响应体泄露断言**、正常路径、Redis 降级路径、依赖探测超时路径、SQLite 故障路径。
5. **（FINN-15 追加）** 给出**逐依赖的探测地址来源矩阵**，明确每种默认值只在哪种运行形态下成立，并让本地开发形态的「假 down」可被一眼解释。
6. **（FINN-15 追加）** 给出**详细模式的公网暴露面结论**：鉴权方式、字段三档裁剪、防缓存要求，并保证默认形态**零可侦察字段**。

### 非目标（明确不做）

- ❌ 不告警、不推送通知、不发邮件。
- ❌ 不做 Prometheus / metrics 导出、不做历史趋势存储（依赖状态不做时序落库）。
- ❌ **不改 `docker-compose.yml` 的 healthcheck、不改 `deploy.sh` 的探活方式**（`deploy.sh:212-227 wait_for_health` 与 `docker-compose.yml:47-52` 原样保留）。
- ❌ **不改 Nginx / Cloudflare 配置，不新增第二个监听端口**（理由见「设计 1.6」——这是「只在直连端口暴露」方案被否掉的直接原因）。
- ❌ 不做 release 管理 / 语义化版本台账，只做「构建期注入一个字符串 + 三级 fallback」。
- ❌ 不改 `AGENTS.md` 模块地图（由 module-architect 负责；本方案只标注需要更新的行号供其参考）。
- ❌ 不新增前端路由（`/monitor` 已存在，见「前端改动清单」）。
- ❌ 不改 `.env.example`（本地开发提示见「设计 1.5.4」，属 R1 之外的写权，列为未决问题）。

---

## 现状

### 被改的端点本体

```go
// backend/routes/health.go:7-11  —— 全部内容就是这 11 行，内联匿名 handler，无 Handler 结构体
func RegisterHealthRoute(api *gin.RouterGroup) {
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}
```

接入点（已 grep 确认唯一，`RegisterHealthRoute` 全仓只出现 2 次：定义 + 调用）：

- `backend/routes/index.go:88` → `RegisterHealthRoute(api)`（**无 h 参数**，这是要改的签名）
- `backend/routes/index.go:12-49` → `RouteHandlers` struct（跨包穿的 handler 集合，注释说明不能 import main 以避免循环依赖）
- `backend/routes.go:12-49` → main 包内 `routeHandlers` struct；`backend/routes.go:51-90` → `setupRoutes` 里逐字段搬运到 `routes.RouteHandlers`（`routes.go:53` 开始字面量）
- `backend/app_handlers.go:16-137` → `buildRouteHandlers` 构造全部 handler，`:99` 起 `return &routeHandlers{...}`

### 依赖的真实可达性（决定探测方式）

| 依赖 | 探测端点 | 源码证据 |
|---|---|---|
| OCR (8000) | `GET /health`，**engine 未就绪返回 503 `{"detail":"warming up"}`** | `ocr-service/main.py:153-158` |
| ASR (9000) | `GET /health`，**whisper_model 未就绪返回 503**，就绪返回 `{"status":"ok", "model":..., "diarize_service_url":...}` | `asr-service/main.py:150-163` |
| TTS (8083) | `GET /health` → `{"status":"ok","edge_tts":bool}` | `tts-service/server.py:88-90` |
| SQLite | 进程内，`models.DB.Conn() *sql.DB` 可 `PingContext` | `backend/models/paste.go:37-39`、`:235`、`:241` |
| Redis | 进程内 client，`state` 包已封装 | `backend/state/transient.go:71-104` |

> ⚠️ `asr-service/main.py:162` 的 `/health` **原样回显 `DIARIZE_SERVICE_URL`**（内网地址）。本方案**不解析任何辅助服务的响应体**——这既是为了不与它们的响应结构耦合，更是为了不把这类内网地址捞进我们自己的响应（见「设计 1.6.3 C 档」）。

**关键事实 1 — TTS 不在 docker-compose 里，而是 devtools 容器内的 sidecar**：`docker-compose.yml` 只有 `devtools`(:2) / `redis`(:54) / `ocr-service`(:68) / `asr-service`(:96) 四个服务，**没有 tts-service**；TTS 由 `entrypoint.sh:153-155` 在 **devtools 容器内**拉起 `127.0.0.1:8083`，且 `Dockerfile:106` 把 `tts-service/server.py` COPY 成镜像内的 `/app/tts_server.py`。因此默认地址 `http://127.0.0.1:8083`（`backend/config/config.go:731-736`）**在容器形态下是正确的**，不是 bug。**这是"一律用服务名"的反例**（见「设计 1.5」）。

**关键事实 2 — 探活端点有 3 秒硬上限**：`docker-compose.yml:48-50`

```yaml
test: ["CMD-SHELL", "http_proxy= ... wget --no-verbose --tries=1 --spider http://localhost:8082/api/health"]
interval: 30s
timeout: 3s        # ← 总预算的硬约束
retries: 3
start_period: 10s
```

`deploy.sh` 侧的 `wait_for_health` 用 `curl -fsS`（`deploy.sh:212-227`），只看 HTTP 状态码，调用点 `deploy.sh:469 / 497 / 578 / 599 / 621 / 638`。**已逐行确认这 6 个调用点用的都是不带 query 的 `/api/health`**，绝不带 `detail=1`。

**关键事实 3 — `/api/health` 已在监控跳过名单里**：`backend/middleware/request_logger.go:73-82` 的 `shouldSkipMonitoringRequest` 前缀表包含 `"/api/health"` → 前端轮询详细形态**不产生 `http_request_logs` 行**，无写放大。

**关键事实 4 — Redis 是软依赖且当前状态不可观测**：`state.New()`（`backend/state/transient.go:71-104`）在 `Ping` 失败时 `log.Printf` 后返回 `MemoryStore`，**进程不退出**；但 `TransientStore` 接口（`:40-49`）**没有任何方法能区分当前到底跑在 Redis 还是内存降级**。这是本次必须补的能力。

### 公网暴露面现状（FINN-15 追加勘察）

| 事实 | 证据 |
|---|---|
| 对外入口是 Cloudflare 前置的 `https://t.jaxiu.cn`；`/api` 经 Nginx 反代到 `devtools:8082` | 项目上下文（jaxiu 2026-09-16 确认）；**Nginx 配置不在本仓库**（`find . -iname "*nginx*" -not -path "*/node_modules/*"` 零命中） |
| 仓库里能改的部署面只有 `docker-compose.yml` 与 `deploy.sh`，而本需求**明确禁止改它们** | 本文「非目标」；`docker-compose.yml:16` 只 publish 一个端口 `"${HOST_PORT:-8082}:8082"` |
| `/api/health` 目前**完全无鉴权**，且 CORS 是 `AllowOrigins: ["*"]` | `backend/app_http.go:23-29` |
| 项目**已有**一套管理员鉴权惯例可复用 | `backend/handlers/monitoring.go:237-258 requireAdmin`：三级密码 fallback（`monitoring.admin_password` → `ai_gateway.super_admin_password` → `console.admin_password`），header `X-Super-Admin-Password`；`X-Super-Admin-Password` 已在 CORS 白名单（`backend/app_http.go:26`） |
| 同域其它管理端点的先例：`/api/monitor/service` 已 requireAdmin，并返回 `uptime_seconds` / `server_time` / storage 统计 | `backend/handlers/monitoring.go` 的 `Service` handler |
| Gin 未配置 `SetTrustedProxies` → `c.ClientIP()` 在反代环境下不可靠 | `backend/middleware/skills.go:19-20`、`:46-50` 项目自己写的注释：「**当前项目未配置** … 显式读最稳」 |
| **前端 bundle 是公开的**：`MonitorTool.vue` 随 dist 由 Nginx 公开服务，`?detail=1` 这个参数名写在 JS 里 | `frontend/src/views/other/MonitorTool.vue` 打进 `dist/`（`backend/app_http.go` 的 `registerStaticRoutes` 服务 `./dist`） |
| 登录门禁先于任何数据加载：`MonitorTool.vue` 只有 `tryStored()` 通过才 `loadAll()` + 起轮询 | `frontend/src/views/other/MonitorTool.vue:965-984` |
| 前端已有现成的带鉴权 header 工具 | `frontend/src/composables/useAdminAuth.js:116` → `authHeader(extra = {}) => ({ 'X-Super-Admin-Password': store.getItem(storageKey) || '', ...extra })` |
| 前端**当前完全没有**消费 `/api/health` | `grep -rn "api/health" frontend/src` 零命中 |

### 地址来源现状（FINN-15 追加勘察）

| 依赖 | 既有读取点 | 读到什么 |
|---|---|---|
| OCR | `backend/handlers/ocr.go:24-27`（`NewOCRHandler`）；同值复现在 `backend/handlers/household.go:1650-1653` | `os.Getenv("OCR_SERVICE_URL")`，空则 `"http://ocr-service:8000"` |
| ASR | `backend/app_handlers.go:85`（`envOrDefault("ASR_SERVICE_URL", "http://asr-service:9000")`）；同值复现在 `backend/handlers/planner.go:31` 常量 `plannerASRServiceURL` + `:294-297`（那里用 `strings.TrimSpace` 包了一层） | env `ASR_SERVICE_URL`，空则 `"http://asr-service:9000"` |
| TTS | `backend/config/config.go:731-736`（env `TTS_SERVICE_URL` 覆盖 + 空则默认）；消费方 `backend/app_handlers.go:22` / `:83` 都传 `cfg.Chat.TTSServiceURL` | `cfg.Chat.TTSServiceURL`，最终默认 `"http://127.0.0.1:8083"` |
| Redis | `backend/config/config.go:642-644`（env `REDIS_ADDR` 覆盖）；内建默认在 `backend/config/config.go:455-462`（`Enabled:false` / `Addr:"127.0.0.1:6379"`），`backend/config.example.yaml:29` 同值 | `cfg.Redis.Addr` |
| SQLite | `backend/config/config.go:638-640`（env `DB_PATH` 覆盖）；内建默认 `backend/config/config.go:452` = `"./data/paste.db"` | `cfg.Database.Path`（**文件路径，无网络地址**） |
| （diarize，本次不做） | `backend/app_handlers.go:85` 的 `os.Getenv("DIARIZE_SERVICE_URL")`；`docker-compose.yml:35` 默认空 | 见「未决问题 6」 |

容器形态下这些 env 由 compose 显式给出（已逐行确认）：

```yaml
# docker-compose.yml
:26  - DB_PATH=/app/data/paste.db
:28  - REDIS_ADDR=redis:6379
:30  - OCR_SERVICE_URL=http://ocr-service:8000
:31  - ASR_SERVICE_URL=http://asr-service:9000
# 注意：没有 TTS_SERVICE_URL —— 故意的，见 1.5.1
```

**关键事实 5 — 本地 `go run` 不加载 `.env`**：`backend/main.go` / `backend/app_runtime.go` 中 grep `godotenv|.env` 零命中。本地开发者必须自己 `export`，而 `.env.example`（34 行，已读全文）里**根本没有** `OCR_SERVICE_URL` / `ASR_SERVICE_URL` / `TTS_SERVICE_URL` / `REDIS_ADDR` 四项 → 本地形态下 OCR/ASR 必然吃内建默认值 `ocr-service:8000` / `asr-service:9000`，**DNS 解析不了，恒 down**。这是「设计 1.5.4」要解决的问题。

### 前端现状

- `frontend/src/views/other/MonitorTool.vue`（1153 行）已有 `服务状态` tab：`<el-tab-pane name="service">` 在 `:439-503`，绑定 `service` ref，`loadService()` 在 `:688`。
- 登录：`useAdminAuth({ storageKey: 'monitor_admin_password', verifyEndpoint: '/api/monitor/verify', credentialMode: 'header', headerName: 'X-Super-Admin-Password', storage: 'session', trustStored: false })`（`:544-566`），解构出 `authHeader`（`:553`）。
- 轮询：`onMounted` 在 `:966-983`，`tryStored()` 通过后 `loadAll()` + `setInterval(..., 60000)`，按 `mainTab` 分派；`onBeforeUnmount`（`:985-990`）清 `pollTimer`。
- `API_BASE = '/api/monitor'`（`:544`）；`loadAll()` 在 `:614`。
- 已存在的展示工具函数：`formatUptime`（`:948`）/ `formatTime` / `formatBytes` / `formatNumber` 在 `:925-965` 一带；CSS class `.status-grid`（`:1080`）与 `.status-cell.ok / .warn / .danger`（`:1093-1095`）已在用。
- 路由 `/monitor` 已存在（`frontend/src/router/index.js:566-575`），**本次不需要动 router**。

### 版本信息现状

**项目当前完全没有版本注入机制**：全仓库 `grep -rn "ldflags" -- Makefile Dockerfile deploy.sh` 只有

- `Dockerfile:39` → `RUN CGO_ENABLED=1 GOOS=linux go build -tags with_utls -a -ldflags '-linkmode external -extldflags "-static"' -o server .`
- `Dockerfile:52-55` → proxy-client 的 4 个平台产物
- `deploy.sh:388` → `go build -o server main.go`（本地起服务用，无 ldflags）
- `Makefile:26` → `go build ./...`（只做编译校验）

也没有任何 `var version = ...` / `BuildTime` / `GitCommit` 变量。参考 `backend/handlers/monitoring.go:20-22`：`MonitoringHandler` 用构造时刻的 `startedAt time.Time` 表示「服务启动时间」，本次 uptime 沿用同一取法，保持项目一致。

---

## 设计

### 1. 响应契约

#### 1.1 默认形态（**向后兼容硬约束，字节级不变**）

```
GET /api/health
→ 200  Content-Type: application/json; charset=utf-8
   {"status":"ok"}
```

规则（下游无条件遵守）：

1. **不新增任何字段**。不返回 `version`、不返回 `dependencies`、不返回 `uptime`。
2. **永不返回非 200**。无论 SQLite 挂掉、Redis 挂掉、探测 panic、请求带没带鉴权 header，默认形态一律 `200`。
3. **默认形态不执行任何探测，也不做任何鉴权判定**，不进 goroutine、不建连接、不读 Redis、不读 header —— 直接 `c.JSON(200, gin.H{"status":"ok"})` 返回（与现状代码逐字节一致）。
4. **默认形态不新增任何响应头**（不加 `Cache-Control` / `Vary` / `X-*`）。理由：`deploy.sh` 与 compose 探活只看状态码，加头虽无功能影响，但"逐字节不变"这条红线值得用最保守的方式守住。新响应头只加在详细形态（含其 401/403 分支）上。
5. gin 的 `gin.H{"status":"ok"}` 序列化结果稳定为 `{"status":"ok"}`，不需要手写字符串。

#### 1.2 详细形态（**FINN-15 修订：现在需要鉴权**）

```
GET /api/health?detail=1
Headers: X-Super-Admin-Password: <管理员密码>
（等价写法：?detail=true / ?detail=yes / ?detail=on —— 大小写不敏感，去空格）
→ 200
```

其余任何值（`detail=0`、`detail=`、`detail=abc`）→ 按默认形态返回。**不返回 400。**

鉴权规则（**复用既有机制，不新造轮子**）：

1. 直接复用 `backend/handlers/monitoring.go:237-258 requireAdmin` 的语义，**逐字照抄**：
   - 密码来源顺序：`cfg.Monitoring.AdminPassword` → `cfg.AIGateway.SuperAdminPassword` → `cfg.Console.AdminPassword`（各自 `strings.TrimSpace` 后判空）；
   - 三个都空 → `403 {"error":"未配置 monitoring.admin_password、ai_gateway.super_admin_password 或 console.admin_password"}`；
   - 取值：header `X-Super-Admin-Password`，为空时回落 query `super_admin_password`（保持与既有端点一致）；
   - 不匹配 → `401 {"error":"管理员密码错误"}`。
2. **不启用** header 时不"静默降级成默认形态"，而是明确 `401`。理由见「1.6.2」。
3. **不给 401 加 `WWW-Authenticate`** —— 那是 HTTP Basic 的语义，本项目用的是自定义 header，加了会让浏览器弹出原生登录框（而 `useAdminAuth` 是表单登录），属于自找麻烦。
4. 前端侧**零 UX 回归**：`MonitorTool.vue` 的所有数据加载都在 `tryStored()` 通过之后（`:966-970`），而 `loadHealth()` 会带 `authHeader()`（见「前端改动清单」）。

响应体（**授权后**）：

```json
{
  "status": "ok",
  "version": "dev",
  "commit": "unknown",
  "build_time": "unknown",
  "uptime_seconds": 3721,
  "started_at": "2026-09-16T03:00:00Z",
  "server_time": "2026-09-16T04:02:01Z",
  "probed_at": "2026-09-16T04:02:01.512Z",
  "probe_ms": 8,
  "cached": false,
  "dependencies": [
    {"name": "sqlite", "criticality": "critical", "status": "up",       "latency_ms": 1, "source": "env",     "message": ""},
    {"name": "redis",  "criticality": "soft",     "status": "down",     "latency_ms": 0, "source": "env",     "message": "已降级到内存存储", "backend": "memory"},
    {"name": "ocr",    "criticality": "soft",     "status": "up",       "latency_ms": 5, "source": "env",     "message": ""},
    {"name": "asr",    "criticality": "soft",     "status": "up",       "latency_ms": 6, "source": "env",     "message": ""},
    {"name": "tts",    "criticality": "soft",     "status": "down",     "latency_ms": 0, "source": "default", "message": "connection refused"}
  ]
}
```

字段级定义：

| 字段 | 类型 | 说明 |
|---|---|---|
| `status` | string enum `ok`\|`degraded`\|`error` | `error` = 任一 `criticality=critical` 的依赖 down；`degraded` = 无 critical 故障但存在 `down`/`warming`；`ok` = 其余（含 `disabled`）。**与实际 HTTP 状态码无关，授权后永远是 200。** |
| `version` | string，非空 | 见「设计 4 / 3.1」。fallback `"dev"` |
| `commit` | string，非空 | ldflags 注入，fallback `"unknown"` |
| `build_time` | string，非空 | ldflags 注入，fallback `"unknown"`（RFC3339 或任意构建方给的字符串，前端只当字符串展示） |
| `uptime_seconds` | int64 ≥ 0 | 进程启动到现在的秒数 |
| `started_at` | string RFC3339 UTC | `now - uptime_seconds`，等价于 handler 构造时刻 |
| `server_time` | string RFC3339 UTC | 本次响应时刻 |
| `probed_at` | string RFC3339（带毫秒）UTC | **该快照的探测时刻**。命中缓存时 = 首次探测时刻，用于前端「上次探测时间」 |
| `probe_ms` | int ≥ 0 | 本次快照探测墙钟耗时（命中缓存时为 0） |
| `cached` | bool | true = 本次是 2s 缓存命中，未真探测 |
| `dependencies` | array，**定长 5、固定顺序** `sqlite, redis, ocr, asr, tts` | 见下 |

`dependencies[i]` 字段级定义：

| 字段 | 类型 | 说明 |
|---|---|---|
| `name` | string enum `sqlite`\|`redis`\|`ocr`\|`asr`\|`tts` | **只返回逻辑 ID**，前端据此选展示名。**绝不返回容器服务名**（`ocr-service`）或地址端口 |
| `criticality` | string enum `critical`\|`soft` | `sqlite` = `critical`，其余 = `soft`。语义写进契约，前端据此区分配色/文案 |
| `status` | string enum `up`\|`down`\|`warming`\|`disabled` | `up`=探测成功；`down`=失败/超时/解析失败；`warming`=依赖返回 503（预热中，见 `ocr-service/main.py:153-158`）；`disabled`=显式未启用 |
| `latency_ms` | int ≥ 0 | 该依赖探测耗时；`disabled` / 立即失败为 0 |
| `source` | string enum `env`\|`default` | **FINN-15 新增**。`env` = 地址来自本次显式设置的环境变量；`default` = 来自 config.yaml 或代码内建字面量。**只暴露"是否显式配置"这一位，不暴露地址原文**。用途见「1.5.4」 |
| `message` | string，可为空 | 短原因，**必须是闭集枚举**（白名单见 1.2.1），**禁止 `err.Error()` 原样落盘** |
| `backend` | string，**仅 redis 有**，enum `redis`\|`memory` | 当前实际生效的瞬时存储后端。`redis` = 进程正在用 Redis；`memory` = 已降级（对应 `state.New` 的 `backend/state/transient.go:90-93` 分支） |

##### 1.2.1 `message` 闭集白名单（**FINN-15 新增的硬约束**）

允许出现的 `message` 只有下列 11 个值（其余一律映射到最接近的一个）：

| 值 | 触发场景 |
|---|---|
| `""` | 探测成功 |
| `connection refused` | TCP 层拒绝连接 |
| `timeout` | 探测超时（含 ctx deadline） |
| `DNS 解析失败（地址可能仅适用于容器内）` | 名字解析不了 |
| `unexpected status 503` | 依赖返回 503 |
| `unexpected status 500` | 其它非 200/503 状态，按实际码填 |
| `warming up` | 依赖返回 503 且我们判定为预热（与上一行二选一，按下文 2.3 的判定表） |
| `未启用（使用内存存储）` | `cfg.Redis.Enabled == false` |
| `已启用但未配置地址` | `cfg.Redis.Enabled == true && Addr == ""` |
| `已降级到内存存储` | 启动时 Ping 失败，store 已是 Memory |
| `探测失败` | 兜底：以上都不匹配时用这个，**不带任何原始错误串** |

> ⚠️ **这是本方案最大的泄露面，必须单独说清楚**：`net/http` 的 `err.Error()` 形如
> `Get "http://ocr-service:8000/health": dial tcp 172.18.0.4:8000: connect: connection refused`
> ——**同时包含 URL、容器服务名、内网 IP、端口**。任何"直接把 err 塞进 message"的写法都会把整条内网拓扑写进响应体。
> 实现要求：写一个 `classifyHTTPErr(err) string` 归一化函数，**返回值只能取自上面这张白名单**；`message` 字段的**唯一赋值路径**就是这个函数（或 Redis/SQLite 那几条固定的中文字面量）。
> 验收方式见「测试要点」T18 / T19（对响应体做字面量黑名单断言 + 白名单断言）。

#### 1.3 错误码（**FINN-15 修订**）

| 场景 | HTTP | Body |
|---|---|---|
| 默认形态（无参数、无 header、任何依赖状态） | 200 | `{"status":"ok"}` |
| `?detail=` 任意非法值 | 200 | `{"status":"ok"}`（按默认走） |
| `?detail=1` 且未带 / 带错 `X-Super-Admin-Password` | **401** | `{"error":"管理员密码错误"}` |
| `?detail=1` 且三个管理员密码字段全空 | **403** | `{"error":"未配置 monitoring.admin_password、ai_gateway.super_admin_password 或 console.admin_password"}` |
| `?detail=1` 已授权，一切正常 | 200 | 完整对象，`status:"ok"` |
| `?detail=1` 已授权，SQLite 挂 | 200 | 完整对象，`status:"error"`，sqlite 项 `down` |
| `?detail=1` 已授权，Redis 挂 | 200 | 完整对象，`status:"degraded"`，redis 项 `down` |
| `?detail=1` 已授权，某探测 panic | 200 | 完整对象，对应项 `down` |

**默认形态不存在 4xx / 5xx；详细形态只在鉴权失败时出现 401 / 403，授权后与依赖健康无关、恒 200。** 这是设计决定，不是遗漏。

#### 1.4 详细形态的响应头（**FINN-15 新增**）

详细形态的**所有出口**（200 / 401 / 403）都必须带：

```
Cache-Control: no-store, private
Vary: X-Super-Admin-Password
X-Content-Type-Options: nosniff
```

理由：`t.jaxiu.cn` 前面是 Cloudflare。带鉴权语义的响应如果被边缘节点或中间缓存留住，会把「已授权」的响应体回给未授权访客；`Vary` 是双保险（防中间层忽略 `no-store`）。

**默认形态不加这三个头**（见 1.1 第 4 条）。

### 1.5 探测地址矩阵（容器服务名 vs localhost）— 追加要求 1 落点

#### 1.5.1 逐依赖地址来源矩阵（**必须逐依赖列明，禁止 localhost 一把梭**）

| 依赖 | 地址来源（env 名 / 配置键） | 代码内建默认值 | 该默认值**只在哪种运行形态下**才对 | 容器形态实际值 | 探测地址取自哪儿（`source` 判定） |
|---|---|---|---|---|---|
| `sqlite` | `DB_PATH` env → `cfg.Database.Path`（`config.go:638-640`） | `./data/paste.db`（`config.go:452`） | **任何形态**（进程内文件，**根本没有网络地址**，不存在 localhost/服务名二选一） | `DB_PATH=/app/data/paste.db`（`docker-compose.yml:26`） | env 非空 → `env`，否则 `default` |
| `redis` | `REDIS_ADDR` env → `cfg.Redis.Addr`（`config.go:642-644`）；也可由 `config.yaml` 的 `redis.addr` 给 | **`127.0.0.1:6379`**（`config.go:457`；`backend/config.example.yaml:29` 同值） | **只对「Redis 与 devtools 同主机 / 本地开发」形态正确** | `REDIS_ADDR=redis:6379`（`docker-compose.yml:28`） | env 非空 → `env`，否则 `default` |
| `ocr` | `OCR_SERVICE_URL` env → 构造期求值（`backend/handlers/ocr.go:24-27`） | **`http://ocr-service:8000`** | **只对「进程在 compose 网络内」形态正确** | `OCR_SERVICE_URL=http://ocr-service:8000`（`docker-compose.yml:30`，与内建默认同值） | env 非空 → `env`，否则 `default` |
| `asr` | `ASR_SERVICE_URL` env → `envOrDefault`（`backend/app_handlers.go:85`） | **`http://asr-service:9000`** | **只对「进程在 compose 网络内」形态正确** | `ASR_SERVICE_URL=http://asr-service:9000`（`docker-compose.yml:31`，与内建默认同值） | env 非空 → `env`，否则 `default` |
| `tts` | `TTS_SERVICE_URL` env → `cfg.Chat.TTSServiceURL`（`config.go:731-733`）；也可由 `config.yaml` 的 `chat.tts_service_url` 给 | **`http://127.0.0.1:8083`**（`config.go:734-736`） | **只对「TTS 与 devtools 同容器」形态正确** —— 而线上恰好就是这种 | **无 env**（compose 刻意不设），吃内建默认 → `127.0.0.1:8083` | env 非空 → `env`，否则 `default` |

**必须写进代码注释的三条判据**（每条都有 `file:line` 支撑，不是猜测）：

1. **`ocr` / `asr` 是独立容器**（`docker-compose.yml:68` / `:96`）。从 devtools 容器里探 `localhost:8000` / `localhost:9000` **永远 down** —— 那是 devtools 自己。所以这两个的**内建默认值必须是 compose 服务名**（`ocr-service:8000` / `asr-service:9000`），且这个默认值已经存在于 `ocr.go:24-27` 与 `app_handlers.go:85`，**直接对齐、不要另创**。
2. **`redis` 是唯一「默认 localhost、容器靠 env 覆盖」的行**：内建默认 `127.0.0.1:6379` 对本地开发是对的、对容器是错的；容器形态靠 `docker-compose.yml:28` 的 `REDIS_ADDR=redis:6379` 覆盖。**这是"不能用 localhost 一把梭"的第二个反面例子**——如果有人"顺手统一成 localhost"，容器里 Redis 会永远 down，而且是静默降级成内存（`state.New` 不退出），最难查。探测实现**只读 `cfg.Redis.Addr`**，不要自己重新拼地址。
3. **`tts` 是唯一「localhost 正确」的行**：`127.0.0.1:8083` 在容器形态下**就是对的**（同容器 sidecar，`entrypoint.sh:153-155` + `Dockerfile:106`）。**绝不要"顺手改成 `tts-service:8083`"** —— compose 里根本没有这个服务，改了就立刻把线上唯一正确的探测打挂。

#### 1.5.2 覆盖语义（四问四答）

1. **覆盖优先于默认？** 是。env 非空 → 用 env，忽略 `config.yaml` 与内建默认。
2. **覆盖值为空 / 未设置时的行为？** 两者的代码行为**等价**，都回落到下一级（`config.yaml` → 内建默认）。四处既有实现的写法是 `if v := os.Getenv(X); v != "" {...}` 或 `if v == "" { v = default }`，**没有区分"未设置"与"设成空串"**：
   - `config.go:731` `if ttsURL := os.Getenv("TTS_SERVICE_URL"); ttsURL != ""`
   - `config.go:642` `if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != ""`
   - `ocr.go:24` `serviceURL := os.Getenv(...); if serviceURL == "" {...}`
   - `app_handlers.go:85` `envOrDefault` → `if value == "" { return fallback }`
3. **trim 与否不统一，必须对齐指定那一处**：`backend/handlers/planner.go:294` 用了 `strings.TrimSpace(os.Getenv(...))`，而 `ocr.go:24` / `app_handlers.go:85` **不 trim**。探测实现**必须与本方案下表指定的既有来源逐字一致**，不要自创第三种写法（否则详情说 up、真实调用却因为多/少一个空格而失败）。
4. **读取时机**：既有实现全是**构造期求值一次**（`NewOCRHandler` / `buildRouteHandlers`）。因此 `HealthHandler` 也必须在**构造期**解析并存入字段，**绝不 per-request 重读 env** —— 否则"探测的地址"与"实际调用用的地址"可能在运行期分叉，详情页会说谎。

| 依赖 | 探测实现**必须对齐**的既有来源（file:line） |
|---|---|
| `sqlite` | 复用进程内 `*models.DB` 实例（`backend/models/paste.go:37-39`），**不新建连接、不读 `DB_PATH`** |
| `redis` | `cfg.Redis`（与 `app_runtime.go` 里 `state.New(rt.cfg.Redis)` 同一份），**不自己拼 `REDIS_ADDR`** |
| `ocr` | `backend/handlers/ocr.go:24-27`（env `OCR_SERVICE_URL`，空则 `http://ocr-service:8000`，**不 trim**） |
| `asr` | `backend/app_handlers.go:85`（`envOrDefault("ASR_SERVICE_URL", "http://asr-service:9000")`，**不 trim**） |
| `tts` | `cfg.Chat.TTSServiceURL`（env 已在 `config.go:731-736` 消费完）——**不要自己 `os.Getenv("TTS_SERVICE_URL")`**，二次读取会产生双份默认值逻辑 |

#### 1.5.3 地址来源与探测实现的示意（**示意，非最终代码**）

```go
// backend/handlers/health.go —— 构造期解析一次，source 同步落定
type depTarget struct {
    name   string
    base   string // 空 = 无网络地址（sqlite）
    source string // "env" | "default"
}

func resolveOCRTarget() depTarget {
    if v := os.Getenv("OCR_SERVICE_URL"); v != "" { // 与 ocr.go:24 逐字一致：不 trim
        return depTarget{name: "ocr", base: v, source: "env"}
    }
    return depTarget{name: "ocr", base: "http://ocr-service:8000", source: "default"}
}
```

> 这三行的**字面量、判空方式、是否 trim** 都必须与 `ocr.go:24-27` 一模一样，评审（C4）请逐字符 diff。

#### 1.5.4 本地开发（8080）与容器（8082）的差异，以及如何不误导开发者

| 形态 | 监听端口 | sqlite | redis | ocr | asr | tts | 详细模式典型画面 |
|---|---|---|---|---|---|---|---|
| 本地 `go run main.go` | **8080**（`config.go:449` 默认 `Port:"8080"`） | up | 看本机 Redis；`go run` **不加载 `.env`**（已 grep 确认无 godotenv），未 `export REDIS_ADDR` 时吃默认 `127.0.0.1:6379` → 本机没跑就 down | **down（`ocr-service` 名字解析不了）** | **down（同上）** | 看本机 8083 起没起 | 可能 4 个红格 |
| Docker compose（线上） | 容器内 **8082**（`PORT=8082`），宿主 `${HOST_PORT:-8082}`（`docker-compose.yml:16`） | up | up（`REDIS_ADDR=redis:6379` 覆盖生效） | up | up | up（同容器 sidecar） | 全绿 |
| Docker 直连宿主端口排障 | 宿主 **8082** | 同 compose | 同 compose | 同 compose | 同 compose | 同 compose | 同 compose |

> 顺带把端口对照写死在这儿，避免下游再问：**本地开发 = 8080；容器内 = 8082；宿主映射 = `${HOST_PORT:-8082}`**。前端 dist 由 Nginx 单独 stage 服务、`/api` 反代到 `devtools:8082` —— 这三条与探活 URL 的写法都无关（探活走容器内 `localhost:8082`，见 `docker-compose.yml:48`）。

**"本地全红"会不会误导开发者？会，如果设计不管它。四条处置：**

1. **`source` 字段**：把「显式配了地址」和「吃了内建默认值」分开。本地形态下 ocr / asr 一定是 `source: "default"`（除非开发者自己 export）。这是判断"环境没配"还是"服务真挂"的第一手信号。
2. **`message` 区分「解析不了」与「连不上」**（见 1.2.1 白名单）：
   - `DNS 解析失败（地址可能仅适用于容器内）` → 本地形态**预期内**的表现，不是故障；
   - `connection refused` → 服务真的没起来。
   这样本地开发者看到的是"地址没配"，而不是"OCR 挂了"。
3. **UI 提示行**：当存在 `status === 'down' && source === 'default'` 的依赖时，卡片底部渲染一行灰色提示：
   > 部分依赖使用内建默认地址（仅适用于容器形态）。本地开发请设置 `OCR_SERVICE_URL` / `ASR_SERVICE_URL` / `REDIS_ADDR`。
   这份数据只在授权后拿得到，**公网看不到**（见 1.6.3）。
4. **明确不做**：不在代码里加"检测到非容器就把默认值自动改成 localhost"的魔法。理由：那会让详情的探测地址与 `ocr.go` / `app_handlers.go` 的**真实调用地址分叉** —— 详情说 up、实际调用却打不通，比现在更误导，而且给"同一份地址来源"引入了第二套真相。

**配套（P2，本方案不改代码）**：`.env.example`（34 行，已读全文）里没有任何服务地址项，本地开发者无从知道要配什么。建议补 4 行注释说明「本地开发请 `export OCR_SERVICE_URL=http://127.0.0.1:8000` / `ASR_SERVICE_URL=http://127.0.0.1:9000` / `REDIS_ADDR=127.0.0.1:6379`」。**改 `.env.example` 超出 R1 写权，列为「未决问题 7」。**

### 1.6 公网暴露面：鉴权决策与字段三档裁剪 — 追加要求 2 落点

#### 1.6.1 结论速览（先给答案）

| 问题 | 结论 |
|---|---|
| 详细模式需要鉴权吗？ | **需要。复用既有 `X-Super-Admin-Password` + `requireAdmin` 三级密码 fallback，不新造中间件/token。** |
| 只在直连端口 / 内网暴露？ | **不采纳**——在本设计的写权边界内**做不到**，理由见 1.6.2。 |
| 未授权请求怎么回？ | **401（密码错）/ 403（未配置密码）**，不是静默降级成默认形态。理由见 1.6.2。 |
| 字段怎么裁？ | **三档**（默认 / 已授权 / 永不返回），逐字段表见 1.6.3。 |
| 默认模式安全吗？ | **是**：仍然只返回 `{"status":"ok"}`，零可侦察字段，仍为 200。 |
| 防缓存？ | 详细形态（含 401/403）加 `Cache-Control: no-store, private` + `Vary: X-Super-Admin-Password`（1.4）。 |

#### 1.6.2 为什么"只在直连端口/内网暴露"不可行，以及为什么未授权返回 401 而不是静默降级

**「只在直连端口 / 内网暴露」被否掉的四条理由（每条都有据）：**

1. **Nginx 配置不在本仓库**：`find . -iname "*nginx*" -not -path "*/node_modules/*"` 零命中。前端 dist 由**另一个 stage 的 Nginx** 服务，其配置不在这里。想"在 Nginx 层不给公网放行 detail 路径"，本仓库**没有可改的文件**。
2. **不存在"只在容器端口可达的路径"**：`docker-compose.yml:16` 只 publish 了一个端口 `"${HOST_PORT:-8082}:8082"`。对内（`deploy.sh:578` 的 `http://localhost:${HOST_PORT}/api/health`）与对外（CF → Nginx → `devtools:8082`）打到的是**同一个监听器上的同一个 handler**。要造出第二条只在容器网络可达的路径，必须新增监听端口 + 改 compose/Nginx，而本需求**明确禁止改部署面**。
3. **靠来源头判定不安全，且失败模式最危险**：`c.ClientIP()` 在未 `SetTrustedProxies` 的 engine 上不可靠 —— 这是**项目自己踩过的坑**，`backend/middleware/skills.go:19-20 / :46-50` 白纸黑字写着「当前项目未配置 … 显式读最稳」。而 `X-Forwarded-For` / `X-Real-IP` / `CF-Connecting-IP` 客户端可以伪造，Gin 侧的 `ClientIP()` 在默认信任链下更是取自客户端可控的 XFF。**用来源头做唯一防线的失败模式是"默认放行"** —— 判错一次就等于把内网拓扑直接摊在公网上。
4. **唯一充分且不依赖部署改动的边界是鉴权**：它的失败模式是"默认拒绝"，且已经有现成机制（`monitoring.go:237-258`）和现成的 header（`X-Super-Admin-Password` 已在 CORS 白名单，`backend/app_http.go:26`）。

**为什么未授权返回 401 而不是静默回 `{"status":"ok"}`（即"用隐蔽性换安全"）：**

- **隐蔽性本来就是零**：`MonitorTool.vue` 会被打进公开的 `dist/`，由 Nginx 公开服务，`?detail=1` 这个参数名就明文写在 JS bundle 里（`backend/app_http.go` 的 `registerStaticRoutes` 服务 `./dist`）。任何人下载前端产物就能看到参数名。靠"不告诉别人有这个参数"做防护，收益是 0。
- **401 的代价只有"确认这是个需要鉴权的端点"**，而这个信息攻击者从 bundle 里本来就有。
- **静默降级会对合法管理员撒谎**：运维 `curl` 时漏带 header 会拿到 `{"status":"ok"}`，看起来"健康"，实际根本没探测 —— 这与本功能"让依赖状态可见"的目的直接冲突。
- **与项目既有惯例一致**：`/api/monitor/*` 全部 requireAdmin，未授权就是 401/403（`monitoring.go:247-256`）。

**残余风险（如实记录，不掩盖）**：`docker-compose.yml:16` 把 8082 publish 到宿主。如果那台机器的宿主端口本身公网可达（绕过 Cloudflare），则 8082 也会被直接扫到 —— 鉴权仍然保护详细模式，默认形态本来就是设计成公开的。**这属于 jaxiu 的网络层决策，不是本设计能解决的**，仅作提示。

> 已排查掉的一条额外入口：`entrypoint.sh:166-196` 的 npc/NPS 反隧道把 `PROXY_TUNNEL_PORT` 映射到 `127.0.0.1:${PROXY_TUNNEL_PORT}`，**不是 8082**，因此它不会把 `/api/health` 单独捅出去。也就是说本设计需要考虑的入口只有两个：CF → Nginx → `devtools:8082`，以及宿主直接访问 8082。

#### 1.6.3 字段三档裁剪（**逐字段**）

三档定义：

- **A 档 = 默认模式返回**：无参数、无鉴权、公网可达、`deploy.sh` / compose 探活用。
- **B 档 = 详细模式且已授权返回**：`?detail=1` + 正确的 `X-Super-Admin-Password`。
- **C 档 = 永不返回**：任何模式、任何鉴权状态下都不出现在任何响应体 / 响应头里。

| 字段 | 是否可侦察 | A 档 | B 档 | C 档 |
|---|---|---|---|---|
| `status`（顶层） | 弱（`degraded` 透露"有依赖不正常"） | ✅ | ✅ | — |
| `version` | **是**（版本指纹 → 已知 CVE 匹配） | ❌ | ✅ | — |
| `commit` | **是**（同上） | ❌ | ✅ | — |
| `build_time` | **是**（推断构建/发版节奏） | ❌ | ✅ | — |
| `uptime_seconds` | **是**（推断重启窗口、判断是否刚打过补丁） | ❌ | ✅ | — |
| `started_at` | **是**（同上） | ❌ | ✅ | — |
| `server_time` | 否 | ❌ | ✅ | — |
| `probed_at` | **是**（探测节奏） | ❌ | ✅ | — |
| `probe_ms` | **是**（探测耗时，可用于推断网络延迟/负载） | ❌ | ✅ | — |
| `cached` | 否 | ❌ | ✅ | — |
| `dependencies[].name` | 弱（逻辑 ID `ocr`，**不含地址**） | ❌ | ✅ | **容器服务名**（`ocr-service` / `asr-service`）、任何 `host:port` |
| `dependencies[].criticality` | 弱（透露"哪些是核心依赖"） | ❌ | ✅ | — |
| `dependencies[].status` | **是**（**降级状态**：哪些辅助服务没起来、Redis 是否在降级窗口） | ❌ | ✅ | — |
| `dependencies[].latency_ms` | **是**（**探测耗时**） | ❌ | ✅ | — |
| `dependencies[].source`（新增） | 弱（透露哪些地址是显式配的） | ❌ | ✅ | 地址原文 |
| `dependencies[].message` | **是**（可能带地址/路径） | ❌ | ✅ **且必须取自 1.2.1 闭集白名单** | 原始 `err.Error()`、URL、内网 IP、文件路径 |
| `dependencies[].backend` | **是**（**降级状态**：Redis 是否已降级到内存） | ❌ | ✅ | — |
| —— | —— | —— | —— | 任何 `http://` / `https://` 串（含 OCR/ASR/TTS/Redis 地址） |
| —— | —— | —— | —— | Redis 密码、API Key、任何管理员密码 |
| —— | —— | —— | —— | 辅助服务 `/health` 的**响应体原样**（`asr-service/main.py:162` 会回显 `diarize_service_url`） |
| —— | —— | —— | —— | 配置文件路径、DB 文件路径（如 `/app/data/paste.db`） |
| —— | —— | —— | —— | 镜像 tag、ASR model / device / compute_type、`gpu_available` 等版本指纹 |

**可侦察字段清单（必须逐条标注，来自本 issue 的明确点名）**：服务名、内部地址、**版本号**、**探测耗时**、**降级状态** —— 五类全部落在 B/C 档，A 档一个都不含。

#### 1.6.4 默认模式的验收底线

`GET /api/health`（无参数）**不含任何**可侦察字段，且仍为 **200 + `{"status":"ok"}`**：

- body 严格等于 `{"status":"ok"}`（字符串相等）；
- 不新增响应头（含不加 `Cache-Control` / `Vary`）；
- 探测零执行、鉴权零判定、header 零读取；
- `deploy.sh:469/497/578/599/621/638` 与 `docker-compose.yml:48` 的行为**零变化**。

> ⚠️ 一个必须写进实现注释的坑：`deploy.sh:469` 用的是 `curl -fsS`，**`-f` 会把任何 4xx/5xx 判为失败**。任何人若把探活 URL 改成带 `?detail=1`，部署立刻会在 `wait_for_health` 处超时回滚。**本方案不改这两处**，但实现完成后要复跑一遍确认 6 个调用点的 URL 仍然是不带 query 的纯 `/api/health`（列为 C7 审查项）。

### 2. 探测策略

#### 2.1 预算

| 项 | 值 | 依据 |
|---|---|---|
| 总预算（硬上限） | **2000 ms** | compose healthcheck `timeout: 3s`（`docker-compose.yml:50`）留 1s 余量；父 `context.WithTimeout(2000ms)` 兜底 |
| 单依赖 HTTP 超时 | **800 ms** | `http.Client{Timeout: 800ms}`，5 个并发 → 串行最坏 800ms（并发后总耗时 ≈ 最慢那一个） |
| Redis 单次超时 | **300 ms** | `ctx` 超时传入 `PING` |
| SQLite 单次超时 | **300 ms** | `PingContext` / `QueryContext` 的 ctx |
| 结果缓存 TTL | **2000 ms** | 抵御 `deploy.sh` 1s 间隔的轮询风暴（`deploy.sh:469` 是 `wait_for_health url 20 1`） |
| 健康时预期耗时 | **< 30 ms** | 容器内网 TCP 连接是毫秒级；此值用于回归基线 |

**默认形态：0 ms（不探测、不鉴权）。** 这是最重要的预算保证 —— 容器探活走默认形态，永远不受依赖健康影响。

#### 2.2 并发还是串行

**并发**（5 个独立 goroutine + `sync.WaitGroup`），理由：

- 5 个依赖互不依赖，串行最坏 `5 × 800ms = 4s` 直接击穿 3s 探活超时；
- 并发最坏 ≈ 最慢单依赖 800ms，总预算内绰绰有余；
- 结果写入预分配的定长 slice 的不同下标（`results[i]`），**无需加锁**。

每个 goroutine **必须 `defer recover()`**：任一探测 panic 只能把该项标成 `down`，绝不能冒泡成 500。

#### 2.3 各依赖怎么探

| 依赖 | 探测动作 | 判定 | 地址来源（**构造期解析一次，禁止硬编码**） |
|---|---|---|---|
| `sqlite` | `db.Conn().PingContext(ctx)` + `QueryRowContext(ctx, "SELECT 1")` | 两个都成功 → `up`；任一 error → `down` | 进程内 `*models.DB`（`backend/models/paste.go:37-39`） |
| `redis` | 见 2.4 | 见 2.4 | `cfg.Redis`（`backend/config/config.go:455-462` 默认 + `:642-644` env 覆盖） |
| `ocr` | `GET {target.base}/health` | 200 → `up`；503 → `warming`；DNS 失败 → `down` + 解析失败消息；其余/错误 → `down` | env `OCR_SERVICE_URL` → 默认 `http://ocr-service:8000`，**与 `backend/handlers/ocr.go:24-27` 逐字一致** |
| `asr` | `GET {target.base}/health` | 同上 | env `ASR_SERVICE_URL` → 默认 `http://asr-service:9000`，**与 `backend/app_handlers.go:85` 逐字一致** |
| `tts` | `GET {target.base}/health` | 同上 | **只读 `cfg.Chat.TTSServiceURL`**（`config.go:731-736` 已消费 env），默认 `http://127.0.0.1:8083` |

HTTP client 构造必须照抄 `backend/handlers/ocr.go:29-34` 的写法：

```go
client := &http.Client{
    Transport: &http.Transport{Proxy: nil}, // 禁用代理，确保能访问 Docker 内部网络服务
    Timeout:   800 * time.Millisecond,
}
```

> 依据：容器 `env_file: .env` 可能带 `HTTP_PROXY`，不禁用代理会出现「探测一个内网服务却走了外网代理」的假 down。**compose 里还专门设了 `no_proxy=ocr-service,asr-service,redis,devtools,...`（`docker-compose.yml:36-37`），说明这条已经真实发生过——两处都要防。**

URL 拼接：`strings.TrimRight(base, "/") + "/health"`。**不解析响应体**（ASR 的 `/health` 返回 8 个字段：`status/model/device/requested_device/compute_type/gpu_available/diarize_enabled/diarize_service_url`，而 OCR 只返回 `{"status":"ok"}`、TTS 返回 `{"status":"ok","edge_tts":bool}`），只看状态码 + 耗时 —— 既避免与辅助服务的响应结构耦合，**也避免把 `diarize_service_url` 这类内网地址捞进我们自己的响应**（见 1.6.3 C 档）。

错误分类统一走 `classifyHTTPErr(err) string`，返回值**只能**取自 1.2.1 白名单。

#### 2.4 Redis 软依赖语义（本次设计的重点）

`state` 包当前无法表达「配置启用了 Redis，但启动时 Ping 失败已降级」。本次**扩展现有接口**（不是新建并行接口）：

```go
// backend/state/transient.go —— 新增类型 + 扩展 TransientStore 接口
type Backend string

const (
    BackendRedis  Backend = "redis"
    BackendMemory Backend = "memory"
)

type TransientStore interface {
    // ... 现有 8 个方法保持不变 ...
    Backend() Backend                 // 当前实际生效的后端
    Ping(ctx context.Context) error   // 实时连通性；MemoryStore 恒返回 nil
}

func (s *MemoryStore) Backend() Backend { return BackendMemory }
func (s *MemoryStore) Ping(context.Context) error { return nil }
func (s *RedisStore) Backend() Backend  { return BackendRedis }
func (s *RedisStore) Ping(ctx context.Context) error { return s.client.Ping(ctx).Err() }
```

判定表（handler 侧）：

| `cfg.Redis.Enabled` | 当前 store | 本次 Ping | → `status` | `backend` | `message`（闭集内） |
|---|---|---|---|---|---|
| false | Memory | 不执行 | `disabled` | `memory` | `未启用（使用内存存储）` |
| true，Addr 为空 | Memory | 不执行 | `down` | `memory` | `已启用但未配置地址` |
| true，Addr 非空，store=Memory（启动时降级） | Memory | 不执行（**没有真实 client 可 ping**） | `down` | `memory` | `已降级到内存存储` |
| true，Addr 非空，store=Redis | Redis | 执行 PING（300ms） | 成功 `up` / 失败 `down` | `redis` | 失败时 `timeout` 或 `探测失败` |

**不 500 / 不退出的实现约束**（写进代码要求）：

- 探测函数签名固定为 `func(ctx) dependency`，**永远返回一个 dependency 值，不返回 error**；error 只能被 `classifyHTTPErr` 归一化后塞进 `message` 字段。
- Redis 探测分支**不调用任何 `log.Fatal` / `os.Exit` / `panic`**，也不复用 `state.New`（那会重跑一次降级日志）。降级状态的真相只有一个来源：`store.Backend()`。
- 探测过程中的任何 error **绝不参与 HTTP 状态码决策**：授权后 `c.JSON(http.StatusOK, payload)` 是唯一出口。

> **为什么「已降级」不复活**：`state.New` 只在进程启动时判断一次（`backend/app_runtime.go:25-28`），降级后即使 Redis 恢复，进程仍用内存存储直到重启。所以 `down + backend:memory` 是**准确描述当前进程状态**，不是误报。恢复方式写进文档（重启服务），不写进 `message`。

#### 2.5 缓存

```
snapshot 结构：{ payload, probedAt }
命中条件：snapshot != nil && time.Since(probedAt) < 2s
命中时：cached = true，probe_ms = 0，probed_at = snapshot.probedAt（不刷新）
未命中：真探测，覆盖 snapshot
```

- 用 `sync.Mutex` 保护 `snapshot` / `snapshotAt` 两个字段的读写。
- 缓存**只作用于探测结果**，不影响 `server_time` / `uptime_seconds`（每次请求实时计算）。
- **鉴权在缓存之外**：未授权的请求**永远不碰缓存**（先鉴权、后读缓存），否则 401 路径会顺带把探测结果算出来，白白消耗预算。
- **不做 single-flight**（缓存冷时的并发请求各自探测一次）。理由：并发探活请求只来自已授权的前端 60s 轮询，且默认形态根本不探测，最坏情况是 2 个并发各花 < 1s，仍在预算内。若要加固列为 P2（见「未决问题 3」）。

### 3. 版本与 uptime

#### 3.1 版本标识

新增独立小包（`handlers` 与 `main` 都可读，无循环依赖）：

```go
// backend/version/version.go （新文件，约 30 行）
package version

var (
    Version   = "dev"       // -ldflags "-X devtools/version.Version=..."
    Commit    = "unknown"   // -ldflags "-X devtools/version.Commit=..."
    BuildTime = "unknown"   // -ldflags "-X devtools/version.BuildTime=..."
)

// Resolve 返回三级 fallback 后的版本串，保证非空。
func Resolve() string {
    if Version != "" && Version != "dev" { return Version }
    if v := strings.TrimSpace(os.Getenv("DEVTOOLS_VERSION")); v != "" { return v }
    return "dev"
}
```

解析优先级：**ldflags 注入 > `DEVTOOLS_VERSION` 环境变量 > 字面量 `"dev"`**。

- 加 `DEVTOOLS_VERSION` 一级的理由：当前 `deploy.sh:388` 的本地构建**没有 ldflags**（且本需求非目标明确不碰部署脚本），有了 env 这一级，运维今天就能用 `DEVTOOLS_VERSION=$(git rev-parse --short HEAD) ./deploy.sh` 拿到真版本，不必等 Dockerfile 改动。
- **不新增配置文件字段**（不动 `config.yaml` / `config.Config`），避免配置膨胀。
- 前端只把它当字符串展示，不解析语义。
- **注意**：`version` / `commit` / `build_time` 属于**可侦察字段**，只在 B 档（已授权）返回（见 1.6.3）。

> **注**：给镜像构建注入真值的 Dockerfile 改法见「4.3 可选步骤 P2」，**不属于本需求验收标准**。

#### 3.2 uptime

- 来源：`NewHealthHandler(...)` 构造时捕获的 `time.Now()`，与 `backend/handlers/monitoring.go:20-22` 的 `MonitoringHandler.startedAt` 完全一致。
- 语义：**「handler 构造时刻」≈「应用启动时刻」**（`buildRouteHandlers` 在 `main.go:12` 紧接 `newAppRuntime` 之后执行），**不是容器创建时刻**。误差在毫秒级，不需要额外引入启动戳。
- 输出：`uptime_seconds = int64(time.Since(h.startedAt).Seconds())`，`started_at = time.Now().Add(-time.Duration(uptimeSeconds) * time.Second).UTC()`（保证 `server_time - started_at == uptime_seconds`，不会出现两个字段互相矛盾）。

### 4. 后端改动清单

#### 4.1 新增文件

| 文件 | 内容 |
|---|---|
| `backend/handlers/health.go` | `HealthHandler` struct + `NewHealthHandler(db *models.DB, cfg *config.Config, store state.TransientStore) *HealthHandler`（构造期解析 5 个 `depTarget`）+ `Handle(c *gin.Context)`（先鉴权、后缓存/探测）+ `probeSQLite/probeRedis/probeHTTP` + `classifyHTTPErr` + 缓存逻辑 + 常量块。**目标 < 350 行**（`Makefile:18` 的 3000 行上限很宽松，但单文件保持单职责） |
| `backend/handlers/health_test.go` | 见「测试要点」 |
| `backend/version/version.go` | 见 3.1 |

#### 4.2 改动文件（**精确到行**）

| # | 文件:行 | 改动 |
|---|---|---|
| 1 | `backend/routes/health.go:7-11` | 签名改为 `func RegisterHealthRoute(api *gin.RouterGroup, h *RouteHandlers)`，函数体改为 `api.GET("/health", h.HealthHandler.Handle)`。**内联匿名 handler 删除** |
| 2 | `backend/routes/index.go:12-49` | `RouteHandlers` struct 增加字段 `HealthHandler *handlers.HealthHandler` |
| 3 | `backend/routes/index.go:88` | `RegisterHealthRoute(api)` → `RegisterHealthRoute(api, h)` |
| 4 | `backend/routes.go:12-49` | main 包 `routeHandlers` struct 增加字段 `healthHandler *handlers.HealthHandler` |
| 5 | `backend/routes.go:51-90` | `setupRoutes` 的 `routes.RouteHandlers{...}`（`:53` 起）字面量增加 `HealthHandler: h.healthHandler,` |
| 6 | `backend/app_handlers.go:16-137` | `buildRouteHandlers` 内 `healthHandler := handlers.NewHealthHandler(db, cfg, rt.transientStore)`；`:99` 起的返回结构体增加 `healthHandler: healthHandler,` |
| 7 | `backend/state/transient.go:40-49` | `TransientStore` 接口增加 `Backend() Backend` + `Ping(ctx context.Context) error` |
| 8 | `backend/state/transient.go:51-60 / :62-65` | `MemoryStore`（`:51`）/ `RedisStore`（`:62`）各补 2 个方法实现 + `Backend` 类型与两个常量 |

> **接入点就是 1+3+6 这「三处」**（路由文件 / `RegisterAllRoutes` 调用 / `app_handlers.go` 构造），另加 struct 字段搬运（2/4/5）。历史教训：漏掉 4/5，handler 非 nil 也永远是 nil，路由一注册就 panic。

#### 4.3 可选步骤 P2（**不属于验收标准，由 deploy-helper 决定**）

`Dockerfile:39` 注入真版本：

```dockerfile
ARG BUILD_VERSION=dev
ARG BUILD_COMMIT=unknown
RUN CGO_ENABLED=1 GOOS=linux go build -tags with_utls -a \
    -ldflags "-linkmode external -extldflags \"-static\" -X devtools/version.Version=${BUILD_VERSION} -X devtools/version.Commit=${BUILD_COMMIT}" \
    -o server .
```

注意：现有那行用的是**单引号**，要插值必须换双引号并转义 `-extldflags` 的内层引号 —— 这正是把它列为 P2 而非 P0 的原因（改错会直接炸镜像构建，而 fallback 已经保证功能可用）。**`deploy.sh` 完全不改。**

### 5. 前端改动清单

#### 5.1 文件

| 文件 | 是否改 |
|---|---|
| `frontend/src/views/other/MonitorTool.vue` | ✅ 唯一改动文件 |
| `frontend/src/router/index.js:566-575` | ❌ **不改**（`/monitor` 路由已存在，重复添加会产生重复路由） |
| 其他 vue / composable | ❌ 不改（`useAdminAuth` 直接复用，不扩） |

#### 5.2 位置

在 `<div v-else class="main-content">` 内、`<el-card class="filter-card">` **之前**，新增一张「依赖健康」卡片 —— 放在 tabs 之上，保证**任何 tab（含默认落地页「总览」）都能一眼看到**，且只有一条渲染路径，不做重复 markup。

> 若 Stage 2 评审认为改动共享布局风险偏高，可降级为放进 `服务状态` tab（`:439-503`）顶部 —— 契约与逻辑代码完全不变，只挪动模板位置。

#### 5.3 脚本改动（`<script setup>`，`:537` 起）

```js
const health = ref(null)
const healthError = ref('')
const healthLoading = ref(false)

const DEP_LABELS = { sqlite: 'SQLite', redis: 'Redis', ocr: 'OCR · 8000', asr: 'ASR · 9000', tts: 'TTS · 8083' }

// 详细模式需要管理员鉴权：必须带 authHeader()（来自 useAdminAuth，:553 已解构）
async function loadHealth() {
  healthLoading.value = true
  healthError.value = ''
  try {
    const res = await fetch('/api/health?detail=1', { headers: authHeader() })
    const data = await res.json().catch(() => ({}))
    if (res.status === 401 || res.status === 403) {
      // 不弹 ElMessage：避免与 tryStored() 的失效回滚抢；只就地提示，保留上一次快照
      healthError.value = '鉴权失效，请重新登录'
      return
    }
    if (!res.ok) throw new Error(data.error || `请求失败 (${res.status})`)
    health.value = data
  } catch (err) {
    healthError.value = err.message   // 保留上一次 health，界面标注"数据可能已过期"
  } finally {
    healthLoading.value = false
  }
}

function depClass(status) {
  if (status === 'up') return 'ok'
  if (status === 'down') return 'danger'
  if (status === 'warming') return 'warn'
  return ''
}
function depStatusLabel(status) {
  return { up: '正常', down: '不可用', warming: '预热中', disabled: '未启用' }[status] || '未知'
}
function formatProbeAgo(value) {
  if (!value) return '—'
  const seconds = Math.max(0, Math.floor((Date.now() - new Date(value).getTime()) / 1000))
  if (seconds < 5) return '刚刚'
  if (seconds < 60) return `${seconds} 秒前`
  return `${Math.floor(seconds / 60)} 分钟前`
}
// 「地址来自默认值」提示（见设计 1.5.4）—— 只在 down 且 source=default 时提示
const usingDefaultAddr = computed(() =>
  (health.value?.dependencies || []).some((d) => d.status === 'down' && d.source === 'default')
)
```

接线：

- `loadAll()`（`:614`）的 `Promise.all` 中加入 `loadHealth()`。
- `loadService()`（`:688`）内 `await loadHealth()`，失败不阻断 service 渲染（loadHealth 自己吞异常）。
- `onMounted`（`:966-983`）的 60s 轮询里，**在 `mainTab` 判断之外**无条件加一次 `loadHealth()`（卡片在 tabs 之上，切任何 tab 都要刷新）。

> **不新增第二个 `setInterval`**：现有 60s 定时器已覆盖，新增定时器会造成 `onBeforeUnmount`（`:985-990`）漏清理 / 双倍请求。60s ≥ compose 30s 探活间隔，且后端有 2s 缓存，节流足够。

> ⚠️ **原稿的一条结论已作废**：FINN-3 原稿写的是「公开端点，无需 authHeader，不要复用 API_BASE」。FINN-15 之后详细模式需要鉴权，**`loadHealth` 必须带 `authHeader()`**。`API_BASE` 那半句仍然成立（`/api/health` 不属于 `/api/monitor` 前缀，不要拼 `API_BASE + '/health'`）。

#### 5.4 模板改动

```html
<el-card class="dep-health-card" v-loading="healthLoading">
  <template #header>
    <div class="card-header">
      <span>依赖健康</span>
      <span class="small-text">
        版本 {{ health?.version || '未知' }} ·
        上次探测 {{ health?.probed_at ? formatProbeAgo(health.probed_at) : '—' }}
      </span>
    </div>
  </template>

  <el-alert v-if="healthError" type="warning" :closable="false" show-icon
    :title="`依赖状态读取失败：${healthError}（下方为上一次结果，可能已过期）`" />

  <div v-if="health?.dependencies?.length" class="status-grid">
    <div v-for="dep in health.dependencies" :key="dep.name"
         class="status-cell" :class="depClass(dep.status)">
      <div class="status-num">{{ depStatusLabel(dep.status) }}</div>
      <div class="status-label">{{ DEP_LABELS[dep.name] || dep.name }}</div>
      <div class="kpi-sub">{{ dep.latency_ms ? `${dep.latency_ms} ms` : (dep.message || '—') }}</div>
    </div>
  </div>
  <el-empty v-else-if="!healthLoading" description="未获取到依赖数据" :image-size="60" />

  <div v-if="usingDefaultAddr" class="small-text dep-addr-hint">
    部分依赖使用内建默认地址（仅适用于容器形态）。本地开发请设置 OCR_SERVICE_URL / ASR_SERVICE_URL / REDIS_ADDR。
  </div>
</el-card>
```

**模板硬约束（下游无条件遵守）**：

- **R3**：模板里**永远不写 `.value`**（Vue 3 `<script setup>` 自动 unwrap，写了直接白屏）。
- **R4**：**禁止裸下标**。用 `v-for` + `:key`，不要写 `health.dependencies[0]`；所有可能缺失的字段用 `health?.dependencies` / `dep.latency_ms`（`|| 0` 兜底）。
- 模板里出现的新函数/常量（`DEP_LABELS` / `depClass` / `depStatusLabel` / `formatProbeAgo` / `usingDefaultAddr`）必须是 `<script setup>` 顶层声明。**`usingDefaultAddr` 是 `computed`，模板里同样不写 `.value`。**
- **`?detail=1` 的响应体绝不能被当成默认形态使用，反之亦然**：默认形态没有 `dependencies` 字段，若误用默认响应去渲染依赖卡片，会走 `el-empty`（有兜底但语义错）。前端只能请求 `?detail=1`。
- 复用既有 CSS class（`.status-grid` / `.status-cell.ok|.warn|.danger` / `.card-header` / `.small-text` / `.kpi-sub`），**不要新造一套配色**，避免与总览 tab 的视觉语言冲突。`.dep-addr-hint` 是唯一新增 class，保持灰色小字即可。

---

## 兼容性与迁移

| 面 | 结论 |
|---|---|
| 默认形态响应体 | **字节级不变** `{"status":"ok"}`，不新增字段、不新增响应头。`deploy.sh:469/497/578/599/621/638` 与 `docker-compose.yml:48` 零影响 |
| HTTP 状态码 | 默认形态**恒 200**；详细形态授权后恒 200，仅在鉴权失败时 401/403 |
| `docker-compose.yml` / `deploy.sh` / Nginx | **零改动**（含非目标约束的探活方式） |
| 数据库 | **无新表、无新列、无迁移、无回填**。依赖状态不落库 |
| `state.TransientStore` 接口 | **扩展**（+2 方法）。已 grep 全仓：实现方只有 `RedisStore` / `MemoryStore` 两个（`backend/state/transient.go:51-65`），测试侧只用 `state.New()` / `state.NewMemoryStore()`（`backend/middleware/skills_test.go:24`、`backend/handlers/skills_db_test.go:418`），**无第三方 mock 实现**，扩展不破坏编译 |
| 既有 `/api/monitor/service`（`backend/handlers/monitoring.go` 的 `Service`） | 不改。它已返回 `uptime_seconds` / `server_time` / `storage`，且同样要求 `X-Super-Admin-Password` —— **`/api/health?detail=1` 与它同鉴权级别，两者并存不冲突**，`服务状态` tab 原内容保持 |
| 监控日志 | `/api/health` 在跳过名单（`backend/middleware/request_logger.go:73-82`），轮询详细形态不产生 DB 写入 |
| 配置 | 不新增 `config.yaml` 字段；环境变量只新增可选的 `DEVTOOLS_VERSION` |
| 前端 | 唯一改动文件 `MonitorTool.vue`；`useAdminAuth` 只消费不修改；登录门禁已先于数据加载，**无 UX 回归** |

---

## 风险与回滚

### R1 探测拖慢探活端点 → 容器被判 unhealthy / deploy.sh 回滚

- **成因**：依赖不可达时 TCP connect 可能挂到超时；若默认形态也探测，3s 探活超时（`docker-compose.yml:50`）会被击穿，`deploy.sh:578-600` 的失败分支会触发回滚。
- **处置（三重）**：
  1. **默认形态不探测、不鉴权**（0ms）—— 探活路径与依赖健康、鉴权判定彻底解耦；
  2. 详细形态 **800ms 单依赖 + 2000ms 父 ctx 硬上限 + 800ms client timeout**；
  3. **2000ms 结果缓存**，已授权前端 60s 轮询不形成探测风暴。
- **判定**：测试 T1 断言默认形态在依赖 3s 不响应时，handler 墙钟 < 50ms。

### R2 Redis 抖动导致状态翻转

- **成因**：Redis 软依赖，网络抖动会让 `PING` 偶发失败；若状态翻转被自动化消费（重启 / 回滚 / 摘流量），会引发雪崩。
- **处置**：
  1. **没有任何自动化消费者**：`status` 字段与 HTTP 状态码解耦，默认形态恒 200 —— 抖动**不可能**触发容器重启或 deploy 回滚（这是最重要的一条）；
  2. **`disabled` 与 `down` 严格区分**：`redis.enabled=false` 报 `disabled` 且不拉低整体 `status`，避免「没开 Redis 的部署永远 degraded」；
  3. 2s 缓存平滑瞬时抖动；`probed_at` 让前端把「一次性抖动」显示成时间戳而不是持续红灯；
  4. 不做「N 次连续失败才置 down」的粘滞逻辑 —— 见「未决问题 3」，因为它会让 `probed_at` 语义变模糊。
- **回滚**：见文末回滚步骤。

### R3 前端误解详细模式

- **成因**：默认形态 `{"status":"ok"}` 里没有 `dependencies`，前端若误用它渲染、或用裸下标访问，会白屏 / 挂载失败；此外详细形态现在会 401，忘了带 header 就会拿到错误码。
- **处置**：
  1. 前端**只**请求 `/api/health?detail=1`，且**必须带 `authHeader()`**；
  2. 401/403 单独分支处理，不弹全局 `ElMessage`（避免与 `tryStored` 的失效回滚打架）；
  3. 渲染前判空 `health?.dependencies?.length`，缺失时走 `el-empty` 而非渲染空网格；
  4. 遵守 R3（模板不写 `.value`）与 R4（不裸下标）；
  5. 后端保证 `dependencies` **永远是定长 5、字段永远存在**（不 omitempty），前端不需要处理「字段偶发缺失」。
  6. Stage 4 代码审查把「模板无 `.value` / 无裸下标 / `detail=1` / 带 `authHeader()`」列为必查项。

### R4 TTS 在 compose 部署下永远 down（误报观感）

- **成因**：`docker-compose.yml` 无 tts-service，而默认地址是 `http://127.0.0.1:8083`（容器内即自身）→ 若 sidecar 没起来就 `connection refused` → 整体 `status` 变 `degraded`。
- **处置**：这是**真实信号**，不掩盖。文档与 UI 双管：
  - `message: "connection refused"` 直接暴露原因；
  - 卡片上方展示整体 `status` 文案区分「核心依赖异常 / 部分依赖不可用 / 健康」，不把 `degraded` 渲染成红色故障；
  - **注意区分两种 down**：`source: "default"` + `connection refused` = sidecar 没起来（真故障，看 `entrypoint.sh:153-155` 的启动日志）；而如果哪天这个探测恒红且机器上根本没有 TTS，那是有人误把 `127.0.0.1:8083` "修正"成了服务名 —— 见 1.5.1 第 3 条。

### R5 未鉴权端点暴露内部信息（**FINN-15 已从"未决"升级为"已处置"**）

- **成因**：`/api/health` 的公网路径可达，详细形态会暴露版本号、依赖拓扑、降级状态；且 CORS 是 `AllowOrigins: ["*"]`（`backend/app_http.go:23-29`）。
- **处置（四层，逐层可验证）**：
  1. **鉴权**：`?detail=1` 走 `requireAdmin` 语义，未授权 401 / 未配置 403（「设计 1.2」）；
  2. **字段裁剪**：三档归类，A 档零可侦察字段（「设计 1.6.3」）；
  3. **闭环消息**：`message` 只能取自 11 条白名单，杜绝 `err.Error()` 把 URL / 容器名 / 内网 IP 带出去（「设计 1.2.1」）；
  4. **防缓存**：详细形态带 `Cache-Control: no-store, private` + `Vary: X-Super-Admin-Password`，防 CF / 中间层把已授权响应回给未授权访客（「设计 1.4」）。
- **验收**：T14–T21 全部通过；特别是 T18 的泄露断言（响应体字符串里 grep 不到任何服务名 / 地址 / 路径 / 内网 IP 正则）。
- **残余**：宿主 8082 若本身公网可达（绕过 CF），详细模式仍由密码保护，默认模式本来就是公开设计 —— 属 jaxiu 的网络层决策，本设计仅提示。

### R6 构建产物落到 SMB 挂载盘（`/Volumes/M20`）

- **处置**：本次只新增 / 修改 `docs/plans/*.md`。Stage 3/5 若需构建或测试：`go build ./...` 使用本地 GOCACHE；单测用 `:memory:` 或 `t.TempDir()`（`backend/models/monitoring_test.go:9-19` 的既有写法），**不在项目盘建临时 DB**。R11：任何跨平台后端二进制一律走 Docker，不从 macOS 直接 `go build -o`。
- **本次实测补充**：本次修订期间 `/Volumes/M20`（SMB 挂载）**发生过一次瞬时掉线**，导致 `git log` 报 `fatal: not a git repository: /Volumes/M20/.../.git/worktrees/worktree4`，数秒后自行恢复。下游 writer 请对 git 操作**串行执行并允许重试一次**，不要因为一次 `bad object HEAD` 就重做。

### 回滚

改动是**纯增量**，回滚成本极低，任一步失败都可独立回退：

1. `backend/routes/health.go` 恢复原始 11 行内联 handler（`RegisterHealthRoute(api)` 无参）；
2. `backend/routes/index.go:88` 去掉 `, h`；`RouteHandlers` 里删掉 `HealthHandler` 字段（2 处 struct 改动同删）；
3. 删除 `backend/handlers/health.go` / `health_test.go` / `backend/version/version.go`；
4. `backend/state/transient.go` 的接口扩展可**按需保留**（它不影响任何既有行为，删不删都不破坏编译）。
5. 前端：`MonitorTool.vue` 回退到本次 commit 之前的版本（唯一改动文件，`git checkout <prev> -- frontend/src/views/other/MonitorTool.vue`）。

**回滚不需要数据迁移、不需要重启配置变更、不影响既有数据。** 唯一需要人工留意的是：回滚后详细模式回到「无鉴权」，若之前已对外暴露过，**考虑轮换管理员密码**（因为版本号 / 依赖拓扑曾在一段时间内可被未授权读取）。

---

## 测试要点（交 devtools-qa，Stage 5）

**授权边界（R1）**：只允许 `go build ./...` + `go test ./handlers/...`。**不起 dev server、不 `go run`、不 `docker compose up`、不 curl 线上环境。** 全部验收断言必须能由单测证明。

测试文件：`backend/handlers/health_test.go`（package `handlers`，plain `testing` + `httptest` + `gin.TestMode`，**NO testify**，可直接复用同包 `backend/handlers/skills_db_test.go:24-33` 的 `newMemDB(t)` helper —— 它已经遵守 R5 的 `SetMaxOpenConns(1)`）。

**鉴权固定装置**：所有详细模式用例在 `cfg` 上设 `cfg.Monitoring.AdminPassword = "test-admin-pw"`，请求带 `X-Super-Admin-Password: test-admin-pw`（照抄 `backend/handlers/minimax_h3_video_test.go:75 / :97` 的既有写法）。

| # | 用例 | 断言（可判定） |
|---|---|---|
| T1 | **默认形态向后兼容**（回归红线） | `GET /api/health`（无参数、**无 header**）→ 状态码 **== 200**，`c.Writer.Body.String()` **严格等于** `{"status":"ok"}`（字符串相等，不是子集判定）。且 handler 内**零探测零鉴权**：注入一个会 sleep 3s 的 stub 依赖，耗时仍 < 50ms |
| T2 | 详细形态正常路径（已授权） | `GET /api/health?detail=1` + 正确 header（全依赖健康 stub）→ 200；`dependencies` 长度 == 5；`name` 顺序 == `[sqlite,redis,ocr,asr,tts]`；每项 `status` / `source` 非空；`version` 非空；`uptime_seconds >= 0`；`probed_at` / `server_time` 可被 `time.RFC3339` 解析；`status == "ok"` |
| T3 | `detail` 取值容错 | `?detail=true` 走详细（+header → 200）；`?detail=0` / `?detail=` / `?detail=abc` → body 严格等于 `{"status":"ok"}`，**即使带了正确 header 也一样** |
| T4 | **Redis 未启用** | `cfg.Redis.Enabled=false` + `state.NewMemoryStore()` → redis 项 `status=="disabled"`、`backend=="memory"`，且整体 `status=="ok"`（disabled 不拉低） |
| T5 | **Redis 不可用降级路径** | `cfg.Redis.Enabled=true` + `cfg.Redis.Addr="127.0.0.1:1"`（不可达）+ `state.NewMemoryStore()` → redis 项 `status=="down"`、`backend=="memory"`、`message` 非空且 ∈ 白名单；整体 `status=="degraded"`；**HTTP 状态码 == 200（不是 500）**；**handler 未 panic、未 os.Exit**（测试进程存活即证明） |
| T6 | **依赖探测超时路径** | `httptest.NewServer` 的 handler 里 `time.Sleep(3 * time.Second)`，把该 URL 作为 ASR 地址 → asr 项 `status=="down"`、`message == "timeout"`；`time.Since(start) < 2500ms` |
| T7 | 预热态（503） | stub 返回 503 → 该项 `status=="warming"`；整体 `degraded` |
| T8 | 非 200/503 状态 | stub 返回 500 → 该项 `status=="down"`、`message == "unexpected status 500"` |
| T9 | **SQLite 故障** | `db.Close()` 后探测 → sqlite 项 `status=="down"`；整体 `status=="error"`；**HTTP 仍 200** |
| T10 | 缓存 | 连续两次 `?detail=1`（间隔 < 2s，都带 header）→ 第二次 `cached == true`、`probe_ms == 0`、`probed_at` 与第一次**完全相同**；`time.Sleep(2100ms)` 后再请求 → `cached == false` 且 `probed_at` 前进 |
| T11 | 探测 goroutine 不倒灌 panic | stub 在 handler 内 `panic("boom")` → 该项 `down`、HTTP 200、测试进程存活 |
| T12 | 版本 fallback 三级 | 不设 ldflags / env → `version=="dev"`；`t.Setenv("DEVTOOLS_VERSION","v9.9.9")` → `version=="v9.9.9"` |
| T13 | `state` 层新方法 | `state.NewMemoryStore().Backend()==state.BackendMemory` 且 `Ping(ctx)==nil`；`cfg.Redis.Enabled=false` 时 `state.New(cfg)` 返回的 store `Backend()==BackendMemory`（放在 `backend/state/` 的测试或 handler 侧均可） |
| **T14** | **未授权不泄露（FINN-15 新增）** | `?detail=1` **不带 header** → `401`；响应体 `does not contain` 任一：`dependencies` / `version` / `uptime` / `"ocr"` / `"asr"` / `"tts"` / `"redis"` / `"sqlite"` |
| **T15** | **已授权正常（FINN-15 新增）** | `?detail=1` + 正确 header → 200 且可解析出 `dependencies` 长度 5 |
| **T16** | **密码错误（FINN-15 新增）** | `?detail=1` + `X-Super-Admin-Password: wrong` → 401，body 只有 `{"error":...}`，无任何依赖信息 |
| **T17** | **未配置任何管理员密码（FINN-15 新增）** | `cfg.Monitoring.AdminPassword` / `cfg.AIGateway.SuperAdminPassword` / `cfg.Console.AdminPassword` 全置空 + `?detail=1` → **403**；**且默认形态 `GET /api/health` 仍是 200 `{"status":"ok"}`**（证明默认形态不做鉴权判定） |
| **T18** | **响应体泄露断言（核心验收项）** | 构造「全依赖 down」（SQLite 已 Close、Redis 降级、OCR/ASR 指向 `127.0.0.1:1`、TTS 指向 `127.0.0.1:1`）→ 取 200 响应体字符串 `body`，断言全部为假：`strings.Contains(body, "ocr-service")` / `"asr-service"` / `"redis:6379"` / `"127.0.0.1"` / `"http://"` / `"https://"` / `"/app/"` / `".db"`；且正则 `\b(10\.|172\.(1[6-9]|2\d|3[01])\.|192\.168\.)` 不匹配 |
| **T19** | **`message` 白名单（FINN-15 新增）** | 对 T5/T6/T7/T8/T11 的响应，每一项 `message` 必须 ∈ 白名单集合；任何其它值（尤其含 `:` 或 `.` 的原始 error 串）判失败 |
| **T20** | **`source` 字段（FINN-15 新增）** | 不设 `OCR_SERVICE_URL` → ocr 项 `source=="default"`；`t.Setenv("OCR_SERVICE_URL", srv.URL)`（`httptest.NewServer` 的地址）→ `source=="env"` 且探测确实打到该 server（server 侧计数器 > 0） |
| **T21** | **响应头（FINN-15 新增）** | 详细形态的 200 / 401 / 403 三个分支：`Cache-Control` 含 `no-store` 且含 `private`，`Vary` == `X-Super-Admin-Password`；**默认形态**：这两个响应头**都不存在**（`Header().Get(...) == ""`） |
| **T22** | **DNS 失败归类（FINN-15 新增）** | 把 ASR 地址设为 `http://asr-service.invalid:9000`（保证解析不了）→ `status=="down"`、`message == "DNS 解析失败（地址可能仅适用于容器内）"`（**不是** `connection refused`），且仍 200 |

**前端验收（人工，Stage 5 只做代码级检查，不起服务）**：

- F1：`MonitorTool.vue` 模板中 grep 不到 `.value`；grep 不到 `dependencies[` 这种裸下标。
- F2：`loadHealth` 请求 URL 含 `detail=1`；未复用 `/api/monitor` 前缀。
- **F5（新增）**：`loadHealth` 的 fetch 带 `headers: authHeader()`；全文 grep 不到「无需 authHeader / 公开端点」这类残留注释。
- **F6（新增）**：401/403 分支不调用 `ElMessage`（避免与 `tryStored` 抢）。
- F3：新增卡片位置在 tabs 之上（或 `服务状态` tab 顶部），登录后 `loadAll()` 会触发 `loadHealth()`。
- F4：60s 轮询有且只有一个 `setInterval`（`onBeforeUnmount` 清理路径未新增）。

**Stage 4 代码审查 checklist（交 devtools-reviewer）**：

- C1：`RegisterAllRoutes` 三处接入点（`routes/index.go:88` + `routes.go:51-90` + `app_handlers.go:99`）全部到位，未遗漏 → 编译期可发现的是前两处，**第 6 项漏写只会在运行期 panic**，必须逐个 grep 确认。
- C2：`backend/routes/health.go` 中已无内联匿名 handler。
- C3：`health.go` 中无 `log.Fatal` / `os.Exit` / `panic(` / `c.JSON(5xx)`（**401/403 除外**）。
- C4：**地址来源矩阵逐行复核**：OCR 默认串 == `http://ocr-service:8000`（与 `ocr.go:24-27` 逐字符一致）、ASR 默认串 == `http://asr-service:9000`（与 `app_handlers.go:85` 一致）、TTS 走 `cfg.Chat.TTSServiceURL`（**不得**出现 `os.Getenv("TTS_SERVICE_URL")` 二次读取）、Redis 走 `cfg.Redis`（**不得**出现 `127.0.0.1:6379` 字面量）、SQLite 不新建连接。**grep 断言：`health.go` 里不出现 `tts-service` 这个字符串**（防误"修正"）。
- C5：HTTP client 有 `Proxy: nil` 与 800ms Timeout。
- C6：R3 / R4 前端约束无违反（模板无 `.value` / 无裸下标）。
- **C7（新增）**：`deploy.sh:469/497/578/599/621/638` 与 `docker-compose.yml:48` 的 URL 仍是**不带 query 的** `/api/health`（本方案不改它们，但要确认没被顺手改）。
- **C8（新增）**：`message` 字段的所有赋值点，其值都能在 1.2.1 白名单里找到；**grep 断言 `health.go` 里没有 `err.Error()` 直接进 message 的路径**。
- **C9（新增）**：详细形态三个出口（200/401/403）都设置了 `Cache-Control: no-store, private` + `Vary: X-Super-Admin-Password`；默认形态一个响应头都没加。
- **C10（新增）**：鉴权逻辑与 `backend/handlers/monitoring.go:237-258` 语义一致（三级密码 fallback 顺序、header 名、query 回落 `super_admin_password`、403 与 401 的区分）。

---

## 未决问题

1. ~~**详细形态是否需要鉴权**？~~ **已裁决：需要**（本 issue「追加要求 2」第 1 条强制）。方案见「设计 1.2 / 1.6」，`X-Super-Admin-Password` + `requireAdmin` 三级密码 fallback，不新造轮子。
2. **SQLite 探测只做「可读」不做「可写」**。
   父需求描述写的是「DB 文件可读写」。本方案实探 `PingContext` + `SELECT 1`（读路径），理由：真写探测要么每次探活向 WAL 写一页（30s 一次 = 2880 次/天，写放大），要么用 `BEGIN IMMEDIATE` 抢写锁（高负载下会误报 down）。写出错本身会以 5xx 出现在监控日志里。**如果评审坚持要写探测，建议放在 `?detail=1&deep=1` 三级形态**（不新增默认与详细两态的语义）。**需要人工裁决。**
3. **并发探测是否加 single-flight**。
   P2 加固项（当前靠 2s 缓存 + 默认形态不探测 + 详细形态需鉴权 兜底）。若 Stage 5 压测出并发风暴再补。
4. **Dockerfile ldflags 注入（P2）由谁做**。
   本方案不把它算进验收；若要做，属 deploy-helper 职责，且**不允许改 `deploy.sh`**。
5. **`AGENTS.md` 模块地图第 40 行（`| 40 | **Health** | 内联 | routes/health.go | — |`）在实现后应更新为 `handlers/health.go`**，并补 `routes/health.go` 详细形态说明。**本方案不写 `AGENTS.md`**，由 module-architect 在 Stage 3 完成后更新。
6. **`DIARIZE_SERVICE_URL`（第 6 个依赖）本次不做**。
   证据：`docker-compose.yml:35` 默认空（= 禁用），`backend/app_handlers.go:85` 读取它。若后续要加，**注意 ASR 的 `/health` 会原样回显 `diarize_service_url`**（`asr-service/main.py:162`）—— 本方案「不解析响应体」正是为了不把它捞进响应。要不要纳入，需人工裁决。
7. **要不要给 `.env.example` 补本地开发用的服务地址项？**
   `.env.example`（34 行）目前完全没有 `OCR_SERVICE_URL` / `ASR_SERVICE_URL` / `TTS_SERVICE_URL` / `REDIS_ADDR`，本地开发者不知道要配（见「设计 1.5.4」的配套建议）。补它超出 R1 写权（本 agent 只写 `docs/plans/`），**需要人工裁决由谁做**。

---

## 附：一页速览（给 Stage 3 的两个 writer）

```
后端（devtools-backend）
  改 routes/health.go       → 签名加 h，改走 handler
  改 routes/index.go        → struct 字段 + :88 调用加 h
  改 routes.go              → struct 字段 + setupRoutes 映射
  改 app_handlers.go        → NewHealthHandler(db, cfg, rt.transientStore) + 返回值
  改 state/transient.go     → 接口 +2 方法，两个实现各补 2 个
  新 handlers/health.go     → 构造期解析 5 个 depTarget(含 source)；
                              Handle()：默认 200 {"status":"ok"}（不探测不鉴权不加头）；
                              ?detail=1 先 requireAdmin 语义鉴权（401/403，带 no-store+Vary），
                              再探测；message 只走 classifyHTTPErr 白名单
  新 handlers/health_test.go→ T1–T22
  新 version/version.go     → 三级 fallback

前端（devtools-frontend）
  只改 views/other/MonitorTool.vue
  不动 router（/monitor 已存在）
  fetch('/api/health?detail=1', { headers: authHeader() })   ← 必须带鉴权 header！
  卡片放 tabs 之上；source=default 且 down 时显示"地址来自默认值"提示行
  模板：无 .value、无裸下标、health?.dependencies?.length 判空

红线
  默认形态必须字节等于 {"status":"ok"} 且恒 200（不加任何响应头、不做鉴权判定）
  ?detail=1 必须鉴权；未授权 401、未配置密码 403，且错误体不含任何依赖信息
  地址来源必须逐依赖对齐既有实现：ocr→ocr-service:8000 / asr→asr-service:9000 /
    tts→127.0.0.1:8083(唯一 localhost 正确的行) / redis→cfg.Redis / sqlite→进程内实例
  message 只能取自 11 条白名单，禁止 err.Error() 原样返回
  不碰 docker-compose.yml / deploy.sh / Nginx
```
