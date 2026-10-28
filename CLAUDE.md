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
├── handlers/            # 130 个 handler 文件（按域分组）
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
REF=${REF:-origin/main}
DOC=$(git show ${REF}:AGENTS.md)
FAIL=/tmp/fin_doc_fail_$$.txt; : > "$FAIL"

# A) routes 枚举树：AGENTS.md 枚举的 .go 文件名集合 == backend/routes/ 实际文件名集合
printf '%s\n' "$DOC" | sed -n '/^### 后端路由拆分/,/^### /p' | grep -o '[a-z0-9_]*\.go' | sort -u > /tmp/enum_routes.txt
git ls-tree -r --name-only ${REF} -- backend/routes/ | grep '\.go$' | xargs -n1 basename | sort -u > /tmp/real_routes.txt
if diff -q /tmp/enum_routes.txt /tmp/real_routes.txt >/dev/null; then
  echo "A) routes 枚举闭合 ✅（$(wc -l < /tmp/enum_routes.txt) 项）"
else
  echo "A) routes 枚举不闭合 ❌"; diff /tmp/enum_routes.txt /tmp/real_routes.txt; echo "A routes 枚举不闭合" >> "$FAIL"
fi

# B) Handler 分组「辅助（N）」：枚举项数 == 括号里的数，且每项在 backend/handlers/ 至少命中 1 个文件
auxline=$(printf '%s\n' "$DOC" | grep -E '^├── 辅助（')
aux=$(printf '%s\n' "$auxline" | sed 's/.*：//' | tr ',' '\n' | sed 's/ //g' | grep -v '^$')
auxn=$(printf '%s\n' "$aux" | grep -c .)
auxdecl=$(printf '%s\n' "$auxline" | grep -oE '[（(][0-9]+[）)]' | grep -oE '[0-9]+')
if [ "$auxn" = "$auxdecl" ]; then echo "B) 辅助 枚举=$auxn 声明=$auxdecl ✅"; else echo "B) 辅助 枚举=$auxn 声明=$auxdecl ❌ 不相等"; echo "B 辅助组 枚举项数 != 声明数" >> "$FAIL"; fi
printf '%s\n' "$aux" | while read -r n; do
  c=$(git ls-tree -r --name-only ${REF} -- backend/handlers/ | grep -cE "/${n%\*}[^/]*\.go$")
  if [ "$c" -gt 0 ]; then echo "   ✅ $n → $c 个文件"; else echo "   ❌ $n 在 backend/handlers/ 下无对应文件"; echo "B 辅助组 $n 无对应文件" >> "$FAIL"; fi
done

# C) per-category View 枚举：已列名数 + 「另 N 个」== 括号里的声明数，且每个名字真实存在
for d in ai dev life other; do
  line=$(printf '%s\n' "$DOC" | grep -E "^├── $d/")
  decl=$(printf '%s\n' "$line" | grep -oE '[（(][0-9]+[）)]' | grep -oE '[0-9]+')
  rest=$(printf '%s\n' "$line" | grep -oE '另 [0-9]+ 个' | grep -oE '[0-9]+'); rest=${rest:-0}
  names=$(printf '%s\n' "$line" | sed 's/.*：//' | tr ',' '\n' | sed 's/ //g' | grep -v '^$' | grep -vE '^另[0-9]+个$')
  listed=$(printf '%s\n' "$names" | grep -c .)
  real=$(git ls-tree -r --name-only ${REF} -- "frontend/src/views/$d" | grep -c '\.vue$')
  if [ "$decl" = "$((listed + rest))" ] && [ "$decl" = "$real" ]; then
    echo "C) $d 声明=$decl 已列=$listed 另=$rest 合计=$((listed + rest)) 真实=$real ✅"
  else
    echo "C) $d 声明=$decl 已列=$listed 另=$rest 合计=$((listed + rest)) 真实=$real ❌"; echo "C $d 枚举数不闭合" >> "$FAIL"
  fi
  printf '%s\n' "$names" | while read -r n; do
    git ls-tree -r --name-only ${REF} -- "frontend/src/views/$d" | grep -q "/${n}\.vue$" || { echo "   ❌ $d/$n 不存在"; echo "C $d/$n 不存在" >> "$FAIL"; }
  done
done

# D) 模块表：行数 == 标题数字
rowN=$(printf '%s\n' "$DOC" | grep -cE '^\| *[0-9]+ *\|')
titleN=$(printf '%s\n' "$DOC" | grep -E '^## [0-9]+ 模块地图' | grep -oE '[0-9]+')
if [ "$rowN" = "$titleN" ]; then echo "D) 模块表行数=$rowN 标题=$titleN ✅"; else echo "D) 模块表行数=$rowN 标题=$titleN ❌"; echo "D 模块表行数 != 标题数字" >> "$FAIL"; fi

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
