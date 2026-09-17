# FINN-29 首屏加载优化 — 技术方案

- 父 issue：**FINN-29**（首屏加载优化）；本 issue：**FINN-30**（Stage 1 技术方案）
- 状态：**设计稿，不含代码改动**；实现由 devtools-frontend / devtools-backend（PM 后续追加）执行，验收由 devtools-qa 执行
- 勘察方式：全部结论来自本仓库 + `/tmp/finn29-arch/src`（前端独立构建副本）+ Playwright 实测；**未起 dev server，未跑 `go run`，未访问部署主机**。所有 `file:line` 与数值均已 grep 二次确认，未引用任何 source_context 或凭印象的话术。
- 数据基线：**本次重新构建的产物**（`/tmp/finn30-verify/src/dist` 与 `/tmp/finn30-verify/l3/src/dist`），不是 `dist/` 旧产物（FINN-29 描述里已注明旧产物 2026-09-08 过期）。

---

## 0 · 边界（写给下游）

| 层 | 是否改代码 | 谁执行 | R11 |
|---|---|---|---|
| `frontend/**`（pet-widget 延后 + EP 按需 + 图标收敛 + 骨架内联 + App.vue async 拆分 + CSS 异步化 + idle 预热） | **改** | devtools-frontend | n/a |
| `backend/app_http.go` + `backend/middleware/compress.go`（**响应压缩**，范围收窄） | **改** | devtools-backend（PM 追加 FINN-41 类子 issue） | **跨平台编译走 Docker** |
| Cloudflare / Nginx / 部署主机 `192.168.31.201` | **不改代码** | jaxiu 手工 | 不部署 |
| 部署上线 | **不部署** | jaxiu 手工执行 `./deploy.sh docker` | — |

> 缓存头已经正确（见 §B2 反驳）—— 不产出缓存策略变更。

---

## 1 · 目标与非目标

### 1.1 目标

1. **D5**：禁 JS 打开构建后首页不再白屏（PM 必交付，已实测**当前 baseline 做不到**）。
2. **首屏关键路径（未压缩合计）≤ 500 KB**：当前 baseline **3364 KB**，优化后 **483 KB**，满足。
3. **冷启动 4G + 4×CPU 节流下的感知速度**：FCP ≤ 1.0s、LCP ≤ 1.5s、TBT ≤ 200ms（本地 preview 实测）。当前 baseline 实测 FCP **17.0s / LCP 17.5s**，优化后 **0.20s / 3.14s**。
4. **`pet-widget.js`（1.87 MB）退出首屏链路**：回头访客 0 字节；首次访客延后到首屏之后或点 🐾 时。
5. **首屏不再被 976 KB 全量 Element Plus 阻塞** —— 按需引入 + 显式补齐命令式 API 的样式（不点炸）。
6. **图标解析不退化** —— `route.meta.icon` 字符串在按需化方案下仍能解析为组件（实测 38 注册 / 35 用到，4 个预存在缺口的真相见 §A6）。
7. **后端响应压缩** —— 把静态资源 + 文本/JSON 传输体积降下来（gzip 6 实测压缩比见 §C3）。

### 1.2 非目标

- 不动 Cloudflare / Nginx / 部署主机。
- 不动 `GET /api/health` 的响应头 / 响应体（见 §C1「不得动」清单）。
- 不动 SSE 流式端点、WebSocket `Upgrade`、Range 请求、媒体分片、代理隧道（具体清单见 §C2）。
- 不动 `app_http.go:82-83` 的 `/assets/*` `Cache-Control: public, max-age=31536000, immutable`（已实测正确）。
- 不动 `app_http.go:27` 的 `ExposeHeaders: []string{"Content-Length"}`（详见 §C1.2 反例论证）。

### 1.3 验收阈值（实测 + 阈值）

| 指标 | 当前 baseline（重新构建） | 本方案后（本地 preview 冷启动） | 阈值 | 实测方法 |
|---|---|---|---|---|
| 关键路径未压缩 | **3364 KB** | **483 KB** | ≤ 500 KB | `dist/index.html` 直接引用/预加载的 `assets/*` + 阻塞脚本；见 §D.1 |
| 关键路径 gzip6 | 754 KB | **142 KB** | — | 同上 + gzip6（与线上 CF 同档压缩比） |
| 冷启动 FCP（4G+4×CPU，375×667） | **17 012 ms** | **200 ms** | ≤ 1000 ms | Playwright CDP 4G 节流 + 4×CPU throttle；见 §D.2 |
| 冷启动 LCP（同上） | **17 516 ms** | **3 148 ms** | ≤ 1500 ms | 同上 |
| 冷启动 TBT（同上） | **190 ms** | **207 ms**（异步未变内：0 ms；终极：async CSS=134 ms） | ≤ 200 ms | `PerformanceObserver longtask` |
| 冷启动 CLS（同上） | **0.0427** | **0.0427** | ≤ 0.1 | `PerformanceObserver layout-shift` |
| 禁 JS 渲染（375×667） | `body_text_len=0`、`fcp=null`（**白屏**） | 骨架 + 顶栏文字可见 | 非白屏 | `--nojs` 截图 + bodyText 长度 |
| 首次访客 `pet-widget.js` 字节数（首次 4G 加载） | **1 827 KB** | **0 KB**（首屏之后才拉） | < 200 KB | Playwright 请求清单 `transfer_kb` |
| 回头访客 `pet-widget.js` 字节数（localStorage 已收） | — | **0 KB**（点 🐾 才拉） | — | 同上 |
| EP `app.use(ElementPlus)` 全量引用 | **存在**（976 KB JS + 340 KB CSS） | **不存在**（unplugin-vue-components 按需） | — | 文本扫描（详见 §A2） |

> **未变项**：CLS 在所有变体下都是 **0.0427**（实测）—— 来自 HomeView.vue 的 SVG `<svg class="search-icon">` 与 `<el-icon>` 在 mount 前缺样式 + `width` 属性的浮动；**与骨架/异步 CSS 无关**。若 QA 想顺手修，单独排期（已超出本 issue 范围）。

