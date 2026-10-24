# 项目长期记忆（devtools）

## 在 WorkBuddy 沙箱里构建前端（必看）
- 现象：pnpm/npm install 在沙箱里挂死或报 `CODEBUDDY_BROKER_DENY ... symlink`。
- 根因：WorkBuddy 通过 `NODE_OPTIONS=--require=.../node-language-shim.cjs` 把 brokered-fs 垫片注入**每个** Node 进程，垫片拒绝软链接。
- 解决（在本机终端跑构建前执行）：
  ```sh
  export PATH=/opt/homebrew/bin:$PATH   # 系统 Node，无垫片
  unset NODE_OPTIONS                    # 关键：去掉垫片注入
  export HTTP_PROXY=http://127.0.0.1:7897 HTTPS_PROXY=http://127.0.0.1:7897 http_proxy=http://127.0.0.1:7897 https_proxy=http://127.0.0.1:7897
  cd frontend && pnpm install --frozen-lockfile --config.store-projects=false --config.node-linker=hoisted && pnpm build
  ```
- 代理只监听本机回环 127.0.0.1:7897，容器内/daemon 侧连不上；不要改 Docker daemon 代理，也不要影响其他正在运行的容器。

## 部署方式
- 受限网络用 `HOT_DEPLOY=1 ./deploy.sh docker`：本机构建前端 + 叠加到已缓存镜像，零拉取。
- `deploy.sh` 的 `build_frontend_host` 已改为：若 `frontend/dist` 已存在则直接复用，不再回退提取旧 dist。
- Dockerfile.hot + .dockerignore.hot 负责"从缓存镜像叠加新 dist"；构建时会临时替换 .dockerignore 再还原。
- 后端二进制沿用已缓存镜像（前端改动无需重编后端）。
