# 阶段一：构建前端
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# 阶段二：构建后端
FROM golang:1.21-alpine AS backend-builder

WORKDIR /app
# 安装 SQLite 依赖
RUN apk add --no-cache gcc musl-dev sqlite-dev

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o server .

# 阶段三：生产镜像
FROM alpine:latest

WORKDIR /app

# 安装时区数据
RUN apk add --no-cache ca-certificates tzdata

# 复制后端二进制文件
COPY --from=backend-builder /app/server .

# 复制前端构建产物
COPY --from=frontend-builder /app/frontend/dist ./dist

# 创建数据目录
RUN mkdir -p /app/data

# 设置环境变量
ENV PORT=8080
ENV DB_PATH=/app/data/paste.db
ENV GIN_MODE=release
ENV TZ=Asia/Shanghai

EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

CMD ["./server"]
