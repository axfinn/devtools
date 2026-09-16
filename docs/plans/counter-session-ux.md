# 敲击计数器：移动端适配 / 息屏音频恢复 / 运动会话与目标 / 防误触 — 技术方案（v2 修订版）

- 原 issue：FINN-10（父 FINN-9）；本修订：FINN-16（Stage 2 评审 FINN-11 判 FAIL，7 条阻塞项）
- 模块：`frontend/src/views/life/CounterTool.vue`（纯前端，无后端）
- 状态：设计稿，**不含任何代码改动**；实现由 frontend-writer 执行，验收由 devtools-qa 执行
- 勘察方式：全部结论来自本仓库实际 grep / 逐段 Read；**未起 dev server、未跑 go build、未在真机验证**。凡属推算的数值均在文中显式标注「推算」

---

## v2 修订记录（对照 FINN-16 要求）

| 项 | 改了什么 | 主要落点 |
|----|---------|---------|
| **B1** | 删除计数面的「按下→抬起提交」与 12px / 1200ms / `isPrimary` / 60ms 去抖四道门限；改为 **`pointerdown` 一击即计**。`shouldCommitTap` 整套常量删除，防误触只剩「结构上无可误触」+「破坏性操作长按」。补 `pointercancel` 口径 | §D2①、§测试要点 L1/L3 |
| **B2** | `visibilitychange → visible` 与 `pageshow` **只置 `needsUnlock=true` 并记 `hiddenAt`**，不调 `resume()` / `close()` / 不新建 ctx。恢复只允许在 `unlock()` 与 `play()` 内 | §B4 |
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

---

## 设计

### A. 移动端适配

#### A1 断点统一（与 `isMobile` 严格对齐）

- 计数器移动断点从 **`max-width: 720px` → `max-width: 767.98px`**。**注意不是 768**：外壳判定是 **`width < 768`**（`App.vue:464`），而 CSS `max-width: 768px` 含 768 —— 用 768 会在**恰好 768px** 这一档出现「CSS 按移动端渲染、JS 不套 `.mobile-main`」的新裂缝（无固定 header、无 71px 顶部内边距）。`767.98px` 与 `width < 768` 在 1/64px 精度下等价。
- 保留 `max-width: 480px`（原 420 微调位，语义不变，仅调数值以对齐通用的 480 断点）与新增横屏断点。
- 最终断点表：

| 断点 | 用途 |
|------|------|
| `max-width: 1024px` | 双列 → 单列过渡（保留原 980 语义，仅调数值） |
| `max-width: 767.98px` | 移动布局（**与外壳 `width < 768` 对齐**） |
| `max-width: 480px` | 小屏字号/内边距微调（原 420） |
| `orientation: landscape and (max-height: 480px)` | 手机横屏专用（左计数、右敲击区），**不嵌套在移动断点内**，见 A6 |

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
  overflow: hidden;     /* 保持：只为裁剪 .ambient-layer 光斑 */
}

