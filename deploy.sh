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
PORT="${PORT:-8080}"
HOST_PORT="${HOST_PORT:-8082}"
LOG_FILE="/tmp/devtools-${PORT}.log"
PID_FILE="/tmp/devtools-${PORT}.pid"

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

command_exists() {
    command -v "$1" > /dev/null 2>&1
}

compose() {
    if command_exists docker-compose; then
        docker-compose "$@"
    else
        docker compose "$@"
    fi
}

pid_file_value() {
    local key="$1"
    if [ ! -f "$PID_FILE" ]; then
        return 1
    fi
    awk -F= -v target="$key" '$1 == target { print substr($0, index($0, "=") + 1) }' "$PID_FILE" | tail -n 1
}

remove_pid_file() {
    rm -f "$PID_FILE"
}

process_listens_on_port() {
    local pid="$1"
    local port="$2"

    if command_exists lsof; then
        lsof -Pan -p "$pid" -iTCP:"$port" -sTCP:LISTEN > /dev/null 2>&1
        return $?
    fi

    return 1
}

server_pid_matches() {
    local pid="$1"
    local expected_cmd expected_port args

    if [ -z "${pid:-}" ] || ! kill -0 "$pid" 2>/dev/null; then
        return 1
    fi

    expected_cmd="$(pid_file_value cmd 2>/dev/null || true)"
    expected_port="$(pid_file_value port 2>/dev/null || true)"

    if command_exists ps; then
        args="$(ps -p "$pid" -o args= 2>/dev/null || true)"
        if [ -n "$expected_cmd" ] && [[ "$args" == *"$expected_cmd"* || "$args" == *"./server"* ]]; then
            return 0
        fi
    fi

    if [ -n "$expected_port" ] && process_listens_on_port "$pid" "$expected_port"; then
        return 0
    fi

    return 1
}

wait_for_pid_exit() {
    local pid="$1"
    local retries="${2:-20}"
    local delay="${3:-1}"
    local i

    for ((i=1; i<=retries; i++)); do
        if ! kill -0 "$pid" 2>/dev/null; then
            return 0
        fi
        sleep "$delay"
    done

    return 1
}

kill_pid_gracefully() {
    local pid="$1"

    kill "$pid" 2>/dev/null || true
    if wait_for_pid_exit "$pid" 15 1; then
        return 0
    fi

    log_warn "进程 $pid 在超时后仍未退出，发送 SIGKILL"
    kill -9 "$pid" 2>/dev/null || true
    wait_for_pid_exit "$pid" 5 1
}

frontend_deps_hash() {
    if [ -f "$FRONTEND_DIR/package-lock.json" ]; then
        if command_exists shasum; then
            cat "$FRONTEND_DIR/package.json" "$FRONTEND_DIR/package-lock.json" | shasum -a 256 | awk '{print $1}'
        else
            cat "$FRONTEND_DIR/package.json" "$FRONTEND_DIR/package-lock.json" | sha256sum | awk '{print $1}'
        fi
    else
        if command_exists shasum; then
            shasum -a 256 "$FRONTEND_DIR/package.json" | awk '{print $1}'
        else
            sha256sum "$FRONTEND_DIR/package.json" | awk '{print $1}'
        fi
    fi
}

ensure_frontend_deps() {
    local hash_file="$FRONTEND_DIR/node_modules/.devtools-deps.hash"
    local current_hash saved_hash

    current_hash="$(frontend_deps_hash)"
    saved_hash=""
    if [ -f "$hash_file" ]; then
        saved_hash="$(cat "$hash_file")"
    fi

    if [ ! -d "$FRONTEND_DIR/node_modules" ] || [ "$current_hash" != "$saved_hash" ]; then
        log_info "安装前端依赖..."
        cd "$FRONTEND_DIR"
        if [ -f "package-lock.json" ]; then
            npm ci
        else
            npm install
        fi
        mkdir -p "$FRONTEND_DIR/node_modules"
        echo "$current_hash" > "$hash_file"
    else
        log_info "前端依赖已是最新，跳过安装"
    fi
}

docker_devtools_container_id() {
    compose ps -q devtools 2>/dev/null | head -n 1
}

compose_other_services() {
    compose config --services 2>/dev/null | awk 'NF > 0 && $0 != "devtools"'
}

