#!/bin/bash

# DevTools 快速部署脚本
# 用法: ./deploy.sh [命令]
# 命令:
#   build    - 构建前端和后端
#   start    - 启动服务
#   stop     - 停止服务
#   restart  - 重启服务
#   status   - 查看服务状态
#   logs     - 查看日志
#   deploy   - 完整部署 (构建 + 启动)

set -e

# 配置
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
FRONTEND_DIR="$SCRIPT_DIR/frontend"
BACKEND_DIR="$SCRIPT_DIR/backend"
LOG_FILE="/tmp/devtools.log"
PID_FILE="/tmp/devtools.pid"
PORT="${PORT:-8080}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 构建前端
build_frontend() {
    log_info "构建前端..."
    cd "$FRONTEND_DIR"

    if [ ! -d "node_modules" ]; then
        log_info "安装前端依赖..."
        npm install
    fi

    npm run build
    log_success "前端构建完成"

    # 复制到后端目录
    log_info "复制静态文件到后端..."
    rm -rf "$BACKEND_DIR/dist"
    cp -r "$FRONTEND_DIR/dist" "$BACKEND_DIR/"
    log_success "静态文件复制完成"
}

# 构建后端
build_backend() {
    log_info "构建后端..."
    cd "$BACKEND_DIR"

    go mod tidy
    go build -o server main.go

    log_success "后端构建完成"
}

# 完整构建
build() {
    build_frontend
    build_backend
    log_success "全部构建完成！"
}

# 停止服务
stop() {
    log_info "停止服务..."

    # 通过 PID 文件停止
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if kill -0 "$PID" 2>/dev/null; then
            kill "$PID"
            rm -f "$PID_FILE"
            log_success "服务已停止 (PID: $PID)"
            return
        fi
    fi

    # 通过端口停止
    if command -v fuser &> /dev/null; then
        fuser -k "$PORT/tcp" 2>/dev/null && log_success "已终止端口 $PORT 上的进程" || log_warn "端口 $PORT 没有运行中的进程"
    elif command -v lsof &> /dev/null; then
        PID=$(lsof -ti:$PORT 2>/dev/null)
        if [ -n "$PID" ]; then
            kill $PID
            log_success "已终止端口 $PORT 上的进程 (PID: $PID)"
        else
            log_warn "端口 $PORT 没有运行中的进程"
        fi
    fi
}

# 启动服务
start() {
    log_info "启动服务 (端口: $PORT)..."

    # 检查端口是否被占用
    if command -v lsof &> /dev/null; then
        if lsof -ti:$PORT &>/dev/null; then
            log_error "端口 $PORT 已被占用，请先停止服务或更换端口"
            exit 1
        fi
    fi

    cd "$BACKEND_DIR"

    if [ ! -f "server" ]; then
        log_error "后端未构建，请先运行 ./deploy.sh build"
        exit 1
    fi

    if [ ! -d "dist" ]; then
        log_error "前端未构建，请先运行 ./deploy.sh build"
        exit 1
    fi

    # 启动服务
    PORT=$PORT nohup ./server > "$LOG_FILE" 2>&1 &
    echo $! > "$PID_FILE"

    sleep 2

    # 健康检查
    if curl -s "http://localhost:$PORT/api/health" > /dev/null 2>&1; then
        log_success "服务启动成功！"
        log_info "访问地址: http://localhost:$PORT"
        log_info "日志文件: $LOG_FILE"
        log_info "PID: $(cat $PID_FILE)"
    else
        log_error "服务启动失败，请检查日志: $LOG_FILE"
        cat "$LOG_FILE"
        exit 1
    fi
}

# 重启服务
restart() {
    stop
    sleep 1
    start
}

# 查看状态
status() {
    log_info "服务状态:"

    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if kill -0 "$PID" 2>/dev/null; then
            log_success "服务运行中 (PID: $PID, 端口: $PORT)"

            # 健康检查
            if curl -s "http://localhost:$PORT/api/health" > /dev/null 2>&1; then
                log_success "健康检查通过"
            else
                log_warn "健康检查失败"
            fi
            return
        fi
    fi

    # 检查端口
    if command -v lsof &> /dev/null; then
        PID=$(lsof -ti:$PORT 2>/dev/null)
        if [ -n "$PID" ]; then
            log_success "端口 $PORT 有服务运行 (PID: $PID)"
            return
        fi
    fi

    log_warn "服务未运行"
}

# 查看日志
logs() {
    if [ -f "$LOG_FILE" ]; then
        tail -f "$LOG_FILE"
    else
        log_warn "日志文件不存在: $LOG_FILE"
    fi
}

# 完整部署
deploy() {
    log_info "开始完整部署..."
    stop 2>/dev/null || true
    build
    start
    log_success "部署完成！"
}

# Docker 部署
docker_deploy() {
    log_info "Docker 部署..."

    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装"
        exit 1
    fi

    cd "$SCRIPT_DIR"
    docker build -t devtools:latest .

    # 停止旧容器
    docker stop devtools 2>/dev/null || true
    docker rm devtools 2>/dev/null || true

    # 启动新容器
    docker run -d \
        --name devtools \
        -p $PORT:8080 \
        -v devtools-data:/app/data \
        --restart unless-stopped \
        devtools:latest

    log_success "Docker 部署完成！"
    log_info "访问地址: http://localhost:$PORT"
}

# 显示帮助
help() {
    echo "DevTools 快速部署脚本"
    echo ""
    echo "用法: ./deploy.sh [命令]"
    echo ""
    echo "命令:"
    echo "  build       构建前端和后端"
    echo "  start       启动服务"
    echo "  stop        停止服务"
    echo "  restart     重启服务"
    echo "  status      查看服务状态"
    echo "  logs        查看日志 (tail -f)"
    echo "  deploy      完整部署 (构建 + 启动)"
    echo "  docker      Docker 部署"
    echo "  help        显示帮助"
    echo ""
    echo "环境变量:"
    echo "  PORT        服务端口 (默认: 8080)"
    echo ""
    echo "示例:"
    echo "  ./deploy.sh deploy              # 完整部署"
    echo "  PORT=8081 ./deploy.sh start     # 指定端口启动"
    echo "  ./deploy.sh logs                # 查看日志"
}

# 主入口
case "${1:-help}" in
    build)
        build
        ;;
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        restart
        ;;
    status)
        status
        ;;
    logs)
        logs
        ;;
    deploy)
        deploy
        ;;
    docker)
        docker_deploy
        ;;
    help|--help|-h)
        help
        ;;
    *)
        log_error "未知命令: $1"
        help
        exit 1
        ;;
esac
