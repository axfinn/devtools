# ssrules 模块 —— 落地方案（v3）

> Stage 1（devtools-architect）产出。本方案基于对上游 commit `e7f37dac9fe94390cf0528e3db8ee5d5fbbb8c8e`
> （本地独有，origin 无）与基线 `origin/main`（`81199bd`）的**逐行实读勘察**产出。
> 下游 Stage 2 评审 / Stage 3 前端 / Stage 4 后端 / Stage 5 测试照此执行。
>
> **本方案里每条 `file:line` 都实读验证过**，勘察命令与原始输出见 §9（证据附录）。
>
> **v3 = FINN-102 回改版**。FINN-90（Stage 2 评审）判 FAIL，阻塞项 **B1（路径穿越断言）+ B2（限流键）**。
> 本版按 **PM 裁决 1–6** 与 **12 条非阻塞项**逐条回改；回改点 → 章节 → 断言 的对照表见 **§5.4**。
> 被本版作废的 v1/v2 论断，文中用 ⛔ 标注。

---

## 0. 一页速览（给下游的 60 秒版）

| 项 | 结论 |
|---|---|
| 模块 | 全站**唯一一份** Shadowsocks 分流规则文件；管理页填域名/URL 列表 → 生成 ACL 正文 → 公开链接分发 |
| 对外端点 | 管理端 4 个（`/api/ssrules*`，需 `X-Super-Admin-Password`）+ 分发端 1 个（`/ss/:token/:filename`，token 门） |
| 分发端点 | **必须**在 `registerPublicRoutes` 显式注册（否则被 NoRoute 的 SPA fallback 兜成 `index.html`，客户端拿到 HTML 却"成功"） |
| 前端 | `frontend/src/views/other/SsRuleTool.vue` + `frontend/src/router/index.js` 一条路由 |
| 取件方式 | `git checkout e7f37dac -- <path>` **单路径**取件；**`backend/config/config.go` / `config.example.yaml` 除外，必须手工打补丁**（见 §5.3 P0-4）；**`backend/routes.go` 只加 2 行后单独 `gofmt -w`**（见 §5.2.1） |
| **必须先修的 P0** | 限流绕过 / 限流器共用 / **限流键可伪造** / 模块关闭语义不一致 / config 取件回退 FINN-56 / 路径穿越判据与回归用例 / **未匹配 `/ss/` 路径回落 SPA** |
| 上游 commit 直接可用吗 | **否**。上游有 7 个 P0、6 个 P1 缺陷（§5.3），必须先修再进 Stage 4 |
| 回滚 | ① 配置关模块（依赖 P0-3 修好才成立；**重启后生效**，config.yaml 无热加载）② 删路由（**NoRoute 的 `/ss/` 特判必须保留**，否则客户端拿到 HTML） |
| **限流键来源（B2）** | **模块内自建中间件** + `middleware.RateLimiter.Allow(ip)`；键 = `CF-Connecting-IP` → XFF **最右**一项 → `RemoteAddr` host。**不用** `c.ClientIP()`（可被客户端伪造，§3.4.4） |
| **未匹配 `/ss/` 路径（裁决 2）** | **404** + body 逐字节 `规则文件不存在`（NoRoute 特判 `/ss/` 前缀，**不回落 SPA index.html**）—— 见 §3.4.1 |
| **鉴权（裁决 5）** | 管理端**只认** header `X-Super-Admin-Password`；`?admin_password=` / `?super_admin_password=` **一律忽略** |
| **时区（裁决 6）** | `updated_at` **统一 UTC**（`Z` 结尾），前端 `toLocaleString()` 渲染 |
| **本版次** | v3 —— FINN-102（Stage 1 回改单），按 PM 裁决 1–6 回改完毕；见 §5.4 回改对照表 |

---

## 1. 目标与非目标

### 1.1 目标

1. 落地一个 **Shadowsocks 规则文件生成器**：用户填域名 / URL / IP / CIDR 列表 → 生成 ACL 正文 → 通过公开链接分发给各客户端作远程规则源。
2. 产出**可执行的 API 契约**（端点、字段、状态码、错误体），前后端两边都照它写。
3. 产出**公网暴露面的安全判据**（t.jaxiu.cn 在 Cloudflare 后公网可达，`/ss/*` 是全站唯一一个"任何人凭 URL 就能拿数据"的端点）。
4. 产出**可照抄的 Stage 4 / Stage 5 命令 + 可判定的验收断言**。

### 1.2 非目标（明确不做）

| 不做 | 理由 |
|---|---|
| 多套规则 / 规则类型建模 / 按客户端区分规则集 | 上游 commit 设计已定："单一事实源，只存原始输入 + 生成的正文" |
| token 过期时间 / 一次性 token | 客户端规则源需要长期可用；止损手段是"换发链接"按钮 |
| token 加密入库 / HMAC 派生 | 单行表 + 可换发，收益不抵复杂度 |
| `.pac` 后缀 | PAC 是 JavaScript，把 ACL 正文当 `.pac` 发出去客户端会静默拿到语法错误的脚本 |
| 把域名解析/格式转换搬后端之外的路径 | R2：客户端能算的不走网络（本模块的"生成 ACL"必须服务端做，因为要落库、要公开分发） |
| 改共享 `middleware.RateLimiter`（加 `KeyPrefix` 之类） | 裁决 1 明令"**不动共享 `middleware.RateLimiter`**"（避免影响全站其它端点）；另开单 |
| 改 `backend/middleware/compress.go` | 非阻塞项 12：保持现有说明，客户端兼容性未验前不碰 |
| 修 `backend/` 里另外 34 个 gofmt 不干净的文件 | S9 越界改动；见 §9.3 证据 —— 那是仓库既有红项，不是本单引入 |

---

## 2. 现状（已核实）

> 本章所有行号以 **`origin/main`（`81199bd`）** 为基准 —— 这是 Stage 3/4 的取件基线。上游 commit 引入的新文件行号另行标注「上游 commit 内」。

### 2.1 基线事实

| 事实 | 证据 |
|---|---|
| 上游 commit 对象在本地对象库可见 | `git cat-file -e e7f37dac…` → `OBJECT OK`；`git show --stat` → 14 files / +1308 −38 |
| 上游 commit 的父提交 | `9992ac652553752cef5c73f6c43012e9b69a7e33` |
| `origin/main` = `81199bd` | `git rev-parse origin/main` |
| `origin/main` 上**不存在**任何 ssrule 文件 | `git ls-tree -r origin/main --name-only \| grep -ci ssrule` → **0** |
| 上游 commit 的父树 vs `origin/main`：**只有 2 个文件分叉** | `git diff --stat 9992ac6 origin/main -- <12 个改动文件>` → `backend/config/config.go`（1 行）+ `backend/config.example.yaml`（1 行） |
| `frontend/src/router/index.js` 在 `9992ac6`→`origin/main` **0 差异** | `git diff --stat 9992ac6 origin/main -- frontend/src/router/index.js` → 空 |
| `AGENTS.md` 模块表当前 **42 行**（末行 `#42 Trip 旅游行程`，`:52`） | `grep -cE '^\| [0-9]+ \|' AGENTS.md` → **42** |

> ⚠️ 也就是说：**除 config 两个文件外，其余 10 个被改文件在 `origin/main` 上与上游父提交逐字节相同** → 单路径 checkout 对它们是干净的。

### 2.2 上游 commit 的接线链（实读结果）

```
main.go
 └─ app_http.go:newHTTPRouter()          ← 创建 createRateLimiter（全站唯一实例, 10/min）
     ├─ setupRoutes(api, createRateLimiter, handlers)
     │   └─ routes.go:setupRoutes()       ← routeHandlers(小写, package main) → routes.RouteHandlers(大写)
     │       └─ routes/index.go:RegisterAllRoutes()
     │           └─ routes/ssrule.go:RegisterSSRuleRoutes()   ← /api/ssrules*
     ├─ registerPublicRoutes(router, handlers)
     │   └─ router.GET("/ss/:token/:filename", handlers.ssRuleHandler.Download)   ← 分发端点
     └─ registerStaticRoutes(router, "./dist")
         └─ router.NoRoute(...)           ← SPA fallback（未注册路径兜成 index.html）
```

### 2.3 关键既有代码位置（`origin/main` 现状，Stage 4 的行号锚点）

| 位置 | 现状内容 | 插入点 |
|---|---|---|
| `backend/routes/index.go:49` | `HealthHandler             *handlers.HealthHandler` | 之后加 `SSRuleHandler` 字段 |
| `backend/routes/index.go:91` | `RegisterTripRoutes(api, h, createRateLimiter)` | 之后加 `RegisterSSRuleRoutes(...)` |
| `backend/routes.go:12` | `type routeHandlers struct {` | 结构体加 `ssRuleHandler` 字段（**该文件 gofmt 不干净，见 §9.3**） |
| `backend/routes.go:91` | `HealthHandler:            h.healthHandler,` | 之后加 `SSRuleHandler: h.ssRuleHandler,` |
| `backend/app_handlers.go:92` | `healthHandler := handlers.NewHealthHandler(db, cfg, rt.transientStore)` | 之后构造 `ssRuleHandler` |
| `backend/app_handlers.go:140` | `healthHandler:             healthHandler,` | 之后加 `ssRuleHandler: ssRuleHandler,` |
| `backend/app_http.go:53` | `router.Any("/mock/:id", handlers.mockAPIHandler.Execute)` | 之后加 `/ss/` 的 **4 条**显式路由（§3.4.1(d)） |
| `backend/app_http.go:78-103` | `router.NoRoute(...)`（SPA fallback） | 在 `/api` 特判（`:80-83`）**之后**插入 `/ss/` 前缀特判分支（§3.4.1(d)） |
| `backend/config/config.go:53` | `Skills              SkillsConfig              \`yaml:"skills"\`` | 之后加 `SSRules` 字段 + `SSRulesConfig` 类型 |
| `backend/config/config.go:530` | `Args: []string{"minimax-coding-plan-mcp", "-y"}` | **`origin/main` 已是 `{"minimax-coding-plan-mcp", "-y"}`**（FINN-56）→ 整文件 checkout 会回退它 |
| `backend/config.example.yaml:237-238` | `- "minimax-coding-plan-mcp"` / `- "-y"` | **`origin/main` 已多一行 `"-y"`** → 同上 |
| `frontend/src/router/index.js:105` | `},`（`/proxy` 路由块结束） | 之后插入 `/ssrules` 路由块 |
| `AGENTS.md:52` | 模块表最后一行 `#42 Trip 旅游行程` | 之后加 `#43 SSRules`；标题 `:7` `## 34 模块地图` 与 `:223` 的 `34 模块地图` 一并改为 **43**（裁决 4） |

### 2.4 相关既有基础设施（实读）

| 位置 | 事实 | 对本模块的影响 |
|---|---|---|
| `backend/app_http.go:39` | `createRateLimiter := middleware.NewRateLimiter(10, time.Minute, rt.transientStore)` | **全站唯一一个 limiter 实例** |
| `backend/middleware/ratelimit.go:20-30` | `NewRateLimiter(limit, window, store)`：`store == nil` → **自建一个私有 `state.NewMemoryStore()`** | ⇒ 每个 `NewRateLimiter(..., nil)` 都是**独立计数桶**（P0-2 的修法基础） |
| `backend/middleware/ratelimit.go:32` | `func (rl *RateLimiter) Allow(ip string) bool`（**已导出**） | §3.4.4 的模块内限流中间件直接调它取桶 |
| `backend/middleware/ratelimit.go:47-49` | `Middleware()` 用 `c.ClientIP()` 作 key | **`c.ClientIP()` 可被客户端伪造**（§3.4.4）→ 本模块**不用**它 |
| `backend/middleware/ratelimit.go:50-53` | 超限响应 `429 {"error":"请求过于频繁，请稍后再试","code":429}` | §3.2.1 / §3.2.2 错误码表 |
| `backend/middleware/ratelimit.go:63-75` | `ContentSizeLimiter`：`ContentLength > maxSize` → `413 {"error":"内容大小超过限制","code":413}`（**绑定前拦截**） | §3.2.1 M3 的 **413** 契约 |
| `backend/state/transient.go:129` / `:242` | Memory：`rateRequests[key]`；Redis：`s.key("ratelimit:" + key)` | **Redis 下所有 limiter 共用 `ratelimit:<ip>` 一个计数桶** |
| `backend/models/paste.go:245` | `generateID(length)` = `hex.EncodeToString(rand.Read(bytes))[:length]` | `generateID(32)` = **32 个 hex 字符 = 128 bit 熵**（crypto/rand） |
| `backend/models/paste.go:71` | `sql.Open("sqlite3", dbPath+"?_parse_time=true")` | `updated_at` 的解析口径 |
| `backend/models/paste.go:235` | `func (db *DB) SetMaxOpenConns(n int)` | `:memory:` 测试可直接调（R5） |
| `backend/middleware/request_logger.go:55` | `Path: truncateMonitoringValue(c.Request.URL.Path, 512)` | **完整 URL path 落 `http_request_logs` 表 → token 会入库** |
| `backend/middleware/request_logger.go:73-87` | skip 列表：`/api/monitor` `/api/health` `/assets/` `/neon/` `/api/bg/cached/` | **不含 `/ss/`** |
| `backend/app_http.go:31` | CORS `AllowHeaders` 含 `X-Super-Admin-Password` | 管理端用 header 传密码可直接用，无需改 CORS |
| `backend/app_http.go:35` | 全局 `ContentSizeLimiter(200 * 1024 * 1024)` | **200MB 请求体上限**；超限响应体见 §3.2.1 |
| `backend/cleanup.go:22` | 每小时清理 goroutine | 见 §4.3 —— 本模块**不接入** |
| `backend/handlers/skills_db_test.go:24` | `newMemDB(t)` 已含 `db.SetMaxOpenConns(1)` | R5 已满足，`ssrule_test.go` 直接复用即可 |
| `backend/handlers/monitoring.go:246` / `:251` | 项目里 query 兜底的**既有约定名**是 `?super_admin_password=`（`monitoring`）；`console`/`nps`/`proxy`/`paste` 用 `?admin_password=` | **本模块两种都不用**（裁决 5）—— 只认 header |
| `frontend/src/composables/useAdminAuth.js` | 参数是 **`headerName`**，不是 `credentialField` | 上游 Vue 传错了参数名（见 §5.3 P1-5） |
| `frontend/src/router/index.js:11` | `other: { name: '系统管理', icon: 'Tools', order: 70 }` | 本模块归 `category: 'other'` |
| `frontend/src/composables/useToolRegistry.js:13` | 工具清单由 `toolRoutes` 自动派生 | **无需**额外注册表登记 |
| `frontend/src/router/index.js:112,225,236` | `icon: 'Share'` 已被 3 处使用 | 图标名合法 |
| `backend/routes/autodev.go:20-24` | 独立 limiter 的**写法先例**（但它最终又回落到了全局 limiter，见 §5.3 P0-2 分析） | 参考写法，不照抄回落逻辑 |

---

## 3. 设计

### 3.1 模块定位

> 全站只有**一份** Shadowsocks 分流规则文件。管理页填域名 / URL 列表 → 生成 ACL 正文 → 公开链接分发给客户端作远程规则源。

**不做**规则引擎、不做多套规则、不做规则类型建模。存两份东西：

- `source` —— 用户原始输入（原样回显到编辑框，让用户下次还能改）
- `content` —— 按 ACL 模板生成好的正文（公开链接直接吐这个）

### 3.2 API 契约（全部对外端点）

#### 3.2.1 管理端（需密码）

鉴权：**只认** `X-Super-Admin-Password: <ssrules.admin_password>` 请求头。

> **query 兜底已删除**（FINN-102 裁决 5）。`?admin_password=` 与 `?super_admin_password=` **一律忽略** —— 不报错、不回退、不参与鉴权。
> **理由**：密码进 URL ⇒ ① Cloudflare 访问日志明文 ② 浏览器历史明文 ③ `Referer` 头明文外泄。密码是凭据，不进 URL。
> **对照**（仅供说明，本模块不采用）：`backend/handlers/monitoring.go:251` 用的是 `?super_admin_password=`；`console` / `nps` / `proxy` / `paste` 用 `?admin_password=`。**本模块两种都不用。**
> 上游实现在 `backend/handlers/ssrule.go:59-62`（`checkAdmin` 里的 `c.Query("admin_password")` 兜底）—— **Stage 4 必须删掉那 3 行**。

> **`ssrules.admin_password` 为空 ⇒ 整个模块关闭**（不是"无密码放行"）。实现位置：`SSRuleHandler.checkAdmin`（`backend/handlers/ssrule.go:55`，上游 commit 内）。
> 该判定必须**同时**作用于管理端与分发端 —— 见 §5.3 P0-3。

| # | 方法 | 路径 | 请求体 / 查询 | 成功响应 | 状态码 |
|---|---|---|---|---|---|
| M1 | `POST` | `/api/ssrules/verify` | 无（仅鉴权头） | `200 {"ok":true}` | 401 `{"error":"密码错误"}` / 429 `{"error":"请求过于频繁，请稍后再试","code":429}` |
| M2 | `GET` | `/api/ssrules` | 无（仅鉴权头） | `200` ssRuleView（§3.2.3） | 401 / 429 / 500 `{"error":"读取规则文件失败"}` |
| M3 | `POST` | `/api/ssrules` | `{"source":"<多行文本>"}` | `200` ssRuleView | 400 `{"error":"请求格式错误"}` / 400 `{"error":"输入过长(超过 512 KB),请分批处理"}` / **413 `{"error":"内容大小超过限制","code":413}`** / 401 / 429 / 500 `{"error":"保存失败"}` |
| M4 | `POST` | `/api/ssrules/regenerate-token` | 无（仅鉴权头） | `200` ssRuleView（token 已换新；**`skipped` 为重算后的实际值**，见 §3.2.3） | 401 / 429 / 500 `{"error":"换发失败"}` |

