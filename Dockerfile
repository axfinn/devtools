FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

RUN npm install -g pnpm@10.28.2

COPY frontend/package.json frontend/pnpm-lock.yaml frontend/.npmrc frontend/pnpm-workspace.yaml ./
COPY frontend/scripts ./scripts
RUN pnpm install --frozen-lockfile

COPY frontend/ ./
ENV NODE_OPTIONS="--max-old-space-size=4096"
RUN pnpm build

# AskIt Chrome 扩展不再在镜像里构建。
#
# 原因：Askit 是私有仓库，容器内既没有凭据也没有 ssh-agent，`git clone` 会以
# "could not read Username for 'https://github.com'" (exit 128) 直接打断发布。
# 改为在宿主机上构建（宿主 git 用 ~/.ssh/id_ed25519，已注册到 GitHub），产物由
# deploy.sh build_askit 写到构建上下文根目录，这里只负责 COPY。
#
# 注意：askit-extension.zip 不进版本库（见 .gitignore），所以 **直接 `docker compose
# build` 会因为缺少这个文件而失败**。请走 `./deploy.sh docker`（它会先跑 build_askit），
# 或先单独执行 `./deploy.sh build-askit`。
# 确实要临时跳过时：SKIP_ASKIT=1 ./deploy.sh docker —— 会写一个占位 zip
# 保证 COPY 成立，扩展改由运行时 SSH 拉取（handlers/askit_extension.go:128）。

FROM golang:1.26.10-alpine AS backend-builder

WORKDIR /app/backend

RUN apk add --no-cache gcc musl-dev sqlite-dev

# 配置 Go 模块代理
ENV GOPROXY=https://goproxy.cn,direct

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
# -tags with_utls 启用 sing-box Reality 所需的 uTLS 指纹(否则启动失败: "uTLS ... is not included in this build")
RUN CGO_ENABLED=1 GOOS=linux go build -tags with_utls -a -ldflags '-linkmode external -extldflags "-static"' -o server .

# 编译 proxy-client 跨平台二进制
FROM golang:1.26.10-alpine AS proxy-client-builder

WORKDIR /app/proxy-client

ENV GOPROXY=https://goproxy.cn,direct

COPY proxy-client/go.mod proxy-client/go.sum ./
RUN go mod download

COPY proxy-client/ ./
RUN GOOS=darwin  GOARCH=arm64 go build -ldflags="-s -w" -o dist/proxy-client-darwin-arm64   . && \
    GOOS=darwin  GOARCH=amd64 go build -ldflags="-s -w" -o dist/proxy-client-darwin-amd64   . && \
    GOOS=linux   GOARCH=amd64 go build -ldflags="-s -w" -o dist/proxy-client-linux-amd64    . && \
    GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/proxy-client-windows-amd64.exe .

FROM alpine:latest

WORKDIR /app

# apk 源切到国内（Aliyun 镜像）
RUN sed -i 's|dl-cdn.alpinelinux.org|mirrors.aliyun.com|g' /etc/apk/repositories

RUN apk add --no-cache tzdata curl python3 py3-pip coreutils ffmpeg \
    nodejs npm git bash openssh-client hugo

# ca-certificates 单独一层，并强制刷新到仓库里的最新版本。
#
# 为什么必须单独成层：Docker 按「指令文本 + 前置层」做缓存，同一条
# `apk add --no-cache ca-certificates` 在 Dockerfile 不变时会一直复用旧层，
# 装到的还是几个月前的根库快照。实测本镜像里装的是 ca-certificates-20260611-r0
# —— 早于 2026-10 的 eMudhra 弱私钥事件，容器内的 curl / wget / python / node
# 仍会信任那 8 张被攻陷的中间 CA（Go 侧靠 1.26.10 的 x509 内置 distrust 兜住，
# 非 Go 客户端没有这层保护）。
#
# `apk update` + `--upgrade` 让这一层每次改动指令时都拿到最新根库；
# 想强制刷新也可以直接 docker build --no-cache。
RUN apk update && apk add --no-cache --upgrade ca-certificates

