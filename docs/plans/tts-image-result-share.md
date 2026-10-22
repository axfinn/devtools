# TTS / 图像理解「保存分享」前端落地方案（FINN-66 Stage1 · v3 修订）

> 基线：`origin/main` @ `542e457f76944b9a23bbe8433c924c28e1784cbf`。本文所有行号以该基线为准。
> v1：`c18040be9cbce7d16172020c92ed7ea231c03427`（Stage 2 评审 FINN-68 判 **FAIL**）。
> v2：`27ae744d9cc13e6293bbeaa74296ef2dfe98deb6`（Stage 2 复审 FINN-75 判 **FAIL**，唯一阻塞项 **M1**）→ **v3 修订（本文）**：按 PM 已拍板的 **U1–U13** 与评审阻塞项 **M1** + 非阻塞项 N1–N5 逐条回改；**不重新讨论范围**，实现方不得自行取舍。
> 依据：FINN-68 / FINN-75 评审报告 + PM 裁决（U1–U13）。参考实现只读：`git diff 3033560 1724eaf8`（fork 于 2026-04-24，早于前端重组，**禁止整枝 merge**）。本次只改本文一个文件，**零业务代码改动**。

## 目标与非目标

### 目标

1. **(A) TTS**：MiniMax Studio「语音 / 音色」标签页的同步 / 异步语音合成结果，可「保存分享」成公开分享页。
2. **(B) 图像理解**：独立路由页 `/image-understanding`（MiniMax MCP 标签）的识别结果，可直接保存成公开分享页。
3. 分享对话框、分享接口契约、后端表结构**均不变**。

### 非目标

- 不改后端：不加端点、不改 `POST /api/minimax/result-shares` 契约、不改 SQLite 表、不改 `AGENTS.md` 模块地图。
- 不移植参考分支的 **MiniMax 模型白名单收窄**（main 保留 `textModels` 超集，`MiniMaxStudioTool.vue:1057-1066` 不动）。
- 不移植参考分支的 **Qwen 分享入口**（`ImageUnderstandingTool.vue` 的「大模型理解 / M3」标签页**不加**保存分享按钮），也不移植其媒体任务表「操作」列（`shareMediaTask`）。
- 不在 `/ai-gateway` 页（`AIGatewayTool.vue:532` 也挂了语音面板）加分享——那里没有分享对话框，另建对话框属于扩范围。
- 不新增分享对话框、不新增 localStorage 键（复用 MiniMax Studio 已有键）；**不统一** (A)/(B) 两条路径保存后的后续动作（见 §2.3、U5）。

## 现状（已逐条 grep / sed 核实，基线 542e457）

### 后端（只读，本次不改）

| 事实 | 位置 |
|------|------|
| 路由 + handler | `POST /api/minimax/result-shares`（`backend/routes/ai_gateway.go:68`，`api` 组 = `router.Group("/api")`，`backend/app_http.go:40`）→ `backend/handlers/minimax_result_share.go:111` `CreateMiniMaxResultShare` |
| 请求体契约 | `backend/handlers/minimax_result_share.go:24-39`：`title` / `summary` / `result_type`(required) / `model` / `payload` / `assets[]`；资产字段 `url` / `data_url` / `filename` / `content_type` / `kind`；`result_type` 小写化 + 非空校验见 `:122-125`（空则 400 `result_type 不能为空`） |
| `title`/`summary`/`payload` 落库 | `minimax_result_share.go:139-140`：**只 `TrimSpace`，不校验长度** → 长度保护必须由前端做（见 §1.4 / §2.3，NB4）；`:152` `share.Payload = string(req.Payload)`（原样入库，**无裁剪**） |
| `data_url` 解析（base64，`data:` 前缀） | `minimax_result_share.go:754-775`；**内容类型取自 data URL 头**，请求里的 `content_type` 字段在 `data_url` 分支被忽略 |
| `url` 分支由后端服务端下载 | `minimax_result_share.go:697-731`（`h.noProxyClient.Do`，**无 host 白名单**）；资产落盘 `:655-694`（落 `data/minimax_result_shares/<id>/`） |
| 硬限制 | 单次最多 **12** 个资产（`:659-661`）；单资产 ≤ **120MB**（`:22`、url 分支 `:720-722`、data_url 分支 `:771-773`）；全局请求体 ≤ **200MB**（`backend/app_http.go:35`） |
| 鉴权 | `minimax_result_share.go:334-343` `authenticateAdminOrAnyAPIKey`：`X-Super-Admin-Password`（等于 `cfg.AIGateway.SuperAdminPassword`）**或** `Authorization: Bearer <dtk_ai_xxx>`（`:345-383` `authenticateAnyAPIKey`）；**超管密码不匹配时回落**到 `authenticateAnyAPIKey`，无 Bearer 时 → 401 `缺少 API Key`。⚠️ `grep -n "密码不匹配" backend/handlers/minimax_result_share.go` → **0 命中**，该 handler **没有**任何「超管密码错误」文案 |
| 响应含 `share_url` | `minimax_result_share.go:402-427` → `:424` `fmt.Sprintf("%s/minimax/share/%s", origin, share.ID)`；公开页路由 `frontend/src/router/index.js:662` |
| `display_type` 计算 | `minimax_result_share.go:541-549`：仅 `result_type == "media"` 时按资产类型特判；`image_understanding` 原样返回 |
| 公开列表 `type` 过滤 | `minimax_result_share.go:216` 取 `type` 查询参数；`minimaxResultShareMatchesType`（`:567-581`）先按 `result_type`/`display_type` 精确匹配、**再**按 `asset_kinds` 匹配 |

### 前端

