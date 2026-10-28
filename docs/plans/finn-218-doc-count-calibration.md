# FINN-218 · AGENTS.md / CLAUDE.md 计数校准清单 + 可持续机制

> 产出者：devtools-architect ｜ 单号：FINN-218（Stage 1，父单 FINN-217）
> 写入边界：本文件（`docs/plans/`）为**唯一**产物；本文档**不含任何**对 `AGENTS.md` / `CLAUDE.md` / 业务代码的实际改动。
> 测量基线：`origin/main` = `e8aef82b5cd5d3ae2e123f363a56b276ece03596`（2026-10-28 现读 `git ls-remote origin refs/heads/main`）
> ⚠️ 下游（Stage 3 实施方）**不得誊抄本文档里的数字**，必须按 §4.1 的命令在**当时的 main tip** 上现算现填。

---

## 1. 目标与非目标

### 1.1 目标

1. 把 `AGENTS.md` + `CLAUDE.md` 中**全部**计数型断言逐条对齐到 `origin/main` 实测值（不止父单列的 11 条 —— 实测出 **26 处必改 + 1 处可选**，见 §3）。
2. 为每条计数**定义可复跑的判据命令**，让后来人现读现算，不再靠"手工重数一遍"。
3. 给出「新增模块 / 新增 View / 新增 routes 文件后，哪几条计数必须同步重算」的**检查清单**（落在 `CLAUDE.md`，父单已拍板）。
4. 明确**哪些计数不该进文档**，避免为了"全"而引入新的易腐断言。

### 1.2 非目标

- ⛔ 不改任何 Go / Vue / JS 代码（本单不涉及，`go build` / `vitest` / `pnpm build` 均不需要）。
- ⛔ 不改 `AGENTS.md` / `CLAUDE.md`（**本单只出方案**，实施是父单 Stage 3 的事）。
- ⛔ 不改 `docs/INDEX.md`、不新增 `scripts/` 下的脚本文件（父单硬约束 #1：只许改 `AGENTS.md` 与 `CLAUDE.md` 两个文件 ⇒ 判据命令必须以**可粘贴的 shell 片段**形态内嵌，不能落成脚本文件）。
- ⛔ 不部署、不跑 `go run` / dev server / `docker compose up` / `./deploy.sh`、不碰部署主机 `192.168.31.201`（R1）。
- ⛔ 不在本文档里**新增**任何"恒真闸门"式断言（父单硬约束 #4）。

---

## 2. 现状（已核实）

### 2.1 根因：口径从来没有被写下来过 —— 但**可以**被反推出来

关键证据：上一次计数校准提交 **`ce51f61`（2026-10-22，sim，"docs: 修正 CLAUDE.md / AGENTS.md 的模块与文件数量漂移"）**。它的 commit message 自述了当时的测量口径：

```
实际计数：views/*.vue 86（+components 12 共 98）、routes/*.go 40、handlers/*.go 118、
AGENTS.md 模块表 42 行。
```

**实测复现（在该 commit 上跑同样的命令）**：

```
$ git ls-tree -r --name-only ce51f61 -- frontend/src/views/ | grep -c '\.vue$'     → 86
$ git ls-tree -r --name-only ce51f61 -- frontend/src/ | grep '\.vue$' | grep -vc '/views/'  → 12
$ git ls-tree -r --name-only ce51f61 -- frontend/src/ | grep -c '\.vue$'             → 98   ← 86 + 12 = 98
$ git ls-tree -r --name-only ce51f61 -- backend/routes/  | grep -c '\.go$'          → 40
$ git ls-tree -r --name-only ce51f61 -- backend/handlers/ | grep -c '\.go$'         → 118
$ git show ce51f61:AGENTS.md | grep -cE '^\| *[0-9]+ *\|'                          → 42
```

⇒ 结论：**三条口径原样延续至今**（views 递归 `.vue` / routes 目录全部 `.go` / handlers 目录全部 `.go`），**只有"12 个公共组件"的口径当时就没写下来** —— 但由 `86 + 12 = 98 = frontend/src 下 .vue 总数` 这个恒等式可以**唯一反推**出来（见 §3.2）。

### 2.2 本单实测（`origin/main` @ `e8aef82`）

```
模块表行数       : 43
View 组件        : 88
公共组件         : 12
routes 文件      : 43      （非测试 41 含 index.go + 测试 2）
routes 分组      : 40      （两种独立算法互证，见 §3.3 #12）
handlers 文件    : 130     （非测试 78 + 测试 52）
ai_gateway 非测试: 13      （其中 ai_gateway_test_helpers.go = 1 个测试脚手架）
内置 skill 数     : 7
--- 各 category View 组件 ---
ai 7 / collab 6 / convert 7 / dev 22 / draw 4 / home 1 / household 10 / life 13 / other 10 / share 8
```

### 2.3 文档里"routes/ 文件数"有**三个互不相同**的写法

| 位置 | 现值 | 实际指向 |
|---|---|---|
| `AGENTS.md:113` | `42 个文件` | 目录总数口径 |
| `AGENTS.md:209` | `42 文件` | 目录总数口径 |
| `CLAUDE.md:52` | `38 文件` | **AGENTS.md 里那棵树实际枚举的文件数**（实测枚举 = 38：index.go + 37 个路由文件） |

⇒ "38" 不是漂移，是**另一套口径**（树里手写的清单）被同一个词"文件"混用了。父单 11 条清单**漏掉了 `CLAUDE.md:52` 与 `CLAUDE.md:37` 这两条口径冲突**。

### 2.4 "N 文件"列（模块表 Handler 文件列）**混用了两套口径**

| 模块 | 文档现值 | 非测试口径 | 含测试口径 | 文档用的是哪套 |
|---|---|---|---|---|
| 20 Planner `planner*.go` | 6 | **6** | 7 | 非测试 ✅ |
| 23 NFSShare `nfsshare*.go` | 5 | **4** | 5 | 含测试 ❌ |
| 26 Askit `askit*.go` | 3 | **3** | 3 | 两套都成立 |
| 37 Skills `skills*.go` | 3 | **1** | 5 | 都不成立 ❌（历史遗留） |
| 25 AIGateway `ai_gateway*.go` | 12「含 1 个测试脚手架」 | **13** | 22 | 都不成立 ❌ |