/* 移动（≤767.98）—— 全程不出现百分比 */
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

    /* 删除原 overflow: auto + overscroll-behavior-y: contain —— 见（4） */
    overflow: visible;
  }

  /* ≤420 的 padding-left/right 覆盖（CounterTool.vue:2752-2755）必须同步改成
     max(8px, env(safe-area-inset-left/right))，否则横向安全区失效 */
}
```

`box-sizing: border-box` 由 `frontend/src/styles/index.css:21-25` 的 `*` 规则保证，所以上面的 `calc` 高度**含** padding。

**（4）移动端滚动容器是文档层 —— `overscroll-behavior` 与 `.app-container{overflow:visible}` 的交互（B3 第二问）**

- ≤768 下 `.main-content` 既不是滚动容器（`overflow-y: visible`）也不是定高容器，`.app-container` 又是 `overflow: visible` → **实际滚动容器是文档层（`html`/`body`），不是 `.main-content`**。所以原方案「全页只保留外层 `.main-content` 一个滚动容器」这句话在 ≤768 **是反的**，本版删除。
- `.app-container { overflow: visible }` 在移动端**对滚动没有任何影响**：它既不裁剪也不滚动，唯一作用是让高度不足时内容能溢出到文档流里被文档层接住。
- 因此 `overscroll-behavior-y: contain` 写在 `.counter-app` 上只在它**自己可滚**时才生效。本版把 `.counter-app` 的 `overflow: auto` 一起删掉 → 它不再是滚动容器 → **内层阻断滚动链的问题（§现状 根因 3）从根上消失**；文档层的 `overscroll-behavior` 不由本模块控制（`html`/`body` 无此声明），保持全站一致的下拉刷新行为即可。
- 本版不再刻意保留任何内层滚动容器：目标视口下首屏放得下（§A3），放不下时**由文档层滚动兜底**，内容永远可达。

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

**为什么这样够用（推算，非真机实测；验收按「可见性」判定，不按像素）**

| 视口 | `.tap-panel` 首屏高度 | 面板内固定开销 | **敲击面高度** | 主按钮可见宽度 | 结论 |
|------|---------------------|--------------|--------------|--------------|------|
| 375×667 | 667 − 71 − 22 = **574** | 22（面板 padding）+ 74（焦点条）+ 54（步长条）+ 10 + 74（操作行） | **≈340** | ≈309 | 全部首屏可见 ✅ |
| 320×568 | 568 − 71 − 22 = **475** | 同上 | **≈241** | ≈254 | 全部首屏可见 ✅ |
| 390×844 | 844 − 71 − 22 = **751** | 同上 | ≈517 | ≈324 | ✅ |
| 414×896 | 896 − 71 − 22 = **803** | 同上 | ≈569 | ≈348 | ✅ |

- 面板底边位置：`71 + 8 + 面板高度` ≤ `100dvh`（375×667 下为 `79 + 574 = 653 ≤ 667`，余 14px 呼吸位 ✅）。加减号是 `.tap-stage` 内的绝对定位元素（`bottom: 12/14px`），天然落在这个盒子内 → 必在首屏。
- 320×568 也**不需要**触发降级（241px > 200px 下限）。若真机实测因字体/系统缩放导致不足，优先按以下顺序降级（每项都已写明代价）：
  1. `≤360px` 时 `display: none` 掉 `.step-strip`（−54px，代价：改步长需先进设置面板）；
  2. `≤360px` 时把 `.mobile-focus-count strong` 从 `42px` 降到 `34px`（−8px）；
  3. `≤360px` 时把 `.mobile-focus-meta` 的两枚胶囊合并成一行（−30px）。
  **这三条是应急手段，不要预先把它们打开**——先按上表实现，QA 真机判定。
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

#### A6 横屏（次要目标，允许降级）

- 新增 `@media (orientation: landscape) and (max-height: 480px)`，**放在文件末尾且不嵌套在移动断点内**——因为 844×390 这类机型宽度 > 768（走桌面分支），而 667×390 这类机型宽度 ≤ 768（走移动分支），两者都要命中。
- 规则：`.work-grid` 改左右分栏（左列计数/目标，右列敲击区），`.tap-panel` 的 `min-height` 复位为 `auto`（横屏由宽度而非高度承载），`.step-strip` / `.mobile-focus-meta` 隐藏。
- 横屏的验收标准放宽为**「主按钮 + 加减号 + 结束训练按钮可达（允许滚动一屏内）」**，不要求首屏完整可见——它不是需求 1 的验收档位。

#### A7 safe-area 全断点生效

```css
.counter-app {
  padding:
    max(12px, env(safe-area-inset-top))
    max(12px, env(safe-area-inset-right))
    calc(16px + env(safe-area-inset-bottom))
    max(12px, env(safe-area-inset-left));
}
```
仅用于**桌面/横屏**分支。移动竖屏（≤767.98）用 §A2 里那份 padding（顶部固定 8px，**不含** `safe-area-inset-top`，理由见 A2(3)）。

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
| `visibilitychange → 'visible'` | **只做两件事**：置 `needsUnlock = true`；记 `hiddenAt = Date.now()`（会话侧结算用）。**不得**调用 `resume()` / `close()` / 新建 ctx / `play()`。真正的恢复只能发生在**下一个用户手势**里。 |
| `pageshow`（`event.persisted`，iOS bfcache 恢复） | 同 `visible`：置 `needsUnlock = true`、记 `hiddenAt`，**不碰音频**；并补跑一次 1s 时钟逻辑（`checkDayRollover()` + `refreshSpeed()`，且必须过 `isActive` 守卫）——休眠期间可能已跨零点。 |
| `visibilitychange → 'hidden'` | 只 `persistAll()`（保持现状 `:1533-1537`）；**不**关闭 ctx（下次前台还要用）；记 `hiddenAt` 并把会话的当前「活跃段」结算进 `activeMs`（§C1.4）。 |
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
    "hiddenSinceMs": null,
    "lastHitAtMs": null,            // 最后一次有效敲击时间（用于 idle 判定的收尾时间点）
    "speedPeak": 0,
    "splitFrom": null,              // 跨零点拆分时指向上一段 id
    "endReason": null,

    // ↓ B5 新增：会话级有上限敲击日志（内存态，不落盘）
    "taps": [ { "at": 1757980001234, "delta": 1 } ],
    "tapsTotal": 103,               // 会话内累计敲击事件数（含已被 FIFO 截断掉的）
    "tapsDropped": 0                // 因上限被丢弃的条数
  },
  "sessions": [ /* 已完成会话，结构见下；按 startedAt 倒序；上限 200 条 */ ]
}
```

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
  "taps": [ /* 仅当未决 #8 定为「允许结束后改单次敲击」时才写入，见 C1.7 */ ]
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
   - 检测到日期翻转且存在 active 会话时：以 `endedAt = 当日 23:59:59.999` 收尾（`endReason='midnight'`），若此刻页面在前台，立即以 `startedAt = 次日 00:00:00.000` 开新会话（同目标，`splitFrom` 指向上一段，`countAtStart` 取当前 `todayCount`）。
   - 只在**前台**检测到翻转时才自动续开（守卫见 C1.8）；后台睡过零点则按「idle 规则」处理（不会出现一段横跨 8 小时的假会话）。
