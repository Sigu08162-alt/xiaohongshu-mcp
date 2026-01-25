# 发布图文 location 与 marker_tags 实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 `POST /publish` 增加 `location` 与 `marker_tags` 字段，并在发布流程中应用地点与标记。

**Architecture:** 请求字段进入 `PublishRequest`，透传到领域模型与发布网关；网关在发布时设置地点与标记。仅扩展图文发布路径，视频发布不变。

**Tech Stack:** Go 1.24、Playwright 浏览器引擎、swaggo/swag 文档生成。

### Task 1: 修复测试桩以支持新测试

**Files:**
- Modify: `internal/app/testkit/fakes.go`
- Modify: `internal/infra/xhs/publish/gateway_playwright_test.go`
- Test: `internal/app/testkit/fakes_test.go`

**Step 1: 补齐 FakePublishGateway 接口方法**

```go
// internal/app/testkit/fakes.go
func (f *FakePublishGateway) SaveImageDraft(ctx context.Context, content publish.ImageContent) error {
	f.ImageCalls++
	f.LastImage = content
	return f.Err
}

func (f *FakePublishGateway) SaveVideoDraft(ctx context.Context, content publish.VideoContent) error {
	f.VideoCalls++
	f.LastVideo = content
	return f.Err
}
```

**Step 2: 为 fakePage/fakeElement 增加缺失方法（最小实现）**

```go
// internal/infra/xhs/publish/gateway_playwright_test.go
// 仅示例，补齐 browser.Page 与 browser.Element 接口方法，返回零值即可。
```

**Step 3: 运行接口实现测试**

Run: `go test ./internal/app/testkit -run TestFakesImplementPorts -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/app/testkit/fakes.go internal/infra/xhs/publish/gateway_playwright_test.go
git commit -m "test: complete publish test fakes"
```

### Task 2: 为结构体新增字段（TDD）

**Files:**
- Modify: `service.go`
- Modify: `internal/domain/publish/content.go`
- Create: `internal/domain/publish/content_test.go`
- Modify: `service_test.go`

**Step 1: 写失败测试（结构体包含字段）**

```go
// internal/domain/publish/content_test.go
func TestImageContent_HasLocationAndMarkerTagsFields(t *testing.T) {
	typ := reflect.TypeOf(ImageContent{})
	if _, ok := typ.FieldByName("Location"); !ok {
		t.Fatalf("missing Location field")
	}
	if _, ok := typ.FieldByName("MarkerTags"); !ok {
		t.Fatalf("missing MarkerTags field")
	}
}
```

```go
// service_test.go
func TestPublishRequest_HasLocationAndMarkerTagsFields(t *testing.T) {
	typ := reflect.TypeOf(PublishRequest{})
	if _, ok := typ.FieldByName("Location"); !ok {
		t.Fatalf("missing Location field")
	}
	if _, ok := typ.FieldByName("MarkerTags"); !ok {
		t.Fatalf("missing MarkerTags field")
	}
}
```

**Step 2: 运行测试并确认失败**

Run: `go test ./internal/domain/publish -run TestImageContent_HasLocationAndMarkerTagsFields -v`
Expected: FAIL with "missing Location field"

Run: `go test . -run TestPublishRequest_HasLocationAndMarkerTagsFields -v`
Expected: FAIL with "missing Location field"

**Step 3: 添加字段实现**

```go
// internal/domain/publish/content.go
Location   string
MarkerTags []string
```

```go
// service.go
Location   string   `json:"location,omitempty"`
MarkerTags []string `json:"marker_tags,omitempty"`
```

**Step 4: 运行测试并确认通过**

Run: `go test ./internal/domain/publish -run TestImageContent_HasLocationAndMarkerTagsFields -v`
Expected: PASS

Run: `go test . -run TestPublishRequest_HasLocationAndMarkerTagsFields -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/domain/publish/content.go internal/domain/publish/content_test.go service.go service_test.go
git commit -m "feat: add publish location and marker fields"
```

### Task 3: 请求字段映射到发布用例（TDD）

**Files:**
- Modify: `service.go`
- Modify: `service_test.go`

