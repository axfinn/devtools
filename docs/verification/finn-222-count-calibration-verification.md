# FINN-222 · Stage 5 独立复验记录 —— AGENTS.md / CLAUDE.md 计数校准（父单 FINN-217 / 实施单 FINN-220）

> 复验人：devtools-qa ｜ 单号：FINN-222（Stage 5）｜ 日期：2026-10-28
> 被验对象：分支 `agent/devtools-architect/finn-220`，tip `c6b3c5ac84ffa533cd3cbb38ad1bf4f4ad46bbfa`
> 现算基准：`origin/main` = `e8aef82b5cd5d3ae2e123f363a56b276ece03596`
> 改前基点：上游分支的 baseline commit `88cb031c078b36fcf11465a13432c84ab913cc82`
> 复验脚本：`scripts/verify_count_calibration.py`（本单产出，只读）

---

## 0. 结论

**PASS** —— 30 条改值断言 + 11 条未改动断言 + 越权/清单/一致性检查共 **102 项检查全部 PASS**，脚本 exit 0。
且已用**负控制**证明本复验闸门**不是恒真闸门**（把「改后文档」换回「改前文档」⇒ 35 项判红、exit 1）。

无 P0，无阻塞项。3 条**非阻塞残留**（含 Stage 4 已登记项）记录在 §8。

---

## 1. 取码与环境（可复核）

```bash
export GIT_SSH_COMMAND='ssh -o ProxyCommand=none -o ConnectTimeout=20'
D=$(mktemp -d /tmp/f217-s5.XXXXXX)
git clone --no-checkout --filter=blob:none --single-branch \
  --branch agent/devtools-architect/finn-220 git@github.com:axfinn/devtools.git "$D"
git -C "$D" config core.sshCommand 'ssh -o ProxyCommand=none -o ConnectTimeout=20'
git -C "$D" fetch origin '+refs/heads/*:refs/remotes/origin/*'
```

实测结果（与单描述的期望值逐项一致）：

| 项 | 期望 | 实测 |
|---|---|---|
| clone 出来的 HEAD | `c6b3c5ac84ffa533cd3cbb38ad1bf4f4ad46bbfa` | ✅ 同值 |
| `origin/main` tip | `e8aef82b5cd5d3ae2e123f363a56b276ece03596` | ✅ 同值 |
| `git log --oneline origin/main..$TIP` | 2 条（1 baseline + 1 产品） | ✅ 2 条 |
| 非 baseline commit 数 | 1 | ✅ 1 |
| `git show --name-only --format= $TIP` | 恰好 `AGENTS.md` + `CLAUDE.md` | ✅ 恰好这 2 个 |
| `git show $TIP:CLAUDE.md \| grep -c MULTICA-RUNTIME` | 0 | ✅ 0 |

⚠️ 全程在 `/tmp` 克隆里操作，**未在 SMB 平台 worktree 里 fetch**，未触碰部署主机，未跑任何服务。

---

## 2. 复验方法（三层判据，逐条跑）

对每条计数断言跑三层：

| 层 | 判据 | 含义 |
|---|---|---|
| **T1 现算** | 判据命令在 `origin/main` 树上的输出 **==** 改后文档写的值 | 该断言在当前 main 上为**真** |
| **T2 反事实** | 改前文档写的值 **!=** T1 的现算值 | 该断言有**区分力**（不是恒真闸门） |
| **T2′ 注入实验** | 把改后文档里的数字改错 ⇒ 闸门判红；还原 ⇒ 判绿 | 该断言的**闸门真判别** |

外加 **§6 闸门级反事实**：把「改后文档」整体换回「改前文档」，30 条非豁免断言必须**全部判红** —— 这是对**整套 T1 闸门**的反事实，不依赖任何单条注入。

### 2.1 改前基点的有效性（前置控制）

「改前」不能随便取。已实测：`88cb031` 这个 baseline commit **未触碰** `AGENTS.md` / `CLAUDE.md`，
故其 blob 与 `origin/main` 的同名文件**逐字节相同**：

```
git rev-parse 88cb031:AGENTS.md  origin/main:AGENTS.md
  4529a70664f46645c3e356566a0a088a6cf5f7fc
  4529a70664f46645c3e356566a0a088a6cf5f7fc
git rev-parse 88cb031:CLAUDE.md  origin/main:CLAUDE.md
  eb522fd5c2018fe42a528143cdabcfe6c5f8e2e5
  eb522fd5c2018fe42a528143cdabcfe6c5f8e2e5
```

