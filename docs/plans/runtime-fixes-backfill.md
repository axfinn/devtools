# runtime 修复回填 —— 移植方案（compress / SSE / header-timeout / 构建）

- 原 issue：FINN-83（Stage 1，父 FINN-82）
- 模块：后端 `backend/middleware/compress.go` + `backend/handlers/ai_gateway*.go` + `backend/models/ai_gateway.go` + `backend/config/config.go`；构建 `Dockerfile` / `deploy.sh` / `Makefile` / `ocr-service/Dockerfile` / `.github/workflows/ci.yml` / `.gitignore` / `package-lock.json`
- 状态：**设计稿，不含任何业务代码改动**。实现由 devtools-backend 执行，验收由 devtools-qa 执行
- 勘察方式：全部结论来自本仓库实际 `git show` / `grep` / `git cherry-pick` 干跑（隔离 worktree，**未起 dev server、未跑 deploy.sh、未 `docker build`**）。本方案中每条 `file:line` 均已实测核对

> **本方案只写「怎么把 6 条 commit 回填到 main」，不重新设计这 6 条改动本身。** 6 条 commit 的作者已经写过 commit message 级的设计说明；本方案负责的是**回填路径上的冲突裁决、可执行命令、验证与回滚**。

---

## 0. 勘察基线（PM 原文与实测的差异，务必先看）

PM 在 issue 里给的基线是 `origin/main = 542e457f76944b9a23bbe8433c924c28e1784cbf`。**实测当前 `origin/main` 已经是 `81199bde62699328c0e278b381a953b25fed476a`**（`git rev-parse origin/main`，已 `git fetch origin` 刷新）。

```
$ git merge-base --is-ancestor 542e457f76944b9a23bbe8433c924c28e1784cbf origin/main && echo YES
YES
$ git diff --stat 542e457f76944b9a23bbe8433c924c28e1784cbf origin/main
 docs/plans/tts-image-result-share.md               | 379 +++++++++++++++++++++
 frontend/src/components/AIGatewaySpeechPanel.vue   | 178 +++++++++++
 frontend/src/views/ai/ImageUnderstandingTool.vue   | 100 ++++-
 ... （共 10 个文件，全部是前端 / docs）
```

**结论：`542e457 → 81199bd` 之间只动了前端与 docs，一个后端 Go 文件都没碰。** 这对本方案是好消息 —— 6 条 commit 里 4 条改后端、1 条改构建、1 条改根 `package-lock.json`，与这段漂移**零重叠**。

**回填基线一律用 `81199bde62699328c0e278b381a953b25fed476a`（= 执行时的最新 `origin/main`），不要用 `542e457`。** 执行 Stage 3 前必须再 `git fetch origin` 确认一次，若 `origin/main` 又前进了，重新核一遍 `git diff --name-only <旧基线> <新基线>` 是否仍与本方案 6 条 commit 的文件集无交集。

### 0.1 6 条 commit 的对象可达性

```
$ for c in 3b402ead... c3119bd... 30b2079... 57d03e9... f5c6f32... 0968289...; do git cat-file -e $c && echo "$c OK"; done
```

6 条全部 `OK`（本地对象库可读）。**`git fetch origin` 找不到这些分支** —— 实测 `git ls-remote origin 'refs/heads/feat/ssrules-and-streaming-fixes' 'refs/heads/fix/ai-gateway-stream-header-timeout-per-provider'` 返回空。**一律按 hash 取**：

```bash
git cat-file -e 3b402ead36afbce4360a0710f761f9dee4976e99 && echo OK
git show --stat 3b402ead36afbce4360a0710f761f9dee4976e99
```

### 0.2 一个必须先讲清的事实：本地工作副本已经含 5 条

**本 agent 的 worktree（`agent/devtools-architect/finn-83`）HEAD 链上已经带了 5 条**：

```
$ git log --oneline -6
0e114cb chore(agent): baseline — the task worktree started here
f5c6f32 chore: package-lock.json 补 name 字段
57d03e9 fix(ocr-service): 用 Debian 官方源，不再改写 apt sources
30b2079 build(askit): 扩展改为宿主机构建，产物作为构建上下文输入
c3119bd fix(ai-gateway): SSE 心跳 goroutine 与主循环共用 sseWriter 串行化，并 join 后再返回
3b402ea fix(middleware): FINN-52 compress.go 去掉全量缓冲，改响应侧按 Content-Type 判定
```

这是**平台建 worktree 时的起点**（本地主检出当时停在 `feat/ssrules-and-streaming-fixes`），不是本方案造成的。**这意味着 Stage 3 绝不能在本 worktree 分支上做 cherry-pick**（会变成空提交 / already applied），必须**从 `origin/main` 新开分支**。见 §D1。

---

## 目标与非目标

### 目标

1. 把 6 条 commit 按**可执行、无语义丢失**的顺序回填到 `origin/main`。
2. 裁决 `0968289` × `c3119bd` 在 `backend/handlers/ai_gateway_anthropic.go` 的**唯一冲突点**，给出**逐处取舍**结论（谁胜 / 语义是否等价 / 会不会丢 header timeout 配置 / 会不会丢 SSE 心跳 join）。
3. 说清 `3b402ea` 改完后 SSE 是否**真的不再被全量缓冲**、`Flush()` 是否**真的下推**、`excludedPathPrefixes` 快路径保留后的正确性。
4. 说清 `0968289` 的 DB 迁移幂等写法、`0` 的默认语义、以及 `UpdateAnthropicProvider` 全列更新**清零其它字段**的老 bug 是否被一并修掉。
5. 给出**每条单独可 revert** 的顺序。
6. 写下 Stage 3 / Stage 5 **可直接照抄**的编译命令。

### 非目标

- **不改这 6 条 commit 的业务逻辑**。若 Stage 3 发现必须改逻辑，那是缺陷，走 issue 回报，不在本方案内悄悄改。
- 不做部署、不连 `192.168.31.201`、不跑 `./deploy.sh docker`（R1：部署由 jaxiu 手工执行）。
- 不做真机 / 线上验收（375×667 首屏、CF 实际 524 表现等一律 jaxiu 合并后自验，见 D5）。
- 不更新 `AGENTS.md` 模块地图（那是 module-architect 的写权范围；本方案不新增/删除 handler 文件、route 分组，故模块地图本轮无需变更 —— 已在 §现状核实）。

---

## 现状（已核实的代码位置）

> 每条都实际 `grep`/`git show` 看过。行号以**对应 commit 的文件版本**为准，方案里显式标注版本。

### S1 · `origin/main` 现状

| 项 | 核实结果 |
|---|---|
| 压缩中间件形态 | `origin/main:backend/middleware/compress.go:53` `type captureWriter struct`；`:84` `func (c *captureWriter) Flush()          {}` —— **空实现**（gofmt 对齐后的原文） |
| SSE 端点是否在排除名单 | `origin/main` 的 `excludedPathPrefixes` **不含** `/api/anthropic` / `/api/ai-gateway/v1/chat` |
| header timeout 配置 | `git grep -n "header_timeout_sec\|HeaderTimeoutSec" origin/main -- backend/` → **0 命中**（字段完全不存在） |
| provider 派生 client | `git grep -n "pickStreamClient\|buildStreamTransport" origin/main -- backend/` → **0 命中** |
| `UpdateAnthropicProvider` 形态 | `origin/main:backend/models/ai_gateway.go:1043` 全列 UPDATE：`UPDATE anthropic_providers SET name=?, api_url=?, api_key=?, api_key_encrypted=?, models=?, aliases=?, enabled=?, is_default=?, default_model=?, updated_at=... WHERE id=?` —— **无 `header_timeout_sec` 列** |
| 旧明文懒迁移 | `origin/main:backend/handlers/ai_gateway_anthropic.go` 异步写回只塞 `ID` / `APIKeyEncrypted` / `APIKey` 三字段 → 全列 UPDATE 会把其余列清零 |
| `sseWriter` 位置 | `origin/main:backend/handlers/ai_gateway_chat_stream.go:27` `func newSSEWriter(...)`；`origin/main:backend/handlers/ai_gateway_anthropic.go:645` 才创建（**在主复制循环前**） |
| AskIt 扩展构建 | `origin/main:Dockerfile` 有 `askit-builder` stage，`git clone https://github.com/axfinn/Askit.git`；`origin/main:Makefile:95` `docker-build:` 直接 `docker compose build` |
| 压缩中间件注册 | `origin/main:backend/app_http.go:26` `router.Use(middleware.ResponseCompression(5))` |
| 路由 | `origin/main:backend/routes/ai_gateway.go:46` `api.POST("/anthropic/v1/messages", ...)`（`/api` 组下） |
| 根 `package-lock.json` | `origin/main:package-lock.json` 顶层有 `"name": "devtools"`，但 `packages[""]` 里**没有** `name` 字段 |
| `handlers/askit_extension.go` | **在 main 上**（`git ls-tree origin/main backend/handlers/` 含 `askit_extension.go`），`:128` 附近是 `git clone --depth 1 --branch askitBranch askitRepoSSH`（`askitBranch = "main"`，`askitRepoSSH = "git@github.com:axfinn/Askit.git"`） |

### S2 · 6 条 commit 的文件清单

| # | commit | 文件 |
|---|---|---|
| 1 | `3b402ea` | `backend/middleware/compress.go`（309 行改）、`backend/middleware/compress_stream_test.go`（**新增** 254 行） |
| 2 | `c3119bd` | `backend/handlers/ai_gateway_anthropic.go`、`ai_gateway_chat_stream.go`、`ai_gateway_qwen.go` |
| 3 | `30b2079` | `Dockerfile`、`deploy.sh`、`Makefile`、`.github/workflows/ci.yml`、`.gitignore` |
| 4 | `57d03e9` | `ocr-service/Dockerfile` |
| 5 | `f5c6f32` | `package-lock.json`（1 行） |
| 6 | `0968289` | `backend/config/config.go`、`backend/handlers/ai_gateway.go`、`ai_gateway_admin.go`、`ai_gateway_anthropic.go`、`backend/models/ai_gateway.go`、**新增** `backend/handlers/ai_gateway_header_timeout_test.go`（111 行） |

---

## 设计

### D1 · 回填基线与分支

**从最新 `origin/main` 新开分支，不在 agent worktree 的现有 HEAD 上做。** 理由见 §0.2 —— 当前 worktree HEAD 已含 5 条，cherry-pick 会 no-op。

```bash
# Stage 3 执行者（devtools-backend），在**自己的** worktree 里：
git fetch origin
git checkout -b feature/runtime-fixes-backfill origin/main
# 基线自检：应打印 81199bd（或更新），且工作区干净
git log --oneline -1
git status --short
```

**基线漂移再确认**（若 `origin/main` 已从 `81199bd` 前进）：

```bash
git diff --name-only 81199bde62699328c0e278b381a953b25fed476a origin/main
```

期望：输出里**不含** `backend/middleware/`、`backend/handlers/ai_gateway*`、`backend/models/ai_gateway.go`、`backend/config/config.go`、`Dockerfile`、`deploy.sh`、`Makefile`、`ocr-service/Dockerfile`、`.github/workflows/ci.yml`、`.gitignore`、根 `package-lock.json`。若含 → **停下来回报**，重新裁决冲突。

### D2 · 合并顺序

**推荐顺序：`3b402ea` → `c3119bd` → `0968289` → `30b2079` → `57d03e9` → `f5c6f32`**

即**把 `0968289` 放在 `c3119bd` 之后**（Order B）。理由见 D5。

**实测（隔离 worktree，`origin/main` = 81199bd 起步）**：

```
### 3b402ead36afbce4360a0710f761f9dee4976e99   CLEAN
### 096828903572ffed07dd0fcc2e4f86fd082569a6   CLEAN
### c3119bdec3dfe89245531b7794a2a2e7733c2f39   CONFLICT: backend/handlers/ai_gateway_anthropic.go
### 30b20797b8fd67d3064295a79f3e54fd55df3fe1   CLEAN
### 57d03e9109a0c2ad14e2701fbb847574a4cac569   CLEAN
### f5c6f32ad66a043d849428a0d98bfcf52946edab   CLEAN
```

（上面 `0968289` 排在 `c3119bd` 前，即 Order B；只有 `c3119bd` 一处冲突。）

**Order A（`c3119bd` 在前、`0968289` 在后）实测同样只有一处冲突**：

```
### c3119bd...  CLEAN
### 0968289...  CONFLICT: backend/handlers/ai_gateway_anthropic.go
```

**两种顺序都只有 1 个冲突文件、1 个冲突块。** 冲突块内容在两种顺序下**完全对称**（同一处代码的两侧互换）—— 见 D5。

**5 条无冲突 commit 的 cherry-pick 一次通过**（`3b402ea` / `30b2079` / `57d03e9` / `f5c6f32` 在两种顺序下都 CLEAN），因为它们与 `81199bd` 的漂移（纯前端）零重叠。

