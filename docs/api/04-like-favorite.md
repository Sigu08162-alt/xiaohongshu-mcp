# 点赞/收藏 API

## 概述

对小红书笔记进行点赞或收藏操作。

## 接口信息

**模块**: `like_favorite.go`, `comment_like.go`
**功能**: 笔记点赞/取消点赞、笔记收藏/取消收藏、评论点赞/取消点赞

## JSON Schema

### 1. 笔记点赞/取消点赞

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "LikeFeed",
  "description": "点赞或取消点赞笔记",
  "type": "object",
  "required": ["feed_id", "xsec_token", "action"],
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
    },
    "action": {
      "type": "string",
      "description": "操作类型",
      "enum": ["like", "unlike"],
      "examples": ["like"]
    }
  }
}
```

### 2. 笔记收藏/取消收藏

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "FavoriteFeed",
  "description": "收藏或取消收藏笔记",
  "type": "object",
  "required": ["feed_id", "xsec_token", "action"],
  "properties": {
    "feed_id": {
      "type": "string",
      "description": "笔记ID"
    },
    "xsec_token": {
      "type": "string",
      "description": "安全令牌"
    },
    "action": {
      "type": "string",
      "description": "操作类型",
      "enum": ["favorite", "unfavorite"],
      "examples": ["favorite"]
    }
  }
}
```

### 3. 评论点赞/取消点赞

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "LikeComment",
  "description": "点赞或取消点赞评论",
  "type": "object",
  "required": ["feed_id", "xsec_token", "comment_id", "user_id", "action"],
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
    },
    "action": {
      "type": "string",
      "description": "操作类型",
      "enum": ["like", "unlike"],
      "examples": ["like"]
    }
  }
}
```

## 字段说明

### 笔记点赞/收藏

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `feed_id` | string | ✅ | 笔记ID |
| `xsec_token` | string | ✅ | 安全令牌 |
| `action` | enum | ✅ | 操作类型：like/unlike/favorite/unfavorite |

### 评论点赞

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `feed_id` | string | ✅ | 笔记ID |
| `xsec_token` | string | ✅ | 安全令牌 |
| `comment_id` | string | ✅ | 评论ID |
| `user_id` | string | ✅ | 评论者用户ID |
| `action` | enum | ✅ | like（点赞）或 unlike（取消点赞） |

## 调用示例

### 示例 1: 点赞笔记

```json
{
  "feed_id": "65f8a3b2c4d1e5f6a7b8c9d0",
  "xsec_token": "abc123def456",
  "action": "like"
}
```

### 示例 2: 取消点赞

```json
{
  "feed_id": "65f8a3b2c4d1e5f6a7b8c9d0",
  "xsec_token": "abc123def456",
  "action": "unlike"
}
```

### 示例 3: 收藏笔记

```json
{
  "feed_id": "65f8a3b2c4d1e5f6a7b8c9d0",
  "xsec_token": "abc123def456",
  "action": "favorite"
}
```

### 示例 4: 取消收藏

```json
{
  "feed_id": "65f8a3b2c4d1e5f6a7b8c9d0",
  "xsec_token": "abc123def456",
  "action": "unfavorite"
}
```

### 示例 5: 点赞评论

```json
{
  "feed_id": "65f8a3b2c4d1e5f6a7b8c9d0",
  "xsec_token": "abc123def456",
  "comment_id": "comment_abc123",
  "user_id": "user_xyz789",
  "action": "like"
}
```

## 响应格式

### 成功响应

```json
{
  "success": true,
  "action": "like",
  "message": "点赞成功"
}
```

### 失败响应

```json
{
  "success": false,
  "error_code": "ALREADY_LIKED",
  "message": "已经点赞过了"
}
```

## 错误代码

| 错误代码 | 说明 |
|----------|------|
| `FEED_NOT_FOUND` | 笔记不存在 |
| `COMMENT_NOT_FOUND` | 评论不存在 |
| `INVALID_XSEC_TOKEN` | 安全令牌无效 |
| `ALREADY_LIKED` | 已经点赞过 |
| `NOT_LIKED_YET` | 尚未点赞，无法取消 |
| `ALREADY_FAVORITED` | 已经收藏过 |
| `NOT_FAVORITED_YET` | 尚未收藏，无法取消 |
| `OPERATION_FAILED` | 操作失败 |

## 功能特性

### 1. 幂等性

- 重复点赞：系统会自动跳过，不会报错
- 重复取消：系统会自动跳过，不会报错
- 保证操作的稳定性

### 2. 状态检测

- 操作前自动检测当前状态
- 已点赞则跳过点赞操作
- 未点赞则跳过取消点赞操作

### 3. 独立操作

- 点赞和收藏是独立的操作
- 可以只点赞不收藏
- 可以只收藏不点赞
- 也可以同时点赞和收藏

## 点赞vs收藏

| 功能 | 点赞 | 收藏 |
|------|------|------|
| 作用 | 表达喜欢、支持 | 保存以便日后查看 |
| 可见性 | 作者可见点赞数 | 仅自己可见收藏夹 |
| 通知 | 作者收到点赞通知 | 作者收到收藏通知 |
| 查找 | 作者查看点赞列表 | 自己的收藏夹 |
| 推荐影响 | 影响推荐算法 | 影响推荐算法 |

## 使用场景

### 点赞

- 快速表达喜欢
- 支持优质内容
- 认可作者努力
- 参与互动

### 收藏

- 保存实用攻略
- 标记想要购买的商品
- 收集灵感创意
- 稍后仔细查看

### 评论点赞

- 认同某条评论观点
- 支持有价值的回复
- 帮助优质评论获得更多曝光

## 注意事项

1. **真实互动**: 避免批量刷赞行为
2. **有选择性**: 只对真正喜欢的内容点赞/收藏
3. **尊重版权**: 收藏不代表可以转载
4. **管理收藏夹**: 定期清理不需要的收藏

## 相关文档

- [获取笔记详情 API](./05-feed-detail.md)
- [评论 API](./03-comment.md)
- [关注 API](./06-follow.md)
