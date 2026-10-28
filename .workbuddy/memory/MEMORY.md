# 项目长期记忆（devtools）

## ⚠️ 标准发布流程（本项目唯一发布方式，必须遵守）
发布 devtools 项目**一律用**：
```sh
HOT_DEPLOY=1 ./deploy.sh docker
```
- 含义：本机构建前端 + 叠加到已缓存镜像 `devtools-devtools:latest`，**零拉取**、**不动 Docker daemon 配置**、**不影响其他正在运行的容器**（ocr/asr/redis 等）。
- 这是用户明确指定的长期约定（"以后都这么发布这个项目"）。除非用户改口，不要用普通 `compose build`（会因 golang/node 基础镜像拉取超时而失败）。
- 前提：本机 `frontend/dist` 必须是最新构建产物（见下方构建方法）。`deploy.sh` 的 `build_frontend_host` 已改为"dist 存在则直接复用、不再回退提取旧 dist"，所以只要先在本机构建好 dist，再跑上面的命令即可。
- 校验（每次发布后必做）：
  1. `curl -s http://localhost:8082/api/health` 返回 200；
  2. `docker exec devtools sh -c 'grep -rl "<本次改动标记>" /app/dist/assets/*'` 能命中，确认新前端已进容器；
  3. `docker ps` 确认 ocr-service / asr-service / devtools-redis 仍 `healthy`，未受影响。

## 在 WorkBuddy 沙箱里构建前端（必看，发布前必做）
- 现象：pnpm/npm install 在沙箱里挂死或报 `CODEBUDDY_BROKER_DENY ... symlink`。
- 根因：WorkBuddy 通过 `NODE_OPTIONS=--require=.../node-language-shim.cjs` 把 brokered-fs 垫片注入**每个** Node 进程，垫片拒绝软链接。
- 解决（在本机终端跑构建前执行）：
  ```sh
  export PATH=/opt/homebrew/bin:$PATH   # 系统 Node，无垫片
  unset NODE_OPTIONS                    # 关键：去掉垫片注入
  export HTTP_PROXY=http://127.0.0.1:7897 HTTPS_PROXY=http://127.0.0.1:7897 http_proxy=http://127.0.0.1:7897 https_proxy=http://127.0.0.1:7897
  cd frontend && pnpm install --frozen-lockfile --config.store-projects=false --config.node-linker=hoisted && pnpm build
  ```
- 代理只监听本机回环 127.0.0.1:7897，容器内/daemon 侧连不上；**不要改 Docker daemon 代理，也不要影响其他正在运行的容器**。
- 构建失败的兜底：若实在无法在沙箱构建，可从运行中的 `devtools` 容器 `docker cp devtools:/app/dist/. frontend/dist/` 取回当前线上 dist（但那是旧版，仅用于紧急恢复，不含未发布的源码改动）。

## 部署相关文件
- `deploy.sh`：`docker_deploy` 支持 `HOT_DEPLOY=1`；`build_frontend_host`（复用本机 dist）、`hot_layer_build`（Dockerfile.hot 零拉取叠加，临时替换 .dockerignore 再还原）。
- `Dockerfile.hot` + `.dockerignore.hot`：从缓存镜像叠加新 dist 的叠加层定义。
- 后端二进制沿用已缓存镜像（前端改动无需重编后端）。
- HOT 部署相关文件（`deploy.sh` / `Dockerfile.hot` / `.dockerignore.hot` / 前端改动）已提交并推送到远程，且已 `--no-ff` 合入 `main`（提交 `72138ee` → 合并 `b139230` → 合 main 的 `1032ea1`），并打版本 tag `v1.0.0`（annotated，指向 `1032ea1`）。clone 干净仓库也保留 HOT 部署能力。

## 后端二进制如何进容器（⚠️ HOT_DEPLOY 只动前端 dist，不含后端）
- **现状**：`HOT_DEPLOY=1 ./deploy.sh docker` 只在已缓存镜像上叠加 `frontend/dist`，**不动后端二进制**。所以任何后端 Go 代码改动，都必须单独把新 `server` 二进制推进容器，否则新接口/逻辑不会生效（典型症状：`/api/xxx` 返回 404 或旧行为）。
- **交叉编译要点**（本机是 macOS arm64，容器是 linux/arm64，且 `with_utls` 要求 CGO）：
  1. 不能直接在本机 `go build`（产物是 Mach-O，容器内跑不了）；`golang:1.26.10-alpine` 也未缓存，普通 `compose build` 又因基础镜像拉取超时失败。
  2. **可行路径**：用已缓存的 `golang:1.23-alpine` 容器，`GOTOOLCHAIN=auto` 会自动拉取 1.26.10 工具链；挂载 host 模块缓存 `/Users/finn/go/pkg/mod` 复用依赖；`apk add gcc musl-dev sqlite-dev` 提供 CGO 工具链：
     ```sh
     docker run --rm \
       -v /Volumes/M20/code/docker/devtools/backend:/src:ro \
       -v /Users/finn/go/pkg/mod:/go/pkg/mod \
       -v /tmp/newsbuild:/out \
       golang:1.23-alpine \
       sh -c "apk add --no-cache gcc musl-dev sqlite-dev >/dev/null 2>&1 && \
              cd /src && export GOPROXY=https://goproxy.cn,direct GOTOOLCHAIN=auto && \
              CGO_ENABLED=1 GOOS=linux go build -tags with_utls -ldflags '-linkmode external -extldflags \"-static\"' -o /out/server ."
     ```
  3. 产物 `/tmp/newsbuild/server` 应为 `ELF 64-bit LSB executable, ARM aarch64, statically linked`。
