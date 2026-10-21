# MiniMax MCP 默认 args 与 `entrypoint.sh` 对齐 + 启动参数兜底加固 — 技术方案（FINN-57 / FINN-56 Stage 1）

- 原 issue：FINN-56（父）→ 本子 issue FINN-57（Stage 1 技术方案设计）
- 模块：后端配置层（`backend/config/`）+ MCP 进程启动点（`backend/handlers/image_understanding.go`）
- 基线：`origin/main` @ `9992ac6`（本文所有 `file:line` 均以该基线为准，行号已用 `git show origin/main:<path> | grep -n` 逐条核过）
- 状态：**设计稿，不含任何代码改动**；实现由 devtools-backend 执行，验收由 devtools-qa 执行
- 勘察方式：全仓 grep + 逐段 Read + `git show origin/main` 逐行核对 + 供应商公开文档检索；**未起 dev server、未跑 go build / go test、未部署、未碰部署机 `192.168.31.201`（R1）**
- 本文不含任何未标注来源的推断；凡属「推算」一律显式标注

---

## 0. 一页纸结论（给下游先看这段）

| 项 | 结论 |
|---|---|
| 必改点是否只有 3 处？ | **是。** 全仓 `minimax-coding-plan-mcp` 硬编码 args 共 **4 处**（`config.go` / `config.example.yaml` / `docs/image-understanding-api.md` / `entrypoint.sh`），其中**前三处缺 `-y`**，`entrypoint.sh` 已正确。另有第 5 处（兜底常量）已带 `-y`。**没有第 6 处遗漏点。** |
| 兜底加固做不做？ | **做（方案乙）**，但必须带「**只对 minimax 包生效**」的门控，否则会误伤用户自定义 MCP Server 调用。PM 倾向的方案乙方向正确，我补一条必要门控（§3.3）。 |
| `-y` 的位置 | **必须保持在包名之后**（`uvx minimax-coding-plan-mcp -y`），这是 MiniMax 官方文档与 Claude Desktop / Claude Code / OpenCode 官方客户端配置的统一写法。**禁止**改写成 `uvx -y minimax-coding-plan-mcp`（那是 uv 自己的 `--yes`，语义不同）。§3.1 有依据。 |
| 对 PM 前提的一处事实修正 | PM 说「容器部署不受影响」→ **不完全成立**，见 §2.3。容器后端生效的配置来自部署机 `./backend/config.yaml`（gitignored）或 `backend/config.example.yaml`；若部署机没有自定义 `config.yaml`，**容器后端跑的就是缺 `-y` 的 example**。本方案修完恰好覆盖这种情况。 |
| 数据面影响 | **零**：无 DB 表、无 Redis key、无 API 契约变更、无前端改动。 |
| 未决 | 2 条非阻塞（见 §9），均不阻塞 Stage 2。 |

---

## 1. 目标与非目标

### 1.1 目标

1. 把仓库内 **3 处** MiniMax MCP 默认 args（`backend/config/config.go` 的 `DefaultConfig`、`backend/config.example.yaml`、`docs/image-understanding-api.md`）与 `entrypoint.sh:55` 的容器配置**逐字对齐**，补上 `-y`。
2. 把 `startMCPProcess` 里那段**当前近乎死代码**的兜底升级为**确定性的启动参数归一化**，让**存量 `config.yaml` 用户**（args 非空但缺 `-y`）也能被兜住。
3. 给出可判定、可自动化的验收清单（DoD 1–6）与单点回滚路径。

### 1.2 非目标（明确不做）

- ❌ 不动 `entrypoint.sh`（其 `:55` 已正确）。
- ❌ 不动 `Dockerfile`（其 `:88` 是 build 期 uvx cache 预热，与本 issue 的运行期 args 无关；另见 §9 未决 2）。
- ❌ 不改 MiniMax MCP 的**接口契约**（`/api/image-understanding/*`、`/api/minimax/result-shares` 等一律不动）。
- ❌ 不做 TTS / 图像理解「保存分享」前端功能（属 FINN-55 的 (A)(B)，待 jaxiu 拍板）。
- ❌ 不移植 `feature/minimax-share-all` 分支上的任何东西（模型白名单收窄等，见 FINN-55 §1 非缺口 D）。
- ❌ 不部署、不起服务、不碰部署机（R1）。

---

## 2. 现状（已逐条核实）

### 2.1 全仓硬编码 args 清单（`grep -rn "minimax-coding-plan-mcp"` 全仓，未遗漏）