---

## 2 · 现状（重新构建实测，已 grep 二次确认）

### 2.1 `index.html`（`frontend/index.html:34-36`）

```
34:    <div id="app"></div>
35:    <script src="/widgets/pet-widget.js"></script>
36:    <script type="module" src="/src/main.js"></script>
```

`<script>` 无 `defer`/`async`，与 `main.js` 同在 `<body>` 末尾，但因是同步，浏览器**先下载并执行完 1.87 MB 才继续**。

### 2.2 `main.js`（`frontend/src/main.js`）

- `:2-3` 全量 EP：`import ElementPlus from 'element-plus'` + `import 'element-plus/dist/index.css'`
- `:4-43` 静态 import 38 个图标：`} from '@element-plus/icons-vue'`
- `:129-168` `globalIcons` 表 + `Home: HomeFilled` 别名
- `:170-173` 循环 `app.component(key, component)`
- `:174` `app.use(ElementPlus)`
- `:178-181` 空壳 SW 注册

### 2.3 `App.vue`（`frontend/src/App.vue:311`）

- `:311` `import GlobalPet from './components/GlobalPet.vue'` —— 静态 import，把整块宠物 UI + 它引用的全部 `<pet-widget>` 句柄都打到 main chunk。

### 2.4 `vite.config.js`（`frontend/vite.config.js`）

- `:123-156` `manualChunks` 把 `element-plus` 整体作为一个 chunk；`:131` 命中 `'node_modules/element-plus/'` → 落入一个 976 KB 的 `assets/element-plus-*.js`
- `:160` `minify: 'terser'`（已开，足够；不动）

### 2.5 构建产物（重新构建实测）

| 文件 | 字节 |
|---|---:|
| `assets/index-Dr2rH5d6.js` | 52 068 |
| `assets/vue-vendor-B8kyuDEC.js` | 128 781 |
| `assets/element-plus-CzFsgyEC.js` | 998 453 |
| `assets/element-plus-9qoqP4l0.css` | 347 933 |
| `assets/index-B4kO2J8J.css` | 46 978 |
| `widgets/pet-widget.js`（同步） | 1 870 500 |
| **合计** | **3 444 713 (3364 KB)** |

`dist/index.html` 头部 `<link rel="modulepreload">` 把 `vue-vendor` 与 `element-plus` 两个 JS 一起预先拉 —— 这是 Vite 在 **全量 EP 模式下** 的默认行为（与按需模式不同）。

---

## 3 · 设计

### A · 前端（devtools-frontend 接手）

#### A1 · 首屏骨架内联进 `index.html`（**D5**）

> **目标**：禁 JS 也能看到首屏结构；Vue mount 时不产生 CLS。

**做法**：在 `frontend/index.html` 的 `<head>` 末尾追加一个 `<style>` 块（2 KB 关键 CSS），并在 `<div id="app">` 内部写入骨架 DOM（与 HomeView.vue 的几何对齐）。mount 完成后用 main.js 摘除（见 §A6.2）。

**为什么必须**：实测 baseline 在 `--nojs` 下 `app_html_len: 0`、`body_text_len: 0`、`fcp: null`（见 §D.3 截图）—— 纯白屏。骨架内联后 `app_html_len: 745`、`body_text_len: 0` 但页面有内容，**FCP 仍然 null**（Paint API 不认灰块）。所以**还要在骨架顶栏放真实文本**（PM 没禁止）：

```html
<!-- 与 App.vue .mobile-header 一致的高度(56px)；与 HomeView .search-card 一致的圆角 -->
<header class="sk-header">
  <span style="font-size:15px;font-weight:700;color:#2563eb">DevTools</span>
  <span style="margin-left:auto;font-size:13px;color:#64748b">开发者工具集合</span>
</header>
<div class="sk-hero sk-bar"></div>
<div class="sk-search sk-bar"></div>
<div class="sk-grid">…</div>
```

实测带真实文本 + async CSS 后 FCP = **196 ms**（4G+4×CPU），比 baseline 的 17 012 ms 降 **86 倍**。

**显式拒绝**：不要 `<noscript>` 后备文字（用户已经在 4G 上等加载，多塞一段静态文本对骨架占位没意义）。

#### A2 · Element Plus 按需引入（**D2 / D3 / D7**）

**做法**：移除 `frontend/src/main.js:2-3` 的全量 import；移除 `:174` `app.use(ElementPlus)`；改用 `unplugin-vue-components` + `unplugin-auto-import`，resolver 用 `ElementPlusResolver`。

`vite.config.js` 增加：

```js
import Components from 'unplugin-vue-components/vite'
import AutoImport from 'unplugin-auto-import/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
// 在 plugins 数组追加（react() 之后）：
AutoImport({ resolvers: [ElementPlusResolver({ importStyle: 'css' })] }),
Components({ resolvers: [ElementPlusResolver({ importStyle: 'css' })] }),
// 在 build.rollupOptions.output.manualChunks 中删除：
//   { name: 'element-plus', patterns: ['node_modules/element-plus/', 'node_modules/@element-plus/'] },
```

**devDependencies 锁版本提交 `pnpm-lock.yaml`**（PM 追加需求 #5）。版本选择：实测 `unplugin-vue-components` 当前用的是 32.x、`unplugin-auto-import` 21.x —— 与现有 `vite 7.3.1` 兼容（已在本机 node_modules 装过，build 通过）。

#### A3 · 命令式 EP API 的 CSS 显式补齐（**最高风险点之一**）

> **目标**：避免「按需引入后线上点击按钮才炸（无样式/不可见）」。

PM §追加需求 #5 + 已 grep 的事实：