| 事实 | 位置 |
|------|------|
| 语音面板**零**分享代码 | `frontend/src/components/AIGatewaySpeechPanel.vue`（1171 行，`grep -n "share\|Share"` → **0 命中**） |
| 面板挂载点（2 处） | `frontend/src/views/ai/MiniMaxStudioTool.vue:165`（语音/音色 tab）、`frontend/src/views/ai/AIGatewayTool.vue:532` |
| 面板内 TTS 结果 | 同步音频块 `AIGatewaySpeechPanel.vue:80-83`（`syncAudioUrl`）；异步音频块 `:126-129`（`asyncAudioUrl`）；均为 **blob URL**（`:721` `URL.createObjectURL(blob)`、`:945` `URL.createObjectURL(blob)`、`:944` `hexToBlob`） |
| 面板凭证 | `AIGatewaySpeechPanel.vue:410-412` `hasAnyCredential()`、`:414-420` `requireCredential()`、`:422-436` `jsonHeaders()` / `:438-450` `authHeaders()`（Bearer 优先，回退 `X-Super-Admin-Password`） |
| 面板已有 MIME 推断 | `AIGatewaySpeechPanel.vue:1038-1049` `guessAudioMime(format)`（mp3/wav/flac/pcm） |
| 同步响应原样落 ref | `AIGatewaySpeechPanel.vue:343` `syncResponse`（赋值点 `:554 syncResponse.value = data`），内容是 `POST /api/minimax/speech/v1/t2a_v2` 的**完整上游响应体**（后端 `backend/handlers/minimax_speech.go:142 writeMiniMaxResponse(c, statusCode, respHeader, respBody)` 原样透传），音频为内联 hex（面板自取 `:941 data.audio_file \|\| data.audio`） |
| 既有分享对话框（父组件） | `MiniMaxStudioTool.vue:826-852`（`el-dialog v-model="createShareVisible"`） |
| 既有状态 / 打开 / 提交 | `MiniMaxStudioTool.vue:1028`（`createShareVisible`）/ `:1029`（`createShareSubmitting`）/ `:1031-1039`（`shareDraft`）/ `:1040-1043`（`shareForm`）→ `:2143-2154` `openShareDialog(source)` → `:2156-2276` `buildShareDraft(source)` → `:2278-2309` `submitResultShare()`（POST body `{title,summary,result_type,model,payload,assets}`，`:2285-2292`；成功后 `:2298` 切 archive tab） |
| 对话框渲染 payload | `MiniMaxStudioTool.vue:1173` `shareDraftPretty = computed(() => pretty(shareDraft.value.payload))` → 模板 `:845` `<pre class="result-json">{{ shareDraftPretty }}</pre>`（**payload 越大，弹窗越卡**） |
| 类型中文映射 | `MiniMaxStudioTool.vue:2455-2467` `shareTypeLabel()`：含 `speech:'语音'`（`:2458`），**缺** `image_understanding` |
| 父组件凭证来源 | `MiniMaxStudioTool.vue:919` `SUPER_ADMIN_KEY = 'minimax_studio_super_admin_password'` / `:920` `API_KEY_STORAGE = 'minimax_studio_api_key'`；`:925-926` 读取、`:1267-1268` 写回、`:1279-1280` 清除 |
| 图像理解页 | `frontend/src/views/ai/ImageUnderstandingTool.vue`（1109 行，`grep -n 分享` → **0 命中**），**独立路由** `/image-understanding`（`frontend/src/router/index.js:443-451`），**未被** MiniMaxStudioTool 引用（`grep -rn "ImageUnderstandingTool" frontend/src/` → 仅 `router/index.js:445` 一处）；该页**没有** AI Gateway 凭证，只有流水用的 `X-Image-Admin-Password`（`:564`），与本接口两种鉴权**都不匹配** → 必须读同源 localStorage |
| 图像理解页 MCP 结果 | `ImageUnderstandingTool.vue:88-94` 结果卡片头（`:92` 复制文本按钮）；状态 `resultText` `:315`、`rawResult` `:316`、`selectedTool` `:306`、`prompt` `:307`、`imageBase64` `:308`（**data URL**）、`imageFile` `:310` |
| 公开分享页类型映射 | `frontend/src/views/share/MiniMaxResultShareView.vue:192-195`（缺 `image_understanding`）；同页 `:98` `<pre class="payload-block">{{ prettyPayload }}</pre>` + `:116` `prettyPayload` computed（**payload 原样渲染**）；画廊 `frontend/src/views/share/MediaGallery.vue:426-429`（缺）、筛选项 `MediaGallery.vue:228-233`（只有 全部/音乐/视频/图片） |

**复核结论**：PM 描述的 endpoint、handler、契约、localStorage 键、`shareTypeLabel` 位置**全部与代码一致**；两处需补：① 资产 `data_url` 分支**忽略**请求里的 `content_type`（`minimax_result_share.go:754-775`），故前端必须保证 data URL 的 MIME 头正确并显式给 `kind`；② 后端**不校验** `title`/`summary` 长度（`:139-140`），长度保护只能由前端做。

## 设计

### 0. 共用：draft 形状与字段口径

draft 是子组件 → 父组件 / 页面内部传递的唯一结构，形状与 `MiniMaxStudioTool.vue:1031-1039` 的 `shareDraft` **逐字段对齐**（1:1 映射到 POST body，`:2285-2292`）：

| draft 字段 | 类型 | 语义 | 落到 POST 的字段 |
|-----------|------|------|----------------|
| `sourceLabel` | string | 对话框「来源」行 | 不上传（仅 UI 展示，`:840`） |
| `title` | string | 分享标题，**≤120** | `title` |
| `summary` | string | 分享摘要，**≤240** | `summary` |
| `resultType` | string | 小写结果类型 | `result_type` |
| `model` | string | 模型 / 工具名 | `model` |
| `payload` | object | 原始结果 | `payload` |
| `assets` | array | 资产数组 | `assets` |

资产对象字段：`{ url?, data_url?, filename?, content_type?, kind }`，其中 `kind` 必填（`minimax_result_share.go:681-684` 优先用 `kind`，缺失才按 MIME/文件名推断，`audio`/`image`/`video`/`file`）。**R6 铁律**：blob URL **绝不**进 draft —— blob → data URL；本来就是 http(s) 的 → 走 `url` 字段让后端下载。

**长度铁律（NB4）**：draft 的 `title` ≤ **120**、`summary` ≤ **240**，**硬截断**（`.slice(0, 120)` / `.slice(0, 240)`，UTF-16 码元口径，与对话框 `el-input maxlength="120"/"240"`（`MiniMaxStudioTool.vue:831/834`）一致）。两条路径都必须在**构造 draft 时**截断：(A) 路径对话框虽有 maxlength，但 `shareForm` 的初值来自 draft，程序化赋值不受 maxlength 约束；(B) 路径完全绕过对话框。后端只 `TrimSpace` 不校验长度（`minimax_result_share.go:139-140`）。

### 1. (A) TTS —— 子组件 emit draft → 父组件既有分享对话框

#### 1.1 路径（PM 裁决 6① 写死）

**不新建对话框、不改 `submitResultShare`、不改契约。**

```
AIGatewaySpeechPanel.shareSyncResult()/shareAsyncResult()
  → emit('share', draft)
  → MiniMaxStudioTool.handleShareFromChild(draft)   ← 新增
  → MiniMaxStudioTool.openShareWithDraft(draft)      ← 新增，复用 shareDraft/shareForm/createShareVisible
  → 既有 el-dialog（MiniMaxStudioTool.vue:826-852）弹窗
  → 既有 submitResultShare()（:2278）POST /api/minimax/result-shares
```

#### 1.2 `frontend/src/components/AIGatewaySpeechPanel.vue` 改动清单

