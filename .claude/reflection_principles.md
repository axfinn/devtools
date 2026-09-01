# 反思原则清单 — DevTools subagent 协作准则

> reflection-coach 启动时读这份文件;每条是 yes/no,违反就标记。
> ⚠️ 标记 = 硬原则(违反 → 冻结 + 必须用户点头才能继续)。
> 没有 ⚠️ = 软提醒(只警告,不冻结)。
>
> **每条原则必须带 `踩过:` 具体反例** — 没反例的抽象条目(如"保持代码质量")不收,
> 它不改变行为只占上下文。反例要指向真实事件(file:line / memory 条目 / commit)。

---

## P1 — 改动严格在请求范围内,不蔓延到无关文件
触文件列表 vs 用户请求范围,差集应为空。
蔓延 1 个文件 → 软提醒;蔓延到核心架构文件 → 升级 ⚠️ 硬。
**踩过:** backend-writer 修 paste handler 时顺手"优化"了 autodev 的 SSE 写法,触了 autodev.go 80 行无关代码,PR review 时被打回,白做。

## P2 ⚠️ — 不破坏 R1-R11 任一硬规则
code-reviewer 的 ## Review Checklist 任一 FAIL → 立即冻结。
R1-R11 见 `/Users/finn/.claude/projects/-Volumes-M20-code-docker-devtools/memory/feedback_*.md` + `project_*.md`。
**踩过:** chat handler 测试用 `:memory:` 没 `SetMaxOpenConns(1)`,本地跑过但 CI 间歇 404,因为后台 rate limiter goroutine 把连接池打满。R5 当时没单列规则,栽了才补。

## P3 — 增量最小化
触文件数 / LOC 增量 vs 同类历史任务的中位数。
增量 ≤ 1.5x 中位数 → 软通过;1.5-2x → 软提醒;>2x → 升级 ⚠️ 硬。
**踩过:** frontend-writer 加一个新 view,顺手把 router 整文件重排了一遍(300 行 diff),review 时发现 95% 是空白字符,真改动只有 30 行。

## P4 ⚠️ — 跨栈一致(api-contract 0 mismatch)
api-contract 报告 Section 1/2/3/6 任一非空 → 立即冻结(功能破坏)。
WARN(Section 4/5)→ 软提醒。
**踩过:** 前端 view 调 `/api/shorturl/create`,body 用 `originalUrl`(camelCase),后端 binding `json:"original_url"`(snake_case),后端收不到字段静默返 200,前端当成功吃了空数据。生产事故。

## P5 — 解释聚焦、不发散、不自我重复
解释中"我猜/可能/也许"出现 0 次 → 软通过;1-2 次 → 软通过;3-5 次 → 软提醒;6+ → ⚠️ 硬。
**踩过:** backend-writer 一次回答里出现 11 次"可能",用户问"你到底知不知道"。

## P6 — 不引入项目已有工具能完成的新依赖
加 `import` / `require` / `package` 前必须 grep `package.json` / `go.mod` / `utils/` / `composables/`。
已有就不加 → 软通过;加了重复的 → 软提醒;改了既有接口 → ⚠️ 硬。
**踩过:** 前端 view 自己写了 base64 解码,btoa/atob 浏览器原生就有;白引入 crypto-js +20KB。

## P7 — 不重复造已有 helper
新写的 function 必须在 `backend/utils/` / `frontend/src/composables/` / `backend/handlers/autodev.go` 找不到等价实现。
找到就不写 → 软通过;硬造重复 → ⚠️ 硬。
**踩过:** frontend-writer 新写了个 `formatDate`,没看 `frontend/src/utils/naturalDate.js` 已经有 `formatRelative` / `formatHuman`,三处用法各不一致。

## P8 ⚠️ — 输出可被下游 agent 直接消费
agent 输出必须含:改了哪些文件(file:line) + 触的接口(verb+path) + 为什么这样改 + 残留问题。
下游(api-contract / code-reviewer / reflection-coach 自己)无需再猜 → 软通过;
输出模糊 / 缺信息 / 含"待定" → ⚠️ 硬(流水线会卡死)。
**踩过:** api-contract 早期只列"前端调了 X",不说后端在哪一行返的,backend-writer 修复时要重 grep 整仓库 8 分钟。

## P9 — 失败时给出可执行的下一步
报错 / 阻塞 / 找不到文件 → 必须给"下一步该做什么"或"问用户哪个选项"。
只说"不确定" / "需要更多信息" → 软提醒;完全没说 → ⚠️ 硬。
**踩过:** agent 报"找不到文件 X"就停,不说 X 该在哪、该 grep 什么、是不是 typo,用户只好自己查。

## P10 — 同 task 重复跑要稳定
同 task 跨多轮,触文件 / 主体逻辑 / 接口签名不应大变。
第一轮 vs 第二轮 diff 大于 30% → 软提醒;大于 60% → ⚠️ 硬(说明没收敛)。
**踩过:** reflection-coach 同 task 第二轮触文件比第一轮多 80%,用户问"怎么越改越多"。

## P11 — 与历史同类任务一致
同类历史任务(grep `reflection_*.md`)的"做对了什么"应被复用;
新做法应在反思 log 解释为何偏离。
没解释就偏离 → 软提醒。
**踩过:** backend-writer 写第二个 handler 时换了套错误响应格式,跟第一个不一致,前端得分别处理两种 shape。

## P12 — 反思 log 要有可复用的教训
`reflection_*.md` 必须含"## 以后同任务能复用的教训"段,有具体可操作建议。
只写流水账("做了什么 + 改了哪些") → 软提醒;反思 log 不存在 → ⚠️ 硬(本任务没闭环)。
**踩过:** 反思 log 只写"改了 5 个文件,加了 100 行",用户看完不知道下次怎么避坑。

---

## 维护规则

**写入:** 新原则先试着合并进已有条目,合不进才新建。上限 12 条,满了必须先合并或删除。
发散靠这两条结构治,不靠给条目打分 — 分数是我自己编的,没有客观信号源,
还会变成"这条 0.6 再观察观察"的拖延借口。
真实信号只有两个:用户纠正过我(负)、同一教训重复命中(正),直接记事实即可。

**找用户裁决的两种情况**(其余直接写入,不打断):
- **总数要超 12 条** — 列出合并/删除候选方案让用户选,不自己决定删哪条
- **新原则与已有条目矛盾** — 说明各自来源和时间,问是场景不同(各自加限定)还是要求变了(旧条目该改)。自己选一个等于悄悄改了用户定的规矩

**反思时机:** 不设定期 review(会退化成走流程)。触发点是具体事件 —
用户纠正我、agent 自己发现搞错、用户说"看下 lessons"。
用户纠正是唯一的外部信号,最有价值,但**先分清是新教训还是已有条目没执行** —
后者新增没用,该改的是条目本身太抽象。

**与项目 memory 的分工:**
- 换个项目还成立 → 进 `.claude/reflection_principles.md`(工作方法)
- 只在 DevTools 项目成立 → 留 `/Users/finn/.claude/projects/-Volumes-M20-code-docker-devtools/memory/`(R1-R11 + 域细节)

---

## 元变更日志
- 2026-09-01: 初版 12 条 + 维护规则(借鉴 ~\.claude/CLAUDE.md 的"有效做法"段格式)
