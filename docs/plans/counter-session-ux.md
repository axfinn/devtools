# 敲击计数器：移动端适配 / 息屏音频恢复 / 运动会话与目标 / 防误触 — 技术方案（v4 修订版）

- 原 issue：FINN-10（父 FINN-9）；v2 修订：FINN-16（Stage 2 评审 FINN-11 判 FAIL，7 条阻塞项）；**v3 修订：FINN-18（Stage 1 复查新增 B8 / B9 两条阻塞项 + 非阻塞 1–7 + 未定义边界 1–3）**；**v4 修订：FINN-20（Stage 1 复查唯一剩余阻塞项 B10 —— 训练层存活时 `undoLast()` 经 `document` 级快捷键可达）**
- 模块：`frontend/src/views/life/CounterTool.vue`（纯前端，无后端）
- 状态：设计稿，**不含任何代码改动**；实现由 frontend-writer 执行，验收由 devtools-qa 执行
- 勘察方式：全部结论来自本仓库实际 grep / 逐段 Read；**未起 dev server、未跑 go build、未在真机验证**。凡属推算的数值均在文中显式标注「推算」

> **v3 的边界（写给下游）**：B1–B7 已由 FINN-17 复查判定**全部闭合**，本版**不推翻方案骨架、不重开 B1–B7**。v3 只做三类改动：① 修 B8（训练层层级）/ B9（`overflow` 自相矛盾）两条新阻塞项；② 收口非阻塞 1–7 与未定义边界 1–3；③ 补 v2 遗漏的两个副作用（EP 全局弹层在训练层下的取舍、`active.taps` 写入时机）。

---

## v4 修订记录（对照 FINN-20 / B10）

> **v4 的范围（写给下游）**：FINN-19 复查判定 B1–B9、非阻塞 1–7、未定义边界 1–3 **全部闭合**，**唯一剩余阻塞项 = B10**。v4 因此**只改 B10 这一处事实错误 + 由它引出的同口径自查**：不推翻骨架、不重开任何已闭合条目。v3 的所有结论在 v4 中原样有效。

| 项 | 改了什么 | 主要落点 |
|----|---------|---------|
| **B10** | **删除未实证前提**「训练层内撤销本就不可达」→ 更正为「训练层存活时 `undoLast()` **可**经 `document` 级 `onKeyDown` 的 `Cmd/Ctrl+Z` 分支到达」，后果是「训练中静默改数 + `ElMessage` 反馈不可见」；给出**训练层存活期的撤销门限**（**屏蔽该快捷键**，理由见 §D2⑦-b.1）；补可自动化判据与静态 grep 判据；同步风险表 | §D2⑦、§D2⑦-b（新增）、§测试要点 L1 #6b/#6c、§测试要点 L2 #42c、风险表第 10 / 12 行 |
| **B10 同口径自查** | 新增 §D2⑦-b：**逐条实证**训练层存活期全部 `document` / `window` 级入口（键盘快捷键、`visibilitychange`、`pageshow`、`resize`、`beforeunload`/`pagehide`、全局错误处理、跨模块 keep-alive 监听残留），每条给出**可达性结论 + 处置**；不再出现任何「某入口不可达」的未实证表述 | §D2⑦-b、风险表第 12–15 行、§测试要点 L1 #6b/#6c、L2 #42c、L3 #81b/#81c |
| **B10 顺带实证的 2 条新入口** | ① 键盘 `ArrowDown`/`Minus`/`Backspace` → `decrement()` 在训练层存活时可改计数；② 其它 keep-alive 视图（`/games`、`/planner`）注册的 `window` keydown 未在 `onDeactivated` 摘除，训练中按 `Cmd/Ctrl+K` / `Enter` 会打开**训练层下方**的 EP 弹层并抢占焦点（键盘计数静默失效） | §D2⑦-b.3 第 2 / 12 行、风险表第 14 行 |

> **本版新增的闸门**：§测试要点 **L1 #6b / #6c**、**L2 #42c** 是 B10 的防回归判据（自动化主判据是 L1，L2 只作路径核实，不做假自动化）；**L3 新增前置条件观察项**：先访问 `/games` 与 `/planner`，再进 `/counter` 训练，按 `Cmd/Ctrl+K` / `Enter` 记录后果。

---

## v3 修订记录（对照 FINN-18 要求）

| 项 | 改了什么 | 主要落点 |
|----|---------|---------|
| **B8** | 训练层 `z-index` 从 **900 → 10050**（**全局唯一高于所有常驻 fixed 浮层的取值**，已 grep 核实全仓库最大常驻值为 10000）；把「全局 fixed 浮层」当**一类**处理而非逐个打补丁；§D2⑦ 重写为「训练层**盖住**一切（含 EP 弹层）→ 层内不得调用任何 EP 全局弹层 API」；补 L2 正向/负向断言；L3 可见性判据从 `bottom ≤ innerHeight` 改为 `elementFromPoint` 命中测试 | §D2②、§D2⑦、§D2⑧（新增）、§测试要点 L2/L3、风险表第 5 行 |
| **B9** | 移动断点下**删除 `overflow: visible` 覆盖**，改为不声明 `overflow` → 继承基类 `overflow: hidden`；补「为什么这不会重新引入内层滚动容器阻断滚动链」的依据句 | §A2、§A5 |
| 非阻塞 1 | §C1.3 跨零点新段的 `countAtStart` 从「取当前 `todayCount`」更正为 **`countAtStart = 0`（清零后）** | §C1.3 |
| 非阻塞 2 | 明确定义**回前台两阶段顺序**：先按 §C1.4 结算 idle、**再**判断是否做 midnight 拆分；新增 `lastHiddenAtMs` / `pendingSplitFrom` 两个 store 级字段，用来覆盖「后台 1s 时钟已自行拆分」的路径 | §C1.4、§C1.8、§C1 数据模型 |
| 非阻塞 3 | §A3 首屏预算改为**逐项明细**（v2 只有聚合值，无法核对）：面板上下 padding = **22px**（≤420 覆盖值；FINN-18 提示的 26px 是 ≤720 值，在 375/320 两个目标视口下**不生效**），补上此前未单列的 `.step-strip` 的 12+40+2；敲击面由 ≈340 / ≈241 重算为 **≈346 / ≈247** | §A3 |
| 非阻塞 4 | 断点统一为 **980 / 767.98 / 420** 三个 + 横屏一条；**撤回 v2 的 420→480 改动**（无需求支撑且会把 421–480 区间的既有样式全部改掉）；`≤360px` 降级阶梯**不写进代码**，只作应急备案 | §A1、§A3 |
| 非阻塞 5 | §A7 的 safe-area 片段**加 `@media (min-width: 768px)` 包裹**，避免盖掉移动端 §A2 的 22px 预算 | §A7 |
| 非阻塞 6 | L2 #42 改为「先取 `unlock()` / `play()` 的函数行区间，再核对命中行号落在区间内」的人工判定，并注明自动化主判据是 L1 #32/#33/#34 | §测试要点 L2 |
| 非阻塞 7 | §C2 落盘时机枚举补第 ⑥ 项「`clearHistory()`（用户清空历史）」 | §C2 |
| 边界 1 | `undoLast()` 与 `active.taps` 的同步：**撤销时同步弹出末条**；把 `taps` 的写入时机从 `registerHit()` 下移到 `applyDelta()` 的 `actualDelta !== 0` 分支，条目加 `kind` 字段，使「`taps` 末条 `delta === lastActionDelta`」成为**不变量** | §C1.7、§D2⑥ |
| 边界 2 | 常驻浮层（`.pet-window` / `.player-bar`）在**日间模式**的遮挡：给出明确产品立场（接受为已知残留 + L3 双前置条件），并说明为什么不改 | §未定义边界、§A5、§测试要点 L3 |
| 边界 3 | L3 布局档位新增**前置条件维度**：「本会话播放过媒体（播放器栏开/关）」×「宠物窗口开/关」，同一视口跑两遍 | §测试要点 L3 |
| **v2 遗漏副作用** | 训练层若高于 EP 弹层，则 `ElMessageBox` / `ElMessage` **也会被盖住** → 「结束训练」的二次确认必须**层内自绘**，训练层存活期间禁止调用任何 EP 全局弹层 API | §D2②、§D2⑦、硬约束 |

> **本版新增的负向验证**：§测试要点 L2 #42 改写、**#42b / #45b / #46b（v3 新增）**、**L1 #25a–#25e（v3 新增）** 是 B8 / B9 / 边界 1 / 非阻塞 2 的防回归闸门。

---

## v2 修订记录（对照 FINN-16 要求）

| 项 | 改了什么 | 主要落点 |
|----|---------|---------|
| **B1** | 删除计数面的「按下→抬起提交」与 12px / 1200ms / `isPrimary` / 60ms 去抖四道门限；改为 **`pointerdown` 一击即计**。`shouldCommitTap` 整套常量删除，防误触只剩「结构上无可误触」+「破坏性操作长按」。补 `pointercancel` 口径 | §D2①、§测试要点 L1/L3 |
| **B2** | `visibilitychange → visible` 与 `pageshow` **只置 `needsUnlock=true` 并写 `lastHiddenAtMs`**，不调 `resume()` / `close()` / 不新建 ctx。恢复只允许在 `unlock()` 与 `play()` 内 | §B4 |
| **B3** | 补勘 `App.vue:1152-1160`（≤768 覆盖）；据此重述高度来源，**移动端不再依赖百分比解析**，改显式 `calc(100dvh - 71px)` 算式；说明文档层滚动与 `overscroll-behavior` / `.app-container{overflow:visible}` 的交互 | §A2 |
| **B4** | 给出**日间模式**（不进训练层）375×667 / 320×568 首屏完整可见的具体方案（`.tap-panel` 首屏高度预算 + 移除 `.figure-button` 固定下限），并补对应 L3 验收 | §A3、§测试要点 L3 |
| **B5** | 新增**会话级有上限敲击日志** `active.taps`（上限 500、FIFO 截断、`tapsDropped` 计数、**不落盘**），时间轴数据源从 15s 窗口的 `tapEvents` 改到它 | §C1.7、§D2⑥ |
| **B6** | 会话目标默认值改为**独立的按次目标**（`0` = 自由训练 / 预设 100·200·500 / 上次目标）；`dailyGoal` 只做只读参考展示，**禁止参与任何默认值计算** | §C1.5、§未决 #2 |
| **B7** | `resetAll()` 追加清 `counter_v4_training`；`clearHistory()` 一并清会话历史；`resetToday()` / 清除全部数据 与会话的交互（结束 or 继续）逐条定义 | §D3、§兼容性与迁移 |
| 非阻塞 1–9 | 降级阶梯接线写死、`endReason` 取值域收敛、跨零点实现顺序、keep-alive 监听摘除、`.tap-panel` 最终生效值、L2 #26 改写、挂载点与裁剪约束、`MiniMaxStudioTool.vue:908→907`、层内不可滚的处理 | 分散见各节 |
| 未定义边界 | 多标签页、`el-dialog` 遗留浮层、`durationMs` 派生统计口径 | §D3、§未定义边界 |
| 加分项 | Wake Lock 从「不做」提为**可选实现项**（默认训练时开启、失败静默降级） | §D4、§未决 #7 |

> **本版新增的负向验证（下游必做）**：见 §测试要点 L2 的 5 条 `grep` 负向断言。它们是 B1/B2/B5/B6 的**防回归闸门**——如果实现方把被否决的设计偷偷加回来，静态检查就会红。

---

## 目标与非目标

### 目标（严格对应需求四条）

| # | 需求 | 本方案的产出 |
|---|------|-------------|
| 1 | Bug：移动端页面展示不能适应窗口（jaxiu 已确认就是 `https://t.jaxiu.cn/counter` 的**默认视图**） | 拆掉「视口高度硬编码」结构；**日间模式**在 320×568 / 375×667 首屏完整可见「今日计数 + 主按钮 + 加减号」；训练层首屏完整可见；补齐 safe-area |
| 2 | Bug：息屏后再点击页面没有声音 | 音频生命周期重写：`interrupted` 状态、`resume()` Promise 必须 await、ctx 被回收后的自愈、回前台只置标志不裸调 resume、失败降级与用户可见反馈 |
| 3 | 功能优化：运动场景（计数口径 / 实际开始结束时间 / 目标） | 新增「训练会话」数据模型 + 独立存储键；会话统计与日统计**隔离**；**按每次运动设目标**，与每日 `dailyGoal` 彻底分离 |
| 4 | 交互要求：方便操作、不能误操作 | 全屏训练面（锁 UI 不锁计数）+ **一击即计** + 训练期禁用/降级清单 |

### 非目标（本次明确不做）

- **不加后端**：无新 handler / 路由 / SQLite 表 / Redis 使用，全模块保持 0 网络请求（见「R2 结论」）
- **不做跨设备同步**、不做账号、不做云端历史（评审已判 PASS，不得借修订扩范围）
- **不做传感器 / 麦克风自动计数**（jaxiu 2026-09-16 裁决：运动时计数 = 继续手敲）
- **不改现有日统计口径**：`dailyRecords` / `streak` / `weekData` / `calCells` / `grandTotal` / `bestSpeed` 语义与计算方式一律不动
- **不改音效合成算法**（`buildMokugyo` / `buildBell` / `buildDrum` / `buildChime` 波形代码不动），只改「什么时候能响」
- **不改形象 / 主题预设体系**，不新增预设
- **不引入新 npm 依赖**（不引 Howler 等；沿用 WebAudio 手写实现）
- **不改 App.vue 外壳**（路由/侧边栏/移动端 header 一律不动；见「未决问题 #11」）
- **不做 PWA / Service Worker / 后台播放**：本工具是前台工具，锁屏期间不产生敲击
- **不做多标签页同步**：明确定义为「不支持，后写覆盖」（见 §未定义边界）

---

## 现状（本仓库已核实）

> 所有 `file:line` 均经 grep/Read 实测确认（对齐 `origin/main` = `1d2cb9b`），非需求描述转抄。凡本版新勘或更正的行号在表中标注 **【v2 新勘】** / **【v2 更正】**。

### 0. 模块边界

- 路由：`frontend/src/router/index.js:591` → `path: '/counter'`，`name: 'Counter'`，**无 `hideSidebar`**（对比：只有分享类路由带 `hideSidebar`，见 `frontend/src/router/index.js:629/635/641/647/653/659/665`）
- 唯一实现文件：`frontend/src/views/life/CounterTool.vue`，2806 行（模板 1–440，脚本 442–1565，样式 1567–2806）
- `grep -ril counter backend --include="*.go"` → **无命中**。确认纯前端
- 无既有测试：`find frontend/src -name "*Counter*"` 只命中 `CounterTool.vue` 本身
- `frontend/src/views/life/` 引用组件的既有惯例是相对路径：`frontend/src/views/ai/MiniMaxStudioTool.vue:907` → `import AIGatewaySpeechPanel from '../../components/AIGatewaySpeechPanel.vue'` **【v2 更正：原方案写 `:908`，实为 `:907`】**
- `MiniMaxStudioTool.vue:905` 已 import `onActivated / onDeactivated / onBeforeUnmount`，并在 `:1237 / :1241 / :1245` 使用——**keep-alive 生命周期的既有写法**，本方案沿用
- 可用的 composable 目录 `frontend/src/composables/`（含 `useTheme.js`、`useAdminAuth.test.js`——**测试文件与源码同目录并列**是本仓库惯例）；组件目录 `frontend/src/components/`

### 1. 移动端适配现状（→ 需求 1）

| 事实 | 位置 |
|------|------|
| `.counter-app { min-height: 100vh; min-height: 100dvh; padding: 24px 16px 40px; overflow: hidden }` | `CounterTool.vue:1568-1576` |
| 只有 ≤720px 才补 safe-area，且顶部写的是 `max(8px, env(safe-area-inset-top))` | `CounterTool.vue:2410-2411` |
| ≤720px：`.counter-app { overflow: auto; overscroll-behavior-y: contain }`（内层滚动容器） | `CounterTool.vue:2412-2413` |
| ≤720px：`.tap-panel { min-height: calc(100dvh - 26px - env(safe-area-inset-top) - env(safe-area-inset-bottom)); display: flex; flex-direction: column }` | `CounterTool.vue:2451-2456` |
| ≤720px：`.tap-stage { position: relative; display: block; flex: 1; min-height: 0; margin-top: 10px }` | `CounterTool.vue:2547-2552` |
| ≤720px：`.figure-button { width: 100%; height: 100%; min-height: 360px }`；≤420px：`min-height: 330px` | `CounterTool.vue:2571-2575` / `2774-2776` |
| ≤720px：`.side-round` 绝对定位 `bottom:14px; left:14px`，56px；≤420px `bottom:12px; left:12px`，52px | `CounterTool.vue:2555-2564` / `2790-2796` |
| ≤420px：`.counter-app { padding-left: 6px; padding-right: 6px }`（会覆盖上面那条 padding 的左右值） | `CounterTool.vue:2752-2755` **【v2 新勘】** |
| 响应式断点仅 3 个：980 / 720 / 420 | `CounterTool.vue:2400 / 2409 / 2751` |
| 外壳：`.main-content { height:100dvh; overflow-y:auto; padding:20px; flex:1 }`（桌面） | `App.vue:1032-1042` |
| 外壳：`.app-container { height:100dvh; overflow:hidden }` | `App.vue:548-553` |
| 外壳移动端：`isMobile` 时套 `.mobile-main { padding:15px; padding-top: calc(56px + 15px); padding-bottom: 100px; height:auto; min-height: calc(100dvh - 56px) }` | `App.vue:190` / `App.vue:1044-1051` |
| **外壳 ≤768px 覆盖：`.app-container { flex-direction: column; overflow: visible }` + `.main-content { height: auto; overflow-y: visible }`** | **`App.vue:1152-1160`【v2 新勘 —— 原方案漏勘，见 B3】** |
| `isMobile` 阈值 = **768px**，判定式是 **`width < 768`**（严格小于，不是 `≤`） | `App.vue:464` |
| 移动端固定 header：`<el-header v-if="isMobile && !hideSidebar" class="mobile-header">`，`position: fixed; top:0; left:0; right:0; height: 56px; z-index: 100`，**自身没有 safe-area padding** | `App.vue:4` / `App.vue:567-580`（`grep -n safe-area App.vue` 只命中 `App.vue:1309-1311` 的 `.player-bar`） |
| 页面底部还有 `.page-footer`（`hideSidebar=false` 时渲染），带 `margin-top: 40px; padding: 20px 0` | `App.vue:197-198` / `App.vue:1054-1059` |
| viewport 已含 `viewport-fit=cover`，safe-area 变量可用 | `frontend/index.html:8` |
| `* { touch-action: manipulation }` 全局已设，禁用了双击缩放 | `frontend/index.html:16-18` |
| 全局 `* { box-sizing: border-box }`（**所以下面所有 `height/calc` 都含 padding**） | `frontend/src/styles/index.css:21-25` **【v2 新勘】** |
| `.tap-panel` / `.insight-panel` 是 `.work-grid` 的 **grid 子项**（≤720 单列）；`.insight-panel`「本周节奏」是独立的一块，在 `tap-panel` 之后 | `CounterTool.vue:135 / 217 / 2462-2464` |

**根因（四条，互相叠加）**

1. **720 / 768 之间存在断点裂缝**：`721–767px` 宽度下 `isMobile=true`（外壳按移动端渲染：56px 固定 header + 71px 顶部内边距 + 100px 底部内边距 + footer），但计数器自身仍走桌面布局（`overflow: hidden`、`.figure-button` 无移动端样式）。该区间内页面内容（hero 卡 + 预设 + work-grid 堆叠）远超可视高度，且外层内边距已经把可用高度吃掉 227px。
2. **移动端高度硬编码叠加**：`.counter-app` 的 `min-height: 100dvh` 是相对**视口**；而它实际被塞进一个顶部已经偏移 71px 的容器里。`.tap-panel` 的 `min-height: calc(100dvh - 26px - 安全区)` 同理，也是相对视口——**两者都没有减掉外壳偏移**。

   **推算**（按上述 CSS 逐项相加，非真机实测）：375×667 竖屏下，`.tap-panel` 的盒子高度 ≈641px 且从 y≈79 开始 → 底边约在 y≈720，**比视口底部低约 53px**；`.side-round` 定位在该盒子**底边**（`bottom:14px`），于是横竖两个加减按钮整体落在首屏之外。

   **⚠️ 实现方注意（B4 评审要求）**：不要照「溢出 78px」这类单一数字去调参数。该数字只在「内容累加」模型下成立；换成 flex 模型则是「首屏只露出 324/427」。**两种模型结论一致（首屏放不下主按钮 + 加减号），但调参必须按 §A2/§A3 的算式来，不要按差值拍。**
3. **嵌套滚动**：≤720px 时 `.counter-app` 自己也是滚动容器（`overflow:auto` + `overscroll-behavior-y: contain`），而外层文档层也是滚动容器。内层 `overscroll-behavior: contain` 会**阻断滚动链传递**，用户在内层滚到底后无法顺势滚到外层。
4. **横屏加宽反而失去安全区**：iPhone 横屏宽度 812/844/852/932px 全部 > 768px → 走桌面分支 → 完全不加 `env(safe-area-inset-*)`，刘海/home indicator 压住内容。

### 2. 音频现状（→ 需求 2）

```js
// CounterTool.vue:1231-1239（原文）
function getAudioContext() {
  if (!audioContext) {
    audioContext = new (window.AudioContext || window.webkitAudioContext)()
  }
  if (audioContext.state === 'suspended') {
    audioContext.resume()          // ← 返回值（Promise）未 await / 未 then
  }
  return audioContext
}
```

| 事实 | 位置 |
|------|------|
| `audioContext` 是模块级 `let`，只在挂载期复用，永不复位 | `CounterTool.vue:731` |
| 只判断 `'suspended'`，**不判断 iOS 的 `'interrupted'`，也不判断 `'closed'`** | `CounterTool.vue:1235` |
| `resume()` 不 await：函数同步返回 ctx，调用方立刻 `source.start(ctx.currentTime)` | `CounterTool.vue:1236` + `1427` |
| `playFigureSound()` 里若缓冲为空才现建缓冲 | `CounterTool.vue:1414` |
| `soundBuffers` 是全局 `Map`，只建一次，与「哪个 ctx 建的」不绑定 | `CounterTool.vue:1242-1250` |
| `primeAudio()` 只在主按钮点击路径调用（`handleFigureTap`），加减号路径不调用 | `CounterTool.vue:1133-1136` vs `163/187` |
| `visibilitychange` 只做 `persistAll()`，**回前台不做任何音频处理** | `CounterTool.vue:1533-1537` |
| 无 `pageshow` 监听（iOS bfcache 恢复无感知） | `CounterTool.vue:1543-1553` |
| 音频失败全部 `catch (_) {}` 静默，用户看不到任何提示 | `CounterTool.vue:1226-1228 / 1428-1430` |

**根因**：锁屏/切后台后 iOS 把 AudioContext 置为 `suspended` 或 `interrupted`（WebKit 专有状态）。`interrupted` 不满足 `state === 'suspended'` 判断 → **完全不调用 resume**；即便是 `suspended`，`resume()` 是异步的，而同一次调用里 `source.start(ctx.currentTime)` 在 ctx 时钟仍冻结的情况下被排程，于是**第一次敲击被吞掉**，第二次才响。这就是「息屏后再点击页面没有声音」的直接机制。