| # | 位置（现状行号） | 改动 |
|---|------------------|------|
| A1 | `:291-300` `defineProps` | 新增 prop `shareEnabled: { type: Boolean, default: false }`。**理由**：本组件被 `AIGatewayTool.vue:532` 复用，那里没有分享对话框；不加门控会出现「点了没反应」的死按钮。 |
| A2 | `:300` 之后 | 新增 `const emit = defineEmits(['share'])` |
| A3 | `:83` 之后（同步音频块 `:80-83` 内） | 插入 `<el-button v-if="shareEnabled" type="primary" size="small" :loading="sharingSync" @click="shareSyncResult">保存分享</el-button>` |
| A4 | `:129` 之后（异步音频块 `:126-129` 内） | 插入 `<el-button v-if="shareEnabled" type="primary" size="small" :loading="sharingAsync" @click="shareAsyncResult">保存分享</el-button>` |
| A5 | `:341` 之后 | 新增 `const sharingSync = ref(false)` / `const sharingAsync = ref(false)` |
| A6 | `:1056` 之后（`hexToBlob` 之后） | 新增 `blobUrlToDataUrl(blobUrl, mime)`（见 1.3）+ `stripInlineAudio(resp)`（见 1.4） |
| A7 | A6 之后 | 新增 `shareSyncResult()` / `shareAsyncResult()`（见 1.4） |

> 面板 loading 只覆盖「blob → data URL 转换」这段；真正的 POST 由父组件的 `createShareSubmitting`（`:1029`）负责。**R3 / R4**：模板段为 1–285 行（`<script setup>` 在 `:287`），模板里**不得**出现 `.value`、不得出现裸下标 `obj.x[0]`。

#### 1.3 `blobUrlToDataUrl(blobUrl, mime)` 口径

- **mime 实参写死（U2）**，三条路径各自取法如下，**不得留空**：

  | 调用路径 | mime 实参 | filename 扩展名 | 依据 |
  |----------|-----------|----------------|------|
  | 同步合成（内联 hex → `syncAudioUrl`） | `guessAudioMime(syncForm.format)` | `.${syncForm.format}`（mp3/wav/flac/pcm） | `:944` `hexToBlob(audioHex, guessAudioMime(syncForm.format))` |
  | 同步合成走 file_id 预览（`filePreviewUrl`，`:348` 声明 / `:703` 赋值） | `guessAudioMime(syncForm.format)` | `.${syncForm.format}` | **与内联 hex 同源**（PM 2026-10-22 02:35 修正拍板）：`syncForm.format` 始终在作用域内（`:351-358`），同步请求已把该格式发给上游（`audio_setting.format`，`:544`），file_id 是**同一响应**里的文件（`:948-955`）→ 字节格式就是 `syncForm.format` |
  | 异步任务产物（`asyncAudioUrl`） | **固定 `audio/mpeg`** | 固定 `.mp3` | 异步任务产物固定 mp3（PM 裁决 U2） |

  `guessAudioMime`（`:1038-1049`）返回 mp3→`audio/mpeg`、wav→`audio/wav`、flac→`audio/flac`、pcm→`audio/pcm`，**扩展名必须与 MIME 一致**。

- 输入 `blob:` URL → `fetch(blobUrl)` → `blob` → **用上面写死的 MIME 重新包装** `new Blob([blob], { type: mime })` → `FileReader.readAsDataURL` → `data:<mime>;base64,...`。
  - 重新包装的原因：**异步任务音频**来自 `res.blob()`（`:721`）、**同步 file_id 预览音频**来自 `res.blob()`（`:702`），两者 MIME 都取自后端代理的响应头、**可能是 `application/octet-stream`**；不重包则 data URL 头 MIME 错 → 后端把它当 `content_type` 落库（`minimax_result_share.go:763-766`）并按它回放响应头（`:204-205`）→ 公开分享页的 `<audio>`（`MiniMaxResultShareView.vue:76`）解不出来 → **T2 挂**。**重包必须用正确的 mime**（同步两条路径都跟随 `syncForm.format`），不能用猜的。同步内联 hex 路径的 `hexToBlob` 已带正确 MIME（`:944`）。
- 输入 `http(s)://` URL → **不转**，返回 `null`，调用方改用资产 `url` 字段（后端服务端下载，避免大 body）。**协议约束（U3）**：走 `url` 分支的 URL **必须满足**① 以 `https:` 开头（**不接受 `http:`**）；② **来自上游 MiniMax 响应**（同步路径即 `syncResponse.data.audio_url`），**不得来自用户输入、不得由前端拼接**；不满足则按 blob 分支处理或丢弃该资产。**SSRF 面说明（NB5）**：后端该分支由**服务端**下载（`minimax_result_share.go:697-731`，`h.noProxyClient.Do`，**无 host 白名单**），属**既有端点能力**、**本方案零后端改动**、**不新增调用方**；但本方案是**首次由浏览器流程触发**该分支，故前端侧的来源与协议约束（上文①②）必须写死并照做。
- **判定顺序（U9，写死）**：① **先判 url 分支** —— 上游 `data.audio_url` 是 `https://` 直链时，资产走 `url` 字段（后端服务端下载，**不进请求体、不受 20MB 闸门约束**）；② **只有走 blob → data URL 分支时**才判体积闸门：原始字节 > **20MB** → 抛 `Error('音频过大，无法直接保存分享')`，**不静默降级**。**闸门口径 = 原始音频字节**（U1：不改成按最终请求体计）。url 分支仍**只收 `https:`**（U3 不变）。
- **两条路径的超限语义不一致是刻意的（U11）**：(A) 超 20MB → **抛错、整次保存失败**（此时没有 url 分支可退，音频就是唯一产物）；(B) 超 10MB → **降级为「只分享文本」**（`assets: []`，**不失败**；识别文本本身已完整）。**不要把两边改成一致**。
- **实现形态要求（供静态检查 11c 判定）**：同步两条路径（内联 hex / file_id 预览）的产物都落在 `syncAudioUrl`（file_id 分支 `:951` 把它赋成 `filePreviewUrl`）→ **共用同一个 blob 调用点**、mime 实参**只写一次** `guessAudioMime(syncForm.format)`；异步是**另一个调用点**（`asyncAudioUrl` + 字面量 `'audio/mpeg'`）。**同步与异步不得合并成同一个调用点**，否则静态检查无法区分两条路径。
- 失败一律 `try/catch` → `ElMessage.error`，`finally` 复位 loading。

#### 1.4 draft 字段（逐字段写死）

**(A-1) 同步合成** `shareSyncResult()`：