⇒ 「改前」= 改前 main，**基点成立**。脚本 §0 把这条做成硬检查（不一致即 FAIL）。

---

## 3. 逐条复算（T1 + T2 + 注入实验）—— 改值断言全集 30 条

脚本输出原文（`现算值` = 判据命令在 `origin/main` 上的输出）：

| ID | 文件 | 断言 | 口径 | 现算值 | 改前值 | 改后值 | T1 | T2 反事实 | 注入实验 |
|---|---|---|---|---|---|---|---|---|---|
| A1 | AGENTS.md | `## 43 模块地图` | C1 | 43 | 42 | 43 | PASS | PASS | 注入 44 → RED |
| A2 | AGENTS.md | `88 个 View 组件`（技术栈行） | C2 | 88 | 86 | 88 | PASS | PASS | 注入 89 → RED |
| A2b | AGENTS.md | `12 个公共组件`（技术栈行） | C3 | 12 | 12 | 12 | PASS | **豁免**（PM 裁决 P2） | 注入 13 → RED |
| A3 | AGENTS.md | `### 前端（88 View 组件）` | C2 | 88 | 86 | 88 | PASS | PASS | 注入 89 → RED |
| A4 | AGENTS.md | `ai/ # AI 相关（7）` | C4-ai | 7 | 6 | 7 | PASS | PASS | 注入 8 → RED |
| A5 | AGENTS.md | `collab/ # 协作（6）` | C4-collab | 6 | 8 | 6 | PASS | PASS | 注入 7 → RED |
| A6 | AGENTS.md | `convert/ # 转换（7）` | C4-convert | 7 | 8 | 7 | PASS | PASS | 注入 8 → RED |
| A7 | AGENTS.md | `dev/ # 开发（22）` | C4-dev | 22 | 10 | 22 | PASS | PASS | 注入 23 → RED |
| A9 | AGENTS.md | `household/ # 家庭（10 个子组件）` | C4-household | 10 | 9 | 10 | PASS | PASS | 注入 11 → RED |
| A10 | AGENTS.md | `life/ # 生活（13）` | C4-life | 13 | 11 | 13 | PASS | PASS | 注入 14 → RED |
| A11 | AGENTS.md | `other/ # 其他（10）` | C4-other | 10 | 9 | 10 | PASS | PASS | 注入 11 → RED |
| A12 | AGENTS.md | `share/ # 分享视图（8）` | C4-share | 8 | 9 | 8 | PASS | PASS | 注入 9 → RED |
| A13 | AGENTS.md | `routes/ 43 个文件`（标题） | C5 | 43 | 42 | 43 | PASS | PASS | 注入 44 → RED |
| A14 | AGENTS.md | `AI 网关（19）` | C8 汇总 | 19 | 18 | 19 | PASS | PASS | 注入 20 → RED |
| A15 | AGENTS.md | `ai_gateway*（13，含 1 个 test helper）` | C8 | 13 | 12 | 13 | PASS | PASS | 注入 14 → RED |
| A18 | AGENTS.md | 模块表 23 `nfsshare*.go (4 文件)` | C8 | 4 | 5 | 4 | PASS | PASS | 注入 5 → RED |
| A0 | AGENTS.md | 模块表 25 `ai_gateway*.go (13 文件…)` | C8 | 13 | 12 | 13 | PASS | PASS | 注入 14 → RED |
| A20 | AGENTS.md | 模块表 37 `skills*.go (1 文件)` | C8 | 1 | 3 | 1 | PASS | PASS | 注入 2 → RED |
| A21 | AGENTS.md | 模块表 37 `7 个 skill` | C9 | 7 | 5 | 7 | PASS | PASS | 注入 8 → RED |
| A22a | AGENTS.md | `routes/ 包（43 文件，…）` | C5 | 43 | 42 | 43 | PASS | PASS | 注入 44 → RED |
| A22b | AGENTS.md | `routes/ 包（…，40 分组）` | C6 | 40 | 34 | 40 | PASS | PASS | 注入 41 → RED |
| A23 | AGENTS.md | `6. 更新本文件 43 模块地图` | C1 | 43 | 42 | 43 | PASS | PASS | 注入 44 → RED |
| A24 | AGENTS.md | `测试（52）` | C7t | 52 | 20 | 52 | PASS | PASS | 注入 53 → RED |
| B1 | CLAUDE.md | `（43 模块地图、…）` | C1 | 43 | 42 | 43 | PASS | PASS | 注入 44 → RED |
| B2 | CLAUDE.md | `88 个 View 组件`（技术栈行） | C2 | 88 | 86 | 88 | PASS | PASS | 注入 89 → RED |
| B2b | CLAUDE.md | `12 个公共组件`（技术栈行） | C3 | 12 | 12 | 12 | PASS | **豁免**（PM 裁决 P2） | 注入 13 → RED |
| B3 | CLAUDE.md | `# 88 Vue 组件` | C2 | 88 | 86 | 88 | PASS | PASS | 注入 89 → RED |
| B4 | CLAUDE.md | `# 43 个路由文件` | C5 | 43 | 40 | 43 | PASS | PASS | 注入 44 → RED |
| B5 | CLAUDE.md | `# 130 个 handler 文件` | C7 | 130 | 118 | 130 | PASS | PASS | 注入 131 → RED |
| B6a | CLAUDE.md | 关键设计模式 `（43 文件，…）` | C5 | 43 | 38 | 43 | PASS | PASS | 注入 44 → RED |
| B6b | CLAUDE.md | 关键设计模式 `（…，40 分组）` | C6 | 40 | 34 | 40 | PASS | PASS | 注入 41 → RED |
| B7 | CLAUDE.md | `6. 更新 AGENTS.md 43 模块地图` | C1 | 43 | 42 | 43 | PASS | PASS | 注入 44 → RED |

