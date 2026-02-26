#!/bin/bash
set -e

cd "$(dirname "$0")"
cd ..

echo "=== Docker 镜像重建 ==="

# 下载最新 release 二进制
echo "[1/5] 下载最新 release..."
mkdir -p release
ASSET="xiaohongshu-mcp-linux-amd64.tar.gz"
RELEASE_TAG=$(gh release list --repo vmxmy/xiaohongshu-mcp --limit 1 --json tagName -q '.[0].tagName')
echo "    版本: $RELEASE_TAG"
echo ""
read -p "确认使用此版本构建？[y/N] " confirm
if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo "已取消构建。"
    exit 0
fi
gh release download "$RELEASE_TAG" --repo vmxmy/xiaohongshu-mcp --pattern "$ASSET" --dir release --clobber
tar -xzf "release/$ASSET" -C release
rm -f "release/$ASSET"

# 构建镜像
echo "[2/5] 停止旧容器..."
cd docker
docker compose down

echo "[3/5] 清理旧镜像..."
docker rmi xiaohongshu-mcp 2>/dev/null || true

echo "[4/5] 构建镜像 ($RELEASE_TAG)..."
docker compose build --no-cache --build-arg VERSION="$RELEASE_TAG"

echo "[5/5] 启动容器..."
docker compose up -d

sleep 3
docker compose ps

# 清理下载文件
rm -rf ../release

echo ""
echo "=== 重建完成 ==="
echo "查看日志: docker compose logs -f"
