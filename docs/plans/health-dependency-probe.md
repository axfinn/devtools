# /api/health 真实依赖探测 + 监控台可视化 — 技术方案

> **状态**：待评审（Stage 2 = FINN-4 devtools-design-reviewer）
> **作者**：devtools-architect · 2026-09-16
> **上游**：FINN-2「健康检查增强」（parent issue `01a0a840-e468-76b4-81ae-6cdcd68f6a06`）
> **勘察方式**：本文所有 `file:line` 均为本次实际 grep / 阅读确认，未采用需求描述里的任何未经核实的路径

---

## 目标与非目标

### 目标

1. `GET /api/health` 增加**详细形态**，返回版本标识、uptime、以及 5 个依赖（SQLite / Redis / OCR / ASR / TTS）的实时 up / down / warming / disabled 状态与探测耗时。
2. **默认形态严格不变**：`GET /api/health`（无参数）继续返回 `200 {"status":"ok"}`，字段一个不多一个不少，且**永不返回非 200**。
3. 监控台 `views/other/MonitorTool.vue` 一处可视化：一眼看到各依赖当前是否可用、上次探测时间、版本号。
4. 新增单测覆盖：默认形态向后兼容、正常路径、Redis 降级路径、依赖探测超时路径、SQLite 故障路径。

### 非目标（明确不做）

- ❌ 不告警、不推送通知、不发邮件。
- ❌ 不做 Prometheus / metrics 导出、不做历史趋势存储（依赖状态不做时序落库）。
- ❌ **不改 `docker-compose.yml` 的 healthcheck、不改 `deploy.sh` 的探活方式**（`deploy.sh:212-227 wait_for_health` 与 `docker-compose.yml:47-52` 原样保留）。
- ❌ 不做 release 管理 / 语义化版本台账，只做「构建期注入一个字符串 + 三级 fallback」。
- ❌ 不改 `AGENTS.md` 模块地图（由 module-architect 负责；本方案只标注需要更新的行号供其参考）。
- ❌ 不新增前端路由（`/monitor` 已存在，见「前端改动清单」）。

---

## 现状

### 被改的端点本体

```go
// backend/routes/health.go:1-11  —— 全部内容就是这 11 行，内联匿名 handler，无 Handler 结构体
func RegisterHealthRoute(api *gin.RouterGroup) {
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}
```

接入点（已 grep 确认唯一）：

- `backend/routes/index.go:88` → `RegisterHealthRoute(api)`（**无 h 参数**，这是要改的签名）
- `backend/routes/index.go:12-49` → `RouteHandlers` struct（跨包穿的 handler 集合，注释说明不能 import main 以避免循环依赖）
- `backend/routes.go:12-49` → main 包内 `routeHandlers` struct；`backend/routes.go:53-90` → `setupRoutes` 里逐字段搬运到 `routes.RouteHandlers`
- `backend/app_handlers.go:16-137` → `buildRouteHandlers` 构造全部 handler，`:99-136` 组装返回

### 依赖的真实可达性（决定探测方式）

| 依赖 | 探测端点 | 源码证据 |
|---|---|---|
| OCR (8000) | `GET /health`，**engine 未就绪返回 503 `{"detail":"warming up"}`** | `ocr-service/main.py:153-158` |
| ASR (9000) | `GET /health`，**whisper_model 未就绪返回 503**，就绪返回 `{"status":"ok", "model":...}` | `asr-service/main.py:150-163` |
| TTS (8083) | `GET /health` → `{"status":"ok","edge_tts":bool}` | `tts-service/server.py:88-90` |
| SQLite | 进程内，`models.DB.Conn() *sql.DB` 可 `PingContext` | `backend/models/paste.go:37-39`、`:235`、`:241` |
| Redis | 进程内 client，`state` 包已封装 | `backend/state/transient.go:62-104` |

**关键事实 1 — TTS 不在 docker-compose 里**：`docker-compose.yml:1-147` 只有 `devtools` / `redis` / `ocr-service` / `asr-service` 四个服务，**没有 tts-service**；而默认地址是 `http://127.0.0.1:8083`（`backend/config/config.go:731-736`）。容器内 127.0.0.1 是 devtools 自己 → **compose 部署下 TTS 探测必然是 down**。这是真实信号不是 bug，但必须在文档和 UI 里说清楚（见「风险与回滚」R4）。

**关键事实 2 — 探活端点有 3 秒硬上限**：`docker-compose.yml:48-50`

```yaml
test: ["CMD-SHELL", "http_proxy= ... wget --no-verbose --tries=1 --spider http://localhost:8082/api/health"]
interval: 30s
timeout: 3s        # ← 总预算的硬约束
retries: 3
start_period: 10s
```

