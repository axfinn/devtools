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
#
# 环境变量:
#   HOT_DEPLOY=1   仅对 docker 命令生效：跳过完整 compose build，改为在本机用终端
#                   代理(HTTP(S)_PROXY)编译前端，再叠加到已缓存的 devtools-devtools
#                   镜像（零拉取，不动 daemon 配置，不影响其他容器）。
#                   当 Docker daemon 拉不到 golang/node 基础镜像时也可被自动回退触发。

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
FRONTEND_DIR="$SCRIPT_DIR/frontend"
BACKEND_DIR="$SCRIPT_DIR/backend"
PORT="${PORT:-8080}"
HOST_PORT="${HOST_PORT:-8082}"
LOG_FILE="/tmp/devtools-${PORT}.log"
PID_FILE="/tmp/devtools-${PORT}.pid"

# AskIt 扩展 zip 的构建上下文路径。
# 必须放在仓库根目录：backend/data 被 .dockerignore 整体排除，而 dockerignore
# 不允许对已排除目录的父级做 re-include —— 把 backend/data 放回来的同时会把
# paste.db 一起塞进镜像。
ASKIT_ZIP="$SCRIPT_DIR/askit-extension.zip"

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
    # 锁文件统一用 pnpm-lock.yaml(与 Dockerfile / CI 一致)
    if [ -f "$FRONTEND_DIR/pnpm-lock.yaml" ]; then
        if command_exists shasum; then
            cat "$FRONTEND_DIR/package.json" "$FRONTEND_DIR/pnpm-lock.yaml" | shasum -a 256 | awk '{print $1}'
        else
            cat "$FRONTEND_DIR/package.json" "$FRONTEND_DIR/pnpm-lock.yaml" | sha256sum | awk '{print $1}'
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
        if [ -f "pnpm-lock.yaml" ] && command_exists pnpm; then
            pnpm install --frozen-lockfile
        else
            # 回落分支：老机器没装 pnpm 时不能直接断（PM 已批准的偏离）。
            # 但仓库只认 pnpm-lock.yaml，package-lock.json 已删 ⇒ 回落时无锁可依，
            # npm 会重新解析 semver 最新版依赖树（产物不可复现），且默认会在
            # frontend/ 下自动生成一个 package-lock.json，与「只认 pnpm-lock.yaml」
            # 的口径冲突 ⇒ 用 --no-package-lock 禁掉这个自动生成。
            log_warn "未找到 pnpm，回落到 npm install（无锁解析，产物不可复现）"
            npm install --no-package-lock
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

duration_to_seconds_ceiling() {
    local input="$1"
    local remaining="$1"
    local total_ns=0
    local value unit factor

    if [ -z "$input" ] || [ "$input" = "0" ]; then
        echo 0
        return 0
    fi

    while [ -n "$remaining" ]; do
        if [[ "$remaining" =~ ^([0-9]+)(ns|us|µs|ms|s|m|h)(.*)$ ]]; then
            value="${BASH_REMATCH[1]}"
            unit="${BASH_REMATCH[2]}"
            remaining="${BASH_REMATCH[3]}"
            case "$unit" in
                ns) factor=1 ;;
                us|µs) factor=1000 ;;
                ms) factor=1000000 ;;
                s) factor=1000000000 ;;
                m) factor=60000000000 ;;
                h) factor=3600000000000 ;;
                *)
                    echo 0
                    return 1
                    ;;
            esac
            total_ns=$(( total_ns + value * factor ))
        else
            echo 0
            return 1
        fi
    done

    echo $(( (total_ns + 999999999) / 1000000000 ))
}

