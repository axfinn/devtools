# MiniMax MCP 默认 args 与 `entrypoint.sh` 对齐 + 启动参数兜底加固 — 技术方案（FINN-57 / FINN-56 Stage 1）

- 原 issue：FINN-56（父）→ 本子 issue FINN-57（Stage 1 技术方案设计）→ 本文档修订 FINN-63（Stage 1 修订 2，闭合 Stage 2 评审 FINN-58 的 2 个阻塞项）
- 模块：后端配置层（`backend/config/`）+ MCP 进程启动点（`backend/handlers/image_understanding.go`）
- 基线：`origin/main` @ `9992ac652553752cef5c73f6c43012e9b69a7e33`
- **行号口径（本次修订补充，N-1）**：本文的行号**分两套口径**，引用前必须看清：
  - **改动点行号**（§3.1 变更清单、§6 静态判据）→ 以 **`origin/main` @ `9992ac6`** 为准。
  - **勘察取证行号**（§2 的引用）→ 取自 **architect 工作树**（`origin/main` + 9 行偏移，**仅涉及 `backend/config/config.go`**（839 vs 848 行）；`backend/handlers/image_understanding.go` 与 `origin/main` **逐字一致、偏移 = 0**）。下游引用前请用 `git show origin/main:<file> | grep -n` 复核。
  - **Stage 3 / 4 / 5 一律以 `origin/main` 口径为准**。
- 状态：**设计稿，不含任何代码改动**；实现由 devtools-backend 执行，验收由 devtools-qa 执行
- 勘察方式：全仓 grep + 逐段 Read + `git show origin/main` 逐行核对 + 供应商公开文档检索；**未起 dev server、未跑 go build / go test、未部署、未碰部署机 `192.168.31.201`（R1）**
- 本文不含任何未标注来源的推断；凡属「推算」一律显式标注

---

## 0. 一页纸结论（给下游先看这段）

| 项 | 结论 |
|---|---|
| 必改点是否只有 3 处？ | **是。** 全仓 `minimax-coding-plan-mcp` 硬编码 args 共 **7 处命中**：**3 处缺 `-y` 必改**（`backend/config/config.go` / `backend/config.example.yaml` / `docs/image-understanding-api.md`），**4 处已正确**（`entrypoint.sh` / `Dockerfile` / 兜底常量 `backend/handlers/image_understanding.go:699` / 纯注释 `.env.example:28`）。**没有第 8 处遗漏点。** |
| 兜底加固做不做？ | **做（方案乙）**，但必须带「**只对 minimax 包生效**」的门控，否则会误伤用户自定义 MCP Server 调用。PM 倾向的方案乙方向正确，我补一条必要门控（§3.3）。 |
| `-y` 的位置 | **必须保持在包名之后**（`uvx minimax-coding-plan-mcp -y`），这是 MiniMax 官方文档与 Claude Desktop / Claude Code / OpenCode 官方客户端配置的统一写法。**禁止**改写成 `uvx -y minimax-coding-plan-mcp`（那是 uv 自己的 `--yes`，语义不同）。§3.1 有依据。 |
| **「已含 `-y`」怎么判定？** | **按位置判定，不按出现顺序判定**（本次修订的核心，闭合 B-1）：先定位 minimax 包名元素下标 `idx`，只认 `args[idx+1:]` 里的 `-y`。**包名之前**的 `-y` / `--yes` 是 **uv 层开关**（uv 会消费掉，MCP 进程收不到）→ **一律不算「已含」** → 末尾补 `-y`。详见 §2.5 / §3.3 B1。 |
| 对 PM 前提的一处事实修正 | PM 说「容器部署不受影响」→ **不完全成立**，见 §2.3。容器后端生效的配置来自部署机 `./backend/config.yaml`（gitignored）或 `backend/config.example.yaml`；若部署机没有自定义 `config.yaml`，**容器后端跑的就是缺 `-y` 的 example**。本方案修完恰好覆盖这种情况。 |
| 数据面影响 | **零**：无 DB 表、无 Redis key、无 API 契约变更、无前端改动。 |
| 未决 | **3 条**非阻塞（见 §9.1），均不阻塞 Stage 2。另有 **3 条未定义边界已由 PM 拍板**（U-1 / U-2 / U-3，见 §9.2），不再留给实现方取舍。 |

---

## 1. 目标与非目标

### 1.1 目标

1. 把仓库内 **3 处** MiniMax MCP 默认 args（`backend/config/config.go` 的 `DefaultConfig`、`backend/config.example.yaml`、`docs/image-understanding-api.md`）与 `entrypoint.sh:55` 的容器配置**逐字对齐**，补上 `-y`。
2. 把 `startMCPProcess` 里那段**当前近乎死代码**的兜底升级为**确定性的启动参数归一化**，让**存量 `config.yaml` 用户**（args 非空但缺 `-y`）也能被兜住。
3. 给出可判定、可自动化的验收清单（DoD 1–6）与单点回滚路径。