`deploy.sh` 侧的 `wait_for_health` 用 `curl -fsS`（`deploy.sh:212-227`），只看 HTTP 状态码，调用点 `deploy.sh:469 / 497 / 578 / 599 / 621 / 638`。**结论：探测总耗时必须显著小于 3s，且默认形态必须直接短路不探测。**

**关键事实 3 — `/api/health` 已在监控跳过名单里**：`backend/middleware/request_logger.go:73-82` 的 `shouldSkipMonitoringRequest` 前缀表包含 `"/api/health"` → 前端轮询详细形态**不产生 `http_request_logs` 行**，无写放大。

**关键事实 4 — Redis 是软依赖且当前状态不可观测**：`state.New()`（`backend/state/transient.go:71-104`）在 `Ping` 失败时 `log.Printf` 后返回 `MemoryStore`，**进程不退出**；但 `TransientStore` 接口（`:40-49`）**没有任何方法能区分当前到底跑在 Redis 还是内存降级**。这是本次必须补的能力。

### 前端现状

- `frontend/src/views/other/MonitorTool.vue`（1153 行）已有 `服务状态` tab：`<el-tab-pane name="service">` 在 `:439-503`，绑定 `service` ref，`loadService()` 在 `:684-700` 请求 `/api/monitor/service`。
- 轮询：`onMounted` 在 `:967-984`，`tryStored()` 通过后 `setInterval(..., 60000)`，按 `mainTab` 分派 `loadService()`。
- 登录复用 `frontend/src/composables/useAdminAuth.js`（header 模式 `X-Super-Admin-Password`，`sessionStorage` key `monitor_admin_password`）。
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
2. **永不返回非 200**。无论 SQLite 挂掉、Redis 挂掉、探测 panic，默认形态一律 `200`。理由见「设计 3 / 风险 R1 R2」。
3. **默认形态不执行任何探测**，不进 goroutine、不建连接、不读 Redis —— 直接 `c.JSON(200, gin.H{"status":"ok"})` 返回（与现状代码逐字节一致）。
4. gin 的 `gin.H{"status":"ok"}` 序列化结果稳定为 `{"status":"ok"}`，不需要手写字符串。

#### 1.2 详细形态

```
GET /api/health?detail=1
（等价写法：?detail=true / ?detail=yes / ?detail=on —— 大小写不敏感，去空格）
→ 200
```

其余任何值（`detail=0`、`detail=`、`detail=abc`）→ 按默认形态返回。**不返回 400。**

