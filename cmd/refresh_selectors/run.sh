#!/bin/bash

# 小红书选择器刷新快速启动脚本

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TOOL="$PROJECT_ROOT/bin/refresh_selectors"

cd "$PROJECT_ROOT"

# 颜色输出
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== 小红书选择器刷新工具 ===${NC}\n"

# 检查工具是否存在
if [ ! -f "$TOOL" ]; then
    echo -e "${YELLOW}⚠️  工具未编译，正在编译...${NC}"
    go build -o "$TOOL" ./cmd/refresh_selectors
    echo -e "${GREEN}✅ 编译完成${NC}\n"
fi

# 显示菜单
echo "请选择操作:"
echo "  1) 采集所有页面 (推荐)"
echo "  2) 仅采集单个页面 (输入页面key)"
echo "  3) 采集所有页面 + 输出JSON"
echo "  4) 自定义参数运行"
echo ""
read -p "请输入选项 [1-4]: " choice

PAGES_FILE=${PAGES_FILE:-discovered_pages.yaml}
if [ ! -f "$PAGES_FILE" ]; then
    echo -e "\n${YELLOW}未找到发现文件: $PAGES_FILE${NC}"
    read -p "请输入 discovered_pages.yaml 路径: " PAGES_FILE
fi

if [ ! -f "$PAGES_FILE" ]; then
    echo -e "\n${YELLOW}未找到发现文件，退出${NC}"
    exit 1
fi

read -p "请输入输出文件名 (例如: selectors_discovered_pages.yaml): " OUTPUT_FILE
if [ -z "$OUTPUT_FILE" ]; then
    echo -e "\n${YELLOW}未输入输出文件名，退出${NC}"
    exit 1
fi

case $choice in
    1)
        echo -e "\n${BLUE}📋 采集所有页面...${NC}\n"
        "$TOOL" --pages "$PAGES_FILE" --output "$OUTPUT_FILE"
        ;;
    2)
        read -p "请输入页面key（来自发现文件 links 的 key）: " page_key
        if [ -z "$page_key" ]; then
            echo -e "\n${YELLOW}未输入页面key，退出${NC}"
            exit 1
        fi
        echo -e "\n${BLUE}📋 采集单个页面: $page_key${NC}\n"
        "$TOOL" --pages "$PAGES_FILE" --page "$page_key" --output "$OUTPUT_FILE"
        ;;
    3)
        echo -e "\n${BLUE}📋 采集所有页面 (YAML + JSON)...${NC}\n"
        "$TOOL" --pages "$PAGES_FILE" --output "$OUTPUT_FILE" --json selectors_all_pages.json
        ;;
    4)
        echo -e "\n${YELLOW}请输入自定义参数 (例如: --pages discovered_pages.yaml --page publish_publish --wait 5):${NC}"
        read -p "> " custom_args
        echo ""
        "$TOOL" $custom_args
        ;;
    *)
        echo -e "\n${YELLOW}无效选项，退出${NC}"
        exit 1
        ;;
esac

echo -e "\n${GREEN}✅ 完成！${NC}\n"

# 显示输出文件
echo "生成的文件:"
ls -lh selectors*.yaml 2>/dev/null || true
ls -lh selectors*.json 2>/dev/null || true

echo -e "\n${BLUE}💡 提示：${NC}"
echo "  - 使用 'yq' 查询 YAML: yq '.pages.<page_key>.buttons[]' <output.yaml>"
echo "  - 使用 'jq' 查询 JSON: jq '.pages' selectors_all_pages.json"
echo "  - 推荐默认命名:"
echo "    discover: discovered_pages_creator.yaml"
echo "    selectors: selectors_discovered_pages_creator.yaml"
echo "    metadata: metadata_discovered_pages_creator.yaml"
echo "  - 查看完整文档: cat cmd/refresh_selectors/README.md"
