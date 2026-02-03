# Swagger API 文档

> 本仓库默认未启用 Swagger UI（已移除内置 docs 包与路由）。
> 如需启用，请按 `docs/SWAGGER_SETUP.md` 手动恢复路由并生成 docs 包。

## 简介

xiaohongshu-mcp 可提供 Swagger/OpenAPI 文档支持，用于交互式测试 API（需手动启用）。

## 访问文档

启用 Swagger UI 后，通过浏览器访问:

```
http://localhost:18060/swagger/index.html
```

## 功能特性

### 1. 交互式 API 测试

- **在线测试**: 直接在浏览器中测试所有 API 接口
- **参数填写**: 表单式参数填写,支持示例值
- **实时响应**: 查看实时的请求和响应数据
- **错误提示**: 清晰的错误信息和状态码说明

### 2. 完整的 API 文档

所有 REST API 都已添加 Swagger 注解,包括:

#### 登录认证 (3个接口)
- `GET /api/v1/login/status` - 检查登录状态
- `GET /api/v1/login/qrcode` - 获取登录二维码
- `DELETE /api/v1/login/cookies` - 删除 Cookies

#### 内容发布 (2个接口)
- `POST /api/v1/publish` - 发布图文笔记
- `POST /api/v1/publish_video` - 发布视频笔记

#### 内容发现 (4个接口)
- `GET /api/v1/feeds/list` - 获取首页 Feeds 列表
- `GET /api/v1/feeds/search` - 搜索笔记 (GET)
- `POST /api/v1/feeds/search` - 搜索笔记 (POST)
- `POST /api/v1/feeds/detail` - 获取笔记详情

#### 用户信息 (2个接口)
- `GET /api/v1/user/me` - 获取我的资料
- `POST /api/v1/user/profile` - 获取用户主页

#### 内容互动 (2个接口)
- `POST /api/v1/feeds/comment` - 发表评论
- `POST /api/v1/feeds/comment/reply` - 回复评论

### 3. OpenAPI 规范

文档同时生成了标准的 OpenAPI 规范文件:

- **JSON 格式**: `docs/swagger.json`
- **YAML 格式**: `docs/swagger.yaml`
- **Go 代码**: `docs/docs.go`

这些文件可用于:
- 生成客户端 SDK
- 导入到 Postman/Insomnia
- 集成到 API 网关
- 自动生成测试用例

## 使用步骤

### 1. 启动服务

```bash
./xiaohongshu-mcp --headless --port :18060
```

### 2. 打开浏览器

访问 Swagger UI:
```
http://localhost:18060/swagger/index.html
```

### 3. 测试 API

1. 选择要测试的接口
2. 点击 "Try it out"
3. 填写必填参数
4. 点击 "Execute"
5. 查看响应结果

## 示例: 检查登录状态

### 请求

```bash
curl -X GET "http://localhost:18060/api/v1/login/status" -H "accept: application/json"
```

### 响应

```json
{
  "success": true,
  "data": {
    "logged_in": true,
    "user_id": "xxx",
    "nickname": "xxx"
  },
  "message": "检查登录状态成功"
}
```

## 示例: 搜索笔记

### 请求

```bash
curl -X POST "http://localhost:18060/api/v1/feeds/search" \
  -H "Content-Type: application/json" \
  -d '{
    "keyword": "深圳美食",
    "filters": {
      "sort_by": "最新",
      "note_type": "图文"
    }
  }'
```

### 响应

```json
{
  "success": true,
  "data": {
    "feeds": [
      {
        "id": "xxx",
        "title": "深圳必吃美食推荐",
        "type": "normal",
        "cover": {
          "url": "https://...",
          "width": 1080,
          "height": 1440
        },
        "author": {
          "user_id": "xxx",
          "nickname": "美食博主"
        },
        "interact_info": {
          "liked_count": "1.2w",
          "collected_count": "8563"
        }
      }
    ]
  },
  "message": "搜索Feeds成功"
}
```

## 响应格式

所有 API 使用统一的响应格式:

### 成功响应

```json
{
  "success": true,
  "data": { ... },
  "message": "操作成功"
}
```

### 失败响应

```json
{
  "success": false,
  "error": "错误描述",
  "code": "ERROR_CODE",
  "details": { ... }
}
```

## 常见错误代码

| 错误代码 | 说明 | 解决方案 |
|----------|------|----------|
| `NOT_LOGGED_IN` | 未登录 | 先调用登录接口 |
| `INVALID_REQUEST` | 请求参数错误 | 检查参数格式 |
| `STATUS_CHECK_FAILED` | 状态检查失败 | 查看详细错误信息 |
| `PUBLISH_FAILED` | 发布失败 | 检查内容合规性 |
| `SEARCH_FEEDS_FAILED` | 搜索失败 | 检查关键词和筛选条件 |

## 与 MCP 协议的关系

xiaohongshu-mcp 同时提供两种接口:

1. **REST API** (`/api/v1/*`)
   - 标准 HTTP/JSON 接口
   - 可用 curl/Postman/浏览器调用
   - 有完整的 Swagger 文档

2. **MCP 协议** (`/mcp`)
   - 专用于 AI 模型的协议
   - Claude Code/Claude.ai 使用
   - 25个 MCP 工具

## 更新文档

修改代码后重新生成文档:

```bash
# 1. 添加/修改 Swagger 注解
# 2. 重新生成文档
~/go/bin/swag init --parseDependency --parseInternal

# 3. 重新编译
go build -o xiaohongshu-mcp

# 4. 重启服务
./xiaohongshu-mcp
```

## Swagger 注解示例

```go
// @Summary 检查登录状态
// @Description 检查当前是否已登录小红书
// @Tags 登录认证
// @Produce json
// @Success 200 {object} SuccessResponse "登录状态信息"
// @Failure 500 {object} ErrorResponse "服务器内部错误"
// @Router /login/status [get]
func (s *AppServer) checkLoginStatusHandler(c *gin.Context) {
    // handler implementation
}
```

## 技术栈

- **Web 框架**: Gin
- **Swagger 生成**: swaggo/swag
- **Swagger UI**: gin-swagger
- **OpenAPI 版本**: 2.0

## 相关链接

- [Swagger UI 文档](https://swagger.io/tools/swagger-ui/)
- [swaggo/swag GitHub](https://github.com/swaggo/swag)
- [OpenAPI 规范](https://swagger.io/specification/)
- [Gin Web Framework](https://gin-gonic.com/)

## 注意事项

1. **文档实时性**: 修改代码后需要重新运行 `swag init` 生成文档
2. **类型支持**: Swagger 注解支持 Go 的基本类型和结构体
3. **嵌套类型**: 复杂的嵌套类型可能需要简化为基础类型
4. **安全性**: 本地开发环境使用,生产环境建议添加认证

## 技术支持

如有问题或建议,请:
1. 查看 Swagger UI 中的 API 文档
2. 检查错误代码说明
3. 查看服务端日志
4. 提交 GitHub Issue