### D3 · `3b402ea` —— 响应压缩语义（问题 3 的正面回答）

**改了什么**

`origin/main` 的 `captureWriter` 把 handler 全部响应字节攒进 `bytes.Buffer`，等 `c.Next()` 返回后才决定是否 gzip。两个后果：

- **a) `captureWriter.Flush()` 是空实现**（`origin/main:backend/middleware/compress.go:84`）。SSE handler 的 `c.Writer.(http.Flusher)` 断言**通过**（captureWriter 确有 Flush 方法），于是**静默吞掉每一次 Flush**。Anthropic 代理精心设计的 ping 心跳（拨号期 1s、生成期 5s）一个字节都到不了客户端。
- **b) 压不压缩只靠请求侧路径白名单**，而主力 LLM 流式端点 `/api/anthropic/v1/messages`、`/api/ai-gateway/v1/chat/completions` 不在名单里。

`3b402ea` 把 `captureWriter` 换成 `compressWriter`（`3b402ea:backend/middleware/compress.go:72`），**去掉全量缓冲**，改成**在响应头真正下发的那一刻按响应 `Content-Type` 判定**。

**新的关键实现（`3b402ea:backend/middleware/compress.go` 行号）**

| 行 | 内容 |
|---|---|
| `:72` | `type compressWriter struct`（嵌入 `gin.ResponseWriter`） |
| `:101` | `WriteHeader(code)` —— **只记录状态码**，不下发 |
| `:114` | `WriteHeaderNow()` → `w.writeHeadNow()` |
| `:117` | `writeHeadNow()` —— 响应头真正下发的唯一时机 |
| `:131` | `decide()` —— 按响应头判定是否 gzip |
| `:154` | `isStreamingContentType(ct)` —— `strings.Contains(lower(ct), "text/event-stream")` |
| `:160` | `startGzip()` —— 惰性建 gzip.Writer（首次 Write 前不写字节，故可放在响应头下发之后） |
| `:181` | `finish()` —— `c.Next()` 返回后的收尾（兜底下发响应头 + `gz.Close()` + 归还 pool） |
| `:197` / `:205` | `Write` / `WriteString` —— **透传路径直接 `w.ResponseWriter.Write(b)`，零缓冲** |
| `:218` | `Flush()` —— 压缩路径 `gz.Flush()` 后转发；透传路径**直接转发** |
| `:261` | `excludedPathPrefixes`（新增 `"/ss/"`） |
| `:297` | `ResponseCompression(level)` |

**问题 3 逐条正面回答**

1. **SSE 是否不再被全量缓冲？** —— **是**。
   - 机制：`:117 writeHeadNow()` → `:131 decide()` → `:154 isStreamingContentType()` 命中 `text/event-stream` → **不进入压缩路径** → `:197 Write` 直接落 `w.ResponseWriter`。
   - 触发前提（已核实）：SSE handler 一定在写第一个字节前设好 `Content-Type`。
     - `3b402ea:backend/handlers/ai_gateway_anthropic.go:558` `c.Writer.Header().Set("Content-Type", "text/event-stream")` —— 在 `:561 c.Writer.WriteHeader(http.StatusOK)` **之前**。
     - `c3119bd:backend/handlers/ai_gateway_anthropic.go:559` 同上（`:562` WriteHeader）。
     - `c3119bd:backend/handlers/ai_gateway_chat_stream.go:158` / `:272` —— 同样在 WriteHeader 之前。
     - `c3119bd:backend/handlers/ai_gateway_qwen.go:282` —— 同样。
   - gin's 写入时序（`c.Status()` → 只记状态码；`writeContentType` → 设响应头 map；`w.Write` → `WriteHeaderNow()` 才真正下发）是这套判定成立的前提，`3b402ea` 的文件头注释把这段时序写清楚了。判定放在 `writeHeadNow` 而非 `WriteHeader` 里，正是为了等到 `Content-Type` 已就位。

2. **`Flush()` 是否真正下推？** —— **是**。
   - `:218 Flush()` 在透传路径上执行 `w.ResponseWriter.Flush()`（**真 Flush**），对比旧的 `captureWriter.Flush()` 空实现。
   - 在压缩路径上是 `w.gz.Flush()` 再 `w.ResponseWriter.Flush()`，gzip 缓冲被吐出来。
   - `3b402ea` 新增的 `backend/middleware/compress_stream_test.go` 里有专门断言：`TestCompress_SSEStreamNotBuffered_RealAnthropicRoute`、`TestCompress_SSEStreamNotBuffered_RealQwenVisionRoute`、`TestCompress_SSEWithAcceptHeaderStillUnbuffered`、`TestCompress_FlushAdvancesCompressedStream` —— 其中「handler 仍在执行时已 Flush 的字节必须已在 writer 上」正是「不缓冲」的可执行判据。

3. **`excludedPathPrefixes` 快路径保留后的正确性？** —— **正确性不再依赖它，但保留无害**。
   - 请求侧白名单现在只是**快路径**（命中即完全不包装 writer，省掉包装开销），`:261` 的注释明确写「正确性不再依赖本列表」。
   - **快路径与响应侧判定不冲突**：快路径命中 → 不包装 → 本来就不压缩；快路径未命中 → 包装 → `decide()` 看到 `text/event-stream` 也不压缩。**两条路径殊途同归**。
   - **不能为了「保险」把 LLM JSON 端点加进排除列表** —— `/api/anthropic/v1/messages` 同时承载流式(SSE) 与非流式(大 JSON) 两种响应，按路径排除会连带丢掉非流式大响应的压缩收益。`3b402ea:backend/middleware/compress.go:261` 的注释专门警告了这一点。
   - **实测**：`3b402ea` 应用后 `go test ./middleware/` 全绿（`ok devtools/middleware 0.956s`），既有 `compress_test.go`（13 个 Test）+ `compress_qa_test.go`（13 个 Test）**全部未受影响**。

4. **与 524 的关系** —— 这条是 524 根因之一（CF origin read timeout）。旧实现把 ping 心跳整个吞掉 → 客户端/CF 侧长时间收不到任何字节 → 按空闲超时断开。修完后心跳逐帧送达，**CF 不会再因为「上游在生成、但一个字节都到不了客户端」而触发 origin read timeout**。
   - ⚠️ 但 524 还有**另一个独立成因**：`0968289` 修的「上游首字节 > 60s 被 `ResponseHeaderTimeout` 掐死」。两条成因**都要修才完整** —— 这正是本方案要回填 6 条而不是只回填 1 条的原因。

**风险**

- `decide()` 会 `header.Del("Content-Length")` —— 压缩后长度未知，靠 chunked。对已声明 `Content-Length` 的响应会改变传输方式。**已在 `decide()` 里被排除的情形**：204/304（无 body）、`text/event-stream`（流式）、已设 `Content-Encoding`、已设 `Content-Range`。
- ⚠️ **已知项，本轮不改（PM 已拍板）**：`decide()` 用 `header.Set("Vary", "Accept-Encoding")`
  （`3b402ea:backend/middleware/compress.go:149`）会**覆盖** handler 既有 Vary 值（改 `Add` 更稳妥）。
  **本轮不改** —— Stage 3 纪律是「cherry-pick 保真、不得自行改写」。**PM 另开跟进单。**
- ⚠️ **已知偏差，本轮不改（PM 已拍板）**：`compressWriter` **嵌入** `gin.ResponseWriter`
  （`3b402ea:backend/middleware/compress.go:72-73`），未覆写 `Size()` → 压缩路径下 `Size()` 走的是
  底层 writer，只统计**已实际吐出**的字节（gzip 缓冲未 flush 时偏小）。
  **非功能缺陷**，只影响日志 / 进度统计的数值口径。**本轮不改。**
- `startGzip()` 在 `level` 越界时兜底到 5（`gzip.NewWriterLevel` 对非法 level 返回 error，而那时 `Content-Encoding: gzip` 已写进响应头无法撤销）—— 构造期 `newCompressWriter` 就把 level 夹到 `[1,9]`，`startGzip` 里的 `_ = gzip.NewWriterLevel(...)` 是**理论不可达**的兜底，不会留下 `gz==nil && compress==true` 的静默损坏。
- 旧的 `captureWriter` 会把捕获到的 headers 逐个 `Add` 回原 writer；新的 `compressWriter` **不捕获**，直接改原 header map。这消除了「headers 重复 Add」的可能，但也意味着**响应头不再有「延迟下发」的机会** —— handler 若在 `WriteHeader` 之后还想改 header，将不再生效（gin 原生语义即如此，行为对齐）。

### D4 · `c3119bd` —— SSE 心跳串行化与 join

**改了什么**

三个流式端点（`proxyAnthropicStream` / `streamOpenAICompatible` / `streamMiniMaxChat`）的共同缺陷：

- 心跳 goroutine 与主复制/扫描循环**并发写同一个 `c.Writer`** → `Write` 与 `Flush` 在底层 bufio 上数据竞争 → 流内容错乱。
- 停止时只 `close(done)` **不发信号 join** → 心跳可能卡在 ticker 分支，在 handler 返回后继续向已完成的 `ResponseWriter` 写入 → `superfluous response.WriteHeader`。

`c3119bd` 的三步修法：

1. **心跳与主循环统一走同一个 `newSSEWriter`**（`c3119bd:backend/handlers/ai_gateway_anthropic.go:566` `writer := newSSEWriter(c.Writer, flusher)` —— **提到函数开头**，不再是 `:645`）。`sseWriter.write`（`backend/handlers/ai_gateway_chat_stream.go:32`）用 `sync.Mutex` 把 `Write` + `Flush` 串行化。
2. **停止改为 `close(done)` 后 `WaitGroup.Wait()`**：`preHeartbeatWG`（`c3119bd:backend/handlers/ai_gateway_anthropic.go:579`）与 `heartbeatWG`（`:668`）。
3. **anthropic 的拨号期心跳 stop 包成幂等 `sync.Once`**（`:581 stopPreHeartbeat := func(){...}`），并在函数入口 `defer stopPreHeartbeat()`（`:611`）兜底 —— 任何提前 return（上游 4xx/5xx、读错误）都能把心跳收干净。

**影响面**：`ai_gateway_anthropic.go`（`proxyAnthropicStream`）、`ai_gateway_chat_stream.go`（`streamOpenAICompatible` + `streamMiniMaxChat`）、`ai_gateway_qwen.go`（`relayVisionSSE`）。

**风险**

- `stopPreHeartbeat` 里的 `preHeartbeatWG.Wait()` 会**阻塞到心跳 goroutine 真正退出**。心跳 goroutine 的 select 分支是 `ticker.C` / `preUpstreamHeartbeatDone` / `ctx.Done()`，正常路径下 `close(preUpstreamHeartbeatDone)` 后立刻返回，Wait 不会长时间阻塞。**但**若 goroutine 正卡在 `writer.write()` 里（Write syscall 阻塞），Wait 会等它 —— 这是**有意的**（不能让它在 handler 返回后写已完成的 ResponseWriter）。
- 幂等 `sync.Once` 保证 `stopPreHeartbeat()` 可被重复调用（`:611` defer + `:616` 错误路径 + 成功路径各一次）。**这是 D5 冲突裁决的关键**。

**验证方式**：`go test ./handlers/ -race`（现有 `ai_gateway_apikey_encrypt_test.go` 等不覆盖心跳；真正的竞态验证靠 `-race` 全量跑 + 人工审 `sseWriter` 的锁覆盖范围）。

### D5 · `0968289` × `c3119bd` 的冲突裁决（问题 2 的正面回答）

#### D5.1 冲突的确切位置与内容

**唯一冲突点**：`backend/handlers/ai_gateway_anthropic.go` 的 `proxyAnthropicStream` 里「发上游请求 + 停拨号期心跳」这两行。

**Order B（先 `0968289`，后 `c3119bd`）的冲突文本（实测原文）**：

```go
<<<<<<< HEAD
	// 使用按 provider 派生的 streamClient（body 无 Timeout），避免 http.Client.Timeout 在长流时强制断开；
	// 连接/握手/响应头超时由对应 transport 兜底（Ollama 等慢上游用 600s,云厂商默认 60s），
	// 瞬时连接错误在拿响应前做一次重试。
	resp, err := doStreamRequest(h.pickStreamClient(provider), req)
	// 上游开始响应后停掉拨号期高频心跳,改用主循环的 20s 心跳(防止 thinking 阶段断流)
	close(preUpstreamHeartbeatDone)
=======
	// 使用 streamClient（body 无超时），避免 http.Client.Timeout 在长流时强制断开；
	// 连接/握手/响应头超时由 streamTransport 兜底，瞬时连接错误在拿响应前做一次重试。
	resp, err := doStreamRequest(h.streamClient, req)
	// 上游开始响应后停掉拨号期高频心跳(并 join)，改用生成期的 5s 心跳(防止 thinking 阶段断流)
	stopPreHeartbeat()
>>>>>>> c3119bd (fix(ai-gateway): SSE 心跳 goroutine 与主循环共用 sseWriter 串行化，并 join 后再返回)
```