| 字段 | 取值 |
|------|------|
| `sourceLabel` | `'语音合成'` |
| `title` | `` `TTS · ${syncForm.text.trim().slice(0, 36)}` ``（`syncForm` 定义于 `:351-358`；文本为空时退化为 `'TTS'`），**再按 ≤120 截断** |
| `summary` | `` `${syncForm.model} 同步合成 · 音色 ${syncForm.voiceId}` ``，**按 ≤240 截断** |
| `resultType` | `'speech'`（`shareTypeLabel` 已有 `speech:'语音'`，`:2458`） |
| `model` | `syncForm.model` |
| `payload` | `{ text: syncForm.text, voice_id: syncForm.voiceId, speed: syncForm.speed, format: syncForm.format, sample_rate: syncForm.sampleRate, mode: 'sync', raw: stripInlineAudio(syncResponse) }`（`syncResponse` = `:343`） |
| `assets` | 1 个：`{ data_url \| url, kind: 'audio', filename: \`tts-${syncForm.model}.${syncForm.format}\` }` —— **文件名统一跟随 format**；同步走 file_id 预览路径时**同样**是 `tts-${model}.${syncForm.format}`，**不再有固定 `.mp3` 特例** |

> **`raw` 必须剔除内联音频（U1 / B1）**：`syncResponse` 是上游 `t2a_v2` 的完整响应体，音频是**内联 hex 字符串**（`data.audio` / `data.audio_file`）。若原样塞进 `payload.raw`：① 分享对话框 `shareDraftPretty`（`MiniMaxStudioTool.vue:1173` → 模板 `:845`）会把整段 hex 渲染进 DOM → 20MB 音频 ≈ 40MB 文本节点 + `JSON.stringify(v,null,2)` 全量格式化 → **弹窗当场卡死**；② 请求体膨胀约 3.3×（base64 1.33× + hex 2×）→ body ≈ 67MB，逼近 CF 单请求体 100MB 上限；③ `share.Payload = string(req.Payload)`（`minimax_result_share.go:152`）原样入库 → 公开分享页 `MiniMaxResultShareView.vue:98/:116` 要下载并渲染几十 MB JSON → **T2 的「页面含 audio 播放器且可播放」实际打不开**。
> 剔除后的口径：**只删内联音频本体，其余字段原样保留**（`file_id` / `trace_id` / `base_resp` / `status` 等元信息一律保留）；`resp.data` 不存在（或不是对象）时 `raw` 保持 `resp` 原样（只去掉顶层 `audio`/`audio_file`）—— t2a_v2 实际只返回 `data.audio`（hex）与 `data.status`，故上述键集已覆盖全部内联音频字段。实现（可直接抄）：

```js
function stripInlineAudio(resp) {
  if (!resp || typeof resp !== 'object') return resp
  const out = { ...resp }
  delete out.audio
  delete out.audio_file
  if (out.data && typeof out.data === 'object') {
    const data = { ...out.data }
    delete data.audio
    delete data.audio_file
    out.data = data
  }
  return out
}
```

**(A-2) 异步合成** `shareAsyncResult()`：

| 字段 | 取值 |
|------|------|
| `sourceLabel` | `'语音合成'` |
| `title` | `` `TTS · ${inputText.trim().slice(0, 36)}` ``，`inputText` = `asyncForm.mode === 'text' ? asyncForm.text : asyncForm.textFileId`（`asyncForm` 定义于 `:360-366`），**再按 ≤120 截断** |
| `summary` | `` `${asyncForm.model} 异步合成 · 音色 ${asyncForm.voiceId} · 任务 ${taskId}` ``，`taskId` 取自 `currentTask`（`:345`，`extractTaskId` 见 `:996`），**按 ≤240 截断** |
| `resultType` | `'speech'` |
| `model` | `asyncForm.model` |
| `payload` | `{ text: inputText, voice_id: asyncForm.voiceId, mode: asyncForm.mode, text_file_id?: asyncForm.textFileId, task_id: taskId, raw: currentTask }` |
| `assets` | 1 个：`{ data_url, kind: 'audio', filename: \`tts-${asyncForm.model}.mp3\` }`（异步任务产物固定 mp3） |

> **异步路径不受 U1 影响**：`currentTask`（`:345`）只含 `file_id` 等任务元信息、**不含内联音频**，故 `raw: currentTask` 原样保留，**不改**。提交前守卫：同步 `if (!syncAudioUrl.value || !hasAnyCredential()) return`、异步 `if (!asyncAudioUrl.value || !hasAnyCredential()) return`（`hasAnyCredential` = `:410`）。

#### 1.5 `frontend/src/views/ai/MiniMaxStudioTool.vue` 改动清单

| # | 位置 | 改动 |
|---|------|------|
| M1 | `:165` | `<AIGatewaySpeechPanel :super-admin-password="superAdminPassword" :prefill-api-key="apiKey" share-enabled @share="handleShareFromChild" />` |
| M2 | `:2154` 之后（`openShareDialog` 之下） | 新增 `handleShareFromChild(draft)` 与 `openShareWithDraft(draft)`；`openShareDialog`（`:2143-2154`）改为调用 `openShareWithDraft(buildShareDraft(source))`，消除重复赋值，**且必须保留 `requireCredential()` 前置与 `try/catch`**（NB1） |

**重构后的完整三段（实现方照抄即可，NB1）**：

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

- **为什么不能只写后两段**：`buildShareDraft(source)` 会 `throw new Error('当前没有文本结果')`（`:2158-2160`）等异常，现状 `openShareDialog` 用 `try/catch` 兜住并 `ElMessage.error`；丢掉 `try/catch` 会让异常变成未捕获 → **回归项 T10（既有 5 个分享入口）直接挂**。`handleShareFromChild` 走同一套 `requireCredential()` 前置。
- `requireCredential()`（`:1288`）用**父组件**凭证（`apiKey` / `superAdminPassword`），因为真正发 POST 的是父组件 `submitResultShare`（`jsonHeaders()` 走父组件凭证，`:1306-1314`）；子组件面板里的调试 Key 与父组件不一致时以父组件为准。弹窗「类型」行显示 `shareDraft.resultType`（`:841`）= `'speech'`，**不改**（既有行为：既有 5 个分享入口同样直接打原始枚举；`shareTypeLabel`（`:2455`）只在别处使用）。

### 2. (B) 图像理解 —— 独立页直接 POST

#### 2.1 路径（PM 裁决 5）

`ImageUnderstandingTool.vue` 是**独立路由页**（`router/index.js:443-451`），**没有父组件**承接 emit（PM 已核实 main 已把它从 MiniMaxStudioTool 里拆出）→ **不新建分享对话框**，页面内直接 POST `/api/minimax/result-shares`。凭证：读**同源** localStorage 已有键（不另造键）：

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

> **只加在 MiniMax MCP 标签页**（`:92` 所在卡片）。「大模型理解 / M3」标签页（`:215-224`）**不加**（PM 裁决 2：不移植 Qwen 分享入口）。

#### 2.3 `shareMcpResult()` draft 字段（逐字段写死）

