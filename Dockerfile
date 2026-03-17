FROM docker.m.daocloud.io/library/node:20-alpine AS frontend-builder

WORKDIR /app/frontend

RUN corepack enable

COPY frontend/package.json frontend/pnpm-lock.yaml ./
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

FROM docker.m.daocloud.io/library/alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata curl python3 py3-pip coreutils ffmpeg \
    nodejs npm git bash

# Install uv (provides uvx) for MCP runtime
RUN curl -Ls https://astral.sh/uv/install.sh | sh
ENV PATH="/root/.local/bin:${PATH}"
ENV UV_PYTHON=python3
ENV UV_CACHE_DIR=/root/.cache/uv

# Warm up uvx tool cache to avoid runtime downloads
RUN uvx minimax-coding-plan-mcp -y --help >/dev/null 2>&1 || true

# Install Claude Code CLI
RUN npm install -g @anthropic-ai/claude-code --unsafe-perm 2>&1 || true

# Install autodev from clawtest
RUN git clone --depth=1 https://github.com/axfinn/clawtest.git /opt/clawtest \
    && chmod +x /opt/clawtest/autodev/autodev \
    && chmod +x /opt/clawtest/autodev/autodev-stop

# Configure Claude Code for autodev
# Settings without sensitive credentials (tokens are injected at runtime via env vars)
RUN mkdir -p /root/.claude
RUN cat > /root/.claude/settings.json << 'EOF'
{
  "skills": {
    "paths": ["~/.claude/skills"]
  },
  "env": {
    "API_TIMEOUT_MS": "3000000",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
    "ANTHROPIC_MODEL": "MiniMax-M2.5",
    "ANTHROPIC_SMALL_FAST_MODEL": "MiniMax-M2.5",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "MiniMax-M2.5",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "MiniMax-M2.5",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "MiniMax-M2.5"
  }
}
EOF

# Create non-root user for running autodev tasks
# Claude Code refuses --dangerously-skip-permissions when running as root
RUN addgroup -g 1001 autodev && \
    adduser -D -u 1001 -G autodev autodev && \
    mkdir -p /home/autodev/.claude && \
    cp /root/.claude/settings.json /home/autodev/.claude/settings.json && \
    chown -R autodev:autodev /home/autodev && \
    chmod -R 755 /home/autodev/.claude

COPY --from=backend-builder /app/backend/server ./server
COPY --from=frontend-builder /app/frontend/dist ./dist

RUN mkdir -p /app/data/autodev && chmod 777 /app/data/autodev

ENV PORT=8082
ENV DB_PATH=/app/data/paste.db
ENV GIN_MODE=release
ENV TZ=Asia/Shanghai
# AutoDev 配置（通过 .env 或 docker-compose environment 覆盖）
ENV AUTODEV_PATH=/opt/clawtest/autodev/autodev
ENV AUTODEV_DATA_DIR=/app/data/autodev
# Claude API 配置（ANTHROPIC_AUTH_TOKEN 必须在 .env 中设置，不要硬编码在此）
ENV ANTHROPIC_BASE_URL=https://api.minimaxi.com/anthropic
ENV ANTHROPIC_MODEL=MiniMax-M2.5
ENV API_TIMEOUT_MS=3000000
ENV CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1

EXPOSE 8082

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8082/api/health || exit 1

CMD ["./server"]