### 3. 会话/目标现状（→ 需求 3）

| 事实 | 位置 |
|------|------|
| 存储键：`counter_v4_state` / `counter_v4_records`，旧键 `counter_v3_*` 兼容读 | `CounterTool.vue:457-459` / `1039` / `1043` / `1052-1054` |
| `saveState()` 逐字段枚举写盘 → 未知字段不会被写入 | `CounterTool.vue:1022-1040` |
| `saveRecords()` 整体覆写 | `CounterTool.vue:1042-1044` |
| `watch([todayCount, dailyGoal, step, autoMode, …]) → saveState()`，**每次敲击已经写盘** | `CounterTool.vue:906-911` |
| **`normalizeRecords()` 会重建每条记录为 `{date, dayName, count, peakSpeed, dailyGoal}` 五个字段**，其余字段静默丢弃；且 `count<=0 && peakSpeed<=0` 的记录直接丢 | `CounterTool.vue:976-1011`（关键约束，见下） |
| 载入时若 state 的 `currentDate` 等于今天，会从 `dailyRecords` 中剔除今天那条 | `CounterTool.vue:1075-1078` |
| `checkDayRollover()`：日期变了就 `archiveDay(旧日期, todayCount, bestSpeed)` 然后 `todayCount.value = 0` | `CounterTool.vue:1108-1123`（清零在 `:1117`） |
| 跨零点只在 1s 时钟（`1545-1548`）和 `applyDelta` 开头被触发 | `CounterTool.vue:1146` |
| 现有唯一「目标」概念 = `dailyGoal`（按天、按次数），驱动 `goalPercent` / `streak` / 日历达标标记 | `CounterTool.vue:704 / 796-822 / 399 / 426-432` |
| 速度口径：15s 滑动窗口，跨度下限 clamp 到 0.03min（=1.8s），`bestSpeed` 取窗口速度最大值 | `CounterTool.vue:1185-1200` / `460` |
| `tapEvents` 是 **15s 滑动窗口**，每次 `refreshSpeed()` 就把 15s 之前的过滤掉 | `CounterTool.vue:727 / 1181 / 1187`（**B5：不能当会话时间轴的数据源**） |

**关键约束（必须写进实现）**：`normalizeRecords()` 在每次载入时重建记录对象并**丢弃未知字段**。因此**绝对不能**把会话数据挂到 `dailyRecords` 的某条记录上（例如给当天记录加 `sessions: []`）——下次刷新就会被静默清空。会话数据必须独立成键。

### 4. 误触与生命周期现状（→ 需求 4）

| 事实 | 位置 |
|------|------|
| 主按钮 `@pointerdown="handleFigureTap"`：**按下即计数**，无位移阈值、无时长阈值、不看 `isPrimary` | `CounterTool.vue:166-171` |
| 右侧加号 `@pointerdown.prevent="increment"`；左侧减号 `@pointerdown.prevent="decrement"`（`todayCount<=0` 时 disabled） | `CounterTool.vue:157-164 / 182-188` |
| 步长 chip（+1/+3/+5/+10/+20）紧贴敲击区上方，单击即生效 | `CounterTool.vue:143-154` |
| 自动连点：`setInterval` 最小 120ms，每 tick 也是 `increment()` → 会计数 | `CounterTool.vue:706-707 / 1206-1218` |
| 目标达成用 `el-dialog` 模态弹出，`close-on-click-modal=false` → 训练中挡住敲击区 | `CounterTool.vue:279-293` |
| **`App.vue:190-193` 用 `<keep-alive :exclude="[]">` 包住 `router-view`，且 `:key="currentViewKey"`（`App.vue:323` = `route.fullPath:viewRefreshKey`）** | `App.vue:190-193 / 323` |
| `currentViewKey` 会在 `goRoute()` 同路径点击（`App.vue:445`）与 `refreshCurrentViewAfterIdle()`（`App.vue:481-484`）时递增 | `App.vue:443-447 / 476-487` |
| `CounterTool.vue` 全文 **没有** `onActivated` / `onDeactivated` | `grep -n "onActivated\|onDeactivated"` → 无命中 |
| 清理逻辑全在 `onUnmounted`（清定时器、摘 `document` 的 keydown、摘 visibilitychange、`persistAll`） | `CounterTool.vue:1555-1564` |
| `resetAll()`「清除全部数据」只 `removeItem` `counter_v4_state` / `counter_v4_records` | `CounterTool.vue:1510-1531`（**B7**） |
| `clearHistory()` 只清 `dailyRecords`，确认文案「清空全部历史记录，但保留今天的计数？」 | `CounterTool.vue:1498-1508` |
| `resetToday()` 只重置 `todayCount`/`speed`/`bestSpeed`/`lastActionDelta`/`tapEvents`/`goalFired` | `CounterTool.vue:1478-1497` |

**由此得出一个此前未被发现的真实缺陷**：因为 keep-alive 命中，**路由离开 `/counter` 时组件不会 unmount**，`onUnmounted` 不执行 → `document` 上的 `keydown` 监听、1s 时钟、`visibilitychange` 监听**全部继续存活**。后果：在别的工具页按空格/↑，`onKeyDown`（`CounterTool.vue:1443-1465`）仍会调用 `increment()`，**计数器在后台静默增长并写盘**；`autoMode` 若开着也会继续自增。这既属于「计数不准」也属于需求 4 的误触范畴，且直接影响会话「路由离开如何收尾」。

### 5. 全局常驻 `position: fixed` 浮层现状（**【v3 新勘】→ B8**）

> 这一节是 v2 漏勘的一整类对象。它们**不在** `CounterTool.vue` 里，但决定训练层能不能被看见、底栏能不能被点到。

| 浮层 | 位置 / 层级 | 何时存在 | 尺寸 / 位置（375×667） |
|------|------------|---------|----------------------|
| `.player-bar` | `App.vue:1167-1176`：`position: fixed; bottom:0; left:0; right:0; z-index: 10000`；经 `App.vue:239-296` 的 `<Teleport to="body">` 挂到 body | **只要本次会话播放过媒体就一直在**（`frontend/src/composables/useMediaPlayer.js:6` 初始 `currentIndex: -1`，**无持久化**）；`App.vue:241` 的 `v-if="playerState.currentIndex >= 0"` | 移动端 `.pb-inner { height: 52px }`（`App.vue:1314`）+ `padding-bottom: env(safe-area-inset-bottom)`（`App.vue:1309-1312`）→ 占 y≈615–667 |
| `.pet-window` | `GlobalPet.vue:341-343`：`position: fixed; z-index: 9998`；由 `App.vue:299` 全局挂载 `<GlobalPet />` | `GlobalPet.vue:121` `const visible = ref(true)` **默认真**；`loadState()`（`:154-156`）会用 `localStorage['pet-widget.global']` 覆盖 → **首次访问 / 未手动收起过时为真** | 尺寸 `GlobalPet.vue:150`：`280 × 348`；默认位置 `GlobalPet.vue:161-164` → `x = 375−280−24 = 71`、`y = 667−348−24 = 295` → 覆盖 **x 71–351 / y 295–643** |
| `.pet-launcher`（宠物收起后的浮标） | `GlobalPet.vue:320-333`：`position: fixed; bottom:24px; right:24px; 48×48px; z-index: 9999` | 与 `.pet-window` **二选一**（`GlobalPet.vue:5-6` 的 `v-if="!visible"`） | 右下角 48×48 |
| `.cheer-toast` | `GlobalPet.vue:454-470`：`position: fixed; z-index: 10000` | 打气文案触发时（瞬态） | 瞬态，不计入布局 |

**全仓库 `z-index` 实测分布（`grep -rn "z-index" frontend/src`，仅列 ≥1000）**：

```
99999  frontend/src/main.js:55          ← 仅开发态报错条（该文件是 dev 专用内联 DOM），不参与产品层级
10000  frontend/src/App.vue:1170        ← .player-bar
10000  frontend/src/components/GlobalPet.vue:470   ← .cheer-toast
 9999  frontend/src/components/GlobalPet.vue:333   ← .pet-launcher
 9999  ImageViewerTool.vue:409 / ExpenseTool.vue:4067 / VirtualAvatarTool.vue:506 / MermaidTool.vue:1698
 9998  frontend/src/components/GlobalPet.vue:343   ← .pet-window
 2400  PlannerTool.vue:12047 · 2350 PlannerTool.vue:12114 · 2000 PlannerTool.vue:15022
 2000  frontend/src/share/PasteView.vue:1574
 1200  frontend/src/views/life/CounterTool.vue:2192 ← .panel-overlay（本模块自己的浮层）
 1000  frontend/src/styles/tools.css:104 · ToolLayout.vue:58 · HouseholdTool.vue:664
```

> **结论**：**常驻产品浮层的最大 `z-index` 是 `10000`**（`.player-bar` / `.cheer-toast`）。Element Plus 的弹层由 popup manager 从 `2000` 起递增分配（`.el-overlay` / `.el-message` / `.el-message-box` 同池），远低于 10000。
> 另：`.counter-app`（`CounterTool.vue:1568-1576`）是 `position: relative` 且 **`z-index: auto`，不创建层叠上下文**；`.app-container`（`App.vue:547-553`）、`.main-content`（`App.vue:1032-1042`）、`.mobile-main`（`App.vue:1044-1051`）**都没有 `position`/`z-index`**（`.sidebar` 的 `z-index:10` 因无 `position` 而无效，且不在祖先链上）。→ 训练层作为 `.counter-app` 的后代，其 `z-index` 直接与上表**同一个（根）层叠上下文**里比较。

**由此产生的两条 MUST（v2 未识别）**：

1. v2 的 `z-index: 900` 会**被上述四个浮层全部盖住**。训练层底栏（「暂停/继续」「结束训练」，对应 §A4 的「距屏幕底边 ≥ 安全区 + 12px」）与 `.pet-launcher` / `.player-bar` **位置重合**；而「结束训练」是训练层内**唯一退出路径**——被盖住后用户只能刷新页面。
2. 这些浮层自带可点控件（宠物 6 个操作按钮 + 拖动条；播放器随机/上一首/播放/下一首/循环/停止 + 进度条 seek），在训练期是**未设防的误触面**，直接违背 §D2① 的「结构上无可误触」与「锁 UI，不锁计数」。

---

## 设计

### A. 移动端适配

#### A1 断点统一（与 `isMobile` 严格对齐）

- 计数器移动断点从 **`max-width: 720px` → `max-width: 767.98px`**。**注意不是 768**：外壳判定是 **`width < 768`**（`App.vue:464`），而 CSS `max-width: 768px` 含 768 —— 用 768 会在**恰好 768px** 这一档出现「CSS 按移动端渲染、JS 不套 `.mobile-main`」的新裂缝（无固定 header、无 71px 顶部内边距）。`767.98px` 与 `width < 768` 在 1/64px 精度下等价。
- 保留 `max-width: 420px`（**v3 撤回 v2 的 420→480 改动**，见下方「为什么不改 480」）与新增横屏断点。
- 最终断点表（**全文件唯一口径，`grep "@media" CounterTool.vue` 只应命中这 4 条 + 既有桌面断点**）：

| 断点 | 用途 |
|------|------|
| `max-width: 1024px` | 双列 → 单列过渡（保留原 980 语义，仅调数值） |
| `max-width: 767.98px` | 移动布局（**与外壳 `width < 768` 对齐**） |
| `max-width: 420px` | 小屏字号/内边距微调（**保持既有数值，不扩范围**） |
| `orientation: landscape and (max-height: 480px)` | 手机横屏专用（左计数、右敲击区），**不嵌套在移动断点内**，见 A6 |

- **为什么不改 480（v3 定稿，收口非阻塞 4）**：v2 的 420→480 写的是「对齐通用的 480 断点」，没有需求或缺陷支撑；而它会把 **421–480px 这一整段的既有样式全部改掉**——`CounterTool.vue:2751-2805`（`.counter-app` 左右 padding、`.tap-panel` padding、`.mobile-focus-count` 字号 48→42、`.side-round` 56→52、`.figure-button min-height 360→330`、`.mobile-main-actions` 按钮 flex-basis）会整体外扩 60px。这属于**非必要扩范围**，且会让 §A3 的首屏预算在两个视口（375 / 320，两者都 ≤420 也 ≤480）之外的机型上出现未验证的变化。**结论：回到 420，保持既有数值。** 若日后确需对齐 480，单开子 issue 并补 421–480 的首屏预算与本表。
- **「≤360px」不是断点**：v2 §A3 的降级阶梯里出现过 `≤360px`，与上面三条并列会造成「到底有几个断点」的歧义。v3 明确：**降级阶梯只作为应急备案写在 §A3 的附注里，不预先写成 `@media` 规则**；只有当真机实测敲击面 < 200px 下限时才由 QA 回报、再由实现方加一条临时断点。

- **恰好 768px 这一档的行为要写进验收**：该档下外壳**不**套 `.mobile-main`（无 header、padding 20px）、但 `App.vue:1152-1160` 的媒体查询**生效**（`.main-content` 变 `height:auto; overflow-y:visible`）。计数器此时走桌面分支 → `.counter-app { min-height: 100% }` 解析为 0（父容器高度由内容决定）→ 容器塌到内容高度，**页面自然滚动、功能不受损**，只是氛围背景不再满高。这是一个 **1px 宽**的既有外壳行为（我们不改 App.vue），明确接受，不作为缺陷。

#### A2 高度模型：移动端**不依赖百分比**（B3）

**（1）现状事实（含 v2 补勘）**

```css
/* App.vue:1152-1160 —— 原方案漏勘，正是本方案的目标区间 */
@media (max-width: 768px) {
  .app-container { flex-direction: column; overflow: visible; }
  .main-content  { height: auto; overflow-y: visible; }
}
```

**（2）高度到底解析成什么**

| 容器 | 桌面（≥769） | 移动（≤767.98） |
|------|-------------|----------------|
| `.app-container`（`App.vue:548-553`） | `height: 100dvh` + `overflow: hidden` | 仍是 `height: 100dvh`（媒体查询**只覆盖了** `flex-direction` / `overflow`），`overflow: visible` |
| `.main-content`（`App.vue:1032-1042`） | `height: 100dvh; overflow-y: auto` → **高度确定、且是全页唯一滚动容器** | 被 `App.vue:1152-1160` 覆盖为 `height: auto; overflow-y: visible` → **高度取决于内容** |
| 于是 `.counter-app { min-height: 100% }` | 父容器高度按 `height` 属性显式确定 → 百分比**按规范成立**，解析为 `.main-content` 内容盒高度（`100dvh − 40px` padding） | 父容器高度由内容决定（indefinite）→ 按 **CSS 2.1 §10.7**，非绝对定位元素的百分比 `min-height` **当作 0 处理** |

需要澄清一个容易混淆的点：`.app-container` 是 `height:100dvh` 的定高 flex 容器，`.main-content` 是它的 `flex: 1` 子项。现代引擎（Chrome / Firefox / Safari）会把 flexing 之后的尺寸视为 definite，因此真机上 `100%` **大概率**仍能算出满高。但这依赖两点：① 引擎实现行为（不是可写进契约的保证）；② `.mobile-main` 的 `min-height` / `padding` 一分不改。任何一处变动都会**静默失效**。所以本方案**不依赖它**。

**（3）结论：桌面留百分比，移动改显式算式**

```css
/* 桌面（≥769）保持不变：父容器显式定高，百分比可依赖 */
.counter-app {
  min-height: 100%;     /* 原 min-height: 100vh / 100dvh 两行改为这一行 */
  overflow: hidden;     /* 保持且【全断点生效，包括移动端】：只为裁剪 .ambient-layer 光斑，见（5） */
}

/* 移动（≤767.98）—— 全程不出现百分比；【v3/B9】不再声明 overflow */
@media (max-width: 767.98px) {
  .counter-app {
    /* 71 = 56(.mobile-header 固定高, App.vue:567-580) + 15(.mobile-main padding-top, App.vue:1044-1047) */
    --ct-chrome-top: 71px;

    /* 不再叠加 --safe-area-inset-top：56px 固定 header 自身无 safe-area padding，
       已经覆盖了顶部刘海区（App.vue:567-580），再叠一次会白吃最多 47px */
    padding: 8px max(8px, env(safe-area-inset-right))
             calc(14px + env(safe-area-inset-bottom))
             max(8px, env(safe-area-inset-left));

    min-height: calc(100vh - var(--ct-chrome-top));    /* 兜底：不支持 dvh 的旧 WebView */
    min-height: calc(100dvh - var(--ct-chrome-top));

    /* 【v3/B9 定稿】删除 v2 写的 `overflow: visible`。
       这里【不声明 overflow】，继承基类的 `hidden`（见（5））。
       同时删除基类之外的 `overflow: auto` + `overscroll-behavior-y: contain`
       （原 CounterTool.vue:2411-2412）——见（4）。 */
  }

  /* ≤420 的 padding-left/right 覆盖（CounterTool.vue:2752-2755）必须同步改成
     max(8px, env(safe-area-inset-left/right))，否则横向安全区失效 */
}
```

`box-sizing: border-box` 由 `frontend/src/styles/index.css:21-25` 的 `*` 规则保证，所以上面的 `calc` 高度**含** padding。

**（4）移动端滚动容器是文档层 —— `overscroll-behavior` 与 `.app-container{overflow:visible}` 的交互（B3 第二问）**

- ≤768 下 `.main-content` 既不是滚动容器（`overflow-y: visible`）也不是定高容器，`.app-container` 又是 `overflow: visible` → **实际滚动容器是文档层（`html`/`body`），不是 `.main-content`**。所以原方案「全页只保留外层 `.main-content` 一个滚动容器」这句话在 ≤768 **是反的**，本版删除。
- `.app-container { overflow: visible }` 在移动端**对滚动没有任何影响**：它既不裁剪也不滚动，唯一作用是让高度不足时内容能溢出到文档流里被文档层接住。
- 因此 `overscroll-behavior-y: contain` 写在 `.counter-app` 上只在它**自己可滚**时才生效。本版把 `.counter-app` 的 `overflow: auto` 一起删掉 → 它不再是**用户可滚**的容器（见（5）对 `hidden` 的精确说明）→ **内层阻断滚动链的问题（§现状 根因 3）从根上消失**；文档层的 `overscroll-behavior` 不由本模块控制（`html`/`body` 无此声明），保持全站一致的下拉刷新行为即可。
- 本版不再刻意保留任何内层滚动容器：目标视口下首屏放得下（§A3），放不下时**由文档层滚动兜底**，内容永远可达。

**（5）移动端 `overflow` 定稿：保留基类 `hidden`（B9 —— v3 的唯一新增处置）**

**v2 的自相矛盾**：桌面规则（`CounterTool.vue:1568-1576` 的基类）**保留** `overflow: hidden` 并注明「只为裁剪 `.ambient-layer` 光斑」；移动断点（原 `CounterTool.vue:2411-2412`）却是 `overflow: auto`，而 v2 方案又把移动端改成 `overflow: visible`。同一份方案对同一属性给出相反理由。

**v3 定稿：移动断点下不声明 `overflow`，继承基类的 `hidden`。** 理由三条：

1. **不裁剪会造成 45px 横向出血**。`.ambient-layer::after`（`CounterTool.vue:1600-1609`）刻意向**右**出血 `right: -60px`；`.counter-app` 的右边界在视口内 15px 处（`App.vue:1046` `.mobile-main { padding: 15px }`）→ 净出血约 **45px** 逃逸到上层。而 L3 #48 明确断言「且无横向滚动条」，需求 1 的用户原话就是「移动端页面展示不能适应窗口」——横向滚动条属于同一症状。
   - **本工作树未能离线核实的部分（不构成不修的理由）**：这 45px 最终落在文档层还是 `.main-content`，取决于 Element Plus `.el-main` 的基样式——`App.vue` 对 `.main-content` 只声明了 `overflow-y`（`App.vue:1038`、`:1160`），**从未声明 `overflow-x`**；本工作树 `frontend/node_modules` 未安装，无法离线核对。**但无论落在哪一层，「装饰出血不再被裁剪」这一点是确定的**，所以必须给明确处置——即下面的 2/3 两条。
2. **`hidden` 不妨碍文档层纵向滚动**。`.counter-app` 是 `min-height` + 内容撑开（**不是定高盒子**），高度随内容增长 → 纵向不存在「内容超出盒子」的情形，`hidden` 裁不到任何东西；全局 `* { box-sizing: border-box }`（`frontend/src/styles/index.css:21-25`）也保证 padding 已被计入 `min-height`。放不下时由文档层滚动兜底（同（4））。
3. **为什么这不会重新引入 v1 的「内层滚动容器阻断滚动链」**（§A2 要求补的依据句）：CSS 的 `overflow: hidden` 确实会让盒子成为**scrollport**，但它 **不可被用户滚动**（无滚动条、无触摸滚动、`scrollTop` 恒为 0），因此**不存在可以「滚到底」的内层容器**，`overscroll-behavior` 也就无从触发滚动链阻断。v1 的病灶是 `overflow: auto` + `overscroll-behavior-y: contain` 这对组合（可滚 + 主动截断链），v3 两个都已删除。**判据**：`grep -n "overscroll-behavior" CounterTool.vue` 在移动断点区块内必须无命中。

**为什么不选「移动端保持 `visible` 再给 `.ambient-layer` 单独加裁剪层」**：那需要新增一层 `overflow: hidden` 的包装 `div`（改动模板结构，且要给该层补 `position/inset/pointer-events` 语义），收益为零——基类本来就已经在裁剪。方案取「**最小改动 = 什么都不声明**」。

#### A3 日间模式首屏预算（B4 —— 这是需求 1 的闭环）

**目标（PM 判定，不得降级为「允许滚动」）**：
- 375×667：首屏完整可见「今日计数 + 主按钮 + 加减号」，操作行（撤销/自动/功能）也在首屏内。
- 320×568：允许降级次要元素（焦点条 / 步长条等），但**主按钮 + 加减号必须在首屏**。

**做法：把 `.tap-panel` 的最小高度从「相对视口」改成「扣掉外壳偏移后的首屏高度」，并拆掉 `.figure-button` 的固定下限，让敲击面吃掉剩余空间。**

现状的病灶是两处常数（`CounterTool.vue:2452` 的 `min-height: calc(100dvh - 26px - …)` 和 `:2574 / :2775` 的 `min-height: 360px / 330px`）：前者没有减掉外壳的 71px 顶部偏移，后者在剩余空间不足时把盒子**撑破**而不是收缩。

```css
@media (max-width: 767.98px) {
  .tap-panel {
    /* 首屏可用高度 = 100dvh − 外壳上偏移(71) − counter-app 上下 padding(8+14) − 底部安全区 */
    /* 删除原 min-height: calc(100dvh - 26px - env(safe-area-inset-top) - env(safe-area-inset-bottom)) */
    min-height: calc(100dvh - var(--ct-chrome-top) - 22px - env(safe-area-inset-bottom));
    display: flex;              /* 现状已有，保留 */
    flex-direction: column;     /* 现状已有，保留 */
  }

  .tap-stage {
    display: flex;              /* 原 display: block */
    flex-direction: column;
    flex: 1 1 auto;
    min-height: 200px;          /* 原 min-height: 0 —— 给敲击面一个下限 */
    margin-top: 10px;
  }

  .figure-button {
    flex: 1 1 auto;             /* 新增：吃掉 tap-stage 的剩余高度 */
    width: 100%;
    height: auto;               /* 原 height: 100%（百分比，避免依赖） */
    min-height: 0;              /* 原 min-height: 360px —— 关键：不再撑破容器 */
    border-radius: 26px;
  }
}

@media (max-width: 420px) {
  /* 原 .figure-button { min-height: 330px } —— 删除（由上面的 min-height: 0 覆盖） */
}
```

