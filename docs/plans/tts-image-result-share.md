# TTS / 图像理解「保存分享」前端落地方案（FINN-66 Stage1）

> 基线：`origin/main` @ `542e457f76944b9a23bbe8433c924c28e1784cbf`。本文所有行号以该基线为准。
> 依据：PM 在 FINN-67 描述里的 6 条裁决（已批准）。本文只回答落地细节，**不重新讨论范围**。
> 参考实现只读：`git diff 3033560 1724eaf8`（fork 于 2026-04-24，早于前端重组，**禁止整枝 merge**，按 main 现状重写）。

## 目标与非目标

### 目标

1. **(A) TTS**：MiniMax Studio「语音 / 音色」标签页的同步 / 异步语音合成结果，可「保存分享」成公开分享页。
2. **(B) 图像理解**：独立路由页 `/image-understanding`（MiniMax MCP 标签）的识别结果，可直接保存成公开分享页。
3. 分享对话框、分享接口契约、后端表结构**均不变**。

### 非目标

- 不改后端：不加端点、不改 `POST /api/minimax/result-shares` 契约、不改 SQLite 表、不改 `AGENTS.md` 模块地图。
- 不移植参考分支的 **MiniMax 模型白名单收窄**（main 保留 `textModels` 超集，`MiniMaxStudioTool.vue:1057-1065` 不动）。
- 不移植参考分支的 **Qwen 分享入口**（`ImageUnderstandingTool.vue` 的「大模型理解 / M3」标签页**不加**保存分享按钮）。
- 不移植参考分支给媒体任务表新增的「操作」列（`shareMediaTask`），与本次范围无关。
- 不在 `/ai-gateway` 页（`AIGatewayTool.vue:532` 也挂了语音面板）加分享——那里没有分享对话框，另建对话框属于扩范围。
- 不新增分享对话框、不新增 localStorage 键（复用 MiniMax Studio 已有键）。

## 现状（已逐条 grep / sed 核实）

### 后端（只读，本次不改）

| 事实 | 位置 |
|------|------|
| 路由 + handler | `POST /api/minimax/result-shares`（`backend/routes/ai_gateway.go:68`，`api` 组 = `router.Group("/api")`，`backend/app_http.go:40`）→ `backend/handlers/minimax_result_share.go:111` `CreateMiniMaxResultShare` |
| 请求体契约 | `backend/handlers/minimax_result_share.go:24-39`：`title` / `summary` / `result_type`(required) / `model` / `payload` / `assets[]`；资产字段 `url` / `data_url` / `filename` / `content_type` / `kind`；`result_type` 小写化 + 非空校验见 `:122-126`（空则 400 `result_type 不能为空`） |
| `data_url` 解析（base64，`data:` 前缀） | `backend/handlers/minimax_result_share.go:754-775`；**内容类型取自 data URL 头**，请求里的 `content_type` 字段在 `data_url` 分支被忽略 |
| `url` 分支由后端服务端下载 | `backend/handlers/minimax_result_share.go:697-731`（`h.noProxyClient.Do`）；资产落盘 `:655-694`（落 `data/minimax_result_shares/<id>/`） |
| 硬限制 | 单次最多 **12** 个资产（`:659-661`）；单资产 ≤ **120MB**（`:22`、url 分支 `:720-722`、data_url 分支 `:771-773`）；全局请求体 ≤ **200MB**（`backend/app_http.go:35`） |
| 鉴权 | `backend/handlers/minimax_result_share.go:334-343`：`X-Super-Admin-Password`（等于 `cfg.AIGateway.SuperAdminPassword`）**或** `Authorization: Bearer <dtk_ai_xxx>`（`:345-383`）；缺失/无效 → 401 |
| 响应含 `share_url` | `backend/handlers/minimax_result_share.go:402-427` → `{origin}/minimax/share/{id}`；公开页路由 `frontend/src/router/index.js:662` |
| `display_type` 计算 | `backend/handlers/minimax_result_share.go:541-549`：仅 `result_type == "media"` 时按资产类型特判；`image_understanding` 原样返回 |

### 前端