| 字段 | 取值 |
|------|------|
| `sourceLabel` | `'图像理解'` |
| `title` | `` `图像理解 · ${(prompt.value || '').trim().slice(0, 36)}` ``；`prompt` 为空 → `'图像理解'`；**再按 ≤120 硬截断**（NB4） |
| `summary` | `` `MiniMax MCP 图像理解 · 工具 ${selectedTool.value \|\| 'MiniMax-MCP'}` ``；**再按 ≤240 硬截断**（NB4） |
| `resultType` | `'image_understanding'`（**新的 result_type 取值**；该字符串在 main 已存在，但只用作 config 键 `backend/config/config.go:43`（`yaml:"image_understanding"`）与 `backend/handlers/ai_gateway_image.go:393` 的**监控 / 日志记录字段**（`h.logAPIRequest` 的场景参数，`:390-396`；同段的实际路径字符串是 `:392 "/api/ai-gateway/v1/image/understanding"`，**不是路由组名**），与本域无关。后端对 `result_type` 无白名单，只小写化 + 非空校验 `minimax_result_share.go:122-125`，唯一特判是 `display_type` 对 `'media'` 的处理 `:543`） |
| `model` | `selectedTool.value \|\| 'MiniMax-MCP'`（**fallback 字面量写死**，U6：工具列表未加载 / 用户未选时就是 `'MiniMax-MCP'`；后端 `model` 字段只 `TrimSpace` 不校验、非必填） |
| `payload` | `{ text: resultText.value, prompt: prompt.value, tool: selectedTool.value, source: 'minimax-mcp', raw: stripInlineImage(rawResult.value, imageBase64.value) }` —— `raw` **必须剔除图片 base64**（**U10**，与 (A) 的 U1 `stripInlineAudio` **同构**：上游 result 若回显 base64，会与 `assets` 里的 data URL 重复 → body 翻倍）。**U12（保持现状，不是缺陷）**：`rawResult.value` 是 `JSON.stringify(...)` 产生的 **JSON 字符串**（`ImageUnderstandingTool.vue:685` / `:758`），**不是对象** —— 与既有 text/media 分享口径**一致**（U7 已定不脱敏），公开页 `prettyPayload` 会把它转义成一坨转义字符串，**这是记录口径、不改**。 |
| `assets` | **0 或 1 个**：**无图时一律 `assets: []`（U8）**，**禁止** `{ data_url: null }` —— 后端 `minimax_result_share.go:701-703` 会返回 400「资产缺少 url 或 data_url」，**整次保存失败**（本意只是没图）；有图时 1 个：`{ data_url: imageBase64.value, kind: 'image', filename: imageFile.value?.name \|\| 'image-understanding.png' }`。`imageBase64` 已是 `data:image/...;base64,`（`:308`，由 `handleFileChange` 的 `FileReader.readAsDataURL` 写入，`:409-417`）→ **天然满足 R6**（不是 blob URL）。原始图片 > **10MB** 时**同样不带资产**（`assets: []`），只分享文本结果并提示「图片过大，仅保存识别文本」（U11：降级**不失败**）。 |

> **10MB 闸门的度量口径（U4 / NB3）**：一律用**原始文件字节** `imageFile.value?.size`（`:310` 声明、`:414` 赋值 `imageFile.value = file.raw`）判 10MB。**不用** data URL 解码后字节 —— base64 会膨胀 4/3，两者不等。
> **(B) 的 body 规模口径（U10，写死具体数字）**：10MB 原始图 → base64 `data_url` ≈ **13.3MB**（×1.33）＋ `payload.raw` 残余 **< 1MB**（图片 base64 已剔除）→ **整包 ≤ ≈14MB**，远低于 CF 单请求体上限 100MB 与后端 200MB（`backend/app_http.go:35`）。

> **`model` 与 `summary` 的 fallback 必须一致（U6）**：两处统一用 `'MiniMax-MCP'`（现方案 summary 写 `'auto'`、model 写 `'MiniMax-MCP'`，自相矛盾）；`payload.tool` 保留 `selectedTool.value` **原样**（未选工具时可能是空串），与 `model` 的兜底值**区分开**。**raw 不脱敏（U7）**：`payload.raw` 保留上游原始 JSON 的**全部非图片内容**（含上游 request id 等内部信息），与既有 text/media 分享的 `raw` 口径**一致**，这是**刻意选择**、非疏漏；唯一的加工是 **U10 剔除图片 base64** 与 **U12 保持 JSON 字符串形态**。

`stripInlineImage` 实现（可直接抄，U10）：
```js
function stripInlineImage(raw, imageBase64) {   // 剔除 raw 里回显的图片 base64（U10，与 U1 stripInlineAudio 同构）
  if (typeof raw !== 'string' || !raw) return raw
  const bare = String(imageBase64 || '').replace(/^data:[^,]*,/, '')
  return (bare ? raw.split(bare).join('[image omitted]') : raw).replace(/data:image\/[\w.+-]+;base64,[A-Za-z0-9+/=]+/g, '"[image omitted]"')
}
```

提交：`POST /api/minimax/result-shares`，headers = `Authorization: Bearer <apiKey>`（无 apiKey 时用 `X-Super-Admin-Password: <password>`）+ `Content-Type: application/json`，body = `{ title, summary, result_type: 'image_understanding', model, payload, assets }`。

- 成功（201）→ `ElMessage.success('图像理解结果已保存为分享页')` + `navigator.clipboard.writeText(data.share_url)` + `ElMessage.success('分享链接已复制')`（`data.share_url` 由 `minimax_result_share.go:424` 返回）。**不弹新对话框**。
- 失败 → `ElMessage.error(data.error || '保存分享失败')`（401 走这条，错误文案由后端给：`缺少 API Key` / `API Key 无效` / `API Key 格式错误` —— `minimax_result_share.go:351` / `:371` / `:355`；**后端没有任何超管密码专属错误文案**，超管密码不匹配时也会回落成上面这条）。
- **保存成功后不跳转（U5，刻意不统一）**：(B) 是**独立路由页**、**没有 archive tab 可切**，故只「复制链接 + toast」；(A) 路径保存后切 archive tab（`MiniMaxStudioTool.vue:2298`）是 MiniMax Studio 页内既有行为。两条路径行为**不一致是刻意为之，不是缺陷**，不强行统一。

### 3. 类型中文映射（PM 裁决 6②）

| 文件 | 位置 | 改动 |
|------|------|------|
| `frontend/src/views/ai/MiniMaxStudioTool.vue` | `:2455-2467` `shareTypeLabel()` 的 map | **必做**：补 `image_understanding: '图像理解'`（`speech:'语音'` 已存在，`:2458`） |
| `frontend/src/views/share/MiniMaxResultShareView.vue` | `:192-195` `typeLabel()` 的 map | **P2 附带**：补 `image_understanding: '图像理解'`，否则公开分享页标签直接显示英文枚举 |
| `frontend/src/views/share/MediaGallery.vue` | `:426-429` `typeLabel()` 的 map | **P2 附带**：同上，画廊封面标签 |