**30 条全部 PASS。**（Stage 4 说「29 条改值断言」；本单按**独立数字**逐个数得 **30 条** —— 差异在 Stage 4 是否把
`AGENTS.md` 技术栈行与 `CLAUDE.md` 技术栈行的「88 个 View 组件」计作同一条。**条数口径不影响任何一条的结论**。）

### 3.1 P2 豁免的处理（按单描述执行，未误判 FAIL）

「12 个公共组件」前后值不变（12 → 12）。按 PM 裁决 P2：
- 判据 = 「改后命令算出 12 == 文档写的 12」⇒ **PASS**（实测现算 12 / 改后文档 12）；
- 改前同样算出 12 ⇒ 属**已批准偏离**，**不因「改前值 == 改后值」判 FAIL**；
- 但**仍做了注入实验**（注入 13 ⇒ 判红）⇒ 该条同样不是恒真闸门。

### 3.2 口径 C8 / C9 / C10 / C11 的实算明细（供复核）

```
# C8（模块表 (N 文件) 一律排除 _test.go）
git ls-tree --name-only ${REF}:backend/handlers/ | grep '^ai_gateway' | grep -vc '_test\.go$'   → 13
  ai_gateway.go ai_gateway_admin.go ai_gateway_anthropic.go ai_gateway_chat.go ai_gateway_chat_stream.go
  ai_gateway_complete.go ai_gateway_image.go ai_gateway_qwen.go ai_gateway_test_helpers.go
  ai_gateway_tokenplan.go ai_gateway_tts.go ai_gateway_vlm.go ai_gateway_voice.go
git ls-tree --name-only ${REF}:backend/handlers/ | grep '^nfsshare'  | grep -vc '_test\.go$'  → 4
  nfsshare.go nfsshare_handlers.go nfsshare_smb.go nfsshare_watch.go
git ls-tree --name-only ${REF}:backend/handlers/ | grep '^skills'    | grep -vc '_test\.go$'  → 1
  skills.go
# C9（内置 skill）
git show ${REF}:backend/handlers/skills.go | grep -cE '^[[:space:]]*Name:[[:space:]]+"'       → 7
# C8 汇总（AI 网关 19）
13(ai_gateway) + 4(minimax_*) + 1(image_understanding.go) + 1(planner_ai.go) = 19
```

---

## 4. 未改动但方案 §3.1 标「不改」的断言（只验当前成立）—— 11 条