**413 与 400 必须分开处理（FINN-102 非阻塞项 3）**：

| 状态码 | 触发条件 | 前端表现 |
|---|---|---|
| **400** `{"error":"请求格式错误"}` | body 不是合法 JSON / `source` 字段类型不对 | 提示"格式错误"，引导用户改内容 |
| **400** `{"error":"输入过长(超过 512 KB),请分批处理"}` | `source` 超过 `ssRuleMaxSourceSize`（512 KB） | 提示"内容过长，分批粘贴" |
| **413** `{"error":"内容大小超过限制","code":413}` | 请求体超过 `/ssrules` 组的 1MB 上限（`ContentSizeLimiter(1<<20)`，**在绑定前拦截**） | 提示"请求体过大"，**不是**"格式错误" |

> 413 由 `backend/middleware/ratelimit.go:63-75` 的 `ContentSizeLimiter` 产生，**在 `ShouldBindJSON` 之前**返回 —— 因此请求体根本没进业务逻辑。前端若把 413 当 400 显示"格式错误"，用户会反复重试同一个必然失败的请求。
> ⚠️ `ContentSizeLimiter` 只看 `c.Request.ContentLength`（`ratelimit.go:65`）—— **chunked 传输（`ContentLength == -1`）不受限**。这是既有中间件的行为，本单不修（S9 越界）；但**前端必须显式设置 `Content-Length`**（`fetch` 带 body 时默认就是定长，OK）。

#### 3.2.2 公开分发端（token 门，无密码）

| # | 方法 | 路径 | 鉴权 | 成功响应 | 状态码 |
|---|---|---|---|---|---|
| P1 | `GET` | `/ss/:token/:filename` | `token` 匹配 DB 中唯一行 | `200` `text/plain; charset=utf-8`，body = ACL 正文 | 404 `规则文件不存在`（纯文本，非 JSON） / 404（模块关闭时，见 P0-3） / **429 `{"error":"请求过于频繁，请稍后再试","code":429}`** / 500 `读取规则文件失败` |

**响应头**（P1）：

```
Content-Type: text/plain; charset=utf-8
Cache-Control: no-store
X-Content-Type-Options: nosniff
```

- `Cache-Control: no-store` —— 已在上游实现。**必须有**：否则用户改完规则客户端拉不到新内容，"改了规则没生效"是极难排查的问题。
- `X-Content-Type-Options: nosniff` —— **采纳（FINN-102 非阻塞项 10）**。上游未设，成本一行，防浏览器 MIME 猜测。
- **不设 `Content-Disposition`** —— 见 §3.4.3。

**`:filename` 白名单**（唯一允许的 4 个值，硬编码常量）：

```
rules.acl   rules.txt   rules.list   rules.conf
```

> 刻意**不含** `.pac`：PAC 是 JavaScript，把 ACL 正文当 `.pac` 发出去，客户端会静默拿到语法错误的脚本，报错完全指不到这里。

**所有"不该存在"的形态共用同一个响应体**（Stage 4 必须把字符串收敛成**一个常量**，避免各处拼错）：

```go
// backend/handlers/ssrule.go（上游 commit 内新增）
// ssRuleNotFoundBody 是本模块所有 404 的唯一文案。
// 白名单拒绝 / token 不存在 / 路径形态不对 / 模块关闭 —— 四者必须逐字节一致，
// 否则外部观察者能区分「哪种情况」，泄露状态。
const ssRuleNotFoundBody = "规则文件不存在"
```

#### 3.2.3 管理端统一响应体 `ssRuleView`

```jsonc
{
  "source":     "google.com\nyoutube.com",        // 用户原始输入，回显到编辑框
  "content":    "# Shadowsocks 分流规则 …",          // 生成的 ACL 正文
  "token":      "3f2a…(32 hex)",                   // 当前分发 token
  "updated_at": "2026-10-22T04:34:56.789Z",        // RFC3339，**统一 UTC**（mattn + _parse_time）
  "paths": [                                        // 相对路径，前端拼 origin
    "/ss/<token>/rules.acl",
    "/ss/<token>/rules.txt",
    "/ss/<token>/rules.list",
    "/ss/<token>/rules.conf"
  ],
  "skipped": [                                      // 逐行跳过原因（不静默丢弃）
    "第 3 行 中文域名.com —— 含非 ASCII 字符,请先转成 punycode(如 xn----dqo34k.com)"
  ]
}
```

- `paths` **返回相对路径**，由前端拼 `window.location.origin` —— 服务端不知道用户是从哪个域名访问的（可能走 CF、可能走内网），拼死反而给错链接。
- `skipped` 恒为数组（空时 `[]`，不是 `null`）—— 见 `newSSRuleView` 的 nil 归一。
- **`skipped` 在 M2（GET）与 M4（换发）里都必须是"实际重算值"**（FINN-102 非阻塞项 11，**采纳**）：上游 `RegenerateToken` 调的是 `newSSRuleView(file, nil)`（`backend/handlers/ssrule.go:175`）⇒ `skipped` 恒 `[]`；而 `Get`（`:125`）会重跑 `buildACLRules(file.Source)` 并填真实跳过原因。**两者语义不一致**。修法：换发 token 时也重跑一次 `buildACLRules(file.Source)`，把算出的 `skipped` 传进 `newSSRuleView`（换发不改 `source`，重算结果必然与 `Get` 一致）。
- `updated_at` 序列化形态由 `mattn/go-sqlite3` + `?_parse_time=true`（`backend/models/paste.go:71`）决定 → RFC3339 字符串 → 浏览器 `new Date()` 可解析。**QA 必须断言这一点**（若退化成 `0001-01-01T00:00:00Z`，页面会显示 1/1/1）。
- **时区口径（FINN-102 裁决 6）= 统一 UTC**：
  - 写入用 `time.Now().UTC()`；与表 DDL 的 `DEFAULT CURRENT_TIMESTAMP` 语义一致（SQLite 的 `CURRENT_TIMESTAMP` 本身就是 **UTC**），也与 `?_parse_time=true` 的解析口径一致；
  - 前端用 `toLocaleString()` 渲染成浏览器本地时区（**渲染层转换，存储层不转换**）；
  - 契约与示例里**不再出现** `+08:00` 这类非 UTC 偏移。
  - **QA 断言**：`time.Parse(time.RFC3339, v)` 成功，且解析出的时间戳的**时区偏移为 0**（即尾部是 `Z` 而不是 `+08:00`）。

### 3.3 token 语义（明确回答）

| 维度 | 结论 | 判据 / 实现位置 |
|---|---|---|
| **可枚举？** | **不可**。`generateID(32)` = `crypto/rand` 取 32 字节 → hex 截 32 字符 = **128 bit 熵** | `backend/models/paste.go:245`；调用点 `backend/models/ssrule.go` 的 `GetSSRuleFile` / `RegenerateSSRuleToken`（上游 commit 内） |
| **有时效？** | **无**。token 一旦生成长期有效，直到手工换发 | 无 TTL 字段（表里没有 expires_at） |
| **一次性？** | **否**。同一个 token 可被任意客户端反复拉取 | `GET /ss/:token/:filename` 无状态校验 |
| **可换发？** | **可以**。`POST /api/ssrules/regenerate-token` 生成新 token，**旧链接立即 404** | `models.RegenerateSSRuleToken`（UPDATE token）+ 换发后旧 token 查不到行 → `sql.ErrNoRows` → 404 |
| **权限范围** | **只读**。token 不能编辑。编辑一律走 M1-M4 的密码门 | `checkAdmin` **只认** `X-Super-Admin-Password`（query 兜底已删除，§3.2.1），**不认 token** |
| **token 是否入库明文** | 是（单行表）。这是本方案显式接受的风险（§5.3 P2-2） | `ss_rule_files.token TEXT` |

**失败响应不得泄露 token 是否存在** —— 判据：

- token 不存在 → `404` + body 恰为 `规则文件不存在`
- filename 不在白名单 → `404` + body 恰为 `规则文件不存在`（**同一句、同一状态码**）
- **路径根本没匹配到路由**（如 `/ss/TOK/../../etc/passwd`、尾斜杠形态）→ NoRoute 特判 → `404` + body 恰为 `规则文件不存在`（§3.4.1）
- ⇒ 外部观察者无法区分「token 错」「文件名错」「路径不匹配」，也区分不了三者与「模块关闭」（P0-3 修好后模块关闭也回同一句）。

### 3.4 公网暴露安全（本单最高优先级）

#### 3.4.1 `:filename` 路径穿越防护（**FINN-102 v3 重写**）

> **本节按 FINN-90 的 B1 判定 + PM 裁决 2 / 3 重写。** v1 里"解码后成为**单个** `:filename` 段"的机制解释**是错的**，载荷表的期望也只对了一半。

##### (a) Gin 的真实匹配机制（订正 v1 的错误论断）

| 事实 | 证据（gin v1.9.1，module cache 实读） |
|---|---|
| `UseRawPath` 默认 **false** | `gin.go:198`（`UseRawPath: false`） |
| 路由匹配用**解码后**的 `c.Request.URL.Path` | `gin.go:594`（`rPath := c.Request.URL.Path`）；`gin.go:596-599` 只有在 `UseRawPath == true` 时才改用 `URL.RawPath` |
| `RedirectTrailingSlash` 默认 **true** | `gin.go:192` |
| tsr 命中时 **301** 到去掉尾斜杠的形态 | `gin.go:625-628` → `redirectTrailingSlash`（`gin.go:671-686`） |
| catch-all 路由（`/x/*name`）的 param 值**含前导 `/`** | `tree.go:544-566`；gin 自带用例 `tree_test.go`：`"/src/some/file.png"` → `filepath = "/some/file.png"` |

**由此得到的正确推论（v1 说反了）**：

- `%2f` 在 **HTTP 解码阶段**就变成了真 `/` ⇒ `/ss/TOK/..%2f..%2fetc%2fpasswd` 的 `URL.Path` = `/ss/TOK/../../etc/passwd` ⇒ **段数变多**（6 段）；
- 路由 `/ss/:token/:filename` 只有 **3 段** ⇒ **根本不匹配** ⇒ 走 **NoRoute**；
- ⇒ **不是**"变成单个 `:filename` 值"。v1 那句"解码后可能成为单个 `:filename` 段的 `../../etc/passwd`"**作废**（⛔）。

**所以 v1 载荷表里 ≥5/10 行的期望与真实行为不符**（真值见下表，已实测，见 §9.9）。

##### (b) 安全论据（**保留，且这两条本来就成立**）

实现位置：`backend/handlers/ssrule.go` → `ssRuleIsAllowedFile(name string) bool`（上游 `:42`）。

**判据（三重，任何一条不满足即 404）**：

1. **白名单精确相等**（不是前缀、不是后缀、不是 `strings.Contains`）：`name` 必须**逐字节等于** `rules.acl` / `rules.txt` / `rules.list` / `rules.conf` 之一。
2. **不触碰文件系统**：分发端点全程**不**做任何 `os.Open` / `filepath.Join` / `c.File`。→ 即使白名单被绕过，也没有任何文件读取面。
3. **token 只进参数化 SQL**：`db.conn.QueryRow("… WHERE token = ?", token)` → 无 SQL 注入面。

> ⛔ **不要**再写"依赖 `..%2f` 解码后不含 `/`"这种论据 —— 它**不成立**（解码后含 `/`）。安全**只**由上面 ① 白名单 ② 不碰文件系统 两条支撑。

##### (c) 载荷表（**10 行**，期望**统一**为 `404` + body 逐字节 `规则文件不存在`）

| # | 载荷（拼在 `/ss/<token>/` 之后） | 落点 | 期望响应 |
|---|---|---|---|
| 1 | `../../etc/passwd` | NoRoute（6 段不匹配） | `404` + `规则文件不存在` |
| 2 | `..%2f..%2fetc%2fpasswd` | NoRoute（解码后同上） | `404` + `规则文件不存在` |
| 3 | `%2e%2e%2f%2e%2e%2fpasswd` | NoRoute（解码后 5 段） | `404` + `规则文件不存在` |
| 4 | `....//....//etc/passwd` | NoRoute（6 段） | `404` + `规则文件不存在` |
| 5 | `rules.acl%00.txt` / `rules.acl%00` | handler（白名单拒绝） | `404` + `规则文件不存在` |
| 6 | `rules.acl.txt`（双扩展名） | handler（白名单拒绝） | `404` + `规则文件不存在` |
| 7 | `rules.acl/`（**尾斜杠**） | **显式占位路由**（§(d)） | `404` + `规则文件不存在` |
| 8 | `rules.acl\..\x` | handler（白名单拒绝） | `404` + `规则文件不存在` |
| 9 | `rules.pac` / `index.html` / `rules.yaml` / `rules` | handler（白名单拒绝） | `404` + `规则文件不存在` |
| 10 | 空 filename ⇒ URL `/ss/<token>/` | **显式占位路由**（§(d)） | `404` + `规则文件不存在` |

> **上表的前提**：§(d) 的 **NoRoute 特判 + 3 条占位路由**必须实现。**不实现的话，NoRoute 会走 SPA fallback（`backend/app_http.go:84-102`）⇒ 200 + `index.html`** —— 这正是 v1 载荷表 5/10 行对不上的原因，也是 PM 裁决 2 点名要治的。

> **载荷计数（断言要精确到条，否则不是可判定断言）**：上表 **10 行**展开为 **14 个具体载荷** ——
> 第 5 行 2 个（`rules.acl%00.txt` / `rules.acl%00`）、第 9 行 4 个（`rules.pac` / `index.html` / `rules.yaml` / `rules`），其余各行 1 个。
> ⇒ **handler 级注入 14 个 filename 值、HTTP 级发 14 条 URL**（§6.3 A3 / A3-HTTP）。

##### (d) 让"未匹配 `/ss/` 路径"回 404 的**两处必做改动**（PM 裁决 2 / 3）

**改动 1 —— NoRoute 里特判 `/ss/` 前缀**（`backend/app_http.go` → `registerStaticRoutes` → `router.NoRoute(...)`，**在 `/api` 特判（`:80-83`）之后**）：

```go
router.NoRoute(func(c *gin.Context) {
	path := c.Request.URL.Path
	if path == "/api" || strings.HasPrefix(path, "/api/") {
		c.JSON(http.StatusNotFound, gin.H{"error": "接口不存在", "path": path})
		return
	}

	// ★ FINN-102 裁决 3：/ss/ 前缀一律 404「规则文件不存在」。
	// 不加这条 → 未匹配路径会落进下面的 SPA fallback，客户端拿到 200 + index.html，
	// 把一坨 HTML 当规则文件用 —— 静默失败，比 404 更难排查。
	if strings.HasPrefix(path, "/ss/") {
		c.String(http.StatusNotFound, handlers.SSRuleNotFoundBody)
		return
	}

	// …原有 SPA fallback 不变…
})
```

- 响应体与 handler 白名单拒绝的响应**逐字节一致**（同一句 `规则文件不存在`、同一状态码 `404`、`text/plain`）。
- **不影响**其它路径的 SPA fallback 行为（分支只拦 `/ss/` 前缀）。
- ⚠️ **边界**：裸 `/ss`（无尾斜杠）**不匹配** `HasPrefix(path, "/ss/")` ⇒ 仍走 SPA fallback（200 + HTML）。实测确认它**不会 301**（§9.9）。它**不泄露任何规则数据**，只是形态不一致。若要一并收口，加一条 `path == "/ss"` 判断即可（本方案**默认不加**，严格照 PM 的 `HasPrefix("/ss/")` 口径；见 §8 Q7）。

**改动 2 —— 关闭 `/ss/` 前缀的 tsr 自动重定向**（尾斜杠载荷落到同款 404）：

gin 默认 `RedirectTrailingSlash = true`（`gin.go:192`）⇒ `/ss/TOK/rules.acl/` 会 **301** 到 `/ss/TOK/rules.acl`（`gin.go:625`）⇒ **不是 404**。

> **为什么不能靠"多注册一条尾斜杠路由"糊过去**：`/ss/:token/:filename/` 匹配到的 `c.Param("filename")` 仍是 `"rules.acl"`（尾斜杠是独立的空段，不进 param）⇒ 若把它指向 `Download`，会**返回 200 + ACL 正文** —— 比 301 更糟。**必须指向独立的 404 handler。**

**修法（不改 gin 全局设置）**：在 `registerPublicRoutes` 里**显式注册**尾斜杠 / 空 filename / 少一段的形态，各配一个直返 404 的 handler。**已实测（§9.9）：这 4 条路由可以共存，gin v1.9.1 不 panic，且所有 `/ss/` 形态都不再产生 3xx。**

