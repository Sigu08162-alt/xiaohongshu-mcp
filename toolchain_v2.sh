#!/bin/bash
# 小红书MCP工具链 - 完整版（无硬编码）
#
# 工具链流程:
#   1. discover_pages   - 发现所有页面链接
#   2. collect_metadata - 无差别采集所有元素的完整元数据
#   3. extract_selectors - (未来) 基于元数据智能提取选择器
#
# 优势:
#   - 无硬编码: 不预设任何元素类型或属性
#   - 完整采集: 50+属性，覆盖所有可能的元数据
#   - 自适应: 页面改版后重新采集即可

set -e

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}==========================================${NC}"
echo -e "${BLUE}小红书MCP工具链 v2.0${NC}"
echo -e "${BLUE}特性: 无硬编码 | 完整元数据 | 智能提取${NC}"
echo -e "${BLUE}==========================================${NC}"
echo ""

# ============================================
# 步骤1: 检查登录
# ============================================
echo -e "${BLUE}📋 步骤1: 检查登录状态${NC}"
echo ""

COOKIE_FILE=""
if [ -f "cookies.json" ]; then
    COOKIE_FILE="cookies.json"
elif [ -f "$HOME/.xiaohongshu/cookies.json" ]; then
    COOKIE_FILE="$HOME/.xiaohongshu/cookies.json"
fi

if [ -z "$COOKIE_FILE" ]; then
    echo -e "${YELLOW}⚠️  未找到Cookie，需要登录${NC}"
    read -p "是否现在登录？(y/N): " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        ./login.sh
    else
        echo -e "${RED}❌ 请先运行 ./login.sh 登录${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}✅ Cookie: $COOKIE_FILE${NC}"
fi

echo ""

# ============================================
# 步骤2: 发现页面
# ============================================
echo -e "${BLUE}📋 步骤2: 发现页面链接${NC}"
echo ""

read -p "选择系统 [1=创作者(默认), 2=用户, 3=两者]: " SYSTEM_CHOICE
echo ""

case $SYSTEM_CHOICE in
    2)
        echo -e "${GREEN}🔍 发现用户系统页面...${NC}"
        ./bin/discover_pages --system user --no-interactive --wait 8 --output discovered_pages_user.yaml
        DISCOVERED_FILE="discovered_pages_user.yaml"
        ;;
    3)
        echo -e "${GREEN}🔍 发现创作者系统页面...${NC}"
        ./bin/discover_pages --system creator --no-interactive --wait 8 --output discovered_pages_creator.yaml

        echo ""
        echo -e "${GREEN}🔍 发现用户系统页面...${NC}"
        ./bin/discover_pages --system user --no-interactive --wait 8 --output discovered_pages_user.yaml

        # 合并两个文件
        echo -e "${YELLOW}📦 合并页面列表...${NC}"
        python3 <<EOF
import yaml

with open('discovered_pages_creator.yaml') as f:
    creator = yaml.safe_load(f)
with open('discovered_pages_user.yaml') as f:
    user = yaml.safe_load(f)

merged = {
    'pages': {
        **creator.get('pages', {}),
        **user.get('pages', {})
    }
}

with open('discovered_pages_all.yaml', 'w') as f:
    yaml.dump(merged, f, allow_unicode=True)

print(f"✓ 合并完成: {len(merged['pages'])} 个页面")
EOF
        DISCOVERED_FILE="discovered_pages_all.yaml"
        ;;
    *)
        echo -e "${GREEN}🔍 发现创作者系统页面...${NC}"
        ./bin/discover_pages --system creator --no-interactive --wait 8 --output discovered_pages_creator.yaml
        DISCOVERED_FILE="discovered_pages_creator.yaml"
        ;;
esac

if [ ! -f "$DISCOVERED_FILE" ]; then
    echo -e "${RED}❌ 页面发现失败${NC}"
    exit 1
fi

# 统计发现的页面
PAGE_COUNT=$(python3 -c "import yaml; print(len(yaml.safe_load(open('$DISCOVERED_FILE'))['pages']))")
echo -e "${GREEN}✅ 发现 $PAGE_COUNT 个页面${NC}"
echo ""

# ============================================
# 步骤3: 采集完整元数据
# ============================================
echo -e "${BLUE}📋 步骤3: 采集完整元数据${NC}"
echo -e "${YELLOW}说明: 无差别采集所有元素的所有属性（50+字段）${NC}"
echo ""

OUTPUT_METADATA="metadata_$(basename $DISCOVERED_FILE)"

read -p "是否使用交互模式（可手动操作页面如上传图片）？(Y/n): " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Nn]$ ]]; then
    # 非交互模式
    echo -e "${GREEN}🤖 非交互模式（自动采集）${NC}"
    ./bin/collect_metadata --input "$DISCOVERED_FILE" --output "$OUTPUT_METADATA" --no-interactive --wait 5
else
    # 交互模式
    echo -e "${GREEN}👤 交互模式${NC}"
    echo -e "${YELLOW}提示: 浏览器会打开每个页面${NC}"
    echo -e "${YELLOW}      如需手动操作（如上传图片），操作后按Enter继续${NC}"
    echo ""
    read -p "按 Enter 开始..."

    ./bin/collect_metadata --input "$DISCOVERED_FILE" --output "$OUTPUT_METADATA" --wait 5
fi

if [ ! -f "$OUTPUT_METADATA" ]; then
    echo -e "${RED}❌ 元数据采集失败${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✅ 元数据采集完成${NC}"

# 统计元数据
echo ""
echo -e "${BLUE}📊 元数据统计:${NC}"
python3 <<EOF
import yaml

with open('$OUTPUT_METADATA') as f:
    data = yaml.safe_load(f)

total_pages = len(data['pages'])
total_elements = sum(p['stats']['total_elements'] for p in data['pages'].values())

print(f"  总页面数: {total_pages}")
print(f"  总元素数: {total_elements}")
print(f"  平均每页: {total_elements // total_pages if total_pages > 0 else 0} 个元素")
print()
print("各页面详情:")
for key, page in data['pages'].items():
    print(f"  {key}: {page['stats']['total_elements']} 个元素")
EOF

echo ""

# ============================================
# 步骤4: 生成选择器（未来功能）
# ============================================
echo -e "${BLUE}📋 步骤4: 智能选择器提取 (开发中)${NC}"
echo -e "${YELLOW}说明: 基于完整元数据，智能识别:${NC}"
echo -e "${YELLOW}  - 标题输入框 (placeholder+位置+type)${NC}"
echo -e "${YELLOW}  - 内容编辑器 (contenteditable+role+classes)${NC}"
echo -e "${YELLOW}  - 发布按钮 (text+visible+position)${NC}"
echo ""
echo -e "${YELLOW}当前请手动查看元数据文件进行提取${NC}"
echo ""

# ============================================
# 完成
# ============================================
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✅ 工具链执行完成${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "生成的文件:"
echo "  📄 $DISCOVERED_FILE - 发现的页面列表"
echo "  📦 $OUTPUT_METADATA - 完整元数据"
echo ""
echo "下一步:"
echo "  1. 查看元数据: cat $OUTPUT_METADATA"
echo "  2. 基于元数据编写智能选择器提取逻辑"
echo "  3. 更新 MCP 服务器配置"
echo ""