**Order A（先 `c3119bd`，后 `0968289`）的冲突文本是同一处的镜像**：`HEAD` 侧是 `c3119bd` 版本（`h.streamClient` + `stopPreHeartbeat()`），`=======` 侧是 `0968289` 版本（`h.pickStreamClient(provider)` + `close(preUpstreamHeartbeatDone)`）。

**两边各自带了对方的「正确部分」和「错误部分」**：
- `0968289` 侧：**client 派生是对的**（`pickStreamClient(provider)`），**心跳停止是旧的**（裸 `close`，无 join、无幂等）。
- `c3119bd` 侧：**心跳停止是对的**（`stopPreHeartbeat()`，幂等 + join），**client 是旧的**（`h.streamClient` 写死 60s）。

**⚠️ 一处必须讲清的细节：`close(preUpstreamHeartbeatDone)` 在两棵树里出现次数不同。**

```
$ git show 0968289...:backend/handlers/ai_gateway_anthropic.go | grep -n "close(preUpstreamHeartbeatDone)"
568:		close(preUpstreamHeartbeatDone)     ← NewRequest 失败路径（裸 close）
604:	close(preUpstreamHeartbeatDone)     ← doStreamRequest 之后（裸 close）★ 这行在冲突块内

$ git show c3119bd...:backend/handlers/ai_gateway_anthropic.go | grep -n "close(preUpstreamHeartbeatDone)\|stopPreHeartbeat"
577:	// ... stopPreHeartbeat 幂等，可安全重复调用。
581:	stopPreHeartbeat := func() {
583:			close(preUpstreamHeartbeatDone)  ← 唯一一处裸 close，在闭包内部
584:			preHeartbeatWG.Wait()
611:	defer stopPreHeartbeat()
615:		stopPreHeartbeat()                  ← NewRequest 失败路径
650:	stopPreHeartbeat()                    ← doStreamRequest 之后（成功路径）
```

- **只有 `:604` 那行落在冲突块内**。`:568`（NewRequest 失败路径）**不在冲突块内** —— `0968289` 没碰那一段，所以 `c3119bd` 把 `:568` 改成 `:615` 的 `stopPreHeartbeat()` 是**干净应用**的，不会冲突。
- 也就是说：**Order B 下 `c3119bd` 的 hunk 只有成功路径那一个冲突**，其余（`:568`→`:615`、`:581-589` 的 `sync.Once` 包装、`:611` 的 defer）全部自动应用。
- **合并后的正确形态**：`close(preUpstreamHeartbeatDone)` 在整个文件里**只应剩 1 处**，且在 `stopPreHeartbeat` 闭包内部（`:583`）。这是 L1 的核心判据。

#### D5.2 逐处取舍

| 处 | 取哪边 | 为什么 |
|---|---|---|
| `resp, err := doStreamRequest(?, req)` 的 client 参数 | **`h.pickStreamClient(provider)`**（`0968289` 侧） | 这是 `0968289` 的**全部目的**。取 `c3119bd` 侧的 `h.streamClient` 会让 `header_timeout_sec` 配置**完全失效** —— Ollama 27B 仍被 60s `ResponseHeaderTimeout` 掐死，首字节 >60s 的场景原地复发。**取错 = 丢 header timeout 配置。** |
| 停拨号期心跳那一行 | **`stopPreHeartbeat()`**（`c3119bd` 侧） | 这是 `c3119bd` 的**全部目的**。取 `0968289` 侧的裸 `close(preUpstreamHeartbeatDone)` 会**丢掉 SSE 心跳 join**：心跳 goroutine 可能卡在 ticker 分支，在 handler 返回后继续向已完成的 ResponseWriter 写入 → `superfluous response.WriteHeader` + 数据竞争。**取错 = 丢 SSE 心跳 join。** |
| 注释 | **合并两边**，保留 `0968289` 版的「按 provider 派生」措辞 + `c3119bd` 版的「(并 join)」措辞 | 注释要同时说清两件事，避免后人照着旧注释改回去 |

**结论：两边都不能整块取，必须取「`0968289` 的 client + `c3119bd` 的心跳」这个并集。**

#### D5.3 ⚠️ 关键陷阱：裸 `close` + `defer stopPreHeartbeat()` = 运行时 panic

**这不是理论风险，是实测可复现的。**

`c3119bd` 在函数入口加了 `defer stopPreHeartbeat()`（`c3119bd:backend/handlers/ai_gateway_anthropic.go:611`），而 `stopPreHeartbeat` 内部是 `sync.Once` 包着 `close(preUpstreamHeartbeatDone)`（`:581-586`，闭包内唯一的 `close` 在 `:583`）。

如果冲突解决时**取了 `0968289` 侧的裸 `close(preUpstreamHeartbeatDone)`**：

- 成功路径上裸 `close`（`:604`）先执行（关掉 channel）；
- 函数返回时 `defer stopPreHeartbeat()`（`:611`）执行 → `sync.Once` 首次触发 → 闭包里的 `close`（`:583`）**再关一次同一个 channel** → `panic: close of closed channel`。
- 同理，NewRequest 失败路径（`0968289:568` / `c3119bd:615`）若也保留裸 `close`，那条路径返回时同样 panic。

**实测复现**（独立最小 Go 程序，语义与 `stopPreHeartbeat` 一致）：

```
reached end
panic: close of closed channel

goroutine 1 [running]:
main.main.func2.1()
	sync.(*Once).doSlow(0xc?, 0x10459bebc?)
```

**这正是「语义是否等价」这一问的答案：不等价，且错的那一侧是运行时 panic，不是静默降级。**

> 附带说明：`sync.Once` 只保证**它自己**的 func 只跑一次，**不阻止外部代码直接 `close` 同一个 channel**。所以 `sync.Once` 救不了这个坑 —— 必须选对边。

#### D5.4 正确解法（可直接照抄）

**Order B（推荐顺序）解决冲突时，把整个冲突块替换为：**

```go
	// 使用按 provider 派生的 streamClient（body 无 Timeout），避免 http.Client.Timeout 在长流时强制断开；
	// 连接/握手/响应头超时由对应 transport 兜底（Ollama 等慢上游用 600s,云厂商默认 60s），
	// 瞬时连接错误在拿响应前做一次重试。
	resp, err := doStreamRequest(h.pickStreamClient(provider), req)
	// 上游开始响应后停掉拨号期高频心跳(并 join)，改用生成期的 5s 心跳(防止 thinking 阶段断流)
	stopPreHeartbeat()
```

**Order A 得到的最终文件与 Order B 完全一致**（实测：两种顺序下解决后 `git diff` 无差异）。

**解决后必须自查的三件事**（贴 grep 输出）：

```bash
# ① 既有 pickStreamClient（header timeout 派生），也有 stopPreHeartbeat（join）—— 两个都要在
grep -n "pickStreamClient(provider)\|stopPreHeartbeat()" backend/handlers/ai_gateway_anthropic.go
#   期望：pickStreamClient(provider) 恰好 1 处（成功路径的 doStreamRequest）
#   期望：stopPreHeartbeat() 恰好 3 处 —— defer 兜底 + NewRequest 失败路径 + 成功路径
#   若 pickStreamClient 为 0 → 丢了 header timeout 配置（见 D5.2）

# ② 裸 close(preUpstreamHeartbeatDone) 只能出现在 stopPreHeartbeat 闭包内部（唯一 1 处）
grep -n "close(preUpstreamHeartbeatDone)" backend/handlers/ai_gateway_anthropic.go
#   期望：只有 1 处命中，且缩进在 stopPreHeartbeat := func(){...} 内部
#   若出现 2 处 → 裸 close 没被替换掉 → 会 panic → 必须重解

# ③ 无冲突残留
#   ⚠️ 必须用**锚定行首**的写法，否则断言不判别（这是 Stage 2 判 FAIL 的 P0 之一）
grep -cE '^(<<<<<<<|=======|>>>>>>>)' backend/handlers/ai_gateway_anthropic.go
#   期望：0
```

> **⚠️ ③ 的写法说明（务必读，否则 QA 会误判）**：**不要**用「`grep -c` + 三个冲突标记、**不带 `^` 行首锚定**」的旧写法 ——
> 那会命中文件里**早就存在**的分节注释
> `// ===================== 异步 Anthropic 代理(task 模式,防 CF 524) =====================`
> （`origin/main:backend/handlers/ai_gateway_anthropic.go` 里就有这一行，回填后行号前移）。
> `=======` 是那行 21 个连续 `=` 的子串 → **基线与正确解法都返回 1**，断言恒真。
> **实测三态互不相同**：锚定版在「已解决」= `0`、「未解决」= `3`、「`origin/main` 基线」= `0`。

**③ 之后**：

```bash
gofmt -l backend/handlers/ai_gateway_anthropic.go   # 期望：无输出
git add backend/handlers/ai_gateway_anthropic.go
git cherry-pick --continue
```

**实测**：按上述解法解决后，`go build ./...` 通过（rc=0），`go vet ./handlers/ ./middleware/ ./models/` 通过（rc=0），`gofmt -l` 无输出。

#### D5.5 顺序选择的最终建议

**用 Order B（`0968289` 在前，`c3119bd` 在后）**，理由：

- Order B 下，`0968289` **一次 CLEAN**（它与 `81199bd` 零重叠），只有 `c3119bd` 冲突 —— **冲突次数 1 次**。
- Order A 下，`c3119bd` CLEAN，`0968289` 冲突 —— 也是 1 次。**两者冲突次数相同**，所以顺序不因冲突数而分胜负。
- **决定性理由**：Order B 的冲突块里，`HEAD` 侧（`0968289`）= 我们要保留的 client 派生，`=======` 侧（`c3119bd`）= 我们要保留的 `stopPreHeartbeat()`。也就是**我们要取的两半正好分居冲突块两侧**，人工裁决时不容易看漏。
- **Order A 的隐忧**：`c3119bd` 的 diff 会把 `newSSEWriter` 从函数中段（`origin/main:backend/handlers/ai_gateway_anthropic.go:645`）**提到函数开头**（`c3119bd:...:566`）。`0968289` 的 diff 是在旧位置上改 `doStreamRequest` 那一行。Order A 下 `0968289` 的 hunk 上下文被 `c3119bd` 挪动过，**冲突块两侧的归属与 Order B 相反**（Order A 是 `HEAD`=c3119bd / `=======`=0968289）—— 两种都可用，但 Order B 的「我们要取的两半分居冲突块两侧」在人工审阅时更直观（Order B 的 `HEAD` 侧正好就是我们要留的 client 派生）。
- **无论哪种顺序，最终解都是同一个并集**（D5.4）。**下游若选了 Order A，只需把 D5.4 的最终内容照抄进冲突块即可，结果文件一致。**

### D6 · `0968289` —— header timeout 配置（问题 4 的正面回答）

#### D6.1 改了什么

- `backend/config/config.go`：`AnthropicProviderConfig` 加 `HeaderTimeoutSec int \`yaml:"header_timeout_sec"\``（`0 = 默认 60s`）。
- `backend/models/ai_gateway.go`：`AnthropicProvider` 加 `HeaderTimeoutSec int`；建表加列；幂等 `ALTER TABLE` 加列；Ollama seed/backfill = 600；SELECT 列表加列。
- `backend/handlers/ai_gateway.go`：新增 `buildStreamTransport(headerTimeoutSec)`（`:273`）、`pickStreamClient(provider)`（`:296`）、`streamClientCacheKey`（`:34`）、`streamClientCache sync.Map`（`:30`）。
- `backend/handlers/ai_gateway_anthropic.go`：`proxyAnthropicStream` 走 `h.pickStreamClient(provider)`；`dbAnthropicProvidersToConfig` 带上 `HeaderTimeoutSec`；**修旧明文懒迁移 bug**。
- `backend/handlers/ai_gateway_admin.go`：`AdminCreateAnthropicProvider` 收 `header_timeout_sec`；`AdminUpdateAnthropicProvider` 用 `*int` 区分「未传=不修改」与「传 0=显式设 0」；`GetAnthropicDocs` 输出 `header_timeout_sec`。

#### D6.2 DB 迁移的幂等 `ALTER TABLE` 写法

**实测写法**（`0968289:backend/models/ai_gateway.go:853`）：

```go
if _, err := db.conn.Exec(`ALTER TABLE anthropic_providers ADD COLUMN header_timeout_sec INTEGER NOT NULL DEFAULT 0`); err != nil {
	if !strings.Contains(err.Error(), "duplicate column") {
		return err
	}
}
```

**逐点核实**