```go
// backend/app_http.go（package main）
// registerSSRulePublicRoutes 注册 SS 规则文件的公开分发路由。
//
// 为什么要拆成独立函数：Stage 5 的 HTTP 级断言（§6.3 A3-HTTP）要在
// package main 的测试里复用**同一份注册逻辑**，否则测试里手抄一遍路由
// 就等于没有测到真实接线（漂移风险）。
func registerSSRulePublicRoutes(router *gin.Engine, h *handlers.SSRuleHandler) {
	// 正常分发端点。必须显式注册:非 /api 前缀的路径没注册就会落到 NoRoute
	// 被 SPA fallback 兜成 index.html,客户端拿到的是一坨 HTML,
	// 而报错信息完全指不到这里。
	router.GET("/ss/:token/:filename", h.Download)

	// ★ FINN-102 裁决 3：显式注册尾斜杠 / 空 filename / 少一段的形态并直返 404，
	// 用来"关闭" /ss/ 前缀的 RedirectTrailingSlash 自动重定向。
	// 不注册的话 gin 会 301 到无尾斜杠形态(gin.go:192/625)，
	// 尾斜杠载荷就拿不到同款 404。
	router.GET("/ss/:token/:filename/", h.NotFound)
	router.GET("/ss/:token/", h.NotFound)
	router.GET("/ss/:token", h.NotFound)
}
```

```go
// backend/handlers/ssrule.go（上游 commit 内新增）
// NotFound 是 /ss/ 前缀下所有"不该存在"的形态的统一响应。
// 与 Download 内部的白名单拒绝 / token 不存在 / 模块关闭响应逐字节一致。
func (h *SSRuleHandler) NotFound(c *gin.Context) {
	c.String(http.StatusNotFound, ssRuleNotFoundBody)
}
```

> **响应体常量必须导出**（`handlers.SSRuleNotFoundBody`），因为 `backend/app_http.go` 的 NoRoute 分支也要用同一句 —— 两处各拼一个字符串字面量，早晚拼错。

##### (e) 断言分两层（FINN-90 B1 第 2 条要求）

| 层 | 做法 | 断言 | 覆盖 |
|---|---|---|---|
| **handler 级** | 单测里**注入** `c.Params`（**不经过 gin 路由**），直接调 `h.Download` | 逐条：`404` 且 body **逐字节等于** `规则文件不存在` | 上表展开后的 **14 个载荷全部**（含第 1–4、7、10 行 —— 这些在 HTTP 层根本到不了 handler，但**白名单函数本身**必须对它们免疫） |
| **HTTP 级** | 用 `httptest` + **真实的 `registerSSRulePublicRoutes` + `registerStaticRoutes`**（package main 测试）发请求 | 逐条：`404` 且 body **逐字节等于** `规则文件不存在`；**且** body 不含 ACL 正文特征（如 `[proxy_list]`）；**且** 状态码 **不是 5xx**、**不是 3xx** | 上表展开后的 **14 条 URL** + `/ss` 裸路径 |

- 两层**都要做**：handler 级保证"白名单函数本身正确"，HTTP 级保证"路由 + NoRoute 的落点正确"。v1 只写了一条混合断言，导致 ≥5/10 行在真实行为下必红。
- 对应断言编号：**§6.3 A3（handler 级）+ §6.3 A3-HTTP（HTTP 级）**。
- HTTP 级测试的骨架（Stage 5 照抄）：

```go
// backend/app_http_ssrule_test.go（package main）
func TestSSRulePublicRouteShapes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := models.NewDB(":memory:")
	if err != nil { t.Fatal(err) }
	db.SetMaxOpenConns(1)          // R5：:memory: 必须单连接
	if err := db.InitSSRuleFiles(); err != nil { t.Fatal(err) }

	r := gin.New()
	registerSSRulePublicRoutes(r, handlers.NewSSRuleHandler(db, "pwd"))
	registerStaticRoutes(r, t.TempDir())   // ← 真实的 NoRoute（含 /ss/ 特判）

	for _, p := range ssRulePayloads {    // §3.4.1(c) 的 10 行
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p, nil))
		// 断言：w.Code == 404 && w.Body.String() == "规则文件不存在"
	}
}
```

#### 3.4.2 `:token` 的生成与校验强度

| 项 | 实现 | 结论 |
|---|---|---|
| 生成 | `generateID(32)`（`backend/models/paste.go:245`，`crypto/rand` + hex） | 128 bit 熵 ✅ |
| 校验 | 参数化 SQL 精确匹配 | 无注入 ✅ |
| 失败返回 | 404 + `规则文件不存在` | 不泄露存在性 ✅ |
| 时序侧信道 | 命中/未命中都是一次 `QueryRow`（µs 级）；token 不可枚举 → 时序差无意义 | ✅ |

#### 3.4.3 响应头 / 缓存策略

| 头 | 现状 | 判据 |
|---|---|---|
| `Content-Type` | `text/plain; charset=utf-8`（常量） | ✅ 不是 `text/html` → 浏览器不会把正文当 HTML 执行 |
| `Cache-Control` | `no-store`（常量） | ✅ 客户端必须每次拿到新规则 |
| `Content-Disposition` | **不设置** | ✅ **无注入面**：filename 是白名单常量，不参与任何 header 拼接。**若将来要加，只能拼常量**（`attachment; filename="rules.acl"`），**严禁** `c.Param("filename")` 直接入 header |
| `Vary` | 不需要（分发端无鉴权头） | ✅ |
| `X-Content-Type-Options` | 上游**未设** → **本单采纳，加 `nosniff`**（非阻塞项 10） | ✅ 防浏览器 MIME 猜测 |
| `Content-Length` | 由 Gin 自动设 | ✅ |

**信息泄露面**（P2，显式记录）：

| 面 | 是否含 token | 缓解 |
|---|---|---|
| Cloudflare 访问日志 | **是**（CF 默认记完整 URL） | 不可控；靠换发止损 |
| 应用 `http_request_logs` 表 | **是**（`request_logger.go:55` 记完整 path） | P1-2：把 `/ss/` 加入 skip 列表 |
| 浏览器历史 / Referer | 是（用户点过链接时） | 靠换发止损 |
| 客户端配置文件 | 是（客户端本地存了规则 URL） | 靠换发止损 |

#### 3.4.4 分发端与管理端的限流（**FINN-102 v3 重写 —— B2 闭环**）

##### (a) 为什么不能用现成的 `middleware.RateLimiter.Middleware()`

现成中间件的键是 `c.ClientIP()`（`backend/middleware/ratelimit.go:49`）。**这个键完全由客户端控制**：

| 环节 | 事实 | 证据 |
|---|---|---|
| gin 默认信任所有代理 | `trustedProxies: []string{"0.0.0.0/0", "::/0"}` | `gin.go:205` |
| gin 默认按 XFF 取客户端 IP | `ForwardedByClientIP: true` | `gin.go:195` |
| gin 默认读哪些头 | `RemoteIPHeaders: []string{"X-Forwarded-For", "X-Real-IP"}` | `gin.go:196` |
| 全仓**没有**任何 `SetTrustedProxies` 调用 | ⇒ 上面三个默认值全部生效 | `grep -rn "SetTrustedProxies" backend/` → **0 命中** |
| `validateHeader` 从右往左遍历 | 因 `0.0.0.0/0` 让每个 IP 都算 trusted ⇒ 命中 `i == 0` ⇒ 返回 **XFF 最左边**那一项 | `gin.go:458-477` |

⇒ 客户端自带 `X-Forwarded-For: <任意>` 即取得任意限流桶 key ⇒ **轮换该头即可无限绕过**；反过来不经 CF 时所有请求聚到反代 IP 一个桶。

> ⛔ v1 §3.4.4 只写了"要给分发端一个独立预算"，**没写键从哪来** —— 这是 B2 判 FAIL 的根因。v3 补齐。

##### (b) 键的取值顺序（**PM 裁决 1，照此实现**）

```
1. CF-Connecting-IP     （Cloudflare 覆写，客户端伪造无效）
2. X-Forwarded-For 最右一项（代理追加的那一项；不是最左）
3. c.Request.RemoteAddr 的 host 部分
```

**代码（Stage 4 照抄，落在 `backend/routes/ssrule.go` 或 `backend/handlers/ssrule.go`，二选一但只写一份）**：

```go
// ssRuleClientIP 取本模块的限流键。
//
// 不用 gin 的 c.ClientIP():gin 默认 trustedProxies = ["0.0.0.0/0","::/0"]
// (gin.go:205) + ForwardedByClientIP=true (gin.go:195) +
// RemoteIPHeaders ["X-Forwarded-For","X-Real-IP"] (gin.go:196),
// validateHeader (gin.go:458) 于是直接返回 XFF 最左边那一项 ——
// 客户端自带 X-Forwarded-For 就能任意伪造限流桶 key,轮换该头即无限绕过。
//
// 现在的取法:CF 覆写过的头最可信,其次取代理追加的 XFF 最右一项,
// 最后退到直连来源地址。
func ssRuleClientIP(c *gin.Context) string {
	if v := strings.TrimSpace(c.GetHeader("CF-Connecting-IP")); v != "" {
		return v
	}
	if v := strings.TrimSpace(c.GetHeader("X-Forwarded-For")); v != "" {
		parts := strings.Split(v, ",")
		return strings.TrimSpace(parts[len(parts)-1]) // 最右一项 = 代理追加的那一项
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}
```

##### (c) 模块内自建限流中间件（**不动共享 `middleware.RateLimiter`**）

```go
// ssRuleLimiter 是本模块专用的限流中间件。
//
// 为什么不用共享的 middleware.RateLimiter.Middleware():
//   ① 它的键是 c.ClientIP(),可被客户端伪造(见 ssRuleClientIP 的注释);
//   ② 它是全站唯一实例(app_http.go:39),共用计数桶。
//
// 共享类型只借用它的"桶"逻辑:Allow(ip) 已导出(ratelimit.go:32),
// 每个 NewRateLimiter(..., nil) 自建一个私有 MemoryStore(ratelimit.go:20-23)
// ⇒ 天然独立计数桶。
func ssRuleLimiter(limit int, window time.Duration) gin.HandlerFunc {
	rl := middleware.NewRateLimiter(limit, window, nil) // nil ⇒ 独立 store
	return func(c *gin.Context) {
		if !rl.Allow(ssRuleClientIP(c)) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "请求过于频繁，请稍后再试",
				"code":  429,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
```

##### (d) 分桶与阈值（**PM 裁决 2：按端点分桶**）

| 端点 | 限流 | 桶 |
|---|---|---|
| `POST /api/ssrules/verify`（登录） | **10 / 分钟** | 独立 |
| `POST /api/ssrules/regenerate-token` | **10 / 分钟** | 独立 |
| `GET /api/ssrules`（读） | **30 / 分钟** | 独立 |
| `POST /api/ssrules`（保存） | **20 / 分钟** | 独立 |
| `GET /ss/:token/:filename`（分发） | **120 / 分钟** | 独立 |

> 5 个桶**各自独立计数**（各自 `NewRateLimiter(..., nil)` ⇒ 各自私有 MemoryStore）。管理端 4 个端点**不共用一个桶** —— v1 的"4 端点共用 10/min"会让正常保存操作被自己的读操作挤掉（非阻塞项 2）。

##### (e) 残余风险（**PM 裁决 1 明示接受，不算缺陷**）

源站直连（不经 CF）时 `CF-Connecting-IP` 可被伪造。本项目对外拓扑**只有 Cloudflare**（`t.jaxiu.cn`，`docker-compose.yml` 由 jaxiu 手工部署），故接受该残余风险。**无 CF 时退 XFF 最右项（不是最左项）** —— 比现状（最左项，完全可伪造）已显著收紧。

#### 3.4.5 「未配密码 = 模块关闭」语义一致性

**这条必须落到端点行为上，不能只停留在管理端。**

| 场景 | 管理端 M1-M4 | 分发端 P1 |
|---|---|---|
| `admin_password` **已配置** | 密码正确 → 正常；错误 → 401 | token 正确 → 200；错误 → 404 |
| `admin_password` **为空**（模块关闭） | **全部 401**（`checkAdmin` 首行 `if h.adminPassword == "" { return false }`） | **必须 404**（`规则文件不存在`）—— ⚠️ **上游未实现，这是 P0-3** |

> **为什么这条重要**：如果用户先用密码存了规则，之后把 `admin_password` 清空以为"模块关了"，但 `/ss/<旧token>/rules.acl` 仍在吐旧内容 —— 公网上任何拿到链接的人照旧能拿到。**配置层面的"关闭"必须等于数据层面的"关闭"。**

#### 3.4.6 其余公网面

- `/api/ssrules*` 全部在 `/api` 分组下 → NoRoute 对 `/api/` 前缀返回 JSON 404，不会被 SPA fallback 兜住 ✅
- 分发端在 `/ss/` 前缀 → 显式注册 ✅；未匹配的 `/ss/` 路径 → NoRoute 特判 → 404 ✅（§3.4.1(d)）
- 不返回内部拓扑 / 服务名 / 版本号 / 依赖健康状态 ✅（ssrule 响应只含规则内容与 token）
- 管理端响应含 `token`（= 分发凭据），但仅在密码正确时返回 ✅

### 3.5 数据模型

**新增表**（`backend/models/ssrule.go`，上游 commit 内）：

```sql
CREATE TABLE IF NOT EXISTS ss_rule_files (
    id         TEXT PRIMARY KEY,                        -- 恒为 'default'（全站单行）
    token      TEXT NOT NULL,                           -- 分发 token，32 hex
    source     TEXT DEFAULT '',                         -- 用户原始输入
    content    TEXT DEFAULT '',                         -- 生成的 ACL 正文
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
-- ★ P1-1：补 token 唯一索引（上游没有）
CREATE UNIQUE INDEX IF NOT EXISTS idx_ss_rule_files_token ON ss_rule_files(token);
```

**唯一键**：

| 键 | 列 | 说明 |
|---|---|---|
| PRIMARY KEY | `id` | 固定值 `"default"`（`ssRuleFileID` 常量，上游 `backend/models/ssrule.go:14`）→ 全站单行 |
| UNIQUE INDEX | `token` | **P1-1 需补**；上游只有 PK，token 列无唯一约束（单行场景无实际危害，但语义上 token 应唯一） |

**生命周期**：

| 动作 | 触发 | 行为 |
|---|---|---|
| 建表 | `models.RegisterInit("Shadowsocks规则文件(ss_rule_files)", (*DB).InitSSRuleFiles)`（上游 `backend/models/ssrule.go:9`）→ `InitAll()` 遍历执行（`backend/models/paste.go:48` `RegisterInit`） | `CREATE TABLE IF NOT EXISTS` |
| 懒创建 | `GetSSRuleFile()` 查不到行时 `INSERT OR IGNORE` 写入空行 + 新 token | 并发首次请求用 `INSERT OR IGNORE` 吞掉竞争，失败方重读 |
| 覆盖保存 | `SaveSSRuleFile(source, content)` | UPDATE `source`/`content`/`updated_at` |
| 换发 token | `RegenerateSSRuleToken()` | UPDATE `token` + `updated_at`；**旧链接立即失效** |
| 删除 | **无**（无任何删除入口） | 见下 |

**`updated_at` 写入口径（裁决 6）**：统一 UTC。SQLite 的 `CURRENT_TIMESTAMP` 本身返回 UTC；**若代码里显式写时间，必须用 `time.Now().UTC()`**，不要用 `time.Now()`（本地时区）—— 否则库里出现带本地偏移的字符串，与 DDL 语义不一致。

**与每小时清理 goroutine 的对接**：`backend/cleanup.go:22` 的 `startCleanupRoutine` 每小时清过期粘贴板/聊天室/短链/Mock/MDShare/Excalidraw/… **本模块不接入清理**：

- 本表是**单行配置表**，不是有生命周期的事件数据 —— 没有"过期"语义
- 清理它等于"每小时把用户的规则文件删掉"，不可接受
- **⚠️ 下游注意**：`backend/cleanup.go` **不需要**任何改动。若有人"顺手"加一行 `db.CleanExpiredSSRuleFiles()` → 直接判 FAIL（无此方法，且语义错误）

**Redis 软依赖**：本模块**完全不依赖 Redis**。token 生成用 `crypto/rand`，存储用 SQLite，限流用进程内 MemoryStore（§3.4.4(c)）。⇒ Redis 挂掉 / 降级时本模块行为不变 ✅

### 3.6 路由接线

| 位置 | 改动 |
|---|---|
| `backend/routes/index.go:49` 之后 | `SSRuleHandler *handlers.SSRuleHandler`（大写导出结构体，`package routes`） |
| `backend/routes/index.go:91` 之后 | `RegisterSSRuleRoutes(api, h, createRateLimiter)` |
| `backend/routes.go`（`type routeHandlers struct`，`:12`） | 加 `ssRuleHandler *handlers.SSRuleHandler`（小写，`package main`） |
| `backend/routes.go`（`setupRoutes`，`:91` 附近） | `SSRuleHandler: h.ssRuleHandler,` 传给 `routes.RouteHandlers` |
| `backend/app_handlers.go:92` 之后 | `ssRuleHandler := handlers.NewSSRuleHandler(db, cfg.SSRules.AdminPassword)` |
| `backend/app_handlers.go:140` 之后 | `ssRuleHandler: ssRuleHandler,` |
| `backend/app_http.go:53` 之后 | `registerSSRulePublicRoutes(router, handlers.ssRuleHandler)` —— **4 条路由**（§3.4.1(d)） |
| `backend/app_http.go:80-83` 之后 | NoRoute 里加 `/ss/` 前缀特判（§3.4.1(d)） |
| `backend/routes/ssrule.go`（新文件） | `RegisterSSRuleRoutes` |

**两条易错点**（上游 commit message 已点出，必须在代码注释里保留）：

1. **`/ss/` 不能改成 `/s/`** —— `/s/:id` 已被短链占用（`backend/app_http.go:50`）。Gin 在同一位置遇到两个不同名的通配段会**启动即 panic**（wildcard conflict）。`/ss/` 首段不同，无冲突。
2. **必须显式注册进 `registerPublicRoutes`** —— 非 `/api/` 前缀的路径没注册就落 NoRoute → SPA fallback → 客户端拿到一坨 HTML，且报错完全指不到这里。