| 事实 | 位置 |
|------|------|
| 语音面板**零**分享代码 | `frontend/src/components/AIGatewaySpeechPanel.vue`（1171 行，`grep -n "share\|Share"` → 0 命中） |
| 面板挂载点（2 处） | `frontend/src/views/ai/MiniMaxStudioTool.vue:165`（语音/音色 tab）、`frontend/src/views/ai/AIGatewayTool.vue:532` |
| 面板内 TTS 结果 | 同步音频块 `AIGatewaySpeechPanel.vue:80-83`（`syncAudioUrl`）；异步音频块 `:126-129`（`asyncAudioUrl`）；均为 **blob URL**（`:721` `URL.createObjectURL(blob)`、`:944` `hexToBlob`） |
| 面板凭证 | `AIGatewaySpeechPanel.vue:410-420` `hasAnyCredential()`、`:422-450` `jsonHeaders()` / `authHeaders()`（Bearer 优先，回退 `X-Super-Admin-Password`） |
| 面板已有 MIME 推断 | `AIGatewaySpeechPanel.vue:1038-1049` `guessAudioMime(format)`（mp3/wav/flac/pcm） |
| 既有分享对话框（父组件） | `MiniMaxStudioTool.vue:826-852`（`el-dialog v-model="createShareVisible"`） |
| 既有状态 / 打开 / 提交 | `MiniMaxStudioTool.vue:1028-1043`（`createShareVisible` / `createShareSubmitting` / `shareDraft` / `shareForm`）→ `:2143-2154` `openShareDialog(source)` → `:2156-2276` `buildShareDraft(source)` → `:2278-2309` `submitResultShare()`（POST body `{title,summary,result_type,model,payload,assets}`，`:2285-2292`） |
| 类型中文映射 | `MiniMaxStudioTool.vue:2455-2467` `shareTypeLabel()`：含 `speech:'语音'`，**缺** `image_understanding` |
| 父组件凭证来源 | `MiniMaxStudioTool.vue:919-920` localStorage 键 `minimax_studio_super_admin_password` / `minimax_studio_api_key`；`:925-926` 读取、`:1267-1268` 写回、`:1281-1282` 清除 |
| 图像理解页 | `frontend/src/views/ai/ImageUnderstandingTool.vue`（1109 行），**独立路由** `/image-understanding`（`frontend/src/router/index.js:443-451`），**未被** MiniMaxStudioTool 引用（`grep -rn "ImageUnderstandingTool" frontend/src/` → 仅路由一处）；该页**没有** AI Gateway 凭证，只有流水用的 `X-Image-Admin-Password`（`:563-565`），与本接口两种鉴权**都不匹配** → 必须读同源 localStorage |
| 图像理解页 MCP 结果 | `ImageUnderstandingTool.vue:88-94` 结果卡片头（`:92` 复制文本按钮）；状态 `resultText` `:315`、`rawResult` `:316`、`selectedTool` `:306`、`prompt` `:307`、`imageBase64` `:308`（**data URL**） |
| 公开分享页类型映射 | `frontend/src/views/share/MiniMaxResultShareView.vue:192-195`（缺 `image_understanding`）；画廊 `frontend/src/views/share/MediaGallery.vue:426-429`（缺）、筛选项 `MediaGallery.vue:228-233`（只有 全部/音乐/视频/图片） |

### 结论（对 PM 描述的复核）

- PM 描述的 endpoint、handler、契约、localStorage 键、`shareTypeLabel` 位置**全部与代码一致**；一处需补：资产 `data_url` 分支**忽略**请求里的 `content_type`（`minimax_result_share.go:754-775`），故前端必须保证 data URL 的 MIME 头正确并显式给 `kind`。
- **B1 事实**：`syncResponse`（`AIGatewaySpeechPanel.vue:343`，赋值 `:554`）是 `t2a_v2` 的**完整响应体**，音频是内联 hex（面板自取 `:941` `data.audio_file || data.audio`）→ 原样塞进 `payload.raw` 会同时撑爆对话框（`MiniMaxStudioTool.vue:1173` + 模板 `:845`）、请求体、落库（`minimax_result_share.go:152` 原样入库）、公开页（`MiniMaxResultShareView.vue:98/117` 全量渲染）。→ §1.4 用 `stripInlineAudio` 剔除（U1）。
- **B2 事实**：超管密码不匹配时后端**回落**到 API Key 鉴权（`minimax_result_share.go:339-342` → `:345-352`），错误文案是「缺少 API Key」；**不存在**「超级管理员密码不匹配」（`grep -rn "超级管理员密码不匹配" backend/` → 0 命中）→ T7 拆 T7a / T7b。

## 设计

### 0. 共用：draft 形状与字段口径

draft 是子组件 → 父组件 / 页面内部传递的唯一结构，形状与 `MiniMaxStudioTool.vue:1031-1039` 的 `shareDraft` **逐字段对齐**（1:1 映射到 POST body，`:2285-2292`）：

| draft 字段 | 类型 | 语义 | 落到 POST 的字段 |
|-----------|------|------|----------------|
| `sourceLabel` | string | 对话框「来源」行 | 不上传（仅 UI 展示，`:840`） |
| `title` | string | 分享标题，**硬截断 ≤120 字符** | `title` |
| `summary` | string | 分享摘要，**硬截断 ≤240 字符** | `summary` |
| `resultType` | string | 小写结果类型 | `result_type` |
| `model` | string | 模型 / 工具名 | `model` |
| `payload` | object | 原始结果 | `payload` |
| `assets` | array | 资产数组 | `assets` |

资产对象字段：`{ url?, data_url?, filename?, content_type?, kind }`，其中 `kind` 必填（`minimax_result_share.go:681-684` 优先用 `kind`，缺失才按 MIME/文件名推断，`audio`/`image`/`video`/`file`）。