- **幂等性**：SQLite 的 `ALTER TABLE ... ADD COLUMN` **不支持 `IF NOT EXISTS`**。重复执行会报 `duplicate column name: header_timeout_sec`。上面的写法靠「捕获错误 + 判定是 `duplicate column` 就放过」实现幂等。**与该文件既有的 `default_model`（`:843`）和 `api_key_encrypted`（`:848`）迁移是同一套模式** —— 沿用既有风格，不是新发明。
- **不是所有错误都放过**：`!strings.Contains(err.Error(), "duplicate column")` → 其它错误（如磁盘满、表锁）**照常 return err**。这点很重要：不能写成「忽略一切错误」。
- **新建表路径也带列**：`CREATE TABLE IF NOT EXISTS anthropic_providers (... header_timeout_sec INTEGER NOT NULL DEFAULT 0, ...)`。所以**新装**走 CREATE（带列），**老装**走 ALTER（补列），两条路径都拿到该列。
- **列位置不影响语义**：SQLite 的 `ALTER TABLE ADD COLUMN` 总是加到表尾。`SELECT` 用**显式列名列表**（`anthropicProviderColumns` / `listAnthropicProviders` 里的 SELECT），**不是 `SELECT *`**，所以列顺序无关紧要。**这点已核实**：`0968289` 把 `anthropicProviderColumns` 变量同步加上了 `header_timeout_sec`（紧邻 `default_model` 之后），`rows.Scan` 的目标顺序也同步更新。

#### D6.3 `0` 的默认语义 —— 会不会覆盖原来写死的 60s？

**答：不会覆盖。`0` 就是「沿用原来的 60s」，语义完全一致。**

**实测**（`0968289:backend/handlers/ai_gateway.go:273` `buildStreamTransport`）：

```go
timeout := time.Duration(headerTimeoutSec) * time.Second
if headerTimeoutSec <= 0 {
	timeout = 60 * time.Second
}
```

- `0` → 60s（= 原来 `streamTransport := buildTransport(true, 60*time.Second)` 的写死值）
- 负数 → 也回落 60s（`0968289` 的 `TestBuildStreamTransport_NegativeFallsBack` 覆盖）
- `>0` → 用该值（Ollama = 600s）

**双保险**：`pickStreamClient(provider)` 在 `provider == nil` 时**直接返回 `h.streamClient`**（全局兜底，其 transport 也是 60s），完全绕开派生逻辑。`0968289` 的 `TestPickStreamClient_NilProviderUsesFallback` 覆盖。

**兼容矩阵**：

| provider | `header_timeout_sec` | 派生前 | 派生后 |
|---|---|---|---|
| MiniMax / DashScope / DeepSeek / PackyAPI / OpenClaudeCode | 0（seed 里显式写 0） | 60s | **60s（不变）** |
| Ollama | 600（seed + backfill） | 60s | **600s** |
| 老装已有 Ollama 行（`header_timeout_sec` 原本不存在 → 迁移后为 0） | 迁移后 0 → backfill 成 600 | 60s | **600s** |
| config.yaml 路径的 provider（`AnthropicProviders`） | 未配 → 0 | 60s | **60s（不变）** |

**结论：云厂商线路**零行为变化**，只有 Ollama 被放宽。**这是安全的默认值设计。

⚠️ **一个需要下游知道的口径**：`dbAnthropicProvidersToConfig`（DB 路径）带 `HeaderTimeoutSec`（`0968289:backend/handlers/ai_gateway_anthropic.go:177`），但 **`builtinAnthropicProviders()`（`backend/handlers/ai_gateway_anthropic.go:44`）没有设 `HeaderTimeoutSec`** —— 内置列表里也没有 Ollama，所以**无影响**。`fillProviderAPIKeys`（`:246`）是 `copy` 整个 struct，config.yaml 里配的 `header_timeout_sec` 会**原样透传**到 `pickStreamClient`。**三条 provider 来源路径（DB / config.yaml / 内置）都覆盖到了。**

#### D6.4 `UpdateAnthropicProvider` 全列更新清零其它字段的老 bug —— 是否被一并修掉？

**答：修掉了，且修法正确。**

**老 bug 的确切形态**

`UpdateAnthropicProvider` 是**全列 UPDATE**：

```go
_, err := db.conn.Exec(`UPDATE anthropic_providers SET name=?, api_url=?, api_key=?, api_key_encrypted=?, models=?, aliases=?, enabled=?, is_default=?, default_model=?, updated_at=datetime('now') WHERE id=?`, ...)
```

`0968289` 之后**多了 `header_timeout_sec=?`** 一列。

而 `origin/main:backend/handlers/ai_gateway_anthropic.go` 的旧明文懒迁移是：

```go
go func(pid int64, cipher string) {
	_ = h.db.UpdateAnthropicProvider(&models.AnthropicProvider{
		ID:              pid,
		APIKeyEncrypted: cipher,
		APIKey:          "",
	})
}(p.ID, enc)
```

**只填 3 个字段** → 全列 UPDATE 把 `name` / `api_url` / `models` / `aliases` / `enabled` / `is_default` / `default_model` 全部**写成零值**（空串 / 0 / false）。

**危害**（这比「header_timeout_sec 被打回 0」严重得多）：**provider 被整个清空** —— `enabled` 变 false → `ListEnabledAnthropicProviders` 查不到它 → **该 provider 直接从网关消失**。

**修法**（`0968289:backend/handlers/ai_gateway_anthropic.go:199-201`）：

```go
// 重要:必须把 p 的其他字段(HeaderTimeoutSec / DefaultModel / Models 等)带过去,
// UpdateAnthropicProvider 是全列更新,只填 ID/APIKeyEncrypted/APIKey 会把其他列清零,
// 例如把 Ollama 的 header_timeout_sec 从 600 打成 0。
snapshot := *p
go func() {
	_ = h.db.UpdateAnthropicProvider(&snapshot)
}()
```

**正确性论证**：

- `p` 是 `dbAnthropicProvidersToConfig` 循环里的 `*models.AnthropicProvider`，**来自 `listAnthropicProviders` 的完整 SELECT**（`0968289` 已把 `header_timeout_sec` 加进 SELECT 与 Scan）→ **`p` 持有该行全部字段的当前值**。
- `snapshot := *p` 是**值拷贝**（不是指针别名）→ 避免 goroutine 与主循环竞争 `p`。
- `snapshot` 在 `go` 之前捕获 → **每次迭代一个独立副本**（`snapshot` 是循环体内新声明的变量，不是循环变量本身，不受 Go 1.22 循环变量语义影响）。
- 因此 UPDATE 写回的是**该行的真实当前值**，只有 `APIKeyEncrypted` / `APIKey` 两项是本次要改的（`p.APIKeyEncrypted = enc; p.APIKey = ""` 在 `snapshot := *p` **之前**执行 → snapshot 拿到的是新密文 + 已清空的明文，**正确**）。

**实测核实**：解决冲突后的最终文件里 `snapshot := *p` 与 `go func(){ _ = h.db.UpdateAnthropicProvider(&snapshot) }()` 都在（`grep` 命中 `:214` / `:216`）。

⚠️ **一个残留的（**非本方案引入**的）隐患**：`AdminUpdateAnthropicProvider`（`backend/handlers/ai_gateway_admin.go:746`）走的是「先 `GetAnthropicProviderByID` 读全行 → 改若干字段 → `UpdateAnthropicProvider(existing)` 写全列」的模式，**`existing` 是完整行**，所以**没有清零问题**。但 `models.AnthropicProvider` 结构体新增字段后，**任何未来新增的「只填部分字段就调 `UpdateAnthropicProvider`」的调用点都会重现这个 bug**。`0968289` 只修了当前**唯一**一个这样的调用点（懒迁移 goroutine）—— 我已 `grep -rn "UpdateAnthropicProvider" backend/` 核实，`origin/main` 合并后的调用点只有 3 处：懒迁移 goroutine（已修）、`AdminUpdateAnthropicProvider`（全行，安全）、测试文件。**当前无遗漏**。

> **✅ PM 已拍板（2026-10-22）：本轮加一行注释，钉住这颗雷。**
> 在 `backend/models/ai_gateway.go` 的 `UpdateAnthropicProvider` 上加注释，明确「**调用方必须传完整行**」。
> **理由**：1 行注释成本，收益是把「未来新增部分字段调用点会重现『清零 provider』」这个雷钉在函数头上。
> **不重构为 partial-update**（那是接口语义变更，需另开 issue）。

### D7 · `30b2079` —— AskIt 扩展改为宿主机构建

**改了什么**

- `Dockerfile`：**删掉** `askit-builder` stage（`FROM node:20-alpine AS askit-builder` + `git clone https://github.com/axfinn/Askit.git` + `npm install` + `npm run build` + `zip`），改成 `COPY askit-extension.zip ./askit-extension.zip`。
- `deploy.sh`：新增 `build_askit`（浅克隆 SSH → 打 `dist/` → 写 `$SCRIPT_DIR/askit-extension.zip` + `$BACKEND_DIR/data/askit/`），新增 `write_askit_placeholder` / `askit_bailout` / `askit_pack_dist` 三个 helper；`docker_deploy` 在 `docker compose build` **之前**调 `build_askit`；新增 `build-askit|askit` case；`help()` 补说明。
- `Makefile`：`docker-build:` 改成先 `./deploy.sh build-askit` 再 `docker compose build`。
- `.github/workflows/ci.yml`：Docker build job 加一步 `SKIP_ASKIT=1 ./deploy.sh build-askit`。
- `.gitignore`：加 `askit-extension.zip`。

**为什么必须改**：`Askit` 是私有仓库，镜像内既无凭据也无 ssh-agent → `git clone` 报 `could not read Username for 'https://github.com'`（exit 128）→ **直接打断发布**。

**关键设计点核实**：

- **构建产物来源与运行时一致**：`build_askit` **不跑 `npm install` / `npm run build`**，只打包**已提交的 `dist/`**。这与运行时 `backend/handlers/askit_extension.go` 的 `buildZipFromGitHub`（**在 main 上**，`:128` 附近 `git clone --depth 1 --branch askitBranch askitRepoSSH` → `askitClonedCommit` → 遍历 `dist/` 打进 zip）**完全对齐**。`askitBranch = "main"`，`askitRepoSSH = "git@github.com:axfinn/Askit.git"` —— `build_askit` 用的是 `git clone --depth 1 --branch main git@github.com:axfinn/Askit.git`，**branch / URL / 协议三项全一致**。
- **构建不阻断发布**：`build_askit` **不返回失败**（任何一步失败 → `askit_bailout` → 写占位 zip + 告警 → `return 0`）。占位包里放的是 `README.txt`（说明「扩展改由运行时 SSH 拉取」），不是空包。
- **CI 不需要凭据**：`SKIP_ASKIT=1` → 直接写占位包。CI 的 Docker build job 因此不需要 Askit 私库凭据。
- **`.dockerignore` 不排除 `askit-extension.zip`**（已核实：`grep -n "askit\|\*\.zip" .dockerignore` → 0 命中）→ 上下文能带上。
- **`.gitignore` 排除它** → 不进版本库（它只是构建上下文输入）。

**影响面 / 风险**

- ⚠️ **直接 `docker compose build` 会失败**（`COPY askit-extension.zip` 找不到源文件）。这是**有意的**行为变更，注释里明确写了「请走 `./deploy.sh docker`」。
- ⚠️ **已知既有项，本轮不修（PM 已拍板）**：`.github/workflows/ci.yml:12` `GO_VERSION: '1.21'`
  与 `backend/go.mod:3` 的 `go 1.25.5` 不匹配 —— CI 的 Go job 可能本来就在报
  `go.mod requires go >= 1.25.5`。
  **这是既有红项，不是本轮回填引入的**（`30b2079` 对 `ci.yml` 只加了那一行 `SKIP_ASKIT`）。
  **本轮不修，文档也不承诺修它** —— PM 单独记一笔。
- ⚠️ **`Makefile` 的 `docker-build` 目标行为变了**：从 `docker compose build` 变成 `./deploy.sh build-askit && docker compose build`。依赖 Askit 私库 SSH key 的开发机才能拿到真包；没 key 的机器走占位包。
- `deploy.sh` 有 `set -euo pipefail`（`:14`）—— `build_askit` 内部的失败都被 `askit_bailout` 吞成 `return 0`，**不会触发 `set -e` 中断**。已核实 `build_askit` 内所有失败路径都走 `askit_bailout` + `return 0`。
- **`ASKIT_DIR` 覆盖路径**：显式指定本地 checkout 时**原样使用其 `dist/`，不 fetch / 不 reset**，绝不覆盖未提交的改动。注释里明确写了这点。

**验证方式**：`bash -n deploy.sh`（语法检查）+ `SKIP_ASKIT=1 ./deploy.sh build-askit`（写占位包，**不构建镜像**）+ `unzip -l askit-extension.zip` 看内容。

