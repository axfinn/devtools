# CLAUDE.md

DevTools 项目快速参考。详细架构和 API 文档见 `AGENTS.md`（45 模块地图、完整 handler/route 索引、详细 API 索引）。

---

## 概览

- **前端**：Vue 3 + Vite + Element Plus + TailwindCSS（88 个 View 组件 + 12 个公共组件）
- **后端**：Go (Gin) + SQLite + **Redis（软依赖，失败自动降级内存存储）**
- **辅助服务**：ocr-service (8000), asr-service (9000), tts-service (8083), diarize-service (9100, 可选)
- **部署**：Docker Compose（4 服务）

---

## 开发命令

```bash
# 前端
cd frontend && pnpm install && pnpm dev      # :5173  (前端统一用 pnpm,锁文件 pnpm-lock.yaml)

# 后端
cd backend && go mod tidy && go run main.go   # :8080

# Docker
docker compose up -d && docker compose logs -f devtools
```

---

## 项目结构速查

```
frontend/src/views/       # 88 Vue 组件，按 category/ 分组
backend/
├── main.go              # 入口：DB 初始化、中间件、每小时清理 goroutine
├── routes/              # 43 个路由文件（routes.go 已拆分，含 index.go）
│   └── index.go         # RouteHandlers struct + RegisterAllRoutes
├── handlers/            # 140 个 handler 文件（按域分组）
├── models/              # SQLite 操作（自动建表）
├── middleware/          # RateLimiter、ContentSizeLimiter 等
└── state/               # Redis/内存状态（Redis 软依赖）
```

---

## 关键设计模式

| 模式 | 说明 |
|------|------|
| **Redis 软依赖** | `state.New()` → Redis 失败返回 `MemoryStore`，进程不退出 |
| **路由拆分** | `routes.go` → `routes/` 包（43 文件，40 分组） |
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
6. 更新 `AGENTS.md` 45 模块地图

### 计数同步清单（改动前后各跑一次，按输出填文档）

```bash
REF=${REF:-origin/main}   # 校准文档时用 origin/main；改自己的分支时用 HEAD
echo "模块表行数       : $(git show ${REF}:AGENTS.md | grep -cE '^\| *[0-9]+ *\|')"
echo "View 组件        : $(git ls-tree -r --name-only ${REF} -- frontend/src/views/ | grep -c '\.vue$')"
echo "公共组件         : $(git ls-tree -r --name-only ${REF} -- frontend/src/ | grep '\.vue$' | grep -vc '/views/')"
echo "routes 文件      : $(git ls-tree -r --name-only ${REF} -- backend/routes/ | grep -c '\.go$')"
echo "routes 分组      : $(git ls-tree --name-only ${REF}:backend/routes/ | grep '\.go$' | grep -v '_test\.go$' | grep -vc '^index\.go$')"
echo "handlers 文件    : $(git ls-tree -r --name-only ${REF} -- backend/handlers/ | grep -c '\.go$')"
echo "handlers 测试    : $(git ls-tree -r --name-only ${REF} -- backend/handlers/ | grep '\.go$' | grep -c '_test\.go$')"
echo "ai_gateway 非测试: $(git ls-tree --name-only ${REF}:backend/handlers/ | grep '^ai_gateway' | grep -vc '_test\.go$')"
echo "内置 skill 数     : $(git show ${REF}:backend/handlers/skills.go | grep -cE '^[[:space:]]*Name:[[:space:]]+\"')"
for d in ai collab convert dev draw home household life other share; do
  printf '%-10s %s\n' "$d" "$(git ls-tree -r --name-only ${REF} -- frontend/src/views/$d | grep -c '\.vue$')"
done
```

口径速查（括号里的数 = 上面的输出，**不要手数**）：
`View 组件` = `frontend/src/views/` 递归 `.vue`；`公共组件` = `frontend/src/` 下除 `views/` 外的 `.vue`（含 `App.vue`）；
`routes 文件` = `backend/routes/` 全部 `.go`；`routes 分组` = 非测试、非 `index.go` 的路由文件数（每文件 1 个 `Register*Routes`）；
`handlers 文件` = `backend/handlers/` 全部 `.go`；`handlers 测试` = 其中 `_test.go` 的数量；
`ai_gateway 非测试` = `ai_gateway*` 去掉 `_test.go`（`ai_gateway_test_helpers.go` 是测试脚手架、**仍计入**该数；脚手架个数随文件增减变化，**不要写死个数**，以命令块里 `ai_gateway 非测试` 那一行的输出为准）；
模块表 `handlers/xxx*.go (N 文件)` 的 N 一律**排除 `_test.go`**。