**`backend/routes/ssrule.go` 的骨架**（含 §5.3 的 P0/P1 修复 + 裁决 1/2 的限流）：

```go
func RegisterSSRuleRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	if h.SSRuleHandler == nil {
		return
	}
	// ★ 按端点分桶(PM 裁决 2)。每个 limiter 都是 NewRateLimiter(..., nil)
	// ⇒ 各自私有 MemoryStore ⇒ 互不干扰;也绝不回落到 createRateLimiter。
	verifyRL := ssRuleLimiter(10, time.Minute)    // 登录
	regenRL := ssRuleLimiter(10, time.Minute)     // 换发 token
	getRL := ssRuleLimiter(30, time.Minute)       // 读
	saveRL := ssRuleLimiter(20, time.Minute)      // 保存

	g := api.Group("/ssrules", middleware.ContentSizeLimiter(1<<20)) // P1-3
	{
		g.POST("/verify", verifyRL, h.SSRuleHandler.Verify)
		g.GET("", getRL, h.SSRuleHandler.Get)          // ★ P0-1：上游漏了限流
		g.POST("", saveRL, h.SSRuleHandler.Save)
		g.POST("/regenerate-token", regenRL, h.SSRuleHandler.RegenerateToken)
	}
}
```

> `createRateLimiter` 参数**保留但不使用**（签名与 `RegisterAllRoutes` 的既有调用点不变，S9 友好）；Stage 4 可加 `_ = createRateLimiter` 或直接改参数名为 `_` 以免"未使用参数"编译告警。**绝不能** `if createRateLimiter != nil { ... = createRateLimiter }` 那样回落（P0-2）。

### 3.7 前端

**文件**：

| 文件 | 动作 |
|---|---|
| `frontend/src/views/other/SsRuleTool.vue` | 新建（上游 244 行） |
| `frontend/src/router/index.js` | 插一条路由（`/proxy` 块之后，约 `:105` 后） |
| `frontend/src/views/other/SsRuleTool.test.js` | **新建**（上游 commit 没有前端测试 —— Stage 5 必须补，见 §6.4） |

**路由条目**（照抄上游，`frontend/src/router/index.js`）：

```js
{
  path: '/ssrules',
  name: 'SSRules',
  component: () => import('../views/other/SsRuleTool.vue'),
  meta: {
    title: 'SS 规则文件',
    icon: 'Share',            // 该图标名项目内已用 3 处（router/index.js:112,225,236）
    category: 'other',        // 「系统管理」分组（router/index.js:11）
    description: '生成 Shadowsocks 分流规则文件'
  }
}
```

**组件状态管理**：

| 状态 | 类型 | 来源 |
|---|---|---|
| `isAuthenticated` / `passwordInput` / `loggingIn` | composable 内部 | `useAdminAuth` |
| `source` | `ref('')` | 用户输入（回显 `data.source`） |
| `content` | `ref('')` | 生成的 ACL 正文 |
| `paths` | `ref([])` | 服务端返回的相对路径 |
| `skipped` | `ref([])` | 逐行跳过原因 |
| `updatedAt` | `ref('')` | RFC3339 字符串 |
| `saving` | `ref(false)` | 保存中 |
| `urls` | `computed` | `paths.map(p => window.location.origin + p)` |
| `formattedUpdatedAt` | `computed` | `new Date(updatedAt)` → `toLocaleString()` |

**API 调用**（`fetch` + `API_BASE` from `../../api`）：

| 前端动作 | 请求 |
|---|---|
| `login()` | `POST ${API_BASE}/api/ssrules/verify`，头 `X-Super-Admin-Password` |
| `load()` | `GET ${API_BASE}/api/ssrules`，头同上 |
| `save()` | `POST ${API_BASE}/api/ssrules`，头同上 + `Content-Type: application/json`，body `{source}` |
| `regenerateToken()` | `POST ${API_BASE}/api/ssrules/regenerate-token`，头同上；先 `ElMessageBox.confirm` |

- **401 一律 `logout()` 回登录页**（凭据失效的兜底）
- **413 与 400 必须分开提示**（非阻塞项 3）：413 → "请求体过大，请分批粘贴"；400 → "格式错误" / "内容过长"。**不得**把 413 显示成"格式错误"。
- `authHeader()` 来自 `useAdminAuth`，取值 `localStorage['ssrules_admin_password']`

**R3（模板里永远不写 `.value`）**：

> 上游 `SsRuleTool.vue` 的 `<template>` 段 `.value` **0 命中**（已 grep 验证，§9.2）。模板里只用 `source` / `content` / `paths` / `skipped` / `updatedAt` / `urls` / `formattedUpdatedAt` 这些**裸名**，Vue 自动 unwrap。`.value` 只允许出现在 `<script setup>` 里。

**R4（无裸下标 `obj.x[0]`）**：

> 上游模板里**无任何裸下标**（`v-for="(s, i) in skipped"` 用的是解构，不是 `skipped[0]`）。保持。

**`@` 别名**：本组件用**相对路径** `../../api` / `../../composables/useAdminAuth` —— **不要**改成 `@/...`（`@` 指向 `src/neon` 不是 `src`，见 `project_vite_at_alias_neon` 记忆）。

**`useAdminAuth` 参数（P1-5）**：上游传的是

```js
credentialMode: 'header',
credentialField: 'X-Super-Admin-Password',   // ← 这个 key 在 header 模式下被忽略
```

`useAdminAuth.js` 里 header 模式生效的是 **`headerName`**（默认值恰好就是 `'X-Super-Admin-Password'`），所以**功能正确但语义误导**。Stage 3/4 应改成：

```js
credentialMode: 'header',
headerName: 'X-Super-Admin-Password',
```

---

## 4. 兼容性与迁移

### 4.1 旧数据

**无**。本模块是全新表，**没有任何旧数据需要迁移或回填**。`git ls-tree -r origin/main --name-only | grep -ci ssrule` → **0**。

### 4.2 旧接口 / 旧响应

**不破坏任何既有接口**：

- 新增端点全部是新路径（`/api/ssrules*` 与 `/ss/`）
- `/s/:id`（短链）**不受影响**（`/ss/` 是不同的首段）
- `backend/config/config.go` 新增 `SSRules` 字段 —— YAML 解析**向后兼容**：旧 `config.yaml` 没有 `ssrules` 段 → `SSRules.AdminPassword` 为零值 `""` → **模块默认关闭**，不影响任何既有功能 ✅
- **管理端鉴权收紧**（裁决 5：删掉 `?admin_password=` 兜底）只影响**本模块自己的新端点**，不触碰任何既有端点 ✅

### 4.3 需要人工配置的一步（部署时）

`config.yaml` 需新增（**留空 = 关闭**，不是"无密码放行"）：

```yaml
ssrules:
  admin_password: "改成你自己的密码"
```

`config.example.yaml` 末尾追加同一段（上游已写好）。

> ⚠️ **配置是启动时加载的，没有热加载** —— 改完 `config.yaml` **必须重启进程**才生效（回滚路径 ① 同理，见 §7.2）。

### 4.4 路由与压缩中间件

- `backend/middleware/compress.go:135-154` 的 `excludedPathPrefixes` 里有 `"/s/"`（`:137`）**但不匹配 `"/ss/"`** ⇒ 分发端响应会被 gzip（客户端发 `Accept-Encoding: gzip` 时）。客户端（shadowsocks-libev / rust / android）都能处理 gzip，且没有 `Content-Length` 依赖问题。**不需要**改动 compress.go（S9 越界）。若 QA 发现某客户端不接受 gzip，再单独开单把 `/ss/` 加进排除列表。
- `backend/middleware/request_logger.go` 的 skip 列表建议加 `/ss/`（P1-2）—— **唯一**需要碰共享中间件的地方。

---

## 5. 前后端分工与取件纪律

### 5.1 文件归属

| 文件 | 归属单 | Stage |
|---|---|---|
| `backend/handlers/ssrule.go`（新） | **后端单** | 4 |
| `backend/handlers/ssrule_rules.go`（新） | **后端单** | 4 |
| `backend/handlers/ssrule_test.go`（新） | 后端单（Stage 4 补） | 4 |
| `backend/handlers/ssrule_rules_test.go`（新） | **后端单** | 4 |
| `backend/models/ssrule.go`（新） | **后端单** | 4 |
| `backend/routes/ssrule.go`（新） | **后端单** | 4 |
| `backend/routes/index.go` | **后端单** | 4 |
| `backend/routes.go` | **后端单**（**只加 2 行**，见 §5.2.1） | 4 |
| `backend/app_handlers.go` | **后端单** | 4 |
| `backend/app_http.go` | **后端单** | 4 |
| `backend/app_http_ssrule_test.go`（新） | 后端单（Stage 5 补，package main） | 5 |
| `backend/config/config.go` | **后端单**（⚠️ 手工补丁，见 §5.3） | 4 |
| `backend/config.example.yaml` | **后端单**（⚠️ 手工补丁，见 §5.3） | 4 |
| `backend/middleware/request_logger.go` | **后端单**（P1-2 一行） | 4 |
| `AGENTS.md`（模块表加 `#43`） | **后端单**（唯一边界例外） | 4 |
| `frontend/src/views/other/SsRuleTool.vue`（新） | **前端单** | 3 |
| `frontend/src/router/index.js` | **前端单** | 3 |
| `frontend/src/views/other/SsRuleTool.test.js`（新） | **前端单**（或 Stage 5） | 3 / 5 |

> `AGENTS.md` 的模块地图按项目惯例归**后端单**更新（唯一边界例外）。**devtools-architect 不改 `AGENTS.md`**（写权边界：只写 `docs/plans/`）。

#### `AGENTS.md` 的口径（**裁决 4 —— 写死，不留"看情况"**）

Stage 4 更新 `AGENTS.md` 时必须做且只做这三件事：

1. **模块表加一行**（在 `:52` 的 `#42 Trip 旅游行程` 之后）：

   ```
   | 43 | **SSRules** | `backend/handlers/ssrule.go` + `backend/handlers/ssrule_rules.go` + `backend/models/ssrule.go` | `backend/routes/ssrule.go` | `frontend/src/views/other/SsRuleTool.vue` | Shadowsocks 分流规则文件：域名列表 → ACL 正文，公开链接 `/ss/:token/:filename` 分发；管理端 4 端点 + 分发端 1 端点；token 32 hex（crypto/rand）可换发 |
   ```

2. **把标题与引用里的模块数同步为表内实际行数**：
   - `AGENTS.md:7` 的 `## 34 模块地图` → `## 43 模块地图`
   - `AGENTS.md:223` 的 `6. 更新本文件 \`34 模块地图\` 表格` → `43 模块地图`
   - **交付时的权威口径 = `grep -cE '^\| [0-9]+ \|' AGENTS.md` 的实际输出**。当前为 **42**，加完第 43 行后应为 **43**。若届时不等于 43，以 `grep` 结果为准改标题。

3. **「73 个 View 组件」不动**（裁决 4 明示：本单不引入组件计数的权威来源，属既有红项，另开单）。

### 5.2 取件方式（**必须**单路径 checkout）

#### 5.2.1 允许的取件命令

```bash
# 在后端单的 worktree 里，逐个路径取（不要一次 checkout 多个路径，除非确认它们无分叉）
git checkout e7f37dac9fe94390cf0528e3db8ee5d5fbbb8c8e -- backend/handlers/ssrule.go
git checkout e7f37dac9fe94390cf0528e3db8ee5d5fbbb8c8e -- backend/handlers/ssrule_rules.go
git checkout e7f37dac9fe94390cf0528e3db8ee5d5fbbb8c8e -- backend/handlers/ssrule_test.go
git checkout e7f37dac9fe94390cf0528e3db8ee5d5fbbb8c8e -- backend/handlers/ssrule_rules_test.go
git checkout e7f37dac9fe94390cf0528e3db8ee5d5fbbb8c8e -- backend/models/ssrule.go
git checkout e7f37dac9fe94390cf0528e3db8ee5d5fbbb8c8e -- backend/routes/ssrule.go
git checkout e7f37dac9fe94390cf0528e3db8ee5d5fbbb8c8e -- backend/routes/index.go
git checkout e7f37dac9fe94390cf0528e3db8ee5d5fbbb8c8e -- backend/routes.go
git checkout e7f37dac9fe94390cf0528e3db8ee5d5fbbb8c8e -- backend/app_handlers.go
git checkout e7f37dac9fe94390cf0528e3db8ee5d5fbbb8c8e -- backend/app_http.go

# 前端单（单路径取件 —— frontend/src/router/index.js 在 9992ac6→origin/main 0 差异,
# 但上游 commit 也改了它,取前端那两行时按前端单的路径走）
git checkout e7f37dac9fe94390cf0528e3db8ee5d5fbbb8c8e -- frontend/src/views/other/SsRuleTool.vue
```

#### `backend/routes.go` 的特殊处理（**非阻塞项 6 —— 照此执行**）

上游 commit 对 `backend/routes.go` 的改动是 **78 行**，但 `git diff -w e7f37dac^ e7f37dac -- backend/routes.go` 显示**实质只有 2 行**（结构体字段 + 赋值），其余 76 行是上游顺手做的 **gofmt 对齐**（把 `HealthHandler: h.healthHandler,` 那一列的空格对齐改了）。

⇒ **不要整文件 checkout `backend/routes.go`**。正确做法：

1. 手工加**这 2 行**：

   ```go
   // backend/routes.go:12 的 struct 里
   ssRuleHandler *handlers.SSRuleHandler
   // backend/routes.go:91 附近
   SSRuleHandler: h.ssRuleHandler,
   ```

2. 加完后**对这一个文件单独跑一次** `gofmt -w backend/routes.go`（把对齐恢复到项目标准）。
3. 交付说明里要写明：`backend/routes.go` 的 diff = 2 行业务改动 + gofmt 对齐。

#### 5.2.2 **禁止**的取件方式

| 禁止 | 原因 |
|---|---|
| `git cherry-pick e7f37dac…` | 会把上游 commit 的**全部 14 个文件**（含 config 两个分叉文件）一起带进来 → 回退 FINN-56（见 P0-4），且违反「后端/前端分单」 |
| `git checkout e7f37dac… -- backend/` / `-- frontend/`（整目录） | 同上，跨单越界 |
| `git checkout e7f37dac… -- backend/routes.go`（整文件） | 带进 76 行与本单无关的 gofmt 对齐改动（§5.2.1）—— **改为手工加 2 行 + `gofmt -w`** |
| `git reset --hard e7f37dac…` / `git reset --hard origin/main` | **会丢掉平台要求的 worktree 起始 commit** → 平台拒绝记录分支，run 以 `agent_error` 收场（`project_worktree_start_commit_contract`） |
| `git add -A` / `git commit -a` | 会把 worktree 里与本单无关的 `M`/`D`/`??`（基线差异 + `.multica/` + `.claude/skills/`）一起提交 → 交付物被污染。**只 `git add` 目标文件** |

#### 5.2.3 提交前自检

```bash
git diff --cached --stat          # 只应列出本单目标文件
git status --short                # 确认没有误入 .multica/ .claude/ CLAUDE.md
```

#### 5.2.4 分支与 push

```bash
git push -u origin agent/<agent>/finn-<单号>     # 平台不代推，必须自己推
git rev-parse HEAD
git ls-remote origin refs/heads/<分支>            # 与 HEAD 一致
```

### 5.3 上游 commit 的缺陷清单（**Stage 4 必须先修**）

> 上游 commit **不能直接落地**。以下 **7 个 P0 + 6 个 P1** 全部有实读证据（§9）。

#### P0-1 ★ `GET /api/ssrules` 没挂限流 → 密码爆破绕过限流

- **证据**：上游 `backend/routes/ssrule.go:22` 是 `g.GET("", h.SSRuleHandler.Get)` —— **没有** `createRateLimiter.Middleware()`；而 `POST /verify`（`:21`）有。
- **为什么是 P0**：`GET` 与 `POST /verify` 是**同一个密码 oracle**（错密码都返回 401 `密码错误`）。攻击者只要改用 GET 就能绕过限流无限次爆破。上游 commit message 写「密码门也要限流：不限流等于给了无限次爆破机会」—— 但它自己漏了 GET。
- **修**：`g.GET("", getRL, h.SSRuleHandler.Get)`（§3.4.4(d)）
- **断言**：§6.3 A6

#### P0-2 ★ 限流器全站共用 ⇒ ssrules 的限流既挡不住爆破、又拖累别的模块

- **证据**：`backend/app_http.go:39` 只有一个 `createRateLimiter`（10/min）；`backend/middleware/ratelimit.go:49` 的 key **只有** `c.ClientIP()`；`backend/state/transient.go:242` 的 Redis key 是 `"ratelimit:" + key` —— **不含端点名、不含 limiter 身份**。
- **两个后果**：
  1. **Redis 后端下**，新建一个 `NewRateLimiter(...)` 实例**并不会**得到独立预算 —— key 还是 `ratelimit:<ip>`，照样和全站共用同一个计数桶。所以「给 ssrules 单开一个 limiter」这个做法**在 Redis 下是无效的**，必须给 limiter 配**独立 store**。
  2. **Memory 后端下**，`NewRateLimiter(n, w, nil)` 会自建一个私有 `MemoryStore`（`ratelimit.go:20-23`）⇒ 独立预算 ✅。`backend/routes/autodev.go:20-24` 的写法虽然传了 `nil`，但紧接着 `if createRateLimiter != nil { writeRL = createRateLimiter }` **又回落到了全局 limiter** —— **不能照抄那个回落**。