**title / summary 长度保护（NB4）**：两处 draft 的 `title` / `summary` 一律**硬截断**到 **120 / 240 字符**（`String(x).slice(0, 120)` / `slice(0, 240)`），与对话框 `el-input maxlength="120"`（`MiniMaxStudioTool.vue:831`）/ `maxlength="240"`（`:834`）同口径。后端**只 TrimSpace、不校验长度**（`minimax_result_share.go:139-140`），所以 (B) 路径**绕过对话框**时前端必须自己截断，否则超长 title/summary 会原样入库。

**R6 铁律**：blob URL **绝不**进 draft。blob → data URL；本来就是 http(s) 的 → 走 `url` 字段让后端下载。

### 1. (A) TTS —— 子组件 emit draft → 父组件既有分享对话框

#### 1.1 路径（PM 裁决 6① 写死）

```
AIGatewaySpeechPanel.shareSyncResult()/shareAsyncResult()
  → emit('share', draft)
  → MiniMaxStudioTool.handleShareFromChild(draft)   ← 新增
  → MiniMaxStudioTool.openShareWithDraft(draft)      ← 新增，复用 shareDraft/shareForm/createShareVisible
  → 既有 el-dialog（MiniMaxStudioTool.vue:826-852）弹窗
  → 既有 submitResultShare()（:2278）POST /api/minimax/result-shares
```

**不新建对话框、不改 `submitResultShare`、不改契约。**

#### 1.2 `frontend/src/components/AIGatewaySpeechPanel.vue` 改动清单

| # | 位置（现状行号） | 改动 |
|---|------------------|------|
| A1 | `:291-300` `defineProps` | 新增 prop `shareEnabled: { type: Boolean, default: false }`。**理由**：本组件被 `AIGatewayTool.vue:532` 复用，那里没有分享对话框；不加门控会出现「点了没反应」的死按钮。 |
| A2 | `:300` 之后 | 新增 `const emit = defineEmits(['share'])` |
| A3 | `:82` 之后（同步音频块 `:80-83` 内） | 插入 `<el-button v-if="shareEnabled" type="primary" size="small" :loading="sharingSync" @click="shareSyncResult">保存分享</el-button>` |
| A4 | `:128` 之后（异步音频块 `:126-129` 内） | 插入 `<el-button v-if="shareEnabled" type="primary" size="small" :loading="sharingAsync" @click="shareAsyncResult">保存分享</el-button>` |
| A5 | `:341` 之后 | 新增 `const sharingSync = ref(false)` / `const sharingAsync = ref(false)` |
| A6 | `:1056` 之后（`hexToBlob` 之后） | 新增 `blobUrlToDataUrl(blobUrl, mime)`（见 1.3）与 `stripInlineAudio(resp)`（见 1.4） |
| A7 | A6 之后 | 新增 `shareSyncResult()` / `shareAsyncResult()`（见 1.4） |

> 面板 loading 只覆盖「blob → data URL 转换」这段；真正的 POST 由父组件的 `createShareSubmitting`（`:1029`）负责。

#### 1.3 `blobUrlToDataUrl(blobUrl, mime)` 口径

- 输入 `blob:` URL → `fetch(blobUrl)` → `blob` → **用正确 MIME 重新包装** `new Blob([blob], { type: mime })` → `FileReader.readAsDataURL` → `data:audio/mpeg;base64,...`。
  - 重新包装的原因：异步任务的音频来自 `res.blob()`（`:721`），其 MIME 取自响应头，可能是 `application/octet-stream`；不重包则 data URL 头 MIME 错、后端 `miniMaxShareAssetKind`（`:793-806`）判成 `file`。同步路径的 `hexToBlob` 已带正确 MIME（`:944`）。
- 输入 `http(s)://` URL → **不转**，返回 `null`，调用方改用资产 `url` 字段（后端服务端下载，避免大 body）。
  - **约束（U3）**：只接受 `https:` 协议、且 URL **必须来自上游 MiniMax 响应**（`setAudioFromSpeechResponse` 的 `data.audio_url || payload?.audio_url`，`AIGatewaySpeechPanel.vue:956-959`），**不接受用户输入**、不接受 `http://`。该分支是后端**既有能力**（`minimax_result_share.go:697-731`），本方案**零后端改动**；SSRF 面属既有面、本方案只是首次由浏览器流程触发。
- 其他 / 空 → 返回 `null`。
- 体积闸门：原始字节 > **20MB** → 抛 `Error('音频过大，无法直接保存分享')`，不静默降级。**口径 = 原始音频字节**（不是 data URL 长度）。
- 失败一律 `try/catch` → `ElMessage.error`，`finally` 复位 loading。

**`mime` 实参写死（U2，三条）**：

| 路径 | mime | 依据 |
|------|------|------|
| 同步 · 内联 hex | `guessAudioMime(syncForm.format)` | `AIGatewaySpeechPanel.vue:1038-1049`；`format` 定义 `:356` |
| 同步 · file_id 预览（`:703` `filePreviewUrl`，blob 同样来自 `res.blob()`） | `guessAudioMime(syncForm.format)` | 与内联 hex 同口径 |
| 异步 | `audio/mpeg`（固定） | 异步任务产物固定 mp3 |

- filename 扩展名必须与 mime 一致（同步 `syncForm.format` ∈ mp3/wav/flac/pcm；异步 `.mp3`）。

