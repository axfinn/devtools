#!/bin/sh

CLAWTEST_DIR=/app/data/clawtest
CLAUDE_HOME=/home/autodev/.claude
CODEX_HOME=/home/autodev/.codex
HOST_CODEX_HOME=/tmp/host-codex
CONFIG_SRC_DIR=/app/config-src
CONFIG_TARGET_PATH=${CONFIG_PATH:-/app/config.yaml}

# 1. 更新 clawtest（在 data volume 里，重建镜像不丢失）
if [ -d "${CLAWTEST_DIR}/.git" ]; then
    echo "[entrypoint] Pulling latest clawtest..."
    git -C "${CLAWTEST_DIR}" pull --depth=1 origin main 2>&1 && \
        echo "[entrypoint] clawtest updated" || \
        echo "[entrypoint] git pull failed, using cached version"
else
    echo "[entrypoint] Cloning clawtest..."
    git clone --depth=1 https://github.com/axfinn/clawtest.git "${CLAWTEST_DIR}" 2>&1 && \
        echo "[entrypoint] clawtest cloned" || \
        echo "[entrypoint] git clone failed, autodev unavailable"
fi

if [ -f "${CLAWTEST_DIR}/autodev/autodev" ]; then
    chmod +x "${CLAWTEST_DIR}/autodev/autodev"
    chmod +x "${CLAWTEST_DIR}/autodev/autodev-stop"
fi

# 2. 每次启动重新生成 settings.json，确保 MINIMAX_API_KEY 等 env 变量实时生效
echo "[entrypoint] Writing ${CLAUDE_HOME}/settings.json..."
mkdir -p "${CLAUDE_HOME}/skills"
cat > "${CLAUDE_HOME}/settings.json" << EOF
{
  "skills": {
    "paths": ["~/.claude/skills"]
  },
  "mcpServers": {
    "MiniMax": {
      "command": "uvx",
      "args": ["minimax-coding-plan-mcp", "-y"],
      "env": {
        "MINIMAX_API_KEY": "${MINIMAX_API_KEY}",
        "MINIMAX_API_HOST": "https://api.minimaxi.com"
      }
    }
  },
  "env": {
    "API_TIMEOUT_MS": "3000000",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
    "ANTHROPIC_MODEL": "MiniMax-M2.7",
    "ANTHROPIC_SMALL_FAST_MODEL": "MiniMax-M2.7",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "MiniMax-M2.7",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "MiniMax-M2.7",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "MiniMax-M2.7"
  }
}
EOF

# 每次启动修正权限（防止重建镜像后 UID 变化）
chown -R autodev:autodev "${CLAUDE_HOME}"

# 3. 如宿主机提供 ~/.codex，则安全同步必要配置到容器内的 Codex Home。
# 只复制配置和认证文件，不把任何敏感内容写进仓库。
echo "[entrypoint] Preparing ${CODEX_HOME}..."
mkdir -p "${CODEX_HOME}"
if [ -d "${HOST_CODEX_HOME}" ]; then
    echo "[entrypoint] Syncing host Codex config from ${HOST_CODEX_HOME}..."
    for file in config.toml auth.json version.json .personality_migration; do
        if [ -f "${HOST_CODEX_HOME}/${file}" ]; then
            cp "${HOST_CODEX_HOME}/${file}" "${CODEX_HOME}/${file}"
        fi
    done
    if [ -d "${HOST_CODEX_HOME}/rules" ]; then
        mkdir -p "${CODEX_HOME}/rules"
        cp -R "${HOST_CODEX_HOME}/rules/." "${CODEX_HOME}/rules/" 2>/dev/null || true
    fi
else
    echo "[entrypoint] Host Codex config not found; using container-local Codex config"
fi
chown -R autodev:autodev "${CODEX_HOME}"

# ──────────────────────────────────────────────────────────────
# 修复：确保 autodev 用户可以运行 uvx (MCP server) 和创建任务目录
# ──────────────────────────────────────────────────────────────

# 修复1: 创建 uv 缓存目录（Go 代码设置 UV_CACHE_DIR=/tmp/uv-cache-autodev）
mkdir -p /tmp/uv-cache-autodev
chown autodev:autodev /tmp/uv-cache-autodev

