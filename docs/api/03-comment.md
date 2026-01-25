# 评论 API

## 概述

对小红书笔记发表评论或回复其他用户的评论。

## 接口信息

**模块**: `comment_feed.go`
**功能**: 发表评论、回复评论

## JSON Schema

### 1. 发表评论

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "PostComment",
  "description": "发表笔记评论",
  "type": "object",
  "required": ["feed_id", "xsec_token", "content"],
  "properties": {
    "feed_id": {
      "type": "string",
      "description": "笔记ID",
      "examples": ["65f8a3b2c4d1e5f6a7b8c9d0"]
    },
    "xsec_token": {
      "type": "string",
      "description": "安全令牌（从搜索或详情接口获取）",
      "examples": ["abc123def456"]
    },
    "content": {
      "type": "string",
      "description": "评论内容",
      "minLength": 1,
      "maxLength": 1000,
      "examples": ["太棒了，收藏了！", "请问在哪里可以买到？"]
    }
  }
}
```

### 2. 回复评论

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "ReplyComment",
  "description": "回复指定评论",
  "type": "object",
  "required": ["feed_id", "xsec_token", "comment_id", "user_id", "content"],
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
      "description": "被回复的评论ID"
    },
    "user_id": {
      "type": "string",
      "description": "被回复的用户ID"
    },
    "content": {
      "type": "string",
      "description": "回复内容",
      "minLength": 1,
      "maxLength": 1000
    }
  }
}
```

## 字段说明

### 发表评论

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `feed_id` | string | ✅ | 笔记ID |
| `xsec_token` | string | ✅ | 安全令牌 |
| `content` | string | ✅ | 评论内容，1-1000字符 |

### 回复评论

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `feed_id` | string | ✅ | 笔记ID |
| `xsec_token` | string | ✅ | 安全令牌 |
| `comment_id` | string | ✅ | 被回复的评论ID |
| `user_id` | string | ✅ | 被回复的用户ID |
| `content` | string | ✅ | 回复内容，1-1000字符 |

## 调用示例

### 示例 1: 发表评论

```json
{
  "feed_id": "65f8a3b2c4d1e5f6a7b8c9d0",
  "xsec_token": "abc123def456",
  "content": "太实用了，已经收藏！感谢分享~"
}
```

### 示例 2: 发表带表情的评论

```json
{
  "feed_id": "65f8a3b2c4d1e5f6a7b8c9d0",
  "xsec_token": "abc123def456",
  "content": "哇，太美了！😍 请问是在哪里拍的？"
}
```

### 示例 3: 回复评论

```json
{
  "feed_id": "65f8a3b2c4d1e5f6a7b8c9d0",
  "xsec_token": "abc123def456",
  "comment_id": "comment_abc123",
  "user_id": "user_xyz789",
  "content": "是的，在深圳湾公园拍的！"
}
```

### 示例 4: 回复并@用户

```json
{
  "feed_id": "65f8a3b2c4d1e5f6a7b8c9d0",
  "xsec_token": "abc123def456",
  "comment_id": "comment_abc123",
  "user_id": "user_xyz789",
  "content": "@小红 确实很不错，值得一去！"
}
```

## 响应格式

### 成功响应

```json
{
  "success": true,
  "comment_id": "comment_new123",
  "message": "评论发布成功"
}
```

### 失败响应

```json
{
  "success": false,
  "error_code": "COMMENT_TOO_LONG",
  "message": "评论内容过长",
  "details": {
    "max_length": 1000,
    "current_length": 1025
  }
}
```

## 错误代码

| 错误代码 | 说明 |
|----------|------|
| `FEED_NOT_FOUND` | 笔记不存在 |
| `INVALID_XSEC_TOKEN` | 安全令牌无效 |
| `COMMENT_EMPTY` | 评论内容为空 |
| `COMMENT_TOO_LONG` | 评论内容过长 |
| `COMMENT_NOT_FOUND` | 被回复的评论不存在 |
| `USER_NOT_FOUND` | 被回复的用户不存在 |
| `COMMENT_DISABLED` | 该笔记不允许评论 |
| `RATE_LIMIT_EXCEEDED` | 评论频率过高 |
| `SENSITIVE_CONTENT` | 包含敏感内容 |

## 使用限制

1. 评论内容不能为空
2. 评论长度不超过1000字符
3. 不能过于频繁发送评论（防止刷屏）
4. 不能包含违规敏感内容
5. 部分笔记可能关闭评论功能

## 功能特性

### 1. 支持表情符号

评论内容可以包含 emoji 表情：
- 😀 😍 🎉 ❤️ 等常见表情
- 小红书内置表情（通过表情选择器）

### 2. @用户功能

在评论中可以@其他用户：
- 格式：`@用户昵称`
- 被@的用户会收到通知

### 3. 回复嵌套

支持对评论的回复：
- 一级评论：直接评论笔记
- 二级回复：回复某条评论

## 使用场景

- 对优质内容发表看法
- 向作者提问
- 与其他用户交流讨论
- 补充相关信息
- 表达赞赏和支持

## 注意事项

1. **文明评论**: 不发表攻击性、侮辱性言论
2. **真实互动**: 避免刷评论、灌水行为
3. **尊重原创**: 不抄袭他人评论
4. **隐私保护**: 不泄露个人敏感信息
5. **遵守规则**: 遵守小红书社区规范

## 评论礼仪建议

- ✅ 具体、有价值的反馈
- ✅ 友善、礼貌的表达
- ✅ 相关、切题的讨论
- ❌ 无意义的水评（"沙发"、"打卡"等）
- ❌ 恶意攻击或负面情绪
- ❌ 广告、引流行为

## 相关文档

- [评论点赞 API](./04-comment-like.md)
- [获取笔记详情 API](./05-feed-detail.md)
- [删除评论 API](./08-delete.md)