**为什么这样够用（推算，非真机实测；验收按「可见性」判定，不按像素 —— 【v3 非阻塞 3 重算】）**

先给**逐项明细**（v2 只给了聚合值 22+74+54+10+74，无法核对，且 FINN-18 指出其中一项的理由写错了）：

| 固定开销项 | 值 | 依据（已核实） |
|-----------|----|--------------|
| `.tap-panel` padding 上下 | **22** | `padding: 12px 10px 10px`（`CounterTool.vue:2757-2759`，**≤420 覆盖**）——**不是** `:2451` 的 `14px 12px 12px`（26px） |
| `.mobile-focus-strip` 高度 | **78**（推算） | `.mobile-focus-count` = 日期 11px + 标签 11px + 计数 42px（≤420 值）+ 间距 6 + `padding-top: 2`；`.mobile-focus-meta` = 28+6+28 = 62，取较大者 |
| `.step-strip` | **54** | `margin-top: 12`（`:2527`）+ 芯片 `min-height: 40`（`:2539-2545`）+ `padding-bottom: 2`（`:2530`） |
| `.tap-stage` margin-top | **10** | `CounterTool.vue:2552` |
| `.mobile-main-actions` 高度 | **64**（推算） | `margin-top: 10` + `padding: 8×2` + 按钮 `min-height: 38`（`:2622-2645`） |
| **合计** | **228** | |

> ✅ **v2 写的 `22（面板 padding）` 数值是对的、但没写来源**；FINN-18 认为「≤720 的最终生效值是 26px」——**这一前提不成立**：375×667 与 320×568 **同时命中 ≤720 与 ≤420 两个区块**，而 ≤420 区块在文件更靠后（`:2751` > `:2409`）、特异性相同 → 覆盖生效，纵向 padding = 12 + 10 = **22px**（v2 原始数值）。（这条不影响任何结论，但必须写清来源，否则 Stage 3 会照 26px 调参。）

| 视口 | `.tap-panel` 首屏高度 | 面板内固定开销 | **敲击面高度** | 主按钮可见宽度 | 结论 |
|------|---------------------|--------------|--------------|--------------|------|
| 375×667 | 667 − 71 − 22 = **574** | **228** | **≈346** | ≈309 | 全部首屏可见 ✅ |
| 320×568 | 568 − 71 − 22 = **475** | **228** | **≈247** | ≈254 | 全部首屏可见 ✅ |
| 390×844 | 844 − 71 − 22 = **751** | 232 | ≈519 | ≈324 | ✅ |
| 414×896 | 896 − 71 − 22 = **803** | 232 | ≈571 | ≈348 | ✅ |

- **与 FINN-18 建议值（324 / 225）的差异要写清，不要静默改数**：两者相差约 22px，全部来自「焦点条 74→78」「操作行 74→64」这两个**推算**项的估算口径；v2 的 340 / 241 落在同一区间的另一端。**任一口径下的结论完全一致**：敲击面 ≈ 225–350px，**远大于 `min-height: 200px` 下限**，且面板底边 ≤ 视口底边 → 主按钮 + 加减号必在首屏。因此 **L3 的判据是 `elementFromPoint` 可见性（§测试要点 L3 #48），不是任何一版像素值**；实现方**不要**照任何单一数字调参。
- 面板底边位置：`71 + 8 + 面板高度` ≤ `100dvh`（375×667 下为 `79 + 574 = 653 ≤ 667`，余 14px 呼吸位 ✅）。加减号是 `.tap-stage` 内的绝对定位元素（`bottom: 12/14px`），天然落在这个盒子内 → 必在首屏。
- 320×568 也**不需要**触发降级（247px > 200px 下限）。
- **降级备案（应急，**不要**预先写进代码）**：v2 曾把它写成「`≤360px` 时…」，与 §A1 的三条断点并列会造成「到底有几个断点」的歧义。v3 明确：**下面三条不写成 `@media` 规则**，只在真机实测敲击面 < 200px 时，由 QA 回报后再由实现方临时加断点，顺序与代价如下：
  1. `display: none` 掉 `.step-strip`（−54px，代价：改步长需先进设置面板）；
  2. 把 `.mobile-focus-count strong` 从 `42px`（≤420 值）再降到 `34px`（−8px）；
  3. 把 `.mobile-focus-meta` 的两枚胶囊合并成一行（−30px）。
- 「本周节奏」面板（`.insight-panel`）在 `.work-grid` 单列布局下自然排在 `.tap-panel` **之后**，因此落在首屏下方，滚动一屏可达。这是**有意的空间让位**，不是残留缺陷；L3 验收不要求它在首屏。

#### A4 主按钮触达尺寸与位置（需求明确要求给出 px/vh 依据）

| 元素 | 尺寸结论 | 依据 |
|------|---------|------|
| 日间模式敲击面 | 宽 = 容器内宽（320 机型 ≈254px）；高 = `.tap-stage` 剩余高度，**下限 200px**（`min-height: 200px`） | 320×568 推算 ≈241px、375×667 ≈340px；200px 下限保证极矮视口下按钮仍是一个「整块面」而不是细条 |
| 训练模式敲击面 | 宽 100%（减左右 `max(24px, safe-area)`），高 = `max(200px, calc(100dvh - 128px - 安全区))`（顶栏 56 + 底栏 72） | 375×667 ≈539px（≈81dvh）；844×390 横屏 ≈262px |
| 训练模式底栏「结束训练」 | 高 56px、宽 ≥132px、距屏幕底边 ≥ 安全区 + 12px | Apple HIG 44pt / Material 48dp 最小值上取一档，运动中手抖也不易误触 |
| 训练层左右边缘留白 | `max(24px, env(safe-area-inset-left/right))` | iOS Safari / Android 都有边缘返回手势（左侧 ~20px 起），贴边敲击会被系统判为返回手势 |
| 日间模式减号/加号 `.side-round` | 保持 56px（≤420 52px），均 ≥ Material 48dp | `CounterTool.vue:2555-2564 / 2790-2796` |
| 任意可点元素底线 | ≥48×48px，相邻可点元素间距 ≥8px | Material 触控目标规范 |
| 拇指可达区 | 主要操作落在视口下 60% 区域内 | 单手竖握时拇指自然活动范围；训练层底栏天然满足，日间模式敲击面占据下半屏 |

#### A5 需要一并修掉的外壳残留（属本模块可改范围）

- `.mobile-main` 的 `padding-bottom: 100px`（`App.vue:1047`）与 `.page-footer`（`App.vue:197` / `1054-1059`）在计数器页产生约 100–140px 的**首屏下方**死滚动。**不修改 App.vue**（外壳属共享代码，见非目标），也**不用负 margin 抵消**（负 margin 依赖外壳内边距的具体数值，属脆弱耦合）。
  **采纳方案**：A3 保证首屏内容完整可见；首屏下方的 100px + footer 由文档层滚动承接，属**已知且可接受**的残留（用户在首屏完成所有主操作，不会触发它）。若后续要彻底消除，单开子 issue 讨论是否给 `/counter` 加 `hideSidebar: true`（见「未决问题 #11」）。
- **【v3 新增】日间模式下常驻浮层对首屏的遮挡（未定义边界 2 的产品立场）**：
  - 事实（§现状 5 已勘）：375×667 下 `.pet-window`（默认位置 x 71–351 / y 295–643）**完整覆盖** `.side-plus`（`.tap-stage` 内 `right:14px`，宽 52px → 约 x 276–328）；`.player-bar`（本次会话播放过媒体时才出现，占 y 615–667）会遮住 `.mobile-main-actions` 操作行的下沿（面板底边 y≈653）。
  - **立场（v3 定稿）**：**接受为已知残留，计数器页不主动收敛这两个浮层**。依据三条：① 它们是**跨页面常驻的全局能力**（媒体播放器 / 魔法宠物），不是计数器页的私有 UI，计数器页无权单方面隐藏；② 任何「隐藏」都要动 `App.vue`（外壳），与 §硬约束「不改 `App.vue`」直接冲突，属**非必要扩范围**；③ 两者都可**用户自行解除**且状态持久化（宠物的 `pet-widget.global` 可收起为 48×48 的 `.pet-launcher`；播放器有「停止播放」按钮，且**不持久化**，重开页面即消失）。
  - **验收上的处置**：不把它们算作「通过」，而是把「浮层开 / 关」作为 L3 的**显式前置条件维度**（§测试要点 L3 #48a/#48b）。被浮层遮住 = `elementFromPoint` 命不中 = 该档 **FAIL**，必须如实回报——**不允许用「浮层是用户自己开的」来掩盖判据**。
  - **若产品要求彻底解决**（推荐**单开子 issue**，不在本方案内）：给 `/counter` 加一条路由 meta（如 `globalOverlays: false`），由 `App.vue` 据此隐藏 `.player-bar` / `<GlobalPet />`——这需要改外壳，且要处理「训练结束后恢复」的状态机，成本不低。

#### A6 横屏（次要目标，允许降级）

- 新增 `@media (orientation: landscape) and (max-height: 480px)`，**放在文件末尾且不嵌套在移动断点内**——因为 844×390 这类机型宽度 > 768（走桌面分支），而 667×390 这类机型宽度 ≤ 768（走移动分支），两者都要命中。
- 规则：`.work-grid` 改左右分栏（左列计数/目标，右列敲击区），`.tap-panel` 的 `min-height` 复位为 `auto`（横屏由宽度而非高度承载），`.step-strip` / `.mobile-focus-meta` 隐藏。
- 横屏的验收标准放宽为**「主按钮 + 加减号 + 结束训练按钮可达（允许滚动一屏内）」**，不要求首屏完整可见——它不是需求 1 的验收档位。

#### A7 safe-area 全断点生效

**【v3 / 非阻塞 5 修正：整段必须包在 `@media (min-width: 768px)` 里】** v2 直接裸写 `.counter-app { … }`，与基类 `.counter-app`（`CounterTool.vue:1568-1576`）**同选择器、同特异性**；只要它落在文件靠后的位置，就会把移动端 §A2 那份 padding（顶部固定 8px）一起覆盖掉，把 §A3 的 22px 预算悄悄变成 28px。

```css
/* 仅用于桌面 / 横屏分支（横屏手机宽度 > 768 也走这里，见 A6）。
   max-width: 767.98px 与 min-width: 768px 互补，恰好 768px 归桌面分支（与 §A1 的说明一致）。 */
@media (min-width: 768px) {
  .counter-app {
    padding:
      max(12px, env(safe-area-inset-top))
      max(12px, env(safe-area-inset-right))
      calc(16px + env(safe-area-inset-bottom))
      max(12px, env(safe-area-inset-left));
  }
}
```

移动竖屏（≤767.98）用 §A2 里那份 padding（顶部固定 8px，**不含** `safe-area-inset-top`，理由见 A2(3)），**不受本段影响**。

---

### B. 音频恢复

#### B1 抽出独立模块 `frontend/src/composables/useCounterAudio.js`

必须抽出的理由：① 组件已 2806 行；② 音频逻辑要能被 vitest 单测（jsdom **没有** `AudioContext`，所以 ctx 必须由外部注入工厂，否则无法测）。

```js
// 契约（实现按此签名落地）
export function createCounterAudio({
  ctxFactory = () => new (window.AudioContext || window.webkitAudioContext)(),
  now = () => performance.now(),
  resumeTimeoutMs = 250
} = {}) {
  return {
    unlock(),                     // 必须在用户手势内同步调用；同步返回，不阻塞调用方，返回 Promise<void>
    play(soundName, volumeRatio), // 永不抛；内部决定是否延后播放
    handleForeground(),           // visibilitychange→visible / pageshow 专用：只置标志，绝不碰 ctx
    handleForegroundClock(),      // bfcache 恢复后补跑时钟的钩子（由组件调用 checkDayRollover/refreshSpeed）
    isDegraded,                   // ref<boolean>，UI 用它显示「音效已暂停，点此恢复」
    rebuild(),                    // 强制丢弃旧 ctx + 缓冲，重建（只在用户手势「恢复音效」按钮里调用）
    dispose()
  }
}
```

#### B2 降级阶梯与 `play()` 的真实接线（非阻塞 1）

原方案的「四级阶梯」与 `play()` 实现对不上（第 2/3 层在自动路径不可达）。本版**写死每层挂在哪**：

| 层 | 动作 | **只在这里执行** | 理由 |
|----|------|----------------|------|
| 0 | 清理 / 重建：`ctx` 不存在或 `state === 'closed'` → `ctxFactory()` + 清空重建 `soundBuffers` | `unlock()`（手势内）优先；`play()` 里也可建（手势外建 ctx 在 iOS 上只会得到 `suspended`，不抛异常） | `soundBuffers` 与 ctx 绑定，换 ctx 必须重建 |
| 1 | `await raceWithTimeout(ctx.resume(), 250)`；**必须判断 `'interrupted'`** | `unlock()`（手势内）+ `play()`（§B3） | 唯一一处允许调 `resume()` 的地方就是这两个函数 |
| 2 | 「静音 buffer 解锁」：`resume()` 成功后 connect 一个 1 帧静音缓冲并 `start()`，把输出管线真正推起来 | **只在 `unlock()` 内**，且每个 ctx 只做一次（`silentPrimed` 标志） | 这本来就是 iOS 的手势解锁手法，放自动路径没有意义 |
| 3 | `ctx.close()` + 新建 ctx + 重建全部音色缓冲 | **只在 `rebuild()` 内，且 `rebuild()` 只由用户手势（「点我恢复」胶囊）调用** | `close()` + `new AudioContext()` 必须在手势调用栈内，否则新 ctx 直接是 `suspended` |
| 4 | 连续 2 次 `resume()` 超时 → `isDegraded = true`，UI 显示提示胶囊 | `play()` 内计数 | 震动与音频无关，降级后仍可用 |

**硬约束（写死给实现方，code-reviewer 按此 grep）**：除 `unlock()` 与 `play()` 外，**任何代码路径都不得调用 `ctx.resume()` / `ctx.close()` / `new AudioContext()`**。特别是 `visibilitychange` / `pageshow` / 定时器 / `play()` 的自动重试里**一律禁止**（B2）。

#### B3 首击不丢（核心修复）

```js
async function play(sound, volume) {
  if (degraded.value) return
  const ctx = ensureCtx()                         // 层 0：缺失/closed 则新建
  if (ctx.state !== 'running') {
    // 关键：不立即 source.start()，而是等 resume 落地后再播
    const ok = await raceWithTimeout(ctx.resume(), resumeTimeoutMs)   // 250ms 上限
    if (!ok) { failCount++; if (failCount >= 2) degraded.value = true; return }
  }
  if (!silentPrimed) primeSilent(ctx)             // 层 2 的幂等版本（仅首次）
  scheduleSource(ctx, sound, volume)              // 此时 ctx 时钟已恢复，第一击有声
  failCount = 0
}
```
要点：`resume()` 返回 Promise **必须被 await/race** 后再 `start()`（这是当前 `:1236` 缺的那一步）。超过 250ms 未恢复就放弃本次播放并标记，避免用户长时间无反馈。

#### B4 回前台 / bfcache 生命周期（B2 —— 硬口径）

| 事件 | 处理 |
|------|------|
| `visibilitychange → 'visible'` | **音频侧只做一件事**：`audio.handleForeground()` 置 `needsUnlock = true`。**不得**调用 `resume()` / `close()` / 新建 ctx / `play()`。真正的恢复只能发生在**下一个用户手势**里。**会话侧**另调 `session.handleForeground()`，严格按 §C1.4(v3) 的 **(a) 先结算 idle → (b) 再 `checkDayRollover()` → (c) 最后消费 `pendingSplitFrom`** 三步执行。 |
| `pageshow`（`event.persisted`，iOS bfcache 恢复） | 同 `visible`：调 `audio.handleForeground()`（只置标志、**不碰音频**）与 `session.handleForeground()`；并补跑一次 1s 时钟逻辑（`checkDayRollover()` + `refreshSpeed()`，且必须过 `isActive` 守卫）——休眠期间可能已跨零点。**注意**：`checkDayRollover()` 已经在 `session.handleForeground()` 的第 (b) 步里跑过，这里不要重复触发拆分（幂等即可，`currentDate` 已更新时 `checkDayRollover` 会提前 return）。 |
| `visibilitychange → 'hidden'` | 只 `persistAll()`（保持现状 `:1533-1537`）；**不**关闭 ctx（下次前台还要用）；**同时写两个字段**：`active.hiddenSinceMs = Date.now()`（若有 active）与 store 级 `lastHiddenAtMs = Date.now()`（**v3 新增**，见 §C1 数据模型），并把会话的当前「活跃段」结算进 `activeMs`（§C1.4）。`lastHiddenAtMs` 在 `active` 被收尾后**不清除**——§C1.4(c) 还要用它判定 `away`。 |
| 任意用户手势（`pointerdown` / `keydown`） | 同步调 `unlock()`：若 `needsUnlock` 或 ctx 非 `running` → `await raceWithTimeout(ctx.resume(), 250)`，成功后 `needsUnlock = false`。**`unlock()` 必须同步返回、不得阻塞调用方**——`increment()` 先执行，`unlock()` 后执行（或至少不 await）。 |
| `play()` | 保持 §B3 的 `await raceWithTimeout(ctx.resume(), 250)` —— 这是唯一另一处允许 resume 的地方。 |

**兼容现状**：`primeAudio()`（`:1220-1229`）扩为「在**所有**手势入口调用」——主按钮 `handleFigureTap`、加号 `increment`、减号 `decrement` 三处（现状只有主按钮调用）。**且调用顺序必须是 `increment()` → `unlock()`**，不能反过来。

#### B5 用户可见反馈（当前完全静默）

- `degraded === true` 时，在敲击区顶部显示胶囊：**「音效已暂停 · 点我恢复」**（可点元素 ≥48px 高）。点击 → `rebuild()`（层 3），成功后 `degraded = false`。
- 切后台/息屏回来且 1 次 `resume()` 未成功时，**不弹**任何提示（静默重试，因为下一次手势大概率自动恢复）；只有连续 2 次失败才显示胶囊。
- 不做 Toast 打扰（训练中 Toast 会遮挡计数）。

#### B6 ctx 数量铁律

全生命周期只允许存在 1 个 AudioContext；`rebuild()` 必须先 `close()` 旧的再建新的。iOS 对同时存在的 AudioContext 数量有硬上限（约 4–6），超限后 `ctxFactory()` 直接抛异常 → 组件必须 catch 并且不阻塞计数（计数与音频解耦，音频失败不抛到 `applyDelta`）。

---

### C. 运动会话（新功能）

#### C1 存储模型（独立键，零迁移风险）

```
counter_v4_state        保持不变（saveState 逐字段枚举，语义不动）
counter_v4_records      保持不变（normalizeRecords 重建逻辑不动）
counter_v4_training     新增 ← 本方案唯一新增键，旧版本代码完全不会读到/写到它
```

`counter_v4_training` 结构（`version` 用于未来演进；读入时走容错归一化，任何异常输入回退到空结构）：

```jsonc
{
  "version": 1,
  "prefs": {
    "trainingMode": false,          // 全屏训练层开关（会话结束自动关）
    "goalMode": "count",            // "count" | "duration"
    "goalValue": 0,                 // 0 = 本次不设目标（自由训练）——独立按次目标，见 C1.5
    "lastGoalValue": 0,             // 上一次会话用过的目标，仅用于「上次：200」的 UI 提示
    "voiceCue": true,               // 训练中是否语音/音效节拍提示（可选实现）
    "lockExitLongPressMs": 700,
    "wakeLockEnabled": true         // 可选加分项（D4）；不支持时静默降级
  },
  "active": null | {
    "id": "s-1757980000000-ab3f",
    "startedAt": 1757980000000,     // 墙钟 epoch ms
    "startDate": "2026-09-16",      // 归属日 = 开始那天（跨零点不换归属）
    "goal": { "mode": "count", "value": 100 },
    "countAtStart": 137,            // todayCount 快照；会话计数 = todayCount - countAtStart（含撤销修正）
    "pausedMs": 0,                  // 显式暂停累计
    "backgroundMs": 0,              // 后台/息屏累计
    "activeSinceMs": 1757980000000, // 当前这段「前台且未暂停」的起点；暂停/后台时置 null
    "pausedSinceMs": null,
    "hiddenSinceMs": null,          // 本段进入后台的时间；是 lastHiddenAtMs 的镜像，active 结束即随之消失
    "lastHitAtMs": null,            // 最后一次有效敲击时间（用于 idle 判定的收尾时间点）
    "speedPeak": 0,
    "splitFrom": null,              // 跨零点拆分时指向上一段 id
    "endReason": null,

    // ↓ B5 新增：会话级有上限敲击日志（内存态，不落盘）
    //   【v3 / 边界 1 修正】写入时机 = applyDelta() 里 actualDelta !== 0 的分支（见 C1.7），
    //   因此 delta 可正可负、且末条 delta 恒等于 lastActionDelta（撤销可安全弹出）
    "taps": [ { "at": 1757980001234, "delta": 1, "kind": "tap" } ],
    "tapsTotal": 103,               // 会话内累计敲击事件数（含已被 FIFO 截断掉的）
    "tapsDropped": 0                // 因上限被丢弃的条数
  },
  "sessions": [ /* 已完成会话，结构见下；按 startedAt 倒序；上限 200 条 */ ],

  // ↓ 【v3 / 非阻塞 2 新增】store 级（与 active 的生命周期解耦），用于「回前台」两阶段顺序
  "lastHiddenAtMs": null,           // 最近一次进入后台 / 路由离开的墙钟时间；active 结束时【不清】
  "pendingSplitFrom": null          // 后台跨零点已收尾但未续开时的上一段 id；回前台消费后置回 null
}
```

**为什么需要这两个 store 级字段（非阻塞 2 的核心）**：`active.hiddenSinceMs` 随 `active` 一起消失，而「后台跨零点」这条路径恰恰会先把 `active` 收尾 —— 收尾后就没有任何地方记得「我们是从几点开始离开前台的」。`lastHiddenAtMs` 补上这个记忆；`pendingSplitFrom` 记住「昨天那段已经按 `midnight` 收尾了，回前台该补开新段」。两者的写入时机见 §C1.8 与 §B4。

已完成会话（`sessions[]` 元素）：

```jsonc
{
  "id": "s-1757980000000-ab3f",
  "startedAt": 1757980000000,
  "endedAt": 1757983600000,      // 必填
  "startDate": "2026-09-16",
  "count": 103,                  // 会话内计数（净增，含撤销修正，下限 0）
  "durationMs": 3600000,         // = endedAt - startedAt（墙钟，「实际时长」）
  "activeMs": 3120000,           // 扣掉显式暂停 + 后台/息屏（「有效时长」）
  "pausedMs": 180000,            // 显式暂停累计
  "backgroundMs": 300000,        // 后台/息屏累计
  "goal": { "mode": "count", "value": 100 },
  "reached": true,
  "speedPeak": 148,              // 次/分
  "endReason": "manual",         // manual | idle | midnight | reset —— v2 收敛取值域，见 C1.9
  "splitFrom": null,
  "tapsTotal": 103,
  "tapsDropped": 0,
  "taps": [ /* 仅当未决 #8 定为「允许结束后改单次敲击」时才写入，见 C1.7；条目结构与 active.taps 一致（含 kind） */ ]
}
```