#### 1.4 draft 字段（逐字段写死）

**(A-1) 同步合成** `shareSyncResult()`：

| 字段 | 取值 |
|------|------|
| `sourceLabel` | `'语音合成'` |
| `title` | `` `TTS · ${syncForm.text.trim().slice(0, 36)}` ``（`syncForm` 定义于 `:351-358`；文本为空时退化为 `'TTS'`）→ 再按 §0 硬截断 ≤120 |
| `summary` | `` `${syncForm.model} 同步合成 · 音色 ${syncForm.voiceId}` `` → 再按 §0 硬截断 ≤240 |
| `resultType` | `'speech'`（`shareTypeLabel` 已有 `speech:'语音'`，`:2458`） |
| `model` | `syncForm.model` |
| `payload` | `{ text: syncForm.text, voice_id: syncForm.voiceId, speed: syncForm.speed, format: syncForm.format, sample_rate: syncForm.sampleRate, mode: 'sync', raw: stripInlineAudio(syncResponse) }`（`syncResponse` = `:343`；**必须剔除内联音频，见下**） |
| `assets` | 1 个：`{ data_url \| url, kind: 'audio', filename: \`tts-${syncForm.model}.${syncForm.format}\` }`（`format` ∈ mp3/wav/flac/pcm，定义于 `:356`、选项见模板 `:56-62`；扩展名随格式走，**不固定 mp3**；mime 取 `guessAudioMime(syncForm.format)`，见 §1.3） |

**（A-1）`raw` 必须剔除内联音频（U1，B1 修法）**：`syncResponse`（`:343`，赋值 `:554`）是 `t2a_v2` 的**完整响应体**，音频是内联 hex 字符串（面板自取 `:941` `data.audio_file || data.audio`）。原样塞进 `payload.raw` 会同时撑爆 ① 分享对话框（`MiniMaxStudioTool.vue:1173` `shareDraftPretty` + 模板 `:845` 渲染 → 40MB 文本节点，弹窗冻结）② 请求体（hex 2× + base64 1.33×）③ 落库与公开页（`minimax_result_share.go:152` 原样入库；`MiniMaxResultShareView.vue:98/117` 全量渲染）。

实现（写死，函数体可直接抄）：

```js
function stripInlineAudio(resp) {
  if (!resp || typeof resp !== 'object') return resp
  const { audio, audio_file, ...topRest } = resp        // 顶层 audio / audio_file 一并剔除
  const data = resp.data
  if (!data || typeof data !== 'object') return topRest
  const { audio: _a, audio_file: _af, ...rest } = data
  return { ...topRest, data: rest }
}
```

- 只删 `audio` / `audio_file` 两个键（hex 音频本体），**其余字段原样保留**（`trace_id` / `status` / `extra_info` 等元信息仍进 `payload.raw`）。
- `resp.data` 不存在 → 返回剔除顶层的 `resp`；`resp` 非对象 → 原样返回。
- **不写脏** `syncResponse` 本身（解构出新对象，原 ref 不被修改）。
- 顶层也剔除的理由：面板取值链 `data.audio_file || data.audio || payload?.audio_file || payload?.audio`（`:941`）兼容顶层音频，且 U1 原文要求同时删顶层 `audio` / `audio_file`。
- **异步路径 (A-2) 不受影响**：`raw: currentTask`（`:345`）只含 `file_id` / 任务元信息、无内联音频，**不改**。

**(A-2) 异步合成** `shareAsyncResult()`：

| 字段 | 取值 |
|------|------|
| `sourceLabel` | `'语音合成'` |
| `title` | `` `TTS · ${inputText.trim().slice(0, 36)}` ``，`inputText` = `asyncForm.mode === 'text' ? asyncForm.text : asyncForm.textFileId`（`asyncForm` 定义于 `:360-366`）→ 再按 §0 硬截断 ≤120 |
| `summary` | `` `${asyncForm.model} 异步合成 · 音色 ${asyncForm.voiceId} · 任务 ${taskId}` ``，`taskId` 取自 `currentTask`（`:345`，`extractTaskId` 见 `:996`）→ 再按 §0 硬截断 ≤240 |
| `resultType` | `'speech'` |
| `model` | `asyncForm.model` |
| `payload` | `{ text: inputText, voice_id: asyncForm.voiceId, mode: asyncForm.mode, text_file_id?: asyncForm.textFileId, task_id: taskId, raw: currentTask }`（`currentTask` 只含 `file_id` / 任务元信息，**无内联音频 → 不受 U1 影响**） |
| `assets` | 1 个：`{ data_url, kind: 'audio', filename: \`tts-${asyncForm.model}.mp3\` }`（异步任务产物固定 mp3，mime 固定 `audio/mpeg`，见 §1.3） |

提交前守卫：同步 `if (!syncAudioUrl.value || !hasAnyCredential()) return`、异步 `if (!asyncAudioUrl.value || !hasAnyCredential()) return`（`hasAnyCredential` = `:410`）。