⇒ 同一个列表格里 5 个数用了 3 套口径 —— 这是"手工誊抄"必然腐烂的机制性原因。

### 2.5 `AGENTS.md` 模块表**收录不全**（本单**不修**，仅登记）

模块表 43 行，但仓库里存在**未被收录**的完整模块：

| 缺失模块 | 证据（均在 `origin/main`） |
|---|---|
| **News 每日新闻** | `backend/handlers/news.go`（726 行）、`backend/routes/news.go`、`backend/models/news.go`、`frontend/src/views/life/NewsTool.vue`、`frontend/src/router/index.js:434` 有路由项、`backend/routes/index.go:95` 有 `RegisterNewsRoutes` |
| **Chat+ 多模型对话** | `backend/handlers/chat_plus.go`、`backend/handlers/ai_chat_verify.go`、`frontend/src/views/ai/AIChatTool.vue`、`frontend/src/router/index.js:478` |
| **Trip 前端** | `frontend/src/views/life/TripTool.vue` + `frontend/src/router/index.js:356`；但模块 41 写的是 `— (CLI-only)` / View `—` |

⇒ 覆盖度实测：模块表 View 列只覆盖 88 个 View 中的 **38 个**（其余 50 个是子组件、share 页、或上述缺失模块）。

---

## 3. 设计

### 3.0 口径总表（**每条计数只有这一套算法**）

| 口径码 | 名称 | 定义 | 判据命令（`REF=origin/main`） |
|---|---|---|---|
| **C1** | 模块表行数 | `AGENTS.md` 模块表的数据行数 | `git show ${REF}:AGENTS.md \| grep -cE '^\| *[0-9]+ *\|'` |
| **C2** | View 组件数 | `frontend/src/views/` 下**递归**全部 `.vue` | `git ls-tree -r --name-only ${REF} -- frontend/src/views/ \| grep -c '\.vue$'` |
| **C3** | 公共组件数 | `frontend/src/` 下**除 `views/` 外**的全部 `.vue`（= `components/` 11 个 + `App.vue` 1 个） | `git ls-tree -r --name-only ${REF} -- frontend/src/ \| grep '\.vue$' \| grep -vc '/views/'` |
| **C4** | 目录级 View 数 | 某个 `views/<category>/` 下**递归**全部 `.vue`（含 `VirtualAvatarTool/` 这类子目录） | `git ls-tree -r --name-only ${REF} -- frontend/src/views/<cat> \| grep -c '\.vue$'` |
| **C5** | routes 文件数 | `backend/routes/` 下**全部** `.go`（含 `index.go` 与 `*_test.go`） | `git ls-tree -r --name-only ${REF} -- backend/routes/ \| grep -c '\.go$'` |
| **C6** | routes 分组数 | 每个**非测试、非 `index.go`** 的路由文件恰好定义 1 个 `Register*Routes` ⇒ 组数 = 这样的文件数 | `git ls-tree --name-only ${REF}:backend/routes/ \| grep '\.go$' \| grep -v '_test\.go$' \| grep -vc '^index\.go$'` |
| **C7** | handlers 文件数 | `backend/handlers/` 下**全部** `.go`（含测试） | `git ls-tree -r --name-only ${REF} -- backend/handlers/ \| grep -c '\.go$'` |
| **C8** | 前缀 handler 文件数 | 模块表 `handlers/<prefix>*.go (N 文件)` 的 N = **排除 `_test.go`** 的匹配文件数 | `git ls-tree --name-only ${REF}:backend/handlers/ \| grep '^<prefix>' \| grep -vc '_test\.go$'` |
| **C9** | 内置 skill 数 | `defaultSkillRegistry()` 返回的条目数 | `git show ${REF}:backend/handlers/skills.go \| grep -cE '^[[:space:]]*Name:[[:space:]]+"'` |
| **C10** | handler 分段数 | 文件内 `// <domain>/sectionN:` 横幅标签的**去重**个数 **+ 1**（`section1` 为横幅前的隐式首段） | `git show ${REF}:backend/handlers/<f>.go \| grep -oE '<domain>/section[0-9]+' \| sort -u \| wc -l`（结果 +1） |
| **C11** | 子组件数 | 模块 View 所在子树 `.vue` 数 − 1（减去主组件本身） | `git ls-tree -r --name-only ${REF} -- frontend/src/views/dev/VirtualAvatarTool \| grep -c '\.vue$'`（14 − 1 = 13） |

**C8 为什么选"排除 `_test.go`"**：只有这一套能让 `AGENTS.md:35` 现有的措辞 `（N 文件，含 1 个测试脚手架）` 保持成立 —— `ai_gateway_test_helpers.go` 不是 `_test.go`，它就落在"非测试"这一侧，正好被"含 1 个测试脚手架"括住。若改用"含测试"口径（22），括注就变成假话。
**C10 为什么 +1**：三个文件的第一段（imports/类型定义）没有 `sectionN` 标签，最小标签是 `section2`（实测 `household/autodev/proxy` 的横幅标签分别从 2 起）。实测：`household` 标签 6 → 7 段；`autodev` 5 → 6 段；`proxy` 3 → 4 段 —— 与文档现值**全部一致**。

### 3.1 逐条校准表（**24 条要改值 / 改口径，7 条保持但要写清口径**）

> "现值" 列 = `origin/main` 上 `AGENTS.md` / `CLAUDE.md` 的**逐字**内容；"实测" 列 = 按 §3.0 口径在 `e8aef82` 上现算的值。
> **Stage 3 实施时必须用 §4.1 的命令在当时的 main tip 上重算，把下表"实测"列整列替换掉再填文档。**

#### A. `AGENTS.md`

