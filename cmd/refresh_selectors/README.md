# 小红书页面组件全量采集工具

使用 Playwright 有头浏览器自动采集小红书所有关键页面的组件信息，并保存到本地 YAML 文件。

## 功能特性

- ✅ **多页面采集**：一次运行采集图文发布、视频发布、创作者中心等多个页面
- ✅ **全量组件信息**：采集按钮、输入框、容器、链接等所有元素
- ✅ **YAML 输出**：结构化存储，易于阅读和版本对比
- ✅ **可选 JSON 输出**：同时支持 JSON 格式
- ✅ **有头浏览器**：可视化操作，方便登录和验证
- ✅ **单次登录**：登录后保持会话，采集多个页面无需重复登录
- ✅ **灵活配置**：可选择采集单个页面或全部页面

## 采集的页面

| 页面名称 | URL | 说明 |
|---------|-----|------|
| `publish_image` | `https://creator.xiaohongshu.com/publish/publish?source=official&target=image` | 图文发布页面 |
| `publish_video` | `https://creator.xiaohongshu.com/publish/publish?source=official&target=video` | 视频发布页面 |
| `creator_home` | `https://creator.xiaohongshu.com/new/home?source=official` | 创作者中心首页 |
| `content_list` | `https://creator.xiaohongshu.com/content` | 内容管理页面 |

## 使用方法

### 基础用法（采集所有页面）

```bash
# 启动工具，采集所有页面，保存到 YAML
./bin/refresh_selectors
```

**交互流程**：
1. 浏览器自动打开
2. 提示登录（如需要），按 Enter 继续
3. 自动依次访问所有页面并采集
4. 保存到 `selectors_all_pages.yaml`
5. 显示汇总信息
6. 按 Enter 关闭浏览器

### 采集单个页面

```bash
# 仅采集图文发布页面
./bin/refresh_selectors --page publish_image

# 仅采集视频发布页面
./bin/refresh_selectors --page publish_video

# 仅采集创作者中心首页
./bin/refresh_selectors --page creator_home

# 仅采集内容管理页面
./bin/refresh_selectors --page content_list
```

### 同时输出 JSON 和 YAML

```bash
# 同时保存为 YAML 和 JSON
./bin/refresh_selectors --json selectors_all_pages.json
```

### 自定义参数

```bash
# 自定义输出文件名
./bin/refresh_selectors --output my_selectors.yaml

# 调整等待时间（页面加载慢时增加）
./bin/refresh_selectors --wait 5

# 无头模式（后台运行）
./bin/refresh_selectors --headless

# 组合使用
./bin/refresh_selectors \
  --output selectors.yaml \
  --json selectors.json \
  --wait 5 \
  --page publish_image
```

## 参数说明

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--output` | `selectors_all_pages.yaml` | 输出 YAML 文件路径 |
| `--json` | ` ` (空) | 可选：同时输出 JSON 文件路径 |
| `--page` | ` ` (全部) | 仅采集单个页面 |
| `--wait` | `3` | 每个页面加载后等待秒数 |
| `--headless` | `false` | 是否使用无头模式 |

## 输出格式

### YAML 文件示例

```yaml
# 小红书页面组件全量快照
# 自动生成，请勿手动编辑
# 生成时间: 2026-01-30T12:00:00+08:00
# 版本: 1.0.0

version: 1.0.0
generated: "2026-01-30T12:00:00+08:00"
pages:
  publish_image:
    page_name: publish_image
    url: https://creator.xiaohongshu.com/publish/publish?source=official&target=image
    timestamp: "2026-01-30T12:00:00Z"
    buttons:
      - text: "发布"
        selector: button.d-button
        classes:
          - d-button
          - primary
        tag_name: button
        visible: true
        disabled: false
        type: button
      - text: "暂存离开"
        selector: button.d-button
        classes:
          - d-button
          - default
        tag_name: button
        visible: true
    inputs:
      - text: ""
        selector: .d-input
        classes:
          - d-input
        tag_name: input
        placeholder: "填写标题会有更多赞哦~"
        visible: true
        type: text
      - text: ""
        selector: div
        tag_name: div
        placeholder: "填写正文"
        visible: true
    containers:
      - selector: .upload-content
        classes:
          - upload-content
        tag_name: div
        visible: true
    links:
      - text: "创作灵感"
        selector: a.nav-item
        classes:
          - nav-item
        tag_name: a
        visible: true

  publish_video:
    # ... 视频发布页面组件

  creator_home:
    # ... 创作者中心首页组件

  content_list:
    # ... 内容管理页面组件
```

### 终端输出示例

```
=== 小红书页面组件全量采集工具 ===
输出YAML: selectors_all_pages.yaml
有头模式: true
等待时间: 3秒/页面

📋 将采集 4 个页面:
  - publish_image: 图文发布页面
  - publish_video: 视频发布页面
  - creator_home: 创作者中心首页
  - content_list: 内容管理页面

