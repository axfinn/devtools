# ssrules 模块 —— 落地方案

> Stage 1（devtools-architect）产出。本方案基于对上游 commit `e7f37dac9fe94390cf0528e3db8ee5d5fbbb8c8e`
> （本地独有，origin 无）与基线 `origin/main`（`81199bd`，含 `542e457`）的**逐行实读勘察**。
> 下游 Stage 2 评审 / Stage 3 前端 / Stage 4 后端 / Stage 5 测试照此执行。
>
> **本方案里每条 `file:line` 都实读验证过**，勘察命令与原始输出见 §9（证据附录）。

---

## 0. 一页速览（给下游的 60 秒版）

| 项 | 结论 |
|---|---|
| 模块 | 全站**唯一一份** Shadowsocks 分流规则文件；管理页填域名/URL 列表 → 生成 ACL 正文 → 公开链接分发 |
| 对外端点 | 管理端 4 个（`/api/ssrules*`，需 `X-Super-Admin-Password`）+ 分发端 1 个（`/ss/:token/:filename`，token 门） |
| 分发端点 | **必须**在 `registerPublicRoutes` 显式注册（否则被 NoRoute 的 SPA fallback 兜成 `index.html`，客户端拿到 HTML 却"成功"） |
| 前端 | `frontend/src/views/other/SsRuleTool.vue` + `frontend/src/router/index.js` 一条路由 |
| 取件方式 | `git checkout e7f37dac -- <path>` **单路径**取件；**`backend/config/config.go` / `config.example.yaml` 除外，必须手工打补丁**（见 §5.3 P0-4） |
| **必须先修的 5 个 P0** | 限流绕过 / 限流器共用 / 模块关闭语义不一致 / config 取件回退 FINN-56 / 路径穿越判据与回归用例 |
| 上游 commit 直接可用吗 | **否**。上游有 5 个 P0、6 个 P1 缺陷（§5.3），必须先修再进 Stage 4 |
| 回滚 | ① 配置关模块（依赖 P0-3 修好才成立）② 删路由（需同时堵 NoRoute 兜底，否则客户端拿到 HTML） |

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
| 修 `backend/` 里另外 34 个 gofmt 不干净的文件 | S9 越界改动；见 §9.3 证据 —— 那是仓库既有红项，不是本单引入 |

---

## 2. 现状（已核实）

### 2.1 基线事实

| 事实 | 证据 |
|---|---|
| 上游 commit 对象在本地对象库可见 | `git cat-file -e e7f37dac…` → `OBJECT OK`；`git show --stat` → 14 files / +1308 −38 |
| 上游 commit 的父提交 | `9992ac652553752cef5c73f6c43012e9b69a7e33` |
| `origin/main` = `81199bd`，且 `542e457` 是其祖先 | `git merge-base --is-ancestor 542e457… origin/main` → rc=0 |
| 本仓 `main` 上**不存在**任何 ssrule 文件 | `git ls-tree -r origin/main --name-only \| grep -i ssrule` → 0 命中 |
| 上游 commit 的父树 vs `origin/main`：**只有 2 个文件分叉** | `git diff --stat 9992ac6 origin/main -- <12 个改动文件>` → `backend/config/config.go`（1 行）+ `backend/config.example.yaml`（1 行） |

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
| `backend/routes/index.go:49` | `HealthHandler *handlers.HealthHandler` | 之后加 `SSRuleHandler` 字段 |
| `backend/routes/index.go:91` | `RegisterTripRoutes(api, h, createRateLimiter)` | 之后加 `RegisterSSRuleRoutes(...)` |
| `backend/routes.go:12` | `type routeHandlers struct {` | 结构体加 `ssRuleHandler` 字段（**该文件 gofmt 不干净，见 §9.3**） |
| `backend/routes.go:91` | `HealthHandler: h.healthHandler,` | 之后加 `SSRuleHandler: h.ssRuleHandler,` |
| `backend/app_handlers.go:92` | `healthHandler := handlers.NewHealthHandler(...)` | 之后构造 `ssRuleHandler` |
| `backend/app_handlers.go:140` | `healthHandler: healthHandler,` | 之后加 `ssRuleHandler: ssRuleHandler,` |
| `backend/app_http.go:53` | `router.Any("/mock/:id", ...)` | 之后加 `/ss/:token/:filename` |
| `backend/config/config.go:54` | `Skills SkillsConfig \`yaml:"skills"\`` | 之后加 `SSRules` 字段 + `SSRulesConfig` 类型 |
| `backend/config/config.go:527-531` | `MiniMaxMCP.Args = []string{"minimax-coding-plan-mcp"}` | **`origin/main` 已是 `{"minimax-coding-plan-mcp", "-y"}`**（FINN-56）→ 整文件 checkout 会回退它 |
| `backend/config.example.yaml:236-237` | `minimax_mcp.args` | **`origin/main` 已多一行 `"-y"`** → 同上 |
| `frontend/src/router/index.js:105` | `},`（`/proxy` 路由块结束） | 之后插入 `/ssrules` 路由块 |
| `AGENTS.md:52` | 模块表最后一行 `#42 Trip 旅游行程` | 之后加 `#43 SSRules` |

### 2.4 相关既有基础设施（实读）

