# 敲击计数器：移动端适配 / 息屏音频恢复 / 运动会话与目标 / 防误触 — 技术方案

- 子 issue：FINN-10（父 FINN-9）
- 模块：`frontend/src/views/life/CounterTool.vue`（纯前端，无后端）
- 状态：设计稿，**不含任何代码改动**；实现由 frontend-writer 执行，验收由 devtools-qa 执行
- 勘察方式：全部结论来自本仓库实际 grep / 逐段 Read；**未起 dev server、未跑 go build、未在真机验证**。凡属推算的数值均在文中显式标注「推算」

---

## 目标与非目标

### 目标（严格对应需求四条）

| # | 需求 | 本方案的产出 |
|---|------|-------------|
| 1 | Bug：移动端页面展示不能适应窗口 | 拆掉「视口高度硬编码 + 嵌套滚动」结构；给出 320/375/390/414 竖屏与 844×390 横屏下的布局与断点方案；补齐 safe-area |
| 2 | Bug：息屏后再点击页面没有声音 | 音频生命周期重写：`interrupted` 状态、`resume()` Promise 未 await、ctx 被回收后的自愈、失败降级与用户可见反馈 |
| 3 | 功能优化：运动场景（计数口径 / 实际开始结束时间 / 目标） | 新增「训练会话」数据模型 + 存储迁移；会话统计与现有日统计**隔离**；会话目标与日目标分离 |
| 4 | 交互要求：方便操作、不能误操作 | 全屏训练面（锁定）+ 指针手势判定器（按下→抬起提交）+ 训练期禁用/降级清单 |

### 非目标（本次明确不做）

- **不加后端**：无新 handler / 路由 / SQLite 表 / Redis 使用，全模块保持 0 网络请求（见「R2 结论」）
- **不做跨设备同步**、不做账号、不做云端历史
- **不改现有日统计口径**：`dailyRecords` / `streak` / `weekData` / `calCells` / `grandTotal` / `bestSpeed` 语义与计算方式一律不动
- **不改音效合成算法**（`buildMokugyo` / `buildBell` / `buildDrum` / `buildChime` 波形代码不动），只改「什么时候能响」
- **不改形象 / 主题预设体系**，不新增预设
- **不引入新 npm 依赖**（不引 Howler 等；沿用 WebAudio 手写实现）
- **不改 App.vue 外壳**（路由/侧边栏/移动端 header 一律不动；见「未决问题 #11」）
- **不做 PWA / Service Worker / 后台播放**：本工具是前台工具，锁屏期间不产生敲击

---

## 现状（本仓库已核实）

> 所有 `file:line` 均经 grep/Read 实测确认，非需求描述转抄。

### 0. 模块边界

- 路由：`frontend/src/router/index.js:591` → `path: '/counter'`，`name: 'Counter'`，**无 `hideSidebar`**（对比：只有分享类路由带 `hideSidebar`，见 `frontend/src/router/index.js:629/635/641/647/653/659/665`）
- 唯一实现文件：`frontend/src/views/life/CounterTool.vue`，2806 行（模板 1–440，脚本 442–1565，样式 1567–2806）
- `grep -ril counter backend --include="*.go"` → **无命中**。确认纯前端
- 无既有测试：`find frontend/src -name "*Counter*"` 只命中 `CounterTool.vue` 本身

### 1. 移动端适配现状（→ 需求 1）

| 事实 | 位置 |
|------|------|
| `.counter-app { min-height: 100vh; min-height: 100dvh; padding: 24px 16px 40px; overflow: hidden }` | `CounterTool.vue:1568-1576` |
| 只有 ≤720px 才补 safe-area，且只补上下、不补左右 | `CounterTool.vue:2410-2414` |
| ≤720px：`.tap-panel { min-height: calc(100dvh - 26px - env(safe-area-inset-top) - env(safe-area-inset-bottom)) }` | `CounterTool.vue:2451-2457` |
| ≤720px：`.figure-button { min-height: 360px }`；≤420px：`330px` | `CounterTool.vue:2571-2576` / `2774-2776` |
| ≤720px：`.side-round` 绝对定位 `bottom:14px; left:14px`，56px；≤420px 52px | `CounterTool.vue:2555-2564` / `2790-2795` |
| 响应式断点仅 3 个：980 / 720 / 420 | `CounterTool.vue:2400 / 2409 / 2751` |
| 外壳：`.main-content { height:100dvh; overflow-y:auto; padding:20px }` | `App.vue:1032-1042` |
| 外壳移动端：`isMobile` 时套 `.mobile-main { padding-top: calc(56px + 15px); padding-bottom: 100px; height:auto; min-height: calc(100dvh - 56px) }` | `App.vue:190` / `App.vue:1044-1051` |
| `isMobile` 阈值 = **768px**（JS 判断，不是媒体查询） | `App.vue:464` |
| 移动端固定 header 高 56px（`position: fixed`） | `App.vue:567-580` |
| 页面底部还有 `.page-footer`（`hideSidebar=false` 时渲染），带 `margin-top: 40px; padding: 20px 0` | `App.vue:197-198` / `App.vue:1054-1059` |
| viewport 已含 `viewport-fit=cover`，safe-area 变量可用 | `frontend/index.html:8` |
| `* { touch-action: manipulation }` 全局已设，禁用了双击缩放 | `frontend/index.html:16-18` |

**根因（三条，互相叠加）**

1. **720 / 768 之间存在断点裂缝**：`721–767px` 宽度下 `isMobile=true`（外壳按移动端渲染：56px 固定 header + 71px 顶部内边距 + 100px 底部内边距 + footer），但计数器自身仍走桌面布局（`overflow: hidden`、`.figure-button` 无移动端样式）。该区间内页面内容（hero 卡 + 预设 + work-grid 堆叠）远超可视高度，且外层内边距已经把可用高度吃掉 227px。
2. **移动端高度硬编码叠加**：`.counter-app` 的 `min-height: 100dvh` 是相对**视口**；而它实际被塞进一个已经扣掉 header(56) + 顶部内边距(71) + 底部内边距(100) 的容器里。`.tap-panel` 的 `min-height: calc(100dvh - 26px - 安全区)` 同理，也是相对视口。

   **推算**（按上述 CSS 逐项相加，非真机实测）：375×667（iPhone SE2/3）竖屏下，非 tap-stage 部分占用 ≈415px（外壳 171 + counter-app 上下 padding 22 + tap-panel 上下 padding 22 + 焦点条 ~74 + 步长条 52 + 底部操作行 64 + 间距 10），留给 tap-stage 只有 **≈252px**，而 `.figure-button` 在该宽度下 `min-height: 330px` → **溢出 ≈78px**；421–720px 宽时是 360px 下限 → 溢出更多。`.side-round` 定位在 tap-stage 的**底边**（`bottom:14px`），于是横竖两个加减按钮整体落在首屏之外。