响应体：

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
    {"name": "sqlite", "criticality": "critical", "status": "up",       "latency_ms": 1, "message": ""},
    {"name": "redis",  "criticality": "soft",     "status": "down",     "latency_ms": 0, "message": "已降级到内存存储", "backend": "memory"},
    {"name": "ocr",    "criticality": "soft",     "status": "up",       "latency_ms": 5, "message": ""},
    {"name": "asr",    "criticality": "soft",     "status": "up",       "latency_ms": 6, "message": ""},
    {"name": "tts",    "criticality": "soft",     "status": "down",     "latency_ms": 0, "message": "connection refused"}
  ]
}
```

字段级定义：

| 字段 | 类型 | 说明 |
|---|---|---|
| `status` | string enum `ok`\|`degraded`\|`error` | `error` = 任一 `criticality=critical` 的依赖 down；`degraded` = 无 critical 故障但存在 `down`/`warming`；`ok` = 其余（含 `disabled`）。**与实际 HTTP 状态码无关，永远是 200。** |
| `version` | string，非空 | 见「设计 4」。fallback `"dev"` |
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
| `name` | string enum `sqlite`\|`redis`\|`ocr`\|`asr`\|`tts` | 固定值，前端据此选展示名 |
| `criticality` | string enum `critical`\|`soft` | `sqlite` = `critical`，其余 = `soft`。语义写进契约，前端据此区分配色/文案 |
| `status` | string enum `up`\|`down`\|`warming`\|`disabled` | `up`=探测成功；`down`=失败/超时；`warming`=依赖返回 503（预热中，见 `ocr-service/main.py:153-158`）；`disabled`=显式未启用 |
| `latency_ms` | int ≥ 0 | 该依赖探测耗时；`disabled` / 立即失败为 0 |
| `message` | string，可为空 | 短原因，**必须脱敏**：不得含完整 URL、密码、API Key、路径。允许值形如 `connection refused` / `timeout` / `unexpected status 500` / `warming up` / `已降级到内存存储` / `未启用（使用内存存储）` |
| `backend` | string，**仅 redis 有**，enum `redis`\|`memory` | 当前实际生效的瞬时存储后端。`redis` = 进程正在用 Redis；`memory` = 已降级（对应 `state.New` 的 `backend/state/transient.go:90-93` 分支） |

#### 1.3 错误码

| 场景 | HTTP | Body |
|---|---|---|
| 默认形态，一切正常 | 200 | `{"status":"ok"}` |
| 默认形态，依赖全挂 | 200 | `{"status":"ok"}` |
| `?detail=1`，一切正常 | 200 | 完整对象，`status:"ok"` |
| `?detail=1`，SQLite 挂 | 200 | 完整对象，`status:"error"`，sqlite 项 `down` |
| `?detail=1`，Redis 挂 | 200 | 完整对象，`status:"degraded"`，redis 项 `down` |
| `?detail=` 任意非法值 | 200 | `{"status":"ok"}`（按默认走） |

**本端点不存在 4xx / 5xx。** 这是设计决定，不是遗漏。

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

**默认形态：0 ms（不探测）。** 这是最重要的预算保证 —— 容器探活走默认形态，永远不受依赖健康影响。

#### 2.2 并发还是串行

**并发**（5 个独立 goroutine + `sync.WaitGroup`），理由：

- 5 个依赖互不依赖，串行最坏 `5 × 800ms = 4s` 直接击穿 3s 探活超时；
- 并发最坏 ≈ 最慢单依赖 800ms，总预算内绰绰有余；
- 结果写入预分配的定长 slice 的不同下标（`results[i]`），**无需加锁**。

每个 goroutine **必须 `defer recover()`**：任一探测 panic 只能把该项标成 `down`，绝不能冒泡成 500。

#### 2.3 各依赖怎么探

| 依赖 | 探测动作 | 判定 | 地址来源（**必须复用既有来源，禁止硬编码**） |
|---|---|---|---|
| `sqlite` | `db.Conn().PingContext(ctx)` + `QueryRowContext(ctx, "SELECT 1")` | 两个都成功 → `up`；任一 error → `down` | 进程内 `*models.DB`（`backend/models/paste.go:37-39`） |
| `redis` | 见 2.4 | 见 2.4 | `cfg.Redis`（`backend/config/config.go:210-218`，env 覆盖在 `:642-656`） |
| `ocr` | `GET {OCR_SERVICE_URL}/health` | 200 → `up`；503 → `warming`；其余/错误 → `down` | `os.Getenv("OCR_SERVICE_URL")`，空则 `http://ocr-service:8000` —— **与 `backend/handlers/ocr.go:24-27` 逐字一致** |
| `asr` | `GET {ASR_SERVICE_URL}/health` | 同上 | `os.Getenv("ASR_SERVICE_URL")`，空则 `http://asr-service:9000` —— **与 `backend/app_handlers.go:85` 逐字一致** |
| `tts` | `GET {cfg.Chat.TTSServiceURL}/health` | 同上 | **只读 `cfg.Chat.TTSServiceURL`，不要自己 `os.Getenv("TTS_SERVICE_URL")`** —— 该 env 已在 `backend/config/config.go:731-736` 被消费并带默认值 `http://127.0.0.1:8083`，二次读取会产生双份默认值逻辑 |

HTTP client 构造必须照抄 `backend/handlers/ocr.go:29-34` 的写法：

```go
client := &http.Client{
    Transport: &http.Transport{Proxy: nil}, // 禁用代理，确保能访问 Docker 内部网络服务
    Timeout:   800 * time.Millisecond,
}
```

> 依据：容器 `env_file: .env` 可能带 `HTTP_PROXY`，不禁用代理会出现「探测一个内网服务却走了外网代理」的假 down。

URL 拼接：`strings.TrimRight(base, "/") + "/health"`。**不解析响应体**（ASR 的 `/health` 返回 8 个字段：`status/model/device/requested_device/compute_type/gpu_available/diarize_enabled/diarize_service_url`，而 OCR 只返回 `{"status":"ok"}`、TTS 返回 `{"status":"ok","edge_tts":bool}`），只看状态码 + 耗时，避免与辅助服务的响应结构耦合。

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

| `cfg.Redis.Enabled` | 当前 store | 本次 Ping | → `status` | `backend` | `message` |
|---|---|---|---|---|---|
| false | Memory | 不执行 | `disabled` | `memory` | `未启用（使用内存存储）` |
| true，Addr 为空 | Memory | 不执行 | `down` | `memory` | `已启用但未配置地址` |
| true，Addr 非空，store=Memory（启动时降级） | Memory | 不执行（**没有真实 client 可 ping**） | `down` | `memory` | `已降级到内存存储` |
| true，Addr 非空，store=Redis | Redis | 执行 PING（300ms） | 成功 `up` / 失败 `down` | `redis` | 失败时 `Ping 失败` |

**不 500 / 不退出的实现约束**（写进代码要求）：

