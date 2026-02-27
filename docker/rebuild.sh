#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "=== Docker 镜像重建（本地编译）==="

if ! command -v go >/dev/null 2>&1; then
    echo "未检测到 go，请先安装 Go 环境。"
    exit 1
fi

VERSION="$(git describe --tags --always --dirty 2>/dev/null || true)"
if [[ -z "$VERSION" ]]; then
    VERSION="local-$(date +%Y%m%d%H%M%S)"
fi

echo "    构建版本: $VERSION"
if [[ "${NON_INTERACTIVE:-}" != "1" ]]; then
    echo ""
    read -p "确认使用本地代码构建并重启容器？[y/N] " confirm
    if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
        echo "已取消构建。"
        exit 0
    fi
fi

echo "[1/5] 本地编译 Linux 二进制..."
mkdir -p release
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o release/xiaohongshu-mcp-linux-amd64 .

echo "[2/5] 停止旧容器..."
cd docker
docker compose down

echo "[3/5] 清理旧镜像..."
docker rmi xiaohongshu-mcp 2>/dev/null || true

echo "[4/5] 构建镜像 ($VERSION)..."
docker compose build --no-cache --build-arg VERSION="$VERSION"

echo "[5/5] 启动容器..."
docker compose up -d

sleep 3
docker compose ps

# 清理本地构建产物
rm -rf ../release

echo ""
echo "=== 重建完成 ==="
echo "查看日志: docker compose logs -f"