3. **嵌套滚动**：≤720px 时 `.counter-app` 自己也是滚动容器（`overflow:auto` + `overscroll-behavior-y: contain`，`CounterTool.vue:2412-2413`），而外层 `.main-content` 也是滚动容器。内层 `overscroll-behavior: contain` 会**阻断滚动链传递**，用户在内层滚到底后无法顺势滚到外层，导致首屏外的控件（含加减按钮、撤销/自动/功能行、周热力面板）在某些姿势下够不到。
4. **横屏加宽反而失去安全区**：iPhone 横屏宽度 812/844/852/932px 全部 > 720px → 走桌面分支 → 完全不加 `env(safe-area-inset-*)`，刘海/home indicator 压住内容；且该分支下 390px 的高度被 hero 卡占掉大半。

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
| **`normalizeRecords()` 会重建每条记录为 `{date, dayName, count, peakSpeed, dailyGoal}` 五个字段**，其余字段静默丢弃；且 `count<=0 && peakSpeed<=0` 的记录直接丢 | `CounterTool.vue:976-1011`（关键约束，见下） |
| 载入时若 state 的 `currentDate` 等于今天，会从 `dailyRecords` 中剔除今天那条（避免与内存中的 `todayCount` 双算） | `CounterTool.vue:1075-1078` |
| `checkDayRollover()`：日期变了就 `archiveDay(旧日期, todayCount, bestSpeed)` 然后清零 | `CounterTool.vue:1108-1123` |
| 跨零点只在 1s 时钟（`1545-1548`）和 `applyDelta` 开头被触发 | `CounterTool.vue:1146` |
| 现有唯一「目标」概念 = `dailyGoal`（按天、按次数），驱动 `goalPercent` / `streak` / 日历达标标记 | `CounterTool.vue:704 / 796-822 / 399 / 426-432` |
| 速度口径：15s 滑动窗口，跨度下限 clamp 到 0.03min（=1.8s），`bestSpeed` 取窗口速度最大值 | `CounterTool.vue:1185-1200` / `460` |

**关键约束（必须写进实现）**：`normalizeRecords()` 在每次载入时重建记录对象并**丢弃未知字段**。因此**绝对不能**把会话数据挂到 `dailyRecords` 的某条记录上（例如给当天记录加 `sessions: []`）——下次刷新就会被静默清空。会话数据必须独立成键。

### 4. 误触与生命周期现状（→ 需求 4）

| 事实 | 位置 |
|------|------|
| 主按钮 `@pointerdown="handleFigureTap"`：**按下即计数**，无位移阈值、无时长阈值、不看 `isPrimary` | `CounterTool.vue:166-171` |
| 右侧加号 `@pointerdown.prevent="increment"`；左侧减号 `@pointerdown.prevent="decrement"`（`todayCount<=0` 时 disabled） | `CounterTool.vue:157-164 / 182-188` |
| 步长 chip（+1/+3/+5/+10/+20）紧贴敲击区上方，单击即生效 | `CounterTool.vue:143-154` |
| 自动连点：`setInterval` 最小 120ms，每 tick 也是 `increment()` → 会计数 | `CounterTool.vue:706-707 / 1206-1218` |
| 目标达成用 `el-dialog` 模态弹出，`close-on-click-modal=false` → 训练中挡住敲击区，用户的下一次敲击落在遮罩上 | `CounterTool.vue:279-293` |
| **`App.vue:191-193` 用 `<keep-alive :exclude="[]">` 包住 `router-view`，且 `:key="currentViewKey"`（`App.vue:323` = `route.fullPath:viewRefreshKey`）** | `App.vue:190-193 / 323` |
| `CounterTool.vue` 全文 **没有** `onActivated` / `onDeactivated` | `grep -n "onActivated\|onDeactivated"` → 无命中 |
| 清理逻辑全在 `onUnmounted`（清定时器、摘 `document` 的 keydown、摘 visibilitychange、`persistAll`） | `CounterTool.vue:1555-1564` |

**由此得出一个此前未被发现的真实缺陷**：因为 keep-alive 命中，**路由离开 `/counter` 时组件不会 unmount**，`onUnmounted` 不执行 → `document` 上的 `keydown` 监听、1s 时钟、`visibilitychange` 监听**全部继续存活**。后果：在别的工具页按空格/↑，`onKeyDown`（`CounterTool.vue:1443-1465`）仍会调用 `increment()`，**计数器在后台静默增长并写盘**；`autoMode` 若开着也会继续自增（`startAuto` 的定时器同样没被清）。这既属于「计数不准」也属于需求 4 的误触范畴，且直接影响会话「路由离开如何收尾」。

---

## 设计

### A. 移动端适配

#### A1 断点统一

- 移动断点从 **720 → 768**，与外壳 `isMobile`（`App.vue:464`）严格对齐，消除 721–767 的裂缝。保留 480（原 420 微调位）与新增横屏断点。
- 最终断点表：

| 断点 | 用途 |
|------|------|
| `max-width: 1024px` | 双列 → 单列过渡（保留原 980 语义，仅调数值） |
| `max-width: 768px` | 移动布局（与外壳一致） |
| `max-width: 480px` | 小屏字号/内边距微调（原 420） |
| `orientation: landscape and (max-height: 480px)` | 手机横屏专用（左计数、右敲击区） |

#### A2 高度模型：从「视口高度」改为「容器高度」