| # | 位置 | 现值 | 实测 | 口径 | 目标写法（示例，基于 `e8aef82`） |
|---|---|---|---|---|---|
| A1 | `AGENTS.md:7` | `## 42 模块地图` | 43 | C1 | `## 43 模块地图` |
| A2 | `AGENTS.md:59` | `86 个 View 组件 + 12 个公共组件` | 88 / **12** | C2 / C3 | `88 个 View 组件 + 12 个公共组件` |
| A3 | `AGENTS.md:98` | `### 前端（86 View 组件）` | 88 | C2 | `### 前端（88 View 组件）` |
| A4 | `AGENTS.md:101` | `ai/ # AI 相关（6）` | 7 | C4 | `ai/ # AI 相关（7）` |
| A5 | `AGENTS.md:102` | `collab/ # 协作（8）` | 6 | C4 | `collab/ # 协作（6）` |
| A6 | `AGENTS.md:103` | `convert/ # 转换（8）` | 7 | C4 | `convert/ # 转换（7）` |
| A7 | `AGENTS.md:104` | `dev/ # 开发（10）` | 22 | C4 | `dev/ # 开发（22）` ⚠️ 见 §3.4-P3 |
| A8 | `AGENTS.md:105` | `draw/ # 绘图（4）` | 4 | C4 | **不改** |
| A9 | `AGENTS.md:107` | `household/ # 家庭（9 个子组件）` | 10 | C4 | `household/ # 家庭（10 个子组件）` |
| A10 | `AGENTS.md:108` | `life/ # 生活（11）` | 13 | C4 | `life/ # 生活（13）` |
| A11 | `AGENTS.md:109` | `other/ # 其他（9）` | 10 | C4 | `other/ # 其他（10）` |
| A12 | `AGENTS.md:110` | `share/ # 分享视图（9）` | 8 | C4 | `share/ # 分享视图（8）` |
| A13 | `AGENTS.md:113` | `### 后端路由拆分（routes/ 42 个文件）` | 43 | C5 | `### 后端路由拆分（routes/ 43 个文件）` |
| A14 | `AGENTS.md:130` | `AI 网关（18）` | 19 | C8 汇总 | `AI 网关（19）` |
| A15 | `AGENTS.md:130` | `ai_gateway*（12，含 1 个 test helper）` | 13 | C8 | `ai_gateway*（13，含 1 个 test helper）` |
| A16 | `AGENTS.md:130` | `minimax_*（4）` | 4 | C8 | **不改**（口径已在括号里写明"文件"，随 C8 统一） |
| A17 | `AGENTS.md:30` | `handlers/planner*.go (6 文件)` | 6 | C8 | **不改** |
| A18 | `AGENTS.md:33` | `handlers/nfsshare*.go (5 文件)` | 4 | C8 | `handlers/nfsshare*.go (4 文件)` |
| A19 | `AGENTS.md:36` | `handlers/askit*.go (3 文件)` | 3 | C8 | **不改** |
| A20 | `AGENTS.md:47` | `handlers/skills*.go (3 文件)` | 1 | C8 | `handlers/skills*.go (1 文件)` |
| A21 | `AGENTS.md:47` | `5 个 skill` | 7 | C9 | `7 个 skill` |
| A22 | `AGENTS.md:209` | `routes/ 包（42 文件，34 分组）` | 43 / 40 | C5 / C6 | `routes/ 包（43 文件，40 分组）` |
| A23 | `AGENTS.md:223` | `更新本文件 \`42 模块地图\` 表格` | 43 | C1 | `更新本文件 \`43 模块地图\` 表格` |
| A24 | `AGENTS.md:137` | `测试（20+）` | 52 | C7(测试子集) | `测试（52）` ⚠️ **可选**，见 §3.4-P5 |
| A25 | `AGENTS.md:50` | `(+ 13 子组件)` | 13 | C11 | **不改** |
| A26 | `AGENTS.md:32/39/41` | `(7 分段)` / `(6 分段)` / `(4 分段)` | 7 / 6 / 4 | C10 | **不改** |
| A27 | `AGENTS.md:67` | `Docker Compose（4 服务）` | 4 | compose `services:` | **不改** |

#### B. `CLAUDE.md`

| # | 位置 | 现值 | 实测 | 口径 | 目标写法（示例，基于 `e8aef82`） |
|---|---|---|---|---|---|
| B1 | `CLAUDE.md:3` | `（42 模块地图、…）` | 43 | C1 | `（43 模块地图、…）` |
| B2 | `CLAUDE.md:9` | `86 个 View 组件 + 12 个公共组件` | 88 / **12** | C2 / C3 | `88 个 View 组件 + 12 个公共组件` |
| B3 | `CLAUDE.md:34` | `# 86 Vue 组件` | 88 | C2 | `# 88 Vue 组件` |
| B4 | `CLAUDE.md:37` | `# 40 个路由文件（routes.go 已拆分，含 index.go）` | 43 | C5 | `# 43 个路由文件（routes.go 已拆分，含 index.go）` |
| B5 | `CLAUDE.md:39` | `# 118 个 handler 文件（按域分组）` | 130 | C7 | `# 130 个 handler 文件（按域分组）` |
| B6 | `CLAUDE.md:52` | `routes/ 包（38 文件，34 分组）` | 43 / 40 | C5 / C6 | `routes/ 包（43 文件，40 分组）` ⚠️ 与 `AGENTS.md:209` 统一 |
| B7 | `CLAUDE.md:67` | `6. 更新 \`AGENTS.md\` 42 模块地图` | 43 | C1 | `6. 更新 \`AGENTS.md\` 43 模块地图` |
| B8 | `CLAUDE.md:12` | `Docker Compose（4 服务）` | 4 | compose `services:` | **不改** |

**合计需要改动：26 处必改**（A 表 19 处：A1–A7、A9–A15、A18、A20–A23；B 表 7 处：B1–B7），**另加 1 处可选**（A24 `测试（20+）`）＝ 27 处。父单清单只列了 11 条 ⇒ **父单漏了 15 处以上**（漏项清单见 §3.5）。

### 3.2 PM 未定义边界之一：「12 个公共组件」到底数什么