docker_container_image_id() {
    local container_id="$1"
    if [ -z "$container_id" ]; then
        return 1
    fi
    docker inspect --format '{{.Image}}' "$container_id" 2>/dev/null || true
}

docker_container_image_ref() {
    local container_id="$1"
    if [ -z "$container_id" ]; then
        return 1
    fi
    docker inspect --format '{{.Config.Image}}' "$container_id" 2>/dev/null || true
}

docker_devtools_health() {
    local container_id
    container_id="$(docker_devtools_container_id)"
    if [ -z "$container_id" ]; then
        echo "missing"
        return 0
    fi

    docker inspect --format '{{if .State.Running}}{{if .State.Health}}{{.State.Health.Status}}{{else}}running{{end}}{{else}}stopped{{end}}' "$container_id" 2>/dev/null || echo "missing"
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

healthcheck_retry_budget() {
    local container_id="$1"
    local default_retries="${2:-40}"
    local delay="${3:-3}"
    local inspect_output start_period interval retries timeout_seconds

    if [ -z "$container_id" ]; then
        echo "$default_retries"
        return 0
    fi

    inspect_output="$(docker inspect --format '{{if .Config.Healthcheck}}{{.Config.Healthcheck.StartPeriod}} {{.Config.Healthcheck.Interval}} {{.Config.Healthcheck.Retries}}{{else}}0 0 0{{end}}' "$container_id" 2>/dev/null || echo "0 0 0")"
    read -r start_period interval retries <<< "$inspect_output"

    if [ "${retries:-0}" -le 0 ] || [ "${interval:-0}" -le 0 ]; then
        echo "$default_retries"
        return 0
    fi

    timeout_seconds=$(( (start_period + interval * retries + 30000000000 + 999999999) / 1000000000 ))
    if [ "$timeout_seconds" -le 0 ]; then
        echo "$default_retries"
        return 0
    fi

    timeout_seconds=$(( timeout_seconds / delay + 1 ))
    if [ "$timeout_seconds" -lt "$default_retries" ]; then
        echo "$default_retries"
        return 0
    fi

    echo "$timeout_seconds"
}

wait_for_container_health() {
    local service="$1"
    local retries="${2:-40}"
    local delay="${3:-3}"
    local i container_id status resolved_retries="$retries"

    for ((i=1; i<=resolved_retries; i++)); do
        container_id="$(compose ps -q "$service" 2>/dev/null | head -n 1)"
        if [ -n "$container_id" ]; then
            if [ "$resolved_retries" = "$retries" ]; then
                resolved_retries="$(healthcheck_retry_budget "$container_id" "$retries" "$delay")"
            fi
            status="$(docker inspect --format '{{if .State.Running}}{{if .State.Health}}{{.State.Health.Status}}{{else}}running{{end}}{{else}}stopped{{end}}' "$container_id" 2>/dev/null || echo "missing")"
            case "$status" in
                healthy|running)
                    return 0
                    ;;
            esac
        fi
        sleep "$delay"
    done

    return 1
}

rollback_devtools() {
    local old_image_id="$1"
    local service_image="$2"

    if [ -z "$old_image_id" ] || [ -z "$service_image" ]; then
        log_warn "缺少旧镜像信息，无法自动回滚"
        return 1
    fi

    log_warn "尝试回滚到旧镜像..."
    docker tag "$old_image_id" "$service_image"
    compose up -d --no-deps --force-recreate devtools
}

