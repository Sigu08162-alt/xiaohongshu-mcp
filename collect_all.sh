#!/bin/bash
set -e

echo "=========================================="
echo "小红书完整采集工作流"
echo "=========================================="
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 步骤1: 检查Cookie
echo -e "${BLUE}📋 步骤1: 检查登录状态${NC}"
echo ""

COOKIE_FILE=""
if [ -f "cookies.json" ]; then
    COOKIE_FILE="cookies.json"
elif [ -f "$HOME/.xiaohongshu/cookies.json" ]; then
    COOKIE_FILE="$HOME/.xiaohongshu/cookies.json"
fi

if [ -z "$COOKIE_FILE" ]; then
    echo -e "${YELLOW}⚠️  未找到Cookie文件，需要先登录${NC}"
    read -p "是否现在登录？(y/N): " -n 1 -r
    echo ""

    if [[ $REPLY =~ ^[Yy]$ ]]; then
        ./login.sh
    else
        echo -e "${RED}❌ 取消采集，请先运行 ./login.sh 登录${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}✅ 找到Cookie文件: $COOKIE_FILE${NC}"

    # 检查Cookie是否过期（简单检查文件修改时间）
    if [ "$(uname)" = "Darwin" ]; then
        # macOS
        COOKIE_AGE=$(( $(date +%s) - $(stat -f %m "$COOKIE_FILE") ))
    else
        # Linux
        COOKIE_AGE=$(( $(date +%s) - $(stat -c %Y "$COOKIE_FILE") ))
    fi

    COOKIE_AGE_HOURS=$(( COOKIE_AGE / 3600 ))

    if [ $COOKIE_AGE_HOURS -gt 24 ]; then
        echo -e "${YELLOW}⚠️  Cookie文件已超过${COOKIE_AGE_HOURS}小时，可能已过期${NC}"
        read -p "是否重新登录？(y/N): " -n 1 -r
        echo ""

        if [[ $REPLY =~ ^[Yy]$ ]]; then
            ./login.sh
        fi
    else
        echo -e "${GREEN}   Cookie文件: ${COOKIE_AGE_HOURS}小时前更新${NC}"
    fi
fi

echo ""

# 步骤2: 发现页面
echo -e "${BLUE}📋 步骤2: 发现页面链接${NC}"
echo ""

read -p "选择系统类型 [1=创作者(默认), 2=用户, 3=两者]: " SYSTEM_CHOICE
echo ""

case $SYSTEM_CHOICE in
    2)
        echo -e "${GREEN}🔍 发现用户系统页面...${NC}"
        ./bin/discover_pages --system user --no-interactive --wait 8 --output discovered_pages_user.yaml
        PAGE_FILES="discovered_pages_user.yaml"
        ;;
    3)
        echo -e "${GREEN}🔍 发现创作者系统页面...${NC}"
        ./bin/discover_pages --system creator --no-interactive --wait 8 --output discovered_pages_creator.yaml

        echo ""
        echo -e "${GREEN}🔍 发现用户系统页面...${NC}"
        ./bin/discover_pages --system user --no-interactive --wait 8 --output discovered_pages_user.yaml

        PAGE_FILES="discovered_pages_creator.yaml discovered_pages_user.yaml"
        ;;
    *)
        echo -e "${GREEN}🔍 发现创作者系统页面...${NC}"
        ./bin/discover_pages --system creator --no-interactive --wait 8 --output discovered_pages.yaml
        PAGE_FILES="discovered_pages.yaml"
        ;;
esac

echo ""

# 步骤3: 采集组件
echo -e "${BLUE}📋 步骤3: 采集页面组件${NC}"
echo ""

for PAGE_FILE in $PAGE_FILES; do
    if [ ! -f "$PAGE_FILE" ]; then
        echo -e "${RED}❌ 页面文件不存在: $PAGE_FILE${NC}"
        continue
    fi

    # 生成输出文件名
    OUTPUT_FILE="selectors_${PAGE_FILE}"

    echo -e "${GREEN}📄 采集页面: $PAGE_FILE${NC}"
    echo -e "${YELLOW}   输出文件: $OUTPUT_FILE${NC}"
    echo ""

    # 检查是否包含发布页面
    if grep -q "publish_publish" "$PAGE_FILE" 2>/dev/null; then
        echo -e "${YELLOW}⚠️  检测到发布页面(publish_publish)${NC}"
        echo -e "${YELLOW}   发布页面需要手动上传图片才能采集完整的输入框${NC}"
        echo ""

        read -p "采集发布页面时是否使用交互模式？(Y/n): " -n 1 -r
        echo ""

        if [[ $REPLY =~ ^[Nn]$ ]]; then
            # 非交互模式 - 跳过发布页面
            echo -e "${YELLOW}📌 将跳过发布页面，仅采集其他页面${NC}"
            ./bin/refresh_selectors --pages "$PAGE_FILE" --output "$OUTPUT_FILE" --no-interactive --wait 6
        else
            # 交互模式
            echo ""
            echo -e "${GREEN}========================================${NC}"
            echo -e "${GREEN}📸 发布页面交互采集指引${NC}"
            echo -e "${GREEN}========================================${NC}"
            echo "1. 浏览器将打开发布页面"
            echo "2. 请手动点击【上传图文】切换到图片上传"
            echo "3. 请手动上传一张图片（任意��片）"
            echo "4. 等待编辑页面完全加载（看到标题和内容输入框）"
            echo "5. 回到终端按 Enter 继续采集"
            echo ""
            read -p "按 Enter 开始..."

            ./bin/refresh_selectors --pages "$PAGE_FILE" --output "$OUTPUT_FILE" --wait 6
        fi
    else
        # 不包含发布页面，直接自动采集
        ./bin/refresh_selectors --pages "$PAGE_FILE" --output "$OUTPUT_FILE" --no-interactive --wait 6
    fi

    echo ""
done

# 步骤4: 显示结果
echo ""
echo -e "${BLUE}📋 步骤4: 采集结果汇总${NC}"
echo ""

echo -e "${GREEN}✅ 采集完成！${NC}"
echo ""
echo "生成的文件："
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

for PAGE_FILE in $PAGE_FILES; do
    OUTPUT_FILE="selectors_${PAGE_FILE}"

    if [ -f "$PAGE_FILE" ]; then
        SIZE=$(ls -lh "$PAGE_FILE" | awk '{print $5}')
        echo -e "📄 $PAGE_FILE (${SIZE})"
    fi

    if [ -f "$OUTPUT_FILE" ]; then
        SIZE=$(ls -lh "$OUTPUT_FILE" | awk '{print $5}')
        echo -e "📦 $OUTPUT_FILE (${SIZE})"

        # 统计组件数量
        if command -v python3 &> /dev/null; then
            python3 << EOF
import yaml
try:
    with open('$OUTPUT_FILE', 'r', encoding='utf-8') as f:
        data = yaml.safe_load(f)

    total_buttons = sum(len(p.get('buttons', [])) for p in data['pages'].values())
    total_inputs = sum(len(p.get('inputs', [])) for p in data['pages'].values())
    total_containers = sum(len(p.get('containers', [])) for p in data['pages'].values())

    print(f"   └─ {len(data['pages'])}个页面, {total_buttons}个按钮, {total_inputs}个输入框, {total_containers}个容器")
except Exception as e:
    print(f"   └─ 无法解析文件")
EOF
        fi
    fi

    echo ""
done

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${GREEN}🎉 全部完成！${NC}"
echo ""
echo "下一步："
echo "  - 查看采集结果: cat selectors_*.yaml"
echo "  - 加载到MCP服务器使用"
echo ""
