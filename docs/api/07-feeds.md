# 获取Feed列表 API

## 概述

获取小红书首页推荐Feed流或用户的笔记列表。

## 接口信息

**模块**: `feeds.go`, `data.go`
**功能**: 获取首页推荐笔记列表、获取用户笔记列表

## JSON Schema

### 1. 获取首页Feed列表

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "GetFeedsList",
  "description": "获取首页推荐Feed流",
  "type": "object",
  "properties": {}
}
```

### 2. 获取用户笔记列表

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "GetMyFeeds",
  "description": "获取我的笔记列表",
  "type": "object",
  "properties": {
    "limit": {
      "type": "integer",
      "description": "限制返回的笔记数量（0=不限制）",
      "default": 0,
      "minimum": 0,
      "examples": [10, 20, 50, 0]
    }
  }
}
```

## 字段说明

### 首页Feed列表

无需参数，直接调用即可。

### 用户笔记列表

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `limit` | integer | ❌ | 0 | 限制笔记数量，0表示不限制 |

## 调用示例

### 示例 1: 获取首页Feed

```json
{}
```

### 示例 2: 获取我的前10条笔记

```json
{
  "limit": 10
}
```

### 示例 3: 获取我的所有笔记

```json
{
  "limit": 0
}
```

### 示例 4: 获取我的前50条笔记

```json
{
  "limit": 50
}
```

## 响应格式

### 成功响应

```json
{
  "success": true,
  "total": 156,
  "feeds": [
    {
      "id": "65f8a3b2c4d1e5f6a7b8c9d0",
      "type": "normal",
      "title": "深圳美食探店",
      "cover": {
        "url": "https://...",
        "width": 1080,
        "height": 1440
      },
      "user": {
        "user_id": "5f7e...",
        "nickname": "美食探店小王",
        "avatar": "https://..."
      },
      "interact_info": {
        "liked_count": "1.2w",
        "collected_count": "8563",
        "comment_count": "234",
        "share_count": "156"
      },
      "xsec_token": "abc123...",
      "last_update_time": 1706169600000
    }
  ]
}
```

### Feed对象字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 笔记ID |
| `type` | string | 类型（normal/video） |
| `title` | string | 标题 |
| `cover` | object | 封面图信息 |
| `user` | object | 作者信息 |
| `interact_info` | object | 互动数据 |
| `xsec_token` | string | 安全令牌 |
| `last_update_time` | integer | 最后更新时间戳 |

### 失败响应

```json
{
  "success": false,
  "error_code": "PAGE_LOAD_FAILED",
  "message": "页面加载失败"
}
```

## 错误代码

| 错误代码 | 说明 |
|----------|------|
| `PAGE_LOAD_FAILED` | 页面加载失败 |
| `NOT_LOGGED_IN` | 未登录 |
| `NO_FEEDS` | 没有Feed数据 |

## 功能特性

### 1. 首页Feed流

- **个性化推荐**: 基于用户兴趣推荐
- **实时更新**: 内容持续更新
- **多样化内容**: 图文、视频混合
- **瀑布流加载**: 滚动加载更多

### 2. 用户笔记列表

- **自己的笔记**: 查看已发布的所有笔记
- **数量限制**: 可限制返回数量
- **时间排序**: 按发布时间倒序
- **状态筛选**: 可查看公开/私密笔记

## Feed类型

| 类型 | 说明 | 特点 |
|------|------|------|
| `normal` | 图文笔记 | 1-18张图片 + 文字 |
| `video` | 视频笔记 | 短视频 + 文字 |

## 使用场景

### 首页Feed

- 发现新内容
- 了解平台热门
- 获取推荐笔记
- 内容灵感收集

### 用户笔记列表

- 查看自己的发布历史
- 管理已发布内容
- 数据分析和统计
- 内容备份

## 数据说明

### 互动数据格式

小红书对大数字采用缩写：
- `1.2w` = 12000
- `5000+` = 超过5000
- `10w+` = 超过100000

实际使用时需要解析这些格式。

### 时间戳

- 单位：毫秒
- 格式：Unix timestamp (milliseconds)
- 示例：`1706169600000` = 2024-01-25 12:00:00

## 性能优化

1. **按需加载**: 使用limit参数限制数量
2. **缓存策略**: 缓存已获取的Feed
3. **增量更新**: 记录lastUpdateTime，只获取新内容
4. **批量处理**: 合理控制请求频率

## 注意事项

1. **登录要求**: 首页Feed无需登录，用户笔记需要登录
2. **数据实时性**: Feed数据会实时变化
3. **个性化差异**: 不同用户看到的Feed不同
4. **加载时间**: 首次加载可能需要等待

## 推荐算法

小红书Feed推荐考虑因素：
- 用户兴趣标签
- 历史浏览行为
- 互动记录（点赞、收藏、评论）
- 关注用户的内容
- 热门话题
- 地理位置
- 时间因素

## 相关文档

- [搜索 API](./02-search.md)
- [获取笔记详情 API](./05-feed-detail.md)
- [数据分析 API](./10-data-analytics.md)