4. **路由离开 / 息屏**：
   - 先修 keep-alive 缺陷（见 C4）：`onDeactivated` / `visibilitychange hidden` **不结束会话**，只把当前「活跃段」结算进 `activeMs`、置 `activeSinceMs = null`，并记 `hiddenSinceMs`。
   - **回到前台**：若 `now - hiddenSinceMs ≤ SESSION_IDLE_MS`（默认 **30 分钟**，常量）→ 自动续跑，`activeSinceMs = now`，`backgroundMs += now - hiddenSinceMs`。
   - **超过 idle 阈值**：以 `endedAt = max(lastHitAtMs, hiddenSinceMs)` 收尾（`endReason='idle'`），不再自动开新会话（下次敲击时给出「开始新训练？」轻提示，不阻塞计数）。
   - `onDeactivated`（路由离开）与 `visibilitychange hidden` 走**同一套**规则，行为一致，**不引入第二种语义**——所以 `endReason` 里**没有** `route-leave`（C1.9）。
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
   - **模型**：`active.taps: [{ at, delta }]`，`MAX_SESSION_TAPS = 500`（常量）。
   - **溢出策略**：FIFO，丢弃最旧的；`tapsDropped++`，`tapsTotal++`。时间轴 UI 在列表底部显示「仅显示最近 500 次，更早的 N 次未列出」（`tapsDropped > 0` 时）。
   - **为什么是 500**：30 分钟高强度训练（4 次/秒）= 7200 条；每条 `{at,delta}` 序列化约 22 字节 → 7200 条约 158KB，单键整写会拖慢敲击路径并逼近 5MB 配额。500 条约 11KB，配合 200 条会话（每条**不内嵌** taps）总占用 < 200KB。
   - **落盘策略**：`active.taps` **不落盘**（内存态）；`counter_v4_training` 只写 `tapsTotal` / `tapsDropped`。理由：① 时间轴只在会话结束后需要；② 避免 5s 节流落盘时序列化 11KB 数组。
   - **若未决 #8 定为「允许结束后改单次敲击」**：在 `end()` 时把 `taps` 快照**一次性**写进 `sessions[]` 的那一条（同样受 500 上限与 FIFO 截断），此后不再更新。
   - **删一条的语义**：从 `taps` 移除该条 → `todayCount -= delta`（下限 0）→ 派生值 `session.count` 自动同步。若目标条目已被 FIFO 截断，UI 不提供入口（时间轴只列 `taps` 中的条目）。