### D8 · `57d03e9` —— ocr-service 用 Debian 官方源

**改了什么**：`ocr-service/Dockerfile` 删掉 `sed -i 's|deb\.debian\.org|mirrors.aliyun.com|g; ...' /etc/apt/sources.list.d/*.sources`，只保留 `Acquire::http::Proxy "false"` 那一段 `printf`。

**原因**：`python:3.11-slim` 现已是 Debian 13 trixie，改写到 `mirrors.aliyun.com` 偶发 `Err:4 ... trixie/main arm64 Packages` 导致构建中断。官方源更稳。

**影响面**：`ocr-service` 镜像构建期。国内拉取变慢 —— commit message 自己写了「国内嫌慢可自行换回镜像站（清华/中科大对 trixie 覆盖更完整）」。

**风险**：低。**唯一影响是构建速度**，不影响运行时行为。**但它是一次「换回官方源」的取舍** —— 若部署主机在墙内且官方源不通，构建会变慢或失败。**部署前由 jaxiu 自行确认（不在本方案验收范围）。**

**验证方式**：`docker build -f ocr-service/Dockerfile ocr-service/` 能跑完（**Stage 3/5 不跑镜像构建，交给 jaxiu**）。

### D9 · `f5c6f32` —— 根 package-lock.json 补 name

**改了什么**：`package-lock.json` 的 `packages[""]` 里加 `"name": "devtools"`（1 行）。顶层本来就有 `"name": "devtools"`，只是 `packages[""]` 缺。

**影响面**：根 `package.json`（内容只有 `dependencies: {@ffmpeg/ffmpeg, events}`，**没有 name 字段**）与根 `package-lock.json`。**注意这跟 `frontend/package.json` / `frontend/package-lock.json` / `frontend/pnpm-lock.yaml` 是三套独立的东西** —— `f5c6f32` 只碰根目录那套。

**实测**：`cp package.json package-lock.json /tmp/npmtest && npm ci --offline --dry-run` → `added 3 packages in 213ms`，**通过**。这条改动是「让 lockfile 与 package.json 对齐」的卫生修复。

**风险**：极低。CI 的 frontend job 用的是 `frontend/package-lock.json`（`ci.yml:34` `cache-dependency-path: frontend/package-lock.json`），**不消费根 lockfile**。

**验证方式**：`cd <根目录> && npm ci --dry-run` 通过。

### D10 · 前端改动清单

**本轮回填的 6 条 commit 对前端的影响：零。**

- `3b402ea` / `c3119bd` / `0968289` —— 纯后端。
- `30b2079` —— 构建文件（`Dockerfile` / `deploy.sh` / `Makefile` / `ci.yml` / `.gitignore`），**不含 `.vue` / `.js`**。
- `57d03e9` —— `ocr-service/Dockerfile`。
- `f5c6f32` —— 根 `package-lock.json`（**不是 `frontend/package-lock.json`**）。

**但 `0968289` 有一处后端契约扩展，前端会「看到但用不上」**：

- `GetAnthropicDocs` 新增返回 `header_timeout_sec` 字段（`0968289:backend/handlers/ai_gateway_admin.go`）。
- `AdminCreateAnthropicProvider` / `AdminUpdateAnthropicProvider` 接受 `header_timeout_sec`。
- **前端 `frontend/src/views/ai/AIGatewayTool.vue` 的 provider 表单目前没有这个字段**：`providerForm`（`:1309`）的字段是 `name / api_url / api_key / models / aliases / enabled / is_default / default_model`；提交体（`:1372-1379`）同理，**不含 `header_timeout_sec`**。
- **这不是缺陷**：`AdminUpdateAnthropicProvider` 用 `*int` 区分「未传 = 不修改」，前端不传 → **不改动**已有值，**向后兼容**。
- **本方案不要求前端补这个输入框**（YAGNI：默认 0 = 60s 对所有云厂商够用；Ollama 由 DB seed/backfill 设 600）。
- ✅ **PM 已拍板（2026-10-22）：本轮不补。** `*int` 语义保证「不传 = 不修改」，向后兼容已核实。
  **若后续要在 UI 上调这个值 → 另开独立前端 issue**，不在本轮回填范围。

### D11 · 兼容性与迁移

| 维度 | 结论 |
|---|---|
| **旧响应体** | `GetAnthropicDocs` **新增** `header_timeout_sec` 字段（纯增量）。前端不读 → 无影响。**无字段被删改。** |
| **旧请求体** | `AdminUpdateAnthropicProvider` 的 `header_timeout_sec` 是 `*int`；不传 → `nil` → 不修改。**向后兼容。** |
| **旧 DB 数据** | 幂等 `ALTER TABLE` 补列，缺省 0 → 60s（与旧行为一致）。Ollama 行被 backfill 成 600。**无需人工回填。** |
| **旧前端** | 不受影响（`f5c6f32` 碰的是根 lockfile，CI 用的是 `frontend/package-lock.json`）。 |
| **旧配置** | `config.yaml` 的 `anthropic_providers` 不配 `header_timeout_sec` → 0 → 60s。**老配置无需改。** |
| **中间件行为** | `3b402ea` 改变**所有**经 `ResponseCompression` 的响应的压缩判定方式。既有 compress 测试全绿（实测；既有个数以 L3 的 `grep -c "^func Test"` 实测为准）。**但**线上响应头可能变化：`Content-Length` 会被删（改 chunked）。**这一条需要 jaxiu 上线后留意**（见「风险与回滚」与 L10）。 |

### D12 · 风险与回滚

#### 风险表

| # | 风险 | 等级 | 缓解 |
|---|---|---|---|
| R1 | 冲突解决时取错边 → **裸 `close` + `defer stopPreHeartbeat()` = `panic: close of closed channel`** | **高**（运行时 panic，整个后端进程挂） | D5.4 给出逐字解法 + 3 条 grep 自查（尤其「裸 close 只能有 1 处」） |
| R2 | 冲突解决时取错边 → 取 `c3119bd` 的 `h.streamClient` → **header timeout 配置完全失效**，Ollama 首字节 >60s 仍被掐死 | 高（功能未生效，但**不崩**） | D5.4 + grep 自查 ① |
| R3 | `3b402ea` 删 `Content-Length` → 响应改 chunked 传输 | 中 | 既有测试全绿；**若前端有依赖 `Content-Length` 的读进度逻辑**需回归（前端本次零改动，风险低） |
| R4 | `30b2079` 让 `docker compose build` 直接失败（缺 `askit-extension.zip`） | 中 | 有意行为；`deploy.sh docker` 与 `make docker-build` 都已自动补。**部署必须走 `./deploy.sh docker`**（R1：部署由 jaxiu 手工执行） |
| R5 | `57d03e9` 换回官方源 → 墙内构建变慢/失败 | 低-中 | 部署主机连通性由 jaxiu 上线前确认 |
| R6 | `0968289` 的 `pickStreamClient` 用 `sync.Map` 缓存 → **改了 provider 配置后旧 client 不会失效** | 低 | key 包含 `header_timeout_sec`（`streamClientCacheKey` = `name + "\|" + duration`）→ **改配置后 key 变了 → 自动重建**。设计正确。 |
| R7 | `0968289` 的 `pickStreamClient` 每次新建 `*http.Transport` → **连接池不复用** | 低 | 每个 (provider, timeout) 组合一个 client，组合数极小。`MaxIdleConnsPerHost: 16` 已配。 |
| R8 | `0968289` 改 `AnthropicProviderConfig` 结构体 → **config.yaml 多一个字段** | 极低 | 纯增量，yaml 缺省即 0。 |
| R9 | `3b402ea` 的 `decide()` 依赖「handler 在首个字节前设好 `Content-Type`」 | 低 | 4 处 SSE Content-Type 均已核实（anthropic `:558`/`:559`、`chat_stream` `:158`/`:272`、`qwen` `:282`）。**未来新写的 SSE handler 若忘了设 `Content-Type`，会被 gzip 缓冲** → 建议在「测试要点」里固化断言。 |

#### 回滚顺序（每条单独可 revert）

**原则：从新到旧逆序 revert，每条之间跑一次编译。** 因为 `3b402ea` 与 `c3119bd`/`0968289` 在**运行时**耦合（前者改中间件，后者改 handler），但**文件级不重叠**，所以逆序 revert 不会产生冲突。

```bash
# 假设回填后 main 上的新 commit 依次是：
#   C1 = 3b402ea  内容（compress.go 重构 + compress_stream_test.go）
#   C2 = c3119bd  内容（SSE 心跳串行化 + join）
#   C3 = 0968289  内容（header timeout + DB 迁移）
#   C4 = 30b2079  内容（askit 宿主机构建）
#   C5 = 57d03e9  内容（ocr 官方源）
#   C6 = f5c6f32  内容（package-lock name）

# 逐条回滚（一次一条，每条之后编译验证）：
git revert --no-edit <C6>   # 1. package-lock —— 零风险
git revert --no-edit <C5>   # 2. ocr 官方源 —— 只影响构建
git revert --no-edit <C4>   # 3. askit 宿主机构建 —— 注意：revert 后 Dockerfile 又需要 askit-builder stage
git revert --no-edit <C3>   # 4. header timeout —— 注意：revert 后 header_timeout_sec 列留在 DB 里（无害，列残留）
git revert --no-edit <C2>   # 5. SSE 心跳 —— 注意：revert 后回到「心跳不 join」的旧行为
git revert --no-edit <C1>   # 6. compress 重构 —— 注意：revert 后回到 captureWriter 全量缓冲（SSE 被吞）
```

**每步之间的验证**：

```bash
cd backend && go build ./... && go vet ./handlers/ ./middleware/ ./models/
```

**回滚注意事项（逐条核实过）**

- **回滚 `C3`（0968289）**：`ALTER TABLE` 加的 `header_timeout_sec` 列**不会被 revert 掉**（SQLite DDL 不随代码回滚）。残留列**无害**（旧代码的 SELECT 用显式列名，不会读它）。**不需要清库。**
- **回滚 `C4`（30b2079）**：revert 会把 Dockerfile 改回有 `askit-builder` stage 的版本 → **重新引入「容器内 clone 私有仓失败」的老问题**。回滚这一条意味着**接受发布会在 askit-builder 阶段失败**。**这是回滚链上最需要斟酌的一条** —— 建议**不回滚 C4**，改为单独修 Dockerfile。
- **回滚 `C1`（3b402ea）**：**必须同时回滚 `C2`/`C3` 已无必要**（文件不重叠），但**运行时会退回「SSE 被中间件吞掉」的状态** —— 524 风险回归。
- **回滚 `C6`（f5c6f32）**：零风险，纯粹是 lockfile 卫生。

**最快的「全量回退」**（若 6 条整体出问题）：

```bash
git revert --no-commit <C1>^..<C6>   # 或直接 revert 整个回填 PR
```

### D13 · 编译命令（Stage 3 / Stage 5 可直接照抄）

> **R11 铁律**：Go + CGO 后端**必须走 Docker 跨平台编译**，绝不从 macOS 直接 `go build` 出二进制。下面的 Dockerfile stage 就是项目已有的官方路径。

#### D13.1 Stage 3 —— 后端编译（Docker CGO）

**方式 1（推荐，与线上产物一致）：只跑 backend-builder stage**

```bash
# 在仓库根目录（不是 backend/）
docker build --target backend-builder -t devtools-backend-check:stage3 .
```

等价于 `Dockerfile:42`：
```
RUN CGO_ENABLED=1 GOOS=linux go build -tags with_utls -a -ldflags '-linkmode external -extldflags "-static"' -o server .
```

**方式 2（整镜像构建，含 askit 前置）**

```bash
# ⚠️ 必须先跑 build-askit，否则 Dockerfile 的 COPY askit-extension.zip 找不到源文件
./deploy.sh build-askit          # 或 SKIP_ASKIT=1 ./deploy.sh build-askit
docker build -t devtools-check:stage3 .
```

**方式 3（只做类型/语法检查，不出二进制 —— 最快）**

```bash
cd backend
go vet ./...                      # 期望 rc=0，无输出
# gofmt 只查**本轮回填触及的 8 个 Go 文件**（口径见 L2）——
# ⛔ 不得写成全仓库 `gofmt -l . | grep -v '^$'`（会命中与本轮无关的既有脏文件）
gofmt -l middleware/compress.go handlers/ai_gateway.go handlers/ai_gateway_admin.go \
  handlers/ai_gateway_anthropic.go handlers/ai_gateway_chat_stream.go \
  handlers/ai_gateway_qwen.go models/ai_gateway.go config/config.go
#   期望：无输出
```