**结论（口径 C3）**：`frontend/src/` 下**除 `views/` 之外**的全部 `.vue` 组件。
- 当前 = `frontend/src/components/` 的 **11 个 `.vue`** + `frontend/src/App.vue` **1 个** = **12**。
- 判据命令：
  ```bash
  git ls-tree -r --name-only origin/main -- frontend/src/ | grep '\.vue$' | grep -vc '/views/'
  ```
- **这条数不用改**（12 → 12）。理由（两条硬证据）：
  1. **可反推**：上一次校准 commit `ce51f61` 自述 `views/*.vue 86（+components 12 共 98）`，而当时 `frontend/src` 下 `.vue` 总数**实测 = 98**（86 views + 11 components + 1 App.vue）。98 = 86 + 12 ⇒ 当时的 12 **必然**是 `components/ 的 11 + App.vue 的 1`，否则凑不出 98。
  2. **划分闭合**：今天 `88 + 12 = 100` = `frontend/src` 下 `.vue` 总数（实测 100）。C2 + C3 是**前端全部组件的一个无重叠划分**，不是两个各数各的数。

- **替代口径（选项 B，不推荐）**：`frontend/src/components/` 目录下的 `.vue` = **11**。
  - 后果：① `88 + 11 = 99 ≠ 100`，`App.vue` 无归属，划分不再闭合；② 12 → 11 变成一次"改值"，父单硬约束 #4 的反事实自检会更好过 —— **但这是"为了过闸门而改口径"**，会把一条本来正确的断言改错。
  - 我**倾向选项 A（C3，值不变）**。若 PM 要求"每条都必须变值"，则走选项 B 并在文档里把 `App.vue` 单列说明。
- **明确排除**：`components/` 下的 2 个 `*.test.js`（`AIGatewaySpeechPanel.share.test.js`、`MusicLyricsPlayer.test.js`）**不是组件**，任何口径都不该计入；`frontend/src/App.vue` 是应用根组件，不属于 `views/`。
- ⚠️ **Stage 5 的反事实判据要按口径写**：这一条的判据是「命令在 main 上算出 12 == 文档写的 12」，**不是**「改前值 ≠ 改后值」。

### 3.3 PM 未定义边界之二：模块表行数是否调整

**结论：本单不增删模块行，标题按行数校准为 43。**

理由：
1. **父单硬约束 #1**：「只许改 `AGENTS.md` 与 `CLAUDE.md`，且只许改**计数与编号**」；新增模块行必然要写「主要功能」描述（**内容列**），明确越界 ⇒ 新增行不在本单可做范围。
2. **行数本身就是易腐数**：每加一个模块就要同时改表格**行号**、**行数**、**标题数字** 三处 ⇒ 新增行会把漂移面从 2 处扩到 3 处，且标题数字每次都要人肉记得改。
3. **口径要钉死**：标题里的 N = **表格行数**（C1），**不是**"项目的模块总数"。后者根本数不清（§2.5：News / Chat+ / Trip 前端都未收录）。把 C1 写进 `CLAUDE.md` 的清单，就消除了"到底几个模块"的歧义。

**遗留缺口（不在本单修，必须显式登记）**：
- 模块表缺 **News**、**Chat+** 两个完整模块行；模块 41 Trip 的 View 列写 `—` 但实际有 `frontend/src/views/life/TripTool.vue` + 路由项。
- ⇒ **建议 PM 另开一个单**补这些行（需要写内容列，超出本单边界）。本单**不**动表格内容。

**`## 43 模块地图` 这个写法本身仍是易腐点**（下次加模块又是 44）。缓解手段 = §4.2 的同步清单里第一条就是「模块表行数变了 ⇒ 标题数字同步」。**备选写法（选项 B）**：标题写成 `## 模块地图`（完全去数字，永不漂移）—— 但父单硬约束 #2 字面要求"改成与行数一致的写法"，故**推荐选项 A（保留数字）**，把选项 B 交给 PM 裁决（§3.4-P1）。

### 3.4 需 PM 裁决的点（父单已拍板的 3 条不再列）

| # | 议题 | 可选 | 我的倾向 | 不裁决的后果 |
|---|---|---|---|---|
| **P1** | 模块标题是否保留数字 | A. `## 43 模块地图`（符合父单 #2 字面）｜B. `## 模块地图`（永不漂移，但失去规模感、且与父单 #2 字面冲突） | **A** | Stage 3 卡住（写哪个都可能被 Stage 2 打回） |
| **P2** | 「12 个公共组件」口径 | A. C3（`frontend/src` 非 `views/` 的 `.vue` = 12，值不变）｜B. `components/` 目录 `.vue` = 11（值变化但划分不闭合） | **A** | Stage 3 写 11 还是 12 无法判定；且父单 #4 的反事实要求会逼出错误口径 |
| **P3** | `dev/` 目录计数怎么写 | A. `dev/ # 开发（22）`（纯计数）｜B. `dev/ # 开发（22，含 VirtualAvatarTool/ 14 个）`（消歧） | **B**（一句话说明子目录归属；B 是纯计数括注的补充，不改语义） | 读者会以为 22 与括号里 9 个名字对不上 |
| **P4** | per-category 名字清单不同步（`ai` 少 RolePlayTool、`dev` 只写 VirtualAvatar、`life` 少 NewsTool/TripTool、`other` 的 "Neon" 不存在） | A. 本单只改数字，名字清单另开单刷新 ｜B. 本单一并刷新名字清单（**越界风险**：父单 #1「只许改计数与编号」） | **A**（本单只改数字；名字清单属内容，另开单） | Stage 3 越权改动 = Stage 4 判 P0 |
| **P5** | `测试（20+）` 改不改 | A. 改精确值 `测试（52）`｜B. 保留 `20+` | **A**（模糊断言恒真、不可能被反事实证伪；且已低估 2.6 倍） | Stage 5 反事实对 `20+` 无法判定（永远成立） |
| **P6** | handler 分组计数（核心工具 8 / 订阅代理 5 / 家庭健康 5 / 实时 4 / 内容创作 3 / 开发者 3 / 辅助 7 / 测试 20+） | A. 本单不动 ｜B. 删掉所有括号数只留枚举 | **A**（本单不动；这些数没有单条判据、且与同行名字对不上，例如"辅助（7）"却列了 9 个名字，其中 `security_enhance.go`/`content_parser.go` 实际在 `backend/utils/` 下不在 `backend/handlers/`） | 留着就还会漂；改成 B 属于"改措辞"，越界 |