8. **跨零点拆分的实现顺序（非阻塞 3 —— 顺序错了会丢数据）**：
   ```js
   function checkDayRollover() {
     // ① 先取会话快照（此刻 todayCount 还是「昨天」的值）
     const sessionSnapshot = session.countAtRollover(todayCount.value)   // = todayCount - active.countAtStart
     // ② 会话收尾 / 拆分（endReason='midnight'，新段 countAtStart = 0）
     session.handleRollover(now)
     // ③ 最后才 archiveDay + 清零（现有 CounterTool.vue:1115-1123）
     archiveDay(currentDate.value, todayCount.value, bestSpeed.value)
     currentDate.value = todayKey
     todayCount.value = 0        // ← CounterTool.vue:1117
     ...
   }
   ```
   必须在 `todayCount.value = 0`（`CounterTool.vue:1117`）**之前**取快照，否则会话计数恒为 0。
   「只在前台才自动续开」需要守卫：1s 时钟在后台**仍在跑**（`CounterTool.vue:1545-1548`，`setInterval` 不受后台影响），所以续开分支必须写成 `if (isActive.value && document.visibilityState === 'visible')`。
9. **`endReason` 取值域（非阻塞 2 —— 收敛为 4 个）**：`manual | idle | midnight | reset`。
   - **删除 `route-leave`**：C1.4 已定「路由离开与切后台走同一套规则、当时不结束会话」，若真的超过 idle 阈值收尾，落到的 reason 就是 `idle`，不需要单独取值。
   - 各值触发点：`manual` = 用户点「结束训练」并二次确认；`idle` = 超过 `SESSION_IDLE_MS` 未回前台；`midnight` = 跨零点拆分；`reset` = `resetToday()` / `resetAll()` 打断。
   - `active.endReason` 在会话进行中恒为 `null`。

#### C2 存储写入频率与配额

- **禁止每次敲击写 `counter_v4_training`**。内存态为权威，落盘时机：① 1s 时钟里节流（距上次落盘 ≥5s 且计数有变化）；② `visibilitychange hidden`；③ `pagehide` / `beforeunload`；④ 会话开始/暂停/结束/拆分等状态迁移；⑤ `onDeactivated`。
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
- `position: fixed; inset: 0; z-index: 900`（高于外壳 `.mobile-header` 的 `z-index:100`，**低于** Element Plus 弹层的 2000 层级），自带 `padding: env(safe-area-inset-*)`
- 结构：顶部信息条（计数 / 目标进度 / 已用时长 / 节奏）+ 中部**整块敲击面** + 底部「暂停/继续」「结束训练」
- 层内 `touch-action: none; overscroll-behavior: none; user-select: none` → 不可能产生滚动/缩放手势
  - **注意（非阻塞 9）**：`touch-action: none` 会让层内**任何**区域都不可滚动 → **会话总结面板不放在层内**。结束训练 → 退出全屏层 → 在日间模式用现有 `panel-overlay`（`showHistory` 那一套）展示总结 + 时间轴。层内只有顶栏 + 敲击面 + 底栏，都不需要滚动。（若将来必须在层内展示，必须给该 `div` 单独 `touch-action: pan-y; overflow-y: auto`，本方案不采用。）
- **锁定态下如何仍然计数**：计数面本身**不加**任何长按/确认门槛——`pointerdown` 即计数（B1）。锁定只作用于**退出/破坏性操作**（结束训练、暂停需要长按）以及**隐藏/禁用**其它所有控件。锁定的语义是「**锁 UI，不锁计数**」。

**③ 训练期禁用 / 降级清单（需求 4 明确要求）**