> `go vet` 在 macOS 上跑**不违反 R11** —— R11 禁的是「从 macOS 直接出（要部署的）二进制」。`go vet` 不产出二进制。
>
> **本链口径（PM 已拍板）**：本轮的交付物是**本机静态检查**（`go build ./...` + `go vet ./...` + 触及文件的 `gofmt -l`），
> **不产出部署产物**。**Docker 跨平台编译（CGO / R11）由 jaxiu 部署时执行** —— 方式 1 / 2 在此仅作为
> jaxiu 部署时的参考路径记录，**不是本链 Stage 3/5 的放行条件**。

**⚠️ 本方案作者的实测记录（在隔离 worktree、`origin/main`=81199bd 起步、6 条全部回填并解决冲突后）**：

```
$ cd backend && go build ./...
ld: warning: ignoring duplicate libraries: '-lobjc'
build rc=0                                    ← 编译通过

$ go vet ./handlers/ ./middleware/ ./models/
vet rc=0                                      ← vet 通过

$ gofmt -l backend/handlers/ai_gateway_anthropic.go
（无输出）                                     ← 格式与语法均正常
```

#### D13.2 Stage 3 —— 前端编译（`/tmp` scratch + pnpm）

**为什么必须用 `/tmp`**：`/Volumes/M20` 是 **SMB 网络挂载**（项目已知事实）。在该盘上 `pnpm install` / `vite build` 会因大量小文件 I/O 而极慢或失败（记忆条目 `project_smb_mounted_volume.md`：SQLite commit 挂死 / native 二进制截断 / rm 超时）。**先 rsync 到本地盘再构建。**

```bash
# 1) 复制到本地盘（排除 node_modules / dist / .git）
rm -rf /tmp/devtools-fe && mkdir -p /tmp/devtools-fe
rsync -a --delete \
  --exclude 'node_modules' --exclude 'dist' --exclude '.git' \
  ./frontend/ /tmp/devtools-fe/frontend/

# 2) 安装 + 构建（在本地盘上）
cd /tmp/devtools-fe/frontend
pnpm install --frozen-lockfile
pnpm build

# 3) 把产物拷回（供 Docker 构建上下文用）
rm -rf ./frontend/dist && cp -R /tmp/devtools-fe/frontend/dist ./frontend/dist
```

> **本轮回填对前端零改动**，所以 Stage 3 的前端构建**只是「确认没被 build 链路搞坏」的例行门禁**，不是功能验证。

**前端单测**（若 Stage 5 需要）：

```bash
cd /tmp/devtools-fe/frontend
pnpm test
```

> ⚠️ **已知红项**（项目既有，非本轮回填引入）：裸跑 vitest 在 Node 26 下 `useAdminAuth` 相关的 localStorage 用例会 exit 1。**加 `--localstorage-file` 参数才绿**。记忆条目 `project_frontend_vitest_node26_localstorage.md`。
> **判据必须写成「基线同样红」才算通过**，不能写成「必须全绿」。

#### D13.3 Stage 5 —— 完整验证命令

```bash
# 后端
cd backend
go build ./... && go vet ./...
# gofmt 只查本轮回填触及的 8 个 Go 文件（⛔ 不用全仓库 `gofmt -l .`，口径见 L2）
gofmt -l middleware/compress.go handlers/ai_gateway.go handlers/ai_gateway_admin.go \
  handlers/ai_gateway_anthropic.go handlers/ai_gateway_chat_stream.go \
  handlers/ai_gateway_qwen.go models/ai_gateway.go config/config.go
#   期望：无输出
go test ./middleware/ -count=1          # 期望 ok（既有 compress 测试 + 6 个新 compress 测试；既有个数以 L3 的 grep 实测为准）
go test ./handlers/ -run 'TestPickStreamClient|TestBuildStreamTransport' -count=1   # 期望 ok（7 个新测试）
go test ./handlers/ -race -run 'TestAnthropic|TestPickStreamClient' -count=1        # 竞态检测

# 关键：**handlers 包全量在 origin/main 上就是红的**（见下），判定必须用「与基线同签名」
go test ./handlers/ -count=1 -timeout 180s

# 前端
cd /tmp/devtools-fe/frontend && pnpm install --frozen-lockfile && pnpm build
```

**handlers 包基线红项（必须写进 QA 的判据，否则会误判 FAIL）**

实测 `origin/main`（81199bd，未回填）`go test ./handlers/ -timeout 180s`：

```
--- FAIL: TestAsyncAnthropicMessages_UpstreamError (0.00s)
--- FAIL: TestDoJSONRequestNoRetryOnCanceled (0.20s)
panic: test timed out after 3m0s
FAIL	devtools/handlers	181.419s
```

实测**回填后**（6 条全部应用 + 冲突按 D5.4 解决）：

```
--- FAIL: TestAsyncAnthropicMessages_UpstreamError (0.00s)
--- FAIL: TestDoJSONRequestNoRetryOnCanceled (0.20s)
--- FAIL: TestParseClashYAML_MarksUnsupportedTypes (0.00s)
panic: test timed out after 3m0s
FAIL	devtools/handlers	181.419s
```

**判据：回填后的 FAIL 集合 ⊆ 基线 FAIL 集合，且不新增「因本轮回填导致的失败」。**
- ⚠️ `TestParseClashYAML_MarksUnsupportedTypes` 在两次运行里**出现顺序不同**（基线那次被前面的 panic 截断，没显示出来）—— 这是同一个 `proxy_mock_protocol_test.go` 挂起 goroutine 导致的**输出顺序抖动**，不是新增失败。**判据应按 FAIL 用例名集合比对，不按输出行序。**
- 项目记忆条目 `project_handlers_preexisting_test_failures.md` 早已记录：main 上就有 3 FAIL + 1 挂起。**跑单测只 `-run` 本模块，别跑全包。**

---

## 测试要点（交给 devtools-qa 的验收清单）

> 每条都是**可执行、可判定**的。P0 = 阻塞放行。

### L1 · 冲突解决正确性（P0，静态 grep，最强判据）

```bash
cd backend/handlers

# ① header timeout 派生 + 心跳 join —— 两个都必须存在
grep -n "pickStreamClient(provider)" ai_gateway_anthropic.go
#   期望：恰好 1 处命中（proxyAnthropicStream 内）

grep -n "stopPreHeartbeat()" ai_gateway_anthropic.go
#   期望：至少 3 处命中 —— defer 兜底(1) + NewRequest 错误路径(1) + 成功路径(1)

# ② 裸 close 只能有 1 处，且必须在 stopPreHeartbeat 闭包内
grep -c "close(preUpstreamHeartbeatDone)" ai_gateway_anthropic.go
#   期望：1
#   若为 2 → 裸 close 未被替换 → 运行时 panic（close of closed channel）

# ③ 无冲突残留
#   ⚠️ 必须锚定行首，否则不判别 —— 见下方说明
grep -cE '^(<<<<<<<|=======|>>>>>>>)' ai_gateway_anthropic.go
#   期望：0

# ④ 兜底自查：确认 defer stopPreHeartbeat 与裸 close 不共存
grep -n "defer stopPreHeartbeat()" ai_gateway_anthropic.go
#   期望：恰好 1 处。若与 ② 的计数矛盾 → 阻塞
```

**反事实自检（必做，否则判据恒真）**：在**被判 FAIL 的版本**上跑 ①②③，必须看到 ② 的计数为 2 或 ③ 的计数非 0。**在正确版本上 ②=1、③=0。**

> **⚠️ ③ 的写法（Stage 2 判 FAIL 的 P0 之一，已修）**：**必须**用锚定行首的
> `grep -cE '^(<<<<<<<|=======|>>>>>>>)'`。
> **不要**用「`grep -c` + 三个冲突标记、**不带 `^` 行首锚定**」的旧写法 —— 它会命中文件里**早就存在**的分节注释
> `// ===================== 异步 Anthropic 代理(task 模式,防 CF 524) =====================`
> （`origin/main` 上就有这行，`=======` 是其子串）→ **基线与正确解法都返回 1**，断言不判别，
> QA 会对一个完全正确的实现判 FAIL。
> **实测三态互不相同（可判别）**：锚定版 = 解决后 `0` / 未解决 `3` / `origin/main` 基线 `0`。

### L2 · 编译与静态检查（P0）

```bash
cd backend
go build ./...            # 期望 rc=0
go vet ./...              # 期望 rc=0，无输出

# gofmt：**只查本轮回填触及的 8 个 Go 文件**
gofmt -l middleware/compress.go handlers/ai_gateway.go handlers/ai_gateway_admin.go \
  handlers/ai_gateway_anthropic.go handlers/ai_gateway_chat_stream.go \
  handlers/ai_gateway_qwen.go models/ai_gateway.go config/config.go
#   期望：无输出
```

> **⚠️ gofmt 判据口径（PM 已拍板，Stage 2 判 FAIL 的 P0 之一）**
>
> **主判据 = 只查上面这 8 个文件。** ⛔ **不得**把「全仓库 `gofmt -l .` 无输出」写成闸门。
>
> **原因**：本机工具链是 **go1.26.6**（`go version`），项目钉的是 **go1.25.5**
> （`Dockerfile:28` `FROM golang:1.25.5-alpine`、`backend/go.mod:3` `go 1.25.5`）。
> go1.26 改了行尾注释对齐规则（如 `avatarMaxFrameCount = 60000 // R10:…` 的对齐），
> 于是在 go1.26.x 上跑全仓库 `gofmt -l .` 会列出一批**与本轮回填完全无关**的既有文件。
> **实测**（本 worktree，`backend/` 下）：`gofmt -l . | wc -l` = **34**，
> 而这批文件与本轮回填触及的 8 个文件**交集为空** —— 用全仓库口径会把一个正确的实现判 FAIL。
>
> **与 Go 版本的关系**：这条闸门**只查触及文件**，实测这 8 个文件在 go1.25.5 / go1.26.x 下**都干净**
> → **本闸门与执行机的 Go 版本无关**。QA 执行机以 `go.mod` 的 `go 1.25.5` 为准；若开发机是 go1.26.x，
> 同样适用本闸门（不会因此误判）。
>
> **「与基线同集」只能作辅助证据**（`diff <(gofmt -l . | sort) <(cd <基线检出> && gofmt -l . | sort)`），
> **不得作为唯一判据**。

### L3 · 压缩中间件行为（P0）

```bash
cd backend
go test ./middleware/ -count=1 -v -run 'TestCompress' 2>&1 | grep -E '^(--- PASS|--- FAIL|ok|FAIL)'
```

**期望**：
- `3b402ea` 新增的 6 个测试全 PASS：
  `TestCompress_SSEStreamNotBuffered_RealAnthropicRoute`、`TestCompress_SSEStreamNotBuffered_RealQwenVisionRoute`、`TestCompress_SSEWithAcceptHeaderStillUnbuffered`、`TestCompress_NonStreamAnthropicJSONStillCompressed`、`TestCompress_FlushAdvancesCompressedStream`、`TestCompress_NoContentNotCompressed`
- 既有 compress 测试**全 PASS**。
- 合计 `ok devtools/middleware`。

> **⚠️ 不要在判据里写死测试个数（PM 已拍板）**：既有用例的个数以**实测输出**为准，
> 不在文档里钉死数字（钉死就会随上游增删测试而失真）。**闸门用命令，不用数字：**
>
> ```bash
> cd backend
> # 既有 compress 测试个数（实测口径）
> grep -c "^func Test" middleware/compress_test.go
> grep -c "^func Test" middleware/compress_qa_test.go
> grep -c "^func Test" middleware/compress_stream_test.go
>
> # 全量 PASS 口径
> go test ./middleware/ -run "Compress" -count=1 -v 2>&1 | grep -cE '^--- PASS'
> #   期望：与上面三个 grep 的计数之和一致；exit 0
> ```
>
> **参考值（PM 在基线实测，仅作参照，不作判据）**：`go test ./middleware/ -run "Compress" -count=1 -v`
> → **32 PASS / exit 0**。

**关键语义断言（这 6 个新测试的实质，QA 要理解）**：

| 断言 | 含义 |
|---|---|
| `SSEStreamNotBuffered_RealAnthropicRoute` | 真实 `/api/anthropic/v1/messages` 路由上，handler 执行途中已 Flush 的字节**必须已在 writer 上**（证明无全量缓冲） |
| `NonStreamAnthropicJSONStillCompressed` | 同一条路径的**非流式 JSON 仍被压缩**（证明没有为了 SSE 把整条路径排除） |
| `FlushAdvancesCompressedStream` | 压缩路径下 `gz.Flush()` 真下推 |
| `NoContentNotCompressed` | 204/304 不压缩 |

### L4 · header timeout（P0）

```bash
cd backend
go test ./handlers/ -count=1 -v -run 'TestPickStreamClient|TestBuildStreamTransport' 2>&1 | grep -E '^(--- PASS|--- FAIL|ok|FAIL)'
```

