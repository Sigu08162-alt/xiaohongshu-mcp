#!/bin/sh
set -e

RELEASE_DIR="release"
ASSET="xiaohongshu-mcp-linux-amd64.tar.gz"
BIN_NAME="xiaohongshu-mcp-linux-amd64"

mkdir -p "$RELEASE_DIR"

# 下载最新 release 产物（gh 自带进度显示）
rm -f "$RELEASE_DIR/$ASSET"
GH_REPO="vmxmy/xiaohongshu-mcp"

START_TS=$(date +%s)
gh release download --repo "$GH_REPO" --pattern "$ASSET" --dir "$RELEASE_DIR"
END_TS=$(date +%s)

ELAPSED=$((END_TS - START_TS))
if [ "$ELAPSED" -lt 1 ]; then
  ELAPSED=1
fi

SIZE_BYTES=$(wc -c < "$RELEASE_DIR/$ASSET" | tr -d ' ')
SPEED_MBPS=$(awk "BEGIN {printf \"%.2f\", $SIZE_BYTES/1024/1024/$ELAPSED}")
echo "downloaded $ASSET: ${SIZE_BYTES} bytes in ${ELAPSED}s (${SPEED_MBPS} MB/s)"

# 解压并放到 bin/app
mkdir -p bin
rm -f bin/app

tar -xzf "$RELEASE_DIR/$ASSET" -C "$RELEASE_DIR"

if [ ! -f "$RELEASE_DIR/$BIN_NAME" ]; then
  echo "release binary not found: $RELEASE_DIR/$BIN_NAME"
  exit 1
fi

mv "$RELEASE_DIR/$BIN_NAME" bin/app
chmod +x bin/app

echo "downloaded and prepared: bin/app"