**Step 1: 写失败测试（请求字段透传）**

```go
// service_test.go
func TestPublishContent_MapsLocationAndMarkerTags(t *testing.T) {
	gw := &testkit.FakePublishGateway{}
	uc := apppublish.Usecase{Gateway: gw, Limits: domainpublish.Limits{MaxTags: 10, MinImages: 1, MaxImages: 9}}
	service := NewXiaohongshuServiceWithUsecase(&uc)
	req := &PublishRequest{
		Title:      "t",
		Content:    "c",
		Images:     []string{"/tmp/placeholder.jpg"},
		Location:   "深圳湾公园",
		MarkerTags: []string{"深圳湾公园", "张三"},
	}
	if _, err := service.PublishContent(context.Background(), req); err != nil {
		t.Fatalf("publish err: %v", err)
	}
	if gw.LastImage.Location != "深圳湾公园" {
		t.Fatalf("unexpected location: %s", gw.LastImage.Location)
	}
	if len(gw.LastImage.MarkerTags) != 2 {
		t.Fatalf("unexpected marker tags: %v", gw.LastImage.MarkerTags)
	}
}
```

**Step 2: 运行测试并确认失败**

Run: `go test . -run TestPublishContent_MapsLocationAndMarkerTags -v`
Expected: FAIL with empty location/marker tags

**Step 3: 实现字段映射**

```go
// service.go
content := xiaohongshu.PublishImageContent{
	Title:        req.Title,
	Content:      req.Content,
	Tags:         req.Tags,
	ImagePaths:   imagePaths,
	Location:     req.Location,
	MarkerTags:   req.MarkerTags,
	ScheduleTime: scheduleTime,
}

// usecase 透传
if err := s.publishUsecase.PublishImage(ctx, domainpublish.ImageContent{
	Title:        content.Title,
	Content:      content.Content,
	Tags:         content.Tags,
	ImagePaths:   content.ImagePaths,
	Location:     content.Location,
	MarkerTags:   content.MarkerTags,
	ScheduleTime: content.ScheduleTime,
}); err != nil {
	...
}
```

**Step 4: 运行测试并确认通过**

Run: `go test . -run TestPublishContent_MapsLocationAndMarkerTags -v`
Expected: PASS

**Step 5: Commit**

```bash
git add service.go service_test.go
git commit -m "feat: map publish location and marker tags"
```

### Task 4: 网关应用地点与标记（TDD）

**Files:**
- Modify: `internal/infra/xhs/publish/gateway.go`
- Modify: `internal/infra/xhs/publish/gateway_playwright_test.go`
- Create: `internal/infra/xhs/publish/location_marker.go`

**Step 1: 写失败测试（触发地点/标记逻辑）**

```go
// internal/infra/xhs/publish/gateway_playwright_test.go
func TestGateway_PublishImage_SetsLocation(t *testing.T) {
	engine := &fakeEngine{page: &fakePage{DropdownText: "深圳湾公园", DropdownVisible: true}}
	cfg := Config{
		PublishImageURL: "https://example.com",
		PublishVideoURL: "https://example.com",
		Selectors: map[string]string{
			"upload_input": "input[type=file]",
			"title_input":  "input[name=title]",
			"content":      "textarea[name=content]",
			"submit":       "button[type=submit]",
		},
	}
	gw, err := NewGateway(cfg, engine)
	if err != nil {
		t.Fatalf("new gateway err: %v", err)
	}
	err = gw.PublishImage(context.Background(), publish.ImageContent{
		Title:      "t",
		Content:    "c",
		ImagePaths: []string{"1.jpg"},
		Location:   "深圳湾公园",
	})
	if err != nil {
		t.Fatalf("publish err: %v", err)
	}
	if !engine.page.HasElementCall(".address-box input.d-text") {
		t.Fatalf("expected location input call")
	}
}

func TestGateway_PublishImage_SetsMarkerTags(t *testing.T) {
	engine := &fakeEngine{page: &fakePage{EvalResult: "selected"}}
	cfg := Config{ /* 同上 */ }
	gw, _ := NewGateway(cfg, engine)
	err := gw.PublishImage(context.Background(), publish.ImageContent{
		Title:      "t",
		Content:    "c",
		ImagePaths: []string{"1.jpg"},
		MarkerTags: []string{"深圳湾公园"},
	})
	if err != nil {
		t.Fatalf("publish err: %v", err)
	}
	if !engine.page.HasWaitForFunctionCall() {
		t.Fatalf("expected marker dialog wait")
	}
}
```