#### 1.5 `frontend/src/views/ai/MiniMaxStudioTool.vue` 改动清单

| # | 位置 | 改动 |
|---|------|------|
| M1 | `:165` | `<AIGatewaySpeechPanel :super-admin-password="superAdminPassword" :prefill-api-key="apiKey" share-enabled @share="handleShareFromChild" />` |
| M2 | `:2154` 之后（`openShareDialog` 之下） | 新增 `handleShareFromChild(draft)` 与 `openShareWithDraft(draft)`；`openShareDialog`（`:2143-2154`）改为调用 `openShareWithDraft(draft)`，消除重复赋值 |

重构后**完整三段**（`requireCredential()` 前置与 `try/catch` **必须保留** —— `buildShareDraft` 会 throw「当前没有文本结果」等，删掉 `try/catch` 会让 T10 回归项挂）：

```js
function openShareDialog(source) {
  if (!requireCredential()) return
  try {
    openShareWithDraft(buildShareDraft(source))
  } catch (err) {
    ElMessage.error(err.message || '当前结果还不能保存分享')
  }
}

function openShareWithDraft(draft) {
  shareDraft.value = draft
  shareForm.value.title = draft.title
  shareForm.value.summary = draft.summary
  createShareVisible.value = true
}

function handleShareFromChild(draft) {
  if (!requireCredential()) return
  openShareWithDraft(draft)
}
```

- `requireCredential()`（`:1288`）用**父组件**凭证（`apiKey` / `superAdminPassword`），因为真正发 POST 的是父组件 `submitResultShare`（`jsonHeaders()` 走父组件凭证，`:1306-1314`）。子组件面板里的调试 Key 与父组件不一致时以父组件为准。
- 弹窗「类型」行显示 `shareDraft.resultType`（`:841`）= `'speech'`，无需改文案。（「类型」行显示原始枚举 `speech` 而非「语音」是**既有行为** —— 既有 5 个分享入口同样表现，非本方案引入，不改。）

### 2. (B) 图像理解 —— 独立页直接 POST

#### 2.1 路径（PM 裁决 5）

`ImageUnderstandingTool.vue` 是**独立路由页**（`router/index.js:443-451`），**没有父组件**承接 emit（PM 已核实 main 已把它从 MiniMaxStudioTool 里拆出）→ **不新建分享对话框**，页面内直接 POST `/api/minimax/result-shares`。

凭证：读**同源** localStorage 已有键（不另造键）：

| localStorage 键 | 来源 | 用作 |
|----------------|------|------|
| `minimax_studio_api_key` | `MiniMaxStudioTool.vue:920`（写回 `:1268`） | `Authorization: Bearer <key>`（**优先**） |
| `minimax_studio_super_admin_password` | `MiniMaxStudioTool.vue:919`（写回 `:1267`） | `X-Super-Admin-Password`（次选） |

优先级与 `MiniMaxStudioTool.authHeaders()`（`:1296-1304`）完全一致：先 API Key，后超管密码，两者都空 → 提示并**不发请求**。

#### 2.2 `frontend/src/views/ai/ImageUnderstandingTool.vue` 改动清单

| # | 位置（现状行号） | 改动 |
|---|------------------|------|
| I1 | `:299`（import 区块 `:295-298` 之后） | 新增常量 `const SHARE_SUPER_ADMIN_KEY = 'minimax_studio_super_admin_password'`、`const SHARE_API_KEY_STORAGE = 'minimax_studio_api_key'` |
| I2 | `:318` 之后 | 新增 `const sharingMcp = ref(false)` |
| I3 | `:92` 之后（结果卡片头） | 插入 `<el-button text :disabled="!resultText" :loading="sharingMcp" @click="shareMcpResult">保存分享</el-button>` |
| I4 | `:813` 之后（`copyResult` 之下） | 新增 `shareMcpResult()`（见 2.3） |

> **只加在 MiniMax MCP 标签页**（`:92` 所在卡片）。「大模型理解 / M3」标签页（`:221` 附近）**不加**（PM 裁决 2：不移植 Qwen 分享入口）。

#### 2.3 `shareMcpResult()` draft 字段（逐字段写死）

