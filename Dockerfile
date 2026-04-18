FROM docker.m.daocloud.io/library/node:20-alpine AS frontend-builder

WORKDIR /app/frontend

RUN npm install -g pnpm

COPY frontend/package.json frontend/pnpm-lock.yaml ./
COPY frontend/scripts ./scripts
RUN pnpm install --frozen-lockfile

COPY frontend/ ./
RUN pnpm build

FROM docker.m.daocloud.io/library/golang:1.21-alpine AS backend-builder

WORKDIR /app/backend

RUN apk add --no-cache gcc musl-dev sqlite-dev

# 配置 Go 模块代理
ENV GOPROXY=https://goproxy.cn,direct

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o server .

# 编译 proxy-client 跨平台二进制
FROM docker.m.daocloud.io/library/golang:1.21-alpine AS proxy-client-builder

WORKDIR /app/proxy-client

ENV GOPROXY=https://goproxy.cn,direct

COPY proxy-client/go.mod proxy-client/go.sum ./
RUN go mod download

COPY proxy-client/ ./
RUN GOOS=darwin  GOARCH=arm64 go build -ldflags="-s -w" -o dist/proxy-client-darwin-arm64   . && \
    GOOS=darwin  GOARCH=amd64 go build -ldflags="-s -w" -o dist/proxy-client-darwin-amd64   . && \
    GOOS=linux   GOARCH=amd64 go build -ldflags="-s -w" -o dist/proxy-client-linux-amd64    . && \
    GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/proxy-client-windows-amd64.exe .

FROM docker.m.daocloud.io/library/alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata curl python3 py3-pip coreutils ffmpeg \
    nodejs npm git bash openssh-client hugo

# 内置 npc 客户端（从 NPS 服务端下载，避免运行时下载）
# 如果构建时无法访问，npc 将在首次启动时由 Go 服务自动下载
ARG NPS_SERVER_URL=https://npss.jaxiu.cn
RUN wget -qO /tmp/npc.tar.gz "${NPS_SERVER_URL}/static/download/client/npc_linux_amd64.tar.gz" && \
    tar -xzf /tmp/npc.tar.gz -C /tmp && \
    mv /tmp/npc /usr/local/bin/npc && \
    chmod +x /usr/local/bin/npc && \
    rm /tmp/npc.tar.gz || echo "npc download skipped (will download at runtime)"

# Create isolated venv for TTS service (edge-tts + FastAPI)
RUN python3 -m venv /app/tts-venv && \
    /app/tts-venv/bin/pip install --upgrade pip && \
    /app/tts-venv/bin/pip install edge-tts fastapi uvicorn

# Install uv (provides uvx) for MCP runtime
RUN curl -Ls https://astral.sh/uv/install.sh | sh
ENV PATH="/root/.local/bin:${PATH}"
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
