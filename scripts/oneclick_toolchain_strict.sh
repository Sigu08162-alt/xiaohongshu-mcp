#!/bin/bash
set -euo pipefail

# 小红书工具链一键脚本（强约束：每层显式输入/输出）

BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

print_header() {
  echo -e "${BLUE}==========================================${NC}"
  echo -e "${BLUE}小红书工具链一键脚本（强约束）${NC}"
  echo -e "${BLUE}每层显式输入/输出，不允许默认回退${NC}"
  echo -e "${BLUE}==========================================${NC}"
  echo ""
}

require_file() {
  local path="$1"
  if [ ! -f "$path" ]; then
    echo -e "${RED}❌ 未找到文件: $path${NC}"
    exit 1
  fi
}

print_header

# Step 0: 登录检查
COOKIE_FILE=""
if [ -f "cookies.json" ]; then
  COOKIE_FILE="cookies.json"
elif [ -f "$HOME/.xiaohongshu/cookies.json" ]; then
  COOKIE_FILE="$HOME/.xiaohongshu/cookies.json"
fi

if [ -z "$COOKIE_FILE" ]; then
  echo -e "${YELLOW}⚠️  未找到Cookie文件，建议先登录${NC}"
  read -p "是否现在登录？(y/N): " -n 1 -r
  echo ""
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    ./login.sh
  else
    echo -e "${RED}❌ 未登录可能导致页面不完整${NC}"
  fi
else
  echo -e "${GREEN}✅ Cookie文件: $COOKIE_FILE${NC}"
fi

echo ""

# Step 1: 发现页面
read -p "选择系统 [1=创作者(默认), 2=用户]: " SYSTEM_CHOICE
SYSTEM_CHOICE=${SYSTEM_CHOICE:-1}

DEFAULT_DISCOVERED_FILE="discovered_pages_creator.yaml"
read -p "请输入 discover 输出文件名 [默认: ${DEFAULT_DISCOVERED_FILE}]: " DISCOVERED_FILE
if [ -z "$DISCOVERED_FILE" ]; then
  DISCOVERED_FILE="$DEFAULT_DISCOVERED_FILE"
fi

if [ "$SYSTEM_CHOICE" = "2" ]; then
  SYSTEM_TYPE="user"
else
  SYSTEM_TYPE="creator"
fi

echo -e "${BLUE}📋 步骤1: 发现页面${NC}"
./bin/discover_pages --system "$SYSTEM_TYPE" --no-interactive --wait 8 --output "$DISCOVERED_FILE"
require_file "$DISCOVERED_FILE"

echo ""

# Step 2: 采集选择器
DEFAULT_SELECTORS_FILE="selectors_discovered_pages_creator.yaml"
read -p "请输入 selectors 输出文件名 [默认: ${DEFAULT_SELECTORS_FILE}]: " SELECTORS_FILE
if [ -z "$SELECTORS_FILE" ]; then
  SELECTORS_FILE="$DEFAULT_SELECTORS_FILE"
fi

read -p "是否使用交互模式采集（可手动上传图片）？(Y/n): " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Nn]$ ]]; then
  INTERACTIVE_FLAG="--no-interactive"
else
  INTERACTIVE_FLAG=""
fi

echo -e "${BLUE}📋 步骤2: 采集选择器${NC}"
./bin/refresh_selectors --pages "$DISCOVERED_FILE" --output "$SELECTORS_FILE" $INTERACTIVE_FLAG --wait 6
require_file "$SELECTORS_FILE"

echo ""

# Step 3: 采集元数据（可选）
read -p "是否采集完整元数据？(y/N): " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
  DEFAULT_METADATA_FILE="metadata_discovered_pages_creator.yaml"
  read -p "请输入 metadata 输出文件名 [默认: ${DEFAULT_METADATA_FILE}]: " METADATA_FILE
  if [ -z "$METADATA_FILE" ]; then
    METADATA_FILE="$DEFAULT_METADATA_FILE"
  fi

  echo -e "${BLUE}📋 步骤3: 采集元数据${NC}"
  if [ -z "$INTERACTIVE_FLAG" ]; then
    ./bin/collect_metadata --input "$DISCOVERED_FILE" --output "$METADATA_FILE" --wait 5
  else
    ./bin/collect_metadata --input "$DISCOVERED_FILE" --output "$METADATA_FILE" --no-interactive --wait 5
  fi
  require_file "$METADATA_FILE"
fi

echo ""

# Summary
echo -e "${GREEN}✅ 完成${NC}"
echo "生成的文件:"
echo "  - $DISCOVERED_FILE"
echo "  - $SELECTORS_FILE"
if [ -n "${METADATA_FILE:-}" ]; then
  echo "  - $METADATA_FILE"
fi