| ID | 断言 | 口径 | 现算值 | 文档值 | 结论 | 注入实验 |
|---|---|---|---|---|---|---|
| U1 | `draw/ # 绘图（4）` | C4-draw | 4 | 4 | PASS | 注入 5 → RED |
| U2 | `minimax_*（4）` | C8-minimax | 4 | 4 | PASS | 注入 5 → RED |
| U3 | 模块表 20 `planner*.go (6 文件)` | C8-planner | 6 | 6 | PASS | 注入 7 → RED |
| U4 | 模块表 26 `askit*.go (3 文件)` | C8-askit | 3 | 3 | PASS | 注入 4 → RED |
| U5 | 模块表 22 `household.go (7 分段)` | C10-household | 7 | 7 | PASS | 注入 8 → RED |
| U6 | 模块表 29 `autodev.go (6 分段)` | C10-autodev | 6 | 6 | PASS | 注入 7 → RED |
| U7 | 模块表 31 `proxy.go (4 分段)` | C10-proxy | 4 | 4 | PASS | 注入 5 → RED |
| U8 | 模块表 40 `(+ 13 子组件)` | C11 | 13 | 13 | PASS | 注入 14 → RED |
| U9 | `Docker Compose（4 服务）` | compose `services:` | 4 | 4 | PASS | 注入 5 → RED |
| U10 | `12 个公共组件`（AGENTS.md） | C3 | 12 | 12 | PASS | 注入 13 → RED |
| U11 | `12 个公共组件`（CLAUDE.md） | C3 | 12 | 12 | PASS | 注入 13 → RED |

分段数口径 C10 实算（`sort -u | wc -l` + 1）：
`household.go` → 6 个 `household/sectionN` + 1 = **7**；`autodev.go` → 5 + 1 = **6**；`proxy.go` → 3 + 1 = **4**。

---

## 5. 负控制 —— 证明本闸门不是恒真闸门

把 `--tip` 换成**改前的** `origin/main`（即「文档没被改」的场景），同一脚本：

```
$ python3 scripts/verify_count_calibration.py --repo "$D" --ref origin/main \
      --base origin/main --tip origin/main
汇总：97 项检查，PASS 62，FAIL 35
exit=1
```

判红的 35 项 = **30 条改值断言全部判红**（T1 现算值 ≠ 改前文档值）+ 越权检查 2 项 + 清单块缺失 2 项 + 内部一致性 1 项。

⇒ **闸门对 30 条改值断言全部具备区分力**；若实施方没改（或改错）任何一条，脚本必然 exit 1。

---

## 6. 越权检查（T3）

```
$ git show --name-only --format= $TIP
AGENTS.md
CLAUDE.md
```
⇒ 实施方的产品 commit **恰好**只碰这 2 个文件，**无越权**。

```
$ git diff --numstat 88cb031..$TIP
19      19      AGENTS.md
55      7       CLAUDE.md
```
- `AGENTS.md`：19 增 / 19 删 ⇒ **一对一替换**，未新增/删除行 ⇒ 模块表行数不变（43 行）、模块顺序不变、内容列文字未动。
- `CLAUDE.md`：55 增 / 7 删 ⇒ 增的部分是**新增的「计数同步清单」小节**（父单 PM 已拍板的新增内容），7 删对应 7 行被替换的计数行。

逐行核对（`git diff 88cb031..$TIP` 全文已审）确认：
- 被删/被增的行**只含数字变化**，无命令文本、无纪律条目、无 R1–R11 语义、无模块表「主要功能」列文字改动；
- `CLAUDE.md` 的环境变量表 / Module Path 段 / 开发命令段**未被触碰**；
- `git show $TIP:CLAUDE.md | grep -c MULTICA-RUNTIME` = **0**（运行时托管块未带入）。

### 6.1 同步清单新增内容是否越权

`CLAUDE.md` 新增了「计数同步清单」小节（命令块 + 口径速查 + 触发条件清单 + 「每次改完必跑」块）。
- 父单 PM 描述明确：「因改成精确值，§4.2 同步清单**必须补一行**…该清单进 `CLAUDE.md`（父单已拍板的位置），属已批准的新增内容，不算越权」；
- 父单硬约束 #3：「只允许改**计数类**表述…⛔ 不许改任何命令、纪律、R1–R11 语义」—— 新增小节是**新增的独立章节**，未改写任何既有命令或纪律条目。

⇒ **判定：不越权**（且属已批准的必需配套，见 §8 残留 #3 的补充说明）。

---

## 7. 同步清单可执行性（T4）

把 `CLAUDE.md` 里新增的 bash 块**逐字**抽出来跑（脚本用 `re.findall(r"```bash\n(.*?)```")` 抽原文，不重写）：