| 字段 | 取值 |
|------|------|
| `sourceLabel` | `'图像理解'` |
| `title` | `` `图像理解 · ${(prompt.value || '').trim().slice(0, 36)}` ``；`prompt` 为空 → `'图像理解'` → 再按 §0 硬截断 ≤120 |
| `summary` | `` `MiniMax MCP 图像理解 · 工具 ${selectedTool.value || 'MiniMax-MCP'}` `` → 再按 §0 硬截断 ≤240 |
| `resultType` | `'image_understanding'`（**新的 result_type 取值**；该字符串在 main 已存在，但只用作 config 键 `backend/config/config.go:43` 与**监控 / 日志记录字段**（`logAPIRequest` 的第 5 个实参，`backend/handlers/ai_gateway_image.go:393`，与路由注册无关），与本域无关。后端对 `result_type` 无白名单，只小写化 + 非空校验 `minimax_result_share.go:122-126`，唯一特判是 `display_type` 对 `'media'` 的处理 `:543`） |
| `model` | `selectedTool.value || 'MiniMax-MCP'`（**与 `summary` 用同一表达式**，U6：工具列表未加载 / 用户未选时两者都取 `'MiniMax-MCP'`，不再一个 `'auto'` 一个 `'MiniMax-MCP'`） |
| `payload` | `{ text: resultText.value, prompt: prompt.value, tool: selectedTool.value, source: 'minimax-mcp', raw: rawResult.value }`（`payload.tool` 原样保留 `selectedTool.value`，可能为空串；`rawResult` 是上游原始 JSON，**不脱敏** —— U7，与既有 text/media 分享的 `raw` 口径一致） |
| `assets` | 1 个（可选）：`{ data_url: imageBase64.value, kind: 'image', filename: imageFile.value?.name || 'image-understanding.png' }`。`imageBase64` 已是 `data:image/...;base64,`（`:308`，由 `handleFileChange` 的 `FileReader.readAsDataURL` 写入，`:409-417`）→ **天然满足 R6**（不是 blob URL）。**10MB 闸门口径 = 原始文件字节 `imageFile.value?.size`**（U4；`imageFile` = `:310`，`file.raw` 写入于 `:414`），**不是** data URL 解码后字节（base64 膨胀 4/3）。原始图片 > **10MB** 时不带资产，只分享文本结果，并提示「图片过大，仅保存识别文本」。 |

提交：

```
POST /api/minimax/result-shares
headers: Authorization: Bearer <apiKey>   （无 apiKey 时）X-Super-Admin-Password: <password>
         Content-Type: application/json
body:    { title, summary, result_type: 'image_understanding', model, payload, assets }
```

- 成功（201）→ `ElMessage.success('图像理解结果已保存为分享页')` + `navigator.clipboard.writeText(data.share_url)` + `ElMessage.success('分享链接已复制')`（`data.share_url` 由 `minimax_result_share.go:426` 返回）。**不弹新对话框**。
- **不跳转**（U5）：(B) 是独立路由页，**没有 archive tab 可切**，保存成功后**只复制链接 + toast**，不切页。(A) 路径保存成功会切 archive tab（`MiniMaxStudioTool.vue:2298` `activeTab.value = 'archive'`），那是 MiniMax Studio 页**页内**行为。**两者差异是刻意的、非缺陷**，不做统一。
- 失败 → `ElMessage.error(data.error || '保存分享失败')`（401 走这条，错误文案由后端给：`缺少 API Key` / `API Key 无效` / `API Key 格式错误`，见 `minimax_result_share.go:351/355/371`）。

### 3. 类型中文映射（PM 裁决 6②）

| 文件 | 位置 | 改动 |
|------|------|------|
| `frontend/src/views/ai/MiniMaxStudioTool.vue` | `:2455-2467` `shareTypeLabel()` 的 map | **必做**：补 `image_understanding: '图像理解'`（`speech:'语音'` 已存在） |
| `frontend/src/views/share/MiniMaxResultShareView.vue` | `:192-195` `typeLabel()` 的 map | **P2 附带**：补 `image_understanding: '图像理解'`，否则公开分享页标签直接显示英文枚举 |
| `frontend/src/views/share/MediaGallery.vue` | `:426-429` `typeLabel()` 的 map | **P2 附带**：同上，画廊封面标签 |

P2 两项各 1 行、不改任何逻辑；PM 若判定超范围可砍，砍后表现仅是英文枚举透出（功能不受影响）。

### 后端改动清单

**无。** 逐条核对结论：路由已存在（`routes/ai_gateway.go:68`）、契约已支持 `data_url`（`minimax_result_share.go:35`）、鉴权两种方式齐备（`:334-383`）、资产上限满足本次用量（1 个资产 ≪ 12；TTS ≤ 20MB ≪ 120MB）。

**请求体口径（U1 落地后的真实数字）**：20MB 音频 → base64 data URL ≈ **26.7MB**（4/3 膨胀）→ **请求体 ≈ 27MB**（`payload.raw` 剔除内联音频后只剩元信息，残余 < 1MB）→ **远低于 Cloudflare 单请求体上限 100MB 与后端 200MB body 限额**（`backend/app_http.go:35` `ContentSizeLimiter(200 * 1024 * 1024)`）。**若不剔除内联音频**，body 会变成 ≈67MB（hex 2× + base64 1.33×），逼近 CF 上限 → 这正是 U1 必须落地的原因。

### 前端改动清单（汇总）

| 文件 | 改动点 | 条数 |
|------|--------|------|
| `frontend/src/components/AIGatewaySpeechPanel.vue` | A1–A7 | 7 |
| `frontend/src/views/ai/MiniMaxStudioTool.vue` | M1–M2 + `shareTypeLabel` 补 map | 3 |
| `frontend/src/views/ai/ImageUnderstandingTool.vue` | I1–I4 | 4 |
| `frontend/src/views/share/MiniMaxResultShareView.vue` | P2 `typeLabel` 补 map | 1 |
| `frontend/src/views/share/MediaGallery.vue` | P2 `typeLabel` 补 map | 1 |