> P6 的深层问题（分组括号数本身不该进文档）已在 §4.3 登记为**建议另开单**。

### 3.5 父单 11 条清单的漏项（本单实测补齐）

| 父单已列 | 本单补充 |
|---|---|
| AGENTS.md:7 / CLAUDE.md:3 / CLAUDE.md:67（42→43） | ✅ 父单已列 |
| AGENTS.md:59 / AGENTS.md:98（86→88） | ✅ + **CLAUDE.md:9**、**CLAUDE.md:34**（同样写 86） |
| AGENTS.md:101（ai 6→7） | ✅ + **collab / convert / dev / household / life / other / share 共 9 个 category 全部漂移** |
| AGENTS.md:113 / AGENTS.md:209（routes 42→43） | ✅ + **CLAUDE.md:37（40）**、**CLAUDE.md:52（38）**、**「34 分组」→ 40** |
| AGENTS.md:35 / AGENTS.md:130（ai_gateway 12） | ✅ + **AI 网关分组总数 18→19**、**minimax_* 口径**、**nfsshare 5→4**、**skills 3→1**、**`5 个 skill`→7** |
| CLAUDE.md:9（12 个公共组件） | ✅ + **CLAUDE.md:39（118→130 handler 文件）** —— 父单完全没提 |

⇒ 父单 11 条 → 本单 **26 处待改**（+1 可选）。

### 3.6 文档改动清单（Stage 3 逐处指令）

**改动文件：恰好 2 个**（父单硬约束 #1）
- `AGENTS.md`：§3.1 表 A 中标记"要改"的 **19 处**（A1–A7、A9–A15、A18、A20–A23）+ 可选 A24；§4.2 同步清单**不写入** AGENTS.md（清单只进 `CLAUDE.md`）。
- `CLAUDE.md`：§3.1 表 B 中 7 处 + **新增** §4.2 的「计数同步清单」小节（插在「添加新工具」第 6 步之后，父单已拍板）。

**改动性质约束**：只允许动**数字**与**编号**；⛔ 不得改动任何命令、纪律、R1–R11 语义、模块表内容列、模块顺序、模块行数、章节标题文字（除标题里的数字）、代码块里的命令文本。

**新增到 `CLAUDE.md` 的同步清单（草案，最终措辞由 Stage 3 定，语义不得变）**：

````markdown
### 计数同步清单（改动前后各跑一次，按输出填文档）

```bash
REF=${REF:-origin/main}   # 校准文档时用 origin/main；改自己的分支时用 HEAD
echo "模块表行数       : $(git show ${REF}:AGENTS.md | grep -cE '^\| *[0-9]+ *\|')"
echo "View 组件        : $(git ls-tree -r --name-only ${REF} -- frontend/src/views/ | grep -c '\.vue$')"
echo "公共组件         : $(git ls-tree -r --name-only ${REF} -- frontend/src/ | grep '\.vue$' | grep -vc '/views/')"
echo "routes 文件      : $(git ls-tree -r --name-only ${REF} -- backend/routes/ | grep -c '\.go$')"
echo "routes 分组      : $(git ls-tree --name-only ${REF}:backend/routes/ | grep '\.go$' | grep -v '_test\.go$' | grep -vc '^index\.go$')"
echo "handlers 文件    : $(git ls-tree -r --name-only ${REF} -- backend/handlers/ | grep -c '\.go$')"
echo "ai_gateway 非测试: $(git ls-tree --name-only ${REF}:backend/handlers/ | grep '^ai_gateway' | grep -vc '_test\.go$')"
echo "内置 skill 数     : $(git show ${REF}:backend/handlers/skills.go | grep -cE '^[[:space:]]*Name:[[:space:]]+\"')"
for d in ai collab convert dev draw home household life other share; do
  printf '%-10s %s\n' "$d" "$(git ls-tree -r --name-only ${REF} -- frontend/src/views/$d | grep -c '\.vue$')"
done
```

口径速查（括号里的数 = 上面的输出，**不要手数**）：
`View 组件` = `frontend/src/views/` 递归 `.vue`；`公共组件` = `frontend/src/` 下除 `views/` 外的 `.vue`（含 `App.vue`）；
`routes 文件` = `backend/routes/` 全部 `.go`；`routes 分组` = 非测试非 `index.go` 的路由文件数（每文件 1 个 `Register*Routes`）；
`handlers 文件` = `backend/handlers/` 全部 `.go`；`ai_gateway 非测试` = `ai_gateway*` 去掉 `_test.go`（含 1 个 `ai_gateway_test_helpers.go` 脚手架）；
模块表 `handlers/xxx*.go (N 文件)` 的 N 一律**排除 `_test.go`**。
````

> ⚠️ 该片段在本仓库已**实测跑通**（输出见 §2.2），注意 `${REF}:path` 必须带花括号（zsh 下 `$REF:path` 会被 `:A` 修饰符吃掉）。

---

## 4. 可持续机制（本单的核心价值）

### 4.1 单一判据命令块

见 §3.6 的代码块 —— **只有这一处**是权威命令来源，其余文档数字都由它导出。Stage 3 必须把它放进 `CLAUDE.md`（父单已拍板），使每个 agent 开工即见。

### 4.2 新增模块 / 新增 View / 新增 routes 文件后的同步清单

**触发条件 → 必须重算的条目**（下表右列即 `AGENTS.md` / `CLAUDE.md` 里要跟着改的位置）：

