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
echo "  2) 仅采集图文发布页面"
echo "  3) 仅采集视频发布页面"
echo "  4) 采集所有页面 + 输出JSON"
echo "  5) 自定义参数运行"
echo ""
read -p "请输入选项 [1-5]: " choice

case $choice in
    1)
        echo -e "\n${BLUE}📋 采集所有页面...${NC}\n"
        "$TOOL"
        ;;
    2)
        echo -e "\n${BLUE}📋 采集图文发布页面...${NC}\n"
        "$TOOL" --page publish_image --output selectors_publish_image.yaml
        ;;
    3)
        echo -e "\n${BLUE}📋 采集视频发布页面...${NC}\n"
        "$TOOL" --page publish_video --output selectors_publish_video.yaml
        ;;
    4)
        echo -e "\n${BLUE}📋 采集所有页面 (YAML + JSON)...${NC}\n"
        "$TOOL" --json selectors_all_pages.json
        ;;
    5)
        echo -e "\n${YELLOW}请输入自定义参数 (例如: --page publish_image --wait 5):${NC}"
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
echo "  - 使用 'yq' 查询 YAML: yq '.pages.publish_image.buttons[]' selectors_all_pages.yaml"
echo "  - 使用 'jq' 查询 JSON: jq '.pages' selectors_all_pages.json"
echo "  - 查看完整文档: cat cmd/refresh_selectors/README.md"
