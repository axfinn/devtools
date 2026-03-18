#!/bin/sh
# 每次启动时从 GitHub 拉取最新 clawtest 代码
echo "[entrypoint] Pulling latest clawtest from GitHub..."
if cd /opt/clawtest && git pull --depth=1 origin main 2>&1; then
    chmod +x /opt/clawtest/autodev/autodev
    chmod +x /opt/clawtest/autodev/autodev-stop
    echo "[entrypoint] clawtest updated successfully"
else
    echo "[entrypoint] git pull failed, using cached version"
fi

cd /app
exec ./server