| 变更动作 | 必须重算并同步的文档条目 |
|---|---|
| **新增一个 View 组件**（`frontend/src/views/**` 下新增 `.vue`） | ① `AGENTS.md` 该 category 行的括号数（C4）→ ② `AGENTS.md:98` / `CLAUDE.md:34` View 总数（C2）→ ③ `AGENTS.md:59` / `CLAUDE.md:9` View 总数（C2） |
| **新增一个公共组件**（`frontend/src/` 非 `views/` 下新增 `.vue`） | `AGENTS.md:59` / `CLAUDE.md:9` 的「N 个公共组件」（C3） |
| **新增一个 routes 文件** | ① `AGENTS.md:113`（C5）→ ② `AGENTS.md:209` + `CLAUDE.md:52`（C5 + C6）→ ③ `CLAUDE.md:37`（C5） |
| **新增一个 handler 文件**（含测试） | `CLAUDE.md:39`（C7） |
| **新增/删除 `ai_gateway*` / `minimax_*` / `planner*` / `nfsshare*` / `askit*` / `skills*` handler** | 模块表对应行的 `(N 文件)`（C8）+ `AGENTS.md:130` 的「AI 网关（N）」（C8 汇总） |
| **新增/删除内置 skill** | `AGENTS.md:47` 的「N 个 skill」（C9） |
| **新增一个模块**（模块表加一行） | ① 模块表新增行（编号连续）→ ② `AGENTS.md:7` 标题数字（C1）→ ③ `AGENTS.md:223` / `CLAUDE.md:3` / `CLAUDE.md:67` 的「N 模块地图」（C1） |
| **新增 handler 分段**（改 `// domain/sectionN` 横幅） | 模块表对应行的 `(N 分段)`（C10） |

**每次改完必跑**（30 秒内可完成，防"改一半"）：

```bash
# 1) 现算
bash <粘贴 §4.1 片段>
# 2) 对照文档里的 12 个数字（C1/C2/C3/C5/C6/C7/C8/C9 + 9 个 category）
# 3) 一致性硬闸门：标题数字 == 模块表行数
grep -E '^## [0-9]+ 模块地图' AGENTS.md
git show ${REF}:AGENTS.md | grep -cE '^\| *[0-9]+ *\|'   # 两个数必须相等
```

### 4.3 哪些计数**不该**进文档（避免引入新的易腐断言）

| 不该进文档 | 理由 |
|---|---|
| **handler 分组的括号数**（核心工具 8 / 订阅代理 5 / 家庭健康 5 / 实时 4 / 内容创作 3 / 开发者 3 / 辅助 7） | 同行括号数与列出的名字**对不上**（"辅助（7）"却列 9 个名字），且 `proxy*`/`skills*`/`askit*` 这类 glob 展开数 ≠ 括号数；**没有单条判据命令** ⇒ 无法反事实证伪。建议另开单：要么把括号数删掉只留枚举，要么逐组定义判据。 |
| **`frontend/src` 的 `.vue` 总数**（今天 100） | 它是 C2+C3 的派生量；写进文档就是第二个易腐数（且必然与前两个不同步）。 |
| **每个 category 的文件名全清单** | 源码里本来就有；文档重复一份必然漂移 —— 本次实测已证明：`ai` 少 `RolePlayTool`、`life` 少 `NewsTool`/`TripTool`、`other` 的 "Neon" 根本不是文件（是 `frontend/src/neon/` 目录）。 |
| **API 端点总数 / 模型表数 / Redis key 数 / 路由注册函数总数** | 与本单口径体系无关，且随每次迭代波动，没有稳定判据。 |
| **"20+" 这类模糊计数** | 恒真 ⇒ 反事实自检永远过不了。已在 §3.4-P5 建议改成精确值。 |
| **模块总数（= 业务上"有几个模块"）** | 模块表**收录不全**（§2.5 缺 News / Chat+ / Trip 前端），数不清。文档里的 N 只应表示**表行数**（C1）。 |

---

## 5. 兼容性与迁移

- **纯文档改动，零运行时影响**：不改任何代码、schema、配置、路由 ⇒ 无数据迁移、无 API 变更、无客户端兼容问题。
- **无后端/前端代码改动清单**：本单**无** `backend/**` 改动、**无** `frontend/**` 改动；Stage 3 亦然（只有 `AGENTS.md` + `CLAUDE.md`）。
- **`AGENTS.md` 的模块表行数保持 43**：不增删行 ⇒ 表内模块编号 1–43 不变 ⇒ 任何按编号引用模块的地方（如 issue 描述里的"模块 25 = AIGateway"）**仍然有效**。
- **旧文档值不必回填历史数据**（无数据）。
- **对下游 agent 的影响**：修复后 agent 读到的文件数 / 组件数与仓库实况一致，**范围判断不再被"86 个 View"这类过期数字误导**（父单描述里的原话）。

---

## 6. 风险与回滚

| 风险 | 概率 | 影响 | 缓解 / 回滚 |
|---|---|---|---|
| **实施时 main 已前进**，Stage 3 誊抄了本文档里的旧数字 | **高** | P0（父单硬约束 #5 明令禁止誊抄） | §3.1 表头已写死"必须用 §4.1 命令在当时的 tip 重算"；Stage 5 的判据就是"文档值 == 命令在 main 上的输出"⇒ 誊抄会当场被证伪 |
| **口径分歧**（尤其「12 个公共组件」）导致 Stage 3 写了 11 或写了别的数 | 中 | P1，Stage 2 打回 | §3.2 已给出口径 + 判据命令 + 反例；P2 交 PM 拍板 |
| **越权改动**（顺手改了措辞/内容列/命令/纪律） | 中 | **P0**（Stage 4 判越权 = P0） | §3.6 明确改动性质约束；Stage 5 用 `git diff` 逐行核对 + §7 的"越权检查"命令 |
| **改了计数但口径没写**，下次照样腐烂 | 中 | 机制价值归零 | §4.1/§4.2 同步清单进 `CLAUDE.md`（父单已拍板位置） |
| **回滚** | — | — | 本单只产出 1 个新文件（`docs/plans/finn-218-doc-count-calibration.md`）；Stage 3 的改动只碰 2 个文档 ⇒ 回滚 = `git revert <Stage3 commit>`，无数据/无部署需回退。**禁止** `git reset --hard`。 |

---

## 7. 测试要点（交给 devtools-qa 的可执行验收清单）

> 前置：Stage 3 的 commit 必须在 `origin/main` 之上；QA 从 `origin/main` 现读 tip，**不得**用实施方给的数字。