**设计要点（逐条对应需求 A/B/C 的追问）**

1. **计数口径不叠加**：`session.count` = 该时段内 `todayCount` 的净增量，**不写入** `dailyRecords`、**不进** `grandTotal`/`streak`/`weekData`/`calCells`。这是硬红线——两者叠加就是双算。
   - 实现上以 `countAtStart` 快照 + 当前 `todayCount` 求差，天然覆盖 `undoLast()`（撤销也同步修正会话计数），下限 clamp 0（`resetToday()` 后差为负，会话计数归 0，且 `endReason='reset'` 直接结束会话）。
   - **不变量**：对任意日期 D，`Σ sessions(startDate=D).count ≤ 当日归档 count`。仅作为 UI 呈现约束与测试断言，不做运行时强制（撤销/重置可破坏）。
2. **`durationMs` vs `activeMs` 双口径**：需求要的是「实际开始 / 结束 / 时长」，所以主展示 = `durationMs`（墙钟，真实起止），次要展示 = `activeMs`（有效训练时长）。两者都在模型里。
   - **派生统计口径（未定义边界补充）**：本次**不做**周/月训练时长统计。若未来新增，口径一律取 `Σ activeMs`（有效时长），**不得**用 `durationMs`——否则用户切后台 30 分钟会看到虚高时长。这条写进本方案作为后续约束。
   - 会话总结面板**必须同时给出两个数**：`实际时长 1:00:00 · 有效时长 52:00`。
3. **跨零点**（`checkDayRollover`，`CounterTool.vue:1108-1123`，**实现顺序见 C1.8**）：
   - 会话归属日 = `startedAt` 所在日，**跨零点不换归属**。
   - 检测到日期翻转且存在 active 会话时：以 `endedAt = 当日 23:59:59.999` 收尾（`endReason='midnight'`）。
   - **新段的 `countAtStart` 恒为字面量 `0`**，**不是**「取当前 `todayCount`」（v2 原文，**v3 更正 —— 非阻塞 1**）。理由：按 §C1.8 的实现顺序，`handleRollover()` 执行时 `todayCount` **还是昨天的值**（清零发生在它之后的 `CounterTool.vue:1117`）。若照 v2 字面写成 `countAtStart = todayCount.value`，新段一开始就背上整天的存量（例：昨天 500 → 新段 `count = 0 − 500 = −500`，被 clamp 成 **0**），**跨零点后的计数会全部丢失**。
     > `:855` 的 L1 #12 只能拦住「计数为 0」这一种**表现**，拦不住这个**写法差异**（写法对了但顺序错了、顺序对了但取值错了，都可能表现成 0）。故 v3 新增 **L1 #25a** 直接断言新段 `countAtStart === 0`。
   - 只在**前台**检测到翻转时才自动续开（守卫见 C1.8）；后台睡过零点则按「idle 规则」处理（不会出现一段横跨 8 小时的假会话）——**「后台睡过零点」的完整口径见 C1.4(v3) 与 C1.8(v3)**。
4. **路由离开 / 息屏**（**【v3 / 非阻塞 2 重写：回前台的两阶段顺序】**）：
   - 先修 keep-alive 缺陷（见 C4）：`onDeactivated` / `visibilitychange hidden` **不结束会话**，只把当前「活跃段」结算进 `activeMs`、置 `activeSinceMs = null`，并**同时写两个字段**：`active.hiddenSinceMs = now` 与 store 级 `lastHiddenAtMs = now`。
   - **回到前台时严格按 (a) → (b) → (c) 三步，顺序不得调换**（`handleForeground()` 的契约）：
     - **(a) 先结算 idle**（只在 `lastHiddenAtMs != null` 时执行）：`away = now − lastHiddenAtMs`。
       - `away > SESSION_IDLE_MS`（默认 **30 分钟**，常量）→ 若 `active` 存在则收尾：`endedAt = max(active.lastHitAtMs ?? active.hiddenSinceMs, active.hiddenSinceMs)`、`endReason = 'idle'`、`active = null`；**并清 `pendingSplitFrom = null`**（已经按 idle 结束了，不再补开）。**进入 (b) 时 `active === null`；midnight 分支不会再自动开新段。**
       - `away ≤ SESSION_IDLE_MS` → 若 `active` 存在则续跑：`backgroundMs += away`、`activeSinceMs = now`、清 `active.hiddenSinceMs`。**不清 `lastHiddenAtMs`**（(c) 还要用）。
     - **(b) 再跑 `checkDayRollover()`**（处理「回前台这一瞬间才发现跨了零点」的常规路径，走 §C1.8 的顺序与前台守卫）。
     - **(c) 最后消费 `pendingSplitFrom`**：若 `pendingSplitFrom !== null` **且** `active === null` **且** 第 (a) 步判定为「未超 idle」→ **补开新段**：`startedAt = 跨零点那天的 00:00:00.000`、`countAtStart = 0`、`splitFrom = pendingSplitFrom`、目标沿用旧段；然后 `pendingSplitFrom = null`。其余情况（超 idle / 已有 active）一律只清 `pendingSplitFrom`。
   - **为什么必须「先 idle 再 midnight」**：`checkDayRollover` 的前台守卫是 `isActive && document.visibilityState === 'visible'`，而它在回前台后被求值时 `visible` **已经成立**——它**无法区分「跨零点时人在前台」与「跨零点后人才回前台」**。若先跑 (b)，一段睡了 8 小时（23:50 离开、次日 07:50 回来）的会话会被当成「前台跨零点」而**自动续开**，产出「昨晚那段 + 今早那段」的假连续训练。先跑 (a) 就把它按 `idle` 结束掉，(b) 的前台守卫分支再也不会被误触发。
   - **为什么还需要 `pendingSplitFrom`（(c)）**：1s 时钟（`CounterTool.vue:1545-1548`）在后台**仍在跑**（`setInterval` 不受后台影响），完全可能在人不在前台时先跑完 `checkDayRollover()`：那时它按 §C1.8 的前台守卫**只收尾、不续开**，并把 `pendingSplitFrom` 置为旧段 id。人回来时 `active` 已是 null，若不查这个标记，**跨零点后的训练就会静默丢弃续段**。这一条是 v3 新定义的边界。
   - `onDeactivated`（路由离开）与 `visibilitychange hidden` 走**同一套**规则（同样写 `lastHiddenAtMs`），行为一致，**不引入第二种语义**——所以 `endReason` 里**没有** `route-leave`（C1.9）。
5. **会话目标（B6 —— 默认值必须独立于每日目标）**：
   - `goal: {mode:'count'|'duration', value}`，`value = 0` 表示本次不设目标（自由训练）。
   - `dailyGoal` 语义与 UI **完全不动**；会话目标是独立的一次性目标，**不是**日目标的派生值。
   - **默认值（本版定稿）**：`prefs.goalValue` 默认 **`0`（自由训练）**。会话开始面板提供 `100 / 200 / 500 / 自定义 / 0`，并高亮 `prefs.lastGoalValue`（「上次：200」）作为一键选项；用户选完即写回 `lastGoalValue`。
   - **禁止**：`dailyGoal`、`todayCount` **不得**出现在任何 `goal.value` 的赋值路径上。原 v1 的 `max(0, dailyGoal - todayCount)` 已作废（PM 于 FINN-16 明示撤回；jaxiu 2026-09-16 03:36 裁决「按每次运动设目标」）。
   - 允许的**只读参考展示**：会话开始面板可显示一行灰字 `每日目标 300 · 今日 80`，纯展示、不参与计算、**不预填输入框**。`dailyGoal = 0` 时显示「未设每日目标」。
   - `reached` 为派生值：`count` 模式 = `count >= value`；`duration` 模式 = `activeMs >= value * 60000`；`value === 0` 时 `reached` 恒为 `false`（自由训练不判定达标）。
6. **计数节奏口径（需求 3 第一点）**：现有 15s 窗口 + 1.8s 跨度下限（`CounterTool.vue:1185-1200`）在运动场景下会虚高。会话内新增**会话级节奏**：`session 平均 = count / (activeMs/60000)`，并在会话总结里同时给出 `speedPeak`（沿用现有窗口峰值，保证与「峰值速度」卡片口径一致）。**不改** `refreshSpeed()` 现有行为。
7. **会话级敲击日志（B5 —— 时间轴的数据源）**：
   - **为什么不能用 `tapEvents`**：它是 15s 滑动窗口（`CounterTool.vue:727 / 1181 / 1187`，每次 `refreshSpeed()` 就把 15s 之前的过滤掉）。会话结束时快照到的只有最后 15 秒——30 分钟训练的时间轴几乎是空的。
   - **模型**：`active.taps: [{ at, delta, kind }]`，`MAX_SESSION_TAPS = 500`（常量）。`kind ∈ 'tap' | 'minus' | 'auto' | 'undo'`（**v3 新增字段**）。
   - **写入时机（【v3 / 边界 1 修正】必须下移）**：v2 把 `taps` 的写入挂在 `registerHit()` 里（`CounterTool.vue:1181-1186`），而 `registerHit` **只在 `actualDelta > 0 && shouldFeedback` 时被调用**（`applyDelta`，`:1148-1150`）→ 减号、键盘 ↓、自动连点都不进 `taps`，于是 `taps` 与 `lastActionDelta` 会脱节。v3 定稿：**把写入点下移到 `applyDelta()` 里 `actualDelta !== 0` 的分支**（`CounterTool.vue:1144-1152`），并带上 `kind`。
     **由此得到一条可断言的不变量**：**`active.taps` 的末条 `delta` 恒等于 `lastActionDelta`**（`CounterTool.vue:1157`）。`undoLast()`（`:1168-1175`）撤销的正是 `lastActionDelta`，所以「撤销 = 弹出末条」在**所有**路径上都精确成立。L1 #25b 断言这条不变量。
   - **`kind` 的用途**：时间轴按 `kind` 渲染（`tap` 正常条目；`minus`/`auto` 标注来源；`undo` 视实现决定是否展示）。训练层内 `minus` / `auto` 不可达（③ 清单不渲染入口，**【v4 / B10】且二者的键盘入口已由 `resolveShortcut` 在 `trainingMode` 下屏蔽** —— 原有的「不可达」结论在 v4 前**只对 UI 入口成立**，见 §D2⑦-b），所以训练时间轴正常只会出现 `tap`——带 `kind` 只是为了**日间模式会话**与**键盘操作**下日志仍然自洽。
   - **溢出策略**：FIFO，丢弃最旧的；`tapsDropped++`，`tapsTotal++`。时间轴 UI 在列表底部显示「仅显示最近 500 次，更早的 N 次未列出」（`tapsDropped > 0` 时）。
   - **为什么是 500**：30 分钟高强度训练（4 次/秒）= 7200 条；每条 `{at,delta,kind}` 序列化约 30 字节 → 7200 条约 216KB，单键整写会拖慢敲击路径并逼近 5MB 配额。500 条约 15KB，配合 200 条会话（每条**不内嵌** taps）总占用 < 250KB。
   - **落盘策略**：`active.taps` **不落盘**（内存态）；`counter_v4_training` 只写 `tapsTotal` / `tapsDropped`。理由：① 时间轴只在会话结束后需要；② 避免 5s 节流落盘时序列化 15KB 数组。
   - **若未决 #8 定为「允许结束后改单次敲击」**：在 `end()` 时把 `taps` 快照**一次性**写进 `sessions[]` 的那一条（同样受 500 上限与 FIFO 截断），此后不再更新。
   - **删一条的语义**：从 `taps` 移除该条 → `todayCount -= delta`（下限 0）→ 派生值 `session.count` 自动同步。`delta` 可正可负（`minus` 条目为负），所以这里是**带符号**的加减，不是「一律减」。若目标条目已被 FIFO 截断，UI 不提供入口（时间轴只列 `taps` 中的条目）。
   - **`undoLast()` 与会话的同步（【v3 / 边界 1 定稿】）**：`undoLast()` 在会话内**同步弹出 `active.taps` 末条**，`session.count` 因为是派生值（`todayCount − countAtStart`，`:510`）自动跟随。**不采用「接受偏差」**——靠上面那条不变量，弹出永远精确。
     - 日间模式下 `taps` 仍会照常累积（只要 `active` 存在）；若 `active === null`，`undoLast()` 只改 `todayCount`，不涉及日志。
8. **跨零点拆分的实现顺序（非阻塞 3 —— 顺序错了会丢数据）**：
   ```js
   function checkDayRollover() {
     // ① 先取会话快照（此刻 todayCount 还是「昨天」的值）
     const sessionSnapshot = session.countAtRollover(todayCount.value)   // = todayCount - active.countAtStart
     // ② 会话收尾 / 拆分（endReason='midnight'；新段 countAtStart 恒为字面量 0，见 C1.3）
     //    前台守卫在这一步求值；不在前台时只收尾、不续开，并把 pendingSplitFrom 记为旧段 id
     const autoContinue = isActive.value && document.visibilityState === 'visible'
     session.handleRollover(now, { autoContinue })
     // ③ 最后才 archiveDay + 清零（现有 CounterTool.vue:1115-1123）
     archiveDay(currentDate.value, todayCount.value, bestSpeed.value)
     currentDate.value = todayKey
     todayCount.value = 0        // ← CounterTool.vue:1117
     ...
   }
   ```
   必须在 `todayCount.value = 0`（`CounterTool.vue:1117`）**之前**取快照，否则会话计数恒为 0。
   「只在前台才自动续开」需要守卫：1s 时钟在后台**仍在跑**（`CounterTool.vue:1545-1548`，`setInterval` 不受后台影响），所以续开分支必须写成 `if (isActive.value && document.visibilityState === 'visible')`。
   **`handleRollover(now, { autoContinue })` 的两个分支（【v3 / 非阻塞 2 补完】）**：
   - `autoContinue === true`（人在前台看到翻转）→ 旧段 `endedAt = 当日 23:59:59.999` / `endReason='midnight'`，**立即**开新段（`startedAt = 次日 00:00:00.000`、`countAtStart = 0`、`splitFrom = 旧段 id`、目标沿用）；`pendingSplitFrom` 保持 `null`。
   - `autoContinue === false`（后台跑到的这次翻转）→ 旧段仍以同样的 `endedAt` / `endReason='midnight'` **收尾**（快照已在①取到，计数不会丢），但**不开新段**；改为置 store 级 `pendingSplitFrom = 旧段 id`，等 §C1.4(c) 回前台时消费。
   - `active === null`（本来就没有会话）→ `handleRollover` 对会话部分 no-op，只写 `pendingSplitFrom = null`。
9. **`endReason` 取值域（非阻塞 2 —— 收敛为 4 个）**：`manual | idle | midnight | reset`。
   - **删除 `route-leave`**：C1.4 已定「路由离开与切后台走同一套规则、当时不结束会话」，若真的超过 idle 阈值收尾，落到的 reason 就是 `idle`，不需要单独取值。
   - 各值触发点：`manual` = 用户点「结束训练」并二次确认；`idle` = 超过 `SESSION_IDLE_MS` 未回前台；`midnight` = 跨零点拆分；`reset` = `resetToday()` / `resetAll()` 打断。
   - `active.endReason` 在会话进行中恒为 `null`。

#### C2 存储写入频率与配额

- **禁止每次敲击写 `counter_v4_training`**。内存态为权威，落盘时机：① 1s 时钟里节流（距上次落盘 ≥5s 且计数有变化）；② `visibilitychange hidden`；③ `pagehide` / `beforeunload`；④ 会话开始/暂停/结束/拆分等状态迁移；⑤ `onDeactivated`；⑥ **【v3 / 非阻塞 7 新增】`clearHistory()`（用户清空历史）**——它会清空 `sessions`（见 §D3），若不在同一函数内落盘，「清完立刻杀进程」会让被清掉的会话在下次打开时复活。
  - 注：`counter_v4_state` 仍走既有 `watch → saveState()`（`CounterTool.vue:906-911`，每次敲击写盘）——**本方案不动它**，避免牵扯日统计回归。
- 用**单键整体写**（`counter_v4_training` 一次性 `setItem`），localStorage 单键写入是原子的，避免多键写到一半崩溃。
- 配额保护：`setItem` 包 try/catch；捕获到配额异常 → 丢弃最旧的 50 条会话后重试一次；仍失败则静默放弃（绝不抛到敲击路径）。
- 上限：`sessions` 保留最近 **200 条**（与 `MAX_HISTORY_DAYS=365` 的日历史解耦，200 条约 40KB，安全）。

#### C3 「会话」与现有统计的关系（一句话结论）

> 会话是**日统计的一个视图切片**，不是新的计数源：会话计数永远是对 `todayCount` 增量的投影；`dailyRecords` / `streak` / `weekData` / `calCells` / `grandTotal` / `bestSpeed` 的计算代码一行不改。

#### C4 顺带修复：keep-alive 导致的跨页误计数与监听累积（必修）

```js
const isActive = ref(true)
onActivated(() => { isActive.value = true;  checkDayRollover(); refreshSpeed(); persistAll() })
onDeactivated(() => { isActive.value = false; stopAuto(); pauseSessionForBackground(); persistAll() })
onBeforeUnmount(() => { /* 同 onUnmounted 的摘除逻辑 */ })
```

- 在 `onKeyDown` 首行加 `if (!isActive.value) return`，在 `applyDelta` 首行加同样的守卫（防止 `autoTimer` 与残留监听继续计数）。
- **监听必须显式摘除（非阻塞 4）**：`§C4` 原写「keep-alive 下组件常驻，监听只注册一次」——这在 `viewRefreshKey` 递增路径下**不成立**：`goRoute()` 同路径点击（`App.vue:445`）与 `refreshCurrentViewAfterIdle()`（`App.vue:481-484`）都会让 `currentViewKey` 变化，而 `keep-alive` 按 key 缓存**旧实例**，新实例会**再注册一遍** `document` 监听，旧实例的监听不会自动摘。因此：
  - 摘除逻辑抽成 `removeGlobalListeners()`，在 `onDeactivated()` 与 `onBeforeUnmount()` 里各调一次；
  - 注册逻辑抽成 `addGlobalListeners()`，在 `onMounted()` 与 `onActivated()` 里各调一次，**幂等**（先摘再注册，或用一个 `listenersBound` 标志）。
  - 参考既有写法：`MiniMaxStudioTool.vue:1237 / 1241 / 1245`。
- 附带收益：`onDeactivated` 里 `stopAuto()` 修掉了「离开页面后自动连点仍在跑」的缺陷。

---

### D. 防误触

#### D1 真实误触来源（逐条，含证据）

| # | 来源 | 证据 | 后果 |
|---|------|------|------|
| 1 | 主按钮落在全宽敲击区内 | `CounterTool.vue:166-171` | 无（**这是计数本身**）。v2 明确：不为此加任何门限 |
| 2 | 减号按钮落在敲击区内的拇指休止位 | `CounterTool.vue:2555-2564`（`bottom:14px; left:14px`，与全宽 `.figure-button` 重叠） | 左手握持时拇指一按就减，直接破坏计数 |
| 3 | 步长 chip 紧贴敲击区上方，单击即改倍率 | `CounterTool.vue:143-154` | 误触后从 +1 变 +20，后续每次敲击都错 20 |
| 4 | 自动连点开关离敲击区近 | `CounterTool.vue:195-203 / 256-264` | 误开后计数器自增，计数完全失真 |
| 5 | 撤销单击即生效 | `CounterTool.vue:192 / 253` | 抹掉刚敲的一次 |
| 6 | 目标达成模态弹出挡住敲击区 | `CounterTool.vue:279-293`（`close-on-click-modal=false`） | 训练中被挡住，连续几次敲击全部丢失（不计也不减） |
| 7 | 路由离开后监听仍活着 | `App.vue:190-193 / 445 / 481-484` + `CounterTool.vue:1443-1465`（无 `onDeactivated`） | 别的页面按空格 → 计数器静默 +N（见 C4） |
| 8 | 贴屏边敲击被系统手势吃掉 | iOS/Android 边缘返回手势 | 敲了但不计数（或直接退出页面） |
| 9 | 训练中屏幕自动熄灭（真机必然发生） | 无 Wake Lock；屏幕 30s–2min 自动灭 | 需求 2 的「息屏后没声音」在真机上**必然复现**（见 D4 可选加分项） |

#### D2 方案

**① 计数面 = 一击即计（B1 —— 硬口径，最高优先级）**

> **判定依据（jaxiu 2026-09-16 03:36 裁决）**：**计数准确 > 防误触**；防误触**只对破坏性操作**（结束 / 暂停 / 减号 / 撤销 / 重置 / 清零 / 切换形象）设防；**任何会吞掉有效敲击的设计直接否决**；主按钮必须一击即计、瞬时响应。

**计数面（主按钮）的最终规则**：

```html
<!-- CounterTool.vue:166-171 —— 保持不变，仅补手势解锁调用 -->
<button type="button" class="figure-button" :class="{ bump: isBumping }"
        @pointerdown="handleFigureTap">
```
```js
function handleFigureTap() {
  increment()     // ← 先计数，一击即计，同步、瞬时
  unlockAudio()   // ← 后解锁音频；同步返回，不 await、不阻塞计数
}
```

- **`pointerdown` 即提交计数**，保持今天 `handleFigureTap → increment()`（`CounterTool.vue:170 / 1133-1140`）的语义。
- **位移 / 时长 / `isPrimary` / 去抖四道门限一律不得作用于计数面**。逐条否决理由（对应 FINN-11 给出的 5 类丢数场景）：

| 被否决的门限 | 丢数场景 | 结论 |
|-------------|---------|------|
| 时长 ≤ 1200ms | 力竭、感受节拍、换手间隙时按住 >1.2s 再抬起 → 今天计 1，加门限后计 0 | 删除 |
| 位移 ≤ 12px | 运动场景手机本身在晃、手指微滑 → 今天计 1，加门限后计 0 | 删除（`touch-action: none` 已经消除了「滑动」这个风险本身） |
| `end.isPrimary === true` | 掌缘/第二根手指先落屏 → 后续**所有**真实敲击都非 primary → 连续丢数 | 删除 |
| `pointerup` 必须落在敲击面内 | 无 `setPointerCapture` 时抬起点漂出边界 → 静默丢弃 | 删除（改为按下提交，抬起完全不参与） |
| 60ms 去抖 | 10 次/秒不会触发，但**和上面 4 条叠在同一条提交路径上**，任一条 false 就是「敲了不计」 | 删除 |

