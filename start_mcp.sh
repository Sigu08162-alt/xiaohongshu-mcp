#!/bin/bash
set -e

echo "=========================================="
echo "小红书 MCP 服务器 - 本地测试"
echo "=========================================="
echo ""

# 检查必需文件
echo "📋 步骤1: 检查必需文件"
echo ""

REQUIRED_FILES=(
    "bin/xiaohongshu-mcp"
    "config.yaml"
    "cookies.json"
    "selectors_discovered_pages_creator.yaml"
)

MISSING_FILES=()

for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo "  ✓ $file"
    else
        echo "  ✗ $file (缺失)"
        MISSING_FILES+=("$file")
    fi
done

echo ""

if [ ${#MISSING_FILES[@]} -gt 0 ]; then
    echo "⚠️  缺少必需文件，请先准备："
    echo ""

    for file in "${MISSING_FILES[@]}"; do
        case "$file" in
            "bin/xiaohongshu-mcp")
                echo "  - $file: 运行 'go build -o bin/xiaohongshu-mcp .'"
                ;;
            "cookies.json")
                echo "  - $file: 运行 './login.sh' 登录"
                ;;
            "selectors_discovered_pages_creator.yaml")
                echo "  - $file: 运行 './collect_all.sh' 采集选择器"
                ;;
        esac
    done
    echo ""
    exit 1
fi

echo "✅ 所有必需文件已就绪"
echo ""

# 询问运行模式
echo "📋 步骤2: 选择运行模式"
echo ""
echo "1. 有头模式（推荐调试） - 可以看到浏览器窗口"
echo "2. 无头模式（后台运行） - 浏览器在后台运行"
echo ""
read -p "请选择 [1/2, 默认1]: " MODE_CHOICE

HEADLESS_FLAG=""
if [ "$MODE_CHOICE" = "2" ]; then
    HEADLESS_FLAG="--headless"
    echo "   选择: 无头模式"
else
    HEADLESS_FLAG="--headless=false"
    echo "   选择: 有头模式"
fi

echo ""

# 询问端口
echo "📋 步骤3: 设置端口"
echo ""
read -p "端口号 [默认: 18060]: " PORT_INPUT

PORT=":18060"
if [ -n "$PORT_INPUT" ]; then
    PORT=":$PORT_INPUT"
fi

echo "   端口: $PORT"
echo ""

# 显示启动命令
echo "=========================================="
echo "🚀 启动 MCP 服务器"
echo "=========================================="
echo ""
echo "启动命令:"
echo "  ./bin/xiaohongshu-mcp $HEADLESS_FLAG --port $PORT"
echo ""
echo "MCP 端点:"
echo "  http://localhost${PORT}/mcp"
echo ""
echo "REST API:"
echo "  http://localhost${PORT}/api/v1/"
echo ""
echo "Swagger 文档:"
echo "  http://localhost${PORT}/swagger/index.html"
echo ""
echo "按 Ctrl+C 停止服务器"
echo "=========================================="
echo ""

# 启动服务器
./bin/xiaohongshu-mcp $HEADLESS_FLAG --port $PORT
