# 获取笔记详情 API

## 概述

获取小红书笔记的完整详情信息，包括笔记内容、评论列表等。

## 接口信息

**模块**: `feed_detail.go`
**功能**: 获取笔记详情和评论列表

## JSON Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "GetFeedDetail",
  "description": "获取笔记详情",
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
      "description": "安全令牌（从搜索接口获取）",
      "examples": ["abc123def456"]
    },
    "load_all_comments": {
      "type": "boolean",
      "description": "是否加载所有评论（默认false）",
      "default": false,
      "examples": [true, false]
    },
    "comment_config": {
      "type": "object",
      "description": "评论加载配置（可选）",
      "properties": {
        "click_more_replies": {
          "type": "boolean",
          "description": "是否展开更多回复",
          "default": false
        },
        "max_replies_threshold": {
          "type": "integer",
          "description": "最大回复数阈值",
          "default": 10,
          "minimum": 0
        },
        "max_comment_items": {
          "type": "integer",
          "description": "最大评论数限制（0表示不限制）",
          "default": 0,
          "minimum": 0
        },
        "scroll_speed": {
          "type": "string",
          "description": "滚动速度",
          "enum": ["slow", "normal", "fast"],
          "default": "normal"
        }
      }
    }
  }
}
```

## 字段说明

### 基础参数

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `feed_id` | string | ✅ | - | 笔记ID |
| `xsec_token` | string | ✅ | - | 安全令牌 |
| `load_all_comments` | boolean | ❌ | false | 是否加载所有评论 |
| `comment_config` | object | ❌ | - | 评论加载配置 |

### 评论配置参数

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `click_more_replies` | boolean | false | 是否展开更多回复 |
| `max_replies_threshold` | integer | 10 | 最大回复数阈值 |
| `max_comment_items` | integer | 0 | 最大评论数（0=不限制） |
| `scroll_speed` | enum | "normal" | 滚动速度：slow/normal/fast |

## 调用示例

### 示例 1: 基础获取详情

```json
{
  "feed_id": "65f8a3b2c4d1e5f6a7b8c9d0",
  "xsec_token": "abc123def456"
}
```

### 示例 2: 加载所有评论

```json
{
  "feed_id": "65f8a3b2c4d1e5f6a7b8c9d0",
  "xsec_token": "abc123def456",
  "load_all_comments": true
}
```

### 示例 3: 自定义评论加载配置

```json
{
  "feed_id": "65f8a3b2c4d1e5f6a7b8c9d0",
  "xsec_token": "abc123def456",
  "load_all_comments": true,
  "comment_config": {
    "click_more_replies": true,
    "max_replies_threshold": 20,
    "max_comment_items": 100,
    "scroll_speed": "fast"
  }
}
```

### 示例 4: 只获取前50条评论

```json
{
  "feed_id": "65f8a3b2c4d1e5f6a7b8c9d0",
  "xsec_token": "abc123def456",
  "load_all_comments": true,
  "comment_config": {
    "max_comment_items": 50,
    "scroll_speed": "normal"
  }
}
```

## 响应格式

### 成功响应

```json
{
  "success": true,
  "note": {
    "id": "65f8a3b2c4d1e5f6a7b8c9d0",
    "type": "normal",
    "title": "深圳美食探店",
    "desc": "今天探店深圳湾附近的网红餐厅...",
    "time": 1706169600000,
    "last_update_time": 1706169600000,
    "user": {
      "user_id": "5f7e...",
      "nickname": "美食探店小王",
      "avatar": "https://..."
    },
    "image_list": [
      {
        "url": "https://...",
        "width": 1080,
        "height": 1440
      }
    ],
    "tag_list": [
      {"name": "深圳美食", "type": "topic"},
      {"name": "探店", "type": "topic"}
    ],
    "interact_info": {
      "liked_count": "1.2w",
      "collected_count": "8563",
      "comment_count": "234",
      "share_count": "156"
    },
    "ip_location": "广东"
  },
  "comments": {
    "comments": [
      {
        "id": "comment_123",
        "content": "太棒了，收藏了！",
        "create_time": 1706170000000,
        "user_info": {
          "user_id": "user_abc",
          "nickname": "用户A",
          "avatar": "https://..."
        },
        "sub_comment_count": 3,
        "sub_comments": [
          {
            "id": "reply_456",
            "content": "确实不错！",
            "user_info": {...}
          }
        ],
        "like_count": 12
      }
    ],
    "has_more": false,
    "cursor": ""
  }
}
```

### 响应字段说明

#### Note（笔记）对象

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 笔记ID |
| `type` | string | 类型（normal/video） |
| `title` | string | 标题 |
| `desc` | string | 正文内容 |
| `time` | integer | 发布时间戳 |
| `user` | object | 作者信息 |
| `image_list` | array | 图片列表 |
| `tag_list` | array | 标签列表 |
| `interact_info` | object | 互动数据 |
| `ip_location` | string | IP归属地 |

#### Comments（评论）对象

| 字段 | 类型 | 说明 |
|------|------|------|
| `comments` | array | 评论列表 |
| `has_more` | boolean | 是否有更多评论 |
| `cursor` | string | 游标（用于分页） |

#### Comment（单条评论）对象

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 评论ID |
| `content` | string | 评论内容 |
| `create_time` | integer | 创建时间戳 |
| `user_info` | object | 评论者信息 |
| `sub_comment_count` | integer | 回复数量 |
| `sub_comments` | array | 回复列表 |
| `like_count` | integer | 点赞数 |

### 失败响应

```json
{
  "success": false,
  "error_code": "FEED_NOT_FOUND",
  "message": "笔记不存在或已被删除"
}
```

## 错误代码

| 错误代码 | 说明 |
|----------|------|
| `FEED_NOT_FOUND` | 笔记不存在 |
| `INVALID_XSEC_TOKEN` | 安全令牌无效 |
| `FEED_PRIVATE` | 笔记为私密，无权访问 |
| `FEED_DELETED` | 笔记已被删除 |
| `PAGE_LOAD_TIMEOUT` | 页面加载超时 |

## 功能特性

### 1. 智能评论加载

- **懒加载**: 默认只加载首屏评论
- **全量加载**: 设置 `load_all_comments=true` 加载所有评论
- **回复展开**: 可选择是否展开评论回复
- **数量限制**: 可限制最大评论数，避免过长等待

### 2. 灵活配置

- **滚动速度**: 根据网络情况调整
  - `slow`: 800ms间隔，适合慢速网络
  - `normal`: 500ms间隔，默认设置
  - `fast`: 300ms间隔，适合快速网络

### 3. 完整信息

获取的信息包括：
- 笔记完整内容
- 所有图片/视频
- 话题标签
- 互动数据
- 作者信息
- 评论及回复
- IP归属地

## 使用场景

- 查看笔记完整内容
- 分析评论讨论内容
- 数据分析和统计
- 内容备份
- 用户行为研究

## 性能优化建议

1. **按需加载评论**: 如果不需要评论，设置 `load_all_comments=false`
2. **限制评论数**: 设置 `max_comment_items` 避免加载过多
3. **调整滚动速度**: 根据网络情况选择合适的速度
4. **批量处理**: 需要获取多个笔记时，适当增加间隔

## 注意事项

1. 加载所有评论可能需要较长时间
2. 评论数量过多时建议设置 `max_comment_items`
3. 私密笔记需要权限才能访问
4. 已删除的笔记无法获取详情

## 相关文档

- [搜索 API](./02-search.md)
- [评论 API](./03-comment.md)
- [点赞收藏 API](./04-like-favorite.md)