- **`pointercancel` 的处置口径（v2 新增）**：iOS 边缘返回手势、来电、系统弹窗、多指竞态都会产生 `pointercancel`。因为计数已在 `pointerdown` **提交且不回滚**，`pointercancel` 在计数面上是 **no-op**。它只有两个用途：① 取消「结束训练 / 暂停」的**长按定时器**；② 清除长按进度 UI。`pointerup` 同理——计数面不监听它。
- **多指（接受的取舍）**：两个手指同时按下 = **+2**（= 今天的行为）。加 `isPrimary` 门限会吞掉掌缘先落屏后的所有真实敲击，违背硬口径；防误触靠「全屏层内没有别的东西可误触」，不靠过滤 pointer 事件。L3 验收按 +2 判（原 v1 的「双指 → 只 +1」条目作废）。
- **不做「滑动回滚」**：按下即计数、不回滚 = 今天的行为。`touch-action: none` 已经不产生滚动手势，所以不存在「滑动误计」这个类别，`备选方案 A`（按下计数 + 滑动回滚）整段删除。
- **60ms 去抖整段删除**，`§兼容性与迁移` 里「连续两次间隔 <60ms 的敲击会合并」这条行为变化也随之删除。

**防误触只剩两条**：

1. **结构上无可误触**：训练层内**不渲染**减号 / 步长 chip / 自动连点 / 撤销 / 形象主题 / 日历历史设置入口（见 ③ 清单）。误触来源 #2–#5 是靠「那些控件根本不存在」消掉的，不是靠事件过滤。
2. **破坏性操作加长按**：见 ②。

**`counterTapGuard.js` 的最终契约（v2 重写）**

```js
// frontend/src/utils/counterTapGuard.js —— 只服务「破坏性操作」的长按判定
export const LONG_PRESS_MS = 600        // 暂停 / 继续
export const EXIT_LONG_PRESS_MS = 700   // 结束训练
export const LONG_PRESS_MOVE_TOLERANCE_PX = 12   // 手指滑出即可取消长按（只用于破坏性操作）

// 纯函数，可单测
export function isLongPressCommit({ startAt, endAt, thresholdMs }) {
  return endAt - startAt >= thresholdMs
}

// 有状态守卫（组件用）
export function createLongPressGuard({ thresholdMs, now = () => performance.now() }) {
  return {
    start(pointerId),          // pointerdown 时调用，记录 startAt
    cancel(pointerId),         // pointerup / pointercancel / 位移超阈值时调用
    isArmed(pointerId),        // 供 UI 显示长按进度
    takeCommit(pointerId)      // 提交时调用；返回是否达到阈值
  }
}
```

**必须删除**：`TAP_MOVE_TOLERANCE_PX`、`TAP_MAX_DURATION_MS`、`TAP_DEBOUNCE_MS`、`shouldCommitTap`。它们原本服务的计数面已经不需要任何门限；保留它们只会诱导实现方把门限加回去（L2 负向 grep 会拦）。

**② 训练模式 = 全屏沉浸层（同时解决需求 4 与需求 1 的空间问题）**

- 新组件 `frontend/src/components/CounterTrainingOverlay.vue`（引用惯例：`../../components/X.vue`，见 `MiniMaxStudioTool.vue:907`）
- **挂载点（非阻塞 7，v2 写死）**：挂在 `CounterTool.vue:2-6` 的 `.counter-app` 根 `<div>` **内**，作为最后一个子节点。**不使用 `<Teleport>`**——因为主题 CSS 变量（`themeVars`，`CounterTool.vue:741+`）是绑在 `.counter-app` 上的 `:style`，Teleport 到 `body` 会丢失变量继承。
  - **随附约束（必须写进代码注释）**：`position: fixed` 要逃出滚动/裁剪，依赖从根到 `.counter-app` 这条链上**没有任何元素创建 containing block**。已核实 `App.vue:548-553`（`.app-container`）、`App.vue:1032-1042`（`.main-content`）、`App.vue:1044-1051`（`.mobile-main`）、`CounterTool.vue:1568-1576`（`.counter-app`）**都没有** `transform` / `filter` / `perspective` / `contain` / `will-change` / `backdrop-filter`（`grep` 已确认，`App.vue` 中的 `transform` 命中全在 `.pb-*` 播放器栏、`backdrop-filter` 在 `.player-bar`，均非祖先）。
  - **后续若给外壳加 `transform`，训练层会静默失效**（不再全屏、被裁剪）。加一条 L2 `grep` 防线（见测试要点）。
- **【v3 / B8 定稿】层级：`position: fixed; inset: 0; z-index: 10050`**，自带 `padding: env(safe-area-inset-*)`。
  - **为什么是 10050**：§现状 5 已 grep 核实**全仓库常驻产品浮层的最大 `z-index` 是 `10000`**（`.player-bar` `App.vue:1170`、`.cheer-toast` `GlobalPet.vue:470`），其次是 `9999`（`.pet-launcher` 等）与 `9998`（`.pet-window`）。`10050` 是**唯一一个同时高于全部常驻浮层、又留出 50 的整数间隔**的取值；唯一的例外是 `main.js:55` 那条 `99999` 的开发态报错条（非产品层级，不在考量内）。
  - **v2 的 `z-index: 900` 是错的**：训练层是 `.counter-app`（`CounterTool.vue:1568-1576`，`position: relative` 且 **`z-index: auto`，不创建层叠上下文**）的后代，所以它的 `900` 与 `App.vue` / `GlobalPet.vue` 上那些浮层处在**同一个（根）层叠上下文**里被比较 → **900 全部被盖**。后果是训练层底栏（「暂停/继续」「结束训练」）被 `.pet-launcher` / `.player-bar` 压住，而「结束训练」是层内**唯一退出路径**（用户只能刷新页面），同时这些浮层的可点控件在训练期成为**未设防的误触面**。
  - **处置口径（把「全局 fixed 浮层」当成一类，不逐个打补丁）**：**统一由训练层盖住**，不做「训练期隐藏浮层」的方案。理由：隐藏需要操作 `App.vue` 的 `.player-bar` 与 `<GlobalPet />`（`App.vue:239-296 / 299`），而 §硬约束明令**不改 `App.vue`**；想在 `CounterTool.vue` 里跨组件隐藏只能靠 `document.querySelector('.pet-window').style.display='none'` 这类 DOM 越权操作，不可接受。抬高 `z-index` 是**局部、确定、可 grep 断言**的解法。
  - **由此产生的必然结果（必须写进 §D2⑦ 与风险表）**：训练层**也高于 Element Plus 的弹层**（popup manager 从 `2000` 起）。所以 v2 「低于 EP 弹层 2000」的表述作废——正确口径是「**训练层盖住一切，含 EP 弹层**」，这与「进层即锁 UI」是一致的。代价是训练层内**不能再用 `ElMessageBox` / `ElMessage` 做任何反馈**（会被自己盖住看不见），详见 §D2⑦。
  - **覆盖清单（reviewer 按此核对，不得只处理其中一个）**：`.pet-window`(9998) · `.pet-launcher`(9999) · `.cheer-toast`(10000) · `.player-bar`(10000) · 其余 `9999` 的页面级浮层（ImageViewer / ExpenseTool / VirtualAvatar / Mermaid，它们与本模块不同时可见，仅作完整性记录）。
- 结构：顶部信息条（计数 / 目标进度 / 已用时长 / 节奏）+ 中部**整块敲击面** + 底部「暂停/继续」「结束训练」
- 层内 `touch-action: none; overscroll-behavior: none; user-select: none` → 不可能产生滚动/缩放手势
  - **注意（非阻塞 9）**：`touch-action: none` 会让层内**任何**区域都不可滚动 → **会话总结面板不放在层内**。结束训练 → 退出全屏层 → 在日间模式用现有 `panel-overlay`（`showHistory` 那一套）展示总结 + 时间轴。层内只有顶栏 + 敲击面 + 底栏，都不需要滚动。（若将来必须在层内展示，必须给该 `div` 单独 `touch-action: pan-y; overflow-y: auto`，本方案不采用。）
- **锁定态下如何仍然计数**：计数面本身**不加**任何长按/确认门槛——`pointerdown` 即计数（B1）。锁定只作用于**退出/破坏性操作**（结束训练、暂停需要长按）以及**隐藏/禁用**其它所有控件。锁定的语义是「**锁 UI，不锁计数**」。

**③ 训练期禁用 / 降级清单（需求 4 明确要求）**

| 控件 | 训练模式（全屏层内） | 理由 |
|------|-------------------|------|
| 主敲击面 | **保留**，全屏、无任何确认门槛，`pointerdown` 即计数 | 计数（B1 硬口径） |
| 结束训练 | 保留，**长按 700ms** + 二次确认「结束并保存本次训练？」——**确认 UI 必须由 `CounterTrainingOverlay` 层内自绘**（层内确认条 / 长按第二段），**禁止**用 `ElMessageBox.confirm`（会被训练层自己盖住，见 ⑦） | 破坏性/会话边界 |
| 暂停 / 继续 | 保留，**长按 600ms** | 运动中单指长按比精准点小按钮容易，且不会误触 |
| 撤销（撤销上一次敲击） | **禁用**（层内不出现） | 运动中几乎必然是误触；改到「结束后总结面板」里按时间轴逐条修正 |
| 减号 | **禁用**（层内不出现） | 误触来源 #2 |
| 步长 chip | **禁用**（冻结当前 step，并显示「步长已锁定 +N」） | 误触来源 #3；改步长需先结束训练 |
| 自动连点 | **禁用**，进入训练时若开着则强制 `stopAuto()` + 提示 | 误触来源 #4；自动连点不是真实敲击，不应计入训练 |
| 形象 / 主题切换 | 禁用（层内不出现） | 训练中无关 |
| 日历 / 历史 / 设置入口 | 禁用（层内不出现） | 训练中无关，且弹层会遮挡（见 ⑦） |
| 目标达成反馈 | **降级**：模态 → 层内横幅 + 震动脉冲（不阻塞敲击） | 误触来源 #6 |
| 音效 / 震动 | 保留（沿用现有开关） | 训练反馈核心 |
| 日统计（今日次数/周热力/连续达标） | 移动端隐藏；桌面端保留在层外 | 空间回收（A3） |

**④ 边缘手势规避**：训练层左右各留 `max(24px, env(safe-area-inset-left/right))`；敲击面有效区不贴边。

**⑤ 目标达成反馈改造**：训练模式内不使用 `el-dialog`（`CounterTool.vue:279-293`），改为层内顶部横幅 3s 自动消失 + `navigator.vibrate([30,60,30])`；**日间模式保持模态不变**（不破坏桌面端既有体验）。

**⑥ 撤销的重新定位（B5）**：保留现有 `undoLast()`（`:1168-1175`）语义不变；训练结束后在**日间模式**的总结面板提供「敲击时间轴」。
- **数据源 = `sessions[i].taps`（会话结束后）或 `active.taps`（进行中），不再是 `tapEvents`**（15s 滑动窗口不可用，见 C1.7）。
- 允许删除某一条 → 同步 `todayCount -= delta`（**带符号**，`minus`/`auto` 类条目 delta 可为负；下限 0，保持 §C1 的计数不变量）。
- `tapsDropped > 0` 时面板底部显示「仅显示最近 500 次，更早的 N 次未列出」。
- **与 `undoLast()` 的分工（v3 / 边界 1）**：`undoLast()` 只撤**最后一次**计数动作（弹出 `taps` 末条，见 C1.7）；时间轴删除可以删**任意**一条。两者是独立路径，都通过 `todayCount` 的增减保证 `session.count` 自动同步。

**⑦ 训练层内的 `el-dialog` / 全局弹层（未定义边界，【v3 重写】）**

**层级事实（v3 定稿）**：训练层 `z-index: 10050` **高于** Element Plus 弹层（popup manager 从 `2000` 起递增）与全部常驻浮层（最大 `10000`）。→ 训练层在时，**任何 EP 浮层都会被压在下层，用户看不见**。

- **定义（保留 v2 的一半）**：进入训练层时**同步关闭所有 EP 浮层**——`showSettings = false; showCalendar = false; showHistory = false; goalReached = false`；且训练层内**不渲染**任何打开它们的入口（③ 清单已禁用）。~~因此训练中**不存在合法触发路径**~~ —— **【v4 / B10 更正】这句「不存在合法触发路径」只对 UI 入口成立**：`document` / `window` 上的监听**不受 `v-if` 影响**，必须逐条实证，见 **§D2⑦-b**（正是这句话让 v3 免掉了 `undoLast()` 的处理）。
- **定义（v3 新增的硬约束）**：**训练层存活期间（`prefs.trainingMode === true`），组件内不得调用任何 EP 全局弹层 API** —— `ElMessage` / `ElMessageBox` / `ElNotification` / `el-dialog` / `el-drawer`。原因不是「会被遮挡后仍能点到」，而是**它们会渲染在用户看不见的地方**：用户会觉得「按了没反应」，这在训练中比报错更糟。
  - **具体受影响点（实现时逐条改）**：
    - 「结束训练」的二次确认 → **层内自绘确认条**（③ 清单已写）。
    - `undoLast()` 的 `ElMessage.success('已撤销上一步')`（`CounterTool.vue:1174`）→ **【v4 / B10 更正：原「训练层内撤销本就不可达」是错的，必须处理】** 训练层存活时该函数**可以**经 `document` 级键盘快捷键到达：
      - **实证**：`document.addEventListener('keydown', onKeyDown)`（`CounterTool.vue:1549`）→ `onKeyDown` 的 `if (event.code === 'KeyZ' && (event.ctrlKey || event.metaKey))` 分支（`CounterTool.vue:1461-1464`）**直接**调 `undoLast()`。监听挂在 `document` 上，**训练层存不存活与它无关**；`onKeyDown` 现有的两个前置守卫（`:1444-1447` 的 `isActive` 与 `target.closest('input, textarea, [contenteditable="true"]')`）在训练层存活时**都不成立**（组件处于 active，且训练层按 ③ 清单**不含任何可编辑元素**）。
      - **为什么 v3 会写错**：③ 清单约束的是「层内**不渲染**撤销入口」（UI 层），它**管不到** `document` 级快捷键。用「某入口不可达」的假设替代逐入口实证，正是 B8 要消灭的形态。
      - **后果**：`todayCount`（以及边界 1 定下的 `active.taps` 末条）在**没有任何可见反馈**的情况下被静默修改 —— `ElMessage` 被 `z-index: 10050` 的训练层盖住。用户看到的是「数莫名其妙少了」，训练结束后时间轴上也少了对应条目。这是 v3 自己新写的硬口径（「训练层存活期禁止调用任何 EP 全局弹层 API」）在同一份文档里被自己豁免掉的唯一一处。
    - `resetToday()` / `clearHistory()` / `resetAll()` 的 `ElMessageBox.confirm` → 它们的入口在设置面板里，进训练层时 `showSettings` 已被关闭，**且 ③ 清单规定层内不渲染这些入口**（这一条是逐入口实证过的：调用点只有 `CounterTool.vue:370-372` 三处，均在 `v-if="showSettings"` 的面板内）。**但 D3 的「训练中点清除全部数据」路径仍需注意**：该路径会先关训练层（`prefs.trainingMode = false`）**再**弹确认框，顺序不能反。
  - **判据**：L2 负向 grep —— `CounterTrainingOverlay.vue` 内 `grep -n "ElMessage\|ElMessageBox\|ElNotification\|el-dialog"` → **无命中**；**新增 L2 #42c**（训练层存活路径的负向判据，覆盖 `CounterTool.vue`）与 **L1 #6b/#6c**（`resolveShortcut` 的纯函数断言）。
- 万一出现意外浮层（实现缺陷），**不在计数面加守卫**（那会吞计数）——记录为实现缺陷，由 code-reviewer / QA 按 ③ 清单核查。

**⑦-b 训练层存活期的 `document` / `window` 级入口清单（【v4 / B10 新增，逐条实证】）**

> **为什么单列这一节**：⑦ 的 v3 版本靠「进层时关闭全部 EP 浮层 + 层内不渲染入口」推出「训练中不存在合法触发路径」。这个推理**只在 UI 入口这一层成立** —— `document` / `window` 上的监听**不受 `v-if` 影响**，训练层存活时它们照样触发。本节把这类入口**逐条枚举并实证**（grep / 逐段 Read，命令与行号见下），每条给出**可达性结论 + 处置**。**此后本方案不得再出现「某个入口不可达」这类未实证的前提**（B10 的成因就是这一句）。

**⑦-b.1 训练层存活期的撤销门限（B10 核心 —— 二选一：**选「屏蔽」**）**

**门限**：`onKeyDown` 的 `Cmd/Ctrl+Z` 分支在**训练层存活时（`prefs.trainingMode === true`）不得调用 `undoLast()`**，因而**不产生任何 EP 全局弹层调用**。

**决策：训练层存活期整体屏蔽 `Cmd/Ctrl+Z`（不派发、不做层内提示）**，而非「层内自绘反馈」。理由逐条：

1. **与 ③ 清单的口径统一**：③ 已定「撤销」在训练层内**整体禁用**。键盘是同一功能的**第二个入口**；只屏蔽 UI 不屏蔽键盘，「训练期不可撤销」就不是无例外的硬口径，也无法被断言。屏蔽后这条规则才是**可判定的全称命题**。
2. **撤销有更强的替代路径，不存在不可替代性**：⑥ 已定训练结束后在总结面板按**时间轴逐条修正**，且**可删任意一条**（强于 `undoLast()` 只能撤最后一次）。
3. **不做「层内自绘反馈」，因为这里的反馈本身有害**：训练层是全屏敲击面，其设计前提是「锁 UI、除敲击面外无可交互物」。为一个**被明确禁用的操作**新增提示条，等于在锁定 UI 上给被禁操作做正反馈，会诱导用户继续按；且它会与 ⑤ 的目标达成横幅**争同一块层内位置**，需要新增状态 + 自动消失定时器，与 ⑦ 的「层内不新增浮层」自相矛盾。
4. **「按了没反应」在这里不构成缺陷**：⑦ 禁止 `ElMessage` 的核心理由是「用户点了一个**看得见的按钮**却没有反馈」。`Cmd/Ctrl+Z` 在训练层内**没有任何对应的可见控件**（③ 已移除），用户没有可归因的点击对象；层内「结束训练 / 暂停」的长按反馈仍然存在。
5. **成本收益**：屏蔽是 `onKeyDown` 一个分支的返回值；层内自绘要新增组件元素 + 状态 + 定时器 + 新断言，收益为负。

**实现契约（门限做成纯函数，而不是内联 `if`）**

```js
// frontend/src/utils/counterTapGuard.js —— 定位由「长按判定」扩为「破坏性输入门限」：
//   ① 破坏性操作的长按判定（v2 起，不变）；② 训练层存活期的快捷键门限（v4 / B10 新增）
export const SHORTCUT_INCREMENT = 'increment'   // Space / ArrowUp / Equal
export const SHORTCUT_DECREMENT = 'decrement'   // ArrowDown / Minus / Backspace
export const SHORTCUT_UNDO      = 'undo'        // Cmd/Ctrl + Z
export const SHORTCUT_BLOCKED   = 'blocked'     // 本模块认识的键，但当前状态下不派发动作

// 纯函数，可单测（L1 #6b / #6c）。
//   返回 null      = 不是本模块处理的键（既不派发动作，也不 preventDefault）
//   返回 'blocked' = 是本模块处理的键但当前状态禁用（**仍要 preventDefault**，只是不派发任何动作）
export function resolveShortcut({ code, ctrlKey, metaKey }, { trainingMode = false } = {}) {
  if ((ctrlKey || metaKey) && code === 'KeyZ') {
    return trainingMode ? SHORTCUT_BLOCKED : SHORTCUT_UNDO       // ← B10：训练层存活 → 屏蔽
  }
  if (code === 'Space' || code === 'ArrowUp' || code === 'Equal') {
    return SHORTCUT_INCREMENT                                    // ← 保留：计数面（B1「锁 UI，不锁计数」）
  }
  if (code === 'ArrowDown' || code === 'Minus' || code === 'Backspace') {
    return trainingMode ? SHORTCUT_BLOCKED : SHORTCUT_DECREMENT  // ← 训练层存活 → 屏蔽（= ③「减号禁用」的键盘入口）
  }
  return null
}
```

> **为什么区分 `null` 与 `'blocked'`（不是设计洁癖，是防一个真实回归）**：`Backspace` / `ArrowDown` / `ArrowUp` 在浏览器里有默认行为（历史后退 / 页面滚动）。今天的 `onKeyDown` 对它们**一律 `preventDefault()`**。若屏蔽时直接 `return`（等价于 `null`），这三个键在训练层存活时就会**漏出浏览器默认行为** —— 训练中误按 `Backspace` 可能触发历史后退、按方向键可能滚动被锁定的页面，属于把「静默改数」换成「静默跳页」，同样是 B10 要消灭的形态。因此：**认识的键一律 `preventDefault`，`'blocked'` 只是不派发动作。**

`onKeyDown` 据此改写为**唯一派发点**（三个守卫的先后顺序是硬性的）：

```js
// CounterTool.vue 顶部新增（相对路径，禁止 @/xxx —— 见 §硬约束）
import { resolveShortcut, SHORTCUT_INCREMENT, SHORTCUT_DECREMENT, SHORTCUT_UNDO, SHORTCUT_BLOCKED } from '../../utils/counterTapGuard'

function onKeyDown(event) {
  if (!isActive.value) return                    // §C4 既有守卫（路由离开），不动
  const target = event.target
  if (target && typeof target.closest === 'function' &&
      target.closest('input, textarea, [contenteditable="true"]')) return    // 既有守卫，不动

  const action = resolveShortcut(event, { trainingMode: prefs.trainingMode })
  if (!action) return                            // 不是本模块的键 → 不吞默认行为

  event.preventDefault()                         // 本模块的键一律吞默认（含 'blocked'，见上面的理由）
  if (action === SHORTCUT_INCREMENT) increment()
  else if (action === SHORTCUT_DECREMENT) decrement()
  else if (action === SHORTCUT_UNDO) undoLast()
  // action === SHORTCUT_BLOCKED → 到此为止：不派发任何动作、不调用任何 EP API
}
```

- **硬性约束**：`onKeyDown` 内对 `undoLast()` / `decrement()` / `increment()` 的调用必须**全部经 `resolveShortcut` 的返回值派发**，**不得**存在绕过 `resolveShortcut` 的直达分支；且 `SHORTCUT_BLOCKED` **不得**出现在任何派发分支里（它只能落到函数末尾什么都不做）。这是 L2 #42c-i 的检查对象 —— 它保证 L1 #6b 的断言（`trainingMode === true` 时返回 `'blocked'`）**是负载的**：若实现另开一条直达 `undoLast()` 的路径，L1 全绿也拦不住 B10 复发。
- **不新增文件**：`resolveShortcut` 放进 `counterTapGuard.js`（定位由「只服务长按判定」扩为「破坏性输入门限」），避免为一次门限再拆一个模块。`§实施拆分` 步骤 1 的产物描述随之更新。
- **不改 §C4 的守卫**：`§C4:711` 的 `if (!isActive.value) return` 保留且仍在首行 —— 两个守卫**正交**（`isActive` 管「组件是否被当前路由激活」，`trainingMode` 管「是否处于训练层」），互不替代。
- **日间模式零变化**：`trainingMode === false` 时 `resolveShortcut` 的返回值与今天 `onKeyDown` 的行为**逐键等价**（`Space`/`ArrowUp`/`Equal` → `increment`；`ArrowDown`/`Minus`/`Backspace` → `decrement`；`Ctrl/Meta+Z` → `undoLast`）。

**⑦-b.2 顺带实证：`increment()` 路径上的目标达成 `el-dialog`（v4 新发现，必须与 ⑤ 对齐）**