healthcheck_retry_budget() {
    local container_id="$1"
    local default_retries="${2:-40}"
    local delay="${3:-3}"
    local inspect_output start_period interval retries timeout_seconds
    local start_period_seconds interval_seconds grace_seconds

    if [ -z "$container_id" ]; then
        echo "$default_retries"
        return 0
    fi

    inspect_output="$(docker inspect --format '{{if .Config.Healthcheck}}{{.Config.Healthcheck.StartPeriod}} {{.Config.Healthcheck.Interval}} {{.Config.Healthcheck.Retries}}{{else}}0 0 0{{end}}' "$container_id" 2>/dev/null || echo "0 0 0")"
    read -r start_period interval retries <<< "$inspect_output"

    start_period_seconds="$(duration_to_seconds_ceiling "$start_period" || echo 0)"
    interval_seconds="$(duration_to_seconds_ceiling "$interval" || echo 0)"

    if [ "${retries:-0}" -le 0 ] || [ "${interval_seconds:-0}" -le 0 ]; then
        echo "$default_retries"
        return 0
    fi

    grace_seconds=30
    timeout_seconds=$(( start_period_seconds + interval_seconds * retries + grace_seconds ))
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

# build_frontend_host：在本机（而非容器内）构建前端。
# 关键场景：Docker daemon 拉不到 node/golang 基础镜像，而代理只监听在本机回环
# 127.0.0.1，容器内/daemon 侧都连不上 —— 这时「在本机用终端代理（HTTP(S)_PROXY）
# 编译前端」是唯一可行路径。pnpm/npm 会自动读取终端里 export 的代理环境变量。
build_frontend_host() {
    # 本机若已存在可用的前端构建产物（例如已在终端用代理 + pnpm 编译好），直接复用，
    # 既不重复构建，也不会回退去「运行容器提取旧 dist」而把新改动覆盖掉。
    if [ -d "$FRONTEND_DIR/dist" ] && [ -n "$(ls -A "$FRONTEND_DIR/dist" 2>/dev/null)" ]; then
        log_info "本机 frontend/dist 已存在，直接复用（如需重新构建请先删除该目录）"
        return 0
    fi
    log_info "尝试本机构建前端（依赖终端已 export 的 HTTP(S)_PROXY 代理）..."
    if ( cd "$FRONTEND_DIR" && npm install --no-package-lock --no-bin-links >/dev/null 2>&1 && npm run build >/dev/null 2>&1 ); then
        log_success "前端本机构建完成 -> $FRONTEND_DIR/dist"
        return 0
    fi
    # 回退：本机常因沙箱禁止 symlink(pnpm) 或 pnpm workspace 不兼容 npm 而构建失败。
    # 此时直接复用「正在运行的 devtools 容器里已经过验证的前端 dist」（即当前线上版本），
    # 它本身就是本次要发布的内容，零构建、零拉取、不依赖代理。
    log_warn "本机构建失败，回退：从正在运行的 devtools 容器提取已验证的前端 dist"
    local cid
    cid="$(docker_devtools_container_id)"
    if [ -n "$cid" ] && docker cp "${cid}:/app/dist/." "$FRONTEND_DIR/dist/"; then
        log_success "已从运行中的 devtools 容器提取 dist 作为回退"
        return 0
    fi
    log_error "前端本机构建失败且无法从运行容器提取 dist"
    return 1
}

# hot_layer_build：零拉取叠加构建。
# 以本地已缓存的 devtools-devtools:latest 为基础，仅把本机编译好的前端 dist 覆盖进去
# （见 Dockerfile.hot）。不拉取任何基础镜像、不动 daemon 配置、不影响其他容器。
# 后端二进制沿用已缓存镜像里的版本（本次前端改动无需重编后端）。
hot_layer_build() {
    log_info "HOT 叠加构建：基于本地缓存的 devtools-devtools:latest，仅覆盖新前端 dist（不拉取任何基础镜像）..."
    # 本机 Docker 不支持 --ignore-file，故构建前把 .dockerignore 临时替换为 .dockerignore.hot
    # （排除 node_modules 等重目录，但保留 frontend/dist，供 Dockerfile.hot 的 COPY 使用），
    # 构建后立即还原，不影响正常的 compose build。
    local ctx="$SCRIPT_DIR"
    local di="$ctx/.dockerignore"
    local di_hot="$ctx/.dockerignore.hot"
    local swapped=0
    if [ -f "$di" ] && [ -f "$di_hot" ]; then
        cp "$di" "$di.bak-hot"
        cp "$di_hot" "$di"
        swapped=1
    fi
    docker build -f Dockerfile.hot -t devtools-devtools:latest "$ctx"
    local rc=$?
    if [ "$swapped" -eq 1 ] && [ -f "$di.bak-hot" ]; then
        mv "$di.bak-hot" "$di"
    fi
    if [ $rc -eq 0 ]; then
        log_success "HOT 镜像已构建并打 tag devtools-devtools:latest"
    else
        log_error "HOT 镜像构建失败"
    fi
    return $rc
}

# write_askit_placeholder <目标路径>：写一个占位 zip，让 Dockerfile 的
# `COPY askit-extension.zip` 始终能找到源文件。
#
# 占位包里放的是说明文件而不是空包：扩展的真正来源是运行时的 SSH clone
# （handlers/askit_extension.go:128 复用 autodev 的 id_ed25519），占位包只在
# 运行时拉取也失败时才会被用户下载到，那时包里的 README 能直接说明问题。
write_askit_placeholder() {
    local target="$1"
    local stage
    stage="$(mktemp -d)"
    cat > "$stage/README.txt" <<'EOF'
AskIt 扩展未在本次构建中预打包。

Askit 是私有仓库，镜像构建阶段拿不到凭据，因此扩展改由宿主机构建后 COPY 进镜像；
本次宿主机构建被跳过或失败，所以这里是占位包。

容器启动后会通过 GET /api/askit/extension 从 GitHub 实时拉取并构建扩展（走 SSH，
复用 autodev 的 id_ed25519），正常情况下用户不需要这个占位包。

若你看到本文件，说明运行时拉取也失败了：请检查容器的 autodev SSH key 与网络出口，
该接口会返回具体错误信息。
EOF
    rm -f "$target"
    ( cd "$stage" && zip -q -r "$target" . )
    rm -rf "$stage"
}

# askit_bailout <原因>：AskIt 是可选能力，任何一步失败都不该阻断发布。
# 统一走这里告警 + 写占位 zip，保证 Dockerfile 的 COPY 有源文件。
askit_bailout() {
    log_warn "$1：写入占位 zip 继续（发布不阻断，扩展改由运行时 SSH 拉取）"
    write_askit_placeholder "$ASKIT_ZIP"
    # 占位包也要落到磁盘兜底目录：非 Docker 本地运行时 handler 从这里读，
    # 原来只在 bail 分支写仓库根那一份，本地跑就找不到扩展包了。
    local data_dir="${2:-}"
    if [ -n "$data_dir" ]; then
        mkdir -p "$data_dir"
        cp "$ASKIT_ZIP" "$data_dir/askit-extension.zip" || log_warn "占位 zip 写入 ${data_dir} 失败"
    fi
}

# askit_pack_dist <源目录> <输出 zip>：把源目录里的 dist/ 打成 zip。
# 失败返回非零，由调用方决定是兜底还是中断。先 rm 再 zip —— zip -r 是追加语义，
# 不先删会把上一次的条目残留在包里。
askit_pack_dist() {
    local src_dir="$1" out="$2"
    [ -d "$src_dir/dist" ] || return 1
    rm -f "$out"
    ( cd "$src_dir/dist" && zip -q -r "$out" . )
}

# build_askit 产出两处 zip：
#   1. $SCRIPT_DIR/askit-extension.zip  —— Docker 构建上下文，供 Dockerfile COPY
#   2. $BACKEND_DIR/data/askit/         —— 非 Docker 本地运行时 handler 的磁盘兜底
#
# 做法与运行时 handlers/askit_extension.go:128 buildZipFromGitHub **完全对齐**：
# 浅克隆 main，直接打包仓库里**已提交的 dist/**。不跑 npm install / npm run build。
#
# 为什么不构建源码：
#   - Askit 把构建产物 dist/ 提交进了版本库（.gitignore 里 dist/ 那行是注释掉的），
#     运行时也是直接取它 —— 本地再 build 一次只会产出与运行时不一致的包。
#   - 它给发布路径引入了 node/npm/tsc 依赖，而这条链路在本机反复出问题：npm install
#     被 Ctrl-C 打断会留下残缺的 node_modules，且后续 npm install 不会修复它
#     （实测 mermaid/dist 只剩 3 个文件、官方 tarball 里的 446 个 .d.ts 全丢，
#     于是 npm run build 报 TS7016）。发布不该依赖一个能被 Ctrl-C 破坏的本地
#     node 环境 —— 实测这两次失败全部源自 npm，与 Askit 源码无关。
#
# 为什么不在镜像里 clone：Askit 是私有仓库，容器内既无凭据也无 ssh-agent，
# `git clone` 以 "could not read Username for 'https://github.com'" (exit 128) 直接
# 打断发布。宿主机 git 用 ~/.ssh/id_ed25519（已注册到 GitHub），走 SSH 即可。
# **不要把 URL 改回 https://** —— 宿主上的 git 同样没有 HTTPS 凭据。
#
# ASKIT_DIR 环境变量可指定一个本地 checkout 直接用它的 dist/（适合"我想打包自己
# 本地改过的版本"）：**原样使用，不 fetch / 不 reset**，绝不覆盖未提交的改动。
#
# 本函数**不返回失败**：AskIt 是可选能力，不该阻断发布。任何一步失败都写占位 zip
# 并告警，发布照常继续（扩展由运行时拉取兜底）。
build_askit() {
    local ASKIT_DATA_DIR="$BACKEND_DIR/data/askit"

    if [ "${SKIP_ASKIT:-0}" = "1" ]; then
        askit_bailout "SKIP_ASKIT=1" "$ASKIT_DATA_DIR"
        return 0
    fi

    if [ -n "${ASKIT_DIR:-}" ]; then
        # 显式指定本地 checkout：原样使用它的 dist/，不 fetch / 不 reset。
        if [ ! -d "$ASKIT_DIR/dist" ]; then
            askit_bailout "ASKIT_DIR=$ASKIT_DIR 下没有 dist/（请先在该 checkout 里构建）" "$ASKIT_DATA_DIR"
            return 0
        fi
        log_info "使用本地 AskIt checkout: $ASKIT_DIR"
        if ! askit_pack_dist "$ASKIT_DIR" "$ASKIT_ZIP"; then
            askit_bailout "本地 AskIt checkout 打包失败" "$ASKIT_DATA_DIR"
            return 0
        fi
    else
        # 默认路径：与运行时 buildZipFromGitHub 一致，浅克隆 main 到临时目录。
        # trap EXIT 在子 shell 里生效，任何退出路径都会清掉临时目录。
        if ! (
            set -e
            tmp_dir="$(mktemp -d)"
            trap 'rm -rf "$tmp_dir"' EXIT
            log_info "克隆 AskIt（SSH, main）..."
            git clone --depth 1 --branch main git@github.com:axfinn/Askit.git "$tmp_dir"
            askit_pack_dist "$tmp_dir" "$ASKIT_ZIP"
        ); then
            askit_bailout "AskIt 克隆或打包失败" "$ASKIT_DATA_DIR"
            return 0
        fi
    fi

    mkdir -p "$ASKIT_DATA_DIR"
    cp "$ASKIT_ZIP" "$ASKIT_DATA_DIR/askit-extension.zip"

    log_success "AskIt 扩展打包完成: $ASKIT_ZIP"
}

build_backend() {
    log_info "构建后端..."
    cd "$BACKEND_DIR"
    # 必须用 `.` 而不是 `main.go`：package main 由 7 个文件组成(main.go 只调用
    # app_runtime.go/app_handlers.go/app_http.go/app_startup.go 里定义的函数),
    # 文件列表模式只编译列出的文件 → undefined: newAppRuntime。
    # -tags with_utls 与 Dockerfile:42 保持一致,否则 sing-box Reality 启动即报错。
    go build -tags with_utls -o server .
    log_success "后端构建完成"
}

build() {
    build_frontend
    build_backend
    build_askit
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

    local skip_compose_build=0

    if [ "${HOT_DEPLOY:-0}" = "1" ]; then
        # 受限网络场景：Docker daemon 拉不到 golang/node 基础镜像，而代理只在本机回环
        # 127.0.0.1，容器内/daemon 侧都连不上。改为「本机用终端代理编译前端 +
        # 叠加到已缓存镜像」，零拉取、不动 daemon 配置、不影响其他容器。
        log_info "HOT_DEPLOY=1：跳过完整 compose build，改为本机构建前端并叠加到已缓存镜像"
        build_frontend_host
        hot_layer_build
        skip_compose_build=1
    else
        # Dockerfile 里是 `COPY askit-extension.zip`，构建前上下文里必须先有它。
        # 放在 compose build 之前，保证 docker 部署仍然是一条命令。
        # 失败不阻断（见 build_askit 注释）。
        build_askit

        log_info "先构建镜像，避免构建阶段影响线上服务..."
        if [ "${NO_CACHE:-0}" = "1" ]; then
            if compose build --no-cache; then
                skip_compose_build=1
            fi
        else
            if compose build; then
                skip_compose_build=1
            fi
        fi

        if [ "$skip_compose_build" -eq 0 ]; then
            # compose build 失败（绝大多数是 golang/node 基础镜像拉取超时）。
            # 自动回退到 HOT 叠加部署，保证发布仍能完成。
            log_warn "compose build 失败，自动回退到 HOT 叠加部署（本机构建前端 + 叠加到已缓存镜像）"
            build_frontend_host
            hot_layer_build
            skip_compose_build=1
        fi
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

docker_warm_asr() {
    local service_wait_retries="${SERVICE_WAIT_RETRIES:-600}"

    log_info "预热 ASR Docker 服务..."
    cd "$SCRIPT_DIR"

    if [ "${NO_CACHE:-0}" = "1" ]; then
        compose build --no-cache asr-service
    else
        compose build asr-service
    fi

    log_info "下载并缓存 ASR 模型到 Docker volume..."
    compose run --rm --no-deps asr-service \
        python -c "import os; from faster_whisper import WhisperModel; WhisperModel(os.getenv('ASR_MODEL', 'small'), device=os.getenv('ASR_DEVICE', 'cpu'), compute_type=os.getenv('ASR_COMPUTE_TYPE', 'int8'))"

    log_info "启动 asr-service 并等待健康检查..."
    compose up -d asr-service
    if wait_for_container_health "asr-service" "$service_wait_retries" 3; then
        log_success "asr-service 已就绪"
        compose ps asr-service
        return
    fi

    log_error "asr-service 预热失败"
    compose ps asr-service || true
    compose logs --tail=100 asr-service || true
    exit 1
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
    echo "  build-askit 单独构建 AskIt 扩展 zip (Docker 构建上下文所需)"
    echo ""
    echo "Docker 命令:"
    echo "  docker            Docker 先构建、再替换 devtools，失败时尝试回滚"
    echo "                    受限网络可加 HOT_DEPLOY=1 走零拉取叠加（本机用终端代理编译前端）"
    echo "  docker-warm-asr   预构建 asr-service、预下载模型并等待就绪"
    echo "  docker-stop       停止 Docker 容器"
    echo "  docker-restart    重启 Docker 容器"
    echo "  docker-logs       查看 Docker 日志"
    echo "  docker-status     查看 Docker 状态"
    echo ""
    echo "  help        显示帮助"
    echo ""
    echo "全局选项:"
    echo "  --no-cache, -n   Docker 构建时禁用缓存 (等价于 NO_CACHE=1)"
    echo ""
    echo "环境变量:"
    echo "  PORT        本地服务端口 (默认: 8080)"
    echo "  HOST_PORT   Docker 映射端口 (默认: 8082)"
    echo "  NO_CACHE    Docker 构建时禁用缓存，设为 1 生效"
    echo "  SKIP_ASKIT  设为 1 时跳过 AskIt 构建，写占位 zip 保证镜像仍可构建"
    echo "  ASKIT_DIR   指定本地 AskIt checkout，直接打包它的 dist/（默认: 浅克隆 main）"
    echo ""
    echo "示例:"
    echo "  ./deploy.sh deploy              # 本地完整部署"
    echo "  ./deploy.sh docker              # Docker 部署"
    echo "  ./deploy.sh docker-logs         # 查看 Docker 日志"
    echo "  PORT=8081 ./deploy.sh start     # 指定端口启动本地服务"
    echo "  ./deploy.sh docker --no-cache   # 无缓存重建 Docker 镜像"
    echo "  NO_CACHE=1 ./deploy.sh docker   # 同上 (环境变量写法)"
    echo "  ./deploy.sh docker-warm-asr     # 预热 ASR 镜像和模型缓存"
}

# 解析全局选项：--no-cache/-n 等价于 NO_CACHE=1，可放在命令前后任意位置
args=()
for arg in "$@"; do
    case "$arg" in
        --no-cache|-n)
            NO_CACHE=1
            ;;
        *)
            args+=("$arg")
            ;;
    esac
done
set -- "${args[@]+"${args[@]}"}"

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
    build-askit|askit)
        build_askit
        ;;
    docker)
        docker_deploy
        ;;
    docker-warm-asr)
        docker_warm_asr
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