**触发条件 → 必须重算并同步的文档条目**：

- 新增/删除 `frontend/src/views/` 下的 `.vue` → 同步 View 组件数 / 各 category 计数
- 新增/删除 `frontend/src/` 下非 `views/` 的 `.vue` → 同步 `AGENTS.md` / `CLAUDE.md` 的「N 个公共组件」
- 新增/删除 `backend/routes/` 下的 `.go` → 同步 routes 文件数 / routes 分组数
- 新增/删除 `backend/handlers/` 下的 `.go` → 同步 `CLAUDE.md` 的「N 个 handler 文件」
- 新增/删除 `backend/handlers/` 下的 `*_test.go` → 同步 `AGENTS.md` 的「测试（N）」
- 新增/删除 `ai_gateway*` / `minimax_*` / `planner*` / `nfsshare*` / `askit*` / `skills*` handler → 同步模块表对应行的 `(N 文件)` 与「AI 网关（N）」
- 新增/删除内置 skill → 同步 `AGENTS.md` 的「N 个 skill」
- 新增/删除模块表的一行 → 同步模块表编号、`AGENTS.md` 标题数字、全部「N 模块地图」
- 新增/删除 handler 分段（改 `// domain/sectionN` 横幅） → 同步模块表对应行的 `(N 分段)`
- 新增/删除 backend/routes/ 下的 *.go → 同步 AGENTS.md「后端路由拆分」枚举树 与「routes/ N 个文件」
- 新增/删除 backend/handlers/ 下的 *.go → 同步 AGENTS.md「Handler 文件分组」各分组的枚举
- 新增/删除 frontend/src/views/<cat>/ 下的 *.vue → 同步 AGENTS.md「View 组件」per-category 枚举
- 新增/删除 backend/handlers/ 下的模块（模块级） → 同步 AGENTS.md「N 模块地图」的模块行

**每次改完必跑**（防"改一半"）：

```bash
# 1) 现算（粘贴上面的命令块，`REF` 在那里定义）
# 2) 对照文档里的计数
# 3) 一致性硬闸门：标题数字 == 模块表行数
#    两行**必须同源**（都读工作区 AGENTS.md）：一读工作区、一读 ${REF}:AGENTS.md 时，
#    工作区有未提交改动就会让两个数来自不同版本 ⇒ 原理上假红。
grep -E '^## [0-9]+ 模块地图' AGENTS.md
grep -cE '^\| *[0-9]+ *\|' AGENTS.md    # 两个数必须相等（都读工作区 AGENTS.md）
```

**枚举闭合判据**（每个被枚举的清单都要能算出「枚举项数 == 声明的数」；没有判据的清单迟早腐）：