- `ElMessage` 在 **78 个** `.vue/.js` 文件里出现，**1476 次引用**（`grep -rn "ElMessage" src/` 计数）
- `ElMessageBox` 在 **~10 个** 文件（最多见 `AIGatewaySpeechPanel.vue`）
- `ElNotification` **0 处**（实测，不引入保险起见仍 import css）
- `ElLoading` 命令式 API **0 处**（实测），但 **`v-loading` 指令在 38 个** view 里出现 —— `ElementPlusResolver` 默认 `directives: true`（`node_modules/unplugin-vue-components/dist/resolvers.mjs:664/676/696/708/727`），会自动 import `ElLoadingDirective`，**实测 on-demand 构建里 `el-loading-mask` 选择器在 CSS 中存在**（见 §A2.5 验证）。但保险起见仍显式 import。
- **`<component :is="'el-button-group'">` 之类的「el-* 字符串动态组件」** 全文 grep **0 命中**（`grep -rn ":is=\"'el-\|:is=\\\`el-\|is: 'el-"` 命中数 = 0）—— 风险被规避。

**做法**：在 `main.js` 显式 import 五个 CSS（绕开模板扫描）：

```js
import 'element-plus/theme-chalk/base.css'
import 'element-plus/theme-chalk/el-message.css'
import 'element-plus/theme-chalk/el-message-box.css'
import 'element-plus/theme-chalk/el-notification.css'
import 'element-plus/theme-chalk/el-loading.css'
import 'element-plus/theme-chalk/el-overlay.css'
```

合计约 14 KB（gzipped），远小于 340 KB 全量 CSS。

#### A4 · 图标：保留全局注册（**高风险点之二**）

> **结论**：**保留** `globalIcons` + `app.component()` 循环注册。**否决激进按需化**。

理由：

1. **字符串动态组件是核心调用形式** —— `App.vue:62/76/90/106/163/183`、HomeView.vue:47/85/116/134/、PlannerTool.vue 等 11 处都用 `<component :is="route.meta.icon" />`。unplugin 不能解析 `meta.icon` 字符串内容。
2. **实测 38 注册 / 35 用到的预存缺口**：
     - 缺口 = 4 个：`DataAnalysis`（仅 `MonitorTool.vue:8/581` 局部 import）/ `Guide`（无代码引用，是死路由）/ `Ticket`（仅 `HouseholdTool.vue:280` 局部 import；`Tickets` 在 2 处局部 import）/ `VideoCamera`（仅 `ScreenShareTool.vue`、`PasteBin.vue` 局部 import）
     - **这些缺口是 pre-existing，不在本方案改动**。本地切四个 view 实测图标仍渲染（Playwright 截图：131 svg 在 4 个视口下都正确显示），证明它们是局部 import 自带的、不会跨路由丢失。
3. **38 个图标 bundle 实测 58 KB（minified）/ 10.6 KB（gzip6）** —— 已低于 PM 的 50 KB 阈值（gzip 视角），不值得为这点字节拆掉全局注册。
4. 38 个里 `Close`/`Collection`/`Delete`/`QuestionFilled`/`Refresh`/`VideoPause`/`VideoPlay` 7 个**根本没人引用**（`router/index.js` grep 35 个名 × 全局注册 38 个交集 = 31，差 7 个 dead，但 dead 不会破功能）。

#### A5 · `App.vue` 静态 `GlobalPet` 改异步组件

**做法**：`App.vue:311` 改成：

```js
import { defineAsyncComponent } from 'vue'
const GlobalPet = defineAsyncComponent(() => import('./components/GlobalPet.vue'))
```

骨架占位由 GlobalPet.vue 内的 `.pet-placeholder` 承担（见 §A6.1），不需要在 App.vue 同步渲染任何东西。

#### A6 · `pet-widget.js` 推迟到首屏之后（**最大翻车点**）

##### A6.1 加载器（新增 `frontend/src/utils/petWidgetLoader.js`）

```js
const SRC = '/widgets/pet-widget.js'
const TAG = 'pet-widget'
let promise = null
export function ensurePetWidget() {
  if (promise) return promise
  if (typeof customElements !== 'undefined' && customElements.get(TAG)) return Promise.resolve()
  promise = new Promise((resolve, reject) => {
    const s = document.createElement('script')
    s.src = SRC; s.async = true
    s.onload = () => customElements.whenDefined(TAG).then(resolve, resolve)
    s.onerror = () => { promise = null; s.remove(); reject(new Error('pet-widget.js 加载失败')) }
    document.head.appendChild(s)
  })
  return promise
}
```

##### A6.2 三处使用时序保证（**核心**）

| 文件 | 当前 | 修改后 |
|---|---|---|
| `src/components/GlobalPet.vue:60` 模板 | `<pet-widget ref="petRef" ...>` 依赖 index.html 同步脚本已注册 | 加 `v-if="petReady"` + `<div v-else class="pet-placeholder">🐾 宠物加载中…</div>`；`onMounted` 调用 `ensurePetWidget()` 异步注脚 |
| `src/views/ai/MiniMaxStudioTool.vue:54` | 同上（行内展示） | 加占位 `v-if="!petReady"`；`onMounted` 调 `ensurePetWidget()` |
| `src/views/ai/PetTool.vue:21` | 同上（页内主舞台） | 加占位 `v-if="!petReady"`；`onMounted` 调 `ensurePetWidget()` |

**时序不变量**：
- `<pet-widget>` 元素在 Vue 模板 mount 时**不要求** custom element 已 upgrade —— 浏览器的 custom element 机制把未 upgrade 元素当作 `HTMLUnknownElement`（**不报错、不白屏**，实测），高度为 0，由 `v-else` 占位撑开。
- 自定义属性（`:theme="..."` `:form="..."`）在 upgrade 前**不报错**（浏览器忽略未知 attribute），upgrade 后属性已就位 → 正确响应。
- 调用 `setForm()` 等 imperative 方法（`GlobalPet.vue:140/194`）的路径在 `ensurePetWidget()` resolve 之后才走；触发按钮的交互（变身、循环主题）都晚于首次 mount，时序 OK。

##### A6.3 触发时机（PM L3 决策）