- **修**：本模块用 `middleware.NewRateLimiter(N, time.Minute, nil)`（独立 store ⇒ 独立计数桶），且绝不回落到 `createRateLimiter`。
  - 代价：限流状态不落 Redis（重启即清零、多副本不共享）。本项目是**单容器单副本**部署（`docker-compose.yml` 一个 `devtools` 服务），代价可接受。
  - 备选（本期不做）：给 `middleware.RateLimiter` 加 `KeyPrefix` 字段（改共享中间件，blast radius 大），留作后续单。
- **断言**：§6.3 A6（隔离性）

#### P0-3 ★ 模块关闭语义在分发端不一致

- **证据**：上游 `checkAdmin`（`backend/handlers/ssrule.go:55`，判空在 `:56`）首行 `if h.adminPassword == "" { return false }` —— 但 `Download`（`:182`）**完全不调 `checkAdmin`**，也没有任何 enabled 判断。
- **为什么是 P0**：用户清空 `admin_password` 以为模块关了，但 `/ss/<旧token>/rules.acl` 继续吐旧内容。**配置层面的"关闭"必须等于数据层面的"关闭"。**
- **修**：`Download` 开头判断 `h.adminPassword == ""` ⇒ 直接返回 `404 规则文件不存在`（**必须与"token 不存在"的响应完全一致**，不泄露模块状态）。
- **断言**：§6.3 A5

#### P0-4 ★ `backend/config/config.go` / `config.example.yaml` 整文件 checkout 会回退 FINN-56

- **证据**：`git diff 9992ac6 origin/main -- backend/config/config.go backend/config.example.yaml` 显示 `origin/main` 上多了 FINN-56 的 `-y`：
  - `backend/config/config.go:530` → `Args: []string{"minimax-coding-plan-mcp", "-y"}`
  - `backend/config.example.yaml:237-238` → `args:` 下多一行 `"-y"`
- **为什么是 P0**：`git checkout e7f37dac… -- backend/config/config.go` 会把这两行**退回**（`-y` 消失）⇒ 直接回退 FINN-56 的 MCP 启动修复。
- **修**：**这两个文件禁止整文件 checkout**，必须手工打补丁：
  1. `backend/config/config.go`：`Config` 结构体（`:53` 之后）加 `SSRules SSRulesConfig \`yaml:"ssrules"\``；其后加 `SSRulesConfig` 类型定义（照上游，但用项目注释风格）。
  2. `backend/config/example.yaml`：**在文件末尾追加** `ssrules:` 段（不要动 `minimax_mcp` 那段）。
- **断言**：§6.3 A7（反向断言 `-y` 必须还在）

#### P0-5 ★ 路径穿越判据与回归用例缺失

- **证据**：上游 `ssrule_test.go` 的白名单用例只覆盖 `rules.pac` / `index.html` / `rules.yaml` / `rules` —— **没有任何 `../`、URL 编码、null byte、双扩展名载荷**。
- **修**：按 §3.4.1(c) 的载荷表补齐用例（**handler 级 + HTTP 级两层**，§3.4.1(e)），断言 **404 且 body 恰为 `规则文件不存在`**。
- **断言**：§6.3 A3 + A3-HTTP

#### P0-6 ★（**FINN-102 新增**）限流键完全由客户端控制 ⇒ 限流可绕过

- **证据**：见 §3.4.4(a) 的完整链条（`c.ClientIP()` → gin 默认 `trustedProxies = 0.0.0.0/0` → `validateHeader` 返回 XFF **最左**项；全仓 `SetTrustedProxies` **0 命中**）。
- **修**：模块内自建限流中间件 + `ssRuleClientIP(c)`（CF-Connecting-IP → XFF 最右 → RemoteAddr），**不动共享 `middleware.RateLimiter`**（裁决 1）。
- **断言**：§6.3 A13

#### P0-7 ★（**FINN-102 新增**）未匹配的 `/ss/` 路径回落 SPA fallback ⇒ 客户端拿到 HTML

- **证据**：`backend/app_http.go:78-103` 的 NoRoute：非 `/api/` 路径 → `c.File(distDir+"/index.html")`（`:102`）⇒ **200 + HTML**。上游只注册了 `/ss/:token/:filename` 一条路由 ⇒ `/ss/TOK/`、`/ss/TOK/rules.acl/`、`/ss/TOK/../../etc/passwd` 等全部落到 NoRoute ⇒ 200 + HTML。
- **为什么是 P0**：客户端会"成功"拿到一坨 HTML 当规则文件 —— **静默失败，比 404 更难排查**；且与 PM 裁决 2 直接冲突。
- **修**：§3.4.1(d) 的**两处改动**（NoRoute 特判 + 3 条占位路由），共 4 条 `/ss/` 路由。
- **断言**：§6.3 A3-HTTP

#### P1-1 `token` 列缺 UNIQUE 索引

`ss_rule_files.token TEXT NOT NULL` 无唯一约束 ⇒ 补 `CREATE UNIQUE INDEX IF NOT EXISTS idx_ss_rule_files_token ON ss_rule_files(token);`（`InitSSRuleFiles` 内，紧跟建表之后）。

#### P1-2 token 落进 `http_request_logs` 表

`backend/middleware/request_logger.go:55` 记完整 `c.Request.URL.Path`（截断 512）⇒ token 明文入库，`/api/monitor` 可读。

**修**：把 `"/ss/"` 加进 `shouldSkipMonitoringRequest` 的前缀列表（`request_logger.go:77-87`）。
**代价**：公开下载不再进监控日志（失去该端点的访问审计）。**取舍由 PM 拍板**（见 §8 Q1）—— 本方案默认**加**（token 是凭据，优先保护凭据）。

#### P1-3 `POST /api/ssrules` 的请求体上限是 200MB

`backend/app_http.go:35` 全局 `ContentSizeLimiter(200MB)`，而 `ssRuleMaxSourceSize`（512KB）的检查发生在 `ShouldBindJSON` **之后** ⇒ 一个 200MB 的 body 会先进内存。虽然 `POST` 需密码（未授权者到不了 `ShouldBindJSON`），但仍应收紧。

**修**：`api.Group("/ssrules", middleware.ContentSizeLimiter(1<<20))`（1MB，远大于 512KB 上限）。超限响应 `413 {"error":"内容大小超过限制","code":413}`（`ratelimit.go:66-69`）—— **契约见 §3.2.1**。

#### P1-4 公开分发端无限流

见 §3.4.4。**修**：在 `registerSSRulePublicRoutes` 里给 `/ss/` 挂**独立预算**的 limiter（`ssRuleLimiter(120, time.Minute)`，独立 store）。

#### P1-5 前端 `useAdminAuth` 参数名

见 §3.7。**修**：`credentialField` → `headerName`（值不变）。

#### P1-6 `Content-Disposition` 未设置

现状**安全**（无注入面），但语义上"下载"应该有 `attachment`。**修**：可选，`c.Header("Content-Disposition", "attachment; filename=\"rules.acl\"")` —— **只能用白名单常量**，严禁拼 `c.Param("filename")`。

#### P2（记录并接受，不阻塞）

| # | 项 | 处置 |
|---|---|---|
| P2-1 | token 无时效、一次性 | 客户端规则源需长期可用；换发是止损手段 |
| P2-2 | token 明文入库 | 单行表 + 可换发；库泄露 ≈ 规则泄露，可接受（规则本身不是高敏感数据） |
| P2-3 | CF 访问日志含 token | 不可控；靠换发止损 |
| P2-4 | 未设 `X-Content-Type-Options: nosniff` | **本单采纳**（非阻塞项 10）→ §3.2.2 响应头清单已加 |
| P2-5 | `RegenerateToken` 的 `skipped` 恒 `[]`（与 `Get` 不一致） | **本单采纳**（非阻塞项 11）→ §3.2.3 已写明修法 |

### 5.4 ★ FINN-102 回改对照表（PM 复核按这张表逐条查）

> **每一行都给出**：回改点 → 方案 §号 → **可证伪的验收断言**（断言原文见 §6.3 / §6.4）。

#### B1 · 路径穿越载荷表 10 行期望与真实行为不符（P0）

| # | 回改点 | 方案 §号 | 验收断言 |
|---|---|---|---|
| B1-1 | 订正机制说明：`UseRawPath` 默认 false（`gin.go:198`）⇒ 用解码后 `URL.Path`（`gin.go:594`）⇒ `%2f` 变真 `/` ⇒ 段数变多 ⇒ 3 段路由**不匹配** ⇒ 走 NoRoute，**不是**"单个 `:filename`" | §3.4.1(a) | §9.9 实测输出（`/ss/TOK/..%2f..%2fetc%2fpasswd` → 404 且**未进入 handler**） |
| B1-2 | 保留安全论据「不触碰文件系统 + 白名单精确相等」 | §3.4.1(b) | §6.3 **A3**（把白名单改成 `HasSuffix(".acl")` → `rules.acl.txt` 那行必须红） |
| B1-3 | 断言分两层：白名单单测在 **handler 级**（注入 `c.Params`）；HTTP 级另立一条只断言「不得返回 ACL 正文 / 不得 5xx」 | §3.4.1(e) | §6.3 **A3** + **A3-HTTP** |
| B1-4 | 载荷表 10 行期望**统一**为 `404 规则文件不存在`（含 `/api` 特判之后的 `/ss/` NoRoute 分支） | §3.4.1(c) | §6.3 **A3-HTTP**（10 行逐条） |
| B1-5 | `rules.acl/`（尾斜杠）必须显式关闭 `RedirectTrailingSlash` → 落到同款 404 | §3.4.1(d) 改动 2 | §6.3 **A3-HTTP** 第 7 行 + **A3-HTTP-3xx**（断言无 3xx） |

#### B2 · 限流器的 IP 键完全由客户端控制（P0）

| # | 回改点 | 方案 §号 | 验收断言 |
|---|---|---|---|
| B2-1 | 写明限流键来源 + **可执行实现**：模块内自建中间件 + `middleware.RateLimiter.Allow(ip)`（`ratelimit.go:32` 已导出），**不改共享 `middleware.RateLimiter`** | §3.4.4(b)(c) | §6.3 **A6**（桶隔离）+ **A13**（键不可伪造） |
| B2-2 | 键取值顺序：`CF-Connecting-IP` → XFF **最右**一项 → `RemoteAddr` host；给出代码片段 | §3.4.4(b) | §6.3 **A13** |
| B2-3 | 写明残余风险：源站直连时 `CF-Connecting-IP` 可伪造（对外拓扑只有 CF，接受，不算缺陷）；无 CF 时退 XFF 最右项 | §3.4.4(e) | §6.3 **A13** 第 3 子项 |

#### PM 裁决 1–6（已批准的实现偏离 —— Stage 3/4 照此实现）

| # | 裁决 | 方案 §号 | 验收断言 |
|---|---|---|---|
| 裁决 1 | 限流键来源 + 模块内自建中间件 + 不动共享 RateLimiter | §3.4.4(b)(c) | §6.3 **A13**、**A6** |
| 裁决 2 | 未匹配 `/ss/` 路径 = `404` + body 逐字节 `规则文件不存在`，**不得**回落 SPA | §3.4.1(d) 改动 1 | §6.3 **A3-HTTP** |
| 裁决 3 | NoRoute 里特判 `/ss/`（在 `/api` 特判之后）+ 关闭 `/ss/` 的自动重定向；后端单写边界 | §3.4.1(d) | §6.3 **A3-HTTP**、**A3-HTTP-3xx** |
| 裁决 4 | `AGENTS.md` 加 `#43 SSRules`；标题 `## 34 模块地图`（`:7`）与 `:223` 的引用同步为表内实际行数（交付时以 `grep -cE '^\| [0-9]+ \|' AGENTS.md` 为准，**当前 42 → 加完为 43**）；「73 个 View 组件」不动 | §5.1 | §6.3 **A17** |
| 裁决 5 | 删除 `?admin_password=` query 兜底；只认 header `X-Super-Admin-Password` | §3.2.1 | §6.3 **A8**、**A18** |
| 裁决 6 | `updated_at` 统一 UTC（`Z` 结尾）；前端 `toLocaleString()` | §3.2.3、§3.5 | §6.3 **A9**（断言偏移为 0） |

#### 评审的 12 条非阻塞项

| # | 评审条目 | 处置 | 方案 §号 | 验收断言 |
|---|---|---|---|---|
| 1 | §3.6 接线表里 `/ss/:token/:filename` **没挂限流中间件**（照表接线就公网裸奔） | **回改方案文本**：§3.4.4(d) 明确 5 个端点的限流阈值，§3.6 骨架里 4 条管理端路由各挂一个 | §3.4.4(d)、§3.6 | §6.3 **A6**（含"分发端第 121 次 → 429"子项） |
| 2 | 管理端 4 端点共用 10/min 一个桶 | **裁决**：按端点分桶 10/10/30/20 + 分发 120 | §3.4.4(d) | §6.3 **A6**（逐端点阈值子项） |
| 3 | M3 契约缺 **413** | **回改方案文本**：§3.2.1 补 413 + 400/413 区分表 + 前端契约 | §3.2.1、§3.7 | §6.3 **A16**、§6.4 **F7** |
| 4 | `?admin_password=` 兜底的代价 + 引用行号指向 `?super_admin_password=` 约定 | 已由**裁决 5** 覆盖（删掉 query 兜底）；引用订正为 `monitoring.go:246` / `:251` | §3.2.1、§2.4 | §6.3 **A18** |
| 5 | §7.2 回滚路径①「秒级生效」不准确（config.yaml 启动时加载，无热加载） | **回改方案文本**：改成「**重启后生效**」 | §0、§4.3、§7.2 | §6.3 **A19**（配置项 `ssrules.admin_password` 为空 ⇒ 模块关闭） |
| 6 | §5.2.1 允许整文件 checkout `backend/routes.go` ⇒ 带进 78 行与本单无关的 gofmt 对齐改动 | **回改方案文本**：改为「只加这 2 行，之后对该文件单独跑一次 `gofmt -w`」；§6.1③ 的 gofmt 期望值随之说明 | §5.2.1 | §6.1 ③（gofmt 本单文件 0 行输出）+ §6.3 **A20**（`git diff --name-only` 复核） |
| 7 | §5.2.1 只给了后端取件命令，**前端侧没有对应行** | **回改方案文本**：补 `git checkout e7f37dac -- frontend/src/views/other/SsRuleTool.vue`（`frontend/src/router/index.js` 在 `9992ac6`→`origin/main` **0 差异**，已实测） | §5.2.1 | §9.11（0 差异的原始输出） |
| 8 | §5.1 的 AGENTS.md 口径留成「看情况」 | **回改方案文本**：按裁决 4 写死 | §5.1 | §6.3 **A17** |
| 9 | 行号漂移 3 处 | **回改方案文本**：订正 —— 实测 `state/transient.go` 的 `fullKey := s.key("ratelimit:" + key)` 在 **:242**（评审说 :240）；`models/paste.go` 的 `generateID` 在 **:245**（评审说 :244）；`app_http.go` 的 `ContentSizeLimiter(200MB)` 在 **:35**（评审说 :35 ✓） | §2.4、§9.5 | §9.5（原始输出） |
| 10 | `X-Content-Type-Options: nosniff` 未加 | **采纳**：分发端响应头加一行 | §3.2.2、§3.4.3 | §6.3 **A14** |
| 11 | `RegenerateToken` 的 `skipped` 恒 `[]` 与 `Get` 不一致 | **采纳**：换发 token 时重跑 `buildACLRules`，`skipped` 填实际跳过的行 | §3.2.3 | §6.3 **A15** |
| 12 | `/ss/` 是否会被 gzip | **保持现有说明，不动 `compress.go`**（`excludedPathPrefixes` 含 `"/s/"` 而 `"/ss/"` 不匹配 ⇒ 无冲突） | §4.4 | §6.4 **F8**（静态检查：compress.go 未被改动） |

---

## 6. 验证计划（Stage 4 / Stage 5 可直接照抄）

### 6.1 Stage 4 · 后端编译与静态检查（R11：走 Docker CGO，**不在 macOS 上出二进制**）

```bash
# 在后端单 worktree 的 backend/ 目录
cd backend

# ① 编译（R11：CGO + linux 目标在容器内完成）
docker run --rm \
  -v "$PWD":/app -w /app \
  -e GOPROXY=https://goproxy.cn,direct \
  -e CGO_ENABLED=1 \
  golang:1.25.5 \
  sh -c 'go build -tags with_utls -o /tmp/server . && echo BUILD_OK'

# ② go vet
docker run --rm -v "$PWD":/app -w /app -e GOPROXY=https://goproxy.cn,direct \
  golang:1.25.5 sh -c 'go vet ./... && echo VET_OK'

# ③ gofmt -s：只检查本单碰过的文件（仓库既有 35 个 gofmt 红项，不在本次范围）
gofmt -s -l app_handlers.go app_http.go config/config.go routes.go routes/index.go \
  routes/ssrule.go handlers/ssrule.go handlers/ssrule_rules.go \
  handlers/ssrule_test.go handlers/ssrule_rules_test.go models/ssrule.go \
  app_http_ssrule_test.go
# 期望：0 行输出
```

> **说明**：`golang:1.25.5`（bookworm）自带 gcc，CGO 可用；输出写到容器内 `/tmp`，**不污染宿主**。`with_utls` tag 与 `Dockerfile:39` 一致。
> **③ 的期望值说明**：上游 commit 的这批文件在 commit 状态下 `gofmt -s -l` **全部干净**（已实测，§9.3）；但 `origin/main` 上 `backend/routes.go` **本来就不干净**（上游顺手修了它）。⇒ **本单按 §5.2.1 只手工加 2 行到 `backend/routes.go`，然后对该文件单独跑一次 `gofmt -w`**，这样 §6.1③ 才能得到 0 行输出。仓库另外 34 个 gofmt 红项**不要碰**（S9 越界）。