### 4. PM 裁决 U1–U13（口径汇总，实现方无条件遵守）

| 编号 | 裁决 | 落点 |
|------|------|------|
| U1 | **采用「raw 剔除内联音频」**（不改成按最终请求体计闸门）：`payload.raw = stripInlineAudio(syncResponse)`，只删内联音频本体，其余字段原样保留；20MB 闸门仍按**原始字节**计；body 口径写成具体数字；异步路径 `raw: currentTask` 不改 | §1.4 (A-1) payload 行 + `stripInlineAudio` 实现；§1.3 闸门；§后端改动清单；§风险表第 1 行 |
| U2 | **mime 实参写死三条**：同步内联 hex **与**同步 file_id 预览 → **同用** `guessAudioMime(syncForm.format)`（PM 2026-10-22 02:35 修正拍板，M1 依此闭合）；**只有异步产物** → 固定 `audio/mpeg`；扩展名与 MIME 一致 | §1.3 mime 实参表 |
| U3 | **保留 http(s) → 资产 `url` 分支**，但收紧：仅接受 `https:`、且 URL 必须来自上游 MiniMax 响应（`data.audio_url`），**不得来自用户输入**；后端 `url` 分支是既有端点能力、服务端下载、本方案零后端改动；T4 保留 | §1.3（协议/来源约束 + SSRF 面说明）；§1.4 assets；§测试要点 T4 |
| U4 | **图片 10MB 闸门按原始文件字节** `imageFile.value?.size`（不是 data URL 解码后字节） | §2.3 assets 行 + 口径说明 |
| U5 | **(B) 路径保存成功后不跳转**，只复制链接 + toast；(A) 路径切 archive tab 是页内既有行为 → **不统一、不算缺陷** | §2.3 提交说明 |
| U6 | **(B) 路径 `model` fallback 固定 `'MiniMax-MCP'`**，summary 与 model 统一用同一表达式；`payload.tool` 保留原值 | §2.3 summary / model / payload 行 + 口径说明 |
| U7 | **`selectedTool` / `rawResult` 不脱敏**，与既有 text/media 分享的 `raw` 口径一致 | §2.3 payload 行 + 口径说明 |
| U8 | **(B) 路径没有图片时 `assets` 传 `[]`（空数组）**，**禁止**塞 `{ data_url: null }` —— 后端 `minimax_result_share.go:701-703` 会返回 400「资产缺少 url 或 data_url」→ 整次保存失败（本意只是没图）。有图才放 data_url 资产 | §2.3 assets 行 + §测试要点 T5b |
| U9 | **(A) 先判 url 分支**：上游给了 `https://` 直链时走 `url` 资产（服务端下载，不受 20MB 限制）；**只有**走 blob → data URL 分支时才判 20MB 原始音频闸门，超限 → **抛错、不静默降级**。url 分支仍**只收 `https://`**（U3 不变） | §1.3 判定顺序 + §测试要点 T4 |
| U10 | **(B) 的 `payload.raw` 必须剔除图片 base64**（与 (A) 的 U1 `stripInlineAudio` 同构）；body 规模写死：10MB 原始图 → data_url ≈13.3MB → 整包 **≤ ≈14MB** | §2.3 payload 行 + body 规模口径 |
| U11 | **两条路径超限语义不一致是刻意的**：(A) 20MB 超限 → 抛错、整次保存失败（无 url 分支可退时）；(B) 10MB 超限 → **降级为「只分享文本」**（`assets: []`），**不失败** | §1.3 + §2.3 口径说明 |
| U12 | **(B) 的 `payload.raw` 是 JSON 字符串不是对象** → **保持现状**（与既有 text/media 分享口径一致，U7 已定不脱敏），**不是缺陷** | §2.3 payload 行（补一句说明，不改实现口径） |
| U13 | **T4 / T4b 改为构造性测试**（步骤语义不变）：T4 断言 https 直链走 `url` 分支且 mime 不写死 mp3；T4b 构造 `http://` 变体、断言前端拒绝该 url 并回退 data URL 路径（U3） | §测试要点 T4 / T4b |

### 后端改动清单

**无。** 逐条核对结论：路由已存在（`routes/ai_gateway.go:68`）、契约已支持 `data_url`（`minimax_result_share.go:35`）、鉴权两种方式齐备（`:334-383`）、资产上限满足本次用量（1 个资产 ≪ 12；TTS ≤ 20MB ≪ 120MB）。**body 规模口径（U1，写死具体数字，不写「≪」）**：20MB 原始音频 → base64 `data_url` ≈ **26.7MB**（×1.33）＋ payload 残余 < 1MB（`raw` 已剔除音频副本）→ **请求体 ≈ 27MB**，**远低于** CF 单请求体上限 100MB（免费档）与后端 200MB body 限额（`backend/app_http.go:35`）。

### 前端改动清单（汇总）

| 文件 | 改动点 | 条数 |
|------|--------|------|
| `frontend/src/components/AIGatewaySpeechPanel.vue` | A1–A7（含 `blobUrlToDataUrl` + `stripInlineAudio`） | 7 |
| `frontend/src/views/ai/MiniMaxStudioTool.vue` | M1–M2（含重构后三段完整代码）+ `shareTypeLabel` 补 map | 3 |
| `frontend/src/views/ai/ImageUnderstandingTool.vue` | I1–I4 | 4 |
| `frontend/src/views/share/MiniMaxResultShareView.vue` | P2 `typeLabel` 补 map | 1 |
| `frontend/src/views/share/MediaGallery.vue` | P2 `typeLabel` 补 map | 1 |

**合计 16 个改动点**（必做 14 + P2 附带 2）。零后端改动、零路由改动、零 DB 改动。

## 兼容性与迁移

- **无接口变更**、无表结构变更、无数据回填。纯前端新增按钮与新函数；老分享记录不受影响（`result_type` 新取值 `image_understanding` 之前不存在；已有分享的 `result_type` 值不变，`display_type` 计算逻辑不变 —— `minimax_result_share.go:541-549` 只对 `media` 特判）。
- 老页面不受影响：`shareTypeLabel` / `typeLabel` 只是 map 增项，命中不到时走 `map[type] || type` 兜底。**既有 v1 方案未落地**（Stage 2 判 FAIL 后才回改），故无「v1 语义需要迁移」的存量问题；`raw` 剔除内联音频（U1）/ 剔除图片 base64（U10）的口径**只影响新增分享**，不改既有记录。
- 已知局限（不在本次范围）：画廊筛选项只有 全部/音乐/视频/图片（`MediaGallery.vue:228-233`）；后端 `minimaxResultShareMatchesType`（`minimax_result_share.go:567-581`）先按 `result_type`/`display_type` 精确匹配、**再**按 `asset_kinds` 匹配（`:575-579`）→ 带图片资产的分享会落在「图片」筛选下，没带图片资产的只出现在「全部」里。详见末节「本次不做」第 2 条。