### T1 · 存在性与口径核对（逐条，脚本化）
```bash
REF=origin/main
git show ${REF}:AGENTS.md | grep -cE '^\| *[0-9]+ *\|'                      # 期望 == 标题里的数字
git ls-tree -r --name-only ${REF} -- frontend/src/views/ | grep -c '\.vue$'  # 期望 == 文档 View 数
git ls-tree -r --name-only ${REF} -- frontend/src/ | grep '\.vue$' | grep -vc '/views/'  # 期望 == 文档公共组件数
git ls-tree -r --name-only ${REF} -- backend/routes/ | grep -c '\.go$'      # 期望 == 文档 routes 文件数
git ls-tree --name-only ${REF}:backend/routes/ | grep '\.go$' | grep -v '_test\.go$' | grep -vc '^index\.go$'  # 期望 == 文档分组数
git ls-tree -r --name-only ${REF} -- backend/handlers/ | grep -c '\.go$'     # 期望 == 文档 handler 文件数
git ls-tree --name-only ${REF}:backend/handlers/ | grep '^ai_gateway' | grep -vc '_test\.go$'  # 期望 == 文档 ai_gateway 数
git show ${REF}:backend/handlers/skills.go | grep -cE '^[[:space:]]*Name:[[:space:]]+"'  # 期望 == 文档 skill 数
for d in ai collab convert dev draw home household life other share; do
  git ls-tree -r --name-only ${REF} -- frontend/src/views/$d | grep -c '\.vue$'   # 期望 == 文档该行括号数
done
```
**判定**：每个期望值都必须与 `origin/main` 上文档里的字面值相等；不等 ⇒ FAIL。

### T2 · 反事实自检（父单硬约束 #4）
- 对每一条**改过值**的断言：在**改前的 main**（Stage 3 的 parent commit）上跑同一条命令，结果必须 ≠ 改后文档值 ⇒ 证明这条断言"可被证伪"，不是恒真闸门。
- 唯二例外及理由（须在 QA 报告里写明，不得静默放过）：
  - 「12 个公共组件」：**值不变**（12 → 12）。判据改为"命令在改后 main 上算出 12 == 文档写的 12"（口径 C3），且必须**先于**改前 main 验证其成立（父单 PM 已把口径留给 Stage 1）。
  - 「draw/ 绘图（4）」「Docker Compose（4 服务）」「13 子组件」「7/6/4 分段」：**未改动**，不参与反事实，只需 T1 断言其当前成立。
- **恒真闸门检测**（必须真的判别，不能恒过）：把某条断言里的数字**故意改错**，确认 T1 脚本会 FAIL。做法：临时改一处文档数字 → 跑 T1 → 必须 FAIL → 还原。（这是"闸门可证伪"的最小充分检验。）

### T3 · 越权改动检查（Stage 4/5 共同）
```bash
git diff --stat <stage3_parent>..<stage3_head>          # 只能出现 AGENTS.md、CLAUDE.md 两个文件
git diff <stage3_parent>..<stage3_head> -- AGENTS.md CLAUDE.md | grep -E '^[+-]' | grep -vE '^(\+\+\+|---)'   # 逐行人工核对
```
- ⛔ 不得出现：命令文本、纪律条目、R1–R11、模块表「主要功能」列文字、模块顺序、模块**行数**（表格行数必须仍为 43）、章节标题文字（除数字）。
- ⛔ 不得出现：删除或改写 `CLAUDE.md` 的环境变量表、Module Path 段、开发命令段。

### T4 · 同步清单可用性
- 把 `CLAUDE.md` 新增的清单片段**整段复制**到 shell 里跑通，输出与 T1 相同 ⇒ 清单可粘贴可用。
- 检查 `${REF}:path` 的花括号未被 zsh 吃掉（在 zsh 下跑一遍，输出不得为空 / 不得报 "not a valid object name"）。

### T5 · 内部一致性
- `AGENTS.md:7` 标题数字 == `AGENTS.md:223` 的 "N 模块地图" == `CLAUDE.md:3` == `CLAUDE.md:67` == 模块表行数（5 个数字必须全等）。
- `AGENTS.md:113` == `AGENTS.md:209` == `CLAUDE.md:37` == `CLAUDE.md:52`（routes 文件数，4 处全等）。
- `AGENTS.md:59` == `CLAUDE.md:9` == `AGENTS.md:98` == `CLAUDE.md:34`（View 总数，4 处全等）。
- `AGENTS.md:59` == `CLAUDE.md:9`（公共组件数，2 处全等）。

### T6 · 已知残留缺陷（**不判 FAIL，但必须出现在报告里**）
- 模块表未收录 News / Chat+ / Trip 前端；模块表 View 列仅覆盖 88 个 View 中的 38 个。
- `ai` / `dev` / `life` 三个 category 的名字清单仍与实际文件不符（本单只改计数，名字清单属内容，另开单）。
- `AGENTS.md:115-126` 的 routes 树仍只枚举 38 个文件（缺 `news.go` / `trip.go` / `ssrule.go` + 2 个测试文件）⇒ 与标题里的 43 不一致（父单 #1 只许改计数与编号 ⇒ 不在本单修）。
- handler 分组括号数（辅助（7）列 9 个名字，其中 2 个文件实际在 `backend/utils/`）未对齐。

---

## 8. 未决问题

**无阻塞性未决问题。** §3.4 的 P1–P6 是**需要 PM 拍板的措辞/口径选择**，不阻塞方案定稿：每个 P 都已给出「两种以上可选解释 + 我的倾向 + 不裁决的后果」，Stage 3 可先按倾向执行，PM 若改口只需调整对应 1–3 处。

已明确**不在本单范围**、建议 PM 另开单：
1. 补齐模块表缺失的 News / Chat+ 行、修正 Trip 行的 View 列（需要写内容列）。
2. 刷新 `AGENTS.md:101-110` 的 per-category 文件名清单。
3. 刷新 `AGENTS.md:115-126` 的 routes 文件树枚举（38 → 43）。
4. 清理 handler 分组的括号数（`AGENTS.md:129-137`）。
5. 修 `AGENTS.md` 模块表与真实路由/handler 的其它收录差（例：`handlers/chat_plus.go`、`handlers/ai_chat_verify.go`、`handlers/english_tutor.go` 无独立模块行）。