**期望 7 个全 PASS**（**数字与名单必须一致，改一个就要改另一个**）：
`TestBuildStreamTransport_DefaultWhenZero`（0 → 60s）、`TestBuildStreamTransport_NegativeFallsBack`（负数 → 60s）、`TestBuildStreamTransport_CustomValue`（600 → 600s）、`TestPickStreamClient_NilProviderUsesFallback`（nil → 全局兜底）、`TestPickStreamClient_DerivesPerProvider`、`TestPickStreamClient_CachesSameKey`、`TestPickStreamClient_DifferentTimeoutRebuilds`。

**注意**：`0968289` 的 commit message 说「7 个单测」，实测该文件里是 **7 个 Test 函数**（3 个 `buildStreamTransport` + 4 个 `pickStreamClient`）。**上面列了 7 个 —— 数字与名单一致。**

> ⚠️ 早前本清单写成「期望 **4** 个全 PASS」而列了 7 个名字（自相矛盾）—— 已按实测纠正为 **7**。

### L5 · DB 迁移幂等（P0，行为判据）

**判据：连续两次初始化，第二次不报错、且列不重复添加。**

> **⚠️ 口径变更（Stage 2 判 FAIL 的 P0 之一）**：本条**不再起服务进程**。
> 早前写法是「用 `go run` 把 `main.go` 拉成后台进程」—— **违反项目 R1**（智能体不跑 `go run`、不启动服务、
> 只写单测）。若把它固化进验收清单，等于把违规动作固化成 Stage 5 的放行条件。
> 现改为 **`backend/models` 包内的一次性 Go 单测**（临时 DB，`t.TempDir()`），
> **该单测本身就是交付物，不依赖任何常驻服务**。

**实现要求（交给 devtools-backend / devtools-qa 落地，文件名自定，建议 `backend/models/ai_gateway_migrate_test.go`）**

```go
// 伪代码骨架 —— 关键是三点：
//   1) DB 落在 t.TempDir()（绝不落 /Volumes/M20：SMB 盘上 SQLite 会挂死）
//   2) 连启两次 InitAnthropicProviders 验证幂等
//   3) 直接查 PRAGMA table_info 与行值断言，不依赖 sqlite3 CLI
func TestAnthropicProviderInit(t *testing.T) {
    dir := t.TempDir()
    db := openTestDB(t, filepath.Join(dir, "migrate-test.db"))

    // 第一次初始化
    if err := db.InitAnthropicProviders(); err != nil {
        t.Fatalf("first init: %v", err)
    }
    // 第二次初始化（幂等）：不得因 duplicate column 以外的错误失败
    if err := db.InitAnthropicProviders(); err != nil {
        t.Fatalf("second init must be idempotent, got: %v", err)
    }

    // 断言：header_timeout_sec 列恰好 1 个
    //   SELECT COUNT(*) FROM pragma_table_info('anthropic_providers') WHERE name='header_timeout_sec'
    //   期望：1

    // 断言：Ollama 行 header_timeout_sec = 600
    //   SELECT header_timeout_sec FROM anthropic_providers WHERE name='Ollama'   期望：600
    // 断言：其它 provider（MiniMax/DashScope/DeepSeek/PackyAPI/OpenClaudeCode）= 0
    //   SELECT name, header_timeout_sec FROM anthropic_providers WHERE name!='Ollama'
}
```

**执行命令（QA 可直接跑）**

```bash
cd backend
go test ./models/ -run TestAnthropicProviderInit -count=1 -v
#   期望：--- PASS: TestAnthropicProviderInit
#   期望 rc=0
```

**⚠️ SQLite `:memory:` 陷阱**：若单测用 `:memory:` 而不是 `t.TempDir()`，
**必须 `SetMaxOpenConns(1)`** —— 否则后台 goroutine 触发「刚插入查不到」的假 404
（项目记忆 `feedback_sqlite_memory_pool.md`）。用文件 DB（`t.TempDir()`）则无此问题。

**反事实**：在**未修 `0968289`** 的版本上跑同一单测 → 列不存在 / 第二次初始化报 `duplicate column` → FAIL。

### L6 · 旧明文懒迁移不再清零（P0，本轮回填最容易被漏掉的行为）

**判据：一个 legacy 明文 provider（`api_key` 非空、`api_key_encrypted` 为空）被读一次后，其余字段不得被清零。**

> **⚠️ 同 L5 的口径变更**：**不起服务进程**（早前写法依赖 L5 那个后台跑 `main.go` 的进程与同一个 DB 文件，
> 同样违反 R1）。改为 **`backend/handlers` 包内的 `httptest` 单测**：直接打一次
> `dbAnthropicProvidersToConfig` 路径（该函数是懒迁移的触发点），再断言行值未被清零。

**实现要求（交给 devtools-backend / devtools-qa 落地，建议 `backend/handlers/ai_gateway_anthropic_lazy_migrate_test.go`）**

```go
// 伪代码骨架 —— 关键点：
//   1) 造一行 legacy 数据：api_key 非空 / api_key_encrypted 为空 / header_timeout_sec=600
//      / models='["m1"]' / aliases='[]' / enabled=1 / default_model='m1'
//   2) 走一次 dbAnthropicProvidersToConfig（懒迁移在此触发）
//   3) 回查该行，断言 models/enabled/default_model/header_timeout_sec 未被清零
func TestAnthropicLegacyPlaintextLazyMigrate(t *testing.T) {
    // 临时 DB（t.TempDir()），插入 legacy 行
    // 调用 dbAnthropicProvidersToConfig(...)
    // 回查：
    //   api_key_encrypted != ''   期望：真（已完成懒迁移）
    //   api_key                   期望：空（已清空）
    //   models                    期望：["m1"]   （没被清零）
    //   enabled                   期望：1        （没被清零）
    //   default_model             期望：m1       （没被清零）
    //   header_timeout_sec        期望：600      （没被打回 0）
}
```

**执行命令（QA 可直接跑）**

```bash
cd backend
go test ./handlers/ -run TestAnthropicLegacyPlaintextLazyMigrate -count=1 -v
#   期望：--- PASS: TestAnthropicLegacyPlaintextLazyMigrate
#   期望 rc=0
```

> ⚠️ `handlers` 包**全量**在 `origin/main` 上就是红的（3 FAIL + 1 挂起，见 D13.3）——
> **必须用 `-run` 只跑本单测**，不要跑全包（项目记忆 `project_handlers_preexisting_test_failures.md`）。

**反事实**：在**未修 `0968289`** 的版本上跑同一单测 → `models` 变 `[]`、`enabled` 变 0、
`header_timeout_sec` 变 0。**这是本条判据的可证伪性证明。**

### L7 · SSE 无缓冲（集成级，P0）

**判据：经中间件的 SSE 响应，首字节延迟 ≠ 整段生成时间。**

```bash
# 起一个最小 SSE 端点（建议在 QA 侧写一个临时 _test.go，或用 3b402ea 新增的
# compress_stream_test.go 里的真实路由用例）
# 观测点：handler 尚未返回时，客户端已收到 ping 帧
```

**可执行替代（推荐，纯 Go 单测，已随 `3b402ea` 提供）**：`go test ./middleware/ -run 'TestCompress_SSEStreamNotBuffered' -v` —— 断言「handler 执行途中已 Flush 的字节已在 writer 上」。**这就是「不缓冲」的可判定形式**，不需要真机。

**若做真机验证（由 jaxiu 合并后自验，不阻塞放行）**：用 Claude Code 经 `https://t.jaxiu.cn/api/anthropic` 发起一个长回复请求，观测 **TTFB 明显小于总生成时间**，且日志里能看到周期性 ping 帧。

### L8 · 构建链路（P1）

> **⚠️ 范围收窄（PM 已拍板）**：`deploy.sh` 属 **R1 禁跑清单**（智能体不跑 `./deploy.sh`）。
> 因此 **`SKIP_ASKIT=1 ./deploy.sh build-askit` 不在 QA 的可执行范围内** ——
> 该命令**由 Stage 3 的执行者跑**；**QA 只查产物存在性与静态契约**。

```bash
# 1) deploy.sh 语法（静态检查，不执行脚本）
bash -n deploy.sh                  # 期望 rc=0

# 2) 占位包产物存在性（**由 Stage 3 执行者跑完 build-askit 之后**，QA 只查产物）
ls -la askit-extension.zip          # 期望：文件存在
unzip -l askit-extension.zip        # 期望：含 README.txt
#   ⚠️ 若 QA 环境里没有该产物（因为 build-askit 未执行）→ 本条按「不在 QA 范围」跳过，
#      **不要**因此判 FAIL。

# 3) CI 契约（静态 grep，QA 可执行）
grep -n "SKIP_ASKIT=1 ./deploy.sh build-askit" .github/workflows/ci.yml
#   期望：恰好 1 处命中

# 4) .gitignore / .dockerignore（静态 grep，QA 可执行）
grep -n "askit-extension.zip" .gitignore
#   期望：恰好 1 处
grep -n "askit-extension.zip" .dockerignore
#   期望：0 处（必须能被带进构建上下文）
```

### L9 · package-lock（P1）

```bash
cp package.json package-lock.json /tmp/fin83-npmtest/ && cd /tmp/fin83-npmtest && npm ci --dry-run
#   期望 rc=0，"added 3 packages"
# 静态：
grep -c '"name": "devtools"' package-lock.json
#   期望：2（顶层 1 处 + packages[""] 1 处）
```

### L10 · 未覆盖项（**不阻塞放行**，由 jaxiu 合并后自验）

- **真机 375×667 首屏 / iOS 锁屏 / Wake Lock / 线上容器行为** → 归 jaxiu（D5）。
- **CF 实际 524 是否消失** → 需要真实长回复 + 公网链路 → 归 jaxiu。
- **墙内 Docker 构建速度 / 官方源连通性** → 归 jaxiu。
- **`/Volumes/M20` 上的 SQLite 迁移性能** → 归 jaxiu（生产 DB 在宿主）。
- **前端 `pnpm build` 产物字节级一致** → 本轮前端零改动，L13 门禁足够。
- **依赖 `Content-Length` 的响应 → 改 chunked 后的下游表现** → 归 jaxiu 上线后看一眼。
  （D11 已写明：`3b402ea` 的 `decide()` 会 `header.Del("Content-Length")` → 压缩响应改 chunked 传输。
  **前端本次零改动**，无进度条相关改动；但若有依赖 `Content-Length` 的读进度逻辑，线上表现需 jaxiu 自验。
  **本条不新增可执行命令**，只是人工确认口径。）
- **生产 DB 的 `ALTER TABLE` 迁移窗口** → 归 jaxiu 部署时确认（PM 已拍板，**不进本轮验收清单**）。

---

## 边界裁决（PM 已全部拍板 · 2026-10-22）

> 以下 6 条原本是「未决问题」。**devtools-pm 已于 2026-10-22 全部拍板**（依据：FINN-84 评审报告的「未定义边界 1–6」）。
> **下游无条件照此执行，不再自行取舍。**

1. **ssrule 模块（`e7f37da`）是否一并回填 → ✅ 裁决：本轮不回填。**
   `3b402ea` 的 `excludedPathPrefixes` 新增了 `"/ss/"`（`backend/middleware/compress.go:261`），
   但 `ssrule` 模块（`backend/handlers/ssrule.go`、`backend/models/ssrule.go`、`backend/routes/ssrule.go`、
   `frontend/src/views/other/SsRuleTool.vue`）**不在 `origin/main` 上**
   （实测 `git ls-tree -r origin/main --name-only | grep -i ssrule` → **0 命中**；
   该模块在 `e7f37da` 这条未合入的分支上）。
   - **裁决**：`"/ss/"` **保持原样**（cherry-pick 保真优先），**无功能影响**。
   - **⚠️ 纠正理由表述**：`"/ss/"` **不是** `"/s/"` 的冗余前缀 ——
     实测 `'/ss/foo'.startswith('/s/')` → **`False`**（`"/s/"` 不会前缀匹配 `"/ss/"`）。
     它只是**当前无路由命中的死条目**。**结论不变（不回填、无影响），但理由按此改写**，
     不要把「冗余前缀」当作理由 —— 两者性质不同（冗余条目会意外生效，死条目只是不生效）。

2. **前端是否补 `header_timeout_sec` 输入框 → ✅ 裁决：本轮不补。**
   `*int` 语义保证「不传 = 不修改」，向后兼容已核实（见 D10）。
   **要补 → 另开独立前端 issue**，不在本轮。

3. **`UpdateAnthropicProvider` 是否加「调用方必须传完整行」注释 → ✅ 裁决：加。**
   1 行注释成本，钉住「未来新增部分字段调用点会重现『清零 provider』」这颗雷。
   **落在 `backend/models/ai_gateway.go` 的 `UpdateAnthropicProvider` 上**（见 D6.4）。
   **不重构为 partial-update**（接口语义变更，另开 issue）。