- 探测函数签名固定为 `func(ctx) dependency`，**永远返回一个 dependency 值，不返回 error**；error 只能被塞进 `message` 字段。
- Redis 探测分支**不调用任何 `log.Fatal` / `os.Exit` / `panic`**，也不复用 `state.New`（那会重跑一次降级日志）。降级状态的真相只有一个来源：`store.Backend()`。
- 探测过程中的任何 error **绝不参与 HTTP 状态码决策**：`c.JSON(http.StatusOK, payload)` 是唯一出口。

> **为什么「已降级」不复活**：`state.New` 只在进程启动时判断一次（`backend/app_runtime.go:25-28`），降级后即使 Redis 恢复，进程仍用内存存储直到重启。所以 `down + backend:memory` 是**准确描述当前进程状态**，不是误报。恢复方式写进 `message` 之外的文档（重启服务）。

#### 2.5 缓存

```
snapshot 结构：{ payload, probedAt }
命中条件：snapshot != nil && time.Since(probedAt) < 2s
命中时：cached = true，probe_ms = 0，probed_at = snapshot.probedAt（不刷新）
未命中：真探测，覆盖 snapshot
```

- 用 `sync.Mutex` 保护 `snapshot` / `snapshotAt` 两个字段的读写。
- 缓存**只作用于探测结果**，不影响 `server_time` / `uptime_seconds`（每次请求实时计算）。
- **不做 single-flight**（缓存冷时的并发请求各自探测一次）。理由：并发探活请求只来自 `deploy.sh` 1s 轮询与 compose 30s 探活，且默认形态根本不探测，最坏情况是 2 个并发各花 < 1s，仍在预算内。若要加固列为 P2（见「未决问题」）。

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

> **注**：给镜像构建注入真值的 Dockerfile 改法见「可选步骤 P2」，**不属于本需求验收标准**。

#### 3.2 uptime

- 来源：`NewHealthHandler(...)` 构造时捕获的 `time.Now()`，与 `backend/handlers/monitoring.go:20-22` 的 `MonitoringHandler.startedAt` 完全一致。
- 语义：**「handler 构造时刻」≈「应用启动时刻」**（`buildRouteHandlers` 在 `main.go:12` 紧接 `newAppRuntime` 之后执行），**不是容器创建时刻**。误差在毫秒级，不需要额外引入启动戳。
- 输出：`uptime_seconds = int64(time.Since(h.startedAt).Seconds())`，`started_at = time.Now().Add(-time.Duration(uptimeSeconds) * time.Second).UTC()`（保证 `server_time - started_at == uptime_seconds`，不会出现两个字段互相矛盾）。

### 4. 后端改动清单

#### 4.1 新增文件

| 文件 | 内容 |
|---|---|
| `backend/handlers/health.go` | `HealthHandler` struct + `NewHealthHandler(db *models.DB, cfg *config.Config, store state.TransientStore) *HealthHandler` + `Handle(c *gin.Context)` + `probeSQLite/probeRedis/probeHTTP` + 缓存逻辑 + 常量块。**目标 < 300 行**（`Makefile:18` 的 3000 行上限很宽松，但单文件保持单职责） |
| `backend/handlers/health_test.go` | 见「测试要点」 |
| `backend/version/version.go` | 见 3.1 |

#### 4.2 改动文件（**精确到行**）

| # | 文件:行 | 改动 |
|---|---|---|
| 1 | `backend/routes/health.go:7-11` | 签名改为 `func RegisterHealthRoute(api *gin.RouterGroup, h *RouteHandlers)`，函数体改为 `api.GET("/health", h.HealthHandler.Handle)`。**内联匿名 handler 删除** |
| 2 | `backend/routes/index.go:12-49` | `RouteHandlers` struct 增加字段 `HealthHandler *handlers.HealthHandler` |
| 3 | `backend/routes/index.go:88` | `RegisterHealthRoute(api)` → `RegisterHealthRoute(api, h)` |
| 4 | `backend/routes.go:12-49` | main 包 `routeHandlers` struct 增加字段 `healthHandler *handlers.HealthHandler` |
| 5 | `backend/routes.go:53-90` | `setupRoutes` 的 `routes.RouteHandlers{...}` 字面量增加 `HealthHandler: h.healthHandler,` |
| 6 | `backend/app_handlers.go:16-137` | `buildRouteHandlers` 内 `healthHandler := handlers.NewHealthHandler(db, cfg, rt.transientStore)`；`:99-136` 的返回结构体增加 `healthHandler: healthHandler,` |
| 7 | `backend/state/transient.go:40-49` | `TransientStore` 接口增加 `Backend() Backend` + `Ping(ctx context.Context) error` |
| 8 | `backend/state/transient.go:51-60 / :62-65` | `MemoryStore` / `RedisStore` 各补 2 个方法实现 + `Backend` 类型与两个常量 |

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
| 其他 vue / composable | ❌ 不改 |