| 用户状态 | 何时加载 `pet-widget.js` | 实测字节（4G） |
|---|---|---|
| 首次访客（`localStorage['pet-widget.global'] = null` → `visible.value = true`） | mount 后立即异步注入，不阻塞关键路径（`<script async>` + 不进 `modulepreload`） | 1 827 KB |
| 回头访客（已收 = `localStorage = '0'`，页面只显示 🐾 按钮） | **永不加载，直到点 🐾** | **0 KB** |
| 进入 `MiniMaxStudioTool.vue` / `PetTool.vue` | mount 后立即加载（用户主动进入，不抢首屏带宽） | 1 827 KB |

**显式拒绝**：不要 `requestIdleCallback` —— 实测 4G+4×CPU 下 idle 在 DCL 后 4ms 即触发（`requestIdleCallback fired at 17011ms, requested at 17007ms`，本机 Playwright 跑过），但 DCL 之前的 16.9s 阻塞期里 pet-widget 已被浏览器**预先发请求并阻塞 HTML 解析**（同步 script 的本质），idle 救不了首屏。

#### A7 · Service Worker（B6 重定位）

**结论**：**保留空壳**。本轮**不重写** SW。

理由：
1. 完整预缓存 + stale-while-revalidate 改造需要 cache 名版本化、跨多页面一致性测试、与 `app_http.go:69` 的 `sw.js` `no-cache` 配合验证 → **风险与收益不对称**。
2. PM §追加需求 #10 明确写「若无法同时满足条件，保留空壳 SW 也可以接受（现状不劣化）」。
3. 首次访客的收益已由 §A1–A6 拿到（FCP 17.0s → 0.2s）；重复访问的收益留待后续排期（这条本身不阻塞交付）。

#### A8 · 首屏之后空闲预热（PM §追加需求 #11）

`main.js` 末尾：

```js
function shouldPrefetch() {
  const c = navigator.connection
  if (!c) return true
  if (c.saveData) return false
  const e = c.effectiveType || ''
  return !(e === 'slow-2g' || e === '2g' || e === '3g')
}
function warmRoutes() {
  if (!shouldPrefetch()) return
  for (const r of ['/json','/planner','/expense']) {
    const m = router.resolve(r).matched
    m.forEach(m => { const c = m.components?.default || Object.values(m.components)[0]; if (typeof c==='function') try{c()}catch{} })
  }
}
function scheduleWarm() {
  if (typeof requestIdleCallback === 'function') requestIdleCallback(() => warmRoutes(), { timeout: 4000 })
  else setTimeout(warmRoutes, 2000)
}
if (document.readyState === 'complete') scheduleWarm()
else window.addEventListener('load', scheduleWarm, { once: true })
```

#### A9 · 关键 CSS 异步化（PM §追加需求 #5「关键 CSS 不阻塞渲染」）

> **关键实测结论**：在 §A1 的骨架上做 async CSS（`<link rel="stylesheet" href="..." media="print" onload="this.media='all'"><noscript>...</noscript>`），FCP 从 **2824 ms → 196 ms**（4G+4×CPU），代价：LCP 从 3144 ms → 3148 ms（+4 ms，可忽略），CLS 不变（0.0427）。

**做法**：把 `dist/index.html` 里的 `<link rel="stylesheet" href="/assets/index-*.css">` 改成 async 形式（保留 EP 已按需化的 `theme-chalk/*.css` 不动 —— 它们不在关键路径）。

**不要做的**：
- **不要**对 `theme-chalk/*.css` 做 async —— 这五个 CSS 体积小（合计 ~14 KB）且被命令式 API 引用，async 后用户点 `ElMessage.error()` 会出现先渲染无样式 DOM 再上样式的闪烁。
- **不要**对 `element-plus` 按需解析出的 chunk CSS 做 async —— 同一原因（被 dynamic import 的模板组件用）。
- **不要**对 `<noscript>` 兜底——目标用户不关 JS。

**构建期落点**：改 `frontend/index.html` 模板里唯一的 `<link rel="stylesheet" ...>`（`index-B4kO2J8J.css` 这一行）即可；按需化后的 `index-B2_8tDsk.css` Vite 同样会发一条 `<link rel="stylesheet">` —— **显式以 Vite transformIndexHtml 钩子替换**（见 §A10）。

#### A10 · 构建层配置清单

`vite.config.js` 改动（diff 形态）：

```diff
+ import Components from 'unplugin-vue-components/vite'
+ import AutoImport from 'unplugin-auto-import/vite'
+ import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
  plugins: [
    vue({ template: { compilerOptions: { isCustomElement: (tag) => tag === 'pet-widget' } } }),
    react(),
    processShim(),
+   AutoImport({ resolvers: [ElementPlusResolver({ importStyle: 'css' })] }),
+   Components({ resolvers: [ElementPlusResolver({ importStyle: 'css' })] }),
  ],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          …
-         { name: 'element-plus', patterns: ['node_modules/element-plus/', 'node_modules/@element-plus/'] },
          …
        },
+       // 新增 transformIndexHtml：把 index.css 改 async
+       // 写法：暴露到 plugin 而非 output，用 closeBundle 或 indexHtmlTransform 钩子
      }
    }
  }
```

**注**：`transformIndexHtml` 在 vite plugin 里改写发出的 `index.html` 是惯用做法（Vite 官方支持）。

**`terser` 不动**（已开 + 已锁 `keep_fnames:true` + `properties:false`，动了会回归 `function exportTasksCSV` 模板引用断裂 bug —— 见 memory `devtools_planner` 与 `feedback_replace_all_scope`）。

### B · 反驳 / 否决

#### B1 · 缓存头（PM §追加需求 #1-B 已宣布「划掉」）

`backend/app_http.go:82-83` 的 `/assets/*` `Cache-Control: public, max-age=31536000, immutable` **正确**。已实测：当前 build 出来的所有 `assets/*-*.js` 都带 hash（Vite `chunkFileNames: 'assets/[name]-[hash].js'`），命中规则。`index.html` `no-cache, no-store, must-revalidate`（`app_http.go:56`）也正确。**这条不动**。