- `.counter-app`：删除 `min-height: 100dvh`，改 `min-height: 100%`；`overflow: hidden` → `overflow: clip`（保留溢出的氛围光斑裁剪，但不再制造滚动容器；`overflow: hidden` 作前一行兜底）。
- 移动端删除 `.counter-app` 自身的 `overflow: auto` + `overscroll-behavior-y: contain`（`CounterTool.vue:2412-2413`）→ **全页只保留外层 `.main-content` 一个滚动容器**，杜绝嵌套滚动与滚动链阻断。
- `.tap-panel`：`min-height: calc(100dvh - 26px - …)` → `min-height: 0; flex: 1 1 auto`，并新增 `min-height: min(56dvh, 460px)` 作为下限（保证敲击区可用高度，同时不撑破容器）。

  为什么这样算得出来：`.counter-app` 的高度改由父容器（`.mobile-main`，flex 子项，高度确定）决定后，`100%` 解析结果就是真实可用高度；此时任何基于 `dvh` 的减法都会与外壳 chrome 重复扣减。
- 横屏：`@media (orientation: landscape) and (max-height: 480px)` 下 `.work-grid` 改左右分栏（左列计数/目标，右列敲击区），`.tap-stage` 高度 = 剩余高度，`min-height: 160px`。
- safe-area 全断点生效（不再是 ≤720 专属）：

```css
.counter-app {
  padding:
    max(12px, env(safe-area-inset-top))
    max(12px, env(safe-area-inset-right))
    calc(16px + env(safe-area-inset-bottom))
    max(12px, env(safe-area-inset-left));
}
```

#### A3 竖屏内的空间回收

仅靠改高度模型仍不够（推算：375×667 下 tap-stage 只有 ≈252px，而敲击区实际需要 ≥300px 才谈得上「运动中好按」）。因此**训练模式（见 C/D）在移动端隐藏非计数控件**，把空间让给敲击区：

- 训练模式移动端隐藏：步长 chip 条（52px）、`.mobile-main-actions` 行（64px）、周热力 panel（整块 ≥250px）、hero/预设（移动端本就 `display:none`，`CounterTool.vue:2420-2423`）
- 回收后 tap-stage 可用高度 ≈ **252 + 116 = 368px**（仅靠 A2+A3，不含训练全屏层）
- **训练模式采用全屏覆盖层（见 D2）时，敲击面高度 = 100dvh − 顶栏 56 − 底栏 72** → 375×667 下 ≈ **539px（≈81dvh）**，844×390 横屏下 ≈ **262px**，均 ≥ 200px 下限

#### A4 主按钮触达尺寸与位置（需求明确要求给出 px/vh 依据）

| 元素 | 尺寸结论 | 依据 |
|------|---------|------|
| 训练模式敲击面 | 宽 100%（减安全区），高 = `max(200px, calc(100dvh - 128px - 安全区))` | 375×667 实测推算 539px≈81dvh；844×390 横屏 262px。参照 Apple HIG 最小 44×44pt，主操作应远大于最小值；45dvh 下限保证最低端机型也有 300px+ |
| 训练模式底栏「结束训练」 | 高 56px、宽 ≥132px、距屏幕底边 ≥ 安全区 + 12px | HIG 44pt / Material 48dp 的最小值上取一档，避免运动中手抖 |
| 左边缘留白 | 训练层左右各 `max(24px, env(safe-area-inset-left))` | **iOS Safari / Android 都有边缘返回手势**（左侧 ~20px 起），贴边敲击会被系统判为返回手势 |
| 日间模式减号 `.side-round` | 保持 56px（≥48dp 达标），但位置必须移出敲击区（见 D） | Material 48dp |
| 任意可点元素底线 | ≥48×48px，相邻可点元素间距 ≥8px | Material 触控目标规范 |
| 拇指可达区 | 主要操作落在视口下 60% 区域内 | 单手竖握时拇指自然活动范围；训练层底栏天然满足，敲击面占满其余全部区域 |

#### A5 需要一并修掉的外壳残留（属本模块可改范围）

- `.mobile-main` 的 `padding-bottom: 100px`（`App.vue:1047`）与 `.page-footer`（`App.vue:197`/`1054`）在计数器页产生约 100–140px 的纯死滚动。**不修改 App.vue**（外壳属共享代码，见非目标），改为在计数器页内用 `margin-bottom: calc(-100px - env(safe-area-inset-bottom))` 抵消？——**否决**：负 margin 依赖外壳内边距的具体数值，属脆弱耦合。
  **采纳方案**：训练模式用 `position: fixed; inset: 0` 全屏覆盖层（脱离外壳布局流），根本不受外壳内边距影响；日间模式的死滚动交给 QA 记录，作为已知残留（若不可接受，再单开子 issue 讨论是否给 `/counter` 加 `hideSidebar: true`，见「未决问题 #11」）。

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
    unlock(),                     // 必须在用户手势内同步调用，返回 Promise<void>
    play(soundName, volumeRatio), // 永不抛；内部决定是否延后播放
    isDegraded,                   // ref<boolean>，UI 用它显示「音效已暂停，点此恢复」
    rebuild(),                    // 强制丢弃旧 ctx + 缓冲，重建（用户手势「恢复音效」按钮调用）
    dispose()
  }
}
```

#### B2 状态机与恢复阶梯（按顺序尝试，前一层失败进下一层）

```
ensureCtx():
  if (!ctx || ctx.state === 'closed')  → ctx = ctxFactory(); buffers.clear(); unlocked=false   ← 第 3 层
  if (ctx.state === 'interrupted' || ctx.state === 'suspended') → 第 1 层
  if (ctx.state === 'running') → 就绪

第 1 层（首选）：ctx.resume()  ← 必须 async/await，且必须判断 'interrupted'
第 2 层：静音 buffer 解锁 —— resume() 成功后，立刻 connect 一个 1 帧静音缓冲区并 start()，
        把 ctx 从「已恢复但输出管线未激活」状态真正推起来（iOS 经典解锁手法）