#### 5.2 位置

在 `<div v-else class="main-content">` 内、`<el-card class="filter-card">` **之前**，新增一张「依赖健康」卡片 —— 放在 tabs 之上，保证**任何 tab（含默认落地页「总览」）都能一眼看到**，且只有一条渲染路径，不做重复 markup。

> 若 Stage 2 评审认为改动共享布局风险偏高，可降级为放进 `服务状态` tab（`:439-503`）顶部 —— 契约与逻辑代码完全不变，只挪动模板位置。

#### 5.3 脚本改动（`<script setup>`，`:537` 起）

```js
const health = ref(null)
const healthError = ref('')
const healthLoading = ref(false)

const DEP_LABELS = { sqlite: 'SQLite', redis: 'Redis', ocr: 'OCR · 8000', asr: 'ASR · 9000', tts: 'TTS · 8083' }

// 公开端点，无需 authHeader；不要复用 API_BASE('/api/monitor')
async function loadHealth() {
  healthLoading.value = true
  healthError.value = ''
  try {
    const res = await fetch('/api/health?detail=1')
    const data = await res.json().catch(() => ({}))
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
```

接线：

- `loadAll()`（`:605-611`）的 `Promise.all` 中加入 `loadHealth()`。
- `loadService()`（`:684-700`）内 `await loadHealth()`，失败不阻断 service 渲染（loadHealth 自己吞异常）。
- `onMounted`（`:967-984`）的 60s 轮询里，**在 `mainTab` 判断之外**无条件加一次 `loadHealth()`（卡片在 tabs 之上，切任何 tab 都要刷新）。

