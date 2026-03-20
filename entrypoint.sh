#!/bin/sh

CLAWTEST_DIR=/app/data/clawtest
CLAUDE_HOME=/home/autodev/.claude

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

cd /app

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

exec ./server
