#!/bin/sh
set -e

RELEASE_DIR="release"
ASSET="xiaohongshu-mcp-linux-amd64.tar.gz"
BIN_NAME="xiaohongshu-mcp-linux-amd64"

mkdir -p "$RELEASE_DIR"

# 下载最新 release 产物（显示进度与速度）
rm -f "$RELEASE_DIR/$ASSET"
GH_REPO="vmxmy/xiaohongshu-mcp"

ASSET_URL=$(gh api "repos/$GH_REPO/releases/latest" --jq ".assets[] | select(.name==\"$ASSET\") | .url")
if [ -z "$ASSET_URL" ]; then
  echo "release asset not found: $ASSET"
  exit 1
fi

TOKEN=$(gh auth token)
if [ -z "$TOKEN" ]; then
  echo "gh auth token not found, run: gh auth login"
  exit 1
fi

curl -L --progress-bar \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/octet-stream" \
  "$ASSET_URL" -o "$RELEASE_DIR/$ASSET"

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
