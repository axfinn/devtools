#!/bin/bash
# Codex 配置切换脚本
# 用法: ./switch_config.sh [devtools|packyapi]

CODEX_DIR="$HOME/.codex"
CONFIG_FILE="$CODEX_DIR/config.toml"
AUTH_FILE="$CODEX_DIR/auth.json"

show_usage() {
    echo "用法: $0 [devtools|packyapi]"
    echo ""
    echo "可用配置:"
    echo "  devtools  - jaxiuaicoding (gpt-5.4)"
    echo "  packyapi  - jaxiuapi (gpt-5.4)"
    echo ""
    echo "当前配置:"
    grep "^model_provider" "$CONFIG_FILE" 2>/dev/null || echo "未知"
}

switch_to() {
    local target=$1
    if [[ ! -f "$CODEX_DIR/config.toml.$target" || ! -f "$CODEX_DIR/auth.json.$target" ]]; then
        echo "错误: 配置 $target 不存在"
        exit 1
    fi

    cp -f "$CODEX_DIR/config.toml.$target" "$CONFIG_FILE"
    cp -f "$CODEX_DIR/auth.json.$target" "$AUTH_FILE"

    echo "已切换到 $target"
    echo "  model_provider: $(grep '^model_provider' "$CONFIG_FILE")"
    echo "  model: $(grep '^model' "$CONFIG_FILE")"
}

if [[ $# -eq 0 ]]; then
    show_usage
    exit 0
fi

case "$1" in
    devtools|packyapi)
        switch_to "$1"
        ;;
    *)
        echo "错误: 未知配置 $1"
        show_usage
        exit 1
        ;;
esac