| # | 位置（origin/main @ `9992ac6`） | 当前内容 | 状态 |
|---|---|---|---|
| 1 | `backend/config/config.go:530` | `Args:           []string{"minimax-coding-plan-mcp"},` | ❌ 缺 `-y` → **必改** |
| 2 | `backend/config.example.yaml:237` | `    - "minimax-coding-plan-mcp"` | ❌ 缺 `-y` → **必改** |
| 3 | `docs/image-understanding-api.md:182` | `    - "minimax-coding-plan-mcp"` | ❌ 缺 `-y` → **必改** |
| 4 | `entrypoint.sh:55` | `      "args": ["minimax-coding-plan-mcp", "-y"],` | ✅ 已正确（**但服务的是另一个消费方**，见 §2.2） |
| 5 | `backend/handlers/image_understanding.go:699` | `args = []string{"minimax-coding-plan-mcp", "-y"}` | ✅ 带 `-y`，但**触发条件几乎不可达**（见 §2.4） |
| 6 | `Dockerfile:88` | `RUN uvx minimax-coding-plan-mcp -y --help >/dev/null 2>&1 \|\| true` | ✅ 带 `-y`（build 期预热，非运行期 args） |
| 7 | `.env.example:28` | `# MiniMax MCP API Key（用于 minimax-coding-plan-mcp 工具）` | 纯注释，无 args |

补充核对：`git grep -n "uvx" origin/main` 全量扫过，**没有任何第 5 处运行期 args 构造点**——`backend/handlers/image_understanding.go:694` 是 `cfg.Args` 的**唯一**消费方：

```
origin/main:backend/handlers/image_understanding.go:696:	command = "uvx"
origin/main:backend/handlers/image_understanding.go:989:	if command == "uvx" {
```
（`grep -rn "\.Args\b" backend/ --include="*.go"` 结果中，除 `image_understanding.go:694` 外全部是 HTTP 请求体的 `req.Args`，与 MCP 启动参数无关。）

### 2.2 `entrypoint.sh:55` 服务的是**另一个消费方**（重要）

`entrypoint.sh:47-72` 写的是 `${CLAUDE_HOME}/settings.json` 里的 `mcpServers.MiniMax` —— 这是给**容器内的 Claude Code CLI**（autodev 任务）用的 MCP 客户端配置，**不是** Go 后端读的配置。

Go 后端读的是 `config.yaml`（`backend/config/config.go:627` 的 `Load(path)`），容器里由 `entrypoint.sh:135-143` 准备：

```sh
CONFIG_SRC_DIR=/app/config-src          # entrypoint.sh:7
if [ -f "${CONFIG_SRC_DIR}/config.yaml" ]; then
    cp "${CONFIG_SRC_DIR}/config.yaml" "${CONFIG_TARGET_PATH}"      # entrypoint.sh:137
elif [ -f "${CONFIG_SRC_DIR}/config.example.yaml" ]; then
    cp "${CONFIG_SRC_DIR}/config.example.yaml" "${CONFIG_TARGET_PATH}"  # entrypoint.sh:140
fi
```

而 `/app/config-src` 是宿主机 `./backend` 的只读挂载（`docker-compose.yml:22`：`- ./backend:/app/config-src:ro`）。

> **结论**：`entrypoint.sh:55` 与后端 args 是**两条独立的配置链路**。把后端默认 args 对齐到 `entrypoint.sh:55` 的**字面形式**是对的（同一个官方调用形式），但**不能**因此推断「容器后端不受影响」——见下一节。

### 2.3 修正 PM 的前提：容器后端**可能**同样受影响

容器后端生效的 `minimax_mcp.args` 取值只有两种可能：

| 部署机上存在 | 容器后端实际用的 args | 是否带 `-y` |
|---|---|---|
| `./backend/config.yaml`（`.gitignore:62` 已忽略，属部署机本地文件） | 该文件里的 `minimax_mcp.args` | **未知**（内容在部署机上，本机不可见） |
| 不存在 `./backend/config.yaml` | `backend/config.example.yaml:237`（仓库文件） | ❌ **缺 `-y`** |

**即**：PM 说的「容器部署不受影响」只在「部署机恰好有一份已写对 `-y` 的私有 `backend/config.yaml`」时成立；若部署机没有该文件，容器后端跑的就是**缺 `-y` 的 example**，与本地 `go run` 是同一个缺陷。

> 这不是反驳 PM 的结论方向，而是**收紧**结论范围：修 `config.example.yaml` 这件事**在容器链路上同样是必要的**，而不是只对本地开发有意义。
> 部署机 `backend/config.yaml` 是否存在、args 写了什么 —— **属 R1 禁探范围**，本方案不做任何假设，写进 §8「未覆盖」。

### 2.4 现有兜底为何近乎死代码

`origin/main backend/handlers/image_understanding.go:692-701`：

```go
func startMCPProcess(cfg config.MiniMaxMCPConfig) (*mcpProcess, error) {
	command := cfg.Command
	args := cfg.Args
	if strings.TrimSpace(command) == "" {
		command = "uvx"
	}
	if len(args) == 0 {
		args = []string{"minimax-coding-plan-mcp", "-y"}
	}
```

`cfg.Args` 的来源（`backend/config/config.go:627-641`）：

