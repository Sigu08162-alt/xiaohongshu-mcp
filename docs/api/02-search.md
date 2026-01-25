# 搜索笔记 API

## 概述

搜索小红书平台上的笔记内容，支持多维度筛选。

## 接口信息

**模块**: `search.go`
**功能**: 搜索笔记并支持多种筛选条件

## JSON Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "SearchNotes",
  "description": "搜索小红书笔记",
  "type": "object",
  "required": ["keyword"],
  "properties": {
    "keyword": {
      "type": "string",
      "description": "搜索关键词",
      "minLength": 1,
      "examples": ["深圳美食", "穿搭", "旅游攻略"]
    },
    "sort_by": {
      "type": "string",
      "description": "排序依据",
      "enum": ["综合", "最新", "最多点赞", "最多评论", "最多收藏"],
      "default": "综合",
      "examples": ["综合", "最新"]
    },
    "note_type": {
      "type": "string",
      "description": "笔记类型",
      "enum": ["不限", "视频", "图文"],
      "default": "不限",
      "examples": ["视频", "图文"]
    },
    "publish_time": {
      "type": "string",
      "description": "发布时间范围",
      "enum": ["不限", "一天内", "一周内", "半年内"],
      "default": "不限",
      "examples": ["一周内"]
    },
    "search_scope": {
      "type": "string",
      "description": "搜索范围",
      "enum": ["不限", "已看过", "未看过", "已关注"],
      "default": "不限",
      "examples": ["已关注"]
    },
    "location": {
      "type": "string",
      "description": "位置距离筛选",
      "enum": ["不限", "同城", "附近"],
      "default": "不限",
      "examples": ["同城"]
    }
  }
}
```

## 字段说明

| 字段 | 类型 | 必填 | 默认值 | 可选值 |
|------|------|------|--------|--------|
| `keyword` | string | ✅ | - | 任意搜索词 |
| `sort_by` | enum | ❌ | "综合" | 综合/最新/最多点赞/最多评论/最多收藏 |
| `note_type` | enum | ❌ | "不限" | 不限/视频/图文 |
| `publish_time` | enum | ❌ | "不限" | 不限/一天内/一周内/半年内 |
| `search_scope` | enum | ❌ | "不限" | 不限/已看过/未看过/已关注 |
| `location` | enum | ❌ | "不限" | 不限/同城/附近 |

## 调用示例

### 示例 1: 简单搜索

```json
{
  "keyword": "深圳美食"
}
```

### 示例 2: 搜索最新视频

```json
{
  "keyword": "旅游攻略",
  "sort_by": "最新",
  "note_type": "视频"
}
```

### 示例 3: 搜索一周内同城笔记

```json
{
  "keyword": "探店",
  "publish_time": "一周内",
  "location": "同城"
}
```

### 示例 4: 搜索已关注用户的笔记

```json
{
  "keyword": "穿搭",
  "search_scope": "已关注",
  "sort_by": "最新"
}
```

### 示例 5: 综合筛选

```json
{
  "keyword": "健身",
  "sort_by": "最多点赞",
  "note_type": "视频",
  "publish_time": "一周内",
  "search_scope": "不限",
  "location": "同城"
}
```

## 响应格式

### 成功响应

```json
{
  "success": true,
  "total": 1523,
  "feeds": [
    {
      "id": "65f8a3b2c4d1e5f6a7b8c9d0",
      "title": "深圳必吃美食推荐",
      "type": "normal",
      "cover": {
        "url": "https://...",
        "width": 1080,
        "height": 1440
      },
      "user": {
        "user_id": "5f7e...",
        "nickname": "美食探店小王"
      },
      "interact_info": {
        "liked_count": "1.2w",
        "collected_count": "8563"
      },
      "xsec_token": "abc123..."
    }
  ]
}
```

### 字段说明

**Feed对象**:
| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 笔记ID |
| `title` | string | 笔记标题 |
| `type` | string | 笔记类型 (normal/video) |
| `cover` | object | 封面图信息 |
| `user` | object | 作者信息 |
| `interact_info` | object | 互动数据（点赞、收藏数） |
| `xsec_token` | string | 安全令牌（用于后续操作） |

### 失败响应

```json
{
  "success": false,
  "error_code": "SEARCH_FAILED",
  "message": "搜索失败",
  "details": {
    "reason": "关键词为空"
  }
}
```

## 错误代码

| 错误代码 | 说明 |
|----------|------|
| `EMPTY_KEYWORD` | 关键词为空 |
| `INVALID_SORT_BY` | 排序选项无效 |
| `INVALID_NOTE_TYPE` | 笔记类型选项无效 |
| `INVALID_PUBLISH_TIME` | 发布时间选项无效 |
| `INVALID_SEARCH_SCOPE` | 搜索范围选项无效 |
| `INVALID_LOCATION` | 位置选项无效 |
| `SEARCH_TIMEOUT` | 搜索超时 |
| `NO_RESULTS` | 没有搜索结果 |

## 使用限制

1. 关键词不能为空
2. 每页返回最多30条结果
3. 搜索结果由小红书平台算法决定
4. 筛选条件会影响搜索结果数量

## 使用场景

- 查找特定主题的笔记
- 寻找最新发布的内容
- 发现热门话题
- 查看已关注用户的相关内容
- 同城内容发现

## 注意事项

1. 多个筛选条件可以组合使用
2. "综合"排序会综合多个维度（相关性、热度等）
3. "同城"和"附近"需要用户授权位置信息
4. 搜索结果会根据用户个人兴趣有所不同

## 相关文档

- [获取笔记详情 API](./05-feed-detail.md)
- [点赞收藏 API](./04-like-favorite.md)