> **不新增第二个 `setInterval`**：现有 60s 定时器已覆盖，新增定时器会造成 `onBeforeUnmount`（`:986-992`）漏清理 / 双倍请求。60s ≥ compose 30s 探活间隔，且后端有 2s 缓存，节流足够。

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
</el-card>
```

**模板硬约束（下游无条件遵守）**：

- **R3**：模板里**永远不写 `.value`**（Vue 3 `<script setup>` 自动 unwrap，写了直接白屏）。
- **R4**：**禁止裸下标**。用 `v-for` + `:key`，不要写 `health.dependencies[0]`；所有可能缺失的字段用 `health?.dependencies` / `dep.latency_ms`（`|| 0` 兜底）。
- 模板里出现的新函数/常量（`DEP_LABELS` / `depClass` / `depStatusLabel` / `formatProbeAgo`）必须是 `<script setup>` 顶层声明。
- **`?detail=1` 的响应体绝不能被当成默认形态使用，反之亦然**：默认形态没有 `dependencies` 字段，若误用默认响应去渲染依赖卡片，会走 `el-empty`（有兜底但语义错）。前端只能请求 `?detail=1`。
- 复用既有 CSS class（`.status-grid` / `.status-cell.ok|.warn|.danger` / `.card-header` / `.small-text` / `.kpi-sub`），**不要新造一套配色**，避免与总览 tab 的视觉语言冲突。

---

## 兼容性与迁移

| 面 | 结论 |
|---|---|
| 默认形态响应体 | **字节级不变** `{"status":"ok"}`，不新增字段。`deploy.sh:469/497/578/599/621/638` 与 `docker-compose.yml:48` 零影响 |
| HTTP 状态码 | 默认与详细**恒 200**，无 4xx / 5xx，探活语义不变 |
| `docker-compose.yml` / `deploy.sh` | **零改动**（含非目标约束的探活方式） |
| 数据库 | **无新表、无新列、无迁移、无回填**。依赖状态不落库 |
| `state.TransientStore` 接口 | **扩展**（+2 方法）。已 grep 全仓：实现方只有 `RedisStore` / `MemoryStore` 两个（`backend/state/transient.go:51-65`），测试侧只用 `state.New()` / `state.NewMemoryStore()`（`backend/middleware/skills_test.go:24`、`backend/handlers/skills_db_test.go:418`），**无第三方 mock 实现**，扩展不破坏编译 |
| 既有 `/api/monitor/service`（`backend/handlers/monitoring.go:124-148`） | 不改。它已返回 `uptime_seconds` / `storage`；本次健康卡片走新端点，**两者并存不冲突**，`服务状态` tab 原内容保持 |
| 监控日志 | `/api/health` 在跳过名单（`backend/middleware/request_logger.go:73-82`），轮询详细形态不产生 DB 写入 |
| 配置 | 不新增 `config.yaml` 字段；环境变量只新增可选的 `DEVTOOLS_VERSION` |

---

## 风险与回滚

### R1 探测拖慢探活端点 → 容器被判 unhealthy / deploy.sh 回滚

- **成因**：依赖不可达时 TCP connect 可能挂到超时；若默认形态也探测，3s 探活超时（`docker-compose.yml:50`）会被击穿，`deploy.sh:578-600` 的失败分支会触发回滚。
- **处置（三重）**：
  1. **默认形态不探测**（0ms）—— 探活路径与依赖健康彻底解耦；
  2. 详细形态 **800ms 单依赖 + 2000ms 父 ctx 硬上限 + 800ms client timeout**；
  3. **2000ms 结果缓存**，`deploy.sh` 1s 轮询不会形成探测风暴。
- **判定**：测试 T5 断言详细形态在依赖 3s 不响应时，handler 墙钟 < 2500ms。

### R2 Redis 抖动导致状态翻转

- **成因**：Redis 软依赖，网络抖动会让 `PING` 偶发失败；若状态翻转被自动化消费（重启 / 回滚 / 摘流量），会引发雪崩。
- **处置**：
  1. **没有任何自动化消费者**：`status` 字段与 HTTP 状态码解耦，默认形态恒 200 —— 抖动**不可能**触发容器重启或 deploy 回滚（这是最重要的一条）；
  2. **`disabled` 与 `down` 严格区分**：`redis.enabled=false` 报 `disabled` 且不拉低整体 `status`，避免「没开 Redis 的部署永远 degraded」；
  3. 2s 缓存平滑瞬时抖动；`probed_at` 让前端把「一次性抖动」显示成时间戳而不是持续红灯；
  4. 不做「N 次连续失败才置 down」的粘滞逻辑 —— 见「未决问题」，因为它会让 `probed_at` 语义变模糊。
- **回滚**：见文末回滚步骤。

### R3 前端误解详细模式

- **成因**：默认形态 `{"status":"ok"}` 里没有 `dependencies`，前端若误用它渲染、或用裸下标访问，会白屏 / 挂载失败。
- **处置**：
  1. 前端**只**请求 `/api/health?detail=1`，不读默认形态；
  2. 渲染前判空 `health?.dependencies?.length`，缺失时走 `el-empty` 而非渲染空网格；
  3. 遵守 R3（模板不写 `.value`）与 R4（不裸下标）；
  4. 后端保证 `dependencies` **永远是定长 5、字段永远存在**（不 omitempty），前端不需要处理「字段偶发缺失」。
  5. Stage 4 代码审查把「模板无 `.value` / 无裸下标 / `detail=1`」列为必查项。

### R4 TTS 在 compose 部署下永远 down（误报观感）

- **成因**：`docker-compose.yml` 无 tts-service，而默认地址是 `http://127.0.0.1:8083`（容器内即自身）→ 恒 `connection refused` → 整体 `status` 恒 `degraded`。
- **处置**：这是**真实信号**，不掩盖。文档与 UI 双管：
  - `message: "connection refused"` 直接暴露原因；
  - 卡片上方展示整体 `status` 文案区分「核心依赖异常 / 部分依赖不可用 / 健康」，不把 `degraded` 渲染成红色故障；
  - 本方案记录处置办法：需要 TTS 时给 devtools 容器设 `TTS_SERVICE_URL=http://host.docker.internal:8083`（`config.go:731-736` 已支持 env 覆盖，**不需要改代码**）。

### R5 未鉴权端点暴露内部信息

- **成因**：`/api/health` 是公开端点（无鉴权），详细形态会暴露版本号、内部容器主机名、依赖拓扑；且 CORS 是 `AllowOrigins: ["*"]`（`backend/app_http.go:23-29`）。
- **处置**：
  1. **payload 严格脱敏**：不返回任何 URL（只返回 `name` + 状态 + 短原因）、不返回密码 / Key / 文件路径；
  2. 不返回 Redis 地址、DB 路径、`.env` 内容；
  3. 详细形态**不包含任何可用于登录的凭据**，即使被扫描也只泄露「有 OCR/ASR/TTS 三类依赖」这一层信息。
- **升级路径**：若评审认为仍过宽，加一行可选的管理员头校验（复用 `backend/handlers/monitoring.go:237-258 requireAdmin` 的 `X-Super-Admin-Password` 语义）——**但这会让 Stage 5 的 QA 必须先拿到密码**，故本方案默认不加。列为「未决问题」。

### R6 构建产物落到 SMB 挂载盘（`/Volumes/M20`）

- **处置**：本次只新增 `docs/plans/*.md`。Stage 3/5 若需构建或测试：`go build ./...` 使用本地 GOCACHE；单测用 `:memory:` 或 `t.TempDir()`（`backend/models/monitoring_test.go:9-19` 的既有写法），**不在项目盘建临时 DB**。R11：任何跨平台后端二进制一律走 Docker，不从 macOS 直接 `go build -o`。

