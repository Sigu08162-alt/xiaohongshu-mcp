#!/bin/bash
set -euo pipefail

REPO="vmxmy/xiaohongshu-mcp"
INSTALL_DIR="$(cd "$(dirname "$0")" && pwd)"
SERVICE_NAME="xiaohongshu-mcp"

# 检测平台
detect_platform() {
    local os arch
    os=$(uname -s)
    arch=$(uname -m)

    case "$os" in
        Linux*)  OS_TYPE="linux" ;;
        Darwin*) OS_TYPE="darwin" ;;
        *)       echo "不支持的系统: $os"; exit 1 ;;
    esac

    case "$arch" in
        x86_64|amd64)   ARCH_TYPE="amd64" ;;
        aarch64|arm64)  ARCH_TYPE="arm64" ;;
        *)              echo "不支持的架构: $arch"; exit 1 ;;
    esac

    PLATFORM="${OS_TYPE}-${ARCH_TYPE}"
    BINARY="${SERVICE_NAME}-${PLATFORM}"
    ASSET="${BINARY}.tar.gz"
}

# 获取当前运行版本的 commit hash
current_commit() {
    if [ -f "$INSTALL_DIR/$BINARY" ]; then
        "$INSTALL_DIR/$BINARY" --version 2>&1 | grep -oE '[0-9a-f]{7,}' | tail -1 || echo ""
    else
        echo ""
    fi
}

# 获取最新 release
latest_release() {
    gh release view --repo "$REPO" --json tagName --jq '.tagName' 2>/dev/null
}

# 备份当前版本
backup() {
    local ts
    ts=$(date +%Y%m%d%H%M%S)
    local backup_dir="$INSTALL_DIR/.backup"
    mkdir -p "$backup_dir"

    for bin in "$BINARY" "xiaohongshu-login-${PLATFORM}"; do
        [ -f "$INSTALL_DIR/$bin" ] && cp "$INSTALL_DIR/$bin" "$backup_dir/${bin}.${ts}"
    done

    # 只保留最近 3 份备份
    ls -t "$backup_dir"/${BINARY}.* 2>/dev/null | tail -n +4 | xargs -r rm -f
    echo "$backup_dir"
}

# 下载���解压
download() {
    local version="$1"
    local tmpdir
    tmpdir=$(mktemp -d)

    gh release download "$version" \
        --repo "$REPO" \
        --pattern "$ASSET" \
        --dir "$tmpdir" \
        --clobber

    tar -xzf "$tmpdir/$ASSET" -C "$tmpdir"

    for bin in "$BINARY" "xiaohongshu-login-${PLATFORM}"; do
        [ -f "$tmpdir/$bin" ] && mv "$tmpdir/$bin" "$INSTALL_DIR/$bin" && chmod +x "$INSTALL_DIR/$bin"
    done

    rm -rf "$tmpdir"
}

# 重启服务
restart_service() {
    if command -v pm2 &>/dev/null && pm2 describe "$SERVICE_NAME" &>/dev/null; then
        pm2 restart "$SERVICE_NAME"
        echo "[restart] pm2 restart $SERVICE_NAME"
        return
    fi

    if command -v systemctl &>/dev/null && systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
        sudo systemctl restart "$SERVICE_NAME"
        echo "[restart] systemctl restart $SERVICE_NAME"
        return
    fi

    echo "[restart] 未检测到 pm2/systemd 服务，请手动重启"
}

# 回滚
rollback() {
    local backup_dir="$INSTALL_DIR/.backup"
    local latest_backup
    latest_backup=$(ls -t "$backup_dir"/${BINARY}.* 2>/dev/null | head -1)

    if [ -z "$latest_backup" ]; then
        echo "[rollback] 无备份可用"
        return 1
    fi

    cp "$latest_backup" "$INSTALL_DIR/$BINARY"
    chmod +x "$INSTALL_DIR/$BINARY"
    restart_service
    echo "[rollback] 已回滚到 $(basename "$latest_backup")"
}

# 主流程
main() {
    detect_platform

    echo "=== 小红书 MCP 更新 ==="
    echo "平台: $PLATFORM"
    echo "目录: $INSTALL_DIR"
    echo ""

    local latest current
    latest=$(latest_release)
    current=$(current_commit)

    if [ -z "$latest" ]; then
        echo "获取最新版本失败，请检查 gh auth status"
        exit 1
    fi

    echo "最新版本: $latest"
    echo "当前版本: ${current:-未安装}"

    # 版本比较：提取 tag 中的 commit hash
    local latest_commit
    latest_commit=$(echo "$latest" | grep -oE '[0-9a-f]{7,}$' || echo "")

    if [ -n "$current" ] && [ "$current" = "$latest_commit" ]; then
        echo ""
        echo "已是最新版本，无需更新"
        exit 0
    fi

    echo ""
    echo "开始更新..."

    local backup_dir
    backup_dir=$(backup)
    echo "[backup] $backup_dir"

    download "$latest"
    echo "[download] $latest"

    restart_service

    echo ""
    echo "=== 更新完成 ==="
}

# 支持 rollback 子命令
case "${1:-}" in
    rollback)
        detect_platform
        rollback
        ;;
    *)
        main
        ;;
esac