```go
cfg := DefaultConfig()          // config.go:629 —— 默认 args = ["minimax-coding-plan-mcp"]（缺 -y）
data, err := os.ReadFile(path)
...
if err := yaml.Unmarshal(data, cfg); err != nil {   // config.go:639 —— 只覆盖 yaml 里出现的键
```

推论（**由上述代码直接推出，非推测**）：

| 用户配置形态 | `cfg.Args` 实际值 | 现有 `len(args)==0` 兜底 | 结果 |
|---|---|---|---|
| 没有 `config.yaml` | `["minimax-coding-plan-mcp"]`（默认值） | **不触发** | ❌ 缺 `-y` |
| `config.yaml` 无 `args:` 键 | 同上（yaml 不覆盖） | **不触发** | ❌ 缺 `-y` |
| `config.yaml` 有 `args: ["minimax-coding-plan-mcp"]`（旧示例抄的） | 同上 | **不触发** | ❌ 缺 `-y` |
| `config.yaml` 有 `args: []`（显式空列表） | `[]` | 触发 | ✅ 带 `-y` |
| `config.yaml` 有 `args: ["minimax-coding-plan-mcp","-y"]` | 已带 | 不触发 | ✅ |

→ 兜底只在「用户显式写了空列表」这一种边缘写法下生效，**默认路径全部绕过**。PM 的「死代码」判断成立。

### 2.5 `-y` 的语义与位置（**别改错位置**）

- MiniMax 官方 MCP 文档与主流客户端官方配置统一写作 `uvx minimax-coding-plan-mcp -y`（Claude Desktop 的 `mcpServers.MiniMax.args`、Claude Code CLI 的 `claude mcp add … -- uvx minimax-coding-plan-mcp -y`、OpenCode 的 `command: ["uvx","minimax-coding-plan-mcp","-y"]`）。
- `-y` 位于**包名之后**，是 MiniMax MCP 包自身的 CLI 标志（跳过确认）。
- uv 自身的 `--yes` 必须放在**包名之前**（`uvx --yes <pkg>`）；放在包名之后会被 uv 转发给被调起的程序。两者是不同层次的开关。
- **因此**：本次改动一律采用 `["minimax-coding-plan-mcp", "-y"]` 的形式，**不得**改写为 `["-y", "minimax-coding-plan-mcp"]`。

依据来源见 §10（供应商文档 + 客户端官方配置示例检索）。

### 2.6 全局配置共享（决定实现写法的一条硬约束）

`backend/config/config.go`：

```
451: var globalConfig *Config
838: globalConfig = cfg
843: func Get() *Config {
844: 	if globalConfig == nil {
845: 		globalConfig = DefaultConfig()
```

`startMCPProcess` 拿到的是 `config.MiniMaxMCPConfig` **值拷贝**，但其中的 `Args` **切片头**与全局配置共享**同一块底层数组**。因此在 `startMCPProcess` 里直接 `append(cfg.Args, "-y")`：

- 若 `cap(cfg.Args) > len(cfg.Args)`（yaml 反序列化出来的切片**可能**有富余容量），`append` 会**原地写入全局切片的共享内存**；
- `startMCPProcess` 由请求 goroutine 调用，而其它 goroutine 可能同时读 `config.Get().MiniMaxMCP.Args` → **data race**。

→ 实现必须**新建切片**，禁止原地 `append`。

---

## 3. 设计

### 3.1 变更清单（改动 1–3：配置对齐）

| # | 文件:行（origin/main `9992ac6`） | 现状 | 改成 |
|---|---|---|---|
| 1 | `backend/config/config.go:530` | `Args:           []string{"minimax-coding-plan-mcp"},` | `Args:           []string{"minimax-coding-plan-mcp", "-y"},` |
| 2 | `backend/config.example.yaml:237` | `    - "minimax-coding-plan-mcp"` | `    - "minimax-coding-plan-mcp"`<br>`    - "-y"` |
| 3 | `docs/image-understanding-api.md:182` | `    - "minimax-coding-plan-mcp"` | `    - "minimax-coding-plan-mcp"`<br>`    - "-y"` |

改动 2 / 3 的 YAML 缩进沿用文件既有层级（`args:` 下 4 空格 + `- `）。

**不动的两处（已正确，仅作为对齐基准）**：`entrypoint.sh:55`、`Dockerfile:88`。

### 3.2 变更清单（改动 4：兜底加固 = **方案乙**）

**定夺：采纳方案乙**（3 处配置对齐 **+** 启动期参数归一化），理由：

1. 方案甲只能修好**新装 / 照抄新示例**的用户；**存量 `config.yaml` 里已写死 `args: ["minimax-coding-plan-mcp"]` 的用户依旧缺 `-y`**，而这类用户恰恰是「已经踩过一次坑、照着旧示例抄了一份」的人 —— 只改默认值对他们零效果。
2. 归一化是**纯函数、无状态、无迁移、无外部依赖**，改动面 = 1 个函数 + 2 个 helper + 1 个常量组，回滚 = revert 单个 commit。
3. 归一化后，配置层与运行期形成**双保险**：新装用户走默认值即正确；存量用户由运行期兜住。
4. 副作用可控：只要加「**只对 minimax 包生效**」的门控（§3.3 规则 4），**非 minimax 的调用一律字节级不变**。