| 控件 | 训练模式（全屏层内） | 理由 |
|------|-------------------|------|
| 主敲击面 | **保留**，全屏、无任何确认门槛，`pointerdown` 即计数 | 计数（B1 硬口径） |
| 结束训练 | 保留，**长按 700ms** + 二次确认「结束并保存本次训练？」 | 破坏性/会话边界 |
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
- 允许删除某一条 → 同步 `todayCount -= delta`（保持 §C1 的计数不变量）。
- `tapsDropped > 0` 时面板底部显示「仅显示最近 500 次，更早的 N 次未列出」。

**⑦ 训练层内的 `el-dialog`（未定义边界，v2 定义）**：
- 训练层 `z-index: 900` **低于** Element Plus 的 2000 层级 → 别处触发的 `el-dialog`（设置面板 / 历史 / 日历 / 目标达成模态）会**盖在训练层之上**并遮挡敲击面。
- **定义**：进入训练层时**同步关闭所有 EP 浮层**——`showSettings = false; showCalendar = false; showHistory = false; goalReached = false`；且训练层内**不渲染**任何打开它们的入口（③ 清单已禁用）。因此训练中**不存在合法触发路径**。
- 万一出现意外浮层（实现缺陷），**不在计数面加守卫**（那会吞计数）——记录为实现缺陷，由 code-reviewer / QA 按 ③ 清单核查。

---

#### D3 `resetAll()` / `clearHistory()` / `resetToday()` 与会话的交互（B7 + 未定义边界）

| 入口 | 位置 | v2 定义 |
|------|------|--------|
| **`resetAll()` 清除全部数据** | `CounterTool.vue:1510-1531` | ① 追加 `localStorage.removeItem('counter_v4_training')`；② 内存态清空 `sessions = []`、`active = null`；③ `prefs` **一并重置为默认**（因为按钮语义是「全部数据」且现有实现已经 `removeItem` 掉整个 `counter_v4_state`，把 `dailyGoal`/`step`/`volume` 一起清了，保持一致）；④ 若会话进行中 → 先按 `resetToday` 的规则结束会话（`endReason='reset'`）并关闭训练层；⑤ 确认文案改为 **「清除今日计数、全部历史记录和训练记录（含训练设置）？此操作不可恢复。」** |
| **`clearHistory()` 清空历史** | `CounterTool.vue:1498-1508` | ① **一并清空会话历史**（`sessions = []`）——会话历史与 `dailyRecords` 属同一类「历史数据」，分开清会让用户以为清干净了其实没有（与 B7 同一类隐私预期问题）；② **不动进行中的 `active`**（清历史 ≠ 结束当前训练）；③ 确认文案改为 **「清空全部历史记录（含训练会话），但保留今天的计数与进行中的训练？」** |
| **`resetToday()` 重置今日** | `CounterTool.vue:1478-1497` | ① 会话**立即结束**：`endReason='reset'`，`endedAt = now`，`count = max(0, todayCount_at_end − countAtStart)`；② `active = null`；③ 若 `prefs.trainingMode === true` → 一并置 `false` 并退出全屏层（今天已经归零，继续待在训练层没有意义）；④ 确认文案改为 **「重置今日计数将结束当前训练会话，确认？」**（现状文案「重置今日计数、节奏和撤销状态？」没说清会结束训练） |
| **会话进行中点「清除全部数据」** | — | 同 `resetToday` 的处理顺序：先结会话（`endReason='reset'`）+ 关训练层，**再**清盘 |

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
| **训练中从别处触发 `el-dialog`** | 见 D2⑦：进层时强制关闭全部 EP 浮层、层内不提供入口；意外浮层不通过计数面守卫处理。 |
| **`durationMs` 主口径下的派生统计** | 本次不做周/月训练时长统计。若未来新增，口径 = `Σ activeMs`（有效时长），不得用 `durationMs`。会话总结面板**必须同时给出**「实际时长」与「有效时长」。 |
| **`resetAll()` / `clearHistory()` 与会话** | 见 D3。 |
| **`endReason` 取值域** | 见 C1.9：`manual \| idle \| midnight \| reset`。 |
| **会话进行中 `dailyGoal` 被改** | 不影响会话：会话目标是 `active.goal` 的快照，`dailyGoal` 改动只影响日统计展示与 `goalPercent`。 |
| **同一天开多个会话** | 允许。`startDate` 相同，`count` 各自独立；不变量 `Σ sessions(startDate=D).count ≤ 当日归档 count` 仍成立（会话之间不重叠）。 |

