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

RUN apk add --no-cache ca-certificates tzdata curl python3 coreutils

# Install uv (provides uvx) for MCP runtime
RUN curl -Ls https://astral.sh/uv/install.sh | sh
ENV PATH="/root/.local/bin:${PATH}"
ENV UV_PYTHON=python3
ENV UV_CACHE_DIR=/root/.cache/uv

# Warm up uvx tool cache to avoid runtime downloads
RUN uvx minimax-coding-plan-mcp -y --help >/dev/null 2>&1 || true

COPY --from=backend-builder /app/backend/server ./server
COPY --from=frontend-builder /app/frontend/dist ./dist

RUN mkdir -p /app/data

ENV PORT=8082
ENV DB_PATH=/app/data/paste.db
ENV GIN_MODE=release
ENV TZ=Asia/Shanghai

EXPOSE 8082

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8082/api/health || exit 1

CMD ["./server"]