## 风险与回滚

| 风险 | 概率 | 影响 | 处置 |
|------|------|------|--------|
| 同步 TTS 音频很大，data URL 撑爆请求体 | 低 | 保存失败 | 20MB 闸门（§1.3，按原始字节计、**仅 blob 分支**，url 分支不受约束 —— U9）；`payload.raw` 剔除内联音频（U1）→ 不产生 hex 副本；body ≈ 1.35× 闸门上限（20MB → ≈27MB），远小于 CF 单请求 100MB 与后端 200MB（`app_http.go:35`）→ 对话框与公开分享页的渲染量与音频大小无关 |
| 音频 MIME 取自后端代理响应头（可能是 `application/octet-stream`）→ 分享页 `<audio>` 解不出来 | 中 | T2 挂 | §1.3 强制重包 MIME：**同步两条路径跟随 `guessAudioMime(syncForm.format)`、异步固定 `audio/mpeg`**（M1 口径）；且 draft 显式给 `kind:'audio'`（后端优先用 `kind`，`:681-684`），后端按 `content_type` 回放响应头（`:204-205`） |
| 资产 `url` 分支由后端服务端下载且**无 host 白名单**（SSRF 面） | 低 | 服务端被请求任意 URL | 该分支是**既有端点能力**（`minimax_result_share.go:697-731`）、本方案零后端改动、不新增调用方；前端侧收紧（U3）：**只接受 `https:` 且必须来自上游 MiniMax 响应的 `data.audio_url`**，不得来自用户输入 |
| 图像理解页没填过 MiniMax Studio 凭证 → 点保存分享报「缺少 API Key」 | 中 | 用户困惑 | 按钮点击时先检查两个 localStorage 键，缺则提示「请先在 MiniMax 工作室填写 AI Gateway API Key 或超级管理员密码」并**不发请求** |
| 面板被 `/ai-gateway` 页复用 → 出现点不动的死按钮 | 中 | 体验降级 | A1 的 `shareEnabled` 默认 false 门控 |
| 语音面板 share 事件被父组件吞掉（父无凭证） | 低 | 无提示 | `handleShareFromChild` 调 `requireCredential()`（`:1288`）弹错误 |
| 重构 `openShareDialog` 时丢掉 `try/catch` → 既有 5 个分享入口报未捕获异常 | 中 | 回归 | §1.5 给出**完整三段**（含 `requireCredential()` + `try/catch`），实现方照抄；回归项 T10 验收 |
| 分享链接在 CF 后 origin 错位 | 低 | 链接打不开 | `share_url` 由后端按 `X-Forwarded-Proto/Host` 拼（`:402-403`、`:807-810`），沿用既有行为 |

**回滚**：全部改动集中在 3（+2）个前端文件、无后端/DB 变更 → `git revert` 单个 commit 即可完全回滚；无残留数据（分享记录留在 `minimax_result_shares` 表，与现有行为一致，删除走既有 `DELETE /api/minimax/result-shares/admin/:id`）。

## 测试要点（交给 devtools-qa，全部可判定）

### 静态检查（可脚本化，PASS/FAIL 二值）

1. 后端零改动：`git diff origin/main --name-only -- backend/` 输出为空。
2. `grep -c "openShareWithDraft" frontend/src/views/ai/MiniMaxStudioTool.vue` ≥ 2（定义 + 至少一处调用）。
3. `grep -n "image_understanding" frontend/src/views/ai/MiniMaxStudioTool.vue | grep "图像理解"` 命中 1 行（N3：放宽引号 / 空格写法差异）。
4. `grep -n "shareEnabled" frontend/src/components/AIGatewaySpeechPanel.vue` 命中 ≥ 3（prop 声明 + 两处模板 `v-if`）。
5. **R3**：`awk 'NR<287 && /\.value/ {print NR": "$0}' frontend/src/components/AIGatewaySpeechPanel.vue` 输出为空（模板段 1–286 行不得出现 `.value`）。
6. **R4**：`awk 'NR<287 && /\[0\]|\[[0-9]+\]/ {print NR": "$0}' frontend/src/components/AIGatewaySpeechPanel.vue` 输出为空。
7. `grep -n "blob:" frontend/src/views/ai/MiniMaxStudioTool.vue frontend/src/views/ai/ImageUnderstandingTool.vue` 输出为空（不得把 blob URL 写进 draft）。
8. `grep -rn "console\.log\|TODO\|FIXME" frontend/src/components/AIGatewaySpeechPanel.vue frontend/src/views/ai/MiniMaxStudioTool.vue frontend/src/views/ai/ImageUnderstandingTool.vue` 只允许命中存量行，且**不得**出现在本次 diff 范围内（用 `git diff origin/main -- frontend/ | grep -n "console.log\|TODO\|FIXME"` 判定为空）。
9. **U1 防回退**：`grep -n "raw: syncResponse" frontend/src/components/AIGatewaySpeechPanel.vue frontend/src/views/ai/MiniMaxStudioTool.vue` 输出为空（不得把上游响应原样塞进 payload）。
10. **U1 落实**：`grep -n "stripInlineAudio" frontend/src/components/AIGatewaySpeechPanel.vue` 命中 ≥ 1（函数定义），且 (A-1) payload 行含 `stripInlineAudio(syncResponse)`。
11. **U2 落实 + M1 防回退（**有鉴别力**；替换原第 11 条恒真闸门）**：
    - **11a / 11b（正反两向，文档级，现在就能跑）**：`git show <ref>:docs/plans/tts-image-result-share.md | grep "^[[:space:]]*| 同步合成走 file_id 预览" | grep -c "<PATTERN>"`（锚定 §1.3 mime 表的那一行，不含本条自述行）→ `<PATTERN>` = `guessAudioMime(syncForm.format)` 时：**修复分支 ≥ 1**、`origin/main`（该文件在基线上不存在）**0**、`27ae744`（被 FAIL 版，写死 `audio/mpeg`）**0**；`<PATTERN>` = `audio/mpeg` 时：**修复分支 0**、`27ae744` **1** → 这组 0/1 证明该断言**有牙齿**、不是恒真闸门。
    - **11c（实现侧，Stage 3 完成后由 QA 跑）**：`grep -n "blobUrlToDataUrl(" frontend/src/components/AIGatewaySpeechPanel.vue` → **同步调用点**（含 `syncAudioUrl` 的那行）必须含 `guessAudioMime(syncForm.format)` 且**不得**含 `'audio/mpeg'`；**异步调用点**（含 `asyncAudioUrl` 的那行）才允许 `'audio/mpeg'`；`guessAudioMime` 函数体 `:1047` 的 default 分支不算。