**具体写法**（`backend/handlers/image_understanding.go`，紧邻 `startMCPProcess` 放置）：

```go
const (
	// minimaxMCPPackage MiniMax MCP Server 的 uvx 包名（与供应商文档一致）
	minimaxMCPPackage = "minimax-coding-plan-mcp"
	// minimaxMCPYesFlag MiniMax MCP 包自身的「跳过确认」标志，须紧跟包名
	minimaxMCPYesFlag = "-y"
)

// normalizeMCPArgs 归一化 MCP Server 启动参数（纯函数，不修改入参底层数组）：
//  1. args 为空              → 官方默认调用 ["minimax-coding-plan-mcp", "-y"]
//  2. args 已含精确元素 "-y" → 原样返回（不重复注入）
//  3. args 含 minimax 包名但缺 "-y" → 末尾追加 "-y"（兜住存量 config.yaml）
//  4. args 与 minimax 无关    → 原样返回（用户自定义 MCP Server，绝不注入）
func normalizeMCPArgs(args []string) []string {
	if len(args) == 0 {
		return []string{minimaxMCPPackage, minimaxMCPYesFlag}
	}
	if !containsMinimaxMCPPackage(args) {
		return args
	}
	for _, a := range args {
		if a == minimaxMCPYesFlag {
			return args
		}
	}
	out := make([]string, 0, len(args)+1)
	out = append(out, args...)
	out = append(out, minimaxMCPYesFlag)
	return out
}

// containsMinimaxMCPPackage 判断 args 是否在调用 MiniMax MCP 包。
// 兼容 uvx 的版本限定写法："minimax-coding-plan-mcp==0.0.4" / "…@0.0.4"。
func containsMinimaxMCPPackage(args []string) bool {
	for _, a := range args {
		if a == minimaxMCPPackage ||
			strings.HasPrefix(a, minimaxMCPPackage+"==") ||
			strings.HasPrefix(a, minimaxMCPPackage+"@") {
			return true
		}
	}
	return false
}
```

`startMCPProcess` 内的替换（`backend/handlers/image_understanding.go:694-700`）：

```go
	command := cfg.Command
	args := cfg.Args
	if strings.TrimSpace(command) == "" {
		command = "uvx"
	}
	args = normalizeMCPArgs(args)     // ← 取代原 `if len(args) == 0 { ... }`
```

**实现约束（下游无条件遵守）**：

- `normalizeMCPArgs` 返回**新切片**或**原切片**均可，但**禁止**在 `cfg.Args` 的底层数组上原地 `append`（§2.6 的 data race）。上面的写法用 `make` + `copy` 语义满足。
- `strings` 包在 `image_understanding.go` 已 import（`strings.TrimSpace` 在用），无需新增 import。
- 常量名沿用同文件既有风格（`normalizeTransport`，同文件 `:1000`）；**禁止**把 `"-y"` / `"minimax-coding-plan-mcp"` 当字面量散落在多处（S6）。
- **禁止**把 `-y` 提到包名之前（§2.5）。
- 归一化只在 `startMCPProcess` 这一个收口点做，**不要**在别处再写一份（§S5）。

### 3.3 边界规则（PM 点名要定的三个问题 + 追加两条）

| # | 问题 | 规则 | 理由 |
|---|---|---|---|
| B1 | `-y` 出现在**任意位置**都算「已含」？ | **是**。扫描整个 args 切片，**精确元素匹配** `-y` 即判定已含 → 原样返回，不重复注入 | 官方形式里 `-y` 就在包名之后；用户也可能写 uv 形式 `["-y","minimax-coding-plan-mcp"]`（此时 `-y` 被 uv 消费，语义已达成，不该再补第二个） |
| B2 | `--yes` 算不算「已含」？ | **不算**。只认精确 `-y` | `--yes` 是 **uv 的** flag；放在包名之后会被 uv 转发给 MCP 进程，而 MCP 进程不认 `--yes`。把它当「已含」会漏掉进程真正需要的 `-y`。若用户写成 `["--yes","minimax-coding-plan-mcp"]`，本规则会补 `-y` → `uvx --yes minimax-coding-plan-mcp -y` → uv 吃 `--yes`、进程收 `-y`，**结果正确** |
| B3 | **空串元素**怎么办？ | **不 TrimSpace、不丢弃空串、不特殊处理**。空串既不等于 `-y`、也不匹配包名 → 不触发「已含」判定。若 `args = ["minimax-coding-plan-mcp", ""]` → 命中包名 → 结果 `["minimax-coding-plan-mcp","","-y"]`（空参数是用户原配置的问题，本方案只加不减）；若 `args = [""]` → 不匹配包名 → 原样透传，由 uv 自己报错 | 最小干预原则：只加 `-y`，不动用户其它参数；误 trim 会把畸形值判成「已含」而**漏掉** `-y` |
| B4 | **非 minimax 的自定义 MCP 调用**会不会被误伤？ | **门控掉**：`args` 不含 minimax 包名时**绝不注入** | 反例：`command: "npx"` + `args: ["@modelcontextprotocol/server-filesystem","/data"]`，盲注入 `-y` 会被 npx 转发给该 server → 启动失败。门控后这类调用**字节级不变** |
| B5 | 重复注入 `-y` 有害吗？ | 无害但**不允许**：已含即返回 | argparse 的 store_true 标志重复传无害，但保持「幂等、确定性」更重要（DoD 5 要求可判定） |