### 实施拆分（交下游的派工建议）

| 步骤 | 负责 | 产物 |
|------|------|------|
| 1 | frontend-writer | `frontend/src/utils/counterTapGuard.js` + `.test.js`（纯函数，先写测试；**只含长按守卫**，无计数门限） |
| 2 | frontend-writer | `frontend/src/composables/useCounterAudio.js` + `.test.js`（注入 fake ctx） |
| 3 | frontend-writer | `frontend/src/composables/useCounterSession.js` + `.test.js`（注入 now/storage） |
| 4 | frontend-writer | `frontend/src/components/CounterTrainingOverlay.vue` |
| 5 | frontend-writer | `CounterTool.vue` 接线（模板/脚本/样式）；**移动端样式改动严格按 §A2/§A3 的算式** |
| 6 | lint-runner | 对改动文件跑 `npx eslint` / `node --check`（按该 agent 现有流程） |
| 7 | code-reviewer | R3（模板无 `.value`）/ R4（裸下标）/ 相对路径引用 / **§测试要点 L2 的 5 条负向 grep** |
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
| 5 | 全屏训练层与外壳 z-index / 弹层冲突 | 弹层被遮或训练层被遮 | 训练层固定 `z-index: 900`（外壳 header 100 / EP 弹层 2000+）；D2⑦ 已定「进层时关闭全部 EP 浮层」，并对 EP 弹层做一次人工回归 |
| 6 | 断点从 720 改到 767.98 触发未知回归 | 平板竖屏（768）进入桌面布局 + 文档层滚动 | 767.98 与外壳 `isMobile` 本来就一致，改动方向是**消除**不一致；恰好 768px 的行为已在 §A1 写明（功能不损），QA 需覆盖 767 / 768 / 769 三档 |
| 7 | localStorage 配额 | 写入失败 | try/catch + 丢弃最旧 50 条重试一次 + 静默放弃；`active.taps` 不落盘（11KB 只在内存），绝不影响计数 |
| 8 | keep-alive 下 `onActivated`/`onDeactivated` 未覆盖所有路径（`currentViewKey` 变化导致重建） | 守卫失效 / 监听累积 | C4：`onMounted` 也置 `isActive=true`；`addGlobalListeners()` 幂等；`applyDelta` + `onKeyDown` 双重守卫；QA 用例覆盖「切走→切回→再切走」 |
| 9 | Wake Lock 被拒绝 / 不支持 | 屏幕仍会熄灭 | D4 的静默降级路径；B2 的音频恢复是主要保障，Wake Lock 只是降低触发概率 |

---

## 测试要点（交 devtools-qa，可执行、可判定）

> 运行方式：`cd frontend && npx vitest run`（仓库已有 vitest，`frontend/package.json:11` → `"test": "vitest run"`；配置 `frontend/vitest.config.js`，environment=jsdom，include `src/**/*.test.js`，`@` 别名指向 `./src/neon`）。
> **不要**起 dev server（R1）。L3 真机项由人工执行并回报。

### L1 纯函数单测（必须全绿）

**`counterTapGuard.test.js`**（v2 重写：只测长按守卫）

1. `isLongPressCommit({ startAt: 0, endAt: 599, thresholdMs: 600 })` → `false`
2. `isLongPressCommit({ startAt: 0, endAt: 600, thresholdMs: 600 })` → `true`（边界含等号）
3. `createLongPressGuard`：`start(1)` 后立刻 `cancel(1)` → `takeCommit(1)` 返回 `false`
4. `createLongPressGuard`：`start(1)` → 时间推进 700ms → `takeCommit(1)` 返回 `true`
5. `cancel()` 对未知 pointerId 不抛异常（防御性）
6. **负向断言（B1 闸门）**：模块**不得**导出 `shouldCommitTap` / `TAP_MOVE_TOLERANCE_PX` / `TAP_MAX_DURATION_MS` / `TAP_DEBOUNCE_MS`（`expect(mod.shouldCommitTap).toBeUndefined()` 等 4 条）

