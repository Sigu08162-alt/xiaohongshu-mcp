# Bug修复: "添加标记"弹窗误触发

## 问题描述

MCP工具在自动化发布过程中会误触发"添加标记"弹窗,导致发布流程中断。

## 根本原因

`gateway.go:198-202`中的代码会检查`content.MarkerTags`,如果有值就主动点击"添加标记"按钮:

```go
if len(content.MarkerTags) > 0 {
    if err := setMarkerTags(page, content.MarkerTags); err != nil {
        return fmt.Errorf("设置标记失败: %w", err)
    }
}
```

但是`MarkerTags`可能被意外设置为包含空字符串的数组,例如`[""]`,导致误触发。

## 修复方案

### 方案1: 添加严格校验(推荐)

在`internal/infra/xhs/publish/gateway.go`中修改:

```go
// 在文件开头添加辅助函数
func hasValidMarkers(markers []string) bool {
    for _, m := range markers {
        if strings.TrimSpace(m) != "" {
            return true
        }
    }
    return false
}

// 修改第198行的判断
if len(content.MarkerTags) > 0 && hasValidMarkers(content.MarkerTags) {
    if err := setMarkerTags(page, content.MarkerTags); err != nil {
        return fmt.Errorf("设置标记失败: %w", err)
    }
}
```

同样在`xiaohongshu/publish.go:357`也需要修改:

```go
// 设置标记（地点或用户）
if len(settings.MarkerTags) > 0 && hasValidMarkers(settings.MarkerTags) {
    slog.Info("开始设置标记", "markers", settings.MarkerTags)
    if err := setMarkerTags(page, settings.MarkerTags); err != nil {
        return errors.Wrap(err, "设置标记失败")
    }
    slog.Info("标记设置完成", "markers", settings.MarkerTags)
}
```

### 方案2: 在`setMarkerTags`函数内部校验

在`xiaohongshu/publish.go:1318`的`setMarkerTags`函数开头添加:

```go
func setMarkerTags(page browser.Page, markers []string) error {
    // 校验是否有有效的标记
    hasValid := false
    for _, m := range markers {
        if strings.TrimSpace(m) != "" {
            hasValid = true
            break
        }
    }
    if !hasValid {
        return nil // 没有有效标记,直接返回
    }

    // 原有逻辑...
    formItems, err := page.Elements(".d-new-form-item")
    // ...
}
```

同样在`internal/infra/xhs/publish/location_marker.go:164`也需要相同修改。

### 方案3: 在MCP工具调用侧过滤

确保MCP工具在调用publish接口时,不传递空的MarkerTags。

## 推荐方案

**方案1 + 方案2组合**:
- 方案1: 在调用侧做第一层防护
- 方案2: 在函数内部做第二层防护(防御性编程)

这样可以确保即使有遗漏的调用路径,也不会误触发弹窗。

## 测试验证

修复后需要测试:

1. **正常发布(无标记)**: 不应该触发"添加标记"弹窗
2. **发布with空MarkerTags**: `MarkerTags: []` 或 `MarkerTags: ["", "  "]` 不应触发弹窗
3. **发布with有效MarkerTags**: `MarkerTags: ["某地点", "某用户"]` 应正常弹出并设置标记

## 相关文件

- `internal/infra/xhs/publish/gateway.go:198-202`
- `internal/infra/xhs/publish/location_marker.go:164`
- `xiaohongshu/publish.go:357, 1318`
- `internal/domain/publish/content.go:14`

## 影响范围

- 图文发布: `PublishImage()`
- 草稿保存: `SaveImageDraft()`
- 所有使用MarkerTags的场景

---

**日期**: 2026-01-31
**严重级别**: 中等(影响用户体验,但不影响核心发布功能)
**优先级**: 高(应尽快修复)