**门控规则（B4）补充说明**：包名匹配用「元素等于 `minimax-coding-plan-mcp`」或「以 `minimax-coding-plan-mcp==` / `minimax-coding-plan-mcp@` 开头」，以兼容 uvx 的版本限定写法。追加位置固定在**末尾**——因为门控已保证 args 内含包名，末尾必然在包名之后，满足 §2.5 的位置约束。

### 3.4 数据模型 / 存储

**无变更**。本方案不新增 SQLite 表、不新增 Redis key、不改 `config.yaml` 的字段结构（`args` 字段早已存在，`backend/config/config.go:329`）。

### 3.5 API 契约

**无变更**。不新增 / 不修改任何 HTTP 端点、请求体字段、响应体字段、状态码、错误码。`/api/image-understanding/*` 的行为差异仅体现在「MCP 子进程能否成功起起来」，属**内部实现**。

### 3.6 后端改动清单（汇总）

| 文件 | 改动 | 行数 | 类型 |
|---|---|---|---|
| `backend/config/config.go` | `DefaultConfig().MiniMaxMCP.Args` 补 `-y` | 1 行（`:530`） | 配置 |
| `backend/config.example.yaml` | 示例 args 补 `-y` | +1 行（`:237` 后） | 配置 |
| `docs/image-understanding-api.md` | 文档示例 args 补 `-y` | +1 行（`:182` 后） | 文档 |
| `backend/handlers/image_understanding.go` | 新增 2 个常量 + 2 个 helper；`startMCPProcess` 改 1 行 | +~35 行 / 改 1 行 | 代码 |
| `backend/handlers/image_understanding_test.go`（**新建**） | `normalizeMCPArgs` 表驱动单测 | 新建 | 测试 |
| `backend/handlers/minimax_mcp_args_regression_test.go`（**新建**） | 默认配置回归护栏（照抄 `singbox_default_config_regression_test.go` 的既有范式） | 新建 | 测试 |

> 测试文件放 `backend/handlers/`（`package handlers`，与该目录既有 40 个 `_test.go` 一致），**不要**新建 `backend/config` 测试包 —— 该目录当前**只有 `config.go`，无任何 `_test.go`**（已核实）。

### 3.7 前端改动清单

**无**。本方案不涉及任何 `frontend/**` 改动、不涉及 `frontend/src/router/index.js`、不涉及任何 Vue 组件。

---

## 4. 兼容性与迁移

| 维度 | 结论 |
|---|---|
| 旧数据 | **无影响**。无 DB / Redis 变更 |
| 旧响应 | **无影响**。API 契约不变 |
| 旧配置（存量 `config.yaml`） | **不需要回填、不需要用户改文件**。改动 4 的运行期归一化会在启动 MCP 子进程时把 `args: ["minimax-coding-plan-mcp"]` 自动补成 `["minimax-coding-plan-mcp","-y"]` |
| 已写对的用户（args 已含 `-y`） | **完全不变**（归一化幂等，见 B5） |
| 显式 `args: []` 的用户 | 行为与现在一致（走默认调用 `["minimax-coding-plan-mcp","-y"]`） |
| 自定义 MCP Server（command ≠ minimax / args 不含 minimax 包名） | **字节级不变**（门控 B4） |
| 迁移脚本 | **不需要** |
| 回滚 | 见 §6 |

---

## 5. 风险与回滚

| # | 风险 | 等级 | 缓解 |
|---|---|---|---|
| K1 | 注入 `-y` 改变了 MCP 启动行为（跳过交互确认） | 低 | `-y` 是 MiniMax 官方调用形式的一部分，官方客户端配置即如此；无功能损失 |
| K2 | 盲注入误伤用户自定义 MCP Server（npx 等） | 中 | 门控 B4：非 minimax 调用一律不动 |
| K3 | 原地 `append` 写全局共享切片 → data race | 中 | §2.6 约束 + 实现约束「必须新建切片」；单测覆盖「入参切片不被修改」 |
| K4 | 追加到末尾导致 MCP 进程收到位置不对的参数 | 低 | 门控保证包名在 args 内，末尾必在包名之后；且 `-y` 是包自身认识的 flag |
| K5 | 部署机 `backend/config.yaml` 内容未知 → 容器实际行为仍可能与预期不同 | 低 | 两种情况在方案下都成立（已含 `-y` → 不动；不含 → 补上）；部署机核对列入 §8「未覆盖」，由 jaxiu 上线后确认 |
| K6 | 归一化函数成为新的「魔法行为」，将来 package 改名要同步 | 低 | 包名提为常量 + 单测锁定 |

