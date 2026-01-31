# 小红书MCP工具链 v2.0

## 设计理念

### 问题：旧版工具链的缺陷
1. **硬编码选择器映射规则** - 在 `wiring_bootstrap.go` 中硬编码了"标题用placeholder判断"等规则
2. **采集不完整** - 只采集了部分属性（text, classes, placeholder等），缺少很多关键信息
3. **不够通用** - 小红书页面改版后需要修改代码

### 解决方案：无硬编码工具链
1. **完整元数据采集** - 采集所有元素的所有属性（50+字段）
2. **智能选择器提取** - 基于完整元数据，使用启发式规则或AI识别关键元素
3. **自适应** - 页面改版只需重新采集，无需改代码

---

## 工具链架构

```
┌─────────────────────────────────────────────────────────────┐
│                     工具链 v2.0                              │
└─────────────────────────────────────────────────────────────┘

阶段1: 页面发现
├─ 工具: bin/discover_pages
├─ 输入: 起始URL（创作者中心/用户首页）
├─ 输出: discovered_pages.yaml
└─ 功能: 自动发现所有相关页面链接

阶段2: 元数据采集（核心）
├─ 工具: bin/collect_metadata ⭐️ 新工具
├─ 输入: discovered_pages.yaml
├─ 输出: metadata_all_pages.yaml
├─ 功能: 无差别采集所有元素的完整元数据
└─ 特点:
    ├─ 50+ 属性字段
    ├─ 无硬编码规则
    ├─ 支持交互式操作（上传图片等）
    └─ 自动去重

阶段3: 选择器提取（规划中）
├─ 工具: bin/extract_selectors (待开发)
├─ 输入: metadata_all_pages.yaml
├─ 输出: selectors_smart.yaml
└─ 功能: 基于元数据智能识别关键元素
    例如:
    ├─ 标题输入框: input[type=text] + placeholder="标题" + 位置上部 + visible
    ├─ 内容编辑器: contenteditable=true + role=textbox + 富文本class
    └─ 发布按钮: button + text="发布" + !disabled + 位置���下
```

---

## 元数据结构

### ElementMetadata（50+字段）

```yaml
- tag_name: input              # 标签名
  id: title-input              # ID
  classes: [d-input, large]    # 类名列表
  name: title                  # name属性

  css_selector: "input.d-input" # CSS选择器
  xpath: /html/body/div[1]/...  # XPath（可选）

  text: ""                     # 文本内容
  value: ""                    # 当前值
  inner_html: "<span>...</span>" # 内部HTML（截取500字符）

  # 表单相关
  type: text                   # 类型
  placeholder: "填写标题会有更多赞哦～" # placeholder
  disabled: false              # 是否禁用
  readonly: false              # 是否只读
  required: false              # 是否必填
  checked: false               # 是否选中
  selected: false              # 是否被选择

  # ARIA属性
  role: textbox                # ARIA角色
  aria_label: ""               # ARIA标签
  aria_labelledby: ""          # 标签关联
  aria_describedby: ""         # 描述关联
  aria_attrs:                  # 其他ARIA属性
    aria-invalid: "false"

  # Data属性
  data_attrs:                  # data-*属性
    data-testid: "title-input"
    data-component: "TitleInput"

  # 位置和尺寸
  position:
    x: 120.5                   # X坐标
    y: 200.0                   # Y坐标
    width: 600.0               # 宽度
    height: 40.0               # 高度

  # 可见性
  visible: true                # 是否在DOM中可见
  display: "block"             # display样式
  visibility: "visible"        # visibility样式
  opacity: "1"                 # 不透明度
  contenteditable: "false"     # 是否可编辑

  # 层级关系
  parent_tag: div              # 父元素标签
  parent_classes: [form-group] # 父元素类名
  children_count: 0            # 子元素数量

  # 其他属性
  other_attrs:                 # 其他非标准属性
    autocomplete: "off"
    spellcheck: "false"
```

---

## 使用方法

### 方式1: 一键执行完整工具链

```bash
./toolchain_v2.sh
```

按提示操作：
1. 选择系统（创作者/用户/两者）
2. 选择交互模式（推荐选择Y，可手动上传图片）
3. 浏览器打开后，手动操作页面（如上传图片）
4. 操作完成后回到终端按Enter
5. 等待采集完成

### 方式2: 单独运行各阶段

#### 阶段1: 发现页面

```bash
# 发现创作者系统页面
./bin/discover_pages \
  --system creator \
  --no-interactive \
  --wait 8 \
  --output discovered_pages_creator.yaml

# 发现用户系统页面
./bin/discover_pages \
  --system user \
  --no-interactive \
  --wait 8 \
  --output discovered_pages_user.yaml
```

#### 阶段2: 采集元数据

```bash
# 交互模式（推荐）
./bin/collect_metadata \
  --input discovered_pages_creator.yaml \
  --output metadata_creator.yaml \
  --wait 5

# 非交互模式
./bin/collect_metadata \
  --input discovered_pages_creator.yaml \
  --output metadata_creator.yaml \
  --no-interactive \
  --wait 5

# 仅采集单个页面
./bin/collect_metadata \
  --input discovered_pages_creator.yaml \
  --output metadata_publish.yaml \
  --page publish_publish \
  --wait 5
```