### 1.2 非目标（明确不做）

- ❌ 不动 `entrypoint.sh`（其 `:55` 已正确）。
- ❌ 不动 `Dockerfile`（其 `:88` 是 build 期 uvx cache 预热，与本 issue 的运行期 args 无关；另见 §9.1 未决 3）。
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

补充核对：`git grep -n "uvx" origin/main` 全量扫过，**除 §2.1 表已列出的位置外，没有任何其它运行期 args 构造点** —— `backend/handlers/image_understanding.go:694` 是 `cfg.Args` 的**唯一**消费方：

```
origin/main:backend/handlers/image_understanding.go:696:	command = "uvx"
origin/main:backend/handlers/image_understanding.go:989:	if command == "uvx" {
```

（`grep -rn "\.Args\b" backend/ --include="*.go"` 结果中，除 `image_understanding.go:694` 外全部是 HTTP 请求体的 `req.Args`，与 MCP 启动参数无关。）

### 2.2 `entrypoint.sh:55` 服务的是**另一个消费方**（重要）

`entrypoint.sh:47-72` 写的是 `${CLAUDE_HOME}/settings.json` 里的 `mcpServers.MiniMax` —— 这是给**容器内的 Claude Code CLI**（autodev 任务）用的 MCP 客户端配置，**不是** Go 后端读的配置。

Go 后端读的是 `config.yaml`（architect 工作树 `backend/config/config.go:627` 的 `Load(path)`，origin/main 口径 `:618`），容器里由 `entrypoint.sh:135-143` 准备：

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
> 部署机 `backend/config.yaml` 是否存在、args 写了什么 —— **属 R1 禁探范围**，本方案不做任何假设，写进 §9.1 未决 1。

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

`cfg.Args` 的来源（architect 工作树 `backend/config/config.go:627-641`，origin/main 口径 `:618-632`）：

```go
cfg := DefaultConfig()          // architect:629 / origin/main:620 —— 默认 args = ["minimax-coding-plan-mcp"]（缺 -y）
...
if err := yaml.Unmarshal(data, cfg); err != nil {   // architect:639 / origin/main:630 —— 只覆盖 yaml 里出现的键
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

**推论（本次修订补充 —— 闭合 B-1 的语义根）**：uv 层的 `--yes` / `-y`（**包名之前**）与包层的 `-y`（**包名之后**）是**两个不同的开关**。因此：

- `["-y","minimax-coding-plan-mcp"]`（uv 位置的**短**形式）与 `["--yes","minimax-coding-plan-mcp"]`（uv 位置的**长**形式）**都不等于「包层 `-y` 已就位」** —— uv 会把包名之前的 `-y` 当作自己的 `--yes` 消费掉，**MCP 进程收不到 `-y`**。
- → 判定「`-y` 已含」必须**以 minimax 包名元素下标为基准**，**只认包名之后**的 `-y`（§3.3 B1 / B2）。这两条写法在本方案下都会被归一化成 `["…","minimax-coding-plan-mcp","-y"]` 的形态。

依据来源见 §10（供应商文档 + 客户端官方配置示例检索）。

### 2.6 全局配置共享（决定实现写法的一条硬约束）

`backend/config/config.go`（architect 工作树行号，origin/main 口径 `-9`）：

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
2. 归一化是**纯函数、无状态、无迁移、无外部依赖**，改动面 = 1 个函数 + 3 个 helper + 1 个常量组，回滚 = revert 单个 commit。
3. 归一化后，配置层与运行期形成**双保险**：新装用户走默认值即正确；存量用户由运行期兜住。
4. 副作用可控：只要加「**只对 minimax 包生效**」的门控（§3.3 规则 4），**非 minimax 的调用一律字节级不变**。

**具体写法**（`backend/handlers/image_understanding.go`，紧邻 `startMCPProcess` 放置）：

```go
const (
	// minimaxMCPPackage MiniMax MCP Server 的 uvx 包名（与供应商文档一致）
	minimaxMCPPackage = "minimax-coding-plan-mcp"
	// minimaxMCPYesFlag MiniMax MCP 包自身的「跳过确认」标志，须紧跟包名
	minimaxMCPYesFlag = "-y"
	// minimaxMCPPackageSeps 包名后允许出现的分隔符（PEP 440 版本约束 / extras 变体）
	minimaxMCPPackageSeps = "=@><~!"
)

// normalizeMCPArgs 归一化 MCP Server 启动参数（纯函数，不修改入参底层数组）：
//  1. args 为空                          → 官方默认调用 ["minimax-coding-plan-mcp", "-y"]
//  2. args 未命中 minimax 包名            → 原样返回（用户自定义 MCP Server，绝不注入）
//  3. args 命中 minimax 包名，且**包名之后**已含 "-y" → 原样返回（不重复注入）
//  4. args 命中 minimax 包名，但**包名之后**没有 "-y" → 末尾追加 "-y"（兜住存量 config.yaml）
//
// 注意规则 3/4 的「包名之后」：包名之前的 "-y" / "--yes" 是 uv 层的开关，
// 会被 uv 消费掉，MiniMax MCP 进程收不到 —— 不能算「已含」（见 §2.5）。
func normalizeMCPArgs(args []string) []string {
	if len(args) == 0 {
		return []string{minimaxMCPPackage, minimaxMCPYesFlag}
	}
	idx := indexOfMinimaxMCPPackage(args)
	if idx < 0 {
		return args
	}
	// 只有包名之后的 "-y" 才算「已含」
	for _, a := range args[idx+1:] {
		if a == minimaxMCPYesFlag {
			return args
		}
	}
	out := make([]string, 0, len(args)+1)
	out = append(out, args...)
	out = append(out, minimaxMCPYesFlag)
	return out
}

// indexOfMinimaxMCPPackage 返回 args 中第一个指向 MiniMax MCP 包的元素下标；未命中返回 -1。
// 命中路径：
//  1. 任意位置直接出现包名 spec（元素自身命中）；
//  2. uvx 的 "--from" / "--with" 之后紧跟的元素即「包名 spec」（PM 拍板 U-2）。
func indexOfMinimaxMCPPackage(args []string) int {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if (a == "--from" || a == "--with") && i+1 < len(args) {
			if isMinimaxMCPPackageSpec(args[i+1]) {
				return i + 1
			}
			i++ // 该 flag 的值已被消费，不再当普通位置参数扫
			continue
		}
		if isMinimaxMCPPackageSpec(a) {
			return i
		}
	}
	return -1
}

// isMinimaxMCPPackageSpec 判断单个元素是否指向 MiniMax MCP 包。
// 命中口径（PM 拍板 N-4 / U-3）：
//  1. 精确等于 "minimax-coding-plan-mcp"；
//  2. 以 "minimax-coding-plan-mcp" + ∈{=, @, >, <, ~, !} 开头（PEP 440 版本约束 / extras 变体）。
// 禁止无脑 strings.HasPrefix("minimax-coding-plan-mcp")：会误命中
// "minimax-coding-plan-mcp-fork" 这类同名第三方包。
func isMinimaxMCPPackageSpec(s string) bool {
	if s == minimaxMCPPackage {
		return true
	}
	if len(s) <= len(minimaxMCPPackage) {
		return false
	}
	if s[:len(minimaxMCPPackage)] != minimaxMCPPackage {
		return false
	}
	// 包名之后紧跟的必须是允许的分隔符，否则视为同名第三方包
	return strings.IndexByte(minimaxMCPPackageSeps, s[len(minimaxMCPPackage)]) >= 0
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

- **位置判定硬约束（本次修订新增，闭合 B-1）**：`normalizeMCPArgs` 必须以 **minimax 包名元素下标 `idx`** 为基准判定 `-y` 的位置 —— **只扫 `args[idx+1:]`**；**禁止**「全切片任意位置命中 `-y` 即判定已含并原样返回」的实现。
- **门控硬约束**：包名匹配必须走 `isMinimaxMCPPackageSpec`（元素相等 或 包名 + ∈{`=`,`@`,`>`,`<`,`~`,`!`} 前缀）；**禁止**无脑 `strings.HasPrefix(包名)`（会误命中 `minimax-coding-plan-mcp-fork`）。
- `normalizeMCPArgs` 返回**新切片**或**原切片**均可，但**禁止**在 `cfg.Args` 的底层数组上原地 `append`（§2.6 的 data race）。上面的写法用 `make` + `copy` 语义满足。
- `strings` 包在 `image_understanding.go` 已 import（`strings.TrimSpace` 在用），无需新增 import。
- 常量名 / 函数名**参照同文件 `normalizeTransport` 的命名风格**（camelCase、动词开头；同文件 `:1000`）；**禁止**把 `"-y"` / `"minimax-coding-plan-mcp"` 当字面量散落在多处（S6）。
- **禁止**把 `-y` 提到包名之前（§2.5）。
- 归一化只在 `startMCPProcess` 这一个收口点做，**不要**在别处再写一份（§S5）。

### 3.3 边界规则（PM 点名要定的三个问题 + 追加两条）

| # | 问题 | 规则 | 理由 |
|---|---|---|---|
| B1 | `-y` 出现在**任意位置**都算「已含」？ | **否 —— 按位置判定**。先定位 minimax 包名元素下标 `idx`（`indexOfMinimaxMCPPackage`），再在 **`args[idx+1:]`** 内精确匹配 `-y`；命中 → 原样返回；**未命中**（含「`-y` 只出现在**包名之前**、即 uv 位置」的情况）→ 在**末尾**追加 `-y`（末尾必在包名之后，满足 §2.5） | uv 层的 `-y`（= `--yes`）与包层的 `-y` 是**两个不同层、不同语义的开关**（§2.5）。`["-y","minimax-coding-plan-mcp"]` 里的 `-y` **被 uv 消费**，MCP 进程收不到 → 「语义已达成」不成立，必须在末尾补第二个 `-y` → 结果 `["-y","minimax-coding-plan-mcp","-y"]`（uv 吃第一个、进程收第二个） |
| B2 | `--yes` 算不算「已含」？ | **不算，且规则与 B1 统一**：`--yes` 与 `-y` 同为 **uv 层开关** → **`--yes` 在任何位置都不算「已含」**（包名**之前** → 它是 uv 层开关，被 uv 消费；包名**之后** → 它被 uv 转发给 MCP 进程，而进程不认它）→ 一律**末尾补 `-y`**。B2 **不再单列**，仅作 B1 的示例 | 若用户写成 `["--yes","minimax-coding-plan-mcp"]`，本规则会补 `-y` → `uvx --yes minimax-coding-plan-mcp -y` → uv 吃 `--yes`、进程收 `-y`，**结果正确** |
| B3 | **空串元素**怎么办？ | **不 TrimSpace、不丢弃空串、不特殊处理**。空串既不等于 `-y`、也不匹配包名 → 不触发「已含」判定。若 `args = ["minimax-coding-plan-mcp", ""]` → 命中包名 → 结果 `["minimax-coding-plan-mcp","","-y"]`（空参数是用户原配置的问题，本方案只加不减）；若 `args = [""]` → 不匹配包名 → 原样透传，由 uv 自己报错 | 最小干预原则：只加 `-y`，不动用户其它参数；误 trim 会把畸形值判成「已含」而**漏掉** `-y` |
| B4 | **非 minimax 的自定义 MCP 调用**会不会被误伤？ | **门控掉**：`args` 不命中 minimax 包名时**绝不注入** | 反例：`command: "npx"` + `args: ["@modelcontextprotocol/server-filesystem","/data"]`，盲注入 `-y` 会被 npx 转发给该 server → 启动失败。门控后这类调用**字节级不变** |
| B5 | 重复注入 `-y` 有害吗？ | 无害但**不允许**：已含（**包名之后**）即返回 | argparse 的 store_true 标志重复传无害，但保持「幂等、确定性」更重要（DoD 5 要求可判定） |

**门控规则（B4）补充说明 —— 本次修订按 PM 拍板放宽（N-4 / U-2 / U-3）**：

1. **包名匹配口径**：元素 **等于** `minimax-coding-plan-mcp`，**或**以 `minimax-coding-plan-mcp` + ∈{`=`,`@`,`>`,`<`,`~`,`!`} **开头**即算命中（兼容 uvx 的版本限定 / PEP 440 约束写法，如 `minimax-coding-plan-mcp==0.0.4`、`minimax-coding-plan-mcp@0.0.4`）。
   **禁止**无脑 `strings.HasPrefix("minimax-coding-plan-mcp")` —— 会误命中 `minimax-coding-plan-mcp-fork` 这类同名第三方包。
   > **已知边界（拍板集合之外，不阻塞）**：`minimax-coding-plan-mcp[extra]` 这种 **extras 括号写法**不在上述拍板集合内 → **不命中** → 不注入（维持原样，**不比现状更差**）。若后续要覆盖，另开 issue 扩展该分隔符集合。
2. **包名藏在 flag 值里的形态（U-2）**：`uvx --from minimax-coding-plan-mcp` / `uvx --with minimax-coding-plan-mcp` 这类写法，**包名 spec 是 `--from` / `--with` 之后紧跟的那个元素** → 门控必须识别它为 minimax 调用 → 注入 `-y`（注入位置 = 该元素之后 = 末尾）。否则该形态永久漏兜底。
3. **追加位置固定在末尾** —— 因为门控已保证 args 内含包名元素，末尾必然在包名之后，满足 §2.5 的位置约束。

### 3.4 数据模型 / 存储

**无变更**。本方案不新增 SQLite 表、不新增 Redis key、不改 `config.yaml` 的字段结构（`args` 字段早已存在，origin/main `backend/config/config.go:329`）。

### 3.5 API 契约

**无变更**。不新增 / 不修改任何 HTTP 端点、请求体字段、响应体字段、状态码、错误码。`/api/image-understanding/*` 的行为差异仅体现在「MCP 子进程能否成功起起来」，属**内部实现**。

### 3.6 后端改动清单（汇总）

| 文件 | 改动 | 行数 | 类型 |
|---|---|---|---|
| `backend/config/config.go` | `DefaultConfig().MiniMaxMCP.Args` 补 `-y` | 1 行（`:530`） | 配置 |
| `backend/config.example.yaml` | 示例 args 补 `-y` | +1 行（`:237` 后） | 配置 |
| `docs/image-understanding-api.md` | 文档示例 args 补 `-y` | +1 行（`:182` 后） | 文档 |
| `backend/handlers/image_understanding.go` | 新增 3 个常量 + 3 个 helper（`normalizeMCPArgs` / `indexOfMinimaxMCPPackage` / `isMinimaxMCPPackageSpec`）；`startMCPProcess` 改 1 行 | +~50 行 / 改 1 行 | 代码 |
| `backend/handlers/image_understanding_test.go`（**新建**） | `normalizeMCPArgs` 表驱动单测 | 新建 | 测试 |
| `backend/handlers/minimax_mcp_args_regression_test.go`（**新建**） | 默认配置回归护栏（照抄 `singbox_default_config_regression_test.go` 的既有范式） | 新建 | 测试 |

> 测试文件放 `backend/handlers/`（`package handlers`，与该目录既有 **38** 个 `_test.go` 一致（`origin/main` 口径；勘察时 architect 树为 40，差异来自 architect 分支自带的 2 个测试文件 `ssrule_test.go` / `ssrule_rules_test.go`）），**不要**新建 `backend/config` 测试包 —— 该目录当前**只有 `config.go`，无任何 `_test.go`**（已核实）。

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
| **args 为空、且 `command` 不是 `uvx`** | **走官方默认调用 `["minimax-coding-plan-mcp","-y"]`** —— 这是**既有行为**（原 `len(args)==0` 分支同款），**本次不改**（PM 拍板 U-1，见 §9.2）。**不要**为此另开分支逻辑 |
| **自定义 MCP Server（args 非空且不含 minimax 包名）** | **字节级不变**（门控 B4）。注意：只有 **args 非空** 时才成立；args 为空时按上一行走官方默认调用（与现状一致） |
| **args 非空 + `command` == `uvx` + 不含 minimax 包名** | **原样不变**（B4 门控不命中） |
| 迁移脚本 | **不需要** |
| 回滚 | 见 §5 |

---

## 5. 风险与回滚

| # | 风险 | 等级 | 缓解 |
|---|---|---|---|
| K1 | 注入 `-y` 改变了 MCP 启动行为（跳过交互确认） | 低 | `-y` 是 MiniMax 官方调用形式的一部分，官方客户端配置即如此；无功能损失 |
| K2 | 盲注入误伤用户自定义 MCP Server（npx 等） | 中 | 门控 B4：非 minimax 调用一律不动 |
| K3 | 原地 `append` 写全局共享切片 → data race | 中 | §2.6 约束 + 实现约束「必须新建切片」；单测覆盖「入参切片不被修改」 |
| K4 | 追加到末尾导致 MCP 进程收到位置不对的参数 | 低 | 门控保证包名在 args 内，末尾必在包名之后；且 `-y` 是包自身认识的 flag |
| K5 | 部署机 `backend/config.yaml` 内容未知 → 容器实际行为仍可能与预期不同 | 低 | 两种情况在方案下都成立（已含 `-y` → 不动；不含 → 补上）；部署机核对列入 §9.1 未决 1，由 jaxiu 上线后确认 |
| K6 | 归一化函数成为新的「魔法行为」，将来 package 改名要同步 | 低 | 包名提为常量 + 单测锁定 |
| K7 | 门控放宽后误命中同名第三方包（如 `minimax-coding-plan-mcp-fork`） | 低 | 匹配限定为「相等 或 包名 + ∈{`=`,`@`,`>`,`<`,`~`,`!`} 前缀」；**禁止**无脑 `HasPrefix`；单测 T15 锁定 |

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

> **基线口径纪律（本次修订补充，闭合 B-2）**：本节凡是判定「**改动后**内容」的判据，一律以**被验收分支的工作树**为准 —— 用**工作树路径** / `git show HEAD:<file>` / `git diff origin/main -- <file>`；`git show origin/main:<file>` **只允许**出现在「基线取证」类说明里，**不得**用于判定改动后内容（origin/main 上永远是未改动的旧内容，据此判定必然 100% 误判）。

```
# 1) 默认配置已带 -y（读被验收分支的工作树，不读 origin/main）
grep -n 'minimax-coding-plan-mcp' backend/config/config.go
#    期望：命中 1 行，内容为 Args: []string{"minimax-coding-plan-mcp", "-y"},
#          （origin/main 口径行号 530；工作树实际行号随验收分支基线浮动 —— 判定看**内容**，不看行号）

# 2) 示例配置与文档各带两行（注意：args: 是 2 空格缩进，必须用 '^  args:$'，'^args:$' 匹配不到）
grep -n -A2 '^  args:$' backend/config.example.yaml
grep -n -A2 '^  args:$' docs/image-understanding-api.md
#    期望：两处都命中，且 -A2 命中的 3 行里同时出现 "minimax-coding-plan-mcp" 与 "-y"

# 3) 全仓无第 8 处遗漏：所有含 "minimax-coding-plan-mcp" 的 args 都要带 -y
grep -rn "minimax-coding-plan-mcp" --include='*.go' --include='*.yaml' --include='*.sh' --include='Dockerfile' --include='*.md' . | grep -v '^\./\.git/'
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
| T3 | `[]string{"minimax-coding-plan-mcp"}` | `["minimax-coding-plan-mcp","-y"]` | 规则 4（**存量配置主场景**） |
| T4 | `[]string{"minimax-coding-plan-mcp","-y"}` | `["minimax-coding-plan-mcp","-y"]` | 规则 3（**不重复注入**） |
| T5 | `[]string{"-y","minimax-coding-plan-mcp"}` | `["-y","minimax-coding-plan-mcp","-y"]` | **B1 位置化（uv 位置的 `-y` 不算已含 → 末尾补）** ← 本次修订改写 |
| T6 | `[]string{"minimax-coding-plan-mcp==0.0.4"}` | `["minimax-coding-plan-mcp==0.0.4","-y"]` | 版本限定写法（N-4 放宽口径） |
| T7 | `[]string{"@modelcontextprotocol/server-filesystem","/data"}` | **原样不变** | B4（门控，不误伤） |
| T8 | `[]string{""}` | **原样不变** | B3（空串） |
| T9 | `[]string{"minimax-coding-plan-mcp",""}` | `["minimax-coding-plan-mcp","","-y"]` | B3 |
| T10 | `[]string{"minimax-coding-plan-mcp","--yes"}` | `["minimax-coding-plan-mcp","--yes","-y"]` | B2（`--yes` 不算已含） |
| T11 | `[]string{"--yes","minimax-coding-plan-mcp"}` | `["--yes","minimax-coding-plan-mcp","-y"]` | B2 示例（uv 位置的长形式 → 末尾补） ← 本次修订新增 |
| T12 | `[]string{"minimax-coding-plan-mcp","-y"}` | `["minimax-coding-plan-mcp","-y"]` | **B1 位置化（包名**之后**已含 `-y` → 原样不动）+ 幂等性**；与 T5 构成对 ← 本次修订新增 |
| T13 | `[]string{"--from","minimax-coding-plan-mcp"}` | `["--from","minimax-coding-plan-mcp","-y"]` | U-2 门控（包名在 `--from` 的值里 → 注入）+ B1 位置化 ← 本次修订新增 |
| T14 | `[]string{"--with","minimax-coding-plan-mcp"}` | `["--with","minimax-coding-plan-mcp","-y"]` | U-2 门控（`--with` 同款） ← 本次修订新增 |
| T15 | `[]string{"minimax-coding-plan-mcp-fork"}` | **原样不变** | N-4 禁止无脑 `HasPrefix`（不误命中同名第三方包） ← 本次修订新增 |
| T17 | `[]string{"-y","--from","minimax-coding-plan-mcp"}` | `["-y","--from","minimax-coding-plan-mcp","-y"]` | U-2 门控（包名在 `--from` 的值里 → 末尾补 `-y`）+ B1 位置化（包名**之前**的 uv 层 `-y` 原样保留、不上提也不删除） ← 本次勘误新增（PM 拍板边界 3） |

> **T4 与 T12 共用同一输入、同一期望，但锁定两条不重叠的性质**：T4 锁「不重复注入（幂等）」，T12 锁「B1 位置化命中分支（包名**之后**的 `-y` 才算已含）」；T5 锁「包名**之前**的 `-y` **不算**已含」。三者合起来才构成 B1 的完整判据。

补充断言（**T16，必须有**）：传入 `[]string{"minimax-coding-plan-mcp"}`（**len==cap==1**）时，函数返回的切片**不得**与入参共享底层数组 —— 断言「调用后原入参 slice 长度与内容不变」+「返回切片追加后再读原入参仍为 1 个元素」（覆盖 K3 / §2.6 的 data race 约束）。

新建 `backend/handlers/minimax_mcp_args_regression_test.go`：

```
cd backend && go test ./handlers/ -run TestMiniMaxMCPArgsDefaultRegression -v
```

- 护栏 1：`config.DefaultConfig().MiniMaxMCP.Args` 必须 `== []string{"minimax-coding-plan-mcp","-y"}`（逐元素比对，不许只判 `len`）。
- 护栏 2：`config.Load(<一个不存在的路径>)` 之后（无 `config.yaml` 的全新部署）仍是同一组 args。可仿既有 `backend/handlers/singbox_default_config_regression_test.go:66` 的 `t.Setenv("CONFIG_PATH", nonexistent)` 写法。
- 护栏 3（**建议加**）：把 **`../config.example.yaml`**（**相对 `backend/handlers/`** 的路径；测试文件在 `backend/handlers/` 下，`backend/config.example.yaml` 的相对路径就是 `../config.example.yaml`）的 `minimax_mcp.args` 用 `yaml.Unmarshal` 读出来，断言与 `DefaultConfig()` 逐字一致 —— 这样**示例配置**与**默认配置**的漂移以后会被单测自动拦住（正是本 issue 的根因）。

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
| 5 | `startMCPProcess` 对「args 为空 / 缺 `-y` / 已含 `-y`」三种输入行为确定且不重复注入 | L2 T1–T4、T12（+T5–T11、T13–T15、T17 边界；T16 为补充断言） |
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
- **位置约束（本次修订新增）**：判定「已含 `-y`」一律以 **minimax 包名元素下标** 为基准，**只认包名之后**的 `-y`；**禁止**「全切片任意位置命中即返回」。**本方案注入的** `-y` 只允许追加在包名之后；**用户已有的 uv 层 `-y` 原样保留、不上提也不删除**（§2.5 / §3.3 B1 / §6 T5）。
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
| **S8 命名风格不一致** | **0 命中** | 常量/函数命名**参照同文件 `normalizeTransport`（`image_understanding.go:1000`）的 camelCase、动词开头风格**；测试文件命名沿用既有范式（`singbox_default_config_regression_test.go`） |
| **S9 越界改动** | **需守住 → 已约束** | 只允许改 §3.6 列出的文件；**禁止**改 `entrypoint.sh` / `Dockerfile` / `frontend/**` / DB / API 契约 |
| **S10 遗留占位** | **0 命中** | 新增代码不含 `TODO` / 占位实现；`normalizeMCPArgs` 四条规则全部有实现 |

---

## 9. 未决问题与已拍板边界

### 9.1 未决问题（需人工裁决或后续处理，**均不阻塞 Stage 2**，共 **3 条**）

| # | 问题 | 我的倾向 | 影响 |
|---|---|---|---|
| 1 | 部署机 `./backend/config.yaml`（gitignored）是否存在、其 `minimax_mcp.args` 写了什么 —— **R1 禁探，本机不可核实** | 不猜、不探；等 jaxiu 上线后核对 | **不影响方案正确性**：两种内容在方案下都正确（已含 `-y` → 幂等不动；不含 → 运行期补上） |
| 2 | `docs/image-understanding-api.md:183` 的 `timeout_seconds: 300` 与 `backend/config.example.yaml:239` / `backend/config/config.go:531` 的 `600` 不一致（**同属配置漂移，但不在本 issue DoD 范围**） | **另开 issue** 单独对齐，**本 issue 不动**（避免扩大爆炸半径） | 无功能影响（真实生效值取 `config.yaml` > 默认 600） |
| 3 | `Dockerfile:88` 的 build 期预热 `uvx minimax-coding-plan-mcp -y --help >/dev/null 2>&1 \|\| true` 用 `\|\| true` 吞掉了所有失败 → **不能证明 uvx cache 一定被预热成功** | **另开 issue**（部署面，不在本 issue 范围） | 可能导致容器首次运行仍需现装包；与 `-y` 对齐无关 |

> 按 D6 纪律：以上 3 条均为**非阻塞**项，不需 PM 拍板即可继续 Stage 2；如需裁决请 @devtools-pm。

### 9.2 PM 已拍板的未定义边界（**已裁决，下游不得另行取舍**）

Stage 2 评审（FINN-58）提出的 3 条未定义边界，PM 已拍板，**本方案按以下口径执行**：

| # | 边界 | PM 拍板口径 | 落在本文哪 |
|---|---|---|---|
| **U-1** | `args` 为空 **且** `command` 不是 `uvx`（如 `command: npx` + 无 args）时，归一化会塞进 minimax 默认调用 | **保持现状、不在本 issue 范围**。归一化在 args 为空时塞进 minimax 默认调用是**既有行为**（原 `len(args)==0` 分支同款），本次**不改**；**不要**为此另开分支逻辑。§4 兼容表已按此改写（N-5） | §4「args 为空、且 `command` 不是 `uvx`」行 / §3.2 规则 1 |
| **U-2** | `uvx --from minimax-coding-plan-mcp` / `--with <pkg>`：包名藏在 flag 的值里，门控匹配不到 → 永久漏兜底 | **纳入门控**（与 N-4 同批处理）：门控除「args 里直接出现包名元素」外，还要识别 **`--from` / `--with` 之后紧跟的包名元素**命中 minimax 包名 → 视为 minimax 调用 → 注入 `-y` | §3.3 门控补充说明第 2 条 / §3.2 `indexOfMinimaxMCPPackage` / §6 L2 T13、T14 |
| **U-3** | 包名带 PEP 440 版本约束 / extras 的门控匹配口径 | 按 N-4 放宽：元素 == 包名，或以 `包名` + ∈{`=`,`@`,`>`,`<`,`~`,`!`} 开头即算命中；**禁止**无脑 `HasPrefix(包名)`（会误命中 `minimax-coding-plan-mcp-fork` 这类同名第三方包） | §3.3 门控补充说明第 1 条 / §3.2 `isMinimaxMCPPackageSpec` / §6 L2 T6、T15 |

---

## 10. 参考依据

- 仓库内（**第一依据**）：`entrypoint.sh:55`、`Dockerfile:88`、`backend/handlers/image_understanding.go:699`（三处 `-y` 基准）
- MiniMax MCP 官方安装 / 客户端配置示例（`uvx minimax-coding-plan-mcp -y`；Claude Desktop / Claude Code CLI / OpenCode 的 MCP 配置块）：
  - https://skillsllm.com/skill/minimax-coding-plan-mcp
  - https://www.mcpworld.com/zh/detail/031076356b5a026f0788fac8435c45d7
  - https://www.codefather.cn/post/2022508136270663682
  - https://skillfed.io/packages/minimax-coding-plan-mcp
- uv `tool run` 的 flag 位置约定（uv 自身 flag 在包名**之前**，包名之后的参数转发给被调程序）：
  - https://learnlinux.appsved.com/man/uvx.html
  - https://www.pattern.institute/uv/reference/uvx.html
  - https://instagit.com/astral-sh/uv/uv-tool-run-install-commands/

> 引用权威性说明：MiniMax / uv 侧链接为**第三方站点**（非官方文档），但**结论不依赖它们** —— 仓库内 `entrypoint.sh:55` + `Dockerfile:88` 已构成权威对齐基准，且 §3.3 B1 的位置化规则**不依赖 uv 的具体行为**（无论 uv 是「吃掉包名之前的 `-y`」还是「报错拒绝」，包层 `-y` 都没就位 → 补 `-y` 都正确）。

---

## 11. 交付说明（给 Stage 2 / Stage 3）

- 本方案**不含代码改动**，Stage 2 按 §3.1 / §3.2 实施即可。
- Stage 2 若对「B4 门控」有异议（认为应盲注入），**必须**先回到本 issue 讨论 —— 盲注入会破坏用户自定义 MCP Server 调用（§5 K2）。
- Stage 2 若对「B1 位置化」有异议（认为包名之前的 `-y` 也算已含），**必须**先回到本 issue 讨论 —— 那样会让存量 uv 短形式写法**永久漏兜底**（§2.5）。
- Stage 3 交付时必须在完成评论里**原文报告分支名与 commit hash**（父 issue 状态纪律）。
- 本方案文档位于 `docs/plans/minimax-mcp-args-alignment.md`，随 Stage 6 一并合入 `main`（注意：Stage 6 合并范围历来只圈代码分支，**本文件需被显式纳入**）。

---

## 12. 修订记录

### v2（FINN-63，本次修订；基线 `d795596`）

| 来源 | 条目 | 落点 |
|---|---|---|
| FINN-58 **B-1**（blocking） | `-y` 判定改为**位置化**（以包名元素下标为基准） | §2.5 新增推论段、§3.2 规则 3/4 + 实现约束「位置判定硬约束」+ `indexOfMinimaxMCPPackage` 草图、§3.3 B1 重写 / B2 降为示例 / B5 措辞、§6 L2 T5 改写 + T11–T15 新增 + T16 补充断言、§7 新增「位置约束」 |
| FINN-58 **B-2**（blocking） | L1 静态判据改工作树口径；`args:` 用 2 空格缩进模式 | 头部「行号口径」、§6 L1 基线口径纪律、L1-1、L1-2 |
| N-1 | 行号口径声明（改动点 = origin/main；§2 取证 = architect 树，+9） | 头部「行号口径」、§2.2 / §2.4 / §2.6 / §3.4 标注双口径 |
| N-2 | §0「未决 2 条」与 §9「3 条」统一 | §0 末行、§9.1 标题与结语 |
| N-3 | §0 命中点计数改为 7（3 缺 + 4 已正确） | §0 第 1 行、§6 L1-3 注释（「无第 5 处遗漏」→「无第 8 处遗漏」） |
| N-4 | 门控放宽（包名 + ∈{`=`,`@`,`>`,`<`,`~`,`!`}；禁无脑 HasPrefix） | §3.2 `isMinimaxMCPPackageSpec`、§3.3 门控补充说明第 1 条、§5 K7、§6 L2 T6/T15 |
| N-5 | §4 兼容表按「args 非空」口径改写 | §4「自定义 MCP Server」行 + 新增「args 为空」行 |
| N-6 | 护栏 3 路径写 `../config.example.yaml`（相对 `backend/handlers/`）+ `t.Setenv("CONFIG_PATH", …)` 范式 | §6 L2 护栏 2 / 护栏 3 |
| N-9 | 常量命名类比措辞修正 | §3.2 实现约束、§8 S8 行 |
| U-1 / U-2 / U-3 | PM 拍板的三条边界落文档 | §9.2（新增小节）、§3.2 规则 1、§3.3 门控补充说明第 2 条、§4、§6 L2 T13/T14 |

> **本次修订只改本文件**（纯文档，1 file changed），不涉及任何代码文件、不涉及编译单元（R11 不适用）。
