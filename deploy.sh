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

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
FRONTEND_DIR="$SCRIPT_DIR/frontend"
BACKEND_DIR="$SCRIPT_DIR/backend"
LOG_FILE="/tmp/devtools.log"
PID_FILE="/tmp/devtools.pid"
PORT="${PORT:-8080}"
HOST_PORT="${HOST_PORT:-8082}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

compose() {
    if command -v docker-compose &> /dev/null; then
        docker-compose "$@"
    else
        docker compose "$@"
    fi
}

wait_for_health() {
    local url="$1"
    local retries="${2:-20}"
    local delay="${3:-3}"
    local i

    for ((i=1; i<=retries; i++)); do
        if curl -fsS "$url" > /dev/null 2>&1; then
            return 0
        fi
        sleep "$delay"
    done

    return 1
}

build_frontend() {
    log_info "构建前端..."
    cd "$FRONTEND_DIR"

    if [ ! -d "node_modules" ]; then
        log_info "安装前端依赖..."
        npm install
    fi

    npm run build
    log_success "前端构建完成"

    log_info "复制静态文件到后端..."
    rm -rf "$BACKEND_DIR/dist"
    cp -r "$FRONTEND_DIR/dist" "$BACKEND_DIR/"
    log_success "静态文件复制完成"
}

build_backend() {
    log_info "构建后端..."
    cd "$BACKEND_DIR"
    go build -o server main.go
    log_success "后端构建完成"
}

build() {
    build_frontend
    build_backend
    log_success "全部构建完成！"
}

stop() {
    log_info "停止服务..."

    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if kill -0 "$PID" 2>/dev/null; then
            kill "$PID"
            rm -f "$PID_FILE"
            log_success "服务已停止 (PID: $PID)"
            return
        fi
    fi

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

start() {
    log_info "启动服务 (端口: $PORT)..."

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

    PORT=$PORT nohup ./server > "$LOG_FILE" 2>&1 &
    echo $! > "$PID_FILE"

    sleep 2

    if curl -s "http://localhost:$PORT/api/health" > /dev/null 2>&1; then
        log_success "服务启动成功！"
        log_info "访问地址: http://localhost:$PORT"
        log_info "日志文件: $LOG_FILE"
        log_info "PID: $(cat "$PID_FILE")"
    else
        log_error "服务启动失败，请检查日志: $LOG_FILE"
        cat "$LOG_FILE"
        exit 1
    fi
}

restart() {
    stop
    sleep 1
    start
}

status() {
    log_info "服务状态:"

    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if kill -0 "$PID" 2>/dev/null; then
            log_success "服务运行中 (PID: $PID, 端口: $PORT)"

            if curl -s "http://localhost:$PORT/api/health" > /dev/null 2>&1; then
                log_success "健康检查通过"
            else
                log_warn "健康检查失败"
            fi
            return
        fi
    fi

    if command -v lsof &> /dev/null; then
        PID=$(lsof -ti:$PORT 2>/dev/null)
        if [ -n "$PID" ]; then
            log_success "端口 $PORT 有服务运行 (PID: $PID)"
            return
        fi
    fi

    log_warn "服务未运行"
}

logs() {
    if [ -f "$LOG_FILE" ]; then
        tail -f "$LOG_FILE"
    else
        log_warn "日志文件不存在: $LOG_FILE"
    fi
}

deploy() {
    log_info "开始完整部署..."
    stop 2>/dev/null || true
    build
    start
    log_success "部署完成！"
}

docker_deploy() {
    log_info "Docker 部署..."

    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装"
        exit 1
    fi

    cd "$SCRIPT_DIR"

    log_info "构建并重建 Docker 容器..."
    if [ "${NO_CACHE:-0}" = "1" ]; then
        compose build --no-cache
        compose up -d --force-recreate --remove-orphans
    else
        compose up -d --build --force-recreate --remove-orphans
    fi

    if wait_for_health "http://localhost:${HOST_PORT}/api/health" 30 2; then
        log_success "Docker 部署完成！"
        log_info "访问地址: http://localhost:${HOST_PORT}"
        compose ps
    else
        log_error "服务启动失败，请检查日志"
        compose ps || true
        compose logs --tail=100 devtools || true
        exit 1
    fi
}

docker_stop() {
    log_info "停止 Docker 容器..."
    cd "$SCRIPT_DIR"
    compose down
    log_success "Docker 容器已停止"
}

docker_restart() {
    log_info "重启 Docker 容器..."
    cd "$SCRIPT_DIR"
    compose restart
    log_success "Docker 容器已重启"
}

docker_logs() {
    cd "$SCRIPT_DIR"
    compose logs -f
}

docker_status() {
    cd "$SCRIPT_DIR"
    compose ps
    echo ""
    if curl -s "http://localhost:${HOST_PORT}/api/health" > /dev/null 2>&1; then
        log_success "健康检查通过"
    else
        log_warn "健康检查失败"
    fi
}

help() {
    echo "DevTools 快速部署脚本"
    echo ""
    echo "用法: ./deploy.sh [命令]"
    echo ""
    echo "本地部署命令:"
    echo "  build       构建前端和后端"
    echo "  start       启动服务"
    echo "  stop        停止服务"
    echo "  restart     重启服务"
    echo "  status      查看服务状态"
    echo "  logs        查看日志 (tail -f)"
    echo "  deploy      完整部署 (构建 + 启动)"
    echo ""
    echo "Docker 命令:"
    echo "  docker            Docker 构建并部署"
    echo "  docker-stop       停止 Docker 容器"
    echo "  docker-restart    重启 Docker 容器"
    echo "  docker-logs       查看 Docker 日志"
    echo "  docker-status     查看 Docker 状态"
    echo ""
    echo "  help        显示帮助"
    echo ""
    echo "环境变量:"
    echo "  PORT        本地服务端口 (默认: 8080)"
    echo "  HOST_PORT   Docker 映射端口 (默认: 8082)"
    echo "  NO_CACHE    Docker 构建时禁用缓存，设为 1 生效"
    echo ""
    echo "示例:"
    echo "  ./deploy.sh deploy              # 本地完整部署"
    echo "  ./deploy.sh docker              # Docker 部署"
    echo "  ./deploy.sh docker-logs         # 查看 Docker 日志"
    echo "  PORT=8081 ./deploy.sh start     # 指定端口启动本地服务"
    echo "  NO_CACHE=1 ./deploy.sh docker   # 无缓存重建 Docker 镜像"
}

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
    docker-stop)
        docker_stop
        ;;
    docker-restart)
        docker_restart
        ;;
    docker-logs)
        docker_logs
        ;;
    docker-status)
        docker_status
        ;;
    help|--help|-h)
        help
        ;;
    *)
        log_error "未知命令: $1"
        echo ""
        help
        exit 1
        ;;
esac
