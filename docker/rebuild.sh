#!/bin/bash
set -e

echo "🔄 开始重建 Docker 镜像..."
echo ""

# 进入 docker 目录
cd "$(dirname "$0")"

# 1. 拉取最新代码
echo "📥 拉取最新代码..."
cd ..
git pull origin main
echo ""

# 2. 下载最新 release
echo "📦 下载最新 release..."
./build_release.sh
echo ""

# 3. 停止旧容器
echo "🛑 停止旧容器..."
cd docker
docker compose down
echo ""

# 4. 强制重新构建（不使用缓存）
echo "🔨 重新构建镜像（不使用缓存）..."
docker compose build --no-cache
echo ""

# 5. 启动新容器
echo "🚀 启动新容器..."
docker compose up -d
echo ""

# 6. 等待启动
echo "⏳ 等待服务启动..."
sleep 3
echo ""

# 7. 查看日志
echo "📋 查看启动日志（按 Ctrl+C 退出）..."
docker compose logs -f