| 块 | shell | REF | rc | 输出 | 结论 |
|---|---|---|---|---|---|
| 计数同步清单 | bash | origin/main | 0 | 19 行 | PASS |
| 计数同步清单 | **zsh** | origin/main | 0 | 19 行 | PASS（`${REF}:path` 花括号未被 zsh `:A` 修饰符吃掉） |
| 「每次改完必跑」 | bash | origin/main | 0 | `## 43 模块地图` / `43` | PASS（标题数字 == 表行数） |
| 「每次改完必跑」 | **zsh** | origin/main | 0 | `## 43 模块地图` / `43` | PASS |
| 计数同步清单 | bash | HEAD | 0 | 19 行 | PASS（但数值有偏差，见 §8 残留 #1） |

计数同步清单（`REF=origin/main`）实测输出：

```
模块表行数       : 43
View 组件        : 88
公共组件         : 12
routes 文件      : 43
routes 分组      : 40
handlers 文件    : 130
handlers 测试    : 52
ai_gateway 非测试: 13
内置 skill 数     : 7
ai         7
collab     6
convert    7
dev        22
draw       4
home       1
household  10
life       13
other      10
share      8
```

⇒ 与文档里写的每一个数字**逐项吻合**（43 / 88 / 12 / 43 / 40 / 130 / 52 / 13 / 7 / 7 / 6 / 7 / 22 / 4 / 10 / 13 / 10 / 8）。

---

## 8. 内部一致性（T5）

| 分组 | 涉及断言 | 实测值 | 结论 |
|---|---|---|---|
| 模块地图数（4 处全等） | A1 / A23 / B1 / B7 | 43 | PASS |
| routes 文件数（4 处全等） | A13 / A22a / B4 / B6a | 43 | PASS |
| View 总数（4 处全等） | A2 / A3 / B2 / B3 | 88 | PASS |
| 公共组件数（2 处全等） | A2b / B2b | 12 | PASS |
| ai_gateway 文件数（2 处全等） | A0 / A15 | 13 | PASS |

---

## 9. 已知残留 / 非阻塞项（**不判 FAIL**，但必须登记）

### 残留 #1（Stage 4 非阻塞项 #1，已复现并量化）—— `REF=HEAD` 在**交付分支**上会算错

计数同步清单声明「改自己的分支时用 HEAD」。在**上游交付分支**上跑 `REF=HEAD` 实测：

```
handlers 文件    : 133     ← 文档写 130
handlers 测试    : 54      ← 文档写 52
ai_gateway 非测试: 14      ← 文档写 13
```

根因：交付分支的树里含**平台 baseline commit `88cb031` 的 3 个在途 WIP 文件**
（`backend/handlers/ai_gateway_comfy.go`、`ai_gateway_comfy_client_test.go`、`ai_gateway_comfy_test.go`）。
这 3 个文件**不是实施方的改动**（Stage 6 按文件取件不会把它们带进 main）。

⇒ **文档按 `origin/main` 写的 130 / 52 / 13 是对的**，不是缺陷。
但「改自己的分支时用 HEAD」这句措辞在**携带 baseline WIP 的分支**上会给出偏大的数 ——
建议后续单把措辞改成「用 `origin/main`（或用 `git diff --stat` 确认自己的分支没有额外在途文件后再用 HEAD）」。

### 残留 #2（Stage 4 非阻塞项 #2）—— 「含 1 个 test helper」是硬编码常量，无判据命令

`CLAUDE.md` 口径速查写死了「含 1 个 `ai_gateway_test_helpers.go` 脚手架」。
现状**为真**（实测 `ai_gateway*` 非测试 13 个，其中恰好 1 个是 `ai_gateway_test_helpers.go`），
但将来加第二个 helper 文件就会腐。**本单不判 FAIL**（父单硬约束 #1 只许改计数与编号）。

建议写法（供后续单采纳）：
`ai_gateway 非测试 = ai_gateway* 去掉 _test.go`（含 `ai_gateway_test_helpers.go` 脚手架）——
即把「1 个」从数字降级为**举例说明**，不作为计数断言。

### 残留 #3（Stage 4 非阻塞项 #3）—— 「每次改完必跑」块的两行数字**不同源**

```
grep -E '^## [0-9]+ 模块地图' AGENTS.md          # 读【工作区】AGENTS.md 的标题
git show ${REF}:AGENTS.md | grep -cE '^\| *[0-9]+ *\|'   # 读【${REF}】AGENTS.md 的表行数
```
第一行读工作区、第二行读 `${REF}` ⇒ 原理上可能不同源（本次恰好相等：43 == 43）。
建议后续单统一为两行都读工作区（或都读 `${REF}`）。