# 内置 npc 客户端（从 NPS 服务端下载，避免运行时下载）
# 如果构建时无法访问，npc 将在首次启动时由 Go 服务自动下载
ARG NPS_SERVER_URL=https://npss.jaxiu.cn
RUN wget -qO /tmp/npc.tar.gz "${NPS_SERVER_URL}/static/download/client/npc_linux_amd64.tar.gz" && \
    tar -xzf /tmp/npc.tar.gz -C /tmp && \
    mv /tmp/npc /usr/local/bin/npc && \
    chmod +x /usr/local/bin/npc && \
    rm /tmp/npc.tar.gz || echo "npc download skipped (will download at runtime)"

# Create isolated venv for TTS service (edge-tts + FastAPI)
# 版本锁定在 tts-service/requirements.txt：edge-tts/fastapi/uvicorn 都会发破坏性
# 升级，不锁版本镜像不可复现，某次升级就能让 TTS 静默起不来。
COPY tts-service/requirements.txt /tmp/tts-requirements.txt
RUN python3 -m venv /app/tts-venv && \
    /app/tts-venv/bin/pip install --upgrade pip && \
    /app/tts-venv/bin/pip install -r /tmp/tts-requirements.txt

# Install uv (provides uvx) for MCP runtime
# 不用 `curl -Ls https://astral.sh/uv/install.sh | sh`：管道执行远端脚本正是
# eMudhra 事件里点名的暴露面 —— 构建期若被中间人用伪造证书劫持(那批弱私钥 CA
# 签出的证书对 curl/Node/Python 默认配置有效),可以直接往镜像里注入代码。
# 改用独立 venv + 钉版本的 PyPI 包(uv 0.13.4+ 内含对被吊销中间 CA 的 distrust 修复)。
RUN python3 -m venv /app/uv-venv && \
    /app/uv-venv/bin/pip install --no-cache-dir uv==0.13.5
ENV PATH="/app/uv-venv/bin:/root/.local/bin:${PATH}"
ENV UV_PYTHON=python3
ENV UV_CACHE_DIR=/root/.cache/uv

# Warm up uvx tool cache to avoid runtime downloads
RUN uvx minimax-coding-plan-mcp -y --help >/dev/null 2>&1 || true

# Install Claude Code CLI and Codex CLI
RUN npm install -g @anthropic-ai/claude-code --unsafe-perm 2>&1 || true
RUN npm install -g @openai/codex --unsafe-perm 2>&1 || true

# Create non-root user for running autodev tasks
# Claude Code refuses --dangerously-skip-permissions when running as root
RUN addgroup -g 1001 autodev && \
    adduser -D -u 1001 -G autodev autodev

# Claude settings are generated dynamically in entrypoint.sh with runtime env vars (e.g. MINIMAX_API_KEY)

COPY --from=backend-builder /app/backend/server ./server
COPY --from=proxy-client-builder /app/proxy-client/dist ./proxy-client-bins/
COPY --from=frontend-builder /app/frontend/dist ./dist
COPY askit-extension.zip ./askit-extension.zip
COPY entrypoint.sh ./entrypoint.sh
COPY tts-service/server.py ./tts_server.py
RUN chmod +x ./entrypoint.sh

RUN mkdir -p /app/data/autodev /app/data/clawtest && chmod 777 /app/data/autodev /app/data/clawtest

ENV PORT=8082
ENV DB_PATH=/app/data/paste.db
ENV GIN_MODE=release
ENV TZ=Asia/Shanghai
ENV TTS_SERVICE_URL=http://127.0.0.1:8083
# AutoDev 配置（通过 .env 或 docker-compose environment 覆盖）
# clawtest 存储在 data volume 中，entrypoint 启动时自动 clone/pull 到最新版
ENV AUTODEV_PATH=/app/data/clawtest/autodev/autodev
ENV AUTODEV_DATA_DIR=/app/data/autodev
# Claude API 配置（ANTHROPIC_AUTH_TOKEN 必须在 .env 中设置，不要硬编码在此）
ENV ANTHROPIC_BASE_URL=https://api.minimaxi.com/anthropic
ENV ANTHROPIC_MODEL=MiniMax-M2.7
ENV API_TIMEOUT_MS=3000000
ENV CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1

EXPOSE 8082

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8082/api/health || exit 1

CMD ["./entrypoint.sh"]
