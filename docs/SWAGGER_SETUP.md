# Swagger 文档配置完成

> 本仓库默认未启用 Swagger UI，需要手动恢复路由与 docs 包导入后再生成文档。

## 已完成的工作

### 1. 安装依赖包

```bash
go get -u github.com/swaggo/swag/cmd/swag
go get -u github.com/swaggo/gin-swagger
go get -u github.com/swaggo/files
```

### 2. 添加 Swagger 注解

**main.go** - 添加 API 文档元信息:
- API 标题和描述
- 版本信息
- 联系方式和许可证
- 主机和基础路径
- 标签定义

**handlers_api.go** - 为每个 HTTP handler 添加注解:
- 11 个 API 接口
- 包含请求参数、响应格式、错误代码
- 分类到 5 个标签组

### 3. 配置路由

**routes.go** - 添加 Swagger UI 路由:
```go
router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
```

### 4. 生成文档

```bash
~/go/bin/swag init --parseDependency --parseInternal
```

生成的文件:
- `docs/docs.go` - Go 代码
- `docs/swagger.json` - JSON 规范
- `docs/swagger.yaml` - YAML 规范

### 5. 编译和测试

```bash
go build -o xiaohongshu-mcp
./xiaohongshu-mcp --headless --port :18060
```

## API 文档结构

### 登录认证 (3个接口)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/login/status | 检查登录状态 |
| GET | /api/v1/login/qrcode | 获取登录二维码 |
| DELETE | /api/v1/login/cookies | 删除 Cookies |

### 内容发布 (2个接口)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/publish | 发布图文笔记 |
| POST | /api/v1/publish_video | 发布视频笔记 |

### 内容发现 (4个接口)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/feeds/list | 获取 Feeds 列表 |
| GET/POST | /api/v1/feeds/search | 搜索笔记 |
| POST | /api/v1/feeds/detail | 获取笔记详情 |

### 用户信息 (2个接口)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/user/me | 获取我的资料 |
| POST | /api/v1/user/profile | 获取用户主页 |

### 内容互动 (2个接口)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/feeds/comment | 发表评论 |
| POST | /api/v1/feeds/comment/reply | 回复评论 |

## 访问方式

### 1. Swagger UI (交互式文档)

```
http://localhost:18060/swagger/index.html
```

### 2. OpenAPI JSON

```
http://localhost:18060/swagger/doc.json
```

### 3. curl 调用

```bash
# 检查登录���态
curl http://localhost:18060/api/v1/login/status

# 搜索笔记
curl -X POST http://localhost:18060/api/v1/feeds/search \
  -H "Content-Type: application/json" \
  -d '{
    "keyword": "深圳美食",
    "filters": {
      "sort_by": "最新",
      "note_type": "图文"
    }
  }'
```

## 文件清单

新增/修改的文件:

```
xiaohongshu-mcp/
├── main.go                   # 添加 Swagger 注解和文档导入
├── routes.go                 # 添加 Swagger UI 路由
├── handlers_api.go           # 为所有 handler 添加注解
├── docs/
│   ├── docs.go              # 自动生成
│   ├── swagger.json         # 自动生成
│   ├── swagger.yaml         # 自动生成
│   ├── SWAGGER.md           # Swagger 使用文档
│   └── SWAGGER_SETUP.md     # 本文件
├── go.mod                   # 新增依赖
└── go.sum                   # 依赖校验和
```

## 技术细节

### Swagger 注解格式

```go
// @Summary 简短描述
// @Description 详细描述
// @Tags 标签名
// @Accept json
// @Produce json
// @Param name location type required "description"
// @Success 200 {object} ResponseType "说明"
// @Failure 400 {object} ErrorResponse "说明"
// @Router /path [method]
```

### 支持的参数位置

- `query` - URL 查询参数
- `path` - URL 路径参数
- `header` - HTTP 头
- `body` - 请求体
- `formData` - 表单数据

### 响应类型

所有 API 使用统一的响应格式:

**成功**:
```json
{
  "success": true,
  "data": {},
  "message": "操作成功"
}
```

**失败**:
```json
{
  "success": false,
  "error": "错误描述",
  "code": "ERROR_CODE",
  "details": {}
}
```

## 维护指南

### 更新文档

1. 修改代码中的 Swagger 注解
2. 重新生成文档:
```bash
~/go/bin/swag init --parseDependency --parseInternal
```
3. 重新编译:
```bash
go build -o xiaohongshu-mcp
```

### 添加新接口

1. 在 handler 函数上方添加 Swagger 注解
2. 使用 `@Router` 指定路由
3. 运行 `swag init` 生成文档
4. 测试新接口

### 常见问题

**Q: 文档不更新?**
A: 运行 `swag init` 重新生成文档

**Q: 类型定义找不到?**
A: 确保类型在同一个包内,或使用完整包路径

**Q: 嵌套类型报错?**
A: 简化为基础类型,避免复杂的嵌套结构

## 下一步

可以考虑添加:

1. **认证支持** - API Key 或 JWT 认证
2. **更多接口** - 数据分析、内容管理等接口
3. **示例值** - 为参数添加示例值
4. **请求/响应示例** - 完整的 JSON 示例
5. **错误码文档** - 详细的错误代码说明

## 参考资源

- [swaggo/swag 文档](https://github.com/swaggo/swag)
- [Swagger 注解说明](https://github.com/swaggo/swag#declarative-comments-format)
- [OpenAPI 规范](https://swagger.io/specification/)
- [Gin Web Framework](https://gin-gonic.com/)