### 残留 #4（Stage 4 非阻塞项 #4/5）—— 内容列不同步（本单无权改，已逐条量化）

| 残留 | 实测 |
|---|---|
| 模块表缺 **News** 行 / 无 `chat_plus.go` 覆盖 | 模块表无 `**News` 行；无任何行引用 `chat_plus.go` |
| `AGENTS.md` routes 树只枚举 **38** 个 `.go`，标题写 43 | 树里缺 `news.go` / `ssrule.go` / `ssrule_qa_test.go` / `ssrule_test.go` / `trip.go`（5 个） |
| per-category 名字清单与实际文件名不符 | `ai` 未列 `RolePlayTool`（实际 7 / 列 6）；`life` 未列 `NewsTool`、`TripTool`（实际 13 / 列 11）；`other` 多列 `Neon`（`frontend/src/neon/` 是目录不是 .vue）、`VibeMotion`（无此文件），未列 `SsRuleTool`（实际 10 / 列 10）；`dev` 实际 22 / 列 9（差 13 个 `VirtualAvatarTool/` 子组件，PM 裁决 P3 已明示不加消歧括注）；`household` 实际 10 / 列 0（该行只写「（10 个子组件）」不列名字） |
| handler 分组括号数与同行枚举对不上 | `辅助（7）` 列 9 个名字，其中 `security_enhance.go` / `content_parser.go` 实际在 `backend/utils/`；其余组括号数无单条判据（PM 裁决 P6 已明示另开单） |

⇒ 全部属**内容列**，父单硬约束 #1 明令本单不许动，**不判 FAIL**。

### 残留 #5（本单新发现 · 非阻塞）—— 方案 §3.1 表 A **漏列**了父单 #7 的一条，实施方已正确补上

父单硬约束清单第 7 条（`AGENTS.md:35` 模块表 `handlers/ai_gateway*.go（12 文件，含 1 个测试脚手架）` → 13）
在方案 §3.1 表 A 中**没有对应行**（表 A 从 A15 的 `AGENTS.md:130` 直接跳到 A18 的 `AGENTS.md:33`），
但**实施方在 Stage 3 实际改了**这一行（`AGENTS.md` 第 25 行模块表：`(12 文件，含 1 个测试脚手架)` → `(13 文件，含 1 个测试脚手架)`）。

本单已把它作为 **A0** 纳入复验并判 PASS（现算 13 == 改后文档 13；改前 12 ≠ 13；注入 14 → RED）。
⇒ **实施方行为正确（按父单而非按方案表执行），是方案表的漏列**，不判 FAIL。
建议把该条登记到方案 §3.1 表 A（补一行），供后续单沿用。

---

## 10. 未覆盖（明确写出，不写 = 默认全覆盖）

1. **未测任何 Go / Vue 代码路径** —— 本单不涉及产品代码；未跑 `go build` / `go test` / `vitest` / `pnpm build`（单描述 §5 明示不需要）。
2. **未跑服务、未部署、未碰部署主机 `192.168.31.201`**（R1）。
3. **未验「Stage 6 合并进 main 之后」的最终态** —— 本单只验到「合并前的 origin/main 树上这些数字成立」。
   Stage 6 按文件取件后，main 的树 = 本单验过的 `origin/main` 树 + 2 个文档文件 ⇒ 结论可继承；但**合并本身**需另单验。
4. **未验 `main` 继续前进后的漂移** —— 本单基线是 `origin/main` @ `e8aef82`（PM 放行时 main 未前进）。若 main 之后新增了 View / routes / handler 文件，
   文档里的数字会**再次漂移** —— 这正是本单新增的「计数同步清单」要防的事，但**清单的长期有效性需要后续多轮维护才能验证**，本单只能验「清单本身可执行」。
5. **未验 `dev/ # 开发（22）` 的「22 含 `VirtualAvatarTool/` 子目录 14 个」这一语义** —— PM 裁决 P3 已明示不加消歧括注、且 P3 副产物「与同行枚举对不上」已明示不判 FAIL；本单只验「22 == `views/dev/` 递归 `.vue` 数」。
6. **未验前端构建 / 运行时**（375×667 首屏等真机项按 D5 由 jaxiu 合并后自行验收）。
7. **未做压力测试 / e2e**（本单无运行时代码）。