第 3 层：ctx.close() + 新建 AudioContext + 重建全部音色缓冲（soundBuffers 属于旧 ctx，必须弃用）
第 4 层：连续 2 次失败 → degraded = true，UI 显示提示胶囊；震动仍可用（vibrate 与音频无关）
```

#### B3 首击不丢（核心修复）

```js
async function play(sound, volume) {
  if (degraded.value) return
  const ctx = ensureCtx()
  if (ctx.state !== 'running') {
    // 关键：不立即 source.start()，而是等 resume 落地后再播
    const ok = await raceWithTimeout(ctx.resume(), resumeTimeoutMs)   // 250ms 上限
    if (!ok) { failCount++; if (failCount >= 2) degraded.value = true; return }
  }
  scheduleSource(ctx, sound, volume)      // 此时 ctx 时钟已恢复，第一击有声
  failCount = 0
}
```

要点：`resume()` 返回 Promise **必须被 await/race** 后再 `start()`（这是当前 `:1236` 缺的那一步）。超过 250ms 未恢复就放弃本次播放并标记，避免用户长时间无反馈。

#### B4 回前台 / bfcache 生命周期

| 事件 | 处理 |
|------|------|
| `visibilitychange → 'visible'` | 若 `ctx.state !== 'running'` → 调 `resume()`（**不 await**，失败也无所谓），并把 `needsUnlock = true` |
| `pageshow`（iOS bfcache 恢复，`event.persisted`） | 同 visible：尝试 resume；并把 1s 时钟与日期检查补跑一次（休眠期间可能已跨零点） |
| `visibilitychange → 'hidden'` | 只 `persistAll()`（保持现状），**不**关闭 ctx（下次前台还要用）；记录 `hiddenAt` 供会话计时 |
| 任意 `pointerdown` / `keydown` 用户手势 | 同步调 `unlock()`：若 `needsUnlock` 或状态非 running → resume，成功后置 `needsUnlock=false` |

**兼容现状**：`primeAudio()`（`:1220-1229`）扩为「在**所有**手势入口调用」——主按钮 `handleFigureTap`、加号 `increment`、减号 `decrement` 三处（现状只有主按钮调用，加减号首次敲是没有预热路径的）。

#### B5 用户可见反馈（当前完全静默）

- `degraded === true` 时，在敲击区顶部显示胶囊：**「音效已暂停 · 点我恢复」**（可点元素 ≥48px 高）。点击即 `rebuild()`（第 3 层），成功后 `degraded=false`。
- 切后台/息屏回来且 1 次 `resume()` 未成功时，**不弹**任何提示（静默重试，因为下一次手势大概率自动恢复）；只有连续 2 次失败才显示胶囊。
- 不做 Toast 打扰（训练中 Toast 会遮挡计数）。

#### B6 ctx 数量铁律

全生命周期只允许存在 1 个 AudioContext；`rebuild()` 必须先 `close()` 旧的再建新的。iOS 对同时存在的 AudioContext 数量有硬上限（约 4–6），超限后 `ctxFactory()` 直接抛异常 → 组件必须 catch 并且不阻塞计数（计数与音频解耦，音频失败不抛到 `applyDelta`）。

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
    "goalValue": 0,                 // 0 = 本次不设目标
    "voiceCue": true,               // 训练中是否语音/音效节拍提示（可选实现）
    "lockExitLongPressMs": 700
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
    "endReason": null
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
  "endReason": "manual",         // manual | idle | midnight | reset | route-leave
  "splitFrom": null
}
```

**设计要点（逐条对应需求 A/B/C 的追问）**

1. **计数口径不叠加**：`session.count` = 该时段内 `todayCount` 的净增量，**不写入** `dailyRecords`、**不进** `grandTotal`/`streak`/`weekData`/`calCells`。这是硬红线——两者叠加就是双算。
   - 实现上以 `countAtStart` 快照 + 当前 `todayCount` 求差，天然覆盖 `undoLast()`（撤销也同步修正会话计数），下限 clamp 0（`resetToday()` 后差为负，会话计数归 0，且 `endReason='reset'` 直接结束会话）。
   - **不变量**：对任意日期 D，`Σ sessions(startDate=D).count ≤ 当日归档 count`。仅作为 UI 呈现约束与测试断言，不做运行时强制（撤销/重置可破坏）。
2. **`durationMs` vs `activeMs` 双口径**：需求要的是「实际开始 / 结束 / 时长」，所以主展示 = `durationMs`（墙钟，真实起止），次要展示 = `activeMs`（有效训练时长）。两者都在模型里，避免以后返工。
3. **跨零点**（`checkDayRollover`，`CounterTool.vue:1108-1123`）：
   - 会话归属日 = `startedAt` 所在日，**跨零点不换归属**。
   - 检测到日期翻转且存在 active 会话时：以 `endedAt = 当日 23:59:59.999` 收尾（`endReason='midnight'`），若此刻页面在前台，立即以 `startedAt = 次日 00:00:00.000` 开新会话（同目标，`splitFrom` 指向上一段，`countAtStart` 取当前 `todayCount`）。
   - 只在**前台**检测到翻转时才自动续开；后台睡过零点则按「idle 规则」处理（不会出现一段横跨 8 小时的假会话）。
4. **路由离开 / 息屏**：
   - 先修 keep-alive 缺陷（见 C4）：`onDeactivated` / `visibilitychange hidden` **不结束会话**，只把当前「活跃段」结算进 `activeMs`、置 `activeSinceMs=null`，并记 `hiddenSinceMs`。
   - **回到前台**：若 `now - hiddenSinceMs ≤ SESSION_IDLE_MS`（默认 **30 分钟**，常量）→ 自动续跑，`activeSinceMs = now`，`backgroundMs += now - hiddenSinceMs`。
   - **超过 idle 阈值**：以 `endedAt = max(lastHitAtMs, hiddenSinceMs)` 收尾（`endReason='idle'`），不再自动开新会话（下次敲击时给出「开始新训练？」轻提示，不阻塞计数）。
   - `onDeactivated`（路由离开）与 `visibilitychange hidden` 走**同一套**规则，行为一致，不引入第二种语义。
5. **会话目标**：`goal: {mode:'count'|'duration', value}`，`value=0` 表示本次不设目标。
   - `dailyGoal` 语义与 UI **完全不动**；会话目标是独立的一次性目标。
   - `reached` 为派生值（不落库时也可重算）：`count` 模式 = `count >= value`；`duration` 模式 = `activeMs >= value*60000`。
   - **默认值建议**：`goalValue` 默认 = `max(0, dailyGoal - todayCount)`（剩余日目标，让一次训练顺带完成日目标），`dailyGoal=0` 时默认 0（自由训练）。这是产品决策，列入未决问题 #2。
6. **计数节奏口径（需求 3 第一点）**：现有 15s 窗口 + 1.8s 跨度下限（`CounterTool.vue:1185-1200`）在运动场景下会虚高（两次急敲 1.8s 内即按 1.8s 折算）。会话内新增**会话级节奏**：`session 平均 = count / (activeMs/60000)`，并在会话总结里同时给出 `speedPeak`（沿用现有窗口峰值，保证与「峰值速度」卡片口径一致）。**不改** `refreshSpeed()` 现有行为，避免动到 `bestSpeed` 卡片与历史记录里的 `peakSpeed`。

