#!/bin/bash
set -e

echo "=== 修复无头模式部署 ==="

# 当前目录
DEPLOY_DIR=$(pwd)
echo "部署目录: $DEPLOY_DIR"

# 1. 检查 gh CLI
echo "步骤1: 检查 gh CLI..."
if ! command -v gh &> /dev/null; then
    echo "错误: 未安装 gh CLI，请先安装:"
    echo "  Ubuntu/Debian: sudo apt install gh"
    echo "  CentOS/RHEL: sudo dnf install gh"
    exit 1
fi

if ! gh auth status &> /dev/null; then
    echo "错误: gh CLI 未登录，请运行: gh auth login"
    exit 1
fi

# 2. 卸载snap版chromium
echo "步骤2: 移除snap版chromium..."
sudo snap remove chromium 2>/dev/null || true
sudo apt remove -y chromium-browser 2>/dev/null || true

# 3. 安装无头Chromium
echo "步骤3: 安装无头Chromium..."
sudo apt update
sudo apt install -y chromium chromium-driver

# 或者使用Google Chrome Headless
if ! command -v chromium &> /dev/null; then
    echo "Chromium未找到，尝试安装Google Chrome..."
    wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | sudo apt-key add -
    sudo sh -c 'echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google-chrome.list'
    sudo apt update
    sudo apt install -y google-chrome-stable
fi

# 4. 下载xiaohongshu-mcp
echo "步骤4: 下载xiaohongshu-mcp二进制文件..."

# 获取最新版本
echo "获取最新版本..."
VERSION=$(gh release view --repo vmxmy/xiaohongshu-mcp --json tagName --jq '.tagName')
if [ -z "$VERSION" ]; then
    echo "错误: 无法获取最新版本"
    exit 1
fi
echo "最新版本: $VERSION"

# 检测架构
ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then
    BINARY="xiaohongshu-mcp-linux-amd64"
elif [ "$ARCH" = "aarch64" ]; then
    BINARY="xiaohongshu-mcp-linux-arm64"
else
    echo "不支持的架构: $ARCH"
    exit 1
fi

echo "下载 $BINARY ..."
gh release download "${VERSION}" \
    --repo "vmxmy/xiaohongshu-mcp" \
    --pattern "${BINARY}" \
    --clobber
mv "${BINARY}" xiaohongshu-mcp
chmod +x xiaohongshu-mcp

# 4. 创建目录
echo "步骤4: 创建必要目录..."
mkdir -p logs pids

# 5. 验证二进制文件
echo "步骤5: 验证二进制文件..."
./xiaohongshu-mcp --help || echo "注意: 可能需要安装依赖"

# 6. 检测浏览器路径
echo "步骤6: 检测浏览器路径..."
BROWSER_PATH=""
if command -v chromium &> /dev/null; then
    BROWSER_PATH=$(which chromium)
elif command -v google-chrome &> /dev/null; then
    BROWSER_PATH=$(which google-chrome)
elif command -v chromium-browser &> /dev/null; then
    BROWSER_PATH=$(which chromium-browser)
fi

echo "浏览器路径: $BROWSER_PATH"

# 7. 更新ecosystem.config.js中的浏览器路径
echo "步骤7: 更新PM2配置..."
if [ -n "$BROWSER_PATH" ]; then
    sed -i "s|// ROD_BROWSER_BIN:.*|ROD_BROWSER_BIN: '$BROWSER_PATH'|g" ecosystem.config.js
    sed -i "s|cwd:.*|cwd: '$DEPLOY_DIR',|g" ecosystem.config.js
fi

echo ""
echo "=== 修复完成 ==="
echo "浏览器: $BROWSER_PATH"
echo "二进制: $DEPLOY_DIR/xiaohongshu-mcp"
echo ""
echo "现在可以运行:"
echo "  pm2 start ecosystem.config.js"
