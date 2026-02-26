# 阶段一：构建后端
FROM golang:1.26-alpine AS backend-builder

WORKDIR /app

# 安装 SQLite 依赖
RUN apk add --no-cache gcc musl-dev sqlite-dev

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o server .

# 阶段二：生产镜像
FROM alpine:latest

WORKDIR /app

# 安装时区数据
RUN apk add --no-cache ca-certificates tzdata

# 复制后端二进制文件
COPY --from=backend-builder /app/server .

# 复制前端构建产物（需要本地先构建好）
COPY frontend/dist ./dist

# 创建数据目录
RUN mkdir -p /app/data

# 设置环境变量
ENV PORT=8082
ENV DB_PATH=/app/data/paste.db
ENV GIN_MODE=release
ENV TZ=Asia/Shanghai

EXPOSE 8082

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8082/api/health || exit 1

CMD ["./server"]