#### C2 存储写入频率与配额

- **禁止每次敲击写 localStorage**。内存态为权威，落盘时机：① 1s 时钟里节流（距上次落盘 ≥5s 且计数有变化）；② `visibilitychange hidden`；③ `pagehide` / `beforeunload`；④ 会话开始/暂停/结束/拆分等状态迁移；⑤ `onDeactivated`。
- 用**单键整体写**（`counter_v4_training` 一次性 `setItem`），localStorage 单键写入是原子的，避免多键写到一半崩溃。
- 配额保护：`setItem` 包 try/catch；捕获到配额异常 → 丢弃最旧的 50 条会话后重试一次；仍失败则静默放弃（绝不抛到敲击路径）。
- 上限：`sessions` 保留最近 **200 条**（与 `MAX_HISTORY_DAYS=365` 的日历史解耦，200 条约 40KB，安全）。

#### C3 「会话」与现有统计的关系（一句话结论）

> 会话是**日统计的一个视图切片**，不是新的计数源：会话计数永远是对 `todayCount` 增量的投影；`dailyRecords` / `streak` / `weekData` / `calCells` / `grandTotal` / `bestSpeed` 的计算代码一行不改。

#### C4 顺带修复：keep-alive 导致的跨页误计数（必修，否则会话收尾不可靠）

在 `CounterTool.vue` 新增：

```js
const isActive = ref(true)
onActivated(() => { isActive.value = true;  checkDayRollover(); refreshSpeed(); persistAll() })
onDeactivated(() => { isActive.value = false; stopAuto(); pauseSessionForBackground('route-leave'); persistAll() })
```

并在 `onKeyDown` 首行加 `if (!isActive.value) return`，在 `applyDelta` 首行加同样的守卫（防止 `autoTimer` 与残留监听继续计数）。`onMounted` / `onUnmounted` 中的 `document` / `window` 监听保持不变（keep-alive 下组件常驻，监听只注册一次）。

> 附带收益：`onDeactivated` 里 `stopAuto()` 修掉了「离开页面后自动连点仍在跑」的缺陷。

### D. 防误触

#### D1 真实误触来源（逐条，含证据）

| # | 来源 | 证据 | 后果 |
|---|------|------|------|
| 1 | `@pointerdown` **按下即计数** | `CounterTool.vue:170 / 163 / 187` | 滑动/翻页/多指握持只要落在敲击区就算一次；双指同时按 = 计 2 次（无 `isPrimary` 判断） |
| 2 | 减号按钮落在敲击区内的拇指休止位 | `CounterTool.vue:2555-2564`（`bottom:14px; left:14px`，与全宽 `.figure-button` 重叠） | 左手握持时拇指一按就减，直接破坏计数 |
| 3 | 步长 chip 紧贴敲击区上方，单击即改倍率 | `CounterTool.vue:143-154` | 误触后从 +1 变 +20，后续每次敲击都错 20 |
| 4 | 自动连点开关离敲击区近 | `CounterTool.vue:195-203` / `256-264` | 误开后计数器自增，计数完全失真 |
| 5 | 撤销单击即生效 | `CounterTool.vue:192` / `253` | 抹掉刚敲的一次 |
| 6 | 目标达成模态弹出挡住敲击区 | `CounterTool.vue:279-293`（`close-on-click-modal=false`） | 训练中被挡住，连续几次敲击全部丢失（不计也不减） |
| 7 | 路由离开后监听仍活着 | `App.vue:190-193` + `CounterTool.vue:1443-1465`（无 `onDeactivated`） | 别的页面按空格 → 计数器静默 +N（见 C4） |
| 8 | 贴屏边敲击被系统手势吃掉 | iOS/Android 边缘返回手势 | 敲了但不计数（或直接退出页面） |

#### D2 方案

**① 指针手势判定器（统一入口，替换所有 `@pointerdown` 计数）**

抽成纯函数模块 `frontend/src/utils/counterTapGuard.js`（可单测）：

```js
// 常量
export const TAP_MOVE_TOLERANCE_PX = 12   // 位移容差
export const TAP_MAX_DURATION_MS   = 1200 // 超过视为长按时长，不计数
export const TAP_DEBOUNCE_MS       = 60   // 同一次物理按压产生的重复事件去抖
export const LONG_PRESS_MS         = 600  // 减号 / 结束 / 解锁

export function shouldCommitTap({ start, end, lastCommitAt, now }) → boolean
```

判定规则：`end.isPrimary === true` && `end.button === 0`（或 pointerType==='touch'）&& `位移 ≤ 12px` && `时长 ≤ 1200ms` && `now - lastCommitAt ≥ 60ms`。

交互时序（关键取舍）：**视觉反馈在按下（CSS `:active` 已有的缩放/高亮），计数提交在抬起**。
- 备选方案 A「按下即计数，滑动时回滚」——**否决**：回滚会引入「计数闪一下又掉」的观感，且回滚期间若用户已经看到数字变化，等于撒谎；运动场景下准确性优先于 <100ms 的延迟。
- 备选方案 B「只用 click 事件」——**否决**：移动端 click 有 ~300ms 历史延迟与 300ms 双击缩放窗口（虽然全局 `touch-action: manipulation` 已压掉大部分），且 click 不提供位移信息。
- 采纳的 pointerdown+pointerup 方案延迟 ≈ 手指抬起瞬间（人体感知阈值以下），且能拿到位移/时长/多指信息。

**② 训练模式 = 全屏沉浸层（同时解决 A 的需求 4 与需求 1 的空间问题）**

- 新组件 `frontend/src/components/CounterTrainingOverlay.vue`（`views/life/*.vue` 的既有引用惯例：`../../components/X.vue`，见 `frontend/src/views/ai/MiniMaxStudioTool.vue:908`）
- `position: fixed; inset: 0; z-index: 900`（高于外壳 `.mobile-header` 的 `z-index:100`，**低于** Element Plus 会话框的 2000 层级），自带 `padding: env(safe-area-inset-*)`
- 结构：顶部信息条（计数 / 目标进度 / 已用时长 / 节奏）+ 中部**整块敲击面** + 底部「暂停/继续」「结束训练」（长按 700ms 或二次确认）
- 层内 `touch-action: none; overscroll-behavior: none; user-select: none` → 不可能产生滚动/缩放手势，从根上消掉误触来源 #1 的滑动分支
- **锁定态下如何仍然计数**：计数面本身**不加**任何长按/确认门槛，保持「按下→抬起」全速响应（这是准确性的前提）；锁定只作用于**退出/破坏性操作**（结束训练、解锁、暂停需要长按或二次确认），以及**隐藏/禁用**其它所有控件。锁定的语义是「锁 UI，不锁计数」。