build_frontend() {
    log_info "构建前端..."
    cd "$FRONTEND_DIR"

    ensure_frontend_deps
    cd "$FRONTEND_DIR"
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
        PID="$(pid_file_value pid 2>/dev/null || true)"
        if server_pid_matches "$PID"; then
            if kill_pid_gracefully "$PID"; then
                remove_pid_file
                log_success "服务已停止 (PID: $PID)"
                return
            fi
            log_error "无法停止服务进程: $PID"
            exit 1
        fi
        log_warn "检测到陈旧 PID 文件，已清理: $PID_FILE"
        remove_pid_file
    fi

    if command_exists fuser; then
        fuser -k "$PORT/tcp" 2>/dev/null && log_success "已终止端口 $PORT 上的进程" || log_warn "端口 $PORT 没有运行中的进程"
    elif command_exists lsof; then
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

    if [ -f "$PID_FILE" ]; then
        PID="$(pid_file_value pid 2>/dev/null || true)"
        if server_pid_matches "$PID"; then
            log_error "服务似乎已经运行 (PID: $PID)"
            exit 1
        fi
        remove_pid_file
    fi

    if command_exists lsof; then
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
    PID=$!
    cat > "$PID_FILE" <<EOF
pid=$PID
port=$PORT
cmd=$BACKEND_DIR/server
EOF

    if wait_for_health "http://localhost:$PORT/api/health" 20 1; then
        log_success "服务启动成功！"
        log_info "访问地址: http://localhost:$PORT"
        log_info "日志文件: $LOG_FILE"
        log_info "PID: $PID"
    else
        log_error "服务启动失败，请检查日志: $LOG_FILE"
        kill_pid_gracefully "$PID" || true
        remove_pid_file
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
        PID="$(pid_file_value pid 2>/dev/null || true)"
        if server_pid_matches "$PID"; then
            log_success "服务运行中 (PID: $PID, 端口: $PORT)"

            if curl -s "http://localhost:$PORT/api/health" > /dev/null 2>&1; then
                log_success "健康检查通过"
            else
                log_warn "健康检查失败"
            fi
            return
        fi
        log_warn "PID 文件已过期，已清理"
        remove_pid_file
    fi

    if command_exists lsof; then
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

    if ! command_exists docker; then
        log_error "Docker 未安装"
        exit 1
    fi

    cd "$SCRIPT_DIR"

    local current_container_id old_image_id service_image
    current_container_id="$(docker_devtools_container_id)"
    old_image_id="$(docker_container_image_id "$current_container_id")"
    service_image="$(docker_container_image_ref "$current_container_id")"

    log_info "先构建镜像，避免构建阶段影响线上服务..."
    if [ "${NO_CACHE:-0}" = "1" ]; then
        compose build --no-cache
    else
        compose build
    fi

    log_info "确保其他服务已就绪..."
    local service_wait_retries="${SERVICE_WAIT_RETRIES:-600}"
    local -a other_services=()
    local service
    while IFS= read -r service; do
        other_services+=("$service")
    done < <(compose_other_services)
    if [ "${#other_services[@]}" -gt 0 ]; then
        compose up -d "${other_services[@]}"
        for service in "${other_services[@]}"; do
            if ! wait_for_container_health "$service" "$service_wait_retries" 3; then
                log_error "$service 未就绪，停止替换 devtools"
                compose ps || true
                exit 1
            fi
        done
    fi

    log_info "快速替换 devtools 容器..."
    compose up -d --no-deps --force-recreate devtools

    if wait_for_health "http://localhost:${HOST_PORT}/api/health" 40 2; then
        log_success "Docker 部署完成！"
        log_info "访问地址: http://localhost:${HOST_PORT}"
        compose ps
        return
    fi

    # 健康检查失败时，只接受 devtools 容器本身 healthy
    if [ "$(docker_devtools_health)" = "healthy" ]; then
        log_success "Docker 部署完成（devtools 容器已 healthy）！"
        log_info "访问地址: http://localhost:${HOST_PORT}"
        compose ps
        return
    fi

    log_error "新版本启动失败，准备回滚"
    log_error "devtools 容器状态: $(docker_devtools_health)"
    compose ps || true
    compose logs --tail=100 devtools || true

    if rollback_devtools "$old_image_id" "$service_image"; then
        if wait_for_health "http://localhost:${HOST_PORT}/api/health" 30 2 || [ "$(docker_devtools_health)" = "healthy" ]; then
            log_warn "已回滚到旧版本"
            compose ps || true
            exit 1
        fi
    fi

    log_error "回滚失败，请手动处理"
    exit 1
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
    if wait_for_health "http://localhost:${HOST_PORT}/api/health" 30 2; then
        log_success "Docker 容器已重启"
    else
        log_warn "容器已重启，但 API 健康检查暂未通过"
        compose ps || true
    fi
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
        log_warn "devtools 容器状态: $(docker_devtools_health)"
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
    echo "  docker            Docker 先构建、再替换 devtools，失败时尝试回滚"
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