同一条 `onKeyDown` 上还有一条**保留**的计数入口：`Space` / `ArrowUp` / `Equal` → `increment()` → `applyDelta()`。它**保留**（B1 硬口径：计数面不加任何门限、不吞有效敲击；键盘与触摸是同一个计数语义）。但 `applyDelta()` 越过日目标时会置 `goalReached = true`（`CounterTool.vue:1160-1165`），而 `goalReached` 绑定的正是 `el-dialog`（`CounterTool.vue:279-293`）—— 这是一条**绕过 UI 入口、直达 EP 弹层**的触发路径（触屏不可达；桌面键盘可达；`handleFigureTap()` 走同一条 `applyDelta`，那才是主路径）。

**处置：把 ⑤ 从「层内不渲染 `el-dialog`」升级为「`applyDelta` 在 `trainingMode` 时必须走层内分支」** —— 分支位置写在 `applyDelta()` 的目标达成处，横幅本身仍渲染在 `CounterTrainingOverlay.vue` 内（⑤ 已定），`applyDelta` 只负责**不置 `goalReached`** 并把「本层已达成目标」的信号交给训练层（prop / 共享 ref 均可，实现自定；本方案只约束**分支位置**这一件事）：

```js
if (dailyGoal.value > 0 && todayCount.value >= dailyGoal.value && !goalFired) {
  goalFired = true
  if (prefs.trainingMode) {
    /* ← 训练层存活：不置 goalReached，改由层内横幅 + 震动脉冲（⑤） */
  } else {
    nextTick(() => { goalReached.value = true })     // 日间模式：保持既有模态不变
  }
}
```

**为什么必须落在这一层**：若只把 `el-dialog` 从模板里挪走而 `goalReached.value = true` 照旧执行，则「训练层存活期不调用 EP 弹层」这条硬口径**仍被 `applyDelta` 绕过**（弹层被 `v-if` 干掉，等价于把 EP 弹层换成一次无反馈的状态写入）。判据 = **L2 #42c-ii**。

**⑦-b.3 训练层存活期的全部 `document` / `window` 级入口（逐条实证）**

> 枚举方式：`grep -rn "document.addEventListener\|window.addEventListener" frontend/src`（全仓库 55 条命中，去除 `.tsx`/其它 view 后与本模块相关者如下）+ 对 `CounterTool.vue`、`App.vue`、`main.js`、`GlobalPet.vue` 的逐段 Read。

| # | 入口（事件 → 处理器） | 注册点 | 训练层存活时的可达性（**实证**） | 处置 |
|---|---|---|---|---|
| 1 | `document` `keydown` → `Space` / `ArrowUp` / `Equal` → `increment()` | `CounterTool.vue:1549` / `:1449-1453` | **可达**。走完整计数路径（`applyDelta` → `registerHit` → `active.taps`）；越过日目标时置 `goalReached = true` → `el-dialog` | **保留计数**（B1 硬口径「锁 UI，不锁计数」）；**目标达成弹层必须走 ⑦-b.2 的层内分支** |
| 2 | 同上 → `ArrowDown` / `Minus` / `Backspace` → `decrement()` | `CounterTool.vue:1455-1459` | **可达**（v4 新记录）。`applyDelta(-step, false)` → 负向改 `todayCount`（`shouldFeedback=false`，不进 `taps`，也不触发目标弹层） | **训练层存活时屏蔽**（`resolveShortcut` 返回 `'blocked'`：不派发动作，但**仍 `preventDefault`**，避免 `Backspace` 触发历史后退 / 方向键滚动页面）—— ③ 已把「减号」在层内禁用，键盘是同一功能的第二个入口 |
| 3 | 同上 → `Cmd/Ctrl+Z` → `undoLast()` → `ElMessage.success` | `CounterTool.vue:1461-1464` / `:1168-1175` | **可达**（v3 写错的那条，B10） | **训练层存活时屏蔽**（⑦-b.1） |
| 4 | `document` `visibilitychange`（hidden）→ `persistAll()` | `CounterTool.vue:1550` / `:1533-1537` | 可达 | **保留**：无 EP、不改计数，只落盘 |
| 5 | `document` `visibilitychange`（visible）/ `window` `pageshow` → `session.handleForeground()`（v3 §B4 `:516-517`） | v3 新增路径 | **可达**。`away > SESSION_IDLE_MS`（30 min）时以 `endReason='idle'` 收尾 → `active = null`（§C1.4(a)）；未超阈值时续跑 | **不得弹任何 EP 弹层、不得自动关闭训练层**。`active === null` 是本方案已允许的状态（§C1.7 `:662`、§C1.8 `:684`）：此后敲击照常进 `todayCount`，只是不再记入会话。**不新增状态机**（残留口径见风险表第 13 行） |
| 6 | `window` `beforeunload` / `pagehide` → `persistAll()` | `CounterTool.vue:1551-1552` / `:1539-1541` | 可达 | **保留**：无 EP、只落盘 |
| 7 | 1s `setInterval` → `checkDayRollover()` + `refreshSpeed()` | `CounterTool.vue:1545-1548` | **可达，且后台仍在跑**（`setInterval` 不受 `document.hidden` 影响）。跨零点时清 `todayCount` / `lastActionDelta` / `tapEvents` | 已由 §C1.8 的「①取快照 → ②会话拆分 → ③归档清零」顺序 + 前台守卫覆盖（L1 #12 / #25a）。训练层**不**因此关闭 —— 跨零点时人还在练属正常路径，与 D3 `reset` 的「当天清零故关层」语义不同 |
| 8 | `autoTimer` `setInterval` → `increment()` | `CounterTool.vue:1206-1211` | **门限已定（不是假设）**：③ 强制「进层时若开着自动连点则 `stopAuto()` + 提示」；§C4 的 `onDeactivated` 再调一次 `stopAuto()` 兜底；`applyDelta` 首行的 `isActive` 守卫是第三道 | 无需额外处理。**但进层时的「已强制停止自动连点」提示必须层内自绘**（同 ⑦，不得 `ElMessage`） |
| 9 | `window` `resize` → `App.vue` 的 `checkMobile()`、`GlobalPet.vue` 的 `onResize()` | `App.vue:507` / `GlobalPet.vue:315` | 可达 | **不处理**：均无 EP 调用、不改计数。训练层为 `position: fixed; inset: 0` + `dvh`，旋转/缩放时自适应重算 |
| 10 | `window` `focus` / `pageshow(persisted)` / `visibilitychange` → `refreshCurrentViewAfterIdle()` → `viewRefreshKey += 1` → `keep-alive` 按新 key **重建** `CounterTool` | `App.vue:469-487` / `:507-510`，`App.vue:323` | **可达**（后台 ≥5 min 回前台，或 bfcache 恢复 —— 正是训练场景）。组件被销毁 ⇒ **训练层随之消失** | **接受**（硬约束「不改 `App.vue`」）。重建后 `prefs.trainingMode` 与 `active` 均已落盘 → `onMounted` 的 `loadAll()` 读回，**训练层自动恢复**；**唯一损失 = `active.taps`（不落盘，L1 #20）⇒ 当前会话的时间轴条目丢失**。§C4 覆盖「监听摘除」，本条补齐「重建 ⇒ 层消失」的后果口径（见风险表第 15 行） |
| 11 | `window` `error` / `unhandledrejection` / `app.config.errorHandler` → `showFatalError()` 建 `#__fatal_error__` | `main.js:107` / `:115` / `:125`，`z-index: 99999`（`main.js:55`） | **可达**，且它是**唯一高于训练层**（10050）的浮层 | **有意为之**：故障态兜底必须可见（否则白屏无解释），训练层**不得**为此调整 z-index。它不是 EP 弹层、不改计数；`top:0` + `max-height:50vh` 只占顶部，不影响敲击面。补记进「覆盖清单」（§现状 5 / L2 #45b 已提及 `main.js:55`） |
| 12 | **跨模块 keep-alive 残留（v4 新实证）**：其它视图注册的 `window` keydown 在离开路由后**未摘除** —— `views/life/GameHall.vue:1071`（`onGlobalKeydown`，全文件 `onDeactivated` 命中 **0**）、`views/life/PlannerTool.vue:9962`（`globalKeydownHandler`，`onDeactivated` 命中 **0**） | `App.vue:190-193` 的 `<keep-alive :exclude="[]">` + `:key="currentViewKey"` | **可达**：① `/planner` 的 `Cmd/Ctrl+K` → `openGlobalQuickAdd()`（`PlannerTool.vue:7463-7470`）打开 `el-dialog`（`PlannerTool.vue:3060-3067`）并 `nextTick` 聚焦其输入框（`:4875-4879`）→ 该弹层在训练层**下方**，且**焦点被抢进 `<input>`** ⇒ `onKeyDown` 首段守卫（`:1444-1447`）直接 return ⇒ **键盘计数静默失效**；② `/games` 的 `Enter` → `joinRoom()` / `createRoom()`（`GameHall.vue:1023-1031`，仅在 `!session.active` 时）。`Space` 已被本模块的 `preventDefault()` 挡下（`GameHall` 首行查 `event.defaultPrevented`，`:1009-1011`；且 `document` 冒泡先于 `window`），**`Enter` 无人拦** | **不在本方案范围内修**（硬约束：不改 `App.vue`、不越权改其它 view）；训练层**不为此做任何让步**（不加计数守卫、不降 z-index）。L3 新增人工观察项：**先访问 `/games` 与 `/planner`，再进 `/counter` 训练，按 `Cmd/Ctrl+K` / `Enter`** 并记录后果。**若 jaxiu 判定必须修 → 单开「keep-alive 全局监听摘除」子 issue**，本方案不扩范围（残留口径见风险表第 14 行） |

**⑦-b.4 本节的口径总结（写给实现方与 reviewer）**

1. 训练层存活期**唯一可改计数**的键是 `Space` / `ArrowUp` / `Equal`（= 计数面，**保留**）；`Cmd/Ctrl+Z` 与 `ArrowDown` / `Minus` / `Backspace` 一律 `resolveShortcut` 返回 `'blocked'`（**屏蔽动作，但仍 `preventDefault`**）。
2. 训练层存活期**不得**出现任何 EP 全局弹层调用；可达的 EP 弹层路径只有两条（⑦-b.1 的 `undoLast`、⑦-b.2 的 `goalReached`），两条都必须在**分支位置**上被拦掉，而不是靠「入口不可达」。
3. 其余入口（4 / 6 / 7 / 9 / 10 / 11 / 12）**均不改动本模块行为**，其中 10 / 12 是**已实证的残留**，口径已写明并交给 L3 观察。

**⑧【v3 新增】常驻浮层的「进层快照 / 出层恢复」不需要做**

训练层盖住 `.player-bar` / `.pet-window` 是**纯视觉覆盖**，不改变它们的状态：播放器**仍在播放**（音频继续，这在训练中是合理的——用户可自行停止），宠物**仍在动画**。

- **不做暂停播放器 / 暂停宠物**：① 会引入跨组件副作用与「出层恢复」状态机；② 音频播放与敲击音效并不冲突（敲击音效由本模块自己的 AudioContext 产生）；③ 一旦做错，「出层后播放器没恢复」是比遮挡更严重的缺陷。
- **接受**：训练中浮层不可见也不可点（被完全覆盖，不构成误触面）——这**正是 B8 想要的「锁 UI」效果**。
- **验收**：L3 **#50a** 断言训练层内「结束训练」按钮 `elementFromPoint` 命中（即未被 `.pet-launcher` / `.player-bar` 覆盖）；L3 **#50b** 断言退出训练层后浮层恢复可见且可操作、媒体播放状态不变。

---

#### D3 `resetAll()` / `clearHistory()` / `resetToday()` 与会话的交互（B7 + 未定义边界）

| 入口 | 位置 | v2 定义 |
|------|------|--------|
| **`resetAll()` 清除全部数据** | `CounterTool.vue:1510-1531` | ① 追加 `localStorage.removeItem('counter_v4_training')`；② 内存态清空 `sessions = []`、`active = null`；③ `prefs` **一并重置为默认**（因为按钮语义是「全部数据」且现有实现已经 `removeItem` 掉整个 `counter_v4_state`，把 `dailyGoal`/`step`/`volume` 一起清了，保持一致）；④ 若会话进行中 → 先按 `resetToday` 的规则结束会话（`endReason='reset'`）并关闭训练层；⑤ 确认文案改为 **「清除今日计数、全部历史记录和训练记录（含训练设置）？此操作不可恢复。」** |
| **`clearHistory()` 清空历史** | `CounterTool.vue:1498-1508` | ① **一并清空会话历史**（`sessions = []`）——会话历史与 `dailyRecords` 属同一类「历史数据」，分开清会让用户以为清干净了其实没有（与 B7 同一类隐私预期问题）；② **不动进行中的 `active`**（清历史 ≠ 结束当前训练）；③ 确认文案改为 **「清空全部历史记录（含训练会话），但保留今天的计数与进行中的训练？」** |
| **`resetToday()` 重置今日** | `CounterTool.vue:1478-1497` | ① 会话**立即结束**：`endReason='reset'`，`endedAt = now`，`count = max(0, todayCount_at_end − countAtStart)`；② `active = null`；③ 若 `prefs.trainingMode === true` → 一并置 `false` 并退出全屏层（今天已经归零，继续待在训练层没有意义）；④ 确认文案改为 **「重置今日计数将结束当前训练会话，确认？」**（现状文案「重置今日计数、节奏和撤销状态？」没说清会结束训练） |
| **会话进行中点「清除全部数据」** | — | 同 `resetToday` 的处理顺序：先结会话（`endReason='reset'`）+ 关训练层，**再**清盘。⚠️ **顺序硬约束（v3 / §D2⑦）**：必须先置 `prefs.trainingMode = false`（训练层从 DOM 移除）**再**调 `ElMessageBox.confirm`——否则确认框会被 `z-index: 10050` 的训练层盖住，用户看不到任何反馈。 |

**`endReason` 取值域（v2 收敛）**：`manual | idle | midnight | reset`（详见 C1.9）。

---

#### D4 可选加分项：训练中屏幕常亮（未决 #7，v2 从「不做」提为「可选实现」）

**理由**（评审 FINN-11 指出，PM 采纳）：真机练 30 分钟屏幕**必然**熄灭，需求 2 会持续复现；Wake Lock 从源头掐掉这个场景，比事后修音频恢复更稳。

**接入成本低则做，且必须带降级路径**：

```js
// 契约（实现按此落地，放在 useCounterSession 或独立 useWakeLock.js）
// 1) 仅在 prefs.wakeLockEnabled === true 且会话 active 时申请
// 2) navigator.wakeLock.request('screen') 必须在用户手势内（训练层「开始」按钮点击）
// 3) 失败（不支持 / 被拒绝 / 非安全上下文）→ 静默降级，不弹提示、不阻塞训练
// 4) 页面 hidden 时浏览器会自动释放，回到 visible 且会话仍 active → 重新申请
//    （重新申请**不需要**手势，这是 Wake Lock API 的既定行为）
// 5) 会话结束 / 暂停 / 组件 deactivated → release()
// 6) prefs.wakeLockEnabled 默认 true，设置面板给一个开关「训练时保持屏幕常亮（耗电）」
```

**降级路径**：`!('wakeLock' in navigator)` 或 `request()` reject → 记 `wakeLockSupported = false`，当天不再重试，功能等同不做。B2 的音频修复仍是主要保障手段——Wake Lock 只是降低触发概率，**不能替代 B2**。

---

### R2 结论：本模块不需要后端

- 计数、速度、会话、目标全部可在客户端计算与存储（`localStorage`），符合 R2「客户端能算的不要走网络」
- 无跨用户、无分享、无协作需求；现有 `counter_v4_*` 已是纯本地
- 引入后端会带来：新表 + 新路由 + 鉴权 + 迁移，收益（跨设备同步）与当前单人单机使用场景不匹配
- 若未来要跨设备同步 → **单开 backend 子 issue**（复用现有 Askit 同步通道，见 `AGENTS.md` 模块 #27），本方案不扩范围

### 未定义边界（v2 补完）

| 边界 | 定义 |
|------|------|
| **多标签页** | **明确不支持**。`counter_v4_training` 采用「单键整体写、后写覆盖」——单键写只保证单次写入原子，不保证跨标签不互相覆盖。**不引入 `storage` 事件同步**（`VoiceInboxTool.vue:633` 有先例，但本模块不引，避免复杂度）。方案与 PR 描述必须写明「不支持多标签页同时使用，后保存的标签页会覆盖另一个」；设置面板可加一行说明文案（可选）。 |
| **训练中从别处触发 `el-dialog` / 全局弹层** | 见 D2⑦：训练层 `z-index: 10050` **盖住一切含 EP 弹层**；进层时强制关闭全部 EP 浮层、层内不提供入口，且**训练层存活期间禁止调用 `ElMessage` / `ElMessageBox` / `ElNotification`**（会渲染在用户看不见的地方）。意外浮层不通过计数面守卫处理。 |
| **常驻 fixed 浮层（`.pet-window` / `.pet-launcher` / `.player-bar`）** | 见 §现状 5 + §A5 + §D2⑦⑧。**训练期**：被训练层完全覆盖（B8），不暂停、不隐藏，出层自然恢复。**日间模式**：接受为已知残留，不做收敛（理由见 §A5）；L3 按「浮层开 / 关」两种前置条件分别验收。 |
| **回前台时 idle 与 midnight 的竞争** | 见 C1.4(v3)：严格 **(a) 先按 `SESSION_IDLE_MS` 结算 idle → (b) 再跑 `checkDayRollover()` → (c) 最后消费 `pendingSplitFrom`**。顺序不可调换；`lastHiddenAtMs` / `pendingSplitFrom` 为 store 级字段，与 `active` 的生命周期解耦。 |
| **`.player-bar` 不持久化带来的口径不一致** | `useMediaPlayer.js:6` 初始 `currentIndex: -1`，且**无持久化** → `App.vue:241` 的 `v-if` 使播放器栏只在「本次会话播放过媒体」后出现。因此**同一台设备两次打开 `/counter` 的首屏结果可能不同**（占 y 615–667，会遮住 `.mobile-main-actions` 下沿）。**定义**：不接受「首屏可用」为无条件结论；L3 #48 必须带前置条件维度分别跑（播放器栏开 / 关）。这是**已知且可接受**的差异，不通过改 `App.vue` 消除。 |
| **`durationMs` 主口径下的派生统计** | 本次不做周/月训练时长统计。若未来新增，口径 = `Σ activeMs`（有效时长），不得用 `durationMs`。会话总结面板**必须同时给出**「实际时长」与「有效时长」。 |
| **`resetAll()` / `clearHistory()` 与会话** | 见 D3。 |
| **`endReason` 取值域** | 见 C1.9：`manual \| idle \| midnight \| reset`。 |
| **会话进行中 `dailyGoal` 被改** | 不影响会话：会话目标是 `active.goal` 的快照，`dailyGoal` 改动只影响日统计展示与 `goalPercent`。 |
| **同一天开多个会话** | 允许。`startDate` 相同，`count` 各自独立；不变量 `Σ sessions(startDate=D).count ≤ 当日归档 count` 仍成立（会话之间不重叠）。 |

### 实施拆分（交下游的派工建议）

| 步骤 | 负责 | 产物 |
|------|------|------|
| 1 | frontend-writer | `frontend/src/utils/counterTapGuard.js` + `.test.js`（纯函数，先写测试；**长按守卫 + 【v4 / B10】`resolveShortcut` 快捷键门限**，无计数门限） |
| 2 | frontend-writer | `frontend/src/composables/useCounterAudio.js` + `.test.js`（注入 fake ctx） |
| 3 | frontend-writer | `frontend/src/composables/useCounterSession.js` + `.test.js`（注入 now/storage） |
| 4 | frontend-writer | `frontend/src/components/CounterTrainingOverlay.vue` |
| 5 | frontend-writer | `CounterTool.vue` 接线（模板/脚本/样式）；**【v4 / B10】`onKeyDown` 按 §D2⑦-b.1 改为经 `resolveShortcut` 派发；`applyDelta` 的目标达成分支按 §D2⑦-b.2 按 `trainingMode` 分流**；**移动端样式改动严格按 §A2/§A3 的算式** |
| 6 | lint-runner | 对改动文件跑 `npx eslint` / `node --check`（按该 agent 现有流程） |
| 7 | code-reviewer | R3（模板无 `.value`）/ R4（裸下标）/ 相对路径引用 / **§测试要点 L2 的 5 条负向 grep** / **【v4 / B10】L2 #42c-i / #42c-ii（训练层存活路径判据，需人工判定并写 review 记录）** |
| 8 | api-contract | **跳过**：本次无 API 契约变更（无后端、无新增请求），该 agent 无输入 |
| 9 | devtools-qa | 按「测试要点」执行；L1/L2 可自动化，L3 需真机 |

---

## 兼容性与迁移

### 数据

| 项 | 结论 |
|----|------|
| `counter_v4_state` | **字段与语义完全不变**。新增训练相关状态一律进新键，避免旧版本 `saveState()` 逐字段枚举时把它们抹掉（`:1022-1040`） |
| `counter_v4_records` | **不改结构**。特别提示：`normalizeRecords()`（`:976-1011`）会重建记录并丢弃未知字段，任何「把会话挂到日记录上」的做法都会被静默清空——本方案不这么做 |
| `counter_v3_*` 兼容读 | 保持不变（`:1052-1054`） |
| 新键 `counter_v4_training` | 单向兼容：新代码读旧数据 → 键不存在 → `active=null, sessions=[], prefs=默认`，功能等同今天；旧代码读新键 → 从不读取 |
| **`resetAll()` 与新键** | v2 起「清除全部数据」会 `removeItem('counter_v4_training')`；旧版本用户升级后第一次点该按钮才生效，无迁移动作 |
| 回填/迁移 | **不需要**。没有老字段需要改写，没有历史数据需要重算 |
| 回滚 | 直接回退前端产物即可。会话数据成为孤儿键（不报错、占 ~40KB），可手动清；**不会**损坏 `_state`/`_records` |

### 行为（对现有用户的实际变化，需在发布说明里写清）

1. **计数提交时机不变**——仍是 `pointerdown` 一击即计（v1 曾计划改成「按下→抬起」，**v2 已撤回**）。
2. **不引入任何计数去抖**（v1 的 60ms 去抖**已撤回**）。
3. 离开 `/counter` 后不再后台计数（原来是缺陷，属修复而非破坏）。
4. 移动端断点 720 → **767.98**（与外壳 `width < 768` 严格对齐）；721–767px 的用户会看到布局变化（这正是修 bug）。
5. **移动端日间布局重排**：敲击面板占满首屏，「本周节奏」面板移到首屏之下（滚动一屏可达）。所有主操作（计数 / 加减 / 撤销 / 自动 / 功能）留在首屏。
6. 新增训练记录（`counter_v4_training`）；「清除全部数据」现在会一并清掉它。
7. `dailyGoal`、历史记录、日历、周热力的显示与统计**零变化**。
8. 会话目标**与每日目标完全解耦**：开始训练时默认「不设目标（自由训练）」或用户显式选择，不会自动带出「剩余日目标」。

---

## 风险与回滚