| 位置 | 事实 | 对本模块的影响 |
|---|---|---|
| `backend/app_http.go:39` | `createRateLimiter := middleware.NewRateLimiter(10, time.Minute, rt.transientStore)` | **全站唯一一个 limiter 实例** |
| `backend/middleware/ratelimit.go:47-58` | `Middleware()` 用 `c.ClientIP()` 作 key | **key 里没有端点名** |
| `backend/state/transient.go:129` / `:241` | Memory：`rateRequests[key]`；Redis：`s.key("ratelimit:" + key)` | **Redis 下所有 limiter 共用 `ratelimit:<ip>` 一个计数桶** |
| `backend/models/paste.go:245` | `generateID(length)` = `hex.EncodeToString(rand.Read(bytes))[:length]` | `generateID(32)` = **32 个 hex 字符 = 128 bit 熵**（crypto/rand） |
| `backend/middleware/request_logger.go:55` | `Path: truncateMonitoringValue(c.Request.URL.Path, 512)` | **完整 URL path 落 `http_request_logs` 表 → token 会入库** |
| `backend/middleware/request_logger.go:77-83` | skip 列表：`/api/monitor` `/api/health` `/assets/` `/neon/` `/api/bg/cached/` | **不含 `/ss/`** |
| `backend/app_http.go:78-103` | NoRoute：非 `/api/` 路径 → SPA fallback 返回 `index.html` | 分发端点必须显式注册，否则客户端拿到 HTML |
| `backend/app_http.go:31` | CORS `AllowHeaders` 含 `X-Super-Admin-Password` | 管理端用 header 传密码可直接用，无需改 CORS |
| `backend/app_http.go:37` | 全局 `ContentSizeLimiter(200 * 1024 * 1024)` | **200MB 请求体上限** |
| `backend/cleanup.go:22` | 每小时清理 goroutine | 见 §4.3 —— 本模块**不接入** |
| `backend/handlers/skills_db_test.go:24` | `newMemDB(t)` 已含 `db.SetMaxOpenConns(1)` | R5 已满足，`ssrule_test.go` 直接复用即可 |
| `frontend/src/composables/useAdminAuth.js` | 参数是 **`headerName`**，不是 `credentialField` | 上游 Vue 传错了参数名（见 §5.3 P1-5） |
| `frontend/src/router/index.js:11` | `other: { name: '系统管理', icon: 'Tools', order: 70 }` | 本模块归 `category: 'other'` |
| `frontend/src/composables/useToolRegistry.js:13` | 工具清单由 `toolRoutes` 自动派生 | **无需**额外注册表登记 |
| `frontend/src/router/index.js:112,225,236` | `icon: 'Share'` 已被 3 处使用 | 图标名合法 |
| `backend/routes/autodev.go:20-24` | 独立 limiter 的**写法先例**（但它最终又回落到了全局 limiter，见 §5.3 P0-2 分析） | 参考写法，不照抄回落逻辑 |

---

## 3. 设计

### 3.1 模块定位

> 全站只有**一份** Shadowsocks 分流规则文件。管理页填域名/URL 列表 → 生成 ACL 正文 → 公开链接分发给客户端作远程规则源。

**不做**规则引擎、不做多套规则、不做规则类型建模。存两份东西：

- `source` —— 用户原始输入（原样回显到编辑框，让用户下次还能改）
- `content` —— 按 ACL 模板生成好的正文（公开链接直接吐这个）

### 3.2 API 契约（全部对外端点）

#### 3.2.1 管理端（需密码）

鉴权：`X-Super-Admin-Password: <ssrules.admin_password>` 请求头。
**兼容**：`?admin_password=<pwd>` query 兜底（项目惯例，见 `backend/handlers/monitoring.go:249-251`）。头优先。

> **`ssrules.admin_password` 为空 ⇒ 整个模块关闭**（不是"无密码放行"）。实现位置：`SSRuleHandler.checkAdmin`（`backend/handlers/ssrule.go:55`）。
> 该判定必须**同时**作用于管理端与分发端 —— 见 §5.3 P0-3。

| # | 方法 | 路径 | 请求体 / 查询 | 成功响应 | 状态码 |
|---|---|---|---|---|---|
| M1 | `POST` | `/api/ssrules/verify` | 无（仅鉴权头） | `200 {"ok":true}` | 401 `{"error":"密码错误"}` / 429 `{"error":"请求过于频繁，请稍后再试","code":429}` |
| M2 | `GET` | `/api/ssrules` | 无（仅鉴权头） | `200` ssRuleView（§3.2.3） | 401 / 429 / 500 `{"error":"读取规则文件失败"}` |
| M3 | `POST` | `/api/ssrules` | `{"source":"<多行文本>"}` | `200` ssRuleView | 400 `{"error":"请求格式错误"}` / 400 `{"error":"输入过长(超过 512 KB),请分批处理"}` / 401 / 429 / 500 `{"error":"保存失败"}` |
| M4 | `POST` | `/api/ssrules/regenerate-token` | 无（仅鉴权头） | `200` ssRuleView（token 已换新） | 401 / 429 / 500 `{"error":"换发失败"}` |

#### 3.2.2 公开分发端（token 门，无密码）

| # | 方法 | 路径 | 鉴权 | 成功响应 | 状态码 |
|---|---|---|---|---|---|
| P1 | `GET` | `/ss/:token/:filename` | `token` 匹配 DB 中唯一行 | `200` `text/plain; charset=utf-8`，body = ACL 正文 | 404 `规则文件不存在`（纯文本，非 JSON） / 404（模块关闭时，见 P0-3） / 500 `读取规则文件失败` |

**响应头**（P1）：

```
Content-Type: text/plain; charset=utf-8
Cache-Control: no-store
```

- `Cache-Control: no-store` —— 已在上游实现。**必须有**：否则用户改完规则客户端拉不到新内容，"改了规则没生效"是极难排查的问题。
- **不设 `Content-Disposition`** —— 见 §3.4.3。

**`:filename` 白名单**（唯一允许的 4 个值，硬编码常量）：

```
rules.acl   rules.txt   rules.list   rules.conf
```

> 刻意**不含** `.pac`：PAC 是 JavaScript，把 ACL 正文当 `.pac` 发出去，客户端会静默拿到语法错误的脚本，报错完全指不到这里。

#### 3.2.3 管理端统一响应体 `ssRuleView`

