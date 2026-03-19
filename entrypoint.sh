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

# 2. 初始化 /home/autodev/.claude（volume 首次挂载时为空）
if [ ! -f "${CLAUDE_HOME}/settings.json" ]; then
    echo "[entrypoint] Initializing ${CLAUDE_HOME}..."
    mkdir -p "${CLAUDE_HOME}/skills"
    cp /app/claude-settings-template.json "${CLAUDE_HOME}/settings.json"
fi

# 每次启动修正权限（防止重建镜像后 UID 变化）
chown -R autodev:autodev "${CLAUDE_HOME}"

cd /app
exec ./server