| # | 风险 | 影响 | 缓解 / 回滚 |
|---|------|------|------------|
| 1 | iOS 在**无用户手势**时可能拒绝 `resume()`（回前台那一刻不一定在手势里） | 极少数情况下回前台首击仍无声 | B2 已把恢复**收敛到手势入口**：下一次敲击必然携带手势 → 必然能 resume。降级阶梯第 2/3/4 层 + 「点我恢复」胶囊兜底；即便全失败也不影响计数。回滚：音频改动集中在 `useCounterAudio.js`，恢复旧内联函数即可 |
| 2 | 移动端高度算式依赖 `App.vue` 的 71px（56 header + 15 padding）。**外壳若改动会静默错位** | 首屏出现空白或溢出 | 把 71 写成 `.counter-app` 上的 `--ct-chrome-top` 单一常量，并在注释里写死来源行号（`App.vue:567-580` / `App.vue:1044-1047`）；L3 验收按「可见性」判定而非像素，外壳小改动不会直接判 FAIL。回滚：改回 `min-height: 100dvh`（即当前行为） |
| 3 | 会话/后台计时在极端情况失真（系统杀进程、时钟被改） | 单条会话时长偏差 | `Math.min` 夹紧（`endedAt` 不得早于 `startedAt`）、`durationMs` 与 `activeMs` 双口径互相校验；进程被杀时按「最后一次落盘状态 + idle 收尾」处理，最坏丢最近 5s 的会话计时（敲击本身走 `saveState` 的既有链路，不受影响） |
| 4 | `CounterTool.vue` 继续膨胀（2806 → 预计 3200+ 行） | 维护性下降 | 抽出 3 个模块 + 1 个组件（见派工表），组件内只做接线 |
| 5 | **全屏训练层与外壳 / 常驻浮层的 z-index 冲突（B8）** | 训练层被 `.player-bar`(10000) / `.pet-launcher`(9999) / `.pet-window`(9998) 盖住 → 「结束训练」不可达、只能刷新页面；浮层自带控件成为误触面 | 训练层固定 **`z-index: 10050`**（全仓库常驻产品浮层最大值 = `10000`，grep 已核实；L2 #45b 断言）；D2⑦ 已定「进层时关闭全部 EP 浮层 + 层内禁用全部 EP 全局弹层 API」；**代价**：训练层同样盖住 EP 弹层与全部常驻浮层（与「进层即锁 UI」一致），故层内不得用任何 EP 全局弹层 API（见第 10 行）；对「本次会话播放过媒体 + 宠物展开」这一最坏组合做人工回归（**L3 #50a**） |
| 6 | 断点从 720 改到 767.98 触发未知回归 | 平板竖屏（768）进入桌面布局 + 文档层滚动 | 767.98 与外壳 `isMobile` 本来就一致，改动方向是**消除**不一致；恰好 768px 的行为已在 §A1 写明（功能不损），QA 需覆盖 767 / 768 / 769 三档 |
| 7 | localStorage 配额 | 写入失败 | try/catch + 丢弃最旧 50 条重试一次 + 静默放弃；`active.taps` 不落盘（11KB 只在内存），绝不影响计数 |
| 8 | keep-alive 下 `onActivated`/`onDeactivated` 未覆盖所有路径（`currentViewKey` 变化导致重建） | 守卫失效 / 监听累积 | C4：`onMounted` 也置 `isActive=true`；`addGlobalListeners()` 幂等；`applyDelta` + `onKeyDown` 双重守卫；QA 用例覆盖「切走→切回→再切走」 |
| 9 | Wake Lock 被拒绝 / 不支持 | 屏幕仍会熄灭 | D4 的静默降级路径；B2 的音频恢复是主要保障，Wake Lock 只是降低触发概率 |
| 10 | **训练层抬到 10050 后，EP 全局弹层（`ElMessageBox` / `ElMessage`）被训练层自己盖住** | 点「结束训练」看不到确认框 → 用户以为无响应；训练中任何 `ElMessage` 反馈不可见 | D2⑦ 硬约束：层内二次确认改**层内自绘**；D3 明令「先置 `trainingMode = false` 关掉训练层，**再**弹确认框」；L2 #42b 负向 grep 断言 `CounterTrainingOverlay.vue` 无 EP 弹层调用。**【v4 / B10 补强】仅靠「层内不渲染入口」推不出「训练中不存在合法触发路径」** —— `document` / `window` 级监听不受 `v-if` 影响，必须逐个入口实证（**§D2⑦-b.3 的 12 行清单**），并把两条实际可达的 EP 路径（`undoLast`、`goalReached`）在**分支位置**上拦掉（⑦-b.1 / ⑦-b.2）；判据 L1 #6b/#6c + L2 #42c |
| 11 | **B9 撤回 `overflow: visible` 后，将来若有人给 `.ambient-layer` 加纵向大尺寸装饰** | 装饰被 `overflow: hidden` 裁掉（视觉降级，非功能问题） | 有意为之：`hidden` 是**唯一**能同时满足「光斑裁剪」与「无横向滚动条」的选择；新增装饰须自行保持在 `.counter-app` 盒内（§A2(5)） |
| 12 | **【v4 / B10】训练层存活期，全局输入入口绕过 UI 门限直达 EP 弹层 / 改计数**（B10 本身：`Cmd/Ctrl+Z` → `undoLast()` → `ElMessage` + 静默改数；同类的键盘 `decrement`；`applyDelta` → `goalReached` → `el-dialog`） | 训练中计数被静默修改、反馈不可见；训练结束后时间轴条目莫名缺失 | **结构性缓解**：把门限做成纯函数 `resolveShortcut`（`counterTapGuard.js`）并令 `onKeyDown` 成为**唯一派发点**（⑦-b.1）；`applyDelta` 的目标达成分支按 `trainingMode` 分流（⑦-b.2）；**按 §D2⑦-b.3 的 12 行清单逐个入口实证**，不留「某入口不可达」的假设。判据：**L1 #6b / #6c（自动化主判据）+ L2 #42c-i（路径核实）+ L2 #42c-ii（负向判定）**，另 L3 #81b 四条键逐个真机确认。回滚：`resolveShortcut` 独立成文件、`onKeyDown` 改动集中在 20 行内，恢复原 `onKeyDown` 即可（但会同时恢复 B10） |
| 13 | **【v4 / B10 顺带】训练层存活期，回前台 `handleForeground()` 以 `idle` 收尾会话（后台 > 30 min）后 `active === null`，而训练层仍在** | 用户继续敲击时**计数照常**（`todayCount` 不受会话影响），但这段训练**不再记入会话**（会话记录里少一段） | **有意接受，不新增状态机**：`active === null` 是 §C1.7（`:662`）与 §C1.8（`:684`）**已允许**的状态；会话是「记账」、训练层是「UI」，D3 的 `reset` 之所以关层是因为**当天计数被清零**（语义不同，不构成类比）。硬约束只有两条：该路径**不得弹任何 EP 弹层**、**不得自动关闭训练层**。不新增判据（沿用 L1 #10 / #25d 对 `handleForeground()` 的既有断言） |
| 14 | **【v4 / B10 顺带】keep-alive 下其它视图的 `window` keydown 未摘除**（`/games` `GameHall.vue:1071`、`/planner` `PlannerTool.vue:9962`，两者 `onDeactivated` 命中均为 0） | 训练中按 `Cmd/Ctrl+K` → `/planner` 的 `el-dialog` 在训练层**下方**打开并**抢焦点** → 键盘计数**静默失效**；按 `Enter` → `/games` 建房/入房 | **不在本方案范围内修**：硬约束「不改 `App.vue`」+ 不越权改其它 view；训练层**不为此做任何让步**（不加计数守卫、不降 z-index）。**取证**：L3 #81c（`/games` → `/planner` → `/counter` 前置条件下按 `Cmd+K` / `Enter`，只记录不判 FAIL）。**升级路径**：若 jaxiu 判定必须修 → 单开「keep-alive 全局监听摘除」子 issue（与 §C4 同型修复，但作用于其它 view），本方案不扩范围 |
| 15 | **【v4 / B10 顺带】后台 ≥5 min 回前台 / bfcache 恢复 → `refreshCurrentViewAfterIdle()` 递增 `viewRefreshKey` → `CounterTool` 被重建**（`App.vue:469-487 / 507-510`） | **训练层随组件销毁而消失**（随后按落盘的 `prefs.trainingMode` 自动恢复）；当前会话的**时间轴条目丢失**（`active.taps` 不落盘） | **接受**（硬约束「不改 `App.vue`」）。缓解来自既有设计：`prefs.trainingMode` 与 `active` 均已落盘 → 重建时 `loadAll()` 读回，**训练层自动恢复、会话不丢**；唯一损失是 `active.taps`（L1 #20 已明确不落盘）。§C4 已覆盖「监听摘除」这一面（风险表第 8 行），本行补齐「重建 ⇒ 层消失」的后果口径。不新增判据 |

---

## 测试要点（交 devtools-qa，可执行、可判定）

> 运行方式：`cd frontend && npx vitest run`（仓库已有 vitest，`frontend/package.json:11` → `"test": "vitest run"`；配置 `frontend/vitest.config.js`，environment=jsdom，include `src/**/*.test.js`，`@` 别名指向 `./src/neon`）。
> **不要**起 dev server（R1）。L3 真机项由人工执行并回报。

### L1 纯函数单测（必须全绿）

**`counterTapGuard.test.js`**（v2 重写：只测长按守卫；【v4 / B10】增加 `resolveShortcut` 组）

1. `isLongPressCommit({ startAt: 0, endAt: 599, thresholdMs: 600 })` → `false`
2. `isLongPressCommit({ startAt: 0, endAt: 600, thresholdMs: 600 })` → `true`（边界含等号）
3. `createLongPressGuard`：`start(1)` 后立刻 `cancel(1)` → `takeCommit(1)` 返回 `false`
4. `createLongPressGuard`：`start(1)` → 时间推进 700ms → `takeCommit(1)` 返回 `true`
5. `cancel()` 对未知 pointerId 不抛异常（防御性）
6. **负向断言（B1 闸门）**：模块**不得**导出 `shouldCommitTap` / `TAP_MOVE_TOLERANCE_PX` / `TAP_MAX_DURATION_MS` / `TAP_DEBOUNCE_MS`（`expect(mod.shouldCommitTap).toBeUndefined()` 等 4 条）
6b. **`resolveShortcut` 训练层撤销门限（【v4 / B10 新增 —— 本条是 B10 的自动化主判据】）**：
    - `resolveShortcut({ code: 'KeyZ', metaKey: true }, { trainingMode: true })` → **`'blocked'`**（等价断言：训练层存活时 `onKeyDown` **不派发** `undo` ⇒ `undoLast()` 不可达 ⇒ 其内部的 `ElMessage.success`（`CounterTool.vue:1174`）**不可能被调用**）
    - `metaKey: true` 与 `ctrlKey: true` 各断言一条（macOS / Windows 两种修饰键都必须屏蔽）
    - 同一入参 + `{ trainingMode: false }` → **`'undo'`**（日间模式零回归）
6c. **`resolveShortcut` 其余键的训练层门限（【v4 / B10 新增】）**：
    - `{ code: 'ArrowDown' }` / `{ code: 'Minus' }` / `{ code: 'Backspace' }` + `{ trainingMode: true }` → **`'blocked'`**（三条都跑）；同一入参 + `{ trainingMode: false }` → **`'decrement'`**
    - `{ code: 'Space' }` / `{ code: 'ArrowUp' }` / `{ code: 'Equal' }` + `{ trainingMode: true }` → **`'increment'`**（**必须保留** —— 断言「训练层不吞计数」，B1 硬口径；这三条是**反向闸门**，防止实现「一刀切屏蔽所有快捷键」）
    - **缺省参数**：`resolveShortcut({ code: 'KeyZ', metaKey: true })`（不传第二参）→ **`'undo'`**（`trainingMode` 默认必须是 `false`；防实现把默认值写成 `true` 导致日间模式撤销整体失效）
    - 未涉及的键一律 **`null`**（**不是 `'blocked'`** —— 这条区分很重要，见 §D2⑦-b.1 的说明）：`{ code: 'KeyA', metaKey: true }` / `{ code: 'Escape' }` / `{ code: 'Enter' }`

**`useCounterSession.test.js`**（注入 `now` / 假 storage；`vi.useFakeTimers` + `vi.setSystemTime`）

7. `start()` → `end()`：`durationMs === endedAt - startedAt`
8. 暂停 60s 后继续：`activeMs === durationMs - 60000`，`pausedMs === 60000`
9. 后台 5 分钟（模拟 `hidden` → `visible`）：`backgroundMs === 300000`，`activeMs` 不含该区间
10. 后台超过 `SESSION_IDLE_MS`（30 分钟）：会话被自动收尾，`endReason === 'idle'`，`endedAt === max(lastHitAtMs, hiddenSinceMs)`
11. **跨零点**：23:59 开始、00:00:30 检测到翻转 → 产出两条会话：第一条 `endedAt === 当日 23:59:59.999`、`endReason==='midnight'`、`startDate = 前一天`；第二条 `startedAt === 次日 00:00:00.000`、`splitFrom` 指向第一条；两条 `count` 之和 === 该时段 `todayCount` 增量
12. **跨零点顺序回归（C1.8）**：在 `todayCount` 被清零**之后**才取快照的实现会失败——断言拆分后第一条会话的 `count` **不为 0**
13. 撤销修正：会话内 `todayCount` 减 1 → `session.count` 同步减 1
14. `resetToday()` 触发：会话收尾且 `endReason === 'reset'`，`count` 不为负，`active === null`
15. **不变量**：任意操作序列后 `session.count === max(0, todayCount - countAtStart)`
16. **B6 回归：会话目标默认值** —— 构造 `dailyGoal = 300, todayCount = 80` 的状态后 `start()`：`active.goal.value === 0`（**不是 220**），且 `active.goal.value` 不等于 `dailyGoal - todayCount`
17. **B6 回归：选择预设目标** —— `start({ goalValue: 200 })` → `active.goal.value === 200`，`prefs.lastGoalValue === 200`；下一次 `start()` 不带参数时 `active.goal.value === 0`（默认仍是自由训练，不自动继承）
18. `value === 0` 时 `reached === false`（自由训练不判定达标）
19. **B5：`taps` 上限与溢出** —— 写入 600 次 → `taps.length === 500`、`tapsDropped === 100`、`tapsTotal === 600`，且保留的是**最新** 500 条（断言 `taps[0].at` 对应第 101 次）
20. **B5：`taps` 不落盘** —— `flush()` 后读 `localStorage['counter_v4_training']`，`active.taps` **不存在**，但 `active.tapsTotal` / `active.tapsDropped` 存在
21. **B5：删除单条敲击** —— 通过时间轴删一条 `delta=1` → `todayCount` 减 1，`session.count` 同步减 1
22. 归一化容错：输入 `null` / `[]`（数组而非对象）/ `{version:99}` / 含缺失字段的会话 / `sessions` 是字符串 / `active.taps` 是字符串 → 一律不抛异常，回退到合法结构
23. 容量：写入 250 条会话 → 保留最新 200 条，且按 `startedAt` 倒序
24. **多标签页口径（文档化断言）**：两个 store 实例先后 `setItem` → 最终内容 = **后写者**（不做合并、不监听 `storage`）
25. **红线回归**：会话模块的任何读写路径执行后，`localStorage` 中的 `counter_v4_records` 与 `counter_v4_state` 内容**逐字节不变**

**`useCounterSession.test.js` —— v3 新增（非阻塞 2 / 边界 1 的闸门）**

25a. **跨零点新段 `countAtStart === 0`（非阻塞 1 直接断言）**：`handleRollover(now, {autoContinue:true})` 后新段的 `countAtStart` **严格等于 `0`**（不是 `todayCount.value`、不是上一段的 `countAtStart`）；且新段 `count === 0`
25b. **`taps` 末条不变量（边界 1）**：任意 `applyDelta` 序列（含 `decrement`、含改 `step` 后继续）后，`active.taps.at(-1).delta === lastActionDelta`；`undoLast()` 弹出末条后该不变量仍成立
25c. **`undoLast()` 同步会话**：撤销后 `active.taps.length` 减 1，且 `session.count === max(0, todayCount - countAtStart)`
25d. **回前台两阶段顺序（非阻塞 2）**：构造 `lastHiddenAtMs = 23:50`、`now = 次日 07:50`（`away > SESSION_IDLE_MS`）→ 跑 `handleForeground()`：会话以 `endReason==='idle'` 收尾，**且不产生**任何 `startDate === 次日` 的新会话（`sessions` 里只有一条）；用同一组时间但 `now = 23:58`（`away ≤ SESSION_IDLE_MS`）重跑 → 会话**续跑**（`backgroundMs` 增加、`active !== null`）
25e. **`pendingSplitFrom` 补开（非阻塞 2）**：后台跑到 `checkDayRollover()`（`autoContinue=false`）→ `pendingSplitFrom` 被置为旧段 id 且 `active === null`；随后 `away ≤ SESSION_IDLE_MS` 的回前台 → **补开**新段（`startedAt === 次日 00:00:00.000`、`splitFrom === 旧段 id`、`countAtStart === 0`），且 `pendingSplitFrom` 被清空。若 `away > SESSION_IDLE_MS` → **不补开**，`active` 保持 `null`

**`useCounterAudio.test.js`**（注入 fake ctx）

26. 初始 `state==='suspended'` → `play()` 内部调用 `resume()`，且 `resume()` 的 Promise **resolve 之前不调用 `source.start()`**（核心回归：当前实现在这步就漏了）
27. `state==='interrupted'` → 仍然调用 `resume()`（核心回归：当前实现完全跳过）
28. `resume()` 挂起超过 `resumeTimeoutMs` → 不抛异常，跳过本次播放，连续 2 次后 `isDegraded === true`
29. `state==='closed'` → 新建 ctx 且 `soundBuffers` 被清空重建（断言两次 `createBuffer` 调用发生在不同 ctx 上）
30. `ctxFactory()` 抛异常 → `play()` 不抛（调用方 `applyDelta` 不中断）
31. `rebuild()` → 旧 ctx 的 `close()` 被调用，全局同时只存在 1 个 ctx
32. **B2 回归：`handleForeground()` 绝不碰 ctx** —— 调用 `handleForeground()` 后，fake ctx 的 `resume()` / `close()` 调用次数均为 **0**，`ctxFactory()` 调用次数为 0
33. **B2 回归：`play()` 不允许 `close()`** —— `state==='closed'` 时 `play()` 只新建 ctx，**不**调用旧 ctx 的 `close()`
34. **B2 回归：`unlock()` 可 `resume()`，且不阻塞调用方** —— `unlock()` 返回 Promise，但调用后**同步**即可断言 `resume()` 已被调用；`unlock()` 的 Promise 未 resolve 前组件侧 `increment()` 仍可执行（用一个同步计数器模拟）
35. **层 2 只在 `unlock()` 内执行**：`play()` 路径不产生「静音 buffer 解锁」的 `createBuffer` 调用（除非是新 ctx 的首次 `unlock()`）

### L2 组件级 / 静态检查

36. `CounterTool.vue` 的 **`<template>` 段内**不存在 `.value`（R3）、不存在裸下标 `x[0]`（R4）**【v2 更正：原写「`CounterTool.vue` 里不存在 `.value`」，按字面不可判定——script 区全是 `.value`，R3 只约束模板】**
37. 新增 import 全部为相对路径（**不得**出现 `@/xxx`，`@` 别名指向 `src/neon`）
38. `grep -n "onActivated\|onDeactivated" CounterTool.vue` 有命中（keep-alive 守卫已加）
39. `grep -rn "backend\|fetch(\|axios" ` 新增文件无命中（确认 0 网络请求）
40. **负向 grep（B1）**：`grep -rn "shouldCommitTap\|TAP_MOVE_TOLERANCE_PX\|TAP_DEBOUNCE_MS\|TAP_MAX_DURATION_MS" frontend/src` → **无命中**
41. **负向 grep（B1）**：`grep -n "isPrimary" frontend/src/views/life/CounterTool.vue frontend/src/utils/counterTapGuard.js frontend/src/components/CounterTrainingOverlay.vue` → **无命中**（计数面不得有任何 pointer 过滤）
42. **负向 grep（B2，【v3 / 非阻塞 6 改写】）**：`grep -n` 的输出**无法证明**命中「只在 `unlock()` / `play()` 函数体内」。改为**三步人工判定**，判定人是 code-reviewer，结论要写进 review 记录：
    ```bash
    # ① 取函数边界（每个函数的起止行）
    grep -n "^\(export \)\?\(async \)\?function \|^  \(async \)\?function " frontend/src/composables/useCounterAudio.js
    # ② 取命中行
    grep -n "\.resume()\|\.close()\|new AudioContext\|new (window.AudioContext" frontend/src/composables/useCounterAudio.js
    # ③ 人工比对：② 的每一条行号，必须落在 ① 里 unlock() 或 play() 的行区间 [函数起, 下一个函数起−1] 内
    ```
    **判定标准**：② 的命中集合全部落在 `unlock()` ∪ `play()` 的行区间内 → PASS；任意一条落在 `handleForeground()` / `pageshow` 相关路径 / 模块顶层 → FAIL。
    > **自动化主判据是 L1 #32 / #33 / #34**（fake ctx 断言 `handleForeground()` 对 `resume()`/`close()`/`ctxFactory()` 的调用次数为 0）。L2 #42 只作辅助，不要把它写成一条 `grep ... && echo PASS` 的假自动化。
42b. **负向 grep（B8 / D2⑦，v3 新增）**：`grep -n "ElMessage\|ElMessageBox\|ElNotification\|el-dialog\|el-drawer" frontend/src/components/CounterTrainingOverlay.vue` → **无命中**（训练层存活期间不得调用任何 EP 全局弹层 API；「结束训练」的二次确认必须层内自绘）
42c. **训练层存活路径的负向判据（【v4 / B10 新增 —— 两部分都要过】）**：
    - **(i) 派发路径核实（保证 L1 #6b 是负载的）**：取出 `onKeyDown` 的函数行区间（`grep -n "^function " frontend/src/views/life/CounterTool.vue` 定位首尾），在该区间内核对三条：
      1. `resolveShortcut` 命中**恰 1 处**；
      2. 承接其返回值的变量（`action`）**赋值恰 1 处**，且唯一来源就是这次 `resolveShortcut(...)` 调用（不得有第二条赋值路径）；
      3. 在 `event.preventDefault()` 与三个派发调用（`undoLast(` / `decrement(` / `increment(`）**之前**，存在一处 `if (!action) return` 形式的早退；
      4. `SHORTCUT_BLOCKED` **不得**出现在任何派发分支的条件里（`grep -n "SHORTCUT_BLOCKED" ` 在 `onKeyDown` 区间内只允许出现 ≤1 次，且只允许在末尾注释中 —— 一旦它进了 `if/else if`，就意味着屏蔽分支被接上了某个动作）。
      判定标准：四条缺一 → **FAIL**（缺任一条都意味着门限可能被绕过，此时 L1 全绿也拦不住 B10 复发）。这条不是形式主义：B10 的成因就是「门限写在 A 处、实际入口在 B 处」。
    - **(ii) 训练层存活路径的 EP 弹层 API 负向判据（三步人工判定，判定人 code-reviewer；写法同 #42，不做假自动化）**：
      ```bash
      # ① 取函数边界
      grep -n "^function " frontend/src/views/life/CounterTool.vue
      # ② 取 EP 命中（模板段与脚本段都算）
      grep -n "ElMessage\|ElMessageBox\|ElNotification\|el-dialog\|el-drawer" frontend/src/views/life/CounterTool.vue
      # ③ 人工比对：对 ② 的每条命中，判定「训练层存活时是否有前置门限拦住」
      ```
      **判定标准**：② 的每条命中必须落进下列**已有门限**之一，否则 **FAIL** ——
      - `:1174`（`undoLast()` 内的 `ElMessage.success`）→ 由 `resolveShortcut` 在 `trainingMode === true` 时返回 `'blocked'` 拦住（门限**在路径上**，由 (i) 保证）
      - `:279-293`（`el-dialog`，`goalReached`）→ 由 ⑦-b.2 的 `applyDelta` 分支拦住（`trainingMode` 时不置 `goalReached`）
      - `:1479 / :1494 / :1499 / :1506 / :1511 / :1529`（`resetToday` / `clearHistory` / `resetAll`）→ 调用点只有 `:370-372` 三处，全部在 `v-if="showSettings"` 面板内；进层时 `showSettings = false` 且层内不渲染这些入口（③）；D3 的「训练中清除全部数据」路径**先**置 `prefs.trainingMode = false` **再**弹确认框
      判定完成后，把「命中行 → 门限 → PASS/FAIL」表写进 review 记录。