4. **生产 DB 迁移窗口（`ALTER TABLE` 需写锁） → ✅ 裁决：归 jaxiu 部署时确认，不进本轮验收清单。**
   `ALTER TABLE anthropic_providers ADD COLUMN header_timeout_sec` 在**进程启动时**执行
   （`RegisterInit("AI Gateway(anthropic_providers)", (*DB).InitAnthropicProviders)`，
   `backend/models/ai_gateway.go:14`）。SQLite 的 `ALTER TABLE ADD COLUMN` 是 O(1) 元数据操作、
   **不重写表**，秒级完成，但需要**写锁**；单实例部署（`docker-compose.yml` 只有 1 个 devtools 容器）
   → **无实际风险**。
   **方案里写明归属即可**（已写入 L10「未覆盖项」），**不作为 QA 放行条件**。

5. **回滚 `30b2079`（askit 改宿主机构建）的策略 → ✅ 裁决：不回滚 C4，改为单独修 Dockerfile。**
   revert 会重新引入「容器内 clone 私有仓失败」（见 D12 的回滚注意事项）。
   **方案 D12 的结论保持原样，本轮不改 D12。**

6. **B2 的 gofmt 判据口径 + QA 执行机 Go 版本 → ✅ 裁决：主判据 = 只查本轮回填触及的 8 个 Go 文件。**
   - ⛔ **不得**把「全仓库 `gofmt -l .` 无输出」写成闸门。
   - **QA 执行机 Go 版本**：以 `backend/go.mod:3` 的 `go 1.25.5` 为准；
     若开发机是 go1.26.x，**本闸门与 go 版本无关**（只查触及文件，实测两个版本下都干净）。
   - **「与基线同集比对」可作辅助证据，不得作为唯一判据。**
   - 完整命令与实测数据见 **L2**。

**未决问题：无。** 以上 6 条边界全部已裁决，下游无需再问。

---

## 附：本方案已执行的验证动作清单

> 全部在**隔离 worktree**（`/tmp/fin83sim`，从 `origin/main`=81199bd 建）里做的，**未触碰真实仓库的任何分支**，验证完已 `git worktree remove` + 删临时分支。**未起 dev server、未跑 `deploy.sh`、未 `docker build`、未连任何部署主机。**

| # | 动作 | 结果 |
|---|---|---|
| 1 | `git cat-file -e` × 6 commit | 全部 OK |
| 2 | `git fetch origin` + `git rev-parse origin/main` | `81199bd`（PM 给的 `542e457` 已过期） |
| 3 | `git diff --stat 542e457 origin/main` | 10 文件，全前端/docs，**与 6 条零重叠** |
| 4 | 隔离 worktree cherry-pick 5 条无冲突 commit | 全部 CLEAN（两种顺序） |
| 5 | cherry-pick `0968289` × `c3119bd` 两种顺序 | 各 **1 个冲突文件、1 个冲突块** |
| 6 | 冲突块原文 dump（两种顺序） | 已录入 D5.1，**互为镜像** |
| 7 | 最小 Go 程序复现双 close | **`panic: close of closed channel`**（D5.3） |
| 8 | 按 D5.4 解法解决冲突 → `gofmt -l` | 无输出 |
| 9 | 合并后 `cd backend && go build ./...` | **rc=0** |
| 10 | 合并后 `go vet ./handlers/ ./middleware/ ./models/` | **rc=0** |
| 11 | `3b402ea` 后 `go test ./middleware/` | **ok**（既有 + 6 新，全 PASS；既有个数见 L3 的 grep 实测口径） |
| 12 | 合并后 `go test ./handlers/ -run 'TestPickStreamClient\|TestBuildStreamTransport' -v` | **7 个 PASS**（另跑 targeted `-run 'TestAnthropic\|TestPickStreamClient\|TestAIGatewayProvider'` 亦 ok） |
| 13 | `origin/main` 基线 `go test ./handlers/ -timeout 180s` | **FAIL**（3 FAIL + 1 挂起，与记忆条目一致） |
| 14 | 合并后 `go test ./handlers/ -timeout 180s` | **FAIL**（3 FAIL + 1 挂起，**同一集合**，无新增） |
| 15 | `npm ci --dry-run`（根 package.json + 合并后 lock） | **rc=0, added 3 packages** |
| 16 | `git diff e7f37da origin/main -- backend/middleware/` | **空**（compress 及其既有测试在 main 上与 `3b402ea` 父提交完全一致 → cherry-pick CLEAN 的根因） |
| 17 | `grep` 核实 `excludedPathPrefixes` 新旧差异 | 新增 `"/ss/"` 一条 |
| 18 | `grep` 核实 `UpdateAnthropicProvider` 全部调用点 | 合并后 3 处（懒迁移 goroutine 已修 / admin 全行安全 / 测试） |
| 19 | `grep` 核实 `askit_extension.go` 在 main 上 + `:128` 行内容 | 在，`:128` 是 `git clone --depth 1 --branch askitBranch askitRepoSSH`；`askitBranch="main"`、`askitRepoSSH="git@github.com:axfinn/Askit.git"` 与 `build_askit` **完全对齐** |
| 20 | `grep` 核实 `.dockerignore` 不排除 `askit-extension.zip` | 0 命中 → 构建上下文能带上 |

**未核实项（明确写出，不默认当已覆盖）**：
- 未跑 `docker build`（任何 target）—— 需 R11 的 Docker 跨平台编译，由 Stage 3/5 执行。
- 未跑 `./deploy.sh build-askit`（会写文件到仓库根）。
- 未做真机 / 线上验证（CF 524、375×667、容器行为）。
- 未跑前端 `pnpm build`（前端零改动；且需先 rsync 到 `/tmp`）。
- 未验证生产 DB 上 `ALTER TABLE` 的实际耗时与锁行为。
- 未验证 `ssrule` 模块（`e7f37da`）与本轮 6 条的关系 —— 该模块不在 main 上，已由 PM 裁决「本轮不回填」（见「边界裁决」第 1 条）。

### 附二：Stage 2 复查修订记录（2026-10-22 · FINN-105）

FINN-84（devtools-design-reviewer）判 **FAIL**，3 条阻塞项全部落在**验收清单层**（设计主体经其独立复现成立）。
本轮修订**只改验收命令与文档口径，不动 D1–D13 的设计结论、不动冲突解法**。

| # | 评审项 | 修订位置 | 修订内容 |
|---|---|---|---|
| **B1** | 冲突残留 grep 不判别（正确实现也返回 1） | D5.4 ③ / L1 ③ | 改为锚定行首 `grep -cE '^(<<<<<<<\|=======\|>>>>>>>)'`（**两处都改**），并写明旧写法为何恒真、三态实测值 |
| **B2** | `gofmt -l .` 不判别（正确实现返回 35 行） | D13.1 方式 3 / D13.3 / **L2** | 主判据改为**只查本轮回填触及的 8 个 Go 文件**；⛔ 不得用全仓库 `gofmt -l .`；写明 go1.26 vs go1.25.5 的根因与「本闸门与 Go 版本无关」 |
| **B3** | L5 用 `go run` 起服务违反 R1，且 L6 依赖它 | **L5 / L6** | 全部改为**包内一次性 Go 单测**（`backend/models` 临时 DB + `backend/handlers` `httptest`），**不起服务进程**；顺带消除 `sqlite3` CLI 依赖 |
| N1 | compress 测试数量算错（写 27，实为 26） | D3 / D11 / D13.3 / **L3** | ⛔ 不写死数字 → 改为 `grep -c "^func Test"` 实测口径（命令写进文档），参考值 32 PASS |
| N2 | L4「期望 4 个」与 7 个名单自相矛盾 | **L4** | 改为 **7 个**，数字与名单绑定 |
| N3 | D13.1「最终交付的二进制必须走方式 1 或 2」与本链口径冲突 | **D13.1** | 删掉该句 → 改为「本链口径 = 本机静态检查，不产出部署产物；Docker 跨平台编译由 jaxiu 部署时执行」 |
| N4 | `ci.yml:12` `GO_VERSION: '1.21'` vs `go.mod` `go 1.25.5` | **D7** | 记为**已知既有项，本轮不修**，文档不承诺修它 |
| N5 | `compress.go` 压缩路径 `Size()` 只统计已吐出的字节 | **D3 风险** | 记为**已知偏差，本轮不改**（非功能缺陷） |
| N6 | `compress.go:149` `header.Set("Vary", …)` 覆盖既有 Vary | **D3 风险** | 记为**已知项，本轮不改**（cherry-pick 保真），PM 另开跟进单 |
| N7 | L8 把 `deploy.sh build-askit` 列为 QA 可执行命令 | **L8** | 从 QA 范围删除 → 改为「由 Stage 3 执行者跑；QA 只查产物存在性」，并注明缺产物**不判 FAIL** |
| N8 | D11 的 `Content-Length` → chunked 缺下游判定口径 | **L10** | 补一条人工确认口径（前端零改动，进度条场景由 jaxiu 上线后看一眼），**不新增可执行命令** |
| 裁决 1–6 | ssrule / 前端输入框 / 注释 / 迁移窗口 / 回滚 C4 / gofmt 口径 | **「边界裁决」章节** | 全部按 PM 拍板写定；`"/ss/"` 的理由改写为「无路由命中的死条目」并纠正「不是 `"/s/"` 冗余前缀」 |

**本轮修订的实测取证（本次亲自跑过）**

```
$ go version
go version go1.26.6 darwin/arm64

$ grep -n "^go " backend/go.mod          → 3:go 1.25.5
$ grep -n "golang:" Dockerfile             → 28:FROM golang:1.25.5-alpine AS backend-builder
$ grep -n "GO_VERSION" .github/workflows/ci.yml → 12:  GO_VERSION: '1.21'

# B1 —— 锚定版可判别（未解决 3 / 解决后 0 / 基线 0）
$ git show origin/main:backend/handlers/ai_gateway_anthropic.go | grep -cE '^(<<<<<<<|=======|>>>>>>>)'   → 0
$ git show origin/main:backend/handlers/ai_gateway_anthropic.go | grep -c <三个冲突标记，无 ^ 锚定>            → 1   ← 旧写法：基线也是 1 → 不判别
$ grep -n '=======' backend/handlers/ai_gateway_anthropic.go
1023:// ===================== 异步 Anthropic 代理(task 模式,防 CF 524) =====================   ← 旧写法命中的就是这行

# B2 —— 只查触及文件无输出；全仓库有 34 个与本轮无关的脏文件
$ cd backend && gofmt -l middleware/compress.go handlers/ai_gateway.go handlers/ai_gateway_admin.go \
    handlers/ai_gateway_anthropic.go handlers/ai_gateway_chat_stream.go \
    handlers/ai_gateway_qwen.go models/ai_gateway.go config/config.go
（无输出）
$ cd backend && gofmt -l . | wc -l            → 34
$ comm -12 <(gofmt -l . | sort) <(printf '%s\n' <上面 8 个文件> | sort)   → （空，交集为空）

# N1 —— compress 测试实际个数
$ grep -c "^func Test" middleware/compress_test.go      → 13
$ grep -c "^func Test" middleware/compress_qa_test.go   → 13
$ grep -c "^func Test" middleware/compress_stream_test.go → 6
（既有 13+13=26，合计 26+6=32）

# N2 —— header timeout 测试 7 个
$ git show 0968289:backend/handlers/ai_gateway_header_timeout_test.go | grep -n "^func Test"
17:TestBuildStreamTransport_DefaultWhenZero   24:TestBuildStreamTransport_NegativeFallsBack
31:TestBuildStreamTransport_CustomValue        38:TestPickStreamClient_NilProviderUsesFallback
48:TestPickStreamClient_DerivesPerProvider     80:TestPickStreamClient_CachesSameKey
94:TestPickStreamClient_DifferentTimeoutRebuilds

# N6 —— compress.go:149
$ git show 3b402ea:backend/middleware/compress.go | grep -n "Vary"
149:	header.Set("Vary", "Accept-Encoding")

# 裁决 1 —— "/ss/" 不是 "/s/" 的冗余前缀
$ python3 -c "print('/ss/foo'.startswith('/s/'))"   → False
$ git ls-tree -r origin/main --name-only | grep -i ssrule   → 0 命中
```

**本轮修订未做的事（明确写出，不默认当已覆盖）**：
- **未验证 `grep -cE '^(<<<<<<<|=======|>>>>>>>)'` 在「未解决」状态下返回 3** —— 该值来自评审方的实测报告，
  本次只在「基线 / 当前工作副本」两个状态上复现了 `0`（未解决态需构造冲突文件才能复现，未构造）。
- **未运行任何 `go test`**（L5/L6 的新单测尚未存在 —— 它们是**本轮之后**的交付物，不是本方案的一部分）。
- **未跑 `docker build` / `deploy.sh` / `go run`**（R1 + R11）。
- **未在 go1.25.5 工具链下复跑 gofmt** —— 当前机器只有 go1.26.6；「两个版本下都干净」这一结论来自评审方实测。