---

## 11. 复现命令

```bash
export GIT_SSH_COMMAND='ssh -o ProxyCommand=none -o ConnectTimeout=20'
D=$(mktemp -d /tmp/f217-s5.XXXXXX)
git clone --no-checkout --filter=blob:none --single-branch \
  --branch agent/devtools-architect/finn-220 git@github.com:axfinn/devtools.git "$D"
git -C "$D" config core.sshCommand 'ssh -o ProxyCommand=none -o ConnectTimeout=20'
git -C "$D" fetch origin '+refs/heads/*:refs/remotes/origin/*'

# 正例（期望：102 项检查全 PASS，exit 0）
python3 scripts/verify_count_calibration.py --repo "$D" --ref origin/main \
  --base 88cb031c078b36fcf11465a13432c84ab913cc82 \
  --tip refs/remotes/origin/agent/devtools-architect/finn-220

# 负控制（期望：35 项 FAIL，exit 1）
python3 scripts/verify_count_calibration.py --repo "$D" --ref origin/main \
  --base origin/main --tip origin/main
```

依赖：`python3`（实测 3.14.7）+ `bash` + `zsh` + `git`。脚本**只读**，不写被验仓库（唯一写入是把改后文档铺到克隆工作树以跑「每次改完必跑」块）。

---

## 12. S1–S10 代码异味自查（本单交付物 = 1 个 Python 脚本 + 1 个 Markdown 记录）

| 规则 | 检查方式 | 结果 |
|---|---|---|
| S1 死代码 | `CRIT` 字典 11 个口径码全部被 `crit()` 或断言表引用；`Runner.show` / `find_one` / `inject` / `check` 均被调用 | → 0 命中 |
| S2 未使用的 import / 参数 / 局部变量 | `argparse/os/re/subprocess/sys` 全部使用；`rows/fails/vals/docs_base/docs_tip` 全部使用 | → 0 命中 |
| S3 调试残留 | `grep -nE 'TODO\|FIXME\|XXX\|HACK' scripts/verify_count_calibration.py docs/verification/finn-222-count-calibration-verification.md` ⇒ 原始输出 **2 行**，均为 `D=$(mktemp -d /tmp/f217-s5.XXXXXX)` 里的 **`mktemp` 模板占位 `XXXXXX`**（假阳性，非调试残留）；**脚本本体 0 命中** | → 0 命中（2 处假阳性已逐条认定） |
| S4 注释掉的代码块 | `grep -nE '^\s*#\s*(if\|for\|while\|def \|return \|import \|git \|echo )' scripts/verify_count_calibration.py` ⇒ rc=1（0 命中） | → 0 命中 |
| S5 重复逻辑 | 口径命令集中在 `CRIT` + `crit()` 一处；断言表是数据不重复实现 | → 0 命中 |
| S6 魔法数与魔法串 | 断言表以「文档正则 → 口径码」成对出现，无裸常量断言；注入偏移固定 `+1` 且在代码注释里说明用途 | → 0 命中 |
| S7 空 catch / 忽略的 error | `grep -nE 'except\s*:' scripts/verify_count_calibration.py` ⇒ rc=1（0 命中）；`r.sh` 返回 `(rc, out, err)`，所有调用点均检查 `rc` 或比较输出 | → 0 命中 |
| S8 命名与项目风格不一致 | 文件名 `verify_count_calibration.py`（kebab/snake，与仓库既有 `asr-service/main.py` 等 Python 文件同风格）；记录文件名 `finn-222-…-verification.md`（kebab-case + `finn-<单号>` 前缀，沿用 `docs/plans/finn-218-…` 惯例） | → 0 命中 |
| S9 越界改动 | 提交前 `git status --short` = `?? docs/verification/` + `?? scripts/verify_count_calibration.py`（+ 平台目录 `.claude/skills/multica-platform/`、`.multica/`，**未纳入提交**）；`git diff HEAD --stat` ⇒ **空**（未修改任何既有文件） | → 0 命中 |
| S10 遗留占位 | `grep -nE 'TBD\|待补\|FIXME' scripts/verify_count_calibration.py docs/verification/finn-222-count-calibration-verification.md` ⇒ 唯一命中是**本表 S10 行自身**（自指命中，已排除）⇒ 实质 0 命中 | → 0 命中 |