### 回滚

改动是**纯增量**，回滚成本极低，任一步失败都可独立回退：

1. `backend/routes/health.go` 恢复原始 11 行内联 handler（`RegisterHealthRoute(api)` 无参）；
2. `backend/routes/index.go:88` 去掉 `, h`；`RouteHandlers` 里删掉 `HealthHandler` 字段（2 处 struct 改动同删）；
3. 删除 `backend/handlers/health.go` / `health_test.go` / `backend/version/version.go`；
4. `backend/state/transient.go` 的接口扩展可**按需保留**（它不影响任何既有行为，删不删都不破坏编译）。
5. 前端：`MonitorTool.vue` 回退到本次 commit 之前的版本（唯一改动文件，`git checkout <prev> -- frontend/src/views/other/MonitorTool.vue`）。

**回滚不需要数据迁移、不需要重启配置变更、不影响既有数据。**

---

## 测试要点（交 devtools-qa，Stage 5）

**授权边界（R1）**：只允许 `go build ./...` + `go test ./handlers/...`。**不起 dev server、不 `go run`、不 `docker compose up`、不 curl 线上环境。** 全部验收断言必须能由单测证明。

测试文件：`backend/handlers/health_test.go`（package `handlers`，plain `testing` + `httptest` + `gin.TestMode`，**NO testify**，可直接复用同包 `backend/handlers/skills_db_test.go:24-33` 的 `newMemDB(t)` helper —— 它已经遵守 R5 的 `SetMaxOpenConns(1)`）。

| # | 用例 | 断言（可判定） |
|---|---|---|
| T1 | **默认形态向后兼容**（回归红线） | `GET /api/health`（无参数、无 header）→ 状态码 **== 200**，`c.Writer.Body.String()` **严格等于** `{"status":"ok"}`（字符串相等，不是子集判定）。且 handler 内**零探测**：注入一个会 sleep 3s 的 stub 依赖，耗时仍 < 50ms |
| T2 | 详细形态正常路径 | `GET /api/health?detail=1`（全依赖健康 stub）→ 200；`dependencies` 长度 == 5；`name` 顺序 == `[sqlite,redis,ocr,asr,tts]`；每项 `status` 非空；`version` 非空；`uptime_seconds >= 0`；`probed_at` / `server_time` 可被 `time.RFC3339` 解析；`status == "ok"` |
| T3 | `detail` 取值容错 | `?detail=true` 走详细；`?detail=0` / `?detail=` / `?detail=abc` → body 严格等于 `{"status":"ok"}` |
| T4 | **Redis 未启用** | `cfg.Redis.Enabled=false` + `state.NewMemoryStore()` → redis 项 `status=="disabled"`、`backend=="memory"`，且整体 `status=="ok"`（disabled 不拉低） |
| T5 | **Redis 不可用降级路径** | `cfg.Redis.Enabled=true` + `cfg.Redis.Addr="127.0.0.1:1"`（不可达）+ `state.NewMemoryStore()` → redis 项 `status=="down"`、`backend=="memory"`、`message` 非空；整体 `status=="degraded"`；**HTTP 状态码 == 200（不是 500）**；**handler 未 panic、未 os.Exit**（测试进程存活即证明） |
| T6 | **依赖探测超时路径** | `httptest.NewServer` 的 handler 里 `time.Sleep(3 * time.Second)`，把该 URL 作为 ASR 地址 → asr 项 `status=="down"`、`message` 含 `timeout` 或 `deadline`（大小写不敏感）；`time.Since(start) < 2500ms` |
| T7 | 预热态（503） | stub 返回 503 → 该项 `status=="warming"`；整体 `degraded` |
| T8 | 非 200/503 状态 | stub 返回 500 → 该项 `status=="down"` |
| T9 | **SQLite 故障** | `db.Close()` 后探测 → sqlite 项 `status=="down"`；整体 `status=="error"`；**HTTP 仍 200** |
| T10 | 缓存 | 连续两次 `?detail=1`（间隔 < 2s）→ 第二次 `cached == true`、`probe_ms == 0`、`probed_at` 与第一次**完全相同**；`time.Sleep(2100ms)` 后再请求 → `cached == false` 且 `probed_at` 前进 |
| T11 | 探测 goroutine 不倒灌 panic | stub 在 handler 内 `panic("boom")` → 该项 `down`、HTTP 200、测试进程存活 |
| T12 | 版本 fallback 三级 | 不设 ldflags / env → `version=="dev"`；`t.Setenv("DEVTOOLS_VERSION","v9.9.9")` → `version=="v9.9.9"` |
| T13 | `state` 层新方法 | `state.NewMemoryStore().Backend()==state.BackendMemory` 且 `Ping(ctx)==nil`；`cfg.Redis.Enabled=false` 时 `state.New(cfg)` 返回的 store `Backend()==BackendMemory`（放在 `backend/state/` 的测试或 handler 侧均可） |

