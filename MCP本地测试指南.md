# MCP 本地测试指南

## 🚀 快速开始

### 方法一：使用启动脚本（推荐）

```bash
./start_mcp.sh
```

脚本会自动：
1. 检查必需文件
2. 询问运行模式（有头/无头）
3. 设置端口
4. 启动服务器

### 方法二：直接命令行启动

```bash
# 有头模式（可以看到浏览器窗口）
./bin/xiaohongshu-mcp --headless=false --port :18060

# 无头模式（后台运行）
./bin/xiaohongshu-mcp --headless --port :18060
```

---

## 📋 启动前准备

### 必需文件清单

| 文件 | 用途 | 如何获取 |
|------|------|----------|
| `bin/xiaohongshu-mcp` | MCP服务器程序 | `go build -o bin/xiaohongshu-mcp .` |
| `config.yaml` | 全局配置 | 已存在（URLs、超时等） |
| `cookies.json` | 登录凭证 | `./login.sh` |
| `selectors_discovered_pages_creator.yaml` | 页面选择器 | `./collect_all.sh` |

### 快速准备所有文件

```bash
# 1. 编译（如果还没编译）
go build -o bin/xiaohongshu-mcp .

# 2. 登录 + 采集（一键完成）
./collect_all.sh

# 3. 启动MCP服务器
./start_mcp.sh
```

---

## 🎮 命令行参数

### 完整参数列表

```bash
./bin/xiaohongshu-mcp [options]

Options:
  --headless        是否无头模式 (默认: true)
  --headless=false  有头模式，可以看到浏览器窗口
  --port            服务器端口 (默认: :18060)
  --config          配置文件路径 (默认: 自动查找 config.yaml)
  --bin             浏览器二进制文件路径 (可选)
```

### 常用启动命令

```bash
# 1. 默认启动（无头模式，端口18060）
./bin/xiaohongshu-mcp

# 2. 有头模式（调试推荐）
./bin/xiaohongshu-mcp --headless=false

# 3. 自定义端口
./bin/xiaohongshu-mcp --port :8080

# 4. 指定配置文件
./bin/xiaohongshu-mcp --config /path/to/config.yaml

# 5. 指定浏览器路径
./bin/xiaohongshu-mcp --bin /path/to/chrome
```

---

## 🔍 验证服务器启动

### 查看启动日志

服务器启动时会显示：

```
INFO[...] 配置文件加载成功
INFO[...] 📂 找到选择器文件: selectors_discovered_pages_creator.yaml
INFO[...] 📦 检测到采集器生成的选择器文件，正在提取发布页面选择器...
INFO[...]   ✓ upload_input: input
INFO[...]   ✓ title_input: .d-text (placeholder: 填写标题会有更多赞哦～)
INFO[...]   ✓ content: .tiptap (富文本编辑器)
INFO[...]   ✓ submit: button.d-button (text: 发布)
INFO[...]   ✓ save_draft: button.d-button (text: 暂存离开)
INFO[...] Starting server on :18060
```

### 测试端点

**1. MCP 端点**
```bash
curl http://localhost:18060/mcp
```

**2. REST API 健康检查**
```bash
curl http://localhost:18060/api/v1/health
```

**3. Swagger 文档**
```
浏览器访问: http://localhost:18060/swagger/index.html
```

---

## 🧪 测试MCP工具

### 1. 使用curl测试

**列出可用工具**
```bash
curl -X POST http://localhost:18060/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list"
  }'
```

**发布笔记示例**
```bash
curl -X POST http://localhost:18060/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "publish_note",
      "arguments": {
        "title": "测试笔记",
        "content": "这是一条测试笔记",
        "images": ["/path/to/image.jpg"]
      }
    }
  }'
```

### 2. 使用Cursor/Claude Code测试

在Cursor或Claude Code中配置MCP服务器：

**配置文件位置**：
- Cursor: `.cursor/mcp.json`
- Claude Code: `.vscode/mcp.json`

**配置内容**：
```json
{
  "mcpServers": {
    "xiaohongshu-mcp": {
      "url": "http://localhost:18060/mcp",
      "description": "小红书内容发布服务"
    }
  }
}
```

然后在编辑器中就可以使用MCP工具了。

---

## 📊 运行模式对比

### 有头模式 (--headless=false)

**优点**：
- ✅ 可以看到浏览器窗口
- ✅ 方便调试和观察执行过程
- ✅ 可以手动介入（如果需要）