📦 启动浏览器...

🔐 请在浏览器中登录小红书（如需要）
   登录后将保持会话，后续页面无需重复登录

⏸️  按 Enter 继续...

[1/4] 📄 采集页面: 图文发布页面 (publish_image)
🌐 URL: https://creator.xiaohongshu.com/publish/publish?source=official&target=image
⏳ 等待 3 秒让页面加载完成...
✅ 采集完成: 按钮=15, 输入框=10, 容器=45, 链接=8

[2/4] 📄 采集页面: 视频发布页面 (publish_video)
🌐 URL: https://creator.xiaohongshu.com/publish/publish?source=official&target=video
⏳ 等待 3 秒让页面加载完成...
✅ 采集完成: 按钮=12, 输入框=8, 容器=40, 链接=6

[3/4] 📄 采集页面: 创作者中心首页 (creator_home)
🌐 URL: https://creator.xiaohongshu.com/new/home?source=official
⏳ 等待 3 秒让页面加载完成...
✅ 采集完成: 按钮=20, 输入框=3, 容器=60, 链接=15

[4/4] 📄 采集页面: 内容管理页面 (content_list)
🌐 URL: https://creator.xiaohongshu.com/content
⏳ 等待 3 秒让页面加载完成...
✅ 采集完成: 按钮=18, 输入框=5, 容器=55, 链接=12

💾 保存到 YAML: selectors_all_pages.yaml

============================================================
📊 采集汇总
============================================================
📅 生成时间: 2026-01-30T12:00:00+08:00
📄 页面数量: 4

🔹 publish_image:
   URL: https://creator.xiaohongshu.com/publish/publish?source=official&target=image
   按钮: 15 | 输入框: 10 | 容器: 45 | 链接: 8
   ✓ 发布按钮: button.d-button (classes: [d-button primary])
   ✓ 暂存按钮: button.d-button (classes: [d-button default])

🔹 publish_video:
   URL: https://creator.xiaohongshu.com/publish/publish?source=official&target=video
   按钮: 12 | 输入框: 8 | 容器: 40 | 链接: 6
   ✓ 发布按钮: button.d-button (classes: [d-button primary])

🔹 creator_home:
   URL: https://creator.xiaohongshu.com/new/home?source=official
   按钮: 20 | 输入框: 3 | 容器: 60 | 链接: 15

🔹 content_list:
   URL: https://creator.xiaohongshu.com/content
   按钮: 18 | 输入框: 5 | 容器: 55 | 链接: 12

============================================================

✅ 采集完成！

⏸️  按 Enter 关闭浏览器...
```

## 采集的信息

每个元素包含以下信息：

### 按钮 (Buttons)
- `text`: 按钮文本
- `selector`: CSS 选择器
- `classes`: CSS 类列表
- `id`: 元素 ID
- `tag_name`: HTML 标签名
- `visible`: 是否可见
- `disabled`: 是否禁用
- `type`: 按钮类型

### 输入框 (Inputs)
- `text`: 输入框当前值
- `selector`: CSS 选择器
- `classes`: CSS 类列表
- `id`: 元素 ID
- `tag_name`: HTML 标签名 (input/textarea/div)
- `placeholder`: 占位符文本
- `visible`: 是否可见
- `type`: 输入框类型

### 容器 (Containers)
- `selector`: CSS 选择器
- `classes`: CSS 类列表
- `id`: 元素 ID
- `tag_name`: HTML 标签名
- `visible`: 是否可见

### 链接 (Links)
- `text`: 链接文本
- `selector`: CSS 选择器
- `classes`: CSS 类列表
- `id`: 元素 ID
- `tag_name`: 固定为 'a'
- `visible`: 是否可见

## 使用场景

### 场景1：定期刷新选择器配置

```bash
# 每周运行一次，检查小红书页面更新
./bin/refresh_selectors --output weekly_snapshot_$(date +%Y%m%d).yaml

# 对比两周的快照
diff selectors_all_pages_20260123.yaml selectors_all_pages_20260130.yaml
```

### 场景2：验证配置准确性

```bash
# 采集当前页面状态
./bin/refresh_selectors --page publish_image

# 对比配置文件中的选择器
grep "submit_button" config.yaml
grep "发布" selectors_all_pages.yaml
```

### 场景3：调试选择器问题

```bash
# 采集发布页面详细信息
./bin/refresh_selectors --page publish_image --output debug_publish.yaml