**③ 训练期禁用 / 降级清单（需求 4 明确要求）**

| 控件 | 训练模式（全屏层内） | 理由 |
|------|-------------------|------|
| 主敲击面 | **保留**，全屏、无确认门槛 | 计数 |
| 结束训练 | 保留，**长按 700ms** + 二次确认「结束并保存本次训练？」 | 破坏性/会话边界 |
| 暂停 / 继续 | 保留，**长按 600ms** | 运动中单指长按比精准点小按钮容易，且不会误触 |
| 撤销（撤销上一次敲击） | **禁用**（层内不出现） | 运动中几乎必然是误触；改到「结束后总结面板」里按时间轴逐条修正 |
| 减号 | **禁用**（层内不出现） | 误触来源 #2 |
| 步长 chip | **禁用**（冻结当前 step，并显示「步长已锁定 +N」） | 误触来源 #3；改步长需先结束训练 |
| 自动连点 | **禁用**，进入训练时若开着则强制 `stopAuto()` + 提示 | 误触来源 #4；自动连点不是真实敲击，不应计入训练 |
| 形象 / 主题切换 | 禁用（层内不出现） | 训练中无关 |
| 日历 / 历史 / 设置入口 | 禁用（层内不出现） | 训练中无关，且弹层会遮挡 |
| 目标达成反馈 | **降级**：模态 → 层内横幅 + 震动脉冲（不阻塞敲击） | 误触来源 #6 |
| 音效 / 震动 | 保留（沿用现有开关） | 训练反馈核心 |
| 日统计（今日次数/周热力/连续达标） | 移动端隐藏；桌面端保留在层外 | 空间回收（A3） |

**④ 边缘手势规避**：训练层左右各留 `max(24px, env(safe-area-inset-left/right))`；敲击面有效区不贴边。

**⑤ 目标达成反馈改造**：训练模式内不使用 `el-dialog`（`CounterTool.vue:279-293`），改为层内顶部横幅 3s 自动消失 + `navigator.vibrate([30,60,30])`；**日间模式保持模态不变**（不破坏桌面端既有体验）。

**⑥ 撤销的重新定位**：保留现有 `undoLast()`（`:1168-1175`）语义不变；训练结束后在总结面板提供「敲击时间轴」（数据源：现有 `tapEvents`，`:727/1181/1187`，需在会话结束时快照一份，因为窗口只有 15s），允许删除某一条 → 同步 `todayCount--`（保持 §C1 的计数不变量）。

### R2 结论：本模块不需要后端

- 计数、速度、会话、目标全部可在客户端计算与存储（`localStorage`），符合 R2「客户端能算的不要走网络」
- 无跨用户、无分享、无协作需求；现有 `counter_v4_*` 已是纯本地
- 引入后端会带来：新表 + 新路由 + 鉴权 + 迁移，收益（跨设备同步）与当前单人单机使用场景不匹配
- 若未来要跨设备同步 → **单开 backend 子 issue**（复用现有 Askit 同步通道，见 `AGENTS.md` 模块 #27），本方案不扩范围

### 实施拆分（交下游的派工建议）

| 步骤 | 负责 | 产物 |
|------|------|------|
| 1 | frontend-writer | `frontend/src/utils/counterTapGuard.js` + `.test.js`（纯函数，先写测试） |
| 2 | frontend-writer | `frontend/src/composables/useCounterAudio.js` + `.test.js`（注入 fake ctx） |
| 3 | frontend-writer | `frontend/src/composables/useCounterSession.js` + `.test.js`（注入 now/storage） |
| 4 | frontend-writer | `frontend/src/components/CounterTrainingOverlay.vue` |
| 5 | frontend-writer | `CounterTool.vue` 接线（模板/脚本/样式） |
| 6 | lint-runner | 对改动文件跑 `npx eslint`/`node --check`（按该 agent 现有流程） |
| 7 | code-reviewer | R3（模板无 `.value`）/ R4（裸下标）/ 相对路径引用 等硬规则审计 |
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
| 回填/迁移 | **不需要**。没有老字段需要改写，没有历史数据需要重算 |
| 回滚 | 直接回退前端产物即可。会话数据成为孤儿键（不报错、占 ~40KB），可手动清；**不会**损坏 `_state`/`_records` |

### 行为（对现有用户的实际变化，需在发布说明里写清）

1. 计数提交从「按下」改为「抬起」——主观无感（<100ms），但偶发误计显著减少。
2. 新增 60ms 去抖：连续两次间隔 <60ms 的敲击会合并为一次（即 >16 次/秒的连击会被吞掉一次）。**取舍**：真实运动中 5–8 次/秒（俯卧撑/深蹲节奏）完全不受影响；自动连点最小间隔 120ms（`:707`）也不受影响。若 QA 实测认为有影响，可把常量降到 40ms。
3. 离开 `/counter` 后不再后台计数（原来是缺陷，属修复而非破坏）。
4. 移动端 ≤768px（原 720）进入移动布局，721–767px 的用户会看到布局变化（这正是修 bug）。
5. `dailyGoal`、历史记录、日历、周热力的显示与统计**零变化**。

---

## 风险与回滚

