#!/bin/sh
set -e

RELEASE_DIR="release"
ASSET="xiaohongshu-mcp-linux-amd64.tar.gz"
BIN_NAME="xiaohongshu-mcp-linux-amd64"

mkdir -p "$RELEASE_DIR"

# 下载最新 release 产物
rm -f "$RELEASE_DIR/$ASSET"
GH_REPO="vmxmy/xiaohongshu-mcp"

gh release download --repo "$GH_REPO" --pattern "$ASSET" --dir "$RELEASE_DIR"

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
