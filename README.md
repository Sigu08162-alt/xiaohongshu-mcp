# xiaohongshu-mcp

> 🚀 **小红书 MCP 服务** — 通过 [Model Context Protocol](https://modelcontextprotocol.io/) 让 AI 助手（Claude、Cursor、Windsurf 等）直接操作小红书：发帖、搜索、互动、数据分析，一切尽在掌握。

---

## ✨ 功能总览

| 模块 | 功能 |
|------|------|
| 🔐 **登录认证** | 二维码扫码登录、Cookie 导入/导出、登录状态检查 |
| 📝 **内容发布** | 发布图文笔记、发布视频笔记、保存草稿（图文/视频）、定时发布 |
| 🔍 **内容发现** | 首页 Feed 流、关键词搜索（含筛选）、笔记详情（含评论） |
| 👤 **用户信息** | 查看用户主页、获取自己的资料和笔记列表 |
| 💬 **内容互动** | 点赞/取消点赞、收藏/取消收藏、发表评论、回复评论、关注/取关、分享 |
| 📊 **数据分析** | 粉丝概览与画像、内容多维指标（曝光/点赞/评论/涨粉等） |

---

## 🏗️ 架构设计

本项目基于 **Go 1.24** 构建，同时提供两套接口：

```
┌─────────────────────────────────────────────────┐
│                 xiaohongshu-mcp                 │
│                                                 │
│  ┌──────────────┐      ┌──────────────────────┐ │
│  │  MCP Server  │      │   REST API Server    │ │
│  │  /mcp        │      │   /api/v1/*          │ │
│  │  (go-sdk)    │      │   (Gin Framework)    │ │
│  └──────┬───────┘      └──────────┬───────────┘ │
│         └──────────┬──────────────┘             │
│              ┌─────▼──────┐                     │
│              │  Service   │                     │
│              │   Layer    │                     │
│              └─────┬──────┘                     │
│              ┌─────▼──────┐                     │
│              │ Playwright │  (无头浏览器自动化)   │
│              │  Browser   │                     │
│              └────────────┘                     │
└─────────────────────────────────────────────────┘
```

- **MCP 协议层**：使用 `modelcontextprotocol/go-sdk`，支持 Streamable HTTP，兼容所有主流 MCP 客户端
- **REST API 层**：基于 Gin，提供完整 REST 接口，支持 Swagger 文档（`/api/v1/*`）
- **浏览器自动化**：基于 `playwright-go`，模拟真实用户操作，支持无头模式

---

## 🚀 快速开始

### 方式一：Docker 部署（推荐）

```bash
# 1. 克隆项目
git clone https://github.com/vmxmy/xiaohongshu-mcp.git
cd xiaohongshu-mcp/docker

# 2. 启动服务
docker compose up -d

# 3. 服务启动后访问
# MCP 端点：http://localhost:18060/mcp
# REST API：http://localhost:18060/api/v1/
# 健康检查：http://localhost:18060/health
```

**docker-compose.yml 配置：**

```yaml
services:
  xiaohongshu-mcp:
    image: ghcr.io/vmxmy/xiaohongshu-mcp:latest
    container_name: xiaohongshu-mcp
    restart: unless-stopped
    volumes:
      - ./data:/app/data      # Cookie 持久化
      - ./images:/app/images  # 图片缓存
    environment:
      - COOKIES_PATH=/app/data/cookies.json
    ports:
      - "18060:18060"
```

### 方式二：二进制直接运行

从 [Releases](https://github.com/vmxmy/xiaohongshu-mcp/releases) 下载对应平台的二进制文件：

```bash
# Linux/macOS
chmod +x xiaohongshu-mcp-linux-amd64
./xiaohongshu-mcp-linux-amd64

# 可选参数
./app --headless=true --port=18060
```

### 方式三：从源码编译

```bash
git clone https://github.com/vmxmy/xiaohongshu-mcp.git
cd xiaohongshu-mcp
go build -o xiaohongshu-mcp .
./xiaohongshu-mcp
```

---

## 🔌 MCP 客户端接入

### Claude Desktop / Cursor / Windsurf

在 MCP 配置文件中添加：

```json
{
  "mcpServers": {
    "xiaohongshu": {
      "url": "http://localhost:18060/mcp"
    }
  }
}
```

### nanobot

```json
{
  "servers": {
    "xhs": {
      "url": "http://localhost:18060/mcp"
    }
  }
}
```

---

## 🔐 登录认证

服务启动后，首次使用需要登录：

### 方式一：二维码扫码（MCP 工具）

通过 MCP 客户端调用 `get_login_qrcode` 工具，用小红书 App 扫码即可。

### 方式二：上传 Cookie（推荐长期使用）

1. 在浏览器中登录小红书，导出 Cookie（推荐使用 EditThisCookie 等插件）
2. 调用 `sync_cookies` 工具或 REST API 上传：

```bash
curl -X POST http://localhost:18060/api/v1/login/sync_cookies \
  -H "Content-Type: application/json" \
  -d '{"cookies_json": "[{\"name\":\"...\",\"value\":\"...\"}]"}'
```

### 检查登录状态

```bash
curl http://localhost:18060/api/v1/login/status
```

---

## 🛠️ MCP 工具列表

| 工具名 | 描述 |
|--------|------|
| `check_login_status` | 检查当前登录状态 |
| `get_login_qrcode` | 获取登录二维码（Base64 图片） |
| `sync_cookies` | 上传 Cookie JSON 完成登录 |
| `delete_cookies` | 删除 Cookie，重置登录状态 |
| `publish_content` | 发布图文笔记 |
| `publish_with_video` | 发布视频笔记 |
| `save_draft` | 保存图文草稿 |
| `save_video_draft` | 保存视频草稿 |
| `list_feeds` | 获取首页 Feed 流 |
| `search_feeds` | 搜索笔记 |
| `get_feed_detail` | 获取笔记详情及评论 |
| `user_profile` | 获取用户主页信息 |
| `get_my_stats` | 获取自己的账号统计 |
| `get_my_feeds` | 获取自己发布的笔记列表 |
| `like_feed` | 点赞/取消点赞笔记 |
| `favorite_feed` | 收藏/取消收藏笔记 |
| `post_comment_to_feed` | 发表评论 |
| `reply_comment_in_feed` | 回复评论 |
| `like_comment` | 点赞评论 |
| `delete_comment` | 删除自己的评论 |
| `follow_user` | 关注/取关用户 |
| `share_feed` | 获取笔记分享链接 |
| `delete_feed` | 删除自己的笔记 |
| `get_fan_analytics` | 获取粉丝分析数据 |
| `get_content_analytics` | 获取内容数据分析 |

---

## 📡 REST API 端点

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/health` | 健康检查 |
| GET | `/api/v1/login/status` | 检查登录状态 |
| GET | `/api/v1/login/qrcode` | 获取登录二维码 |
| POST | `/api/v1/login/sync_cookies` | 上传 Cookie |
| DELETE | `/api/v1/login/cookies` | 删除 Cookie |
| POST | `/api/v1/publish` | 发布图文笔记 |
| POST | `/api/v1/publish_video` | 发布视频笔记 |
| POST | `/api/v1/draft` | 保存图文草稿 |
| POST | `/api/v1/draft_video` | 保存视频草稿 |
| GET | `/api/v1/feeds/list` | 首页 Feed 流 |
| GET/POST | `/api/v1/feeds/search` | 搜索笔记 |
| POST | `/api/v1/feeds/detail` | 笔记详情 |
| POST | `/api/v1/feeds/like` | 点赞/取消点赞 |
| POST | `/api/v1/feeds/favorite` | 收藏/取消收藏 |
| POST | `/api/v1/feeds/comment` | 发表评论 |
| POST | `/api/v1/feeds/comment/reply` | 回复评论 |
| POST | `/api/v1/feeds/comment/like` | 点赞评论 |
| POST | `/api/v1/feeds/share` | 分享笔记 |
| DELETE | `/api/v1/feeds/:feed_id` | 删除笔记 |
| DELETE | `/api/v1/feeds/:feed_id/comments/:comment_id` | 删除评论 |
| POST | `/api/v1/user/profile` | 用户主页 |
| POST | `/api/v1/user/follow` | 关注/取关 |
| GET | `/api/v1/user/me` | 我的资料 |
| GET | `/api/v1/user/me/feeds` | 我的笔记列表 |
| GET | `/api/v1/analytics/fans` | 粉丝分析 |
| GET | `/api/v1/analytics/content` | 内容分析 |

---

## 📋 发布笔记示例

### 发布图文（MCP）

```json
{
  "title": "普吉岛隐藏宝藏🏖️",
  "content": "今天发现了一个超小众的海滩，人少景美...",
  "images": [
    "https://your-cdn.com/image1.jpg",
    "/local/path/to/image2.jpg"
  ],
  "tags": ["普吉岛", "泰国旅行", "小众海滩"],
  "location": "Phuket, Thailand"
}
```

> ⚠️ **注意**：`content` 正文中**不要**添加 `#标签`，所有话题标签统一通过 `tags` 参数传入（无需加 `#`），工具会自动处理。

### 定时发布

```json
{
  "schedule_at": "2024-03-01T10:00:00+08:00"
}
```

---

## 🔧 配置说明

| 启动参数 | 默认值 | 说明 |
|----------|--------|------|
| `--headless` | `true` | 是否使用无头浏览器模式 |
| `--bin` | 自动检测 | 自定义浏览器二进制路径 |
| `--port` | `18060` | 服务监听端口 |

| 环境变量 | 说明 |
|----------|------|
| `COOKIES_PATH` | Cookie 文件存储路径（默认 `./cookies/cookies.json`） |

---

## 🐳 Docker 环境变量

```yaml
environment:
  - COOKIES_PATH=/app/data/cookies.json  # Cookie 持久化路径
```

数据目录挂载：
- `/app/data` — Cookie 等持久化数据
- `/app/images` — 图片临时缓存

---

## 🔨 开发者指南

### 技术栈

| 组件 | 技术 |
|------|------|
| 语言 | Go 1.24 |
| MCP SDK | `modelcontextprotocol/go-sdk` |
| Web 框架 | Gin |
| 浏览器自动化 | `playwright-go` |
| 日志 | `sirupsen/logrus` |

### 本地开发

```bash
# 安装依赖
go mod download

# 安装 Playwright 浏览器
go run github.com/playwright-community/playwright-go/cmd/playwright install chromium

# 以有头模式运行（方便调试）
go run . --headless=false

# 运行测试
go test ./...
```

---

## ❓ 常见问题

**Q: 浏览器启动失败？**
> 确保已安装 Playwright 所需的系统依赖。Docker 镜像已预装全部依赖，推荐使用 Docker 部署。

**Q: 登录状态丢失？**
> 将 `./data` 目录挂载为 Volume 以持久化 Cookie。Cookie 文件位于 `COOKIES_PATH` 配置路径。

**Q: 图片上传失败？**
> - 本地路径需为绝对路径（如 `/home/user/image.jpg`）
> - 远程 MCP 服务无法访问客户端本地文件，请先上传至 CDN（如 Cloudflare R2）再使用公开 URL

**Q: 遇到风控/验证码？**
> 建议使用 Cookie 方式登录，并避免频繁操作。可尝试以有头模式启动并手动完成验证。

---

## 📄 License

[MIT License](./LICENSE)

---

## 🙏 致谢

本项目 Fork 自 **[xpzouying/xiaohongshu-mcp](https://github.com/xpzouying/xiaohongshu-mcp)**，感谢原作者 [@xpzouying](https://github.com/xpzouying) 的出色工作！

原项目是一个功能完整的小红书 MCP Server，支持登录、笔记发布、评论互动、数据分析等丰富功能。作者还将项目所有赞赏款项悉数捐赠慈善，精神令人钦佩。

- 🔗 原项目：[github.com/xpzouying/xiaohongshu-mcp](https://github.com/xpzouying/xiaohongshu-mcp)
- 📝 作者博客：[haha.ai/xiaohongshu-mcp](https://www.haha.ai/xiaohongshu-mcp)
- 💝 原项目支持慈善捐赠，欢迎前往原仓库支持作者

---

## 🌟 Star History

如果这个项目对你有帮助，欢迎给个 Star ⭐

[![Star History Chart](https://api.star-history.com/svg?repos=vmxmy/xiaohongshu-mcp&type=Date)](https://star-history.com/#vmxmy/xiaohongshu-mcp&Date)