#### 阶段3: 查看元数据

```bash
# 查看YAML
cat metadata_creator.yaml

# 统计
python3 <<EOF
import yaml
with open('metadata_creator.yaml') as f:
    data = yaml.safe_load(f)
    print(f"总页面: {len(data['pages'])}")
    for key, page in data['pages'].items():
        print(f"{key}: {page['stats']['total_elements']} 个元素")
EOF

# 查找特定元素（例如：包含"标题"的输入框）
python3 <<EOF
import yaml
with open('metadata_creator.yaml') as f:
    data = yaml.safe_load(f)

for page_key, page in data['pages'].items():
    print(f"\n页面: {page_key}")
    for elem in page['elements']:
        if elem.get('placeholder') and '标题' in elem['placeholder']:
            print(f"  找到标题输入框:")
            print(f"    - 选择器: {elem['css_selector']}")
            print(f"    - placeholder: {elem['placeholder']}")
            print(f"    - 类型: {elem['type']}")
            print(f"    - 可见: {elem['visible']}")
EOF
```

---

## 对比：旧版 vs 新版

### 旧版（collect_all.sh + refresh_selectors）

```yaml
# 采集结果示例
inputs:
  - text: ""
    selector: .d-text
    placeholder: "填写标题会有更多赞哦～"
    type: text
    classes: [d-text]
```

**缺陷**:
- ❌ 只有6个字段
- ❌ 缺少位置、可见性、ARIA属性
- ❌ 缺少data-*属性
- ❌ 无法判断元素层级关系

### 新版（toolchain_v2.sh + collect_metadata）

```yaml
# 采集结果示例
elements:
  - tag_name: input
    css_selector: "input.d-text"
    placeholder: "填写标题会有更多赞哦～"
    type: text
    classes: [d-text]
    id: ""
    name: ""

    # ���增50+字段
    position: {x: 120, y: 200, width: 600, height: 40}
    visible: true
    display: "block"
    opacity: "1"

    data_attrs:
      data-testid: "title-input"

    aria_attrs: {}

    parent_tag: div
    parent_classes: [form-group]
    children_count: 0
```

**优势**:
- ✅ 50+字段
- ✅ 完整的位置、可见性信息
- ✅ 所有 data-* 和 aria-* 属性
- ✅ 层级关系
- ✅ 无硬编码规则

---

## 下一步：智能选择器提取

基于完整的元数据，可以实现智能选择器提取：

### 规则示例

```python
# 识别标题输入框
def find_title_input(elements):
    for elem in elements:
        if (elem['tag_name'] == 'input' and
            elem['type'] == 'text' and
            elem['visible'] and
            '标题' in elem.get('placeholder', '') and
            elem['position']['y'] < 300):  # 在页面上部
            return elem['css_selector']

# 识别内容编辑器
def find_content_editor(elements):
    for elem in elements:
        if (elem['contenteditable'] == 'true' and
            elem['role'] == 'textbox' and
            elem['visible'] and
            any('tiptap' in c or 'editor' in c for c in elem['classes'])):
            return elem['css_selector']

# 识别发布按钮
def find_publish_button(elements):
    for elem in elements:
        if (elem['tag_name'] == 'button' and
            '发布' in elem['text'] and
            not elem['disabled'] and
            elem['visible'] and
            elem['position']['x'] > 500):  # 在页面右侧
            return elem['css_selector']
```

这样即使小红书改版，只需：
1. 重新运行 `./toolchain_v2.sh`
2. 智能提取器自动识别新的选择器
3. 无需修改代码

---

## 常见问题

### Q: 为什么需要交互模式？

A: 某些页面（如发布页面）的元素是动态加载的，需要手动操作才能显示：
- 发布页面：需要上传图片后才会显示标题和内容输入框
- 编辑页面：需要点击编辑按钮才会显示编辑器

### Q: 采集要多久？

A: 取决于页面数量和等待时间：
- 单个页面：5-10秒（wait=5）
- 10个页面：约1-2分钟（非交互模式）
- 10个页面：约5-10分钟（交互模式，需手动操作）

### Q: 元数据文件太大怎么办？

A:
1. 可以只采集关键页面（使用 --page 参数）
2. 元数据主要用于分析和开发，不需要随代码提交
3. 可以添加到 .gitignore: `metadata_*.yaml`

### Q: 如何更新选择器？

A:
1. 运行 `./toolchain_v2.sh` 重新采集元数据
2. 查看新的元数据文件
3. 更新 `wiring_bootstrap.go` 中的选择器映射逻辑
4. 或者等待智能提取器开发完成（自动更新）

---

## 文件说明

- `toolchain_v2.sh` - 完整工具链脚本
- `cmd/collect_metadata/main.go` - 元数据采集器
- `cmd/discover_pages/` - 页面发现器（已有）
- `cmd/refresh_selectors/` - 旧版采集器（已废弃）

生成的文件:
- `discovered_pages_*.yaml` - 发现的页面列表
- `metadata_*.yaml` - 完整元数据
- `metadata_*.json` - JSON格式（可选）