**合计 16 个改动点**（必做 14 + P2 附带 2）。零后端改动、零路由改动、零 DB 改动。

## 兼容性与迁移

- **无接口变更**、无表结构变更、无数据回填。纯前端新增按钮与新函数。
- 老分享记录不受影响：`result_type` 新取值 `image_understanding` 之前不存在；已有分享的 `result_type` 值不变，`display_type` 计算逻辑不变（`minimax_result_share.go:541-549` 只对 `media` 特判）。
- 老页面不受影响：`shareTypeLabel` / `typeLabel` 只是 map 增项，命中不到时走 `map[type] || type` 兜底。
- 已知局限（不在本次范围）：画廊筛选项只有 全部/音乐/视频/图片（`MediaGallery.vue:228-233`）；后端 `minimaxResultShareMatchesType`（`minimax_result_share.go:567-581`）先按 `result_type`/`display_type` 精确匹配、**再**按 `asset_kinds` 匹配（`:575-579`）→ 带图片资产的分享会落在「图片」筛选下，没带图片资产的只出现在「全部」里。详见未决问题 2。

## 风险与回滚

| 风险 | 概率 | 影响 | 处置 |
|------|------|------|--------|
| 同步 TTS 音频很大，data URL 撑爆请求体 | 低 | 保存失败 | 前端 20MB 闸门（1.3，量**原始字节**）；后端 120MB/200MB 双闸门兜底；且 **`payload.raw` 剔除内联音频（U1），不产生 hex 副本** → body ≈27MB |
| `payload.raw` 塞入内联 hex 音频 | 中（实现易漏） | 对话框冻结 / 公开页打不开 / 落库膨胀 | `raw: stripInlineAudio(syncResponse)`（1.4）+ 静态检查「`grep -n "raw: syncResponse"` 输出为空」 |
| 异步任务音频 MIME 是 `application/octet-stream` → 分享页不出现播放器 | 中 | 体验降级 | 1.3 强制重包 MIME；且 draft 显式给 `kind:'audio'`（后端优先用 `kind`） |
| 图像理解页没填过 MiniMax Studio 凭证 → 点保存分享报「缺少 API Key」 | 中 | 用户困惑 | 按钮点击时先检查两个 localStorage 键，缺则提示「请先在 MiniMax 工作室填写 AI Gateway API Key 或超级管理员密码」并**不发请求** |
| 面板被 `/ai-gateway` 页复用 → 出现点不动的死按钮 | 中 | 体验降级 | A1 的 `shareEnabled` 默认 false 门控 |
| 语音面板 share 事件被父组件吞掉（父无凭证） | 低 | 无提示 | `handleShareFromChild` 调 `requireCredential()`（`:1288`）弹错误 |
| 分享链接在 CF 后 origin 错位 | 低 | 链接打不开 | `share_url` 由后端按 `X-Forwarded-Proto/Host` 拼（`:402-403`、`:807-810`），沿用既有行为 |

**回滚**：全部改动集中在 3（+2）个前端文件、无后端/DB 变更 → `git revert` 单个 commit 即可完全回滚；无残留数据（分享记录留在 `minimax_result_shares` 表，与现有行为一致，删除走既有 `DELETE /api/minimax/result-shares/admin/:id`）。

## 测试要点（交给 devtools-qa，全部可判定）

### 静态检查（可脚本化，PASS/FAIL 二值）

1. 后端零改动：`git diff origin/main --name-only -- backend/` 输出为空。
2. `grep -c "openShareWithDraft" frontend/src/views/ai/MiniMaxStudioTool.vue` ≥ 2（定义 + 至少一处调用）。
3. `grep -n "image_understanding: '图像理解'" frontend/src/views/ai/MiniMaxStudioTool.vue` 命中 1 行。
4. `grep -n "shareEnabled" frontend/src/components/AIGatewaySpeechPanel.vue` 命中 ≥ 3（prop 声明 + 两处模板 `v-if`）。
5. **R3**：`awk 'NR<287 && /\.value/ {print NR": "$0}' frontend/src/components/AIGatewaySpeechPanel.vue` 输出为空（模板段 1–286 行不得出现 `.value`）。
6. **R4**：改动涉及的模板段 `awk 'NR<287 && /\[0\]|\[[0-9]+\]/ {print NR": "$0}' frontend/src/components/AIGatewaySpeechPanel.vue` 输出为空。
7. `grep -n "blob:" frontend/src/views/ai/MiniMaxStudioTool.vue frontend/src/views/ai/ImageUnderstandingTool.vue` 输出为空（不得把 blob URL 写进 draft）。
8. **U1 防回退**：`grep -n "raw: syncResponse" frontend/src/views/ai/MiniMaxStudioTool.vue frontend/src/components/AIGatewaySpeechPanel.vue` 输出为**空**（防止实现时把整份响应原样塞进 `payload.raw`）。
9. `grep -n "stripInlineAudio" frontend/src/components/AIGatewaySpeechPanel.vue` 命中 ≥ 2（函数定义 + `shareSyncResult` 里的调用）。
10. `grep -rn "console\.log\|TODO\|FIXME" frontend/src/components/AIGatewaySpeechPanel.vue frontend/src/views/ai/MiniMaxStudioTool.vue frontend/src/views/ai/ImageUnderstandingTool.vue` 只允许命中存量行，且**不得**出现在本次 diff 范围内（用 `git diff origin/main -- frontend/ | grep -n "console.log\|TODO\|FIXME"` 判定为空）。