# 修复2: uvx 安装在 /root/.local/bin/ 下，但 /root/ 是 700 权限，autodev 无法访问
# /root/.local/bin/ 本身也是 700，所以即使 symlink 也无法访问目标
# 必须复制二进制文件到 /usr/local/bin/（autodev 的 PATH 包含此目录）
if [ ! -f /usr/local/bin/uvx ] && [ -f /root/.local/bin/uvx ]; then
    cp /root/.local/bin/uvx /usr/local/bin/uvx
    chmod 755 /usr/local/bin/uvx
    echo "[entrypoint] Copied uvx to /usr/local/bin/"
fi
if [ ! -f /usr/local/bin/uv ] && [ -f /root/.local/bin/uv ]; then
    cp /root/.local/bin/uv /usr/local/bin/uv
    chmod 755 /usr/local/bin/uv
    echo "[entrypoint] Copied uv to /usr/local/bin/"
fi

# 修复3: 确保 autodev 用户可以在 /app/data/autodev 中创建任务目录
chown autodev:autodev /app/data/autodev

# 修复4: /tmp 需要对所有用户可写（Claude Code 等工具需要）
chmod 777 /tmp

# 修复5: Claude Code 运行时需要可写的临时目录
mkdir -p /tmp/claude-1001
chown autodev:autodev /tmp/claude-1001

# 修复6: 设置 TMPDIR 作为默认临时目录
export TMPDIR=/tmp/uv-cache-autodev

cd /app

# 准备运行时配置文件：优先使用自定义 config.yaml，不存在时回落到 config.example.yaml
if [ -d "${CONFIG_SRC_DIR}" ]; then
    if [ -f "${CONFIG_SRC_DIR}/config.yaml" ]; then
        cp "${CONFIG_SRC_DIR}/config.yaml" "${CONFIG_TARGET_PATH}"
        echo "[entrypoint] Using custom config.yaml"
    elif [ -f "${CONFIG_SRC_DIR}/config.example.yaml" ]; then
        cp "${CONFIG_SRC_DIR}/config.example.yaml" "${CONFIG_TARGET_PATH}"
        echo "[entrypoint] Using config.example.yaml"
    fi
fi

# Start TTS HTTP service in background (edge-tts via FastAPI on 127.0.0.1:8083)
echo "[entrypoint] Starting TTS service on port 8083..."
TTS_OUTPUT_DIR=/app/data/uploads /app/tts-venv/bin/python /app/tts_server.py &
TTS_PID=$!

# Wait for TTS service to be ready (max 10s)
for i in $(seq 1 10); do
    if wget -q --spider http://127.0.0.1:8083/health 2>/dev/null; then
        echo "[entrypoint] TTS service ready"
        break
    fi
    sleep 1
done

# 启动 npc，将 tunnel_port 注册到 NPS（需要 NPS_SERVER / NPS_VKEY / PROXY_TUNNEL_PORT 环境变量）
if [ -n "$NPS_SERVER" ] && [ -n "$NPS_VKEY" ] && [ -n "$PROXY_TUNNEL_PORT" ]; then
    NPC_BIN=/app/data/npc
    if [ ! -f "$NPC_BIN" ]; then
        echo "[entrypoint] Downloading npc..."
        wget -qO /tmp/npc.tar.gz "${NPS_SERVER}/static/download/client/npc_linux_amd64.tar.gz" && \
            tar -xzf /tmp/npc.tar.gz -C /tmp && \
            mv /tmp/npc "$NPC_BIN" && \
            chmod +x "$NPC_BIN" && \
            echo "[entrypoint] npc downloaded" || \
            echo "[entrypoint] npc download failed, skipping"
    fi
    if [ -f "$NPC_BIN" ]; then
        mkdir -p /app/data/npc-conf
        cat > /app/data/npc-conf/npc.conf << EOF
[common]
server_addr=${NPS_SERVER}
conn_type=tcp
vkey=${NPS_VKEY}
auto_reconnect=true
log_level=3

[proxy]
mode=tcp
server_port=${PROXY_TUNNEL_PORT}
target_addr=127.0.0.1:${PROXY_TUNNEL_PORT}
EOF
        echo "[entrypoint] Starting npc (server=${NPS_SERVER} tunnel_port=${PROXY_TUNNEL_PORT})..."
        "$NPC_BIN" -config /app/data/npc-conf/npc.conf &
        echo "[entrypoint] npc started"
    fi
fi

exec ./server