| # | 风险 | 影响 | 缓解 / 回滚 |
|---|------|------|------------|
| 1 | iOS 在**无用户手势**时可能拒绝 `resume()`（回前台那一刻不一定在手势里） | 极少数情况下回前台首击仍无声 | 降级阶梯第 2/3/4 层 + 「点我恢复」胶囊兜底；即便全失败也不影响计数。回滚：音频改动集中在 `useCounterAudio.js`，恢复旧内联函数即可 |
| 2 | 会话/后台计时在极端情况失真（系统杀掉进程、时钟被改） | 单条会话时长偏差 | `Math.min` 夹紧（`endedAt` 不得早于 `startedAt`）、`durationMs` 与 `activeMs` 双口径互相校验；进程被杀时按「最后一次落盘状态 + idle 收尾」处理，最坏丢最近 5s 的计数（敲击本身走 `saveState` 的既有链路，不受影响） |
| 3 | `CounterTool.vue` 继续膨胀（2806 → 预计 3200+ 行） | 维护性下降 | 抽出 3 个模块 + 1 个组件（见派工表），组件内只做接线 |
| 4 | 全屏训练层与外壳 z-index / 弹层冲突 | 弹层被遮或训练层被遮 | 训练层固定 `z-index: 900`（外壳 header 100 / EP 弹层 2000+），并对 EP 弹层做一次人工回归 |
| 5 | 断点从 720 改到 768 触发未知回归 | 平板竖屏（768）进入移动布局 | 768 与外壳 `isMobile` 本来就一致，改动方向是**消除**不一致；QA 需覆盖 768/769 两档 |
| 6 | localStorage 配额 | 写入失败 | try/catch + 丢弃最旧 50 条重试一次 + 静默放弃；绝不影响计数 |
| 7 | keep-alive 下 `onActivated`/`onDeactivated` 未覆盖所有路径（如 `currentViewKey` 变化导致重建） | 守卫失效 | `onMounted` 里同时置 `isActive=true`；`applyDelta` + `onKeyDown` 双重守卫；QA 用例覆盖「切走→切回」 |

---

## 测试要点（交 devtools-qa，可执行、可判定）

> 运行方式：`cd frontend && npx vitest run`（仓库已有 vitest，`frontend/package.json:11` → `"test": "vitest run"`；配置 `frontend/vitest.config.js`，environment=jsdom，include `src/**/*.test.js`）。
> **不要**起 dev server（R1）。L3 真机项由人工执行并回报。

### L1 纯函数单测（必须全绿）

**`counterTapGuard.test.js`**

1. 位移 0px、时长 200ms、`isPrimary=true` → `shouldCommitTap` 返回 `true`
2. 位移 20px → `false`（滑动不计数）
3. 时长 1500ms → `false`（长按不计数）
4. `isPrimary: false`（第二根手指）→ `false`
5. `button: 2`（右键）→ `false`
6. 距上次提交 30ms → `false`（60ms 去抖）
7. 距上次提交 120ms → `true`
8. 边界：位移恰好 12px → `true`；12.01px → `false`

**`useCounterSession.test.js`**（注入 `now` / 假 storage；`vi.useFakeTimers` + `vi.setSystemTime`）

9. `start()` → `end()`：`durationMs === endedAt - startedAt`
10. 暂停 60s 后继续：`activeMs === durationMs - 60000`，`pausedMs === 60000`
11. 后台 5 分钟（模拟 `hidden` → `visible`）：`backgroundMs === 300000`，`activeMs` 不含该区间
12. 后台超过 `SESSION_IDLE_MS`（30 分钟）：会话被自动收尾，`endReason === 'idle'`，`endedAt === max(lastHitAt, hiddenAt)`
13. **跨零点**：23:59 开始、00:00:30 检测到翻转 → 产出两条会话：第一条 `endedAt === 当日 23:59:59.999`、`endReason==='midnight'`、`startDate = 前一天`；第二条 `startedAt === 次日 00:00:00.000`、`splitFrom` 指向第一条；两条 `count` 之和 === 该时段 `todayCount` 增量
14. 撤销修正：会话内 `todayCount` 减 1 → `session.count` 同步减 1
15. `resetToday()` 触发：会话收尾且 `endReason === 'reset'`，`count` 不为负
16. **不变量**：任意操作序列后 `session.count === max(0, todayCount - countAtStart)`
17. 归一化容错：输入 `null` / `[]`（数组而非对象）/ `{version:99}` / 含缺失字段的会话 / `sessions` 是字符串 → 一律不抛异常，回退到合法结构
18. 容量：写入 250 条 → 保留最新 200 条，且按 `startedAt` 倒序
19. **红线回归**：会话模块的任何读写路径执行后，`localStorage` 中的 `counter_v4_records` 与 `counter_v4_state` 内容**逐字节不变**

**`useCounterAudio.test.js`**（注入 fake ctx）

20. 初始 `state==='suspended'` → `play()` 内部调用 `resume()`，且 `resume()` 的 Promise **resolve 之前不调用 `source.start()`**（核心回归：当前实现在这步就漏了）
21. `state==='interrupted'` → 仍然调用 `resume()`（核心回归：当前实现完全跳过）
22. `resume()` 挂起超过 `resumeTimeoutMs` → 不抛异常，跳过本次播放，连续 2 次后 `isDegraded === true`
23. `state==='closed'` → 新建 ctx 且 `soundBuffers` 被清空重建（断言两次 `createBuffer` 调用发生在不同 ctx 上）
24. `ctxFactory()` 抛异常 → `play()` 不抛（调用方 `applyDelta` 不中断）
25. `rebuild()` → 旧 ctx 的 `close()` 被调用，全局同时只存在 1 个 ctx

### L2 组件级 / 静态检查

26. `CounterTool.vue` 里不存在 `.value`（R3）、不存在裸下标 `x[0]`（R4）
27. 新增 import 全部为相对路径（**不得**出现 `@/xxx`，`@` 别名指向 `src/neon`）
28. `grep -n "onActivated\|onDeactivated" CounterTool.vue` 有命中（keep-alive 守卫已加）
29. `grep -rn "backend\|fetch(\|axios" ` 新增文件无命中（确认 0 网络请求）

### L3 真机 / 手工（需人工执行并回报结论）

> 设备矩阵：iOS Safari（含刘海机型）、Android Chrome；视口 320×568 / 375×667 / 390×844 / 414×896，横屏 844×390。