**回滚方式（单点）**：

- 只回滚改动 1–3：`git revert <commit>`（3 个纯配置行，无副作用）。
- 只回滚改动 4：revert `backend/handlers/image_understanding.go` 的改动 → 恢复原 `if len(args) == 0 { ... }` 分支；**无状态、无迁移、无数据回填**，立即生效。
- 全量回滚：revert 整个 commit。
- 回滚后**不会**破坏容器：`entrypoint.sh` / `Dockerfile` 未被改动。

---

## 6. 测试要点（交给 devtools-qa，可执行可判定）

> **纪律**：不要跑全包单测。`backend/handlers` 包在 main 上**既有 3 FAIL + 1 挂起**（历史遗留，非本 issue 引入），只 `-run` 限定到本 issue 范围。
> **R11**：本地只做 `go build ./...` 编译校验，**不出**跨平台二进制产物（跨平台产物必须走 Docker CGO）。

### L1 · 静态判据（grep，零歧义）

```
# 1) 默认配置已带 -y
git show origin/main:backend/config/config.go | grep -n 'minimax-coding-plan-mcp'
#    期望（改动后）: Args: []string{"minimax-coding-plan-mcp", "-y"},

# 2) 示例配置与文档各带两行
git show origin/main:backend/config.example.yaml | grep -n -A2 '^  args:$'
git show origin/main:docs/image-understanding-api.md | grep -n -A2 '^args:$'

# 3) 全仓无第 5 处遗漏：所有含 "minimax-coding-plan-mcp" 的 args 都要带 -y
grep -rn "minimax-coding-plan-mcp" --include='*.go' --include='*.yaml' --include='*.sh' --include='Dockerfile' . | grep -v '^\./\.git/'
#    期望：config.go / config.example.yaml / docs / entrypoint.sh / Dockerfile / image_understanding.go —— 每一处的 args 元素集合均含 "-y"

# 4) entrypoint.sh / Dockerfile 未被改动
git diff origin/main -- entrypoint.sh Dockerfile   # 期望：空输出
```

### L2 · 单测（新增，plain `testing`，**禁止 testify**）

新建 `backend/handlers/image_understanding_test.go`：

```
cd backend && go test ./handlers/ -run TestNormalizeMCPArgs -v
```

表驱动用例（**每条都要有断言，不许只跑不判**）：

| # | 输入 `args` | 期望输出 | 覆盖的规则 |
|---|---|---|---|
| T1 | `nil` | `["minimax-coding-plan-mcp","-y"]` | 规则 1（空 → 默认） |
| T2 | `[]string{}` | `["minimax-coding-plan-mcp","-y"]` | 规则 1 |
| T3 | `[]string{"minimax-coding-plan-mcp"}` | `["minimax-coding-plan-mcp","-y"]` | 规则 3（**存量配置主场景**） |
| T4 | `[]string{"minimax-coding-plan-mcp","-y"}` | `["minimax-coding-plan-mcp","-y"]` | 规则 2（**不重复注入**） |
| T5 | `[]string{"-y","minimax-coding-plan-mcp"}` | `["-y","minimax-coding-plan-mcp"]` | B1（任意位置已含 → 不动） |
| T6 | `[]string{"minimax-coding-plan-mcp==0.0.4"}` | `["minimax-coding-plan-mcp==0.0.4","-y"]` | 版本限定写法 |
| T7 | `[]string{"@modelcontextprotocol/server-filesystem","/data"}` | **原样不变** | B4（门控，不误伤） |
| T8 | `[]string{""}` | **原样不变** | B3（空串） |
| T9 | `[]string{"minimax-coding-plan-mcp",""}` | `["minimax-coding-plan-mcp","","-y"]` | B3 |
| T10 | `[]string{"minimax-coding-plan-mcp","--yes"}` | `["minimax-coding-plan-mcp","--yes","-y"]` | B2（`--yes` 不算已含） |

补充断言（**T11，必须有**）：传入 `[]string{"minimax-coding-plan-mcp"}`（**len==cap==1**）时，函数返回的切片**不得**与入参共享底层数组 —— 断言「调用后原入参 slice 长度与内容不变」+「返回切片追加后再读原入参仍为 1 个元素」（覆盖 K3 / §2.6 的 data race 约束）。

新建 `backend/handlers/minimax_mcp_args_regression_test.go`：

```
cd backend && go test ./handlers/ -run TestMiniMaxMCPArgsDefaultRegression -v
```