**`useCounterSession.test.js`**（注入 `now` / 假 storage；`vi.useFakeTimers` + `vi.setSystemTime`）

7. `start()` → `end()`：`durationMs === endedAt - startedAt`
8. 暂停 60s 后继续：`activeMs === durationMs - 60000`，`pausedMs === 60000`
9. 后台 5 分钟（模拟 `hidden` → `visible`）：`backgroundMs === 300000`，`activeMs` 不含该区间
10. 后台超过 `SESSION_IDLE_MS`（30 分钟）：会话被自动收尾，`endReason === 'idle'`，`endedAt === max(lastHitAt, hiddenAt)`
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
42. **负向 grep（B2）**：`grep -n "\.resume()\|\.close()\|new AudioContext\|new (window.AudioContext" frontend/src/composables/useCounterAudio.js` → 只允许出现在 `unlock()` 与 `play()` 两个函数体内；**`handleForeground()` / `pageshow` 相关路径零命中**
43. **负向 grep（B6）**：`grep -n "dailyGoal" frontend/src/composables/useCounterSession.js` → **无命中**（会话模块不得读日目标）
44. **负向 grep（B5）**：`grep -n "tapEvents" frontend/src/composables/useCounterSession.js` → **无命中**；`grep -n "tapEvents" frontend/src/views/life/CounterTool.vue` 的命中只允许在 `refreshSpeed` / `registerHit` / `checkDayRollover` 的既有逻辑里，**不得出现在会话快照代码中**
45. **负向 grep（D2 挂载点约束）**：`grep -n "transform\|will-change\|perspective\|contain:" frontend/src/App.vue` 中 `.app-container` / `.main-content` / `.mobile-main` 选择器下**无命中**（否则 `position: fixed` 训练层会被裁剪）
46. **移动端算式自检**：`grep -n "100dvh\|100vh" frontend/src/views/life/CounterTool.vue` → 移动断点（`max-width: 767.98px`）区块内**不得**出现未减去 `--ct-chrome-top` 的裸 `100dvh`
47. `grep -n "counter_v4_training" frontend/src/views/life/CounterTool.vue` → `resetAll()` 内必须有一处 `removeItem`

### L3 真机 / 手工（需人工执行并回报结论）

> 设备矩阵：iOS Safari（含刘海机型）、Android Chrome；视口 320×568 / 375×667 / 390×844 / 414×896，横屏 844×390。

#### 布局（需求 1 核心 —— B4 闭环）

48. **日间模式（不进训练层）首屏可见性**：上列每个**竖屏**视口下，把文档滚到顶部（`scrollTop === 0`）后断言——`.figure-button` 与两个 `.side-round` 的 `getBoundingClientRect().bottom ≤ window.innerHeight`，且无横向滚动条
49. **375×667 首屏完整性**：`今日次数`（`.mobile-focus-count strong`）、主敲击按钮、加减号、`.mobile-main-actions` 操作行，**全部**在首屏（`bottom ≤ innerHeight`）
50. **320×568 首屏完整性**：`今日次数`、主敲击按钮、加减号全部在首屏；步长条/焦点条允许被挤到首屏外（按 §A3 的降级顺序处理并回报实际结果）
51. **训练层首屏**：进入训练后，计数 + 敲击面 + 「结束训练」按钮**无需滚动**即可全部看到；刘海/home indicator 不遮挡
52. **断点切换**：767 / 768 / 769 三档宽度切换窗口——767↔769 的布局切换点与外壳 header 出现/消失**同一时刻**；恰好 768px 时按 §A1 的预期行为（桌面布局 + 文档层滚动，功能不损）并回报实际观感
53. **横屏 844×390**：内容不重叠，安全区左右有留白，主按钮 + 加减号 + 结束训练可在滚动一屏内到达

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