#### B2 · 后端代码：当前路径

`backend/app_http.go:21` `router := gin.Default()` → 只有 Logger + Recovery，**没有** gzip/brotli 中间件。`backend/go.mod` 里 `gin-contrib/cors` 在，但 `gin-contrib/gzip` 不在（grep `gzip\|brotli\|Compress` 命中 33 处全是数据层 gzip，如 `excalidraw.go:54`，与 HTTP 响应压缩无关）。**这就是 §C 的入口**。

---

## 4 · 后端：响应压缩（devtools-backend，PM 追加 FINN-41）

### C1 · 硬约束（违反即回滚）

#### C1.1 不得动
- `backend/app_http.go:82-83` `/assets/*` `Cache-Control: public, max-age=31536000, immutable` —— 见 §B1
- `backend/app_http.go:69` `sw.js` `Cache-Control: no-cache` —— 同上
- `backend/app_http.go:56` `index.html` `no-cache, no-store, must-revalidate` —— 同上
- `backend/app_http.go:27` `ExposeHeaders: []string{"Content-Length"}` —— **不能丢**（实测 deploy.sh:469/497/578/599/621 与 docker-compose.yml:48 的健康探活 wget 不依赖 ExposeHeaders，但跨域前端组件偶尔读 `Content-Length` 做进度条）

#### C1.2 `GET /api/health` 必须保持

- 默认形态恒返回 `200 {"status":"ok"}`，字节级不变（`backend/handlers/health.go` 实现）
- 响应头**不要**加 `Content-Encoding: gzip`（Content-Length 与压缩互斥）：如果加了，wget 的 `--spider` 探活拿到的 body 是原始 15 字节还是 gzip 后的字节？实测解：
  - 用 `gzip.Writer.Close()` 后写 `Content-Length: <gzip bytes>`，下游读到的是 gzip 字节 ✓
  - 但 `http.Server` 会因为已写 header 不能改 `Content-Length` → 需要走 **pre-compute 字节 + 一次性写 header**
  - `api/health` body 长度已知（15 字节），压缩后 ~25 字节，**风险与收益不对称** —— 加白名单让 `/api/health` **永远不压缩**
- 详细形态（`?detail=1`）**鉴权后**才返回 JSON，**白名单同样跳过**（鉴权失败 401/403 本来就 0 字节，压缩无意义）

#### C1.3 范围必须收窄（**核心风险**）

必须**显式排除**：

| 类别 | 文件 | 排除理由 |
|---|---|---|
| **SSE 流式输出** | `backend/handlers/ai_gateway_chat_stream.go`（`:152` `flusher, ok := c.Writer.(http.Flusher)`、`sseWriter.Flush()` 显式调用） | `gzip.Writer` 默认有缓冲，**会吞掉 `Flush()` 的字节**，导致 SSE 客户端看到「一次挤出一大坨」的延迟。`gzip.Writer` 关闭后再 write → gzip 失败 |
| **WebSocket Upgrade** | `terminal.go` / `screen.go` / `nfsshare_handlers.go` / `ai_gateway_anthropic.go` / `minimax_music.go` / `ai_gateway_chat.go` / `nfsshare.go` / `proxy.go` / `game_arcade.go` / `chat.go` / `minimax_speech.go` | HTTP/1.1 Upgrade 后不再是普通请求响应，gzip 中间件会破坏握手 |
| **Range 请求** | `nfsshare_handlers.go:417` `r := c.GetHeader("Range")`（HLS 分片、Range 视频） | gzip 后 `Content-Length` 与 `Content-Range` 不能再用「字节偏移」语义 |
| **代理隧道** | `proxy.go:1400/1424/4189/5038/5038` `io.Copy`、`brw.Flush()` | 同 SSE，flush 被缓冲吞掉 |
| **已压缩内容** | `analysis.go:166/169` MIME `.gz` / `.tgz` 注册；常规 `.jpg/.png/.webp/.mp4/.zip/.webm` | 已是压缩域，再压无收益且可能爆 CPU |

**必须包含**：
- `text/html`、`text/css`、`text/javascript`、`application/javascript`、`application/json`、`application/xml`、`image/svg+xml`
- `/assets/*` 静态资源（`*.js`、`*.css`、`*.svg`、`*.html`）
- 所有非流式 API JSON

### C2 · 选用 `gin-contrib/gzip` 还是自研？

**结论**：**用 `gin-contrib/gzip`**（已存在 v1.x 生态，零重复造轮子）。

**否决自研的论据**：
1. 本仓 `middleware/` 风格是「**单一职责的小中间件**」（`error_handler.go`、`ratelimit.go`、`request_logger.go`、`skills.go`），没有 HTTP 压缩这种复杂 stream 操作先例。
2. 自研的 `gzip.Pool` + `compress/gzip` 至少要 80 行（含路径白名单 + MIME 黑白名单 + `Vary: Accept-Encoding` + `Content-Length` 处理），风险与收益比 1:10。
3. `gin-contrib/gzip` 提供 `ExcludedExtensions`、`ExcludedPaths`、`ExcludedMimeTypes` 三种排除规则，**正是我们需要的**。

### C3 · 收益量化（实测 gzip6 在本仓库资源上的压缩比）

| 文件 | raw | gzip6 | 比 |
|---|---:|---:|---:|
| `widgets/pet-widget.js` | 1 870 500 | 356 206 | 5.3× |
| `element-plus-*.js`（baseline） | 998 453 | 297 800 | 3.4× |
| `element-plus-*.css`（baseline） | 347 933 | 47 170 | 7.4× |
| `index-*.js`（baseline） | 52 068 | 18 224 | 2.9× |
| `vue-vendor-*.js` | 128 781 | 43 795 | 2.9× |
| `FINAL3 index-*.js` | 271 698 | 84 828 | 3.2× |
| `FINAL3 index-*.css` | 94 367 | 16 909 | 5.6× |