43. **负向 grep（B6）**：`grep -n "dailyGoal" frontend/src/composables/useCounterSession.js` → **无命中**（会话模块不得读日目标）
44. **负向 grep（B5）**：`grep -n "tapEvents" frontend/src/composables/useCounterSession.js` → **无命中**；`grep -n "tapEvents" frontend/src/views/life/CounterTool.vue` 的命中只允许在 `refreshSpeed` / `registerHit` / `checkDayRollover` 的既有逻辑里，**不得出现在会话快照代码中**
45. **负向 grep（D2 挂载点约束）**：`grep -n "transform\|will-change\|perspective\|contain:" frontend/src/App.vue` 中 `.app-container` / `.main-content` / `.mobile-main` 选择器下**无命中**（否则 `position: fixed` 训练层会被裁剪）
45b. **B8 层级断言（v3 新增，三条都要过）**：
    - **正向（B8 主断言）**：`grep -n "z-index" frontend/src/components/CounterTrainingOverlay.vue` → 训练层根元素取值**必须 `> 10000`**（本方案定值 **`10050`**）；**不得**再出现 `900`
    - **正向（覆盖清单）**：训练层取值必须 **大于** §现状 5 表中每一个常驻浮层的取值 —— `.pet-window`(9998) / `.pet-launcher`(9999) / `.cheer-toast`(10000) / `.player-bar`(10000)。**逐项写出比较结果，不得只写一句「已高于」**
    - **负向（防回归）**：`grep -rn "z-index: *[0-9]\{4,\}" frontend/src` → 除 `main.js:55`（开发态报错条 `99999`，非产品层级）外，当前最大常驻值为 `10000`。**若将来新增取值 ≥ `10050` 的常驻浮层，本条断言必须重跑并相应调整训练层取值**
46. **移动端算式自检**：`grep -n "100dvh\|100vh" frontend/src/views/life/CounterTool.vue` → 移动断点（`max-width: 767.98px`）区块内**不得**出现未减去 `--ct-chrome-top` 的裸 `100dvh`
46b. **B9 负向断言（v3 新增）**：
    - `grep -n "overflow" frontend/src/views/life/CounterTool.vue` → `max-width: 767.98px` 区块内**不得**出现任何 `overflow` 声明（既不声明 `visible` 也不声明 `auto`；继承基类 `hidden`）
    - `grep -n "overscroll-behavior" frontend/src/views/life/CounterTool.vue` → **无命中**（原 `overscroll-behavior-y: contain` 必须删除）
47. `grep -n "counter_v4_training" frontend/src/views/life/CounterTool.vue` → `resetAll()` 内必须有一处 `removeItem`

### L3 真机 / 手工（需人工执行并回报结论）

> 设备矩阵：iOS Safari（含刘海机型）、Android Chrome；视口 320×568 / 375×667 / 390×844 / 414×896，横屏 844×390。

#### 布局（需求 1 核心 —— B4 闭环；【v3 重写判据 + 新增前置条件矩阵】）

> **前置条件矩阵（未定义边界 2 / 3 —— 必须逐格跑，不得只跑最干净的一格）**：48 / 48a / 48b / 49 四条要在下面 **4 个组合**上各跑一遍，逐格记录结果：
>
> | 格 | 宠物窗口 | 播放器栏 | 说明 |
> |----|---------|---------|------|
> | **A** | 展开 | 无 | **首次访问者的默认视图**（`pet-widget.global` 未设 → `GlobalPet.vue:121` 默认 `visible = true`；`useMediaPlayer.js:6` 无持久化 → 无播放器栏） |
> | **B** | 展开 | 有 | **最坏组合**（B8 的主验收格） |
> | **C** | 已收起（`.pet-launcher` 在右下角） | 无 | |
> | **D** | 已收起 | 有 | |
>
> 造「播放器栏在」的方式：先在任意页面播放一段媒体（`playerState.currentIndex >= 0`），**不刷新页面**直接导航到 `/counter`。造「宠物已收起」：点宠物窗口的收起按钮（写入 `pet-widget.global = '0'`）后刷新。

48. **判据（【v3 改写】48 / 48a / 48b / 49 / 50a / 50b 全部共用这一条，不再用 `bottom ≤ innerHeight`）**：把文档滚到顶部（`scrollTop === 0`）后，对目标元素 `el` 执行
    ```js
    const r = el.getBoundingClientRect()
    if (r.width === 0 || r.height === 0) return FAIL          // 不可见即零尺寸
    const cx = Math.min(Math.max(r.left + r.width / 2, 1), innerWidth - 1)
    const cy = Math.min(Math.max(r.top + r.height / 2, 1), innerHeight - 1)
    const hit = document.elementFromPoint(cx, cy)
    return (hit === el || el.contains(hit)) ? PASS : FAIL      // 命中测试
    ```
    同时断言 `document.documentElement.scrollWidth <= window.innerWidth`（**无横向滚动条**；这条与 B9 直接相关）。
    - **为什么必须换判据**：`getBoundingClientRect().bottom ≤ window.innerHeight` **对 `position: fixed` 覆盖层完全不敏感** —— 元素几何上在视口内，实际却被 `.player-bar` / `.pet-window` 压着点不到，会产生「L3 通过、用户仍点不到」的假通过。「可见」与「可点」不是一回事。
    - 元素中心点若因滚动/尺寸落在视口之外 → 直接 **FAIL**（在首屏外）。
    - 命中结果是被浮层挡住（`hit` 落在 `.player-bar` / `.pet-window` / `.pet-launcher` 的子树内）→ **如实记 FAIL**。**不允许用「浮层是用户自己开的」来掩盖判据**（§A5 的立场是「接受为已知残留」，不是「判它通过」）。
48a. **日间模式首屏可见性**：用 48 的判据断言 `.figure-button` 与两个 `.side-round`（`.side-round` 与 `.side-round.side-plus` 各一次）。
48b. **375×667 首屏完整性**：用 48 的判据断言 `今日次数`（`.mobile-focus-count strong`）、主敲击按钮、加减号、**`.mobile-main-actions` 的三个按钮**（撤销 / 自动 / 功能）全部命中。
49. **320×568 首屏完整性**：用 48 的判据断言 `今日次数`、主敲击按钮、加减号全部命中；步长条/焦点条允许被挤到首屏外（按 §A3 的降级备案处理并回报实际结果）。
50. **训练层首屏**：进入训练后，计数 + 敲击面 + 「结束训练」按钮**无需滚动**即可全部看到；刘海/home indicator 不遮挡。
50a. **【v3 / B8 主验收】训练层不被常驻浮层盖住**：在**前置条件格 B（最坏组合：宠物展开 + 播放器栏在）**下进入训练层：
    - 用 48 的判据断言 **「结束训练」按钮命中**（它现在是层内唯一退出路径）；
    - 断言训练层矩形与 `.player-bar` 矩形的重叠区域内、以及与非空 `.pet-window` 矩形的重叠区域内，`elementFromPoint` 命中的是**训练层元素**（即浮层被完全盖住、其控件不可点）；
    - 任一不满足 → **B8 FAIL**。
50b. **【v3 / B8 回归】退出训练层后浮层恢复**：退出训练层 → `.player-bar` / `.pet-window` 重新可见且其按钮可点（48 的判据）；媒体仍在正常播放（§D2⑧：**不**暂停播放器/宠物，退出层后状态与进入前一致）。
51. **断点切换**：767 / 768 / 769 三档宽度切换窗口——767↔769 的布局切换点与外壳 header 出现/消失**同一时刻**；恰好 768px 时按 §A1 的预期行为（桌面布局 + 文档层滚动，功能不损）并回报实际观感。
52. **横屏 844×390**：内容不重叠，安全区左右有留白，主按钮 + 加减号 + 结束训练可在滚动一屏内到达（同样建议在格 B 下跑一遍）。

#### 音频（需求 2 核心）

54. 训练中锁屏 30s → 解锁 → **第一次敲击就有声**
55. 切到别的 App 5 分钟后回来 → 第一次敲击有声
56. 锁屏 5 分钟回来 → 第一次敲击有声（或 1 次无声后自动恢复，且出现「点我恢复」胶囊）
57. **回前台不裸调 resume 的可见效果**：锁屏 1 分钟后回来，**在敲击之前**观察——不应有任何异常音/爆音；第一次敲击时 `resume()` 发生在该次手势内

#### 计数准确性（需求 4 核心 —— B1 闭环）

58. **按下即计**：在敲击面上做 10 次「按下 → 抬起」，其中 5 次按下后**滑动 30px 再抬起** → 计数 **+10**（按下即计、滑动不回滚）
59. **无时长门限**：按住敲击面 **3 秒**再抬起 → 计数 **+1**
60. **无 primary 门限**：掌缘先落在敲击面（第一指不抬起），随后用食指连续敲 5 次 → 计数 **+5**
61. **多指口径**：双指**同时**按下敲击面 → **+2**（= 今天的行为；防误触不得吞掉有效敲击）。若产品要求只 +1，需 jaxiu 另裁，本方案默认 +2
62. **`pointercancel` 不回滚**：敲击后立即触发系统级 `pointercancel`（模拟来电/边缘手势）→ 已提交的计数**不回滚**
63. 60 秒内快速敲击 200 次（约 3.3 次/秒）→ 计数**恰好 200**（不得有任何去抖吞并）
64. 训练中长按「结束训练」<700ms → 不结束；≥700ms → 弹出二次确认
65. 训练中确认：减号/步长/自动/形象/主题/日历/历史/设置入口**均不可达**
66. 目标达成时不再弹模态 → 敲击不中断
67. 在别的工具页按空格 → 计数器**不再**增长（keep-alive 修复验证）
68. 训练中开着自动连点 → 被强制关闭且有提示

#### 会话（需求 3 核心）

69. 开始训练（默认**自由训练**，目标显示「不设目标」）→ 结束 → 历史里能看到该会话的**开始时间 / 结束时间 / 实际时长 / 有效时长 / 次数 / 是否达标**
70. 选预设目标 100 → 完成后 `reached === true`；`每日目标` 的数值与进度条**不受影响**
71. **B6 真机回归**：设 `每日目标 = 300`、`今日已敲 80` → 开始训练时目标默认是 **不设目标 / 预设选项**，**不是 220**
72. 会话跨零点（可改系统时间模拟）：历史里出现两段，日归属正确，且两段计数之和不丢数
73. 训练中切到别的工具页再切回 → 计时正确（不含离开时段，`backgroundMs` 正确），会话未丢失
74. **B5 时间轴**：单次训练敲击 >500 次 → 结束后总结面板列出最近 500 条并显示「更早的 N 次未列出」；<500 次时全部列出
75. 时间轴删一条 → 今日计数同步 −1

#### 数据清理（B7 闭环）

76. 点「**清除全部数据**」→ `localStorage.getItem('counter_v4_training') === null`，且 `counter_v4_state` / `counter_v4_records` 也为 null
77. 点「**清空历史**」→ 会话历史为空、`dailyRecords` 为空，但进行中的会话**继续**（`active !== null`）
78. 训练中点「**重置今日**」→ 会话立即结束（`endReason === 'reset'`）、训练层退出、今日计数归零
79. 训练中点「**清除全部数据**」→ 先结会话再清盘，训练层退出，三个键均为 null

#### 回归

80. `todayCount` / 累计总数 / 连续达标 / 本周节奏 / 日历达标标记 / 历史记录的数值与改动前**完全一致**（同样的敲击序列，同样的结果）
81. 多标签页：两个标签页同开 `/counter`，各自敲击后刷新 → 后写的标签页数据生效（符合「不支持多标签」的既定口径，不作为缺陷）
81b. **【v4 / B10 新增：训练层存活期的全局快捷键，桌面端必跑】** 进入全屏训练层后依次按：
    - `Cmd/Ctrl+Z` → **计数不变、时间轴条目不变、屏幕上无任何 EP 弹层**（屏蔽生效；`ElMessage` 不得被调用 —— 即便被调用也看不见，所以判据取「计数与 `active.taps` 长度不变」这一**可观察量**，不取「有没有看到提示」）
    - `ArrowDown` / `Minus` / `Backspace` → 同上（计数不变），**且页面不滚动、不触发浏览器历史后退**（`'blocked'` 仍 `preventDefault`；这条是 v4 的第二个断言点，防止实现把「屏蔽」写成裸 `return`）
    - `Space` / `ArrowUp` / `Equal` → **计数 +step**（**必须仍然生效** —— 反向判据：训练层不得吞掉计数面）。在**未达日目标**时重复敲到跨越日目标 → 断言出现的是**层内横幅**（⑤），而不是被盖住的 `el-dialog`；退出训练层后日间模式的模态行为不变
    - 退出训练层 → 重复以上四条 → 与改动前的既有行为**逐条一致**（日间模式零回归）
81c. **【v4 / B10 新增：前置条件观察项（残留，非阻塞）】** 按顺序访问 `/games` → `/planner` → `/counter` 并进入训练层，然后按 `Cmd/Ctrl+K` 与 `Enter`，**记录**（不作为本方案的验收判据，只作残留取证）：
    - `Cmd/Ctrl+K` 是否打开了 `el-dialog`（应被训练层盖住）、**焦点是否被抢进其输入框**、此后 `Space` 是否还能计数；
    - `Enter` 是否触发了 `/games` 的房间创建 / 加入；
    - 观察结论写进 QA 报告，供 jaxiu 决定是否单开「keep-alive 全局监听摘除」子 issue（见风险表第 14 行）。

---

## 未决问题（需 PM / jaxiu 裁决，不要由实现方拍板）

> 标注「阻塞」的事项若在动工前无答复，实现方按**给出的默认值**落地并在 PR 里注明，不挂起流水线。

1. **会话是否自动开始？**（**阻塞**）
   (a) 手动点「开始训练」进入全屏训练层【推荐】；(b) 当天首次敲击自动开始、N 分钟无操作自动结束；(c) 两者兼有（默认关）。
   默认：**(a)**。

2. **会话目标的默认口径？**（**阻塞 —— 本版已按 jaxiu 裁决定稿**）
   ✅ **已裁决（jaxiu 2026-09-16 03:36）**：按**每次运动**设目标。因此默认 = **`0`（自由训练）**，会话开始面板提供 `100 / 200 / 500 / 自定义 / 上次目标`。
   ❌ **已作废**：`max(0, dailyGoal - todayCount)`（剩余日目标）。PM 已在 FINN-16 明示撤回该建议。
   `dailyGoal` 只在会话开始面板作**只读参考展示**，不得作为会话目标的默认值或计算输入。若 jaxiu 后续明确改口，再按其答复调整。

3. **时长主口径？**（**阻塞**）
   默认：**主展示实际时长（墙钟，直接对应「实际开始/结束时间」的字面要求），有效时长作次要指标同时展示**。两个数都会存、都会展示。

4. **静默多久自动结束会话？** 默认 **30 分钟**（`SESSION_IDLE_MS`，常量可调）。是否接受「回来后自动续跑，超过阈值才收尾」这一规则？

5. **是否需要「组」的概念？** 需求只提到「中途暂停」，本方案只做 暂停/继续（`pausedMs`）。若需要「第 N 组 / 组间休息」，数据模型要再扩一层（`sets: []`）——**这会显著加大改动量**，请明确是否需要。

6. **训练中是否隐藏日统计（今日次数 / 周热力）？** 移动端建议隐藏（空间回收 + 减少干扰），桌面端保留。默认：**移动端隐藏**。

7. **是否需要屏幕常亮（Screen Wake Lock API）？**（**v2 从「不做」提为「可选加分项」**）
   真机练 30 分钟屏幕必然熄灭，需求 2 会持续复现；接入成本低（API + 开关，失败静默降级，见 §D4）。**默认：训练时开启（`prefs.wakeLockEnabled = true`），设置面板可关**，耗电为代价。若 jaxiu 明确要关，改成默认 `false` 即可（降级路径不变）。

8. **结束后能否回溯修改单次敲击？**（**阻塞**）
   建议提供「敲击时间轴 + 删除某一条」，但删除会**同时扣减 `todayCount`**（保证 §C1 不变量）。是否接受「修改会话 = 修改当天日统计」？
   ⚠️ **v2 已把落地方式改为 `active.taps`（会话级有上限日志），不再依赖 15s 窗口的 `tapEvents`**。若定为「允许」，实现需在 `end()` 时把 `taps` 快照写进 `sessions[]`（见 C1.7）。

9. **会话历史保留量**：默认 200 条 / 不按天裁剪。是否需要导出（CSV/JSON）？

10. **是否需要跨设备同步？**（若「是」→ **必须另开 backend 子 issue**，本方案不含）
    我的建议：**否**。理由见「R2 结论」。

11. **训练层是全屏覆盖（隐藏外壳 header）还是保留页头？**
    全屏覆盖（本方案）会让移动端失去 56px 的菜单按钮，训练结束后自动退出层、恢复页头。
    备选：给 `/counter` 路由加 `hideSidebar: true`（`App.vue:364` 已支持），但那样**桌面端也会失去侧边栏**且需要自建返回入口——**不推荐**。请确认接受「训练期间全屏、其余时间保留页头」。

12. **目标达成反馈**：训练内改横幅（不阻塞敲击）+ 震动，日间模式保持模态。是否接受两种模式行为不一致？

13. **多指同时敲击（B1 的取舍）**：本版按「一击即计」硬口径，双指同时按下 = **+2**。是否接受？若要求「双指 → 只 +1」，与「不得吞掉有效敲击」冲突，需要 jaxiu 明确豁免。

---

## 硬约束（下游无条件遵守）

- **R1**：只写 + review。不跑 `go run` / dev server / `docker compose up` / `./deploy.sh` / `npm run dev`
- **R2**：0 后端改动、0 网络请求（客户端能算的一律不联网）
- **R3**：Vue 3 **模板**里永远不写 `.value`
- **R4**：模板裸下标 `obj.x[0]` 一律写 `obj.x?.[0] || ''`（尤其适用于新增的会话/时间轴数据：`session.taps?.[0] || null`、`session.goal?.value || 0`）
- **import 路径**：一律相对路径（`@` 别名指向 `src/neon`，不是 `src`）；`views/life/*.vue` → 组件用 `'../../components/X.vue'`，composable 用 `'../../composables/X'`，工具函数用 `'../../utils/X'`（与 `CounterTool.vue:455` 现有写法一致）
- **R5**：本模块无 SQLite / `:memory:` 测试，不适用（但**不得**为此新增后端）
- **R6**：不适用（无分享资产、无 blob URL 上传）
- **R8**：本模块无长耗时 AI 调用；`resume()` 的 250ms 超时上限是同类「不能让 UI 无限等」思路
- **R9**：不涉及 MCP 协议版本
- **R10**：不涉及 Planner（**严禁**碰 `PLANNER_PHILOSOPHY.md` 相关模块）
- **R11**：不涉及后端产物编译
- **文件系统**：`/Volumes/M20` 是 SMB 网络挂载——不做任何构建产物落盘，临时文件一律 `/tmp`
- **写入边界**：本方案只落在 `docs/plans/`；`AGENTS.md` 的模块地图**不需要**改（计数器是纯前端视图，本就不在该表中）。若 module-architect 认为应补一行，由其自行决定
- **不改 `App.vue`**：外壳属共享代码，本方案所有修复都在 `views/life/CounterTool.vue` + 新增模块内完成
- **🆕 一击即计（B1 硬口径）**：计数面只能是 `pointerdown` 直接提交；**不得**对计数面施加位移 / 时长 / `isPrimary` / 去抖 / 二次确认 / 长按等任何门槛；主按钮必须瞬时响应
- **🆕 音频恢复收敛（B2 硬口径）**：`ctx.resume()` / `ctx.close()` / `new AudioContext()` 只允许出现在 `unlock()` 与 `play()` 内；`visibilitychange → visible` 与 `pageshow` 只置 `needsUnlock` 标志
- **🆕 会话目标独立（B6 硬口径）**：`dailyGoal` 不得出现在任何 `goal.value` 的赋值路径上，也不得作为默认值
- **🆕 训练层层级与 EP 弹层（B8 硬口径）**：训练层根元素 `z-index` 固定 **`10050`**（不得低于 `10000`，也不得回退到 `900`）；**训练层存活期间（`prefs.trainingMode === true`）组件内不得调用任何 Element Plus 全局弹层 API**——`ElMessage` / `ElMessageBox` / `ElNotification` / `el-dialog` / `el-drawer`；「结束训练」的二次确认必须由 `CounterTrainingOverlay.vue` **层内自绘**；任何「先关训练层再弹确认框」的路径必须**先**置 `prefs.trainingMode = false`
- **🆕 训练层存活期的全局快捷键（B10 硬口径 —— v4 新增）**：`onKeyDown` 必须是**唯一派发点** —— 三个动作只能经 `counterTapGuard.js` 的纯函数 `resolveShortcut(event, { trainingMode: prefs.trainingMode })` 的返回值触达，**不得**存在绕过它的直达分支；`prefs.trainingMode === true` 时 **`Cmd/Ctrl+Z` 与 `ArrowDown` / `Minus` / `Backspace` 必须返回 `SHORTCUT_BLOCKED`（不派发动作，但**仍要 `preventDefault`**，不得漏出浏览器默认行为）**，而 **`Space` / `ArrowUp` / `Equal` 必须仍返回 `SHORTCUT_INCREMENT`**（计数面不得被训练层吞掉）；`resolveShortcut` 的 `trainingMode` **默认值必须是 `false`**。同口径：`applyDelta()` 的目标达成分支在 `trainingMode === true` 时**不得置 `goalReached = true`**（改走 ⑤ 的层内横幅 —— 该分支位置是硬性的，只改模板不叫修好）
- **🆕 移动端 `overflow`（B9 硬口径）**：移动断点内**不得**为 `.counter-app` 声明 `overflow`（继承基类 `hidden`）；`overscroll-behavior` 在本文件内**不得出现**
- **🆕 `active.taps` 写入时机（边界 1 硬口径）**：只在 `applyDelta()` 的 `actualDelta !== 0` 分支写入，**不得**在 `registerHit()` 写入；`undoLast()` 必须同步弹出末条，保证不变量 `active.taps.at(-1).delta === lastActionDelta` 恒成立