```jsonc
{
  "source":     "google.com\nyoutube.com",        // 用户原始输入，回显到编辑框
  "content":    "# Shadowsocks 分流规则 …",          // 生成的 ACL 正文
  "token":      "3f2a…(32 hex)",                   // 当前分发 token
  "updated_at": "2026-10-22T12:34:56.789+08:00",   // RFC3339（mattn + _parse_time）
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
- `updated_at` 序列化形态由 `mattn/go-sqlite3` + `?_parse_time=true`（`backend/models/paste.go:71`）决定 → RFC3339 字符串 → 浏览器 `new Date()` 可解析。**QA 必须断言这一点**（若退化成 `0001-01-01T00:00:00Z`，页面会显示 1/1/1）。

### 3.3 token 语义（明确回答）

| 维度 | 结论 | 判据 / 实现位置 |
|---|---|---|
| **可枚举？** | **不可**。`generateID(32)` = `crypto/rand` 取 32 字节 → hex 截 32 字符 = **128 bit 熵** | `backend/models/paste.go:245`；调用点 `backend/models/ssrule.go` 的 `GetSSRuleFile` / `RegenerateSSRuleToken` |
| **有时效？** | **无**。token 一旦生成长期有效，直到手工换发 | 无 TTL 字段（表里没有 expires_at） |
| **一次性？** | **否**。同一个 token 可被任意客户端反复拉取 | `GET /ss/:token/:filename` 无状态校验 |
| **可换发？** | **可以**。`POST /api/ssrules/regenerate-token` 生成新 token，**旧链接立即 404** | `models.RegenerateSSRuleToken`（UPDATE token）+ 换发后旧 token 查不到行 → `sql.ErrNoRows` → 404 |
| **权限范围** | **只读**。token 不能编辑。编辑一律走 M1-M4 的密码门 | `checkAdmin` 只认 `X-Super-Admin-Password` / `?admin_password=`，**不认 token** |
| **token 是否入库明文** | 是（单行表）。这是本方案显式接受的风险（§5.3 P2-4） | `ss_rule_files.token TEXT` |

**失败响应不得泄露 token 是否存在** —— 判据：

- token 不存在 → `404` + body 恰为 `规则文件不存在`
- filename 不在白名单 → `404` + body 恰为 `规则文件不存在`（**同一句、同一状态码**）
- ⇒ 外部观察者无法区分「token 错」「文件名错」，也区分不了二者与「模块关闭」（P0-3 修好后模块关闭也回同一句）。

### 3.4 公网暴露安全（本单最高优先级）

#### 3.4.1 `:filename` 路径穿越防护

**实现位置**：`backend/handlers/ssrule.go` → `ssRuleIsAllowedFile(name string) bool`（上游在 `:42`）。

**判据（三重，任何一条不满足即 404）**：

1. **白名单精确相等**（不是前缀、不是后缀、不是 `strings.Contains`）：`name` 必须**逐字节等于** `rules.acl` / `rules.txt` / `rules.list` / `rules.conf` 之一。
2. **不触碰文件系统**：分发端点全程**不**做任何 `os.Open` / `filepath.Join` / `c.File`。→ 即使白名单被绕过，也没有任何文件读取面。
3. **token 只进参数化 SQL**：`db.conn.QueryRow("… WHERE token = ?", token)` → 无 SQL 注入面。

**必须覆盖的载荷（回归用例，见 §6.3 A3）**：

| 载荷 | 期望 |
|---|---|
| `../../etc/passwd` | 404 |
| `..%2f..%2fetc%2fpasswd` | 404 |
| `%2e%2e%2f%2e%2e%2fpasswd` | 404 |
| `....//....//etc/passwd` | 404 |
| `rules.acl%00.txt` / `rules.acl%00` | 404 |
| `rules.acl.txt`（双扩展名） | 404 |
| `rules.acl/` `rules.acl\..\x` | 404 |
| `rules.pac` / `index.html` / `rules.yaml` / `rules` | 404 |
| 空 filename（路由不匹配） | 404（非 `/api/` 路径会落到 NoRoute → **见 §7.2 的回滚陷阱**） |

> **注意**：Gin 默认按 `r.URL.Path`（已解码）匹配路由，因此 `..%2f..%2f` 解码后可能成为**单个** `:filename` 段的 `../../etc/passwd`。这正是必须依赖白名单的原因 —— 本模块**不依赖**"解码后就不含 `/`" 这种假设。

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
| `X-Content-Type-Options: nosniff` | 上游**未设** | P2：建议加，防浏览器 MIME 猜测 |
| `Content-Length` | 由 Gin 自动设 | ✅ |

**信息泄露面**（P2，显式记录）：

| 面 | 是否含 token | 缓解 |
|---|---|---|
| Cloudflare 访问日志 | **是**（CF 默认记完整 URL） | 不可控；靠换发止损 |
| 应用 `http_request_logs` 表 | **是**（`request_logger.go:55` 记完整 path） | P1-2：把 `/ss/` 加入 skip 列表 |
| 浏览器历史 / Referer | 是（用户点过链接时） | 靠换发止损 |
| 客户端配置文件 | 是（客户端本地存了规则 URL） | 靠换发止损 |

#### 3.4.4 分发端点是否需要限流

**需要。** 理由：

1. token 不可枚举 ⇒ **防不住**"已拿到 token 的人"反复拉；也没必要防（读一个纯文本，无副作用）。
2. 但 **`/ss/` 在公网上会被机器人无差别扫描**，每个随机 token 请求都会打一次 SQLite（`QueryRow`）。无限流 = 无界 DB 压力。
3. 因此给分发端一个**独立预算**的限流：`120 次/分钟/IP`。

**关键约束（详见 §5.3 P0-2）**：限流必须**独立计数桶**，不能与全站共用。实现方式见 §5.2.3。

#### 3.4.5 「未配密码 = 模块关闭」语义一致性

**这条必须落到端点行为上，不能只停留在管理端。**

| 场景 | 管理端 M1-M4 | 分发端 P1 |
|---|---|---|
| `admin_password` **已配置** | 密码正确 → 正常；错误 → 401 | token 正确 → 200；错误 → 404 |
| `admin_password` **为空**（模块关闭） | **全部 401**（`checkAdmin` 首行 `if h.adminPassword == "" { return false }`） | **必须 404**（`规则文件不存在`）—— ⚠️ **上游未实现，这是 P0-3** |

> **为什么这条重要**：如果用户先用密码存了规则，之后把 `admin_password` 清空以为"模块关了"，但 `/ss/<旧token>/rules.acl` 仍在吐旧内容 —— 公网上任何拿到链接的人照旧能拿到。**配置层面的"关闭"必须等于数据层面的"关闭"。**

#### 3.4.6 其余公网面

- `/api/ssrules*` 全部在 `/api` 分组下 → NoRoute 对 `/api/` 前缀返回 JSON 404，不会被 SPA fallback 兜住 ✅
- 分发端在 `/ss/` 前缀 → 显式注册 ✅
- 不返回内部拓扑 / 服务名 / 版本号 / 依赖健康状态 ✅（ssrule 响应只含规则内容与 token）
- 管理端响应含 `token`（= 分发凭据），但仅在密码正确时返回 ✅

### 3.5 数据模型

**新增表**（`backend/models/ssrule.go`）：

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
| PRIMARY KEY | `id` | 固定值 `"default"`（`ssRuleFileID` 常量）→ 全站单行 |
| UNIQUE INDEX | `token` | **P1-1 需补**；上游只有 PK，token 列无唯一约束（单行场景无实际危害，但语义上 token 应唯一） |

**生命周期**：