### 手工路径（浏览器，MiniMax Studio + 图像理解页）

| 编号 | 步骤 | 期望 |
|------|------|------|
| T1 | `/minimax-studio` → 语音/音色 → 同步合成 → 点「保存分享」 | 弹窗打开；「来源」= 语音合成；标题 = `TTS · <文本前 36 字>`；摘要含模型与音色；资产数 = 1 |
| T2 | T1 弹窗点「保存分享」 | toast「MiniMax 结果已保存为分享页」；新标签打开 `https://t.jaxiu.cn/minimax/share/<id>`，页面含 **audio 播放器**且可播放；管理列表出现该条，类型标签显示「语音」 |
| T3 | 异步：创建任务 → 轮询出音频 → 保存分享 → 保存 | 同 T2，摘要含任务号；分享页 payload 里 `task_id` 存在 |
| T4 | 同步合成时上游返回 **https 直链**（`data.audio_url`，非 blob）→ 保存分享 | 保存成功；请求体**不含**音频 base64（走 `url` 资产，后端服务端下载）；分享页可播放。`http://` 直链 → 前端**不采纳**（只接受 https，见 §1.3 U3） |
| T5 | `/image-understanding` → MiniMax MCP → 上传图 → 识别 → 点「保存分享」 | toast「图像理解结果已保存为分享页」+「分享链接已复制」；剪贴板内容为 `https://t.jaxiu.cn/minimax/share/<id>`；打开该页可见识别文本 + 输入图片；类型标签「图像理解」 |
| T6 | 同 T5，但**不**在 MiniMax Studio 存过凭证（清 `minimax_studio_api_key` / `minimax_studio_super_admin_password` 后刷新） | 提示「请先在 MiniMax 工作室填写…」，**DevTools Network 无** `POST /api/minimax/result-shares` 请求 |
| T7a | **清空** `minimax_studio_api_key`，只留**错误**的 `minimax_studio_super_admin_password` → T5 | 服务端 **401**，页面提示 **「缺少 API Key」**（`minimax_result_share.go:351`），**无 500**；DevTools Network 可见 `POST /api/minimax/result-shares` 发出 |
| T7b | 保留**正确**的 `minimax_studio_api_key` + **错误**的 `minimax_studio_super_admin_password` → T5 | **201 成功**（Bearer 优先，§2.1 优先级链），证明鉴权优先级没写反 |
| T8 | `/ai-gateway` 页的语音面板 | **不出现**「保存分享」按钮（`shareEnabled` 默认 false） |
| T9 | `/image-understanding` 的「大模型理解 / M3」标签页 | **不出现**「保存分享」按钮（PM 裁决 2） |
| T10 | 回归：MiniMax Studio 原有 5 个分享入口（文本/媒体/歌词/翻唱/一键音乐） | 行为与改动前一致（`openShareDialog` 重构后仍正常） |
| T11 | 图像理解页上传 > 10MB 的图 → 识别 → 保存分享 | 保存成功，分享页**无图片资产**但有文本结果，且有「图片过大，仅保存识别文本」提示 |

## 未决问题

1. **P2 附带项是否保留**：`MiniMaxResultShareView.vue:193` 与 `MediaGallery.vue:427` 的 `typeLabel` 补 `image_understanding`（各 1 行，纯展示）。默认**保留**（不补则公开页显示英文枚举）；若 PM 判定超范围，砍掉这两行不影响功能。
2. **画廊筛选**：是否给画廊加「图像理解」筛选项（`MediaGallery.vue:228-233`）。本次**不做**（后端 `type` 查询参数已支持，`minimax_result_share.go:216`，`minimaxResultShareMatchesType` 也能精确匹配 `image_understanding`，故纯前端加一项即可；留作后续单独立项）。
3. **`/ai-gateway` 页是否也支持 TTS 分享**：本次**不做**（那里没有分享对话框，另建对话框超范围）。若 PM 要，等价做法是把 AIGatewayTool 也接入同一套对话框（需单独一个对话框或复用 MiniMaxStudio 组件），另立单。
4. **无阻塞项**：本次方案不依赖任何人工裁决即可开工。Stage 2 评审提出的 U1–U7 已由 PM 全部拍板并写死进本文（U1 → §1.4 `stripInlineAudio` + §后端改动清单 + 风险表 + 静态检查 8；U2 → §1.3 mime 表；U3 → §1.3 https 约束 + T4；U4 → §2.3 图片闸门；U5 → §2.3 不跳转；U6 → §2.3 summary/model 同表达式；U7 → §2.3 `payload` 不脱敏），**实现方不得自行取舍**。