**缺点**：
- ❌ 占用屏幕空间
- ❌ 稍微慢一些

**适用场景**：
- 开发和调试
- 首次测试
- 排查问题

### 无头模式 (--headless 或 --headless=true)

**优点**：
- ✅ 后台运行，不占用屏幕
- ✅ 稍微快一些
- ✅ 适合服务器部署

**缺点**：
- ❌ 看不到执行过程
- ❌ 调试相对困难

**适用场景**：
- 生产环境
- 持续运行
- 自动化任务

---

## 🔧 故障排查

### Q: 提示"找不到选择器文件"

**现象**：
```
WARN[...] 初始化发布用例失败: open selectors_discovered_pages_creator.yaml: no such file or directory
```

**解决**：
```bash
# 运行采集工具生成选择器文件
./collect_all.sh
```

### Q: 提示"Cookie过期"或"需要登录"

**现象**：
```
WARN[...] Cookie 可能已过期
```

**解决**：
```bash
# 重新登录
./login.sh

# 重启MCP服务器
./bin/xiaohongshu-mcp
```

### Q: 端口已被占用

**现象**：
```
FATA[...] failed to run server: listen tcp :18060: bind: address already in use
```

**解决**：
```bash
# 方法1: 使用其他端口
./bin/xiaohongshu-mcp --port :18061

# 方法2: 停止占用端口的进程
lsof -ti:18060 | xargs kill -9
```

### Q: 浏览器启动失败

**现象**：
```
ERROR[...] 启动浏览器失败
```

**解决**：
```bash
# 检查playwright是否安装
# macOS
brew install playwright

# 或使用系统Chrome
./bin/xiaohongshu-mcp --bin /Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome
```

---

## 🎯 测试检查清单

启动MCP服务器前：
- [ ] 已编译 `bin/xiaohongshu-mcp`
- [ ] 已登录（有 `cookies.json`）
- [ ] 已采集选择器（有 `selectors_discovered_pages_creator.yaml`）
- [ ] 端口未被占用

启动后验证：
- [ ] 日志显示"配置文件加载成功"
- [ ] 日志显示"找到选择器文件"
- [ ] 日志显示提取了5个选择器（upload_input, title_input等）
- [ ] 日志显示"Starting server on :18060"
- [ ] curl测试MCP端点返回正常
- [ ] Swagger文档可以访问

---

## 📝 推荐工作流

### 开发调试

```bash
# 1. 有头模式启动，便于观察
./bin/xiaohongshu-mcp --headless=false

# 2. 使用curl或Swagger测试API

# 3. 遇到问题查看浏览器窗口和日志

# 4. 修改代码后重新编译
go build -o bin/xiaohongshu-mcp .

# 5. 重启服务器（Ctrl+C 停止，再次启动）
```

### 生产运行

```bash
# 1. 无头模式后台运行
nohup ./bin/xiaohongshu-mcp --headless --port :18060 > mcp.log 2>&1 &

# 2. 查看日志
tail -f mcp.log

# 3. 停止服务
pkill xiaohongshu-mcp
```

---

## 🌐 MCP端点说明

### MCP协议端点
- **URL**: `http://localhost:18060/mcp`
- **协议**: MCP (Model Context Protocol)
- **用途**: Claude/Cursor等AI工具调用

### REST API端点
- **Base URL**: `http://localhost:18060/api/v1/`
- **协议**: REST HTTP
- **用途**: 直接HTTP调用

### 文档端点
- **Swagger**: `http://localhost:18060/swagger/index.html`
- **用途**: API文档和���试

---

## 🎉 快速测试成功示例

```bash
# 1. 启动服务器
$ ./start_mcp.sh
选择: 有头模式
端口: :18060

🚀 启动 MCP 服务器
INFO[...] 📂 找到选择器文件: selectors_discovered_pages_creator.yaml
INFO[...] ✓ upload_input: input
INFO[...] ✓ title_input: .d-text
INFO[...] ✓ content: .tiptap
INFO[...] ✓ submit: button.d-button
INFO[...] Starting server on :18060

# 2. 测试MCP端点（新终端）
$ curl http://localhost:18060/mcp
{"jsonrpc":"2.0","id":null,"result":{"protocolVersion":"2024-11-05","capabilities":{...}}}

# 3. 访问Swagger文档
浏览器打开: http://localhost:18060/swagger/index.html

✅ 测试成功！
```

---

**提示**: 首次启动建议使用有头模式 (`./start_mcp.sh`)，可以直观看到选择器是否正确加载！