```bash
# ── 用法 ────────────────────────────────────────────────────────────────
# REF 同时决定「文档来源」与「文件树来源」，两者必须是**同一棵树**，否则数会跨版本 ⇒ 原理上假红。
#   REF=${REF:-origin/main}   # 默认（口径不变）：origin/main 的文档 + origin/main 的文件树
#   REF=$(git write-tree)     # 文档有【未提交】改动时用：write-tree 产出 tree 对象，git show <tree>:AGENTS.md 可读
#   REF=HEAD                  # 改自己的分支时用（改动已提交）
#
# 并发安全：所有中间文件写进 mktemp -d 出来的唯一目录。
# 中间文件用固定文件名会被并发 run 互撞 ⇒ 结果随机（E/N3 判据会拦住回退）。
REF=${REF:-origin/main}
DOC=$(git show ${REF}:AGENTS.md)
TMPD=$(mktemp -d /tmp/doc_gate.XXXXXX)
FAIL=$TMPD/fail.txt; : > "$FAIL"

# A) routes 枚举树：AGENTS.md 枚举的 .go 文件名集合 == backend/routes/ 实际文件名集合
printf '%s\n' "$DOC" | sed -n '/^### 后端路由拆分/,/^### /p' | grep -o '[a-z0-9_]*\.go' | sort -u > "$TMPD/enum_routes.txt"
git ls-tree -r --name-only ${REF} -- backend/routes/ | grep '\.go$' | xargs -n1 basename | sort -u > "$TMPD/real_routes.txt"
if diff -q "$TMPD/enum_routes.txt" "$TMPD/real_routes.txt" >/dev/null; then
  echo "A) routes 枚举闭合 ✅（$(wc -l < "$TMPD/enum_routes.txt") 项）"
else
  echo "A) routes 枚举不闭合 ❌"; diff "$TMPD/enum_routes.txt" "$TMPD/real_routes.txt"; echo "A routes 枚举不闭合" >> "$FAIL"
fi

# B1) Handler 分组（普通形态）：枚举项数 == 括号里的声明数，且每个枚举名在 backend/handlers/ 至少命中 1 个**非测试**文件
#     覆盖：核心工具 / 订阅/代理 / 家庭/健康 / 实时功能 / 内容创作 / 开发者 / 辅助
#     ⚠ 名字存在性必须先滤掉 *_test.go：否则枚举名能被测试文件「充数」（如 apigateway_proxy 只命中 *_test.go）
for g in 核心工具 订阅/代理 家庭/健康 实时功能 内容创作 开发者 辅助; do
  line=$(printf '%s\n' "$DOC" | grep -E "^├── ${g}（")
  if [ -z "$line" ]; then echo "B1) 分组「$g」在 AGENTS.md 里找不到 ❌"; echo "B1 分组 $g 行缺失" >> "$FAIL"; continue; fi
  items=$(printf '%s\n' "$line" | sed 's/.*：//' | tr ',' '\n' | sed 's/ //g' | grep -v '^$')
  n=$(printf '%s\n' "$items" | grep -c .)
  decl=$(printf '%s\n' "$line" | grep -oE '（[0-9]+）' | grep -oE '[0-9]+')
  if [ "$n" = "$decl" ]; then echo "B1) $g 枚举=$n 声明=$decl ✅"; else echo "B1) $g 枚举=$n 声明=$decl ❌ 不相等"; echo "B1 $g 枚举项数 != 声明数" >> "$FAIL"; fi
  printf '%s\n' "$items" | while read -r it; do
    c=$(git ls-tree -r --name-only ${REF} -- backend/handlers/ | grep '\.go$' | grep -v '_test\.go$' | grep -cE "/${it%\*}[^/]*\.go$")
    if [ "$c" -gt 0 ]; then echo "   ✅ $it → $c 个文件"; else echo "   ❌ $it 在 backend/handlers/ 下无对应非测试文件"; echo "B1 $g/$it 无非测试文件" >> "$FAIL"; fi
  done
done

# B2) Handler 分组「AI 网关」（复合子计数形态）：ai_gateway*（N）+ image_understanding（1）+ minimax_*（M）+ planner_ai（1） == 声明数
#     ⚠ helper（脚手架）个数随文件增减变化，文档里**不得写死**；N / M 一律由子命令实测，不看文档里的旧数。
ailine=$(printf '%s\n' "$DOC" | grep -E '^├── AI 网关（')
aidecl=$(printf '%s\n' "$ailine" | grep -oE 'AI 网关（[0-9]+' | grep -oE '[0-9]+')
aisub=$(printf '%s\n' "$ailine" | grep -oE 'ai_gateway\*（[0-9]+' | grep -oE '[0-9]+')
mmsub=$(printf '%s\n' "$ailine" | grep -oE 'minimax_\*（[0-9]+' | grep -oE '[0-9]+')
aireal=$(git ls-tree --name-only ${REF}:backend/handlers/ | grep '^ai_gateway' | grep -vc '_test\.go$')
mmreal=$(git ls-tree --name-only ${REF}:backend/handlers/ | grep '^minimax_' | grep -vc '_test\.go$')
iureal=$(git ls-tree --name-only ${REF}:backend/handlers/ | grep -cE '^image_understanding\.go$')
pareal=$(git ls-tree --name-only ${REF}:backend/handlers/ | grep -cE '^planner_ai\.go$')
if [ "$aisub" = "$aireal" ]; then echo "B2) ai_gateway* 子计数=$aisub 实测=$aireal ✅"; else echo "B2) ai_gateway* 子计数=$aisub 实测=$aireal ❌"; echo "B2 ai_gateway* 子计数 != 实测" >> "$FAIL"; fi
if [ "$mmsub" = "$mmreal" ]; then echo "B2) minimax_* 子计数=$mmsub 实测=$mmreal ✅"; else echo "B2) minimax_* 子计数=$mmsub 实测=$mmreal ❌"; echo "B2 minimax_* 子计数 != 实测" >> "$FAIL"; fi
sum=$((aireal + iureal + mmreal + pareal))
if [ "$sum" = "$aidecl" ]; then echo "B2) 复合求和=$sum 声明=$aidecl ✅"; else echo "B2) 复合求和=$sum 声明=$aidecl ❌"; echo "B2 AI 网关复合求和 != 声明数" >> "$FAIL"; fi
ailines=$(printf '%s\n' "$DOC" | grep -E '^(├── AI 网关（|\| 25 \| \*\*AIGateway\*\*)')
if printf '%s\n' "$ailines" | grep -qE '含 *[0-9]+ *个 *(test helper|测试脚手架)'; then
  echo "B2) ❌ 文档写死了 test helper 个数（helper 个数随文件增减变化，不得写死）"; echo "B2 AI 网关组写死 helper 个数" >> "$FAIL"
else
  echo "B2) test helper 个数未写死 ✅"
fi

# B3) Handler 分组「测试（N）」（glob 计数形态）：声明数 == backend/handlers/ 下 *_test.go 的实测数
tline=$(printf '%s\n' "$DOC" | grep -E '^└── 测试（')
tdecl=$(printf '%s\n' "$tline" | grep -oE '（[0-9]+）' | grep -oE '[0-9]+')
treal=$(git ls-tree -r --name-only ${REF} -- backend/handlers/ | grep '\.go$' | grep -c '_test\.go$')
if [ "$tdecl" = "$treal" ]; then echo "B3) 测试组 声明=$tdecl 实测=$treal ✅"; else echo "B3) 测试组 声明=$tdecl 实测=$treal ❌"; echo "B3 测试组 声明数 != 实测 _test.go 数" >> "$FAIL"; fi

# C) per-category View 枚举：已列名数 + 「另 N 个」== 括号里的声明数 == 实测 .vue 数；每个已列名真实存在
#    截断形态（行尾 `...`）：`...` 是「后面还有」的标记，不计入已列名数；
#    没写「另 N 个」时，隐含的「另 N 个」= 声明数 − 已列名数（与 dev/ 的「另 N 个」写法同口径）。
#    没有名字清单的 category（home / household / share）：只校验「声明数 == 实测数」，不校验名字。
for d in ai collab convert dev draw home household life other share; do
  line=$(printf '%s\n' "$DOC" | grep -E "^(├──|└──) ${d}/")
  if [ -z "$line" ]; then echo "C) $d 在 AGENTS.md 里找不到 ❌"; echo "C $d 行缺失" >> "$FAIL"; continue; fi
  decl=$(printf '%s\n' "$line" | grep -oE '（[0-9]+' | grep -oE '[0-9]+')
  real=$(git ls-tree -r --name-only ${REF} -- "frontend/src/views/$d" | grep -c '\.vue$')
  if [ -z "$decl" ]; then echo "C) $d 无声明数（实测=$real）❌ 没有声明数就没得比"; echo "C $d 缺声明数" >> "$FAIL"; continue; fi
  rest=$(printf '%s\n' "$line" | grep -oE '另 [0-9]+ 个' | grep -oE '[0-9]+'); rest=${rest:-0}
  case "$line" in
    *：*) names=$(printf '%s\n' "$line" | sed 's/.*：//' | tr ',' '\n' | sed 's/ //g' | grep -v '^$' | grep -vE '^另[0-9]+个$' | grep -vE '^\.\.\.$') ;;
    *)    names="" ;;
  esac
  listed=$(printf '%s\n' "$names" | grep -c .)
  if [ "$listed" -eq 0 ]; then
    if [ "$decl" = "$real" ]; then echo "C) $d 声明=$decl 真实=$real ✅（无名字清单，只校验计数）"; else echo "C) $d 声明=$decl 真实=$real ❌"; echo "C $d 声明数 != 实测数" >> "$FAIL"; fi
    continue
  fi
  if printf '%s\n' "$line" | grep -q '\.\.\.' && [ "$rest" -eq 0 ] && [ "$listed" -lt "$decl" ]; then rest=$((decl - listed)); fi
  if [ "$((listed + rest))" = "$decl" ] && [ "$decl" = "$real" ]; then
    echo "C) $d 声明=$decl 已列=$listed 另=$rest 合计=$((listed + rest)) 真实=$real ✅"
  else
    echo "C) $d 声明=$decl 已列=$listed 另=$rest 合计=$((listed + rest)) 真实=$real ❌"; echo "C $d 枚举数不闭合" >> "$FAIL"
  fi
  # 名字存在性：精确命中 <name>.vue，或唯一命中 <name>*.vue（命中数必须 == 1）
  #   convert/ draw/ 的枚举是短名（Base64 → Base64Tool.vue）；唯一性可断言 ⇒ 判据不会因短名恒红。
  printf '%s\n' "$names" | while read -r n; do
    [ -z "$n" ] && continue
    exact=$(git ls-tree -r --name-only ${REF} -- "frontend/src/views/$d" | grep -cE "/${n}\.vue$")
    glob=$(git ls-tree -r --name-only ${REF} -- "frontend/src/views/$d" | grep -cE "/${n}[^/]*\.vue$")
    if [ "$exact" -eq 1 ] || [ "$glob" -eq 1 ]; then
      echo "   ✅ $d/$n（精确 $exact / 唯一前缀 $glob）"
    else
      echo "   ❌ $d/$n 不存在或前缀不唯一（精确 $exact / 前缀 $glob）"; echo "C $d/$n 不存在或不唯一" >> "$FAIL"
    fi
  done
done

# D) 模块表：行数 == 标题数字
rowN=$(printf '%s\n' "$DOC" | grep -cE '^\| *[0-9]+ *\|')
titleN=$(printf '%s\n' "$DOC" | grep -E '^## [0-9]+ 模块地图' | grep -oE '[0-9]+')
if [ "$rowN" = "$titleN" ]; then echo "D) 模块表行数=$rowN 标题=$titleN ✅"; else echo "D) 模块表行数=$rowN 标题=$titleN ❌"; echo "D 模块表行数 != 标题数字" >> "$FAIL"; fi

# E) 文本判据（N1 / N3 / N4）：必须**同时**读 ${REF}:AGENTS.md 与 ${REF}:CLAUDE.md，
#    否则这三条只能看出「工作区」的措辞、在 ${REF} 上做不了反事实 ⇒ 恒真闸门。
CDOC=$(git show ${REF}:CLAUDE.md)

# N1 · AGENTS.md 不得写死 test helper / 测试脚手架 的个数，且必须给出「个数不写死」口径
n1hard=$(printf '%s\n' "$DOC" | grep -cE '含 *[0-9]+ *个 *(test helper|测试脚手架)')
n1note=$(printf '%s\n' "$DOC" | grep -c '个数不写死')
if [ "$n1hard" -eq 0 ] && [ "$n1note" -ge 1 ]; then
  echo "E/N1) helper 个数写死=$n1hard（应 0） 口径说明=$n1note（应 ≥1） ✅"
else
  echo "E/N1) helper 个数写死=$n1hard（应 0） 口径说明=$n1note（应 ≥1） ❌"; echo "E N1 helper 个数写死或缺口径说明" >> "$FAIL"
fi

# N3 · 闸门的中间文件不得用固定路径，必须走 mktemp -d 的唯一目录
#      ⚠ 本判据读的就是 CLAUDE.md 自己 ⇒ 模式写成 `rout[e]s`，让判据自身的文本不命中自己（否则恒绿、判据失效）
n3fixed=$(printf '%s\n' "$CDOC" | grep -cE '/tmp/(enum_rout[e]s|real_rout[e]s)\.txt')
n3mktemp=$(printf '%s\n' "$CDOC" | grep -cE 'mktemp -d')
if [ "$n3fixed" -eq 0 ] && [ "$n3mktemp" -ge 1 ]; then
  echo "E/N3) 固定中间文件名=$n3fixed（应 0） mktemp -d=$n3mktemp（应 ≥1） ✅"
else
  echo "E/N3) 固定中间文件名=$n3fixed（应 0） mktemp -d=$n3mktemp（应 ≥1） ❌"; echo "E N3 闸门仍用固定中间文件名或缺 mktemp -d" >> "$FAIL"
fi

# N4 · 文档必须说明「文档有未提交改动」时怎么跑（REF=<git write-tree 产出的 tree>）
#      ⚠ 同 N3：模式写成 `write-tre[e]`，判据自身不算数，只认用法说明那几行
n4=$(printf '%s\n' "$CDOC" | grep -cE 'git write-tre[e]')
if [ "$n4" -ge 1 ]; then
  echo "E/N4) 未提交改动的跑法（git write-tree）出现 $n4 次 ✅"
else
  echo "E/N4) 文档未说明未提交改动怎么跑（git write-tree 出现 0 次）❌"; echo "E N4 缺 git write-tree 用法" >> "$FAIL"
fi

# 汇总：有任一条不闭合 ⇒ 退出码 1（可直接当闸门用；交互 shell 里请写成脚本再跑）
if [ -s "$FAIL" ]; then echo "枚举闭合判据未通过 ❌"; cat "$FAIL"; exit 1; fi
echo "枚举闭合判据全部通过 ✅"
```

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