### 6.2 Stage 5 · 后端单测（只跑本模块，避开既有红项）

```bash
cd backend
docker run --rm \
  -v "$PWD":/app -w /app \
  -e GOPROXY=https://goproxy.cn,direct \
  golang:1.25.5 \
  sh -c "go test ./handlers/ -run 'TestSSRule|TestBuildACLRules' -count=1 -v"

# ★ HTTP 级断言在 package main（§6.3 A3-HTTP），单独跑：
docker run --rm \
  -v "$PWD":/app -w /app \
  -e GOPROXY=https://goproxy.cn,direct \
  golang:1.25.5 \
  sh -c "go test . -run 'TestSSRulePublicRouteShapes' -count=1 -v"
```

> **不要跑 `go test ./handlers/` 全包**：`main` 上就有 3 FAIL + 1 挂起（`TestAsyncAnthropicMessages_UpstreamError` / `TestDoJSONRequestNoRetryOnCanceled` / `TestParseClashYAML_MarksUnsupportedTypes` / `TestSendAlertSMTPFlow` 挂起），与本单无关（`project_handlers_preexisting_test_failures`）。
>
> **依赖注意**：`ssrule_test.go` 复用 `backend/handlers/skills_db_test.go:24` 的 `newMemDB(t)`（已含 `SetMaxOpenConns(1)`，满足 R5）。**不要删 `skills_db_test.go`** —— 删了本模块测试编译不过。
> `mustJSON` 是 `ssrule_test.go` **新引入**的 helper（`grep "func mustJSON" backend/handlers/*.go` → 0 命中），无重名冲突。
> **package main 的测试**（`backend/app_http_ssrule_test.go`）用 `models.NewDB(":memory:")` + **必须紧跟 `db.SetMaxOpenConns(1)`**（`models/paste.go:235` 有导出方法）+ `db.InitSSRuleFiles()` —— 满足 R5。

### 6.3 后端验收断言（**可判定、不恒真**）

> 每条断言都要能「改了就会红」。括号里是**反事实自检**（mutation）—— Stage 5 必须做这一步，否则断言可能是恒真闸门。

| # | 断言 | 反事实自检 |
|---|---|---|
| **A1** | 管理端密码门：无密码头 → 401；错密码 → 401；对密码 → 200 且 body 有 `paths`（4 条）与 `source` | 把 `checkAdmin` 的 `return false` 改成 `return true` → A1 必须红 |
| **A2** | `GET /ss/<token>/rules.acl` → 200；body 含 `(^\|\.)google\.com$`；头 `Content-Type: text/plain; charset=utf-8`、`Cache-Control: no-store` | 去掉 `Cache-Control` → A2 必须红 |
| **A3**（**handler 级**） | **白名单单测**：对 §3.4.1(c) 展开后的 **14 个 filename 值**，注入 `c.Params`（`gin.Params{{"token","TOK"},{"filename",<载荷>}}`）直接调 `h.Download` → **逐条** 404 且 body **逐字节等于** `规则文件不存在`；且 body **不含** `[proxy_list]` | 把白名单改成 `strings.HasSuffix(name, ".acl")` → `rules.acl.txt` 那条必须红；把 `c.String(404, ssRuleNotFoundBody)` 改成 `c.String(404, "not found")` → 全部必须红 |
| **A3-HTTP**（**HTTP 级**） | 用**真实的** `registerSSRulePublicRoutes` + `registerStaticRoutes`（package main 测试）发**同样 14 条 URL** → 逐条 **404** + body **逐字节等于** `规则文件不存在`；body 不含 `[proxy_list]`；状态码 **不是 5xx** | 删掉 NoRoute 里的 `/ss/` 特判 → 第 1/2/3/4/7/10 行那 6 条必须红（变成 200 + `<!DOCTYPE html>`）；把 `/ss/:token/:filename/` 占位路由指向 `h.Download` → 第 7 行必须红（会返回 200 + ACL 正文） |
| **A3-HTTP-3xx** | 上述 **14 条** + `/ss/TOK` + `/ss/` 的响应**全部不是 3xx**（`Location` 头为空、状态码 ≠ 301/302/307/308） | 删掉 3 条占位路由 → `/ss/TOK/rules.acl/` 必须红（变 301） |
| **A4** | 换发 token 后：旧 token → 404；新 token → 200；旧 token 404 的 body 与"乱 token"404 的 body **完全一致** | 让 `RegenerateSSRuleToken` 不 UPDATE token → A4 必须红 |
| **A5** | **未配密码（`adminPassword == ""`）时**：M1-M4 全 401；**且** `GET /ss/<任意 token>/rules.acl` → 404（P0-3） | 把 P0-3 的 enabled 判断删掉 → A5 必须红 |
| **A6** | **限流（按端点分桶）**：<br>(a) 同一 IP 连打 `POST /api/ssrules/verify` 第 **11** 次 → 429<br>(b) 同一 IP 连打 `POST /api/ssrules/regenerate-token` 第 **11** 次 → 429<br>(c) 同一 IP 连打 `GET /api/ssrules` 第 **31** 次 → 429<br>(d) 同一 IP 连打 `POST /api/ssrules` 第 **21** 次 → 429<br>(e) 同一 IP 连打 `GET /ss/<token>/rules.acl` 第 **121** 次 → 429<br>(f) **桶隔离**：把 (a) 打满 10 次后，同 IP 的 `GET /api/ssrules` 第 1 次仍 200；把 (e) 打满 120 次后，`POST /api/paste` 第 1 次仍 200 | 去掉 GET 上的限流 → (c) 必须红；把 ssrules 的 limiter 换回 `createRateLimiter` → (f) 必须红；把 4 个管理端 limiter 合成 1 个 → (b)/(c)/(d) 的阈值必须红 |
| **A7** | **反向断言**：`backend/config/config.go` 里 `minimax-coding-plan-mcp` 的 `Args` **仍含 `"-y"`**；`config.example.yaml` 的 `minimax_mcp.args` **仍含 `"-y"`**（防 P0-4 取件回退） | 用 `git checkout e7f37dac… -- backend/config/config.go` 覆盖一次 → A7 必须红 |
| **A8** | **token 权限边界**：把 token 当管理凭据（`GET /api/ssrules?token=<token>`、`POST /api/ssrules?admin_password=<token>`）→ 必须 401 | 让 `checkAdmin` 接受 token → A8 必须红 |
| **A9** | `updated_at` 可被浏览器解析：后端返回 RFC3339 字符串，`time.Parse(time.RFC3339, v)` 成功、非零值，**且时区偏移为 0**（尾部是 `Z`，不是 `+08:00`） | 把 `updated_at` 写成零值 → A9 必须红；把写入改成 `time.Now()`（本地时区）→ A9 的偏移子项必须红 |
| **A10** | 表结构：`ss_rule_files` 有 `idx_ss_rule_files_token` 唯一索引（P1-1） | 去掉 `CREATE UNIQUE INDEX` → A10 必须红 |
| **A11** | 规则生成：域名 → `(^\|\.)example\.com$`（**不是** `\|\|example.com`）；`[bypass_all]` + `[proxy_list]` 结构完整；非法行逐条进 `skipped` 且**不出现在产物里** | 把 `ssRuleHostToACLLine` 改成输出 `\|\|` → A11 必须红 |
| **A12** | 监控日志**不含** `/ss/` 路径（P1-2，若 PM 采纳该取舍） | 把 `/ss/` 从 skip 列表删掉 → A12 必须红 |
| **A13**（**B2 核心**） | **限流键不可伪造**：<br>(a) 同一"真实客户端"用**不同**的 `X-Forwarded-For: 1.2.3.4` / `5.6.7.8` 连打 `POST /api/ssrules/verify` → **第 11 次仍 429**（说明键**不是** XFF 最左项）；<br>(b) 同一 IP 用**不同**的 `X-Forwarded-For` 打满 10 次后，**不带任何头**再打 1 次 → 仍 429（说明键**不是** XFF）；<br>(c) 不带 `CF-Connecting-IP`、不带 `X-Forwarded-For`、只靠 `RemoteAddr` 时，10 次后第 11 次 → 429（说明退到了 RemoteAddr） | 把 `ssRuleClientIP` 改回 `c.ClientIP()` → (a)/(b) 必须红；把 XFF 取值改成 `parts[0]`（最左）→ (a) 必须红 |
| **A14** | 分发端响应头含 `X-Content-Type-Options: nosniff` | 删掉该 header → A14 必须红 |
| **A15** | `POST /api/ssrules/regenerate-token` 的响应里 `skipped` 是**实际重算值**（存了 3 行非法域名的 source，换发后 `skipped` 长度 ≥ 1，且与 `GET /api/ssrules` 的 `skipped` **逐条相等**） | 改回 `newSSRuleView(file, nil)` → A15 必须红 |
| **A16** | **413 与 400 可区分**：`POST /api/ssrules` 带 **> 1MB** 的 body → **413** 且 body `{"error":"内容大小超过限制","code":413}`；带**非法 JSON** 的小 body → **400** 且 body `{"error":"请求格式错误"}` | 把组上的 `ContentSizeLimiter(1<<20)` 去掉 → A16 的 413 子项必须红（变成 400 或 500） |
| **A17** | **AGENTS.md 口径**：`grep -cE '^\| [0-9]+ \|' AGENTS.md` 的结果 **等于** `grep -m1 -E '^## [0-9]+ 模块地图' AGENTS.md` 里的数字 **等于** `AGENTS.md:223` 引用的数字（裁决 4） | 只加模块行不改标题 → A17 必须红（42 ≠ 43） |
| **A18** | **query 兜底已删**：`GET /api/ssrules?admin_password=<正确密码>`（**不带** header）→ **401**；`?super_admin_password=<正确密码>` → 同样 **401** | 恢复 `c.Query("admin_password")` 兜底 → A18 必须红 |
| **A19** | **模块关闭语义**：配置 `ssrules.admin_password` 为空（= 模拟"关模块"）时，管理端全 401、分发端全 404 —— **且** 说明"配置改动**需重启**才生效"（断言只覆盖行为，不覆盖重启） | 删 P0-3 判断 → A19 必须红 |
| **A20** | **取件纪律**：`git diff --name-only origin/main..HEAD` 里**不含** `backend/config/config.go` 之外的意外文件；`backend/routes.go` 的 diff 实质只有 **2 行**业务改动（用 `git diff -w --numstat` 复核）；该文件 `gofmt -s -l` 输出为空 | 整文件 checkout `backend/routes.go` → A20 的"实质 2 行"子项必须红 |

### 6.4 Stage 5 · 前端（`/tmp` scratch + pnpm，**不在 SMB 盘上装依赖**）

```bash
# ① /tmp scratch（/Volumes/M20 是 SMB：不要在那儿 npm install / 落大量小文件）
SCRATCH=$(mktemp -d /tmp/devtools-frontend-ssrule-XXXX)
rsync -a --exclude node_modules --exclude dist frontend/ "$SCRATCH/"
cd "$SCRATCH"

# ② 装依赖
pnpm install --frozen-lockfile

# ③ 跑本模块测试
#    Node 26 的实验性 globalThis.localStorage 会顶掉 jsdom 的 → 必须带 --localstorage-file
#    （FINN-100 修好 useAdminAuth.test.js 后可去掉；在此之前必须带）
NODE_OPTIONS='--localstorage-file=/tmp/vitest-localstorage-ssrule.json' \
  pnpm exec vitest run src/views/other/SsRuleTool.test.js

# ④ 构建（router 条目编译通过）
pnpm build

# ⑤ 清理
rm -rf "$SCRATCH"
```

**前端验收断言**（`frontend/src/views/other/SsRuleTool.test.js`，上游 commit **没有**这个文件 —— Stage 3/5 必须补）：

| # | 断言 | 反事实自检 |
|---|---|---|
| **F1** | **R3**：`<template>` 段对 `ref`/`computed` 绑定**无** `.value` | 在模板里手写一个 `content.value` → 必须红 |
| **F2** | **R4**：`<template>` 段**无裸下标** `x[y]`（可选链 `?.[]` 允许） | 在模板里写 `paths[0]` → 必须红 |
| **F3** | **调试残留**：改动文件里无 `console.log` / `TODO` / `FIXME` / `XXX` / `HACK` | 加一行 `console.log` → 必须红 |
| **F4** | **路由条目**：`toolRoutes` 里存在 `path === '/ssrules'`，且 `meta.title` / `meta.icon` / `meta.category` 三个字段**都非空**（防"少写 meta"—— 少写 `meta.category` 会被 `isNavigableTool` 过滤掉、工具在侧边栏里**静默不出现**） | 删掉 `category: 'other'` → 必须红 |
| **F5** | **行为**：mock `fetch` → 返回 401 时组件调用 `logout()`（回到密码门）；`fetch` 返回正常时 `urls` 每条 = `window.location.origin + path` | 让 `load()` 不处理 401 → 必须红 |
| **F6** | **R6/R8 不适用性**：本组件不含 `blob:`（本模块不涉及分享资产） | — |
| **F7** | **413 与 400 分开提示**：mock `fetch` 返回 413 → 提示文案**不是**"格式错误"（含"过大"或"超限"字样）；返回 400 → 提示"格式错误" | 把 413 分支删掉（统一走 400 分支）→ 必须红 |
| **F8** | **gzip 说明不误改**：`frontend/` 改动文件里**不含**任何对 `compress.go` 的引用；`backend/middleware/compress.go` **不在**本次 diff 里（静态检查 `git diff --name-only origin/main..HEAD \| grep -c 'backend/middleware/compress.go'` → **0**） | 改一次 compress.go → 必须红 |

> **F1/F2 的实现方式**：照抄既有先例 `frontend/src/views/ai/resultShareStaticRules.test.js`（读 `.vue` 源码文本 → 切 `<template>` 段 → 正则扫描）。**不要**用 `@vue/test-utils` mount 做静态断言 —— 静态规则用文本扫描更稳、更不会因为环境差异假红。

### 6.5 反事实自检总则（Stage 5 必做）

对每条断言，必须产出**三组输出**：

1. **基线**（未改本单代码时）：该断言应当**红**或**不适用**
2. **被 FAIL 版**（人为植入缺陷）：必须**红**
3. **修复分支**：必须**绿**

只有 2、3 两组、且 2 确实红、3 确实绿，才算断言"咬合"。**恒真断言（怎么改都绿）= FAIL**。

> ⚠️ **恒真闸门自查**：A17 / A20 / F8 这类"静态检查"断言必须锚定**代码块行**或**具体数字**（如"42 ≠ 43"），**不得**写成"grep 某标识符存在"这种基线已命中的形式（那恒真 = FAIL）。§6.3 A17 的写法（"计数 == 标题里的数字"）是**可证伪**的：只加模块行不改标题 → 立刻红。

---

## 7. 风险与回滚

### 7.1 风险登记

| # | 风险 | 概率 | 影响 | 缓解 |
|---|---|---|---|---|
| R-1 | 整文件 checkout `config.go` 回退 FINN-56 | **高**（最自然的手滑） | MCP 启动失败 | P0-4 + 断言 A7 |
| R-2 | 只给 `POST /verify` 限流，攻击者改用 `GET` 爆破 | **高**（上游现状） | 密码被爆破 ⇒ 规则可被篡改（进而影响客户端分流） | P0-1 + 断言 A6 |
| R-3 | 清空密码以为关了模块，公开链接仍在服务 | 中 | 内容继续外泄 | P0-3 + 断言 A5 |
| R-4 | `/ss/` 未注册 → NoRoute 兜成 `index.html` | 中（改动接线时容易漏） | 客户端拿到 HTML 当规则，**静默失败** | 断言 A2（body 必须是 ACL 正文而不是 `<!DOCTYPE html>`） |
| R-5 | `git add -A` 把无关改动带进交付 | 中 | 交付物污染 | §5.2.3 自检 |
| R-6 | 共享限流桶导致别的模块被连带 429 | 中 | 其它工具 429 | P0-2 独立 store + A6(f) |
| R-7 | 公开端点被机器人扫描打 DB | 中 | DB 压力 | P1-4 独立 limiter（120/min） |
| R-8 | `updated_at` 序列化形态退化 ⇒ 页面显示 1/1/1 | 低 | 体验 | 断言 A9 |
| R-9 | **限流键被伪造 ⇒ 限流形同虚设** | **高**（上游现状，且现成中间件无法修） | 密码爆破 / DB 被打爆 | **P0-6** + 断言 A13 |
| R-10 | **未匹配 `/ss/` 路径回落 SPA ⇒ 客户端拿到 HTML** | **高**（上游现状） | 静默失败；与裁决 2 冲突 | **P0-7** + 断言 A3-HTTP |
| R-11 | 4 条 `/ss/` 路由注册冲突 → **启动即 panic** | 低（已实测不冲突，§9.9） | **整个服务起不来** | §9.9 实测输出 + A3-HTTP（路由构造在测试里跑一遍即暴露 panic） |

### 7.2 回滚（两条路径）

#### 路径 ① · 配置级关闭（首选；**重启后生效**，不动代码）

```yaml
# config.yaml
ssrules:
  admin_password: ""     # 清空
```

> ⚠️ **配置是启动时加载的，没有热加载** —— 改完 `config.yaml` **必须重启进程**才生效。v1 写的"秒级生效"**不准确**，本版订正。

