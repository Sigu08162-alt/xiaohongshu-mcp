# 删除 API

## 概述

删除小红书笔记或评论。

## 接口信息

**模块**: `delete.go`
**功能**: 删除笔记、删除评论

## JSON Schema

### 1. 删除笔记

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "DeleteFeed",
  "description": "删除笔记",
  "type": "object",
  "required": ["feed_id", "xsec_token"],
  "properties": {
    "feed_id": {
      "type": "string",
      "description": "笔记ID",
      "examples": ["65f8a3b2c4d1e5f6a7b8c9d0"]
    },
    "xsec_token": {
      "type": "string",
      "description": "安全令牌",
      "examples": ["abc123def456"]
    }
  }
}
```

### 2. 删除评论

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "DeleteComment",
  "description": "��除评论",
  "type": "object",
  "required": ["feed_id", "xsec_token", "comment_id", "user_id"],
  "properties": {
    "feed_id": {
      "type": "string",
      "description": "笔记ID"
    },
    "xsec_token": {
      "type": "string",
      "description": "安全令牌"
    },
    "comment_id": {
      "type": "string",
      "description": "评论ID"
    },
    "user_id": {
      "type": "string",
      "description": "评论者用户ID"
    }
  }
}
```

## 字段说明

### 删除笔记

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `feed_id` | string | ✅ | 笔记ID |
| `xsec_token` | string | ✅ | 安全令牌 |

### 删除评论

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `feed_id` | string | ✅ | 笔记ID |
| `xsec_token` | string | ✅ | 安全令牌 |
| `comment_id` | string | ✅ | 评论ID |
| `user_id` | string | ✅ | 评论者用户ID |

## 调用示例

### 示例 1: 删除笔记

```json
{
  "feed_id": "65f8a3b2c4d1e5f6a7b8c9d0",
  "xsec_token": "abc123def456"
}
```

### 示例 2: 删除评论

```json
{
  "feed_id": "65f8a3b2c4d1e5f6a7b8c9d0",
  "xsec_token": "abc123def456",
  "comment_id": "comment_abc123",
  "user_id": "user_xyz789"
}
```

## 响应格式

### 成功响应

```json
{
  "success": true,
  "message": "删除成功"
}
```

### 失败响应

```json
{
  "success": false,
  "error_code": "PERMISSION_DENIED",
  "message": "无权删除此笔记"
}
```

## 错误代码

| 错误代码 | 说明 |
|----------|------|
| `FEED_NOT_FOUND` | 笔记不存在 |
| `COMMENT_NOT_FOUND` | 评论不存在 |
| `PERMISSION_DENIED` | 无权删除 |
| `ALREADY_DELETED` | 已经被删除 |
| `DELETE_FAILED` | 删除操作失败 |

## 权限说明

### 删除笔记

只有以下情况可以删除笔记：
- ✅ 笔记作者本人
- ❌ 其他用户（即使是管理员）

### 删除评论

可以删除评论的情况：
- ✅ 评论作者本人
- ✅ 笔记作者（删除自己笔记下的任何评论）
- ❌ 其他用户

## 删除影响

### 删除笔记

删除后的影响：
- ❌ 笔记内容永久删除
- ❌ 所有图片/视频永久删除
- ❌ 所有评论永久删除
- ❌ 点赞、收藏数据清除
- ⚠️ **无法恢复**

### 删除评论

删除后的影响：
- ❌ 评论内容永久删除
- ❌ 评论的回复全部删除
- ❌ 评论的点赞数据清除
- ⚠️ **无法恢复**

## 使用场景

### 删除笔记

- 内容质量不满意
- 包含错误信息
- 过时的内容
- 隐私保护需求
- 账号清理

### 删除评论

- 评论内容不当
- 发表错误信息
- 收到恶意评论
- 管理笔记评论区

## 最佳实践

### ⚠️ 删除前确认

1. **二次确认**: 删除是不可逆操作
2. **备份内容**: 删除前保存重要内容
3. **考虑编辑**: 小错误可以编辑而非删除
4. **检查数据**: 确认是否影响数据统计

### 替代方案

| 需求 | 建议方案 |
|------|----------|
| 修改内容 | 使用编辑功能 |
| 暂时隐藏 | 改为"仅自己可见" |
| 数据保留 | 下载/备份后再删除 |

## 删除限制

1. **频率限制**: 短时间内不能删除过多
2. **数据保留**: 平台可能保留删除记录
3. **统计影响**: 影响账号数据统计
4. **恢复期**: 某些情况下可能有短暂恢复期

## 注意事项

1. **不可恢复**: 删除操作不可撤销
2. **权限检查**: 只能删除自己的内容
3. **数据丢失**: 所有相关数据都会丢失
4. **影响范围**: 删除笔记会删除所有评论
5. **谨慎操作**: 删除前请三思

## 数据保护建议

在删除之前：

1. **截图保存**: 保存笔记截图
2. **导出数据**: 保存图片、视频
3. **备份评论**: 保存有价值的评论
4. **检查链接**: 确认是否有外部引用
5. **统计记录**: 记录互动数据

## 法律说明

根据平台规定：
- 删除内容不代表完全清除痕迹
- 平台可能保留备份用于审计
- 涉及违规内容可能被永久记录
- 不能以删除方式逃避责任

## 相关文档

- [发布图文 API](./00-publish-image.md)
- [评论 API](./03-comment.md)
- [数据分析 API](./10-data-analytics.md)