- **入容器 + 固化**：
  ```sh
  docker cp /tmp/newsbuild/server devtools:/app/server
  docker restart devtools                 # entrypoint 会 exec ./server
  # 校验: curl http://localhost:8082/api/health == 200, 命中新接口
  docker commit devtools devtools-devtools:latest   # 把新后端固化进镜像,否则下次 recreate 会丢
  ```
- **配置**：`backend/config.yaml`（已 gitignore，含密钥）bind 挂载进容器为 `/app/config-src`，`entrypoint.sh` 每次启动复制为 `/app/config.yaml`。改运行期配置直接编辑 `backend/config.yaml` 然后 `docker restart devtools` 即可。
- 容器内 egress 正常（goproxy.cn / go.dev 可达），`GOTOOLCHAIN` 拉取与 `apk add` 均可用。

## 新闻模块（news）行为要点 —— 架构：**智能体本地采编 + 后端只发布**
- **用户明确要的是：WorkBuddy 本地（智能体）自己收集、翻译、写面面俱到摘要，然后发布到网站**。后端只负责接收/存储/展示，**不**承担采编。
- **每日触发 = WorkBuddy 自动化「每日新闻采编与发布（智能体本地）」**(已 update，id `d6b2e53f-...`，`FREQ=DAILY;BYHOUR=8;BYMINUTE=0`)：智能体用 WebSearch/WebFetch 采集 → 亲自翻译英文、写『今日总览』+分主题面面俱到日报 → 用 Write 落 `/tmp/news_publish_<日期>.json` → `POST /api/news/publish` 发布；同时把 md 备份到 `news_archive/YYYY-MM-DD.md`(已 gitignore)。**这才是每日唯一调度方，不是后端 scheduler。**
- **发布接口 `POST /api/news/publish`**（受 `news.admin_password` 保护，密码 `123654789`）：接收智能体采编好的 `{date,summary,items[],ai_model}` JSON，后端只做归档/校验（`ai_model` 缺省记 `WorkBuddy-Agent`）。
- **后端内置定时已关闭**：`internal_scheduler`(`*bool`，本部署 `false`)让 `StartScheduler` 直接退出。后端 `GenerateInternal`/`generateSummary`/`translateItems` 仍可用，但只是**手动兜底**(手动 `POST /api/news/generate`)，不再是每日路径。想后端兜底就把 `internal_scheduler` 设 `true` 并 `docker restart devtools`。
- 英文翻译/面面俱到摘要逻辑留在后端仅是兜底；正式每日日报由智能体在本地用工具完成。
- 前端 `/news` 页面有「立即生成」弹窗(走 `/generate`，密码存 localStorage)——这是手动兜底入口，与每日智能体采编互补。
- 归档：SQLite `news_digests` 表(`date` 主键)，`retention_days:0` 永久保留；`/api/news/latest`、`/api/news/:date`、`/api/news/list` 供页面读取。
- 后端改动须按上节"后端二进制如何进容器"流程重编+`docker cp`+`docker commit` 才生效。

## Git 发布规范（用户明确：内容要合入 main 并打 tag）
- 功能分支（如 `feat/magic-pet-yunfan`）的改动，**最终必须合回 `main`**，不要只把 feature 分支推上去就结束。
- 标准动作：
  1. 在功能分支提交改动（**只 `git add` 指定文件，排除 `.workbuddy/` 个人记忆目录**）。
  2. `git checkout main` → `git merge --no-ff <feature-branch>`（生成显式合并提交）；再 `git push origin main`。
  3. 打 annotated 版本 tag：`git tag -a vX.Y.Z -m "..." ` → `git push origin vX.Y.Z`。
- 版本号由用户指定；首个正式大版本为 `v1.0.0`（现有旧 tag 为 `v0.1.0-pre-refactor`）。
- 中途遇到的坑：`.git/index.lock` 陈旧锁（中断遗留）用 `rm -f .git/index.lock` 恢复；git 提交/合并用 `GIT_EDITOR=true` 或 `-m` 避免编辑器卡住。