# 查看所有发布相关按钮
yq '.pages.publish_image.buttons[] | select(.text | contains("发布"))' debug_publish.yaml
```

### 场景4：生成测试数据

```bash
# 采集所有页面，同时生成 JSON 供自动化测试使用
./bin/refresh_selectors --json test_selectors.json
```

## 与运行时组件采集的区别

| 功能 | 全量采集工具 | 运行时组件采集 |
|------|------------|--------------|
| **触发时机** | 手动运行 | 发布/保存失败时自动 |
| **采集范围** | 所有页面全量组件 | 当前页面错误场景 |
| **输出格式** | YAML + JSON | JSON |
| **浏览器** | 有头可视化 | 后台运行 |
| **用途** | 预防性维护、配置更新 | 调试定位问题 |
| **交互** | 需要用户登录 | 无用户交互 |

## 数据分析示例

### 使用 yq 查询 YAML

```bash
# 查看所有页面的发布按钮选择器
yq '.pages.*.buttons[] | select(.text | contains("发布")) | .selector' selectors_all_pages.yaml

# 统计每个页面的按钮数量
yq '.pages | to_entries | .[] | .key + ": " + (.value.buttons | length | tostring)' selectors_all_pages.yaml

# 查找所有 d-button 类的按钮
yq '.pages.*.buttons[] | select(.classes[] == "d-button") | .text' selectors_all_pages.yaml

# 查看图文发布页面的所有输入框
yq '.pages.publish_image.inputs[]' selectors_all_pages.yaml
```

### 使用 jq 查询 JSON

```bash
# 查看所有页面 URL
jq '.pages | to_entries[] | {page: .key, url: .value.url}' selectors_all_pages.json

# 统计总组件数
jq '.pages | to_entries[] | {
  page: .key,
  total: (.value.buttons | length) + (.value.inputs | length) + (.value.containers | length)
}' selectors_all_pages.json

# 查找所有可见的按钮
jq '.pages.publish_image.buttons[] | select(.visible == true) | .text' selectors_all_pages.json
```

## 故障排查

### 问题1：浏览器未启动

```bash
# 安装 Playwright 驱动
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium
```

### 问题2：页面加载不完整

```bash
# 增加等待时间
./bin/refresh_selectors --wait 10
```

### 问题3：某个页面采集失败

```bash
# 仅采集该页面，增加等待时间
./bin/refresh_selectors --page publish_image --wait 10
```

### 问题4：需要重新登录

```bash
# 使用有头模式，手动登录
./bin/refresh_selectors  # 默认就是有头模式
```

## 最佳实践

1. **定期运行**：每周或小红书更新后运行一次
2. **版本控制**：将快照文件纳入 git，跟踪选择器变化历史
3. **对比分析**：使用 `diff` 或 `yq`/`jq` 对比快照，发现变化
4. **单页调试**：遇到问题时使用 `--page` 单独采集问题页面
5. **增加等待**：页面加载慢时使用 `--wait` 增加等待时间

## 技术细节

- **语言**：Go
- **浏览器驱动**：Playwright (Chromium)
- **解析**：原生 JavaScript `querySelectorAll`
- **输出格式**：YAML (主) + JSON (可选)
- **并发**：串行访问页面（复用同一浏览器会话）

## 扩展页面

如需采集其他页面，修改 `targetPages` 变量：

```go
var targetPages = []PageDefinition{
    {
        Name: "my_custom_page",
        URL:  "https://creator.xiaohongshu.com/custom",
        Desc: "自定义页面描述",
    },
    // ... 其他页面
}
```

## 输出文件说明

- **selectors_all_pages.yaml**: 主输出文件，包含所有页面的完整组件信息
- **selectors_all_pages.json**: 可选输出，JSON 格式，便于程序化处理
- **文件结构**: 版本信息 + 生成时间 + 多个页面快照

每个页面快照包含：
- 页面名称、URL、时间戳
- 按钮列表（所有 `<button>` 元素）
- 输入框列表（`<input>`, `<textarea>`, `[contenteditable]`）
- 容器列表（`<div>`, `<section>`, `<main>` with class）
- 链接列表（`<a href>` 有文本的链接）

## Cookie 自动加载

工具会自动查找并加载 cookie 文件，无需每次手动登录！

### Cookie 文件查找顺序

1. `cookies.json` (当前目录)
2. `~/.xiaohongshu/cookies.json` (用户目录)
3. `./xiaohongshu_cookies.json` (备用名称)

### 使用 login.sh 生成 Cookie

```bash
# 1. 运行登录脚本
./login.sh

# 2. 浏览器打开，扫码登录

# 3. Cookie 自动保存到 cookies.json

# 4. 直接运行刷新工具（自动使用 cookie）
make refresh-selectors-run
```

### 手动指定 Cookie 文件

```bash
# 使用自定义 cookie 文件
./bin/refresh_selectors --cookies /path/to/cookies.json
```

### Cookie 过期处理

如果运行时提示需要登录：

```bash
# 重新生成 cookie
./login.sh

# 再次运行刷新工具
make refresh-selectors-run
```

### 优势

- ✅ **无需手动登录**：自动加载已保存的 cookie
- ✅ **多次复用**：一次登录，多次使用
- ✅ **自动查找**：智能查找多个位置
- ✅ **灵活指定**：支持自定义 cookie 路径