30. **音频（需求 2 核心）**：训练中锁屏 30s → 解锁 → **第一次敲击就有声**
31. 切到别的 App 5 分钟后回来 → 第一次敲击有声
32. 锁屏 5 分钟回来 → 第一次敲击有声（或 1 次无声后自动恢复，且出现「点我恢复」胶囊）
33. **布局（需求 1 核心）**：上列每个视口下——无横向滚动条；敲击面 + 「结束训练」按钮**无需滚动**即可全部看到；刘海/home indicator 不遮挡内容
34. 768px 与 769px 两档宽度切换窗口：布局切换点与外壳 header 出现/消失**同一时刻**（不再有 721–767 的裂缝）
35. 横屏 844×390：内容不重叠，安全区左右有留白
36. **防误触（需求 4 核心）**：在敲击面上做 10 次「按下后滑动 30px 再抬起」→ 计数**不变**
37. 双指同时按住敲击面 → 只 +1
38. 60 秒内快速敲击 200 次（约 3.3 次/秒）→ 计数**恰好 200**
39. 训练中长按「结束训练」<700ms → 不结束；≥700ms → 弹出二次确认
40. 训练中确认：减号/步长/自动/形象/主题/日历/历史/设置入口**均不可达**
41. 目标达成时不再弹模态 → 敲击不中断
42. **会话（需求 3 核心）**：开始训练 → 结束 → 历史里能看到该会话的**开始时间 / 结束时间 / 时长 / 次数 / 是否达标**
43. 会话跨零点（可改系统时间模拟）：历史里出现两段，日归属正确
44. 训练中切到别的工具页再切回 → 计时正确（不含离开时段，`backgroundMs` 正确），会话未丢失
45. 在别的工具页按空格 → 计数器**不再**增长（keep-alive 修复验证）
46. 训练中开着自动连点 → 被强制关闭且有提示
47. 回归：`todayCount` / 累计总数 / 连续达标 / 本周节奏 / 日历达标标记 / 历史记录的数值与改动前**完全一致**（同样的敲击序列，同样的结果）

---

## 未决问题（需 PM / jaxiu 裁决，不要由实现方拍板）

> 标注「阻塞」的事项若在动工前无答复，实现方按**给出的默认值**落地并在 PR 里注明，不挂起流水线。

1. **会话是否自动开始？**（**阻塞**）
   (a) 手动点「开始训练」进入全屏训练层【推荐】；(b) 当天首次敲击自动开始、N 分钟无操作自动结束；(c) 两者兼有（默认关）。
   默认：**(a)**。

2. **会话目标的默认口径？**（**阻塞**）
   (a) 默认 = 剩余日目标 `max(0, dailyGoal - todayCount)`【推荐】；(b) 固定预设（100/200/500）；(c) 每次开始都询问。
   默认：**(a)**，`dailyGoal=0` 时为 0（自由训练）。

3. **时长主口径？**（**阻塞**）
   详情主展示「实际时长」（墙钟，扣不扣后台）还是「有效时长」（扣后台与暂停）？
   默认：**主展示实际时长（墙钟，直接对应「实际开始/结束时间」的字面要求），有效时长作次要指标同时展示**。两个数都会存。

4. **静默多久自动结束会话？** 默认 **30 分钟**（`SESSION_IDLE_MS`，常量可调）。是否接受「回来后自动续跑，超过阈值才收尾」这一规则？

5. **是否需要「组」的概念？** 需求只提到「中途暂停」，本方案只做 暂停/继续（`pausedMs`）。若需要「第 N 组 / 组间休息」，数据模型要再扩一层（`sets: []`）——**这会显著加大改动量**，请明确是否需要。

6. **训练中是否隐藏日统计（今日次数 / 周热力）？** 移动端建议隐藏（空间回收 + 减少干扰），桌面端保留。默认：**移动端隐藏**。

7. **是否需要屏幕常亮（Screen Wake Lock API）？** 训练中防止自动息屏，可从源头避免需求 2 的场景。Chrome/Android 与 iOS Safari 16.4+ 支持；需在 `visibilitychange` 后重新申请。代价：耗电。默认：**不做**（需明确要求再加入）。

8. **结束后能否回溯修改单次敲击？**（**阻塞**）
   建议提供「敲击时间轴 + 删除某一条」，但删除会**同时扣减 `todayCount`**（保证 §C1 不变量）。是否接受「修改会话 = 修改当天日统计」？

9. **会话历史保留量**：默认 200 条 / 不按天裁剪。是否需要导出（CSV/JSON）？

10. **是否需要跨设备同步？**（若「是」→ **必须另开 backend 子 issue**，本方案不含）
    我的建议：**否**。理由见「R2 结论」。

11. **训练层是全屏覆盖（隐藏外壳 header）还是保留页头？**
    全屏覆盖（本方案）会让移动端失去 56px 的菜单按钮，训练结束后自动退出层、恢复页头。
    备选：给 `/counter` 路由加 `hideSidebar: true`（`App.vue:364` 已支持），但那样**桌面端也会失去侧边栏**且需要自建返回入口——**不推荐**。请确认接受「训练期间全屏、其余时间保留页头」。

12. **目标达成反馈**：训练内改横幅（不阻塞敲击）+ 震动，日间模式保持模态。是否接受两种模式行为不一致？

---

## 硬约束（下游无条件遵守）

- **R1**：只写 + review。不跑 `go run` / dev server / `docker compose up` / `./deploy.sh` / `npm run dev`
- **R2**：0 后端改动、0 网络请求（客户端能算的一律不联网）
- **R3**：Vue 3 模板里永远不写 `.value`
- **R4**：模板裸下标 `obj.x[0]` 一律写 `obj.x?.[0] || ''`
- **import 路径**：一律相对路径（`@` 别名指向 `src/neon`，不是 `src`）；`views/life/*.vue` → 组件用 `'../../components/X.vue'`，composable 用 `'../../composables/X'`（与 `CounterTool.vue:455` 现有写法一致）
- **R5**：本模块无 SQLite / `:memory:` 测试，不适用（但**不得**为此新增后端）
- **R6**：不适用（无分享资产、无 blob URL 上传）
- **R8**：本模块无长耗时 AI 调用；`resume()` 的 250ms 超时上限是同类「不能让 UI 无限等」思路
- **R9**：不涉及 MCP 协议版本
- **R10**：不涉及 Planner（**严禁**碰 `PLANNER_PHILOSOPHY.md` 相关模块）
- **R11**：不涉及后端产物编译
- **文件系统**：`/Volumes/M20` 是 SMB 网络挂载——不做任何构建产物落盘，临时文件一律 `/tmp`
- **写入边界**：本方案只落在 `docs/plans/`；`AGENTS.md` 的模块地图**不需要**改（计数器是纯前端视图，本就不在该表中，`AGENTS.md:7` 起的 42 行里无 counter 条目）。若 module-architect 认为应补一行，由其自行决定
- **不改 `App.vue`**：外壳属共享代码，本方案所有修复都在 `views/life/CounterTool.vue` + 新增模块内完成
