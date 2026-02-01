# Config-Driven Polling Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 MCP 工具链内所有轮询/等待逻辑改为配置驱动，且缺失配置即报错并拒绝启动。

**Architecture:** 在 `config.yaml` 中新增 `polling.<module>` 模块化配置，所有等待/轮询均从配置读取超时与间隔；选择器仍由 `configs/selectors.yaml` 管理。启动阶段添加严格校验，缺一即失败。

**Tech Stack:** Go, Gin, YAML 配置, MCP SDK, Playwright/Rod 抽象浏览器接口。

### Task 1: 盘点 MCP 工具链内所有等待/轮询点并归类模块

**Files:**
- Modify: `docs/plans/2026-02-01-config-driven-polling.md`

**Step 1: 记录轮询/等待点清单**
- 用 `rg -n "time.Sleep|WaitVisible|WaitFor|WaitDOMStable|WaitIdle|WaitLoad" internal/ xiaohongshu/ cmd/` 列出等待点
- 归类到模块：`publish`、`draft`、`video`、`interaction`、`analytics`、`auth`（若存在）
- 在本计划文档末尾追加清单（便于执行时对照）

**Step 2: Commit 计划清单补充**
```bash
git add docs/plans/2026-02-01-config-driven-polling.md
git commit -m "docs: add polling inventory for mcp tools"
```

### Task 2: 定义配置结构与必填字段（TDD）

**Files:**
- Modify: `internal/infra/config/store.go`
- Modify: `internal/infra/config/store_test.go`
- Modify: `internal/infra/config/testdata/config.yaml`

**Step 1: 写失败测试（配置结构与必填校验）**
在 `internal/infra/config/store_test.go` 新增测试：
```go
func TestConfig_LoadRequiresPollingModules(t *testing.T) {
  // 缺 polling.publish.upload_timeout_ms 等字段时应报错
}
```

**Step 2: 运行测试确认失败**
```bash
go test ./internal/infra/config -run TestConfig_LoadRequiresPollingModules
```
期望：FAIL（缺少校验逻辑）。

**Step 3: 最小实现**
- 在 `internal/infra/config/store.go` 增加 `Polling` 结构：
  - `Publish`, `Draft`, `Video`, `Interaction`, `Analytics`, `Auth`（仅 MCP 工具链涉及的模块）
  - 每个模块至少包含 `timeout_ms`、`interval_ms`、`max_retries`（如适用）
- 为 `LoadFromFile` 增加严格校验函数（缺失即返回 error）

**Step 4: 运行测试确认通过**
```bash
go test ./internal/infra/config -run TestConfig_LoadRequiresPollingModules
```

**Step 5: Commit**
```bash
git add internal/infra/config/store.go internal/infra/config/store_test.go internal/infra/config/testdata/config.yaml
git commit -m "feat: add strict polling config schema"
```

### Task 3: 发布/草稿模块轮询配置化（TDD）

**Files:**
- Modify: `internal/infra/xhs/publish/gateway.go`
- Modify: `internal/interfaces/wiring/wiring.go`
- Modify: `config.yaml`
- Test: `internal/infra/xhs/publish/gateway_playwright_test.go`

**Step 1: 写失败测试（发布等待必须读取配置）**
在 `gateway_playwright_test.go` 增加测试，断言当 `polling.publish` 缺失时启动失败；以及等待函数使用配置超时/间隔。

**Step 2: 运行测试确认失败**
```bash
go test ./internal/infra/xhs/publish -run TestPublishPollingConfig
```

**Step 3: 最小实现**
- 在 `wiring.go` 注入 `Polling` 配置
- 在 `gateway.go` 将所有 `time.Sleep` 与硬编码超时/间隔替换为 `polling.publish.*`
- 上传等待、发布结果等待、页面稳定等待、提交前渲染等待均来自配置

**Step 4: 运行测试确认通过**
```bash
go test ./internal/infra/xhs/publish -run TestPublishPollingConfig
```

