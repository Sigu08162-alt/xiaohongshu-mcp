# 小红书页面链接自动发现工具

从小红书创作者中心首页自动发现所有真实链接，适应页面地址变化。

## 问题背景

**当前问题**：
- `refresh_selectors` 工具使用硬编码URL
- 页面地址可能变化（如 `/content` 返回404）
- 无法自动适应小红书更新

**解决方案**：
- 自动从首页爬取所有真实链接
- 保存到 `discovered_pages.yaml`
- `refresh_selectors` 可动态加载这些链接

## 功能特性

- ✅ **动态发现**：从首页自动发现所有创作者中心链接
- ✅ **智能分类**：自动识别页面类别（发布、内容、数据等）
- ✅ **去重整理**：自动去重并生成唯一键
- ✅ **YAML输出**：结构化保存，易于使用
- ✅ **Cookie支持**：自动加载cookie，无需登录

## 使用方法

### 步骤1：发现页面链接

```bash
# 运行页面发现工具
./bin/discover_pages

# 或使用 Makefile
make discover-pages
```

**输出文件**：`discovered_pages.yaml`

```yaml
version: 1.0.0
generated: "2026-01-30T14:00:00+08:00"
home_page: https://creator.xiaohongshu.com/new/home?source=official
links:
  publish_publish:
    text: "发布笔记"
    url: "https://creator.xiaohongshu.com/publish/publish?source=official&target=image"
    description: "内容发布相关页面"
    category: "publish"
  content:
    text: "内容管理"
    url: "https://creator.xiaohongshu.com/creator/content"
    description: "内容管理相关页面"
    category: "content"
  # ... 更多链接
```

### 步骤2：使用发现的链接采集组件

```bash
# 使用动态发现的页面列表
./bin/refresh_selectors --pages discovered_pages.yaml

# 或使用 Makefile
make refresh-selectors-from-discovered
```

## 工作流程

```
┌─────────────────────────────────────────────┐
│  1. 登录小红书 (一次性)                      │
│     ./login.sh                              │
└──────────────────┬──────────────────────────┘
                   │
                   v
┌─────────────────────────────────────────────┐
│  2. 发现页面链接                             │
│     ./bin/discover_pages                    │
│     ↓                                       │
│     discovered_pages.yaml (真实链接)         │
└──────────────────┬──────────────────────────┘
                   │
                   v
┌─────────────────────────────────────────────┐
│  3. 采集组件信息                             │
│     ./bin/refresh_selectors                 │
│       --pages discovered_pages.yaml         │
│     ↓                                       │
│     selectors_all_pages.yaml (组件信息)      │
└─────────────────────────────────────────────┘
```

## 参数说明

### discover_pages

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--output` | `discovered_pages.yaml` | 输出YAML文件路径 |
| `--json` | (空) | 可选：同时输出JSON |
| `--home` | `creator.xiaohongshu.com/new/home` | 首页URL |
| `--wait` | `5` | 页面加载等待秒数 |
| `--cookies` | (自动查找) | Cookie文件路径 |
| `--headless` | `false` | 是否无头模式 |

### refresh_selectors（新增参数）

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--pages` | (空) | 从YAML文件加载页面列表 |

## 输出示例

### 终端输出

```
=== 小红书页面链接自动发现工具 ===
首页URL: https://creator.xiaohongshu.com/new/home?source=official
输出YAML: discovered_pages.yaml
有头模式: true
等待时间: 5秒
🍪 Cookie文件: cookies.json

📦 启动浏览器...

✅ 已加载Cookie文件，无需手动登录

⏸️  按 Enter 开始发现页面...

🌐 访问首页: https://creator.xiaohongshu.com/new/home?source=official
⏳ 等待 5 秒让页面加载完成...

🔍 开始发现页面链接...
📊 发现 25 个创作者中心链接
📊 发现 8 个导航链接

💾 保存到 YAML: discovered_pages.yaml

============================================================
📊 发现结果汇总
============================================================
📅 生成时间: 2026-01-30T14:00:00+08:00
🏠 首页: https://creator.xiaohongshu.com/new/home?source=official
🔗 发现链接: 25 个

📋 按类别统计:
  - publish: 3 个
  - content: 5 个
  - data: 4 个
  - fans: 2 个
  - income: 1 个
  - other: 10 个

🔑 关键页面:
  - publish_publish: https://creator.xiaohongshu.com/publish/publish?source=official&target=image
    文本: 发布笔记
  - content: https://creator.xiaohongshu.com/creator/content
    文本: 内容管理

============================================================

✅ 发现完成！
```

## 链接分类

工具自动将链接分类到以下类别：

- **publish**: 内容发布相关（发布笔记、发布视频）
- **content**: 内容管理相关（内容列表、笔记管理）
- **data**: 数据分析相关（数据概览、粉丝分析）
- **fans**: 粉丝管理相关
- **comment**: 评论管理相关
- **income**: 收益相关
- **setting**: 设置相关（自动跳过）
- **help**: 帮助相关（自动跳过）
- **other**: 其他功能

## 与 refresh_selectors 集成

### 传统方式（硬编码）

```bash
# 使用内置的硬编码URL
./bin/refresh_selectors
```

### 动态方式（推荐）

```bash
# 1. 发现真实链接
./bin/discover_pages

# 2. 使用发现的链接
./bin/refresh_selectors --pages discovered_pages.yaml
```

**优势**：
- 自动适应页面地址变化
- 覆盖更多页面
- 避免404错误

## 定期更新建议

```bash
# 每月运行一次，更新页面列表
# 建议添加到 cron 或 GitHub Actions

# 1. 发现最新链接
./bin/discover_pages --output discovered_pages_$(date +%Y%m).yaml

# 2. 对比变化
diff discovered_pages_202601.yaml discovered_pages_202602.yaml

# 3. 更新主文件
cp discovered_pages_202602.yaml discovered_pages.yaml

# 4. 重新采集组件
./bin/refresh_selectors --pages discovered_pages.yaml
```

## 故障排查

### 问题1：发现的链接太少

```bash
# 增加等待时间，确保页面完全加载
./bin/discover_pages --wait 10
```

### 问题2：需要登录

```bash
# 重新生成 cookie
./login.sh

# 再次运行
./bin/discover_pages
```

### 问题3：某些链接不存在

某些链接可能需要特定权限或仅在特定条件下显示，这是正常的。

## 技术细节

### 链接发现策略

1. 查找所有 `<a href>` 标签
2. 过滤：只保留 `creator.xiaohongshu.com` 域名
3. 去重：基于 URL 去重
4. 分类：根据 URL 和文本自动分类
5. 生成键：从 URL 路径生成唯一键

### 键生成规则

```
URL: https://creator.xiaohongshu.com/publish/publish?source=official&target=image
     ↓
路径: /publish/publish
     ↓
键:   publish_publish
```

## 最佳实践

1. **定期更新**：每月运行一次 `discover_pages`
2. **版本控制**：将 `discovered_pages.yaml` 纳入 git
3. **对比分析**：对比历史版本，发现变化
4. **优先使用**：优先使用动态发现的链接
5. **保留备份**：保留硬编码列表作为备选

## 与其他工具配合

```bash
# 完整工作流
./login.sh                                  # 1. 登录
./bin/discover_pages                        # 2. 发现链接
./bin/refresh_selectors --pages discovered_pages.yaml  # 3. 采集组件
```

## 输出文件

- **discovered_pages.yaml**: 发现的所有链接
- **discovered_pages.json**: 可选，JSON 格式