**前端验收（人工，Stage 5 只做代码级检查，不起服务）**：

- F1：`MonitorTool.vue` 模板中 grep 不到 `.value`；grep 不到 `dependencies[` 这种裸下标。
- F2：`loadHealth` 请求 URL 含 `detail=1`；未复用 `/api/monitor` 前缀。
- F3：新增卡片位置在 tabs 之上（或 `服务状态` tab 顶部），登录后 `loadAll()` 会触发 `loadHealth()`。
- F4：60s 轮询有且只有一个 `setInterval`（`onBeforeUnmount` 清理路径未新增）。

**Stage 4 代码审查 checklist（交 devtools-reviewer）**：

- C1：`RegisterAllRoutes` 三处接入点（`routes/index.go:88` + `routes.go:53-90` + `app_handlers.go:99-136`）全部到位，未遗漏 → 编译期可发现的是前两处，**第 6 项漏写只会在运行期 panic**，必须逐个 grep 确认。
- C2：`backend/routes/health.go` 中已无内联匿名 handler。
- C3：`health.go` 中无 `log.Fatal` / `os.Exit` / `panic(` / `c.JSON(5xx)`。
- C4：探测地址来源未被硬编码（OCR/ASR 的默认值必须来自 env + 既有默认串，TTS 必须来自 `cfg.Chat.TTSServiceURL`）。
- C5：HTTP client 有 `Proxy: nil` 与 800ms Timeout。
- C6：R3 / R4 前端约束无违反。

---

## 未决问题

1. **详细形态是否需要鉴权**？（R5）
   本方案默认**不加鉴权**（保持 QA 可直接 curl / 单测可直接断言，且 `status` 已脱敏）。若评审要求加 `X-Super-Admin-Password`，需要同步告知 Stage 5：验收请求必须带头，否则 `?detail=1` 会 401 —— 这会改变验收标准 #2 的执行方式。**需要人工裁决。**
2. **SQLite 探测只做「可读」不做「可写」**。
   父需求描述写的是「DB 文件可读写」。本方案实探 `PingContext` + `SELECT 1`（读路径），理由：真写探测要么每次探活向 WAL 写一页（30s 一次 = 2880 次/天，写放大），要么用 `BEGIN IMMEDIATE` 抢写锁（高负载下会误报 down）。写出错本身会以 5xx 出现在监控日志里。**如果评审坚持要写探测，建议放在 `?detail=1&deep=1` 三级形态**（不新增默认与详细两态的语义）。**需要人工裁决。**
3. **并发探测是否加 single-flight**。
   P2 加固项（当前靠 2s 缓存 + 默认形态不探测兜底）。若 Stage 5 压测出并发风暴再补。
4. **Dockerfile ldflags 注入（P2）由谁做**。
   本方案不把它算进验收；若要做，属 deploy-helper 职责，且**不允许改 `deploy.sh`**。
5. **`AGENTS.md` 模块地图第 40 行（`| 40 | **Health** | 内联 | routes/health.go | — |`）在实现后应更新为 `handlers/health.go`**，并补 `routes/health.go` 详细形态说明。**本方案不写 `AGENTS.md`**，由 module-architect 在 Stage 3 完成后更新。

---

## 附：一页速览（给 Stage 3 的两个 writer）

```
后端（devtools-backend）
  改 routes/health.go       → 签名加 h，改走 handler
  改 routes/index.go        → struct 字段 + :88 调用加 h
  改 routes.go              → struct 字段 + setupRoutes 映射
  改 app_handlers.go        → NewHealthHandler(db, cfg, rt.transientStore) + 返回值
  改 state/transient.go     → 接口 +2 方法，两个实现各补 2 个
  新 handlers/health.go     → Handle()；默认 200 {"status":"ok"}；?detail=1 才探测
  新 handlers/health_test.go→ T1–T13
  新 version/version.go     → 三级 fallback

前端（devtools-frontend）
  只改 views/other/MonitorTool.vue
  不动 router（/monitor 已存在）
  fetch('/api/health?detail=1') → 卡片放 tabs 之上
  模板：无 .value、无裸下标、health?.dependencies?.length 判空

红线
  默认形态必须字节等于 {"status":"ok"} 且恒 200
  TTS/OCR/ASR 地址必须复用既有 env 来源，不硬编码
  不碰 docker-compose.yml / deploy.sh
```