**预期行为**（依赖 **P0-3 已修**）：

| 端点 | 关闭后 |
|---|---|
| M1-M4 | 401 `密码错误` |
| P1 `/ss/*` | **404 `规则文件不存在`** |
| 未匹配的 `/ss/` 路径 | **404 `规则文件不存在`**（NoRoute 特判仍在） |

> ⚠️ **P0-3 未修时，路径 ① 只能关掉管理端，公开链接继续吐旧内容** —— 所以 **P0-3 是这条回滚路径成立的前提**。

#### 路径 ② · 代码级回滚（删路由）

1. 删 `backend/routes/ssrule.go` 整个文件 + `routes/index.go` 的 `RegisterSSRuleRoutes` 调用 + `routes.RouteHandlers` 的 `SSRuleHandler` 字段
2. 删 `backend/routes.go` 的 `ssRuleHandler` 字段与 `SSRuleHandler: h.ssRuleHandler,`
3. 删 `backend/app_handlers.go` 的构造与赋值
4. 删 `backend/app_http.go` 的 `registerSSRulePublicRoutes(router, handlers.ssRuleHandler)` 调用
5. 删 `backend/handlers/ssrule*.go`、`backend/models/ssrule.go`

> ⚠️ **陷阱**：删完路由后，`/ss/<token>/rules.acl` 会落到 `NoRoute` → **SPA fallback 返回 `index.html` 且状态码 200**（`backend/app_http.go:78-103`，`:102` 是 `c.File(distDir+"/index.html")`）。客户端会"成功"拿到一坨 HTML 当规则文件 —— **静默失败，比 404 更难排查**。
>
> **因此路径 ② 必须同时保留 NoRoute 里的 `/ss/` 特判**（§3.4.1(d) 改动 1）。**那条特判不随路由一起删** —— 它是 `/ss/` 前缀的安全兜底，不是分发逻辑。
>
> 若连特判也不保留，退而求其次：保留一条**只返回 404 的空路由**占位 `router.GET("/ss/:token/:filename", func(c *gin.Context){ c.String(404, handlers.SSRuleNotFoundBody) })` —— 但**不要**删掉特判，两者不冲突。

#### 回滚后的数据残留

`ss_rule_files` 表会**留在库里**（单行，无害）。彻底清除：
```sql
DELETE FROM ss_rule_files;
```
（下次访问会懒创建新行 + 新 token。）

---

## 8. 未决问题（需人工裁决，不阻塞 Stage 2/3/4 开工）

| # | 问题 | 可选解释 | 本方案倾向 | 裁决人 |
|---|---|---|---|---|
| **Q1** | P1-2：公开下载是否进监控日志？ | ① 加 `/ss/` 到 skip → token 不入库，但失去该端点的访问审计<br>② 不加 → 有审计，但 token 明文落 `http_request_logs` | **① 加**（token 是凭据；CF 日志本来也会记，审计价值有限） | @devtools-pm |
| **Q2** | P1-4：分发端限流阈值 | ① **120/min/IP**（裁决 2 已定）<br>② 30/min/IP（更严，但多人共享出口 IP 时可能误伤）<br>③ 不限流 | **① 120/min/IP**（已裁决，不再讨论） | 已裁决 |
| **Q3** | 独立限流器用内存（不落 Redis）是否可接受？ | ① 接受（单容器单副本，重启清零）<br>② 给 `middleware.RateLimiter` 加 `KeyPrefix`，落 Redis（改共享中间件，blast radius 大） | **① 接受**，② 另开单 | 已裁决（裁决 1 明令不动共享 RateLimiter） |
| **Q4** | token 要不要加有效期？ | ① 不加（客户端规则源需长期可用，换发是止损手段）<br>② 加 90 天 TTL（客户端要定期重填链接） | **① 不加** | @devtools-pm |
| **Q5** | `ssrules.admin_password` 是否复用全局 `super_admin_password`？ | ① 独立配置（照 proxy / nps 惯例，本方案采用）<br>② 复用全局（少一个配置项，但两个模块同密码 ⇒ 一处泄露两处失守） | **① 独立** | @devtools-pm |
| **Q6** | 是否要支持"多套规则"（按客户端区分）？ | ① 不做（上游设计已定"单一事实源"）<br>② 加 `id` 维度做成多行 | **① 不做** | @devtools-pm |
| **Q7** | 裸 `/ss`（无尾斜杠）是否也收口成 404？ | ① 收口（NoRoute 里加 `path == "/ss"`）<br>② 不收口（严格照裁决 3 的 `HasPrefix("/ss/")` 口径） | **② 不收口**（裁决 3 明写 `strings.HasPrefix(path, "/ss/")`；裸 `/ss` 不泄露任何规则数据，实测也不 301） | @devtools-pm |

> **D6 说明**：以上 7 项都已给出「两种以上可选解释 + 我的倾向」，**Stage 2/3/4 不必等裁决即可开工**（Q1 涉及一行代码，Stage 4 按倾向实现，PM 若改判再回改一行）。Q2/Q3 已被 PM 裁决覆盖。

---

## 9. 证据附录（勘察原始输出）

### 9.1 上游 commit 存在性与改动面

```
$ git cat-file -e e7f37dac9fe94390cf0528e3db8ee5d5fbbb8c8e && echo OK
OK

$ git show --stat e7f37dac9fe94390cf0528e3db8ee5d5fbbb8c8e
commit e7f37dac9fe94390cf0528e3db8ee5d5fbbb8c8e
Author: axfinn <axiu@jaxiu.cn>
Date:   Thu Oct 15 19:27:55 2026 +0800
    feat(ssrules): Shadowsocks 规则文件生成器 —— 域名列表生成 ACL + 多缀公开链接
 backend/app_handlers.go                 |   5 +
 backend/app_http.go                     |   4 +
 backend/config.example.yaml             |   9 ++
 backend/config/config.go                |   9 ++
 backend/handlers/ssrule.go              | 204 +++++++++++++++++++++++++
 backend/handlers/ssrule_rules.go        | 164 +++++++++++++++
 backend/handlers/ssrule_rules_test.go   | 205 ++++++++++++++++++++++++
 backend/handlers/ssrule_test.go         | 262 ++++++++++++++++++++++++
 backend/models/ssrule.go                | 123 +++++++++++++++
 backend/routes.go                       |  78 +++++-----
 backend/routes/index.go                 |   2 +
 backend/routes/ssrule.go                |  26 ++++
 frontend/src/router/index.js            |  11 ++
 frontend/src/views/other/SsRuleTool.vue | 244 ++++++++++++++++++++++
 14 files changed, 1308 insertions(+), 38 deletions(-)
```

### 9.2 R3 / R4 / S3 / S4 / S7 自查（上游文件）

```
$ git show e7f37dac…:frontend/src/views/other/SsRuleTool.vue | sed -n '/<template>/,/<\/template>/p' | grep -n "\.value"
0 hits

$ git show e7f37dac…:frontend/src/views/other/SsRuleTool.vue | sed -n '/<template>/,/<\/template>/p' | grep -nE "[A-Za-z_$][A-Za-z0-9_$]*\s*\["
0 hits

$ for f in <5 个上游新文件>; do git show e7f37dac…:$f | grep -n "console\.log\|fmt\.Print\|TODO\|FIXME\|XXX\|HACK"; done
--- backend/handlers/ssrule.go ---            0 hits
--- backend/handlers/ssrule_rules.go ---      0 hits
--- backend/models/ssrule.go ---              0 hits
--- backend/routes/ssrule.go ---              0 hits
--- frontend/src/views/other/SsRuleTool.vue --- 0 hits

$ 同上，注释掉的代码块正则（^\s*//\s*(if |for |return |func |c\.[A-Z]|[a-zA-Z_]+ :?=)）
5 个文件全部 → 0 hits

$ 前端 catch 块（无空 catch）
176-  } catch {
177-    ElMessage.error('读取规则文件失败')
200-  } catch {
201-    ElMessage.error('保存失败')
214-  } catch {
215-    return // 用户取消
230-  } catch {
231-    ElMessage.error('换发失败')
```

### 9.3 gofmt 事实（用于 §6.1 ③ 的期望值）

```
$ gofmt -s -l <上游 commit 里本单涉及的 12 个文件>     # 在 commit 状态下
(空 = 全部干净)

$ gofmt -s -l <origin/main 同批文件>
backend/routes.go          ← origin/main 上本来就不干净（上游顺手修了它）

$ gofmt -s -l backend/ | wc -l
35                        ← 仓库既有 gofmt 红项数量（与本单无关，S9 不碰）
```

### 9.4 config 分叉（P0-4 的直接证据）

```
$ git diff --stat 9992ac652553752cef5c73f6c43012e9b69a7e33 origin/main -- <12 个被改文件>
 backend/config.example.yaml | 1 +
 backend/config/config.go    | 2 +-
 2 files changed, 2 insertions(+), 1 deletion(-)

$ git diff 9992ac6 origin/main -- backend/config/config.go backend/config.example.yaml
--- a/backend/config/example.yaml
+++ b/backend/config.example.yaml
@@ -235,6 +235,7 @@ minimax_mcp:
   args:
     - "minimax-coding-plan-mcp"
+    - "-y"
--- a/backend/config/config.go
+++ b/backend/config/config.go
@@ -527,7 +527,7 @@ func DefaultConfig() *Config {
-			Args:           []string{"minimax-coding-plan-mcp"},
+			Args:           []string{"minimax-coding-plan-mcp", "-y"},
```

### 9.5 限流器共用与**行号订正**（P0-2 / P0-6 的直接证据）

```
$ git show origin/main:backend/app_http.go | grep -n "createRateLimiter :="
39:  createRateLimiter := middleware.NewRateLimiter(10, time.Minute, rt.transientStore)

$ git show origin/main:backend/middleware/ratelimit.go | grep -n "func (rl \*RateLimiter) Allow\|ip := c.ClientIP()\|func NewRateLimiter\|func ContentSizeLimiter"
20:func NewRateLimiter(limit int, window time.Duration, store state.TransientStore) *RateLimiter {
32:func (rl *RateLimiter) Allow(ip string) bool {
49:		ip := c.ClientIP()
63:func ContentSizeLimiter(maxSize int64) gin.HandlerFunc {

$ git show origin/main:backend/state/transient.go | grep -n "fullKey := s.key(\"ratelimit:\""
242:	fullKey := s.key("ratelimit:" + key)      ← key 里没有端点名 / limiter 身份
   ⚠️ FINN-90 评审稿写的是 :240 —— 实测为 :242，本版订正

$ git show origin/main:backend/models/paste.go | grep -n "func generateID"
245:func generateID(length int) string {
   ⚠️ FINN-90 评审稿写的是 :244 —— 实测为 :245，本版订正

$ git show origin/main:backend/app_http.go | grep -n "ContentSizeLimiter(200"
35:	router.Use(middleware.ContentSizeLimiter(200 * 1024 * 1024))
   ✅ 评审稿的 :35 正确

$ grep -rn "SetTrustedProxies" backend/
0 hits                          ← gin 默认 trustedProxies = 0.0.0.0/0 全部生效
```

### 9.6 gin v1.9.1 路由内部（§3.4.1(a) 与 §3.4.4(a) 的直接证据）

```
$ GIN=$(go env GOMODCACHE)/github.com/gin-gonic/gin@v1.9.1

$ grep -n "RedirectTrailingSlash:\|ForwardedByClientIP:\|RemoteIPHeaders:\|UseRawPath:\|trustedProxies:" $GIN/gin.go
192:		RedirectTrailingSlash:  true,
195:		ForwardedByClientIP:    true,
196:		RemoteIPHeaders:        []string{"X-Forwarded-For", "X-Real-IP"},
198:		UseRawPath:             false,
205:		trustedProxies:         []string{"0.0.0.0/0", "::/0"},

$ grep -n "func (engine \*Engine) validateHeader\|func (engine \*Engine) handleHTTPRequest\|rPath := c.Request.URL.Path" $GIN/gin.go
458:func (engine *Engine) validateHeader(header string) (clientIP string, valid bool) {
592:func (engine *Engine) handleHTTPRequest(c *Context) {
594:	rPath := c.Request.URL.Path

$ sed -n '458,477p' $GIN/gin.go   # validateHeader：从右往左，命中 i==0 就返回 ⇒ XFF 最左项
	for i := len(items) - 1; i >= 0; i-- {
		ipStr := strings.TrimSpace(items[i])
		ip := net.ParseIP(ipStr)
		if ip == nil { break }
		if (i == 0) || (!engine.isTrustedProxy(ip)) {
			return ipStr, true
		}
	}
```

### 9.7 请求日志记路径（P1-2 的直接证据）

```
$ git show origin/main:backend/middleware/request_logger.go | grep -n "Path:\|func shouldSkipMonitoringRequest"
55:			Path:          truncateMonitoringValue(c.Request.URL.Path, 512),
73:func shouldSkipMonitoringRequest(method, path, upgrade string) bool {

$ git show origin/main:backend/middleware/request_logger.go | awk 'NR>=77 && NR<=87'
77:	for _, prefix := range []string{
78:		"/api/monitor",
79:		"/api/health",
80:		"/assets/",
81:		"/neon/",
82:		"/api/bg/cached/",
83:	} {
   ↑ 不含 "/ss/"
```

### 9.8 `:memory:` 池约定（R5）与前端别名 / 路由

```
$ grep -rn "func newMemDB" backend/handlers/*.go
backend/handlers/skills_db_test.go:24:func newMemDB(t *testing.T) *models.DB {
   ↑ 内部已含 db.SetMaxOpenConns(1)

$ grep -rn "func mustJSON" backend/handlers/*.go
0 hits                       ← ssrule_test.go 新引入，无重名冲突

$ grep -n "func (db \*DB) SetMaxOpenConns" backend/models/paste.go
235:func (db *DB) SetMaxOpenConns(n int) {     ← package main 的测试用它满足 R5

$ grep -n "other:" frontend/src/router/index.js
11:  other: { name: '系统管理', icon: 'Tools', description: '代理、控制台与辅助工具', accent: '#475569', order: 70 }

$ grep -n "icon: 'Share'" frontend/src/router/index.js
112:  icon: 'Share',   225:  icon: 'Share',   236:  icon: 'Share'      ← 图标名合法

$ grep -n "const allTools" frontend/src/composables/useToolRegistry.js
13:  const allTools = computed(() => toolRoutes.filter(isNavigableTool))   ← 工具清单自动派生，无需另注册
```

### 9.9 ★ `/ss/` 路由形态的**实测**（§3.4.1(d) 的直接证据 —— 非阻塞项 B1 的核心）

> 用一个**独立的临时模块**（`/tmp/ginprobe/`，复用项目 `go.mod`/`go.sum`，**不在仓库内、不启动任何服务**）复刻 §3.4.1(d) 的 4 条路由 + NoRoute 特判，对 §3.4.1(c) 的载荷逐条发请求。
> 目的：确认 ① 4 条路由**共存不 panic** ② 所有 `/ss/` 形态都**不产生 3xx** ③ 各载荷的真实落点。

```
$ cat /tmp/ginprobe/main.go        # 关键片段
    r.GET("/ss/:token/:filename", download)
    r.GET("/ss/:token/:filename/", nf404)
    r.GET("/ss/:token/", nf404)
    r.GET("/ss/:token", nf404)
    r.NoRoute(noRoute)             # noRoute: path 前缀 "/ss/" → 404「规则文件不存在」，否则 SPA

$ cd /tmp/ginprobe && GOFLAGS=-mod=mod GOPROXY=off go run .
== F: 4 条显式路由(无尾斜杠/有尾斜杠 各 404) ==
      [handler] fullPath=/ss/:token/:filename filename="rules.acl"
  /ss/TOK/rules.acl              -> 200 loc=""                       body="# ACL BODY for rules.acl"  <<< NOT 404
  /ss/TOK/rules.acl/             -> 404 loc=""                       body="规则文件不存在"
  /ss/TOK/                      -> 404 loc=""                       body="规则文件不存在"
  /ss/TOK                        -> 404 loc=""                       body="规则文件不存在"
  /ss/TOK/                       -> 404 loc=""                       body="规则文件不存在"
  /ss                            -> 200 loc=""                       body="<!DOCTYPE html>index.html"  <<< NOT 404
      [handler] fullPath=/ss/:token/:filename filename="rules.pac"
  /ss/TOK/rules.pac              -> 404 loc=""                       body="规则文件不存在"
      [handler] fullPath=/ss/:token/:filename filename="rules.acl.txt"
  /ss/TOK/rules.acl.txt          -> 404 loc=""                       body="规则文件不存在"
  /ss/TOK/../../etc/passwd       -> 404 loc=""                       body="规则文件不存在"
  /ss/TOK/..%2f..%2fetc%2fpasswd -> 404 loc=""                       body="规则文件不存在"
  /ss/TOK/%2e%2e%2f%2e%2e%2fpasswd -> 404 loc=""                    body="规则文件不存在"
  /ss/TOK/....//....//etc/passwd -> 404 loc=""                       body="规则文件不存在"
      [handler] fullPath=/ss/:token/:filename filename="rules.acl\x00.txt"
  /ss/TOK/rules.acl%00.txt       -> 404 loc=""                       body="规则文件不存在"
      [handler] fullPath=/ss/:token/:filename filename="rules.acl\x00"
  /ss/TOK/rules.acl%00           -> 404 loc=""                       body="规则文件不存在"
      [handler] fullPath=/ss/:token/:filename filename="rules.acl\\..\\x"
  /ss/TOK/rules.acl\..\x         -> 404 loc=""                       body="规则文件不存在"
      [handler] fullPath=/ss/:token/:filename filename="index.html"
  /ss/TOK/index.html             -> 404 loc=""                       body="规则文件不存在"
      [handler] fullPath=/ss/:token/:filename filename="rules.yaml"
  /ss/TOK/rules.yaml             -> 404 loc=""                       body="规则文件不存在"
      [handler] fullPath=/ss/:token/:filename filename="rules"
  /ss/TOK/rules                  -> 404 loc=""                       body="规则文件不存在"
  非 404 计数 = 2
```