| 动作 | 触发 | 行为 |
|---|---|---|
| 建表 | `models.RegisterInit("Shadowsocks规则文件(ss_rule_files)", (*DB).InitSSRuleFiles)`（`backend/models/ssrule.go:9`）→ `InitAll()` 遍历执行（`backend/models/paste.go:56`） | `CREATE TABLE IF NOT EXISTS` |
| 懒创建 | `GetSSRuleFile()` 查不到行时 `INSERT OR IGNORE` 写入空行 + 新 token | 并发首次请求用 `INSERT OR IGNORE` 吞掉竞争，失败方重读 |
| 覆盖保存 | `SaveSSRuleFile(source, content)` | UPDATE `source`/`content`/`updated_at` |
| 换发 token | `RegenerateSSRuleToken()` | UPDATE `token` + `updated_at`；**旧链接立即失效** |
| 删除 | **无**（无任何删除入口） | 见下 |

**与每小时清理 goroutine 的对接**：`backend/cleanup.go:22` 的 `startCleanupRoutine` 每小时清过期粘贴板/聊天室/短链/Mock/MDShare/Excalidraw/… **本模块不接入清理**：

- 本表是**单行配置表**，不是有生命周期的事件数据 —— 没有"过期"语义
- 清理它等于"每小时把用户的规则文件删掉"，不可接受
- **⚠️ 下游注意**：`backend/cleanup.go` **不需要**任何改动。若有人"顺手"加一行 `db.CleanExpiredSSRuleFiles()` → 直接判 FAIL（无此方法，且语义错误）

**Redis 软依赖**：本模块**完全不依赖 Redis**。token 生成用 `crypto/rand`，存储用 SQLite，限流用进程内 MemoryStore（§5.2.3）。⇒ Redis 挂掉 / 降级时本模块行为不变 ✅

### 3.6 路由接线

| 位置 | 改动 |
|---|---|
| `backend/routes/index.go:49` 之后 | `SSRuleHandler *handlers.SSRuleHandler`（大写导出结构体，`package routes`） |
| `backend/routes/index.go:91` 之后 | `RegisterSSRuleRoutes(api, h, createRateLimiter)` |
| `backend/routes.go`（`type routeHandlers struct`，`:12`） | 加 `ssRuleHandler *handlers.SSRuleHandler`（小写，`package main`） |
| `backend/routes.go`（`setupRoutes`，`:91` 附近） | `SSRuleHandler: h.ssRuleHandler,` 传给 `routes.RouteHandlers` |
| `backend/app_handlers.go:92` 之后 | `ssRuleHandler := handlers.NewSSRuleHandler(db, cfg.SSRules.AdminPassword)` |
| `backend/app_handlers.go:140` 之后 | `ssRuleHandler: ssRuleHandler,` |
| `backend/app_http.go:53` 之后 | `router.GET("/ss/:token/:filename", handlers.ssRuleHandler.Download)` |
| `backend/routes/ssrule.go`（新文件） | `RegisterSSRuleRoutes` |

**两条易错点**（上游 commit message 已点出，必须在代码注释里保留）：

1. **`/ss/` 不能改成 `/s/`** —— `/s/:id` 已被短链占用（`backend/app_http.go:50`）。Gin 在同一位置遇到两个不同名的通配段会**启动即 panic**（wildcard conflict）。`/ss/` 首段不同，无冲突。
2. **必须显式注册进 `registerPublicRoutes`** —— 非 `/api/` 前缀的路径没注册就落 NoRoute → SPA fallback → 客户端拿到一坨 HTML，且报错完全指不到这里。

**`backend/routes/ssrule.go` 的骨架**（含 §5.3 的 P0/P1 修复）：

```go
func RegisterSSRuleRoutes(api *gin.RouterGroup, h *RouteHandlers, createRateLimiter *middleware.RateLimiter) {
	if h.SSRuleHandler == nil {
		return
	}
	// 独立预算的限流器（不复用全站 createRateLimiter —— 见方案 §5.3 P0-2）
	adminRL := middleware.NewRateLimiter(10, time.Minute, nil)
	g := api.Group("/ssrules", middleware.ContentSizeLimiter(1<<20))
	{
		g.POST("/verify", adminRL.Middleware(), h.SSRuleHandler.Verify)
		g.GET("", adminRL.Middleware(), h.SSRuleHandler.Get)          // ★ P0-1：上游漏了限流
		g.POST("", adminRL.Middleware(), h.SSRuleHandler.Save)
		g.POST("/regenerate-token", adminRL.Middleware(), h.SSRuleHandler.RegenerateToken)
	}
}
```

### 3.7 前端

**文件**：

| 文件 | 动作 |
|---|---|
| `frontend/src/views/other/SsRuleTool.vue` | 新建（244 行） |
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

**无**。本模块是全新表，**没有任何旧数据需要迁移或回填**。`git ls-tree -r origin/main | grep ssrule` → 0 命中。

### 4.2 旧接口 / 旧响应

**不破坏任何既有接口**：

- 新增端点全部是新路径（`/api/ssrules*` 与 `/ss/`）
- `/s/:id`（短链）**不受影响**（`/ss/` 是不同的首段）
- `backend/config/config.go` 新增 `SSRules` 字段 —— YAML 解析**向后兼容**：旧 `config.yaml` 没有 `ssrules` 段 → `SSRules.AdminPassword` 为零值 `""` → **模块默认关闭**，不影响任何既有功能 ✅

### 4.3 需要人工配置的一步（部署时）

`config.yaml` 需新增（**留空 = 关闭**，不是"无密码放行"）：

```yaml
ssrules:
  admin_password: "改成你自己的密码"
```

`config.example.yaml` 末尾追加同一段（上游已写好）。

### 4.4 路由与压缩中间件

- `backend/middleware/compress.go:135-154` 的 `excludedPathPrefixes` **不含 `/ss/`** ⇒ 分发端响应会被 gzip（客户端发 `Accept-Encoding: gzip` 时）。客户端（shadowsocks-libev / rust / android）都能处理 gzip，且没有 `Content-Length` 依赖问题。**不需要**改动 compress.go（S9 越界）。若 QA 发现某客户端不接受 gzip，再单独开单把 `/ss/` 加进排除列表。
- `backend/middleware/request_logger.go` 的 skip 列表建议加 `/ss/`（P1-2）—— **唯一**需要碰共享中间件的地方。

---

## 5. 前后端分工与取件纪律

### 5.1 文件归属

| 文件 | 归属单 | Stage |
|---|---|---|
| `backend/handlers/ssrule.go`（新） | **后端单** | 4 |
| `backend/handlers/ssrule_rules.go`（新） | **后端单** | 4 |
| `backend/handlers/ssrule_test.go`（新） | 后端单（Stage 4 补） | 4 |
| `backend/handlers/ssrule_rules_test.go`（新） | 后端单 | 4 |
| `backend/models/ssrule.go`（新） | **后端单** | 4 |
| `backend/routes/ssrule.go`（新） | **后端单** | 4 |
| `backend/routes/index.go` | **后端单** | 4 |
| `backend/routes.go` | **后端单** | 4 |
| `backend/app_handlers.go` | **后端单** | 4 |
| `backend/app_http.go` | **后端单** | 4 |
| `backend/config/config.go` | **后端单**（⚠️ 手工补丁，见 §5.3） | 4 |
| `backend/config.example.yaml` | **后端单**（⚠️ 手工补丁，见 §5.3） | 4 |
| `backend/middleware/request_logger.go` | **后端单**（P1-2 一行） | 4 |
| `AGENTS.md`（模块表加 `#43`） | **后端单**（唯一边界例外） | 4 |
| `frontend/src/views/other/SsRuleTool.vue`（新） | **前端单** | 3 |
| `frontend/src/router/index.js` | **前端单** | 3 |
| `frontend/src/views/other/SsRuleTool.test.js`（新） | **前端单**（或 Stage 5） | 3 / 5 |

> `AGENTS.md` 的 34/42 模块地图按项目惯例归**后端单**更新（唯一边界例外）。**devtools-architect 不改 `AGENTS.md`**（写权边界：只写 `docs/plans/`）。

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
```

#### 5.2.2 **禁止**的取件方式

| 禁止 | 原因 |
|---|---|
| `git cherry-pick e7f37dac…` | 会把上游 commit 的**全部 14 个文件**（含 config 两个分叉文件）一起带进来 → 回退 FINN-56（见 P0-4），且违反「后端/前端分单」 |
| `git checkout e7f37dac… -- backend/` / `-- frontend/`（整目录） | 同上，跨单越界 |
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

> 上游 commit **不能直接落地**。以下 5 个 P0 + 6 个 P1 全部有实读证据（§9）。

#### P0-1 ★ `GET /api/ssrules` 没挂限流 → 密码爆破绕过限流

- **证据**：上游 `backend/routes/ssrule.go` 里 `g.GET("", h.SSRuleHandler.Get)` —— **没有** `createRateLimiter.Middleware()`；而 `POST /verify` 有。
- **为什么是 P0**：`GET` 与 `POST /verify` 是**同一个密码 oracle**（错密码都返回 401 `密码错误`）。攻击者只要改用 GET 就能绕过限流无限次爆破。上游 commit message 写「密码门也要限流：不限流等于给了无限次爆破机会」—— 但它自己漏了 GET。
- **修**：`g.GET("", adminRL.Middleware(), h.SSRuleHandler.Get)`
- **断言**：§6.3 A6

#### P0-2 ★ 限流器全站共用 ⇒ ssrules 的限流既挡不住爆破、又拖累别的模块

- **证据**：`backend/app_http.go:39` 只有一个 `createRateLimiter`（10/min）；`backend/middleware/ratelimit.go:47` 的 key **只有** `c.ClientIP()`；`backend/state/transient.go:241` 的 Redis key 是 `"ratelimit:" + key` —— **不含端点名、不含 limiter 身份**。
- **两个后果**：
  1. **Redis 后端下**，新建一个 `NewRateLimiter(...)` 实例**并不会**得到独立预算 —— key 还是 `ratelimit:<ip>`，照样和全站共用同一个计数桶。所以「给 ssrules 单开一个 limiter」这个做法**在 Redis 下是无效的**，必须给 limiter 配**独立 store**。
  2. **Memory 后端下**，`NewRateLimiter(n, w, nil)` 会自建一个私有 `MemoryStore`（`ratelimit.go:21-23`）⇒ 独立预算 ✅。`backend/routes/autodev.go:20-24` 的写法虽然传了 `nil`，但紧接着 `if createRateLimiter != nil { writeRL = createRateLimiter }` **又回落到了全局 limiter** —— **不能照抄那个回落**。
- **修**：ssrules 用 **`middleware.NewRateLimiter(N, time.Minute, nil)`（独立 store ⇒ 独立计数桶），且绝不回落到 `createRateLimiter`**。
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
  - `backend/config.example.yaml:238` → `args:` 下多一行 `"-y"`
- **为什么是 P0**：`git checkout e7f37dac… -- backend/config/config.go` 会把这两行**退回**（`-y` 消失）⇒ 直接回退 FINN-56 的 MCP 启动修复。
- **修**：**这两个文件禁止整文件 checkout**，必须手工打补丁：
  1. `backend/config/config.go`：`Config` 结构体（`:54` 之后）加 `SSRules SSRulesConfig \`yaml:"ssrules"\``；其后加 `SSRulesConfig` 类型定义（照上游，但用项目注释风格）。
  2. `backend/config.example.yaml`：**在文件末尾追加** `ssrules:` 段（不要动 `minimax_mcp` 那段）。
- **断言**：§6.3 A7（反向断言 `-y` 必须还在）

#### P0-5 ★ 路径穿越判据与回归用例缺失

- **证据**：上游 `ssrule_test.go` 的白名单用例只覆盖 `rules.pac` / `index.html` / `rules.yaml` / `rules` —— **没有任何 `../`、URL 编码、null byte、双扩展名载荷**。
- **修**：按 §3.4.1 的载荷表补齐用例，断言 **404 且 body 恰为 `规则文件不存在`**。
- **断言**：§6.3 A3

#### P1-1 `token` 列缺 UNIQUE 索引

`ss_rule_files.token TEXT NOT NULL` 无唯一约束 ⇒ 补 `CREATE UNIQUE INDEX IF NOT EXISTS idx_ss_rule_files_token ON ss_rule_files(token);`（`InitSSRuleFiles` 内，紧跟建表之后）。

#### P1-2 token 落进 `http_request_logs` 表

`backend/middleware/request_logger.go:55` 记完整 `c.Request.URL.Path`（截断 512）⇒ token 明文入库，`/api/monitor` 可读。

**修**：把 `"/ss/"` 加进 `shouldSkipMonitoringRequest` 的前缀列表（`request_logger.go:77-83`）。
**代价**：公开下载不再进监控日志（失去该端点的访问审计）。**取舍由 PM 拍板**（见 §8 Q2）—— 本方案默认**加**（token 是凭据，优先保护凭据）。

#### P1-3 `POST /api/ssrules` 的请求体上限是 200MB

`backend/app_http.go:37` 全局 `ContentSizeLimiter(200MB)`，而 `ssRuleMaxSourceSize`（512KB）的检查发生在 `ShouldBindJSON` **之后** ⇒ 一个 200MB 的 body 会先进内存。虽然 `POST` 需密码（未授权者到不了 `ShouldBindJSON`），但仍应收紧。

**修**：`api.Group("/ssrules", middleware.ContentSizeLimiter(1<<20))`（1MB，远大于 512KB 上限）。

#### P1-4 公开分发端无限流

见 §3.4.4。**修**：在 `app_http.go` 里给 `/ss/` 挂**独立预算**的 limiter（同样 `NewRateLimiter(120, time.Minute, nil)`，独立 store）。

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
| P2-4 | 未设 `X-Content-Type-Options: nosniff` | 低风险（`text/plain` 不会被当 HTML），可顺手加 |
| P2-5 | `RegenerateToken` 的响应里 `content` 是存量正文、`skipped` 恒 `[]`（与 `Get` 的重算行为不一致） | 语义可接受（换发不改内容）；如需一致可在 `RegenerateToken` 里也重跑 `buildACLRules` |

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
  handlers/ssrule_test.go handlers/ssrule_rules_test.go models/ssrule.go
# 期望：0 行输出
```

> **说明**：`golang:1.25.5`（bookworm）自带 gcc，CGO 可用；输出写到容器内 `/tmp`，**不污染宿主**。`with_utls` tag 与 `Dockerfile:39` 一致。
> **③ 的期望值说明**：上游 commit 的这批文件在 commit 状态下 `gofmt -s -l` **全部干净**（已实测，§9.3）；但 `origin/main` 上 `backend/routes.go` **本来就不干净**（上游顺手修了它）。仓库另外 34 个 gofmt 红项**不要碰**（S9 越界）。

### 6.2 Stage 5 · 后端单测（只跑本模块，避开既有红项）

```bash
cd backend
docker run --rm \
  -v "$PWD":/app -w /app \
  -e GOPROXY=https://goproxy.cn,direct \
  golang:1.25.5 \
  sh -c "go test ./handlers/ -run 'TestSSRule|TestBuildACLRules' -count=1 -v"