- 护栏 1：`config.DefaultConfig().MiniMaxMCP.Args` 必须 `== []string{"minimax-coding-plan-mcp","-y"}`（逐元素比对，不许只判 `len`）。
- 护栏 2：`config.Load(<一个不存在的路径>)` 之后（无 `config.yaml` 的全新部署）仍是同一组 args。
- 护栏 3（**建议加**）：把 `backend/config.example.yaml` 的 `minimax_mcp.args` 用 `yaml.Unmarshal` 读出来，断言与 `DefaultConfig()` 逐字一致 —— 这样**示例配置**与**默认配置**的漂移以后会被单测自动拦住（正是本 issue 的根因）。

### L3 · 编译门槛

```
cd backend && go build ./... && go vet ./...
```
（只做编译/静态校验；**禁止**在 macOS 上产出跨平台二进制 —— R11）

### L4 · 回归范围

- 只跑：`go test ./handlers/ -run 'TestNormalizeMCPArgs|TestMiniMaxMCPArgsDefaultRegression' -v`。
- **禁止**跑 `go test ./handlers/` 全包（main 上已有 3 FAIL + 1 挂起，属历史遗留）。
- 禁止跑 `go test ./...`。

### L5 · 人工核对（1 分钟）

- 打开 `backend/config/config.go` / `backend/config.example.yaml` / `docs/image-understanding-api.md` / `entrypoint.sh` 四处并排看 —— **四处 args 逐字一致**（DoD 3）。
- 确认 `git diff origin/main --stat` 的改动文件集合**只有** §3.6 列出的那几个（无越界改动，S9）。

### L6 · 不需要（也禁止）做的

- ❌ 不起 dev server、不跑 `go run`、不 `docker compose up`、不 `./deploy.sh`（R1）。
- ❌ 不连部署机 `192.168.31.201`（R1）。
- ❌ 不做真机 / 线上容器验收（由 jaxiu 合并后手工上线验收，D5）。

### L7 · 验收标准对照（父 issue FINN-56 DoD 1–6）

| DoD | 判据 | 由哪条覆盖 |
|---|---|---|
| 1 | `DefaultConfig().MiniMaxMCP.Args == ["minimax-coding-plan-mcp","-y"]` | L1-1、L2 护栏 1/2/3 |
| 2 | `config.example.yaml` 与 `docs/image-understanding-api.md` 的示例 args 与之逐字一致 | L1-2、L2 护栏 3 |
| 3 | 三处与 `entrypoint.sh:55` 完全一致（4 处对齐） | L1-3、L5 |
| 4 | `go build ./... && go vet ./...` 通过 | L3 |
| 5 | `startMCPProcess` 对「args 为空 / 缺 `-y` / 已含 `-y`」三种输入行为确定且不重复注入 | L2 T1–T5（+T6–T10 边界） |
| 6 | 存量单测不回归 | L4（只 `-run` 限定范围；`handlers` 包既有 FAIL 属历史遗留） |

---

## 7. 硬约束（写给下游，无条件遵守）

- **R1**：只写代码 + review；**不部署**、不 `go run`、不 `docker compose up`、不 `./deploy.sh`；**不访问 / 不 curl / 不 SSH 部署机 `192.168.31.201`**。部署永远由 jaxiu 手工执行。
- **R11**：后端跨平台产物必须走 Docker（CGO），**不从 macOS 直接出二进制**；本地只 `go build ./...` 校验。
- **R2**：本改动是纯服务端进程编排，**没有**客户端可替代的计算 → 走服务端合理。
- **R3 / R4**：本方案**无前端改动** → 不涉及。
- **R6**：不涉及分享资产。
- **R8**：不涉及长耗时 AI 调用。
- **R9**：不涉及 MCP 协议版本。
- **R10**：不涉及 Planner。
- **R5**：不涉及 SQLite（`:memory:` 连接池陷阱无需处理）。
- **PM 拍板纪律**：不碰 `feature/minimax-share-all` 分支的任何内容；不扩展到 FINN-55 的 (A)(B)。
- **代码清真（S1–S10）**：见 §8 的自查表，命中即返工。

---

## 8. S1–S10 代码异味自查（本次改动面）

> 自查范围 = 本方案的全部改动面（3 个配置/文档文件 + 1 个 Go 文件 + 2 个新建测试文件）。
> 逐条给 grep 原始输出；**未命中的也写「0 命中」**。