12. **U4 落实**：`grep -n "imageFile.value?.size\|imageFile\.value\.size" frontend/src/views/ai/ImageUnderstandingTool.vue` 命中 ≥ 1（闸门按原始文件字节）。
13. **B2 文案**：`grep -rn "密码不匹配" backend/handlers/minimax_result_share.go` 输出为空（后端无超管密码错误文案）；`grep -rn "缺少 API Key" backend/handlers/minimax_result_share.go` 命中 ≥ 1（T7a 的 401 文案来自**后端实现**，N2：不再拿设计文档自身当断言对象）。

### 手工路径（浏览器，MiniMax Studio + 图像理解页）

| 编号 | 步骤 | 期望 |
|------|------|------|
| T1 | `/minimax-studio` → 语音/音色 → 同步合成 → 点「保存分享」 | 弹窗打开；「来源」= 语音合成；标题 = `TTS · <文本前 36 字>`；摘要含模型与音色；资产数 = 1；弹窗「原始结果」JSON **不含** `audio`/`audio_file` 音频字段（U1） |
| T2 | T1 弹窗点「保存分享」 | toast「MiniMax 结果已保存为分享页」；新标签打开 `https://t.jaxiu.cn/minimax/share/<id>`，页面含 **audio 播放器**且可播放（公开页 JSON 体积 ≪ 音频大小，不卡死）；管理列表出现该条，类型标签显示「语音」 |
| T3 | 异步：创建任务 → 轮询出音频 → 保存分享 → 保存 | 同 T2，摘要含任务号；分享页 payload 里 `task_id` 存在；`raw` 中无内联音频 |
| T4 | **构造性测试**（U13）：让面板拿到上游 `data.audio_url` 的 **https 直链**（打桩 `setAudioFromSpeechResponse` 的直链分支 / devtools override 返回 `{data:{audio_url:'https://…'}}`）→ 保存分享 | 断言 `assets[]` 走 **`url` 分支**（`assets[0].url` 有值、`data_url` 为空）；**mime 不写死 mp3**（同步路径实参来自 `guessAudioMime(syncForm.format)`）；请求体**不含**音频 base64；**不受 20MB 闸门约束**（U9） |
| T4b | **构造性测试**（U13）：把 `data.audio_url` 打成 `http://`（非 https）变体 → 保存分享 | 断言前端**拒绝**该 url（不产生 `assets[0].url`）并**回退 data URL 路径**（mime 仍为 `guessAudioMime(syncForm.format)`）或按口径丢弃该资产（U3：仅接受 `https:` 且必须来自上游响应） |
| T5 | `/image-understanding` → MiniMax MCP → 上传图 → 识别 → 点「保存分享」 | toast「图像理解结果已保存为分享页」+「分享链接已复制」；剪贴板内容为 `https://t.jaxiu.cn/minimax/share/<id>`；打开该页可见识别文本 + 输入图片；类型标签「图像理解」；**页面不跳转**（U5，仍停在 `/image-understanding`） |
| T5b | **构造性测试**（U8）：图像理解页**不传图片**（清空图片 / 只给纯文本提示）→ 识别 → 点「保存分享」 | 断言请求体 `assets` 为 **`[]`**（**不是** `[{data_url:null}]`）→ 后端**不返回 400**、保存成功；分享页有识别文本、无图片资产 |
| T6 | 同 T5，但**不**在 MiniMax Studio 存过凭证（清 `minimax_studio_api_key` / `minimax_studio_super_admin_password` 后刷新） | 提示「请先在 MiniMax 工作室填写…」，**DevTools Network 无** `POST /api/minimax/result-shares` 请求 |
| **T7a** | **清空 `minimax_studio_api_key`**，只保留**错误**的 `minimax_studio_super_admin_password` → T5 | 请求**确实发出**，服务端 **401**，页面提示后端文案**「缺少 API Key」**，**无 500**（超管密码不匹配时后端回落到 `authenticateAnyAPIKey`，无 Bearer → 401 `缺少 API Key`；后端无「超管密码错误」这类专属文案） |
| **T7b** | 保留**正确**的 `minimax_studio_api_key` + **错误**的 `minimax_studio_super_admin_password` → T5 | **201 成功**（Bearer 优先），剪贴板拿到分享链接 —— 证明鉴权优先级链没写反 |
| T8 | `/ai-gateway` 页的语音面板 | **不出现**「保存分享」按钮（`shareEnabled` 默认 false） |
| T9 | `/image-understanding` 的「大模型理解 / M3」标签页 | **不出现**「保存分享」按钮（PM 裁决 2） |
| T10 | 回归：MiniMax Studio 原有 5 个分享入口（文本/媒体/歌词/翻唱/一键音乐），其中无结果时点分享 | 行为与改动前一致：弹窗正常打开；**无结果时**提示「当前没有文本结果」类错误且**不产生未捕获异常**（`openShareDialog` 的 `try/catch` 保留） |
| T11 | 图像理解页上传 > 10MB 的图 → 识别 → 保存分享 | 保存成功，分享页**无图片资产**但有文本结果，且有「图片过大，仅保存识别文本」提示；闸门按**原始文件字节**判定（U4） |
| T12 | (B) 路径：prompt / 工具名极长（>120 / >240 字符）→ 保存分享 | 落库 `title` ≤ 120、`summary` ≤ 240（前端硬截断，NB4）；后端不报错、分享页正常打开 |

## PM 已拍板口径与本次不做项

> **本次无未决问题**：FINN-75 交回的 5 条未定义边界，PM 已于 2026-10-22 全部拍板（U8–U12，见 §4）；U13 是测试写法要求。实现方**照做、不得自行取舍**。
以下 4 项为**已拍板的「本次不做」**（非未决、非待裁决）：
1. **P2 附带项（公开页 / 画廊补 `image_understanding` 中文标签）**：`MiniMaxResultShareView.vue:193` 与 `MediaGallery.vue:427` 的 `typeLabel` 各补 1 行（纯展示、不改任何逻辑）→ **保留**（不补则公开页显示英文枚举）。
2. **画廊筛选加「图像理解」筛选项**（`MediaGallery.vue:228-233`）→ **不做**：后端 `type` 查询参数已支持（`minimax_result_share.go:216`），`minimaxResultShareMatchesType`（`:567-581`）也能精确匹配 `image_understanding`，故纯前端加一项即可 → 留作后续单独立项。
3. **`/ai-gateway` 页是否也支持 TTS 分享** → **不做**：那里没有分享对话框，另建对话框超范围；若 PM 要，等价做法是把 AIGatewayTool 也接入同一套对话框（需单独一个对话框或复用 MiniMaxStudio 组件），另立单。
4. **file_id 预览路径的 MIME（曾为未决项）** → **已按 PM 2026-10-22 02:35 修正拍板**：同步 file_id 预览路径**同用** `guessAudioMime(syncForm.format)`（与内联 hex 同源，理由见 §1.3 mime 表第 2 行），**不再是未决项**；M1 至此闭合。
