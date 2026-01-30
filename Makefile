# 小红书MCP工具 Makefile

VERSION ?= $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT := $(shell git rev-parse --short HEAD)

LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.Commit=$(COMMIT)"

# 默认目标
.PHONY: all
all: build

# 构建当前平台
.PHONY: build
build:
	go build $(LDFLAGS) -o bin/xiaohongshu-mcp .

# 构建所有平台
.PHONY: build-all
build-all: build-linux build-darwin build-windows

# Linux (amd64 & arm64)
.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/xiaohongshu-mcp-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/xiaohongshu-mcp-linux-arm64 .

# macOS (amd64 & arm64)
.PHONY: build-darwin
build-darwin:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/xiaohongshu-mcp-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/xiaohongshu-mcp-darwin-arm64 .

# Windows (amd64 & arm64)
.PHONY: build-windows
build-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/xiaohongshu-mcp-windows-amd64.exe .
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o dist/xiaohongshu-mcp-windows-arm64.exe .

# 清理
.PHONY: clean
clean:
	rm -rf bin/ dist/

# 测试
.PHONY: test
test:
	go test -v ./...

# 格式化代码
.PHONY: fmt
fmt:
	go fmt ./...
	gofmt -w .

# 安装
.PHONY: install
install:
	go install $(LDFLAGS) .

# 开发模式运行
.PHONY: run
run:
	go run . --headless=false

# 生产模式运行
.PHONY: run-prod
run-prod:
	go run . --headless=true

# 帮助
.PHONY: help
help:
	@echo "可用目标:"
	@echo "  make build         - 构建当前平台"
	@echo "  make build-all     - 构建所有平台"
	@echo "  make build-linux   - 构建Linux平台"
	@echo "  make build-darwin  - 构建macOS平台"
	@echo "  make build-windows - 构建Windows平台"
	@echo "  make clean         - 清理构建文件"
	@echo "  make test          - 运行测试"
	@echo "  make fmt           - 格式化代码"
	@echo "  make install       - 安装到GOPATH"
	@echo "  make run           - 开发模式运行"
	@echo "  make run-prod      - 生产模式运行"

# 刷新选择器工具
.PHONY: refresh-selectors
refresh-selectors:
	@echo "🔄 构建选择器刷新工具..."
	@go build -o bin/refresh_selectors ./cmd/refresh_selectors
	@echo "✅ 工具已构建到 bin/refresh_selectors"
	@echo ""
	@echo "使用方法:"
	@echo "  make refresh-selectors-run    # 采集所有页面"
	@echo "  make refresh-selectors-image  # 仅采集图文发布页面"
	@echo "  或直接运行: ./cmd/refresh_selectors/run.sh"

.PHONY: refresh-selectors-run
refresh-selectors-run: refresh-selectors
	@./bin/refresh_selectors

.PHONY: refresh-selectors-image
refresh-selectors-image: refresh-selectors
	@./bin/refresh_selectors --page publish_image --output selectors_publish_image.yaml

.PHONY: refresh-selectors-video
refresh-selectors-video: refresh-selectors
	@./bin/refresh_selectors --page publish_video --output selectors_publish_video.yaml

.PHONY: refresh-selectors-all-json
refresh-selectors-all-json: refresh-selectors
	@./bin/refresh_selectors --json selectors_all_pages.json

# 页面发现工具
.PHONY: discover-pages
discover-pages:
	@echo "🔍 构建页面发现工具..."
	@go build -o bin/discover_pages ./cmd/discover_pages
	@echo "✅ 工具已构建到 bin/discover_pages"

.PHONY: discover-pages-run
discover-pages-run: discover-pages
	@./bin/discover_pages

# 使用动态发现的页面列表采集组件
.PHONY: refresh-selectors-from-discovered
refresh-selectors-from-discovered: refresh-selectors
	@if [ ! -f discovered_pages.yaml ]; then \
		echo "❌ 未找到 discovered_pages.yaml，请先运行: make discover-pages-run"; \
		exit 1; \
	fi
	@./bin/refresh_selectors --pages discovered_pages.yaml