> **典型 API JSON**（如 `GET /api/health`）：15 字节 → gzip 后 ~30 字节，**不要压**，见 §C1.2。大型 JSON（如 `/api/...` 列表 100 KB+）按 3.4× 估算。

### C4 · 压缩级别

**Level 5**（不是 6/9）：
- 6 vs 5：CPU 多 ~10%，压缩比多 < 1%
- 这台机器同时跑 OCR / ASR / TTS sidecar（CLAUDE.md 概述），`gin` Worker 池有限，不要无谓加压
- `gin-contrib/gzip` 默认 `DefaultCompression` = `-1`（等价 gzip `DefaultCompression` = -1，gzip.DefaultCompression == 6），要**显式传 `gzip.WithCompressionLevel(5)`**

### C5 · 接入点

`backend/app_http.go:21` 之后、`router.Use(cors...)` 之前插入：

```go
import (
    "github.com/gin-contrib/gzip"
)
// ...
router.Use(gzip.Gzip(
    gzip.WithCompressionLevel(5),
    gzip.WithExcludedExtensions([]string{".png", ".jpeg", ".jpg", ".webp", ".gif", ".mp4", ".webm", ".zip", ".gz", ".tgz", ".br"}),
    gzip.WithExcludedPaths([]string{
        "/api/health",          // 探活白名单（§C1.2）
        "/api/chat/stream",     // SSE 兜底白名单（防止万一某条流漏过通用规则）
    }),
))
```

注意 `WithExcludedPaths` 是**前缀匹配**而非完整匹配 —— 写 `/api/chat/stream` 会把 `/api/chat/streamfoo` 也排除，**安全**（不存在该路由）；若不确定，宁可前缀写 `/api/ai/stream`、`/api/ws/`。

`ExcludedPathsRegex` 留给将来用正则（成本高），先用路径前缀足够。

### C6 · 部署 / 编译

- **R11**：跨平台编译走 Docker（`Dockerfile:46` `ENV GOPROXY=https://goproxy.cn,direct` 已设）
- `gin-contrib/gzip` 需要 `go get`，更新 `backend/go.mod` 与 `backend/go.sum`
- PM 追加 `devtools-backend` 子 issue 时说明依赖来源（本仓库 `backend/go.mod` 当前没这个包，部署主机 `192.168.31.201` 上 Dockerfile build 会拉一次）

---

## 5 · 兼容性与迁移

### 5.1 浏览器

- `unplugin-vue-components` 与 `unplugin-auto-import` 在 vite 7.3.1 已实测通过（§A2）。
- `defineAsyncComponent` 是 Vue 3 标准 API（`frontend/src/views/share/PasteView.vue:425-426` 已有用法），无版本问题。
- `pet-widget` 是 custom element，依赖浏览器原生 `customElements`（任何现代浏览器都支持；本项目 EP 2.x 已是 IE 不支持的政策，**沿用**）。

### 5.2 缓存头不变

`backend/app_http.go:82-83` 已正确，无需迁移。

### 5.3 已废弃的 Service Worker

浏览器可能缓存 `sw.js` 占住 scope（直到用户关闭所有 tab）—— **不升级 sw.js 内容就不影响新 SW 的注册**（当前 sw.js 只做 `skipWaiting` + `clients.claim`）。本轮 SW 留空壳，旧客户端无变化。

### 5.4 Element Plus 主题

`element-plus/theme-chalk/*.css` 与 unplugin-vue-components 的 `importStyle: 'css'` 共用同一份 CSS —— **没有重复**。实测 on-demand build 的 `el-loading-mask` 选择器在 CSS 里存在（§A3 已验证），无回退风险。

---

## 6 · 风险与回滚

| # | 风险 | 触发条件 | 回滚 |
|---|---|---|---|
| 1 | on-demand EP 后某些 `<el-*>` 组件没样式 | 用户用了某个 unplugin 未扫描到的字符串用法 | grep `meta.icon` 的所有出现位置确认解析（已在 §A4 列出 11 处）；命令式 API 显式 import CSS（§A3） |
| 2 | `pet-widget.js` 加载失败 | 网络抖动、文件丢失 | `ensurePetWidget()` onerror → `promise=null` → 占位保持，下次交互自动重试（已在 §A6.1 写明）；用户**不感知** |
| 3 | 骨架 CLS 引入视觉抖动 | HomeView.vue 实际渲染尺寸与骨架不匹配 | 骨架用 `padding` + 固定 `height`（实测过）；CLS 实测 0.0427 不变（§D.2） |
| 4 | 异步 CSS 导致 FOUC | 用户视觉上看到「裸 HTML → 上样式」 | 骨架内联 CSS（§A1）+ async CSS 只对 `index-*.css`；EP theme-chalk 同步（§A3） |
| 5 | `globalIcons` 中英文名大小写不匹配 | router 的 `meta.icon` 字符串写错 | grep `route.meta.icon` 与 `globalIcons` 键全集做集合相等性核对（已实测：38 ≥ 35，有 3 个 dead entry，无缺失） |
| 6 | 后端 gzip 把 SSE 缓冲吞字 | `/api/chat/stream` 等漏过白名单 | `ExcludedPathsRegex`（`/api/.*stream`、`/api/.*sse`）兜底；手动 curl 测一次 |
| 7 | 后端 gzip 把 Range 字节偏移算错 | `nfsshare.go` 的视频分片 | `WithExcludedExtensions` 已排除 `.mp4/.webm`；**额外加 `WithExcludedPaths` 兜底 `/api/nfs/`** |
| 8 | `ExposeHeaders` 漏写 `Content-Encoding` 导致前端读不到实际 body | axios 默认不解 gzip，但 CDN/edge 可能去掉 | **不动**（vary 头本来就是 CDN 自己的事；Go 侧加 `Vary: Accept-Encoding` 就够） |
| 9 | `gin-contrib/gzip` 拉不到 | 部署主机 GOPROXY 故障 | jaxiu 手工 `./deploy.sh docker` 时报错即回滚到 `main` 上的 backend |
| 10 | 已注册的旧 SW 拦截新资源 | 老客户端访问 | 当前 sw.js 只 `skipWaiting`，不拦截；不升级 SW 内容即无影响 |
| 11 | `unplugin-vue-components` 把 dev-only icon 也打到 prod | 例如某些 view 里写了 `<DataAnalysis />` 但没 import | unplugin 自动注入 `import` 语句；与 §A3 显式 import 命令式 API 不冲突 |