---

## 附录 A · 勘察原始输出（`origin/main` @ `e8aef82`，2026-10-28）

```console
$ git ls-remote origin refs/heads/main
e8aef82b5cd5d3ae2e123f363a56b276ece03596	refs/heads/main

$ git show origin/main:AGENTS.md | grep -cE '^\| *[0-9]+ *\|'
43
$ git ls-tree -r --name-only origin/main -- frontend/src/views/ | grep -c '\.vue$'
88
$ git ls-tree -r --name-only origin/main -- frontend/src/ | grep '\.vue$' | grep -vc '/views/'
12
$ git ls-tree -r --name-only origin/main -- frontend/src/ | grep -c '\.vue$'
100
$ git ls-tree -r --name-only origin/main -- backend/routes/ | grep -c '\.go$'
43
$ git ls-tree --name-only origin/main:backend/routes/ | grep '\.go$' | grep -v '_test\.go$' | grep -vc '^index\.go$'
40
$ git show origin/main:backend/routes/index.go | grep -cE '^\s+Register[A-Za-z]*Route'
40
$ git ls-tree -r --name-only origin/main -- backend/handlers/ | grep -c '\.go$'
130
$ git ls-tree --name-only origin/main:backend/handlers/ | grep '^ai_gateway' | grep -vc '_test\.go$'
13
$ git ls-tree --name-only origin/main:backend/handlers/ | grep '^ai_gateway' | grep -c '_test\.go$'
9
$ git show origin/main:backend/handlers/skills.go | grep -cE '^[[:space:]]*Name:[[:space:]]+"'
7

$ for f in household autodev proxy; do git show origin/main:backend/handlers/$f.go | grep -oE "$f/section[0-9]+" | sort -u | wc -l; done
6
5
3
（+1 ⇒ 7 段 / 6 段 / 4 段，与文档现值一致）

$ git ls-tree -r --name-only origin/main -- frontend/src/views/dev/VirtualAvatarTool | grep -c '\.vue$'
14        （14 − 1 = 13 子组件，与文档现值一致）

$ git show origin/main:docker-compose.yml | grep -nE '^  [a-z_-]+:'
2:  devtools:
54:  redis:
68:  ocr-service:
96:  asr-service:
（4 个 service，其余为 volumes ⇒ 文档现值 4 正确）
```

## 附录 B · S1–S10 代码异味自查（本单交付物 = 1 个 Markdown 文档，无代码）

| 规则 | 检查方式 | 结果 |
|---|---|---|
| S1 死代码 | 文档无代码块以外的可执行逻辑；shell 片段逐行执行过（§2.2 / §4.1） | → 0 命中 |
| S2 未使用的 import/参数/局部变量 | 文档无 import / 变量声明 | → 0 命中 |
| S3 调试残留（`console.log`·`fmt.Println`·`TODO`·`FIXME`·`XXX`·`HACK`） | `grep -nE 'console\.log\|fmt\.Println\|TODO\|FIXME\|XXX\|HACK' docs/plans/finn-218-doc-count-calibration.md \| grep -v 'S3 调试残留'`（**必须排除本行自身**，否则清单里出现的关键词会让闸门恒真） | → 0 命中（唯一原始命中是本行自身的关键词清单，已按上式排除；见 §附录 C 复核） |
| S4 注释掉的代码块 | 无被注释掉的旧实现代码 | → 0 命中 |
| S5 重复逻辑 | §4.1 与 §3.6 的命令块内容一致（同一份，唯一权威源） | → 0 命中 |
| S6 魔法数与魔法串 | 所有数字均以"实测值 + 判据命令"成对出现，无裸常量断言 | → 0 命中 |
| S7 空 `catch` / 忽略的 error | 文档内无代码错误处理 | → 0 命中 |
| S8 命名与项目风格不一致 | 文件名 `finn-218-doc-count-calibration.md` 沿用 `docs/plans/` 的 kebab-case + `finn-<单号>` 前缀惯例 | → 0 命中 |
| S9 越界改动 | 本单只新增 `docs/plans/finn-218-doc-count-calibration.md` 一个文件；未动 `AGENTS.md` / `CLAUDE.md` / `backend/**` / `frontend/**` / `docs/INDEX.md` | → 0 命中 |
| S10 遗留占位 | 无 `TBD` / `待补` 占位（§3.4 的 P1–P6 已给出倾向 + 后果，非占位） | → 0 命中 |

## 附录 C · 待复核项（交付时由本单自己在 commit 后回填）

- `git diff --stat <base>..HEAD`：应只有 `docs/plans/finn-218-doc-count-calibration.md`
- 分支名 / commit hash / push 状态：见交付评论。

### ⚠️ 给 Stage 3 实施方的一个坑（实测）

本仓库的 agent worktree 里，**Multica 运行时会把一段 `<!-- BEGIN MULTICA-RUNTIME (auto-managed; do not edit) -->` 块自动追加到 `CLAUDE.md` 末尾**（本单开工时 `git status` 实测：` M CLAUDE.md`，+269 行，全部是运行时指令文本）。

- 该块**追加在文件末尾**，**不改变第 1–99 行的行号** ⇒ 本方案里所有 `CLAUDE.md` 行号（3 / 9 / 12 / 34 / 37 / 39 / 52 / 67）在 worktree 与 `origin/main` 上**一致**，§3.1 的定位有效。
- Stage 3 实施时**不要**把这段自动托管文本提交进仓库（它是运行时产物，不属于项目文档）；提交前用 `git diff --stat` 确认 `CLAUDE.md` 的改动行数 ≈ 本单要求的 7 处 + 新增清单，而**不是** 269 行。
- 判据：`git diff --numstat -- CLAUDE.md` 的 additions 应是「7 处改值 + 新增清单块」量级；若是 269 左右，说明误把运行时块带上了。
