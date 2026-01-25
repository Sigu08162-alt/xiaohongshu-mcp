#!/bin/bash
set -e

echo "=== 小红书MCP服务器部署脚本 ==="

# 配置变量
DEPLOY_DIR="${DEPLOY_DIR:-$(pwd)}"  # 默认使用当前目录，可通过环境变量覆盖
GITHUB_REPO="vmxmy/xiaohongshu-mcp"

echo "部署目录: $DEPLOY_DIR"

# 获取最新版本
echo "获取最新版本..."
VERSION=$(gh release view --repo $GITHUB_REPO --json tagName --jq '.tagName')
if [ -z "$VERSION" ]; then
    echo "错误: 无法获取最新版本"
    exit 1
fi
echo "最新版本: $VERSION"

# 1. 检查必要工具
echo "步骤1: 检查必要工具..."

# 检查 gh CLI
if ! command -v gh &> /dev/null; then
    echo "错误: 未安装 gh CLI，请先安装:"
    echo "  Ubuntu/Debian: sudo apt install gh"
    echo "  CentOS/RHEL: sudo dnf install gh"
    exit 1
fi

# 检查 gh 登录状态
if ! gh auth status &> /dev/null; then
    echo "错误: gh CLI 未登录，请运行: gh auth login"
    exit 1
fi

# 安装浏览器依赖
echo "检查浏览器依赖..."
if command -v apt &> /dev/null; then
    sudo apt update
    # 尝试安装 chromium，兼容不同的包名
    if apt-cache show chromium &> /dev/null; then
        sudo apt install -y chromium curl
    elif apt-cache show chromium-browser &> /dev/null; then
        sudo apt install -y chromium-browser curl
    else
        echo "警告: 未找到 chromium 包，Playwright 将自动下载浏览器"
        sudo apt install -y curl
    fi
elif command -v yum &> /dev/null; then
    sudo yum install -y chromium curl
fi

# 2. 安装PM2
echo "步骤2: 安装PM2..."
if ! command -v pm2 &> /dev/null; then
    if ! command -v npm &> /dev/null; then
        echo "错误: 未安装Node.js，请先安装Node.js和npm"
        echo "访问: https://nodejs.org/ 或使用包管理器安装"
        exit 1
    fi
    npm install -g pm2
fi

# 3. 进入部署目录
echo "步骤3: 进入部署目录..."
mkdir -p "$DEPLOY_DIR"
cd "$DEPLOY_DIR"

# 4. 下载二进制文件
echo "步骤4: 下载MCP服务器..."
gh release download "${VERSION}" \
    --repo "${GITHUB_REPO}" \
    --pattern "xiaohongshu-mcp-linux-amd64.tar.gz" \
    --clobber

# 解压
echo "解压文件..."
tar -xzf xiaohongshu-mcp-linux-amd64.tar.gz
rm xiaohongshu-mcp-linux-amd64.tar.gz
chmod +x xiaohongshu-mcp

# 5. 创建目录
echo "步骤5: 创建必要目录..."
mkdir -p logs pids

# 6. 创建配置文件
echo "步骤6: 创建PM2配置文件..."
cat > ecosystem.config.js << EOFCONFIG
module.exports = {
  apps: [
    {
      name: 'xiaohongshu-mcp',
      script: './xiaohongshu-mcp',
      args: '--headless=true --port=:18060',
      cwd: '${DEPLOY_DIR}',
      instances: 1,
      exec_mode: 'fork',
      autorestart: true,
      watch: false,
      max_memory_restart: '500M',
      env: {
        NODE_ENV: 'production',
        PORT: '18060'
      },
      log_date_format: 'YYYY-MM-DD HH:mm:ss Z',
      error_file: './logs/mcp-error.log',
      out_file: './logs/mcp-out.log',
      log_file: './logs/mcp-combined.log',
      max_restarts: 10,
      min_uptime: '10s',
      pid_file: './pids/mcp.pid',
      kill_timeout: 5000,
      wait_ready: true,
      listen_timeout: 10000,
      merge_logs: true,
      time: true
    }
  ]
}
EOFCONFIG

# 7. 启动/重启服务
echo "步骤7: 启动/重启PM2服务..."
if pm2 list | grep -q "xiaohongshu-mcp"; then
    echo "检测到已运行的服务，执行重启..."
    pm2 restart xiaohongshu-mcp
else
    echo "首次部署，启动新服务..."
    pm2 start ecosystem.config.js
fi

# 8. 配置开机自启
echo "步骤8: 配置开机自启..."
pm2 save
echo ""
echo "请执行以下命令以启用开机自启："
pm2 startup systemd | grep "sudo"

echo ""
echo "=== 部署完成 ==="
echo ""
echo "常用命令:"
echo "  查看状态: pm2 status"
echo "  查看日志: pm2 logs xiaohongshu-mcp"
echo "  重启服务: pm2 restart xiaohongshu-mcp"
echo "  停止服务: pm2 stop xiaohongshu-mcp"
echo ""
echo "测试API:"
echo "  curl http://localhost:18060/health"
echo ""
echo "部署目录: $DEPLOY_DIR"