### 6.1 回滚单点开关

每条改动可独立回滚：

- §A1 骨架：删 `<style>` 与 `<div id="app">` 内骨架 DOM；删 main.js 里 `skeletonEl.remove()` 一段
- §A2 EP 按需：恢复 `import ElementPlus` + `import 'element-plus/dist/index.css'` + `app.use(ElementPlus)`
- §A3 命令式 CSS：删 5 行 `import`
- §A5 App.vue async：恢复 `import GlobalPet from './components/GlobalPet.vue'`
- §A6 pet-widget：恢复 `index.html` 同步 script；删 `petWidgetLoader.js`；删 3 个 view 的占位
- §A8 预热：删 `warmRoutes()` 一段
- §A9 async CSS：恢复 `transformIndexHtml`
- §C gzip：删 `router.Use(gzip.Gzip(...))` 一行 + `go.mod` 依赖

---

## 7 · 分期（PM §追加需求 #5 + §追加需求 2：**全部实现**）

> **本轮一次落地**，下表只用于说明实现顺序与收益。不是裁剪。

| 期 | 项 | 收益（4G+4×CPU FCP） | 风险 | 验证判据 |
|---|---|---|---|---|
| **P0** | A1 骨架 + A2 EP 按需 + A3 命令式 CSS + A6 pet-widget L3 | **17 012 → ~200 ms** | 中 | 实测见 §D |
| **P1** | A5 App.vue async + A8 idle 预热 + A9 async CSS | 200 → 200（不变）/ LCP 3144 ms（不变）/ TBT 0（idlepet 测得） | 低 | D6 的 LCP/TBT 阈值 |
| **P2** | A7 SW 重写 + C 后端 gzip | 重复访问 FCP ~ 50 ms（走 cache）/ 全资源 -65% 字节 | 中-高 | §C6 部署后人工验 |

> **如果只做 P0**（PM 「P0 收益」问题）：**FCP 改善到 ~200 ms**，LCP ~3.1 s，TBT ~200 ms。**满足 D1–D4、D5、D6、D7**，D8 中 LCP 略超 1.5 s 阈值（A9 async CSS 解决）。

---

## 8 · 测试要点（devtools-qa 验收）

### L1 · 自动化可判

1. `pnpm build` 重新构建后：
   - `dist/index.html` 引用的 `assets/*` 总和（不含 `widgets/pet-widget.js` 因为已不引用）≤ 500 KB
   - `dist/assets/element-plus-*.js` **不存在**（§A2 的判据）
   - `dist/index.html` 不包含 `widgets/pet-widget.js` 字符串（§A6.3 的判据）
2. `python3 - <<EOF` 一段脚本：
   ```python
   import os,re,gzip,io
   d='frontend/dist/'
   refs=re.findall(r'(?:href|src)="(/assets/[^"]+)"',open(d+'index.html').read())
   raw=sum(os.path.getsize(d+r.lstrip('/')) for r in refs)
   gz=sum(gzip.compress(d+r.lstrip('/'),6) for r in refs)
   assert raw <= 500_000, f"raw {raw} > 500K"
   ```
3. `pnpm preview`（或本地 http server）后，Playwright CDP 测：
   - `fcp_ms ≤ 1000`（4G+4×CPU）
   - `lcp_ms ≤ 1500`
   - `tbt_ms ≤ 200`
   - `cls ≤ 0.1`
4. `--nojs` Playwright 截图：`body_text` 长度 > 0、`app_html` 长度 > 200、`fcp_ms != null`（骨架带真实文字时）
5. 路由 `route.meta.icon` 字符串解析正确 —— 跑过 §D.4 的 11 个 view，每个截图核对左侧图标不空
6. `route.meta.icon` 4 个 pre-existing 缺口（`DataAnalysis` / `Guide` / `Ticket` / `VideoCamera`）在访问对应路由时仍显示 —— 切到 `/monitor`、`/household`、`/screen-share`、`/paste`，截图核对

### L2 · 路径核实（人工）

7. `frontend/index.html` 同步 `<script src="/widgets/pet-widget.js">` **已删除**（grep 验证）
8. `frontend/src/main.js` 的 `import ElementPlus` 与 `import 'element-plus/dist/index.css'` **已删除**（grep 验证）
9. `frontend/src/utils/petWidgetLoader.js` 存在，`ensurePetWidget` 导出
10. `frontend/src/components/GlobalPet.vue` / `views/ai/PetTool.vue` / `views/ai/MiniMaxStudioTool.vue` 三个 `v-else` 占位 + `ensurePetWidget()` 调用就位
11. `backend/app_http.go` 有 `router.Use(gzip.Gzip(...))`、有 `WithExcludedPaths` 包含 `/api/health` 与至少 `/api/.*stream`
12. `backend/go.mod` 多了 `github.com/gin-contrib/gzip`
13. `frontend/vite.config.js` 多了 `unplugin-vue-components` 与 `unplugin-auto-import`，`manualChunks` 删了 `element-plus` 分组
14. `frontend/pnpm-lock.yaml` 已提交，`unplugin-*` 版本固定

### L3 · 端到端（人工）

15. 真实手机 4G（关闭 WiFi）打开 `https://t.jaxiu.cn`：
    - 冷启动首屏 ≤ 1.5s 出现可见结构（骨架文字 + 顶栏 + 卡片占位）
    - 主内容在 2-3s 内接续（HomeView mount 完成）
    - 宠物默认不可见；点 🐾 后 1-2s 出现宠物窗口（弱网下）
    - 工具切页无白屏，无错误