```

> **不要跑 `go test ./handlers/` 全包**：`main` 上就有 3 FAIL + 1 挂起（`TestAsyncAnthropicMessages_UpstreamError` / `TestDoJSONRequestNoRetryOnCanceled` / `TestParseClashYAML_MarksUnsupportedTypes` / `TestSendAlertSMTPFlow` 挂起），与本单无关（`project_handlers_preexisting_test_failures`）。
>
> **依赖注意**：`ssrule_test.go` 复用 `backend/handlers/skills_db_test.go:24` 的 `newMemDB(t)`（已含 `SetMaxOpenConns(1)`，满足 R5）。**不要删 `skills_db_test.go`** —— 删了本模块测试编译不过。
> `mustJSON` 是 `ssrule_test.go` **新引入**的 helper（`grep "func mustJSON" backend/handlers/*.go` → 0 命中），无重名冲突。

### 6.3 后端验收断言（**可判定、不恒真**）

> 每条断言都要能「改了就会红」。括号里是**反事实自检**（mutation）—— Stage 5 必须做这一步，否则断言可能是恒真闸门。

| # | 断言 | 反事实自检 |
|---|---|---|
| **A1** | 管理端密码门：无密码头 → 401；错密码 → 401；对密码 → 200 且 body 有 `paths`（4 条）与 `source` | 把 `checkAdmin` 的 `return false` 改成 `return true` → A1 必须红 |
| **A2** | `GET /ss/<token>/rules.acl` → 200；body 含 `(^\|\.)google\.com$`；头 `Content-Type: text/plain; charset=utf-8`、`Cache-Control: no-store` | 去掉 `Cache-Control` → A2 必须红 |
| **A3** | **路径穿越载荷表（§3.4.1）逐条 → 404，且 body 逐字节等于 `规则文件不存在`**（既验状态码也验不泄露路径） | 把白名单改成 `strings.HasSuffix(name, ".acl")` → `rules.acl.txt` 那条必须红 |
| **A4** | 换发 token 后：旧 token → 404；新 token → 200；旧 token 404 的 body 与"乱 token"404 的 body **完全一致** | 让 `RegenerateSSRuleToken` 不 UPDATE token → A4 必须红 |
| **A5** | **未配密码（`adminPassword == ""`）时**：M1-M4 全 401；**且** `GET /ss/<任意 token>/rules.acl` → 404（P0-3） | 把 P0-3 的 enabled 判断删掉 → A5 必须红 |
| **A6** | **限流**：同一 IP 连打 `POST /api/ssrules/verify` 第 11 次 → 429；**同一 IP 同时打 `POST /api/paste` 不被连带 429**（独立预算，P0-2）；`GET /api/ssrules` 也被限流（P0-1） | 去掉 GET 上的限流 → A6 的 GET 子项必须红；把 ssrules 的 limiter 换回 `createRateLimiter` → 隔离性子项必须红 |
| **A7** | **反向断言**：`backend/config/config.go` 里 `minimax-coding-plan-mcp` 的 `Args` **仍含 `"-y"`**；`config.example.yaml` 的 `minimax_mcp.args` **仍含 `"-y"`**（防 P0-4 取件回退） | 用 `git checkout e7f37dac… -- backend/config/config.go` 覆盖一次 → A7 必须红 |
| **A8** | **token 权限边界**：把 token 当管理凭据（`GET /api/ssrules?token=<token>`、`POST /api/ssrules?admin_password=<token>`）→ 必须 401 | 让 `checkAdmin` 接受 token → A8 必须红 |
| **A9** | `updated_at` 可被浏览器解析：后端返回 RFC3339 字符串，`time.Parse(time.RFC3339, v)` 成功且非零值 | 把 `updated_at` 写成零值 → A9 必须红 |
| **A10** | 表结构：`ss_rule_files` 有 `idx_ss_rule_files_token` 唯一索引（P1-1） | 去掉 `CREATE UNIQUE INDEX` → A10 必须红 |
| **A11** | 规则生成：域名 → `(^\|\.)example\.com$`（**不是** `||example.com`）；`[bypass_all]` + `[proxy_list]` 结构完整；非法行逐条进 `skipped` 且**不出现在产物里** | 把 `ssRuleHostToACLLine` 改成输出 `||` → A11 必须红 |
| **A12** | 监控日志**不含** `/ss/` 路径（P1-2，若 PM 采纳该取舍） | 把 `/ss/` 从 skip 列表删掉 → A12 必须红 |

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

> **F1/F2 的实现方式**：照抄既有先例 `frontend/src/views/ai/resultShareStaticRules.test.js`（读 `.vue` 源码文本 → 切 `<template>` 段 → 正则扫描）。**不要**用 `@vue/test-utils` mount 做静态断言 —— 静态规则用文本扫描更稳、更不会因为环境差异假红。

### 6.5 反事实自检总则（Stage 5 必做）

对每条断言，必须产出**三组输出**：

1. **基线**（未改本单代码时）：该断言应当**红**或**不适用**
2. **被 FAIL 版**（人为植入缺陷）：必须**红**
3. **修复分支**：必须**绿**

只有 2、3 两组、且 2 确实红、3 确实绿，才算断言"咬合"。**恒真断言（怎么改都绿）= FAIL**。

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
| R-6 | 共享限流桶导致别的模块被连带 429 | 中 | 其它工具 429 | P0-2 独立 store |
| R-7 | 公开端点被机器人扫描打 DB | 中 | DB 压力 | P1-4 独立 limiter |
| R-8 | `updated_at` 序列化形态退化 ⇒ 页面显示 1/1/1 | 低 | 体验 | 断言 A9 |

### 7.2 回滚（两条路径）

#### 路径 ① · 配置级关闭（首选，秒级生效，不动代码）

```yaml
# config.yaml
ssrules:
  admin_password: ""     # 清空
```

**预期行为**（依赖 **P0-3 已修**）：

| 端点 | 关闭后 |
|---|---|
| M1-M4 | 401 `密码错误` |
| P1 `/ss/*` | **404 `规则文件不存在`** |

> ⚠️ **P0-3 未修时，路径 ① 只能关掉管理端，公开链接继续吐旧内容** —— 所以 **P0-3 是这条回滚路径成立的前提**。

#### 路径 ② · 代码级回滚（删路由）

1. 删 `backend/routes/ssrule.go` 整个文件 + `routes/index.go` 的 `RegisterSSRuleRoutes` 调用 + `routes.RouteHandlers` 的 `SSRuleHandler` 字段
2. 删 `backend/routes.go` 的 `ssRuleHandler` 字段与 `SSRuleHandler: h.ssRuleHandler,`
3. 删 `backend/app_handlers.go` 的构造与赋值
4. 删 `backend/app_http.go` 的 `router.GET("/ss/:token/:filename", ...)`
5. 删 `backend/handlers/ssrule*.go`、`backend/models/ssrule.go`

> ⚠️ **陷阱**：删完路由后，`/ss/<token>/rules.acl` 会落到 `NoRoute` → **SPA fallback 返回 `index.html` 且状态码 200**（`backend/app_http.go:78-103`）。客户端会"成功"拿到一坨 HTML 当规则文件 —— **静默失败，比 404 更难排查**。
>
> **因此路径 ② 必须同时堵住 NoRoute 兜底**，二选一：
> - (a) 在 `registerStaticRoutes` 的 `NoRoute` 里加一条：`if strings.HasPrefix(path, "/ss/") { c.String(404, "规则文件不存在"); return }`；或
> - (b) 保留一条**只返回 404 的空路由**占位：`router.GET("/ss/:token/:filename", func(c *gin.Context){ c.String(404, "规则文件不存在") })`
>
> **推荐 (b)** —— 改动最小、不碰 NoRoute 逻辑。

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
| **Q2** | P1-4：分发端限流阈值 | ① 120/min/IP（推荐）<br>② 30/min/IP（更严，但多人共享出口 IP 时可能误伤）<br>③ 不限流 | **① 120/min/IP** | @devtools-pm |
| **Q3** | 独立限流器用内存（不落 Redis）是否可接受？ | ① 接受（单容器单副本，重启清零）<br>② 给 `middleware.RateLimiter` 加 `KeyPrefix`，落 Redis（改共享中间件，blast radius 大） | **① 接受**，② 另开单 | @devtools-pm |
| **Q4** | token 要不要加有效期？ | ① 不加（客户端规则源需长期可用，换发是止损手段）<br>② 加 90 天 TTL（客户端要定期重填链接） | **① 不加** | @devtools-pm |
| **Q5** | `ssrules.admin_password` 是否复用全局 `super_admin_password`？ | ① 独立配置（照 proxy / nps 惯例，本方案采用）<br>② 复用全局（少一个配置项，但两个模块同密码 ⇒ 一处泄露两处失守） | **① 独立** | @devtools-pm |
| **Q6** | 是否要支持"多套规则"（按客户端区分）？ | ① 不做（上游设计已定"单一事实源"）<br>② 加 `id` 维度做成多行 | **① 不做** | @devtools-pm |

> **D6 说明**：以上 6 项都已给出「两种以上可选解释 + 我的倾向」，**Stage 2/3/4 不必等裁决即可开工**（Q1/Q2 涉及一行代码，Stage 4 按倾向实现，PM 若改判再回改一行）。

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
 backend/handlers/ssrule_rules.go       | 164 +++++++++++++++
 backend/handlers/ssrule_rules_test.go   | 205 +++++++++++++++++++++++
 backend/handlers/ssrule_test.go         | 262 ++++++++++++++++++++++++
 backend/models/ssrule.go                | 123 +++++++++++++++
 backend/routes.go                       |  78 +++++-----
 backend/routes/index.go                 |   2 +
 backend/routes/ssrule.go                |  26 ++++
 frontend/src/router/index.js            |  11 ++
 frontend/src/views/other/SsRuleTool.vue | 244 +++++++++++++++++++++++++
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
backend/routes.go          ← origin/main 上本来就不干净（上游顺手修了）

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
--- a/backend/config.example.yaml
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

### 9.5 限流器共用（P0-2 的直接证据）

```
$ grep -n "createRateLimiter :=" backend/app_http.go
39:  createRateLimiter := middleware.NewRateLimiter(10, time.Minute, rt.transientStore)

$ grep -n "ClientIP" backend/middleware/ratelimit.go
49:      ip := c.ClientIP()

$ grep -n -A2 "func (s \*RedisStore) AllowRateLimit" backend/state/transient.go
241:  fullKey := s.key("ratelimit:" + key)      ← key 里没有端点名 / limiter 身份
```

### 9.6 请求日志记路径（P1-2 的直接证据）

```
$ grep -n "Path:" backend/middleware/request_logger.go
55:  Path:          truncateMonitoringValue(c.Request.URL.Path, 512),

$ grep -n -A8 "func shouldSkipMonitoringRequest" backend/middleware/request_logger.go
73: func shouldSkipMonitoringRequest(method, path, upgrade string) bool {
74:   if method == "OPTIONS" || strings.EqualFold(upgrade, "websocket") { return true }
76:   for _, prefix := range []string{ "/api/monitor", "/api/health", "/assets/", "/neon/", "/api/bg/cached/" } {
       ↑ 不含 "/ss/"
```

### 9.7 `:memory:` 池约定（R5）

```
$ grep -rn "func newMemDB" backend/handlers/*.go
backend/handlers/skills_db_test.go:24:func newMemDB(t *testing.T) *models.DB {
   ↑ 内部已含 db.SetMaxOpenConns(1)

$ grep -rn "func mustJSON" backend/handlers/*.go
0 hits                       ← ssrule_test.go 新引入，无重名冲突
```

### 9.8 前端别名与路由

```
$ grep -n "other:" frontend/src/router/index.js
11:  other: { name: '系统管理', icon: 'Tools', description: '代理、控制台与辅助工具', accent: '#475569', order: 70 }

$ grep -n "icon: 'Share'" frontend/src/router/index.js
112:  icon: 'Share',   225:  icon: 'Share',   236:  icon: 'Share'      ← 图标名合法

$ grep -n "const allTools" frontend/src/composables/useToolRegistry.js
13:  const allTools = computed(() => toolRoutes.filter(isNavigableTool))   ← 工具清单自动派生，无需另注册
```

---

## 10. 硬约束（下游无条件遵守）

| ID | 约束 | 本模块的具体落点 |
|---|---|---|
| **R2** | 工具范围 = 只能服务端做的 | 规则生成在服务端（要落库 + 公开分发）；前端**不做**任何编码/hash/格式转换 |
| **R3** | Vue 模板里永远不写 `.value` | `SsRuleTool.vue` 模板只用裸名（上游已合规，实测 0 命中） |
| **R4** | Vue 模板无裸下标 `obj.x[0]` | 同上（实测 0 命中）；用 `?.` + `\|\| ''` |
| **R5** | SQLite `:memory:` 测试必须 `SetMaxOpenConns(1)` | 复用 `newMemDB(t)`（已含），**不得**自己 `models.NewDB(":memory:")` |
| **R6** | 后端分享资产必须用上游 http(s) URL | 本模块的分发 URL 是 `window.location.origin + /ss/<token>/<file>`（http(s)），**不涉及 blob** |
| **R8** | 长耗时 AI 调用异步化 | 本模块**无** AI 调用，不涉及 |
| **R9** | MCP 协议版本白名单 | 不涉及 |
| **R11** | 后端跨平台编译走 Docker | §6.1 的编译命令在 `golang:1.25.5` 容器内跑，**不从 macOS 出二进制** |
| **别名** | `@` 指向 `src/neon` 不是 `src` | 本组件用相对路径 `../../api` / `../../composables/useAdminAuth`，**不写 `@/...`** |
| **SMB** | `/Volumes/M20` 是 SMB 网络挂载 | 前端装依赖/构建走 `/tmp` scratch（§6.4）；**不在该盘建 SQLite 库 / 跑 npm install / 落大量小文件** |
| **S9** | 不越界改动 | 不修另外 34 个 gofmt 红项；不改 `backend/middleware/compress.go`；`AGENTS.md` 由后端单更新，architect 不改 |

---

## 11. 交付清单（Stage 4 / Stage 5 收工前逐条打勾）

**后端单（Stage 4）**

- [ ] 12 个后端文件就位（§5.1 表格）
- [ ] `backend/config/config.go` + `config.example.yaml` **手工补丁**，FINN-56 的 `-y` **仍在**（断言 A7）
- [ ] **P0-1** `GET /api/ssrules` 已挂限流
- [ ] **P0-2** ssrules 用独立 store 的 limiter，不回落全局
- [ ] **P0-3** `Download` 在未配密码时返回 404（与 token 不存在响应完全一致）
- [ ] **P0-5** 路径穿越载荷回归用例齐（§3.4.1 表）
- [ ] **P1-1** token 唯一索引
- [ ] **P1-2** `/ss/` 进监控 skip 列表（若 Q1 按倾向裁定）
- [ ] **P1-3** `/ssrules` 组 1MB 请求体上限
- [ ] **P1-4** 分发端独立限流
- [ ] `AGENTS.md` 模块表加 `#43 SSRules`
- [ ] §6.1 三条命令全绿（compile / vet / gofmt 本单文件）
- [ ] 提交前 `git diff --cached --stat` 只含目标文件

**前端单（Stage 3）**

- [ ] `SsRuleTool.vue` + router 条目（`meta.title` / `meta.icon` / `meta.category` 齐全）
- [ ] **P1-5** `credentialField` → `headerName`
- [ ] R3 / R4 合规（模板无 `.value`、无裸下标）
- [ ] 无 `console.log` / `TODO` / `FIXME`
- [ ] 用相对路径，不用 `@/...`

**Stage 5**

- [ ] 后端单测只跑 `-run 'TestSSRule|TestBuildACLRules'`（避开既有 3 FAIL + 1 挂起）
- [ ] §6.3 的 A1–A12 全部绿，且**每条都做过反事实自检**（植入缺陷后必须红）
- [ ] §6.4 前端 F1–F6 全部绿
- [ ] `pnpm build` 通过
- [ ] 报告贴出**三组输出**（基线 / 被 FAIL 版 / 修复分支）

---

## 12. 变更记录

| 版本 | 日期 | 内容 |
|---|---|---|
| v1 | 2026-10-22 | 首版。基于对上游 commit `e7f37dac…` 与 `origin/main`(`81199bd`) 的实读勘察产出。识别 5 个 P0 + 6 个 P1 + 5 个 P2。 |