| 规则 | 结论 | 依据 |
|---|---|---|
| **S1 死代码** | **0 命中**（改动后反而**减少**死代码） | 原 `if len(args) == 0 { args = []string{...} }`（`image_understanding.go:698-700`）在默认路径永不触发 → 被 `normalizeMCPArgs` 取代后成为**被真实调用**的活代码 |
| **S2 未使用（import / 参数 / 局部变量）** | **0 命中** | 新增 helper 的每个入参都被使用；`strings` 包已 import（`image_understanding.go` 内已在用 `strings.TrimSpace`），**不新增 import** |
| **S3 调试残留**（`console.log` / `fmt.Println` / `TODO` / `FIXME` / `XXX` / `HACK`） | **0 命中** | grep 4 个在改文件：`backend/config/config.go` → 0 命中；`backend/config.example.yaml` → 0 命中；`docs/image-understanding-api.md` → 0 命中；`backend/handlers/image_understanding.go` → 0 命中 |
| **S4 注释掉的代码块** | **0 命中** | 本次不新增注释代码块；既有文件无成块注释代码 |
| **S5 重复逻辑** | **0 命中** | 归一化**只在 `startMCPProcess` 一处收口**做；**禁止**在别处再写一份（写进实现约束） |
| **S6 魔法数 / 魔法串** | **需处理 → 已给方案** | `"-y"` 与 `"minimax-coding-plan-mcp"` **必须**提为常量 `minimaxMCPYesFlag` / `minimaxMCPPackage`（§3.2），**禁止**散落字面量 |
| **S7 空 `catch` / 忽略的 error** | **0 命中** | 本次无新增 error 忽略路径；`normalizeMCPArgs` 不返回 error（纯函数） |
| **S8 命名风格不一致** | **0 命中** | 常量/函数命名沿用同文件既有风格（参照同文件 `normalizeTransport`，`image_understanding.go:1000`）；测试文件命名沿用既有范式（`singbox_default_config_regression_test.go`） |
| **S9 越界改动** | **需守住 → 已约束** | 只允许改 §3.6 列出的文件；**禁止**改 `entrypoint.sh` / `Dockerfile` / `frontend/**` / DB / API 契约 |
| **S10 遗留占位** | **0 命中** | 新增代码不含 `TODO` / 占位实现；`normalizeMCPArgs` 四条规则全部有实现 |

---

## 9. 未决问题（需人工裁决或后续处理，**均不阻塞 Stage 2**）

| # | 问题 | 我的倾向 | 影响 |
|---|---|---|---|
| 1 | 部署机 `./backend/config.yaml`（gitignored）是否存在、其 `minimax_mcp.args` 写了什么 —— **R1 禁探，本机不可核实** | 不猜、不探；等 jaxiu 上线后核对 | **不影响方案正确性**：两种内容在方案下都正确（已含 `-y` → 幂等不动；不含 → 运行期补上） |
| 2 | `docs/image-understanding-api.md:183` 的 `timeout_seconds: 300` 与 `backend/config.example.yaml:239` / `backend/config/config.go:531` 的 `600` 不一致（**同属配置漂移，但不在本 issue DoD 范围**） | **另开 issue** 单独对齐，**本 issue 不动**（避免扩大爆炸半径） | 无功能影响（真实生效值取 `config.yaml` > 默认 600） |
| 3 | `Dockerfile:88` 的 build 期预热 `uvx minimax-coding-plan-mcp -y --help >/dev/null 2>&1 \|\| true` 用 `\|\| true` 吞掉了所有失败 → **不能证明 uvx cache 一定被预热成功** | **另开 issue**（部署面，不在本 issue 范围） | 可能导致容器首次运行仍需现装包；与 `-y` 对齐无关 |

> 按 D6 纪律：以上 3 条均为**非阻塞**项，不需 PM 拍板即可继续 Stage 2；如需裁决请 @devtools-pm。

---

## 10. 参考依据

- 仓库内：`entrypoint.sh:55`、`Dockerfile:88`、`backend/handlers/image_understanding.go:699`（三处 `-y` 基准）
- MiniMax MCP 官方安装 / 客户端配置示例（`uvx minimax-coding-plan-mcp -y`；Claude Desktop / Claude Code CLI / OpenCode 的 MCP 配置块）：
  - https://skillsllm.com/skill/minimax-coding-plan-mcp
  - https://www.mcpworld.com/zh/detail/031076356b5a026f0788fac8435c45d7
  - https://www.codefather.cn/post/2022508136270663682
  - https://skillfed.io/packages/minimax-coding-plan-mcp
- uv `tool run` 的 flag 位置约定（uv 自身 flag 在包名**之前**，包名之后的参数转发给被调程序）：
  - https://learnlinux.appsved.com/man/uvx.html
  - https://www.pattern.institute/uv/reference/uvx.html
  - https://instagit.com/astral-sh/uv/uv-tool-run-install-commands/

---

## 11. 交付说明（给 Stage 2 / Stage 3）

- 本方案**不含代码改动**，Stage 2 按 §3.1 / §3.2 实施即可。
- Stage 2 若对「B4 门控」有异议（认为应盲注入），**必须**先回到本 issue 讨论 —— 盲注入会破坏用户自定义 MCP Server 调用（§5 K2）。
- Stage 3 交付时必须在完成评论里**原文报告分支名与 commit hash**（父 issue 状态纪律）。
- 本方案文档位于 `docs/plans/minimax-mcp-args-alignment.md`，随 Stage 6 一并合入 `main`（注意：Stage 6 合并范围历来只圈代码分支，**本文件需被显式纳入**）。