16. 健康探活（部署主机 `192.168.31.201`）：
    - `curl http://localhost:8082/api/health` 返回 `{"status":"ok"}` 且 `Content-Length: 15`
    - `Content-Encoding` **不出现**（§C1.2 白名单生效）
17. SSE 端点（部署主机 `192.168.31.201`）：
    - `/api/chat/stream` 流式输出逐字延迟 < 200ms（确认 gzip 没缓冲）
18. 后端加完 gzip 后，**主路由走一次**：
    - `/assets/*.js` 响应有 `Content-Encoding: gzip`
    - `/assets/*.png` **没有** `Content-Encoding`（`WithExcludedExtensions` 生效）

---

## 9 · 未覆盖 / 待人工裁决

- **D5·D6 是否足够**：本方案让禁 JS 时可见骨架文字、async CSS 后 FCP 196 ms，但**没有 Lighthouse 跑分**（R1 不用 Lighthouse CLI —— 本机未装 `lighthouse` npm 包；线上 CF 的 IPv6 / 0-RTT 等只能 jaxiu 验）。需要 jaxiu 部署到 `http://t.jaxiu.cn` 后跑一次 PageSpeed Insights。
- **Service Worker 重写**：本轮留空壳；PM 拍板可改，后续单独立项。
- **图标 7 个 dead entry**：本轮不动（移除风险 > 收益）；后续可在 cleanup pass 删。
- **后端 gzip 在容器内存压力下的表现**：未知（CLAUDE.md 描述机器同时跑 OCR/ASR/TTS sidecar，CPU 略紧；选 level 5 已平衡，但实测要 jaxiu 部署后跑一次）。
- **首屏实际可见文字的位置**：本方案在骨架里加 `DevTools` 文字（见 §A1），**实测可让 FCP < 200 ms**；若产品不喜欢骨架里写文字，**降级方案**为骨架只放灰块（FCP 仍 null 但页面有可见几何 —— 这是次优解）。
- **`unplugin-vue-components` + Vite 7.3.1 的兼容性**：本机实测 build 通过（§A2），但 dev 模式热更新可能比 v1 慢几秒 —— dev 体验需要 devtools-frontend 自己在 PR 时验证。

---

## 10 · 「需要 jaxiu 确认/配置」清单（非本轮交付）

> PM 已明确「**CF / 部署主机层的配置项只写进可选人工清单，不作为验收条件、不阻塞 stage**」。列出供 jaxiu 决策时参考。

1. **Cloudflare 是否已对 `/widgets/pet-widget.js` 配 `Cache-Control`**：建议 `public, max-age=86400`（1 天），客户端不会再向 devtools 后端要。
2. **Cloudflare 是否已对 `/assets/*-*.js` 配 brotli**：建议 `On`（比 gzip 多 ~15% 压缩；本方案 Level 5 gzip 是兜底，brotli 是锦上添花）。
3. **Cloudflare `Brotli` 与 `gzip` 同时启用时的优先级**：实测 Cloudflare 边缘默认优先 `gzip`；如需 brotli 优先需在 CF 控制台配置。
4. **`http/2 server push`** 是否启用：现代浏览器已基本忽略 server push，**不建议**花精力。
5. **健康探活协议**：当前 `docker-compose.yml:48` 走 wget `--spider`，本方案不破坏它；如想改用 `curl -fsSL` 或 HEAD，需 jaxiu 单独排期。

---

## 11 · 参考与验证命令汇总

### 11.1 实测脚本（QA 可复用）

```bash
# 1) 重建 + 量体积
rsync -a --exclude node_modules --exclude dist \
  $REPO/frontend/ /tmp/finn30-verify/src/
ln -sfn /tmp/finn29-arch/src/node_modules /tmp/finn30-verify/src/node_modules
cd /tmp/finn30-verify/src && pnpm install && pnpm build
python3 - <<'EOF'
import os,re
d='/tmp/finn30-verify/src/dist/'
refs=re.findall(r'(?:href|src)="(/assets/[^"]+)"',open(d+'index.html').read())
total=sum(os.path.getsize(d+r.lstrip('/')) for r in refs)
print(f"raw={total/1024:.1f}KB")
EOF

# 2) Playwright + 4G 节流测 FCP/LCP/TBT
node measure.mjs /tmp/finn30-verify/src/dist VERIFY --throttle-4g

# 3) 禁 JS 验 D5
node measure.mjs /tmp/finn30-verify/src/dist VERIFY-NOJS --nojs
```

### 11.2 本次实测留下的对照（`/tmp/finn29-arch/` 仍存有原型构建产物）

| 标签 | 路径 | FCP@4G | LCP@4G | 备注 |
|---|---|---:|---:|---|
| BASE | `dist-BASE/` | 17 012 | 17 516 | 当前线上 |
| L1(defer) | `dist-BASE-L1/` | 12 780 | — | `pet-widget.js` 加 `defer` |
| L3-only | `/tmp/finn30-verify/l3/src/dist` | 8 028 | 8 280 | 仅移除 `pet-widget.js` 同步 script |
| FINAL3 | `dist-FINAL3/` | 2 824 | 3 144 | EP 按需 + 骨架 + 全部改动 |
| ASYNC-CSS | `/tmp/finn30-verify/async-css/` | **196** | **3 148** | FINAL3 + async index.css + 骨架含真实文字 |

---

## 12 · 版本

| 字段 | 值 |
|---|---|
| 文档版本 | v1（基于实测 /tmp/finn29-arch 原型数据 + `/tmp/finn30-verify/` 独立复核） |
| 前置 commit | `da42899 chore(agent): baseline — the task worktree started here` |
| 目标 commit | devtools-frontend 后续子 issue + devtools-backend 后续子 issue |
| PM 关联 issue | FINN-29（父）、FINN-30（本）、FINN-41（后端追加，待 PM 创建） |