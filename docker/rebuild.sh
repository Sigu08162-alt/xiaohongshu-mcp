#!/bin/bash
set -e

cd "$(dirname "$0")"

echo "=== Docker 镜像重建 ==="

cd ..
git pull origin main

cd docker
docker compose down
docker rmi xiaohongshu-mcp 2>/dev/null || true
docker compose build --no-cache
docker compose up -d

sleep 3
docker compose ps

echo ""
echo "=== 重建完成 ==="
echo "查看日志: docker compose logs -f"