**从这份实测读出的 5 条结论（全部写进 §3.4.1(c)/(d)/(e)）：**

1. **4 条路由共存，不 panic** —— gin v1.9.1 的 radix tree 允许 `/ss/:token/:filename` 与 `/ss/:token/:filename/`、`/ss/:token/`、`/ss/:token` 同时注册（`:filename` 与 `:filename/` 落在同一 param 节点的不同层级，不是两个通配符）。⇒ §7.1 R-11 的 panic 风险**实测排除**。
2. **所有 `/ss/` 形态都不产生 3xx** —— 输出里 `loc=` **全为空** ⇒ 没有 301/302。**尾斜杠载荷不再被 301 带走**（§3.4.1(d) 改动 2 生效）。⇒ 断言 **A3-HTTP-3xx** 有实测支撑。
3. **除两条外全部 404 + body 逐字节 `规则文件不存在`** —— 两条例外是：① `/ss/TOK/rules.acl`（合法白名单 → 200 + ACL 正文，**这是正确行为**，见 A2）；② 裸 `/ss`（→ 200 + SPA HTML，因为 NoRoute 的前缀是 `/ss/` 不含裸 `/ss`，见 §8 Q7）。
4. **载荷 1–4（`../`、`..%2f`、编码点、双斜杠）根本没进 handler** —— 输出里**没有**对应的 `[handler] fullPath=…` 行 ⇒ **印证 §3.4.1(a)**：解码后段数变多 ⇒ 3 段路由不匹配 ⇒ 走 NoRoute，**不是**"单个 `:filename`"。⇒ ⛔ v1 的机制论断在此作废。
5. **白名单类载荷（5、6、8、9）进了 handler 但被白名单拒绝** —— 输出里**有** `[handler] fullPath=… filename="rules.pac"` 这类行 ⇒ **印证 §3.4.1(b)**：安全由「白名单精确相等 + 不触碰文件系统」支撑，**与"解码后不含 `/`"无关**（⛔ v1 那条论据作废）。

⇒ 由此 **§3.4.1(c) 的 10 行期望统一为 `404` + body 逐字节 `规则文件不存在`**；其中第 7 行（`rules.acl/`）由占位路由 `/ss/:token/:filename/` 兜住 —— ⚠️ 它匹配到的 `filename` 仍是 `rules.acl`（尾斜杠是独立的空段，不进 param），**所以这条路由绝不能指向 `Download`**，否则会返回 200 + ACL 正文（比 301 更糟）。

### 9.10 上游实现实读（关键行 —— §3.2.1 / §3.2.3 / §5.3 P0-1 / P0-3 的直接证据）

```
$ git show e7f37dac…:backend/handlers/ssrule.go | grep -n "func \|c.Query(\"admin_password\")\|newSSRuleView(file, nil)\|buildACLRules\|c.Param(\|Cache-Control\|Content-Type"
24:func NewSSRuleHandler(db *models.DB, adminPassword string) *SSRuleHandler {
42:func ssRuleIsAllowedFile(name string) bool {
55:func (h *SSRuleHandler) checkAdmin(c *gin.Context) bool {
61:		given = c.Query("admin_password")
83:func newSSRuleView(file *models.SSRuleFile, skipped []string) ssRuleView {
102:func (h *SSRuleHandler) Verify(c *gin.Context) {
111:func (h *SSRuleHandler) Get(c *gin.Context) {
125:	content, skipped, err := buildACLRules(file.Source)
135:func (h *SSRuleHandler) Save(c *gin.Context) {
147:	content, skipped, err := buildACLRules(req.Source)
164:func (h *SSRuleHandler) RegenerateToken(c *gin.Context) {
175:	c.JSON(http.StatusOK, newSSRuleView(file, nil))
182:func (h *SSRuleHandler) Download(c *gin.Context) {
183:	filename := c.Param("filename")
189:	file, err := h.db.GetSSRuleFileByToken(c.Param("token"))
201:	c.Header("Content-Type", ssRuleFileContentType)
202:	c.Header("Cache-Control", "no-store")
   ↑ 上游**没有** X-Content-Type-Options（非阻塞项 10 的依据）；也没有 ssRuleNotFoundBody 常量（§3.2.2 要求新增）

$ git show e7f37dac…:backend/handlers/ssrule.go | awk 'NR>=55 && NR<=64'
55: func (h *SSRuleHandler) checkAdmin(c *gin.Context) bool {
56: 	if h.adminPassword == "" {
57: 		return false // 未配置密码 = 功能关闭,而不是「无密码放行」
58: 	}
59: 	given := c.GetHeader("X-Super-Admin-Password")
60: 	if given == "" {
61: 		given = c.Query("admin_password")
62: 	}
63: 	return subtle.ConstantTimeCompare([]byte(given), []byte(h.adminPassword)) == 1
64: }
   ↑ 裁决 5：Stage 4 删掉 **60–62 这 3 行**（query 兜底）—— 只留 header

$ git show e7f37dac…:backend/routes/ssrule.go | awk '{printf "%d: %s\n", NR, $0}'
13: func RegisterSSRuleRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
14: 	if h.SSRuleHandler == nil {
15: 		return
16: 	}
17: 	g := api.Group("/ssrules")
18: 	{
19: 		// 密码门也要限流:这是本模块唯一的密码校验点,
20: 		// 不限流等于给了无限次爆破机会。
21: 		g.POST("/verify", createRateLimiter.Middleware(), h.SSRuleHandler.Verify)
22: 		g.GET("", h.SSRuleHandler.Get)
23: 		g.POST("", createRateLimiter.Middleware(), h.SSRuleHandler.Save)
24: 		g.POST("/regenerate-token", createRateLimiter.Middleware(), h.SSRuleHandler.RegenerateToken)
25: 	}
26: }
   ↑ :22 的 GET **确实没有限流**（P0-1 的直接证据）；4 个端点**共用同一个 createRateLimiter**（P0-2）

$ git show origin/main:backend/app_http.go | awk 'NR>=78 && NR<=103'   # NoRoute 现状（§3.4.1(d) 改动 1 的插入点）
78: 	router.NoRoute(func(c *gin.Context) {
79: 		path := c.Request.URL.Path
80: 		if path == "/api" || strings.HasPrefix(path, "/api/") {
81: 			c.JSON(http.StatusNotFound, gin.H{"error": "接口不存在", "path": path})
82: 			return
83: 		}
84:
85: 		// hashed chunk 文件名带 hash 即可长缓存;其他文件(包含 index.html 兜底)
86: 		// 必须每次校验,避免浏览器拿旧版引用旧 chunk。
87: 		if strings.HasPrefix(path, "/assets/") && (strings.Contains(path, "-") || strings.Contains(path, ".")) {
88: 			c.Header("Cache-Control", "public, max-age=31536000, immutable")
89: 		} else {
90: 			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
91: 		}
92:
93: 		cleanPath := filepath.Clean(strings.TrimPrefix(path, "/"))
94: 		if cleanPath != "." && cleanPath != "" && !strings.HasPrefix(cleanPath, "..") {
95: 			filePath := filepath.Join(distDir, cleanPath)
96: 			if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
97: 				c.File(filePath)
98: 				return
99: 			}
100: 		}
101:
102: 		c.File(distDir + "/index.html")
103: 	})
   ↑ ★ 插入点 = **:83 之后**（`/api` 特判之后、:87 的静态文件分支之前）
   ↑ ★ :93-94 的 filepath.Clean + HasPrefix("..") 是**既有**的目录穿越防护 —— 但它只保护静态文件分支，
     与 §3.4.1 的规则分发端无关（分发端在 :102 之前根本走不到这里，除非路由没注册）
```

### 9.11 `frontend/src/router/index.js` 在 `9992ac6` → `origin/main` **0 差异**（非阻塞项 7 的证据）

```
$ git diff --stat 9992ac652553752cef5c73f6c43012e9b69a7e33 origin/main -- frontend/src/router/index.js
(空 = 0 差异)

$ git diff 9992ac6 origin/main -- frontend/src/router/index.js
(空 = 0 差异)

$ grep -cE '^\| [0-9]+ \|' AGENTS.md
42
$ grep -n '^## [0-9]* 模块地图' AGENTS.md
7:## 34 模块地图
$ grep -n '模块地图' AGENTS.md
7:## 34 模块地图
223:6. 更新本文件 `34 模块地图` 表格

$ git ls-tree -r origin/main --name-only | grep -ci ssrule
0
```

⇒ **前端取件**：`frontend/src/views/other/SsRuleTool.vue` 是新文件，可单路径 checkout；`frontend/src/router/index.js` **上游也改了 11 行**，但它在 `9992ac6`→`origin/main` 之间**没动过** ⇒ 单路径 checkout 是安全的（§5.2.1 已给出该行）。
⇒ **AGENTS.md 口径**：表内当前 **42** 行、标题写的是 **34**（既有红项）⇒ 本单加完第 43 行后，标题必须同步为 **43**（裁决 4，断言 A17）。

---

## 10. 硬约束（下游无条件遵守）

> 这些是**项目级硬约束**在 ssrules 模块上的落地形态。违反任意一条 = 判 FAIL。

| # | 约束 | 本模块的具体含义 |
|---|---|---|
| **R2** | 工具范围 = 只能服务端做的 | 「域名列表 → ACL 正文 + 落库 + 公开分发」必须服务端做；**不要**把"格式转换 / 拼 URL"设计成后端端点（前端本地拼即可，见 §3.7 的 `urls` computed） |
| **R3** | Vue 3 模板里永远不要写 `.value` | §3.7 + 断言 F1 |
| **R4** | Vue 模板裸下标 `obj.x[0]` → `obj.x?.[0] \|\| ''` | 本模块模板**不需要**裸下标（用 `v-for`）；若新增代码写了裸下标 = FAIL（断言 F2） |
| **R5** | SQLite `:memory:` 测试必须 `SetMaxOpenConns(1)` | handler 级测试复用 `newMemDB(t)`（已含）；package main 的 HTTP 级测试**必须**紧跟 `db.SetMaxOpenConns(1)`（§6.2 末尾） |
| **R6** | 后端分享资产必须用上游 http(s) URL | 本模块**不涉及**分享资产（分发的是自产的 ACL 文本，不是 blob）→ F6 |
| **R8** | 长耗时 AI 调用必须异步 | 本模块**无**长耗时调用（纯本地字符串生成 + SQLite）→ 不适用 |
| **R9** | MCP 协议版本白名单 | 本模块**不涉及** MCP → 不适用 |
| **R10** | Planner 改动对齐 `PLANNER_PHILOSOPHY.md` | 本模块**不涉及** Planner → 不适用 |
| **R11** | 后端跨平台编译走 Docker | §6.1 的编译/静态检查**全部在容器内**跑，**不在 macOS 上出二进制** |
| **别名** | `@` 指向 `src/neon` 不是 `src` | 本组件一律用**相对路径** `../../api` / `../../composables/useAdminAuth`（§3.7） |
| **SMB** | `/Volumes/M20` 是 SMB 网络挂载 | **不要**在该盘上建 SQLite 库 / `npm install` / 落大量小文件；前端测试走 `/tmp` scratch（§6.4 ①） |
| **部署** | 部署由 jaxiu 手工 `./deploy.sh docker` | 智能体**不部署、不启动服务、不探测部署主机** `192.168.31.201`；线上验收由 jaxiu 做（§11 未覆盖项） |

---

## 11. 交付前 checklist（Stage 4 / Stage 5 逐条打勾）

**Stage 4（后端单）**：

- [ ] 按 §5.2.1 **单路径** checkout 10 个后端文件；`backend/routes.go` **只加 2 行** + `gofmt -w`；`backend/config/config.go` 与 `config.example.yaml` **手工打补丁**（不整文件 checkout）
- [ ] 修 P0-1（GET 挂限流）、P0-2（独立 store，绝不回落 `createRateLimiter`）、P0-3（`Download` 里加 enabled 判断）、P0-4（config 手工补丁）、P0-5（载荷用例）、P0-6（模块内限流中间件 + `ssRuleClientIP`）、P0-7（NoRoute 特判 + 3 条占位路由）
- [ ] 修 P1-1（唯一索引）、P1-2（skip 列表加 `/ss/`）、P1-3（组级 `ContentSizeLimiter(1<<20)`）、P1-4（分发端 120/min）、P1-5（`headerName`）、P1-6（`Content-Disposition` 可选）
- [ ] 裁决 5：删掉 `checkAdmin` 的 60–62 行（query 兜底）
- [ ] 裁决 6：`updated_at` 一律 `time.Now().UTC()`
- [ ] 新增 `ssRuleNotFoundBody` 常量并**导出**（`handlers.SSRuleNotFoundBody`），handler 与 NoRoute 两处共用
- [ ] 分发端响应头加 `X-Content-Type-Options: nosniff`
- [ ] `RegenerateToken` 重跑 `buildACLRules` ⇒ `skipped` 是实际值
- [ ] `AGENTS.md`：加 `#43 SSRules` 行 + 标题/引用同步为 **43**（`grep -cE '^\| [0-9]+ \|'` 为准）；**「73 个 View 组件」不动**
- [ ] §6.1 的 ①②③ 全绿（gofmt 只查本单文件，期望 **0 行输出**）
- [ ] `git diff --name-only origin/main..HEAD` 只含本单文件（无 `compress.go`、无 `.multica/`、无 `CLAUDE.md`）

**Stage 5（测试单）**：

- [ ] handler 级 **A3**（14 个注入值）+ HTTP 级 **A3-HTTP**（14 条 URL）+ **A3-HTTP-3xx**
- [ ] **A6**（5 个端点阈值 + 桶隔离）、**A13**（限流键不可伪造）、**A16**（413 vs 400）
- [ ] **A15**（`skipped` 实际值）、**A17**（AGENTS.md 计数 == 标题数字）、**A18**（query 兜底已删）、**A19**（模块关闭语义）、**A20**（取件纪律）
- [ ] **反事实自检**：每条断言产出「基线 / 被 FAIL 版 / 修复分支」三组输出（§6.5）
- [ ] 前端 **F1–F8**（`SsRuleTool.test.js` 必须存在）
- [ ] `go test` **只跑本模块**（`-run 'TestSSRule|TestBuildACLRules'`），**不跑 `./handlers/` 全包**（既有红项）

**不属于本单的验收**（由 jaxiu 合并后自行上线验收，**不作为本单阻塞项**）：

- 真实客户端（shadowsocks-libev / rust / android）能否正常拉取规则并生效
- `/ss/` 响应经 Cloudflare 压缩后的实际表现
- 线上容器内 4 服务（devtools / redis / ocr-service / asr-service）的实际连通性
- `t.jaxiu.cn` 公网暴露面的实际表现

---

## 12. 变更记录

| 版本 | 来源 | 变更 |
|---|---|---|
| v1 | Stage 1 初版 | 全模块落地方案（API 契约 / 数据模型 / 接线 / 分工 / 取件纪律） |
| v2 | Stage 1（未完成） | 起了个头（到 §3.4.1 为止），**未完成**；且其中「实测（§9.9）」指向一个**并不存在的 §9.9** —— 属未验证引用 |
| **v3** | **FINN-102（本次）** | ① **B1** 重写 §3.4.1：订正 gin 匹配机制（`UseRawPath` 默认 false ⇒ 用解码后 `URL.Path` ⇒ `%2f` 变真 `/` ⇒ 段数变多 ⇒ 3 段路由不匹配 ⇒ NoRoute，**不是**"单个 `:filename`"）；保留「不触碰文件系统 + 白名单精确相等」两条安全论据；10 行载荷表期望**统一**为 `404` + body 逐字节 `规则文件不存在`；断言拆 **handler 级 / HTTP 级**两层<br>② **B2** 重写 §3.4.4：限流键 = `CF-Connecting-IP` → XFF **最右** → `RemoteAddr` host；模块内自建中间件 + `middleware.RateLimiter.Allow(ip)`（**不动共享 `RateLimiter`**）；写明残余风险<br>③ 裁决 1–6 全部落地（分桶限流 / 未匹配 `/ss/` → 404 / NoRoute 特判 + 关闭尾斜杠重定向 / AGENTS.md 口径写死 / 删 query 兜底 / `updated_at` 统一 UTC）<br>④ 12 条非阻塞项全部处置（文本见 §5.4）<br>⑤ §6.3 断言 A1–A20 + A3-HTTP + A3-HTTP-3xx、§6.4 断言 F1–F8，**每条带反事实自检**<br>⑥ §9 证据附录补齐 §9.9（**真实实测输出**）/ §9.10 / §9.11，并订正 3 处行号漂移<br>⑦ §7.2 回滚① 改成「**重启后生效**」；路径② 明确 **NoRoute 的 `/ss/` 特判必须保留** |

> **v3 作废的 v1/v2 论断**（⛔ 标记处）：①「`%2f` 解码后成为单个 `:filename` 段」；②「安全依赖解码后不含 `/`」；③「回滚①秒级生效」；④「`/ss/` 载荷表 5/10 行的真实落点」；⑤ 限流键「用现成的 `createRateLimiter`」。
