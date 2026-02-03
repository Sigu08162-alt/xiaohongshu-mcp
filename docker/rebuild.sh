#!/bin/bash
set -e

echo "========================================="
echo "🔄 Docker 镜像重建脚本"
echo "========================================="
echo ""

# 进入 docker 目录
cd "$(dirname "$0")"

# 1. 拉取最新代码
echo "📥 [1/6] 拉取最新代码..."
cd ..
git pull origin main
echo ""

# 2. 下载最新 release（强制覆盖）
echo "📦 [2/6] 下载最新 release（强制覆盖）..."
./build_release.sh
echo ""

# 3. 停止旧容器
echo "🛑 [3/6] 停止旧容器..."
cd docker
docker compose down
echo ""

# 4. 清理旧镜像（可选，节省空间）
echo "🗑️  [4/6] 清理旧镜像..."
docker rmi xiaohongshu-mcp 2>/dev/null || echo "   没有找到旧镜像，跳过"
echo ""

# 5. 强制重新构建（不使用缓存）
echo "🔨 [5/6] 重新构建镜像（不使用缓存）..."
docker compose build --no-cache --pull
echo ""

# 6. 启动新容器
echo "🚀 [6/6] 启动新容器..."
docker compose up -d
echo ""

# 等待启动
echo "⏳ 等待服务启动..."
sleep 3
echo ""

# 显示容器状态
echo "📊 容器状态："
docker compose ps
echo ""

echo "========================================="
echo "✅ 重建完成！"
echo "========================================="
echo ""
echo "查看日志: docker compose logs -f"
echo "停止服务: docker compose down"
echo "重启服务: docker compose restart"
echo ""