**Step 2: 运行测试并确认失败**

Run: `go test ./internal/infra/xhs/publish -run TestGateway_PublishImage_SetsLocation -v`
Expected: FAIL with missing location behavior

Run: `go test ./internal/infra/xhs/publish -run TestGateway_PublishImage_SetsMarkerTags -v`
Expected: FAIL with missing marker behavior

**Step 3: 实现地点与标记逻辑（复用现有选择器）**

```go
// internal/infra/xhs/publish/location_marker.go
func setLocation(page browser.Page, location string) error { /* 参考 xiaohongshu/publish.go */ }
func setMarkerTags(page browser.Page, markers []string) error { /* 参考 xiaohongshu/publish.go */ }
```

```go
// internal/infra/xhs/publish/gateway.go
if content.Location != "" {
	if err := setLocation(page, content.Location); err != nil {
		return fmt.Errorf("set location: %w", err)
	}
}
if len(content.MarkerTags) > 0 {
	if err := setMarkerTags(page, content.MarkerTags); err != nil {
		return fmt.Errorf("set markers: %w", err)
	}
}
```

**Step 4: 运行测试并确认通过**

Run: `go test ./internal/infra/xhs/publish -run TestGateway_PublishImage_SetsLocation -v`
Expected: PASS

Run: `go test ./internal/infra/xhs/publish -run TestGateway_PublishImage_SetsMarkerTags -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/infra/xhs/publish/gateway.go internal/infra/xhs/publish/location_marker.go internal/infra/xhs/publish/gateway_playwright_test.go
git commit -m "feat: apply location and marker tags in publish gateway"
```

### Task 5: 更新 Swagger 与 API 文档

**Files:**
- Modify: `docs/API.md`
- Generate: `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`

**Step 1: 更新 API 文档示例与参数说明**

```md
// docs/API.md
- `location` (string, optional): 地点名称，支持城市/商圈/POI
- `marker_tags` (array, optional): 标记的地点或用户昵称列表
```

**Step 2: 生成 Swagger 文档**

Run: `~/go/bin/swag init --parseDependency --parseInternal`
Expected: 生成/更新 `docs/docs.go`、`docs/swagger.json`、`docs/swagger.yaml`

**Step 3: Commit**

```bash
git add docs/API.md docs/docs.go docs/swagger.json docs/swagger.yaml
git commit -m "docs: update publish swagger fields"
```

### Task 6: 格式化与回归验证

**Files:**
- Modify: `*.go`

**Step 1: gofmt**

Run: `gofmt -w service.go internal/domain/publish/content.go internal/app/testkit/fakes.go internal/infra/xhs/publish/gateway.go internal/infra/xhs/publish/location_marker.go internal/infra/xhs/publish/gateway_playwright_test.go service_test.go internal/domain/publish/content_test.go`
Expected: 无输出

**Step 2: 运行关键测试**

Run: `go test . -run TestPublishRequest_HasLocationAndMarkerTagsFields -v`
Expected: PASS

Run: `go test . -run TestPublishContent_MapsLocationAndMarkerTags -v`
Expected: PASS

Run: `go test ./internal/domain/publish -run TestImageContent_HasLocationAndMarkerTagsFields -v`
Expected: PASS

Run: `go test ./internal/infra/xhs/publish -run TestGateway_PublishImage_SetsLocation -v`
Expected: PASS

Run: `go test ./internal/infra/xhs/publish -run TestGateway_PublishImage_SetsMarkerTags -v`
Expected: PASS

**Step 3: Final Commit (if needed)**

```bash
git status -sb
# 如有遗漏，按需补提一个整理提交
```
