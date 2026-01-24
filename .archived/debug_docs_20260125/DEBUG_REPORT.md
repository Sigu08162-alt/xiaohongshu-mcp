# 小红书发布功能调试报告

## 测试日期
2026-01-25

## 测试目的
使用 Chrome DevTools MCP 调试小红书发布功能，确定正确的发布按钮元素和API调用流程。

## 测试结果：成功 ✓

### 1. 发布按钮元素信息

#### 按钮HTML结构
```html
<button
  data-v-e9a72776=""
  data-v-f643e8aa=""
  data-v-58cf5588-s=""
  type="button"
  class="d-button d-button-large --size-icon-large --size-text-h6 d-button-with-content --color-static bold --color-bg-fill --color-text-paragraph custom-button red publishBtn"
>发布</button>
```

#### 按钮CSS类名
- 主类: `d-button`
- 特征类: `publishBtn`, `red`, `custom-button`
- 完整类名: `d-button d-button-large --size-icon-large --size-text-h6 d-button-with-content --color-static bold --color-bg-fill --color-text-paragraph custom-button red publishBtn`

#### 按钮父元素
- 标签: `DIV`
- 类名: `submit`

#### 可用的选择器（已验证）
✓ `.publishBtn` - 推荐使用
✓ `button.publishBtn`
✓ `.submit button`
✓ `button.red.publishBtn`
✓ `.d-button.publishBtn`
✓ `text=发布` - 当前配置使用

#### 按钮位置和状态
- 位置: `{ top: 823, left: 248, width: 104, height: 36 }`
- 可见: `true`
- 在视口内: `true`
- 未被遮挡: `true`
- disabled状态: 点击前为`false`，点击后变为`true`

### 2. 点击行为

#### 使用Chrome DevTools MCP测试结果
- **方法**: 使用 `click` 工具点击按钮（uid=5_99）
- **结果**: 成功触发点击，按钮变为disabled状态
- **页面跳转**: URL从 `?source=official&target=image` 变为 `?source=official&published=true`

#### 点击后效果
1. 按钮立即变为 disabled 状态
2. 触发图片上传流程
3. 调用发布API
4. 页面URL添加 `published=true` 参数

### 3. API调用流程

#### 完整发布流程
1. **获取上传许可**
   ```
   GET https://creator.xiaohongshu.com/api/media/v1/upload/creator/permit?biz_name=spectrum&scene=image&file_count=1&version=1&source=web
   Status: 200
   ```

2. **上传图片**
   ```
   PUT https://ros-upload.xiaohongshu.com/spectrum/FOo3k9zAJdDrYgbZs1hKdEuuOLGoPD1zdUQaygCc7FW2qjM
   Status: 200
   ```

3. **发布笔记** (核心API)
   ```
   POST https://edith.xiaohongshu.com/web_api/sns/v2/note
   Status: 200
   ```

#### 发布API详细信息

**请求URL**: `https://edith.xiaohongshu.com/web_api/sns/v2/note`

**请求方法**: POST

**关键请求头**:
- `content-type: application/json`
- `x-s-common`: 签名参数（动态生成）
- `x-s`: 签名参数（动态生成）
- `x-t`: 时间戳
- `x-b3-traceid`: 追踪ID

**请求体结构**:
```json
{
  "common": {
    "type": "normal",
    "note_id": "",
    "source": "{\"type\":\"web\",\"ids\":\"\",\"extraInfo\":\"{\\\"subType\\\":\\\"official\\\",\\\"systemId\\\":\\\"web\\\"}\"}",
    "title": "测试标题",
    "desc": "这是测试内容",
    "ats": [],
    "hash_tag": [],
    "business_binds": "...",
    "privacy_info": {...},
    "goods_info": {},
    "biz_relations": [],
    "capa_trace_info": {...}
  },
  "image_info": {
    "images": [{
      "file_id": "spectrum/FOo3k9zAJdDrYgbZs1hKdEuuOLGoPD1zdUQaygCc7FW2qjM",
      "width": 800,
      "height": 600,
      "metadata": {"source": -1},
      "stickers": {"version": 2, "floating": []},
      "extra_info_json": "..."
    }]
  },
  "video_info": null
}
```

**响应体**:
```json
{
  "msg": "",
  "data": {
    "id": "69753fd5000000000a033ac0",
    "score": 10
  },
  "share_link": "https://www.xiaohongshu.com/discovery/item/69753fd5000000000a033ac0",
  "business_bind_results": [],
  "result": 0,
  "success": true
}
```

**响应状态**: 200 OK

### 4. 当前代码问题分析

#### 代码位置
`/Users/xumingyang/app/xiaohongshu-mcp/internal/infra/xhs/publish/gateway.go:92`

#### 当前实现
```go
// 使用强制点击
if err := page.ClickForce(submitSelector); err != nil {
    return fmt.Errorf("publish image submit(%s): %w", submitSelector, err)
}
```

#### 配置的选择器
`config.yaml` 中配置: `submit_button: "text=发布"`

#### 问题原因
通过Chrome DevTools MCP实测，使用普通的 `click` 方法即可成功触发发布。当前代码使用了 `ClickForce`（JavaScript强制点击），这可能导致某些Vue事件处理器被绕过。

### 5. 建议修复方案

#### 方案1: 使用普通Click（推荐）
```go
// 等待按钮可见
if err := page.WaitVisible(submitSelector); err != nil {
    return fmt.Errorf("等待发布按钮可见失败: %w", err)
}

// 使用普通点���
if err := page.Click(submitSelector); err != nil {
    return fmt.Errorf("点击发布按钮失败: %w", err)
}
```

#### 方案2: 更换选择器
如果 `text=发布` 不稳定，可以改用CSS选择器：
```yaml
submit_button: "button.publishBtn"
```

#### 方案3: 监听网络请求确认成功
添加网络请求监听，等待发布API返回成功：
```go
// 监听发布API响应
expectedURL := "https://edith.xiaohongshu.com/web_api/sns/v2/note"
// 等待响应成功
```

### 6. 成功验证方法

#### 验证发布成功的标志
1. **网络请求**: 发布API返回 `{"success": true, "data": {"id": "..."}}`
2. **URL变化**: 页面URL包含 `published=true` 参数
3. **按钮状态**: 发布按钮变为disabled状态
4. **返回笔记ID**: 响应中包含新创建的笔记ID

#### 建议的验证流程
```go
// 1. 点击发布按钮
page.Click(submitSelector)

// 2. 等待URL变化
page.WaitForURL("*published=true*")

// 3. 或等待特定的网络响应
// 监听 /web_api/sns/v2/note 的响应，检查 success: true
```

### 7. 测试截图

- 发布前: `debug_before_publish.png`
- 发布后: `debug_after_publish.png`

### 8. 总结

通过Chrome DevTools MCP的实际测试，证实了：

1. ✓ 当前的选择器 `text=发布` 可以找到正确的按钮
2. ✓ 使用普通的 `click` 方法可以成功触发发布
3. ✓ 发布流程完整且成功（获得了笔记ID: 69753fd5000000000a033ac0）
4. ✓ 页面成功跳转到发布完成状态

**问题所在**: 需要确认 Playwright 的实现是否正确处理了点击事件，以及是否等待了足够的时间来接收API响应。

**下一步行动**:
1. 修改代码使用普通 `Click` 替代 `ClickForce`
2. 添加网络请求监听来确认发布成功
3. 添加适当的等待时间确保API响应被处理
4. 考虑添加发布成功的明确验证（检查返回的笔记ID）