**Step 5: Commit**
```bash
git add internal/infra/xhs/publish/gateway.go internal/interfaces/wiring/wiring.go internal/infra/xhs/publish/gateway_playwright_test.go config.yaml
git commit -m "refactor: config-driven polling for publish/draft"
```

### Task 4: 视频发布/草稿模块轮询配置化（TDD）

**Files:**
- Modify: `internal/infra/xhs/publish/gateway.go`
- Test: `internal/infra/xhs/publish/gateway_playwright_test.go`

**Step 1: 写失败测试（视频模块读取 polling.video）**

**Step 2: 运行测试确认失败**
```bash
go test ./internal/infra/xhs/publish -run TestVideoPollingConfig
```

**Step 3: 最小实现**
- 替换视频上传等待、标题/内容等待、保存草稿等待为 `polling.video.*`

**Step 4: 运行测试确认通过**
```bash
go test ./internal/infra/xhs/publish -run TestVideoPollingConfig
```

**Step 5: Commit**
```bash
git add internal/infra/xhs/publish/gateway.go internal/infra/xhs/publish/gateway_playwright_test.go
git commit -m "refactor: config-driven polling for video"
```

### Task 5: 互动/数据等 MCP 工具轮询配置化（TDD）

**Files:**
- Modify: `service.go` / `internal/infra/xhs/*`（根据盘点清单）
- Test: 对应模块测试文件

**Step 1: 为每个模块写失败测试**
- 互动模块（like/follow/favorite）若存在轮询/等待，新增测试验证读取 `polling.interaction.*`
- 数据模块若存在等待逻辑，新增测试验证读取 `polling.analytics.*`

**Step 2: 运行测试确认失败**
```bash
go test ./internal/... -run TestInteractionPollingConfig
```

**Step 3: 最小实现**
- 替换硬编码等待/轮询为配置驱动

**Step 4: 运行测试确认通过**
```bash
go test ./internal/... -run TestInteractionPollingConfig
```

**Step 5: Commit**
```bash
git add internal/... config.yaml
git commit -m "refactor: config-driven polling for interaction/analytics"
```

### Task 6: 更新配置示例与文档（TDD）

**Files:**
- Modify: `config.yaml`
- Modify: `docs/*`（如有工具链文档）

**Step 1: 写失败测试（配置校验覆盖）**
- 若有配置校验测试集，增加覆盖所有模块字段

**Step 2: 更新配置示例**
- 在 `config.yaml` 添加 `polling.<module>` 示例，明确单位（ms）

**Step 3: 运行测试确认通过**
```bash
go test ./internal/infra/config -run TestConfig_LoadRequiresPollingModules
```

**Step 4: Commit**
```bash
git add config.yaml docs/*
git commit -m "docs: add polling config examples"
```

### Task 7: 全量测试

**Step 1: 运行相关测试**
```bash
go test ./internal/infra/config ./internal/infra/xhs/publish
```

**Step 2: Commit（若有额外改动）**
```bash
git add -A
git commit -m "test: verify polling config changes"
```

---

## MCP 轮询/等待点盘点（待补充）
- publish:
  - `internal/infra/xhs/publish/gateway.go`（发布流程、上传等待、完成等待）
  - `xiaohongshu/publish.go`（发布旧链路）
- draft:
  - `internal/infra/xhs/publish/gateway.go`（图文/视频草稿保存等待）
- video:
  - `internal/infra/xhs/publish/gateway.go`（视频发布/草稿流程）
  - `xiaohongshu/publish_video.go`
- interaction:
  - `xiaohongshu/like_favorite.go`
  - `xiaohongshu/follow.go`
  - `xiaohongshu/comment_like.go`
  - `xiaohongshu/comment_feed.go`
- analytics:
  - `xiaohongshu/data.go`
  - `xiaohongshu/feeds.go`
  - `xiaohongshu/search.go`
  - `xiaohongshu/user_profile.go`
  - `xiaohongshu/feed_detail.go`
- auth:
  - `xiaohongshu/login.go`
  - `xiaohongshu/navigate.go`
  - `xiaohongshu/delete.go`
  - `xiaohongshu/share.go`
