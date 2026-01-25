# 数据分析 API

## 概述

获取小红书创作者数据、运营分析、粉丝洞察等数据。

## 接口信息

**模块**: `data.go`
**功能**: 获取统计数据、粉丝分析、内容分析

## JSON Schema

### 1. 获取统计数据

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "GetMyStats",
  "description": "获取创作者统计数据",
  "type": "object",
  "properties": {}
}
```

### 2. 粉丝分析

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "GetFanAnalytics",
  "description": "获取粉丝分析数据",
  "type": "object",
  "properties": {
    "period": {
      "type": "string",
      "description": "分析周期",
      "enum": ["7d", "30d"],
      "default": "7d",
      "examples": ["7d", "30d"]
    }
  }
}
```

### 3. 内容分析

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "GetContentAnalytics",
  "description": "获取内容分析数据",
  "type": "object",
  "properties": {
    "limit": {
      "type": "integer",
      "description": "返回笔记数量限制（0=不限制）",
      "default": 0,
      "minimum": 0
    },
    "sort_by": {
      "type": "string",
      "description": "排序字段",
      "enum": ["exposure", "likes", "comments", "shares", "favorites"],
      "default": "exposure"
    },
    "sort_order": {
      "type": "string",
      "description": "排序顺序",
      "enum": ["asc", "desc"],
      "default": "desc"
    }
  }
}
```

## 字段说明

### 统计数据

无需参数。

### 粉丝分析

| 字段 | 类型 | 必填 | 默认值 | 可选值 |
|------|------|------|--------|--------|
| `period` | enum | ❌ | "7d" | "7d"（7天）/ "30d"（30天） |

### 内容分析

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `limit` | integer | ❌ | 0 | 笔记数量，0=不限制 |
| `sort_by` | enum | ❌ | "exposure" | 排序字段 |
| `sort_order` | enum | ❌ | "desc" | 排序顺序（asc升序/desc降序） |

## 调用示例

### 示例 1: 获取统计数据

```json
{}
```

### 示例 2: 获取7天粉丝分析

```json
{
  "period": "7d"
}
```

### 示例 3: 获取30天粉丝分析

```json
{
  "period": "30d"
}
```

### 示例 4: 按曝光量排序的内容分析

```json
{
  "limit": 20,
  "sort_by": "exposure",
  "sort_order": "desc"
}
```

### 示例 5: 按点赞数排序的内容分析

```json
{
  "limit": 10,
  "sort_by": "likes",
  "sort_order": "desc"
}
```

## 响应格式

### 1. 统计数据响应

```json
{
  "success": true,
  "stats": {
    "follower_count": 12500,
    "follow_count": 256,
    "liked_count": 56789,
    "exposure_count": 125000,
    "view_count": 89000,
    "like_count_7d": 1234,
    "comment_count_7d": 89,
    "share_count_7d": 45,
    "favorite_count_7d": 567,
    "new_fans_7d": 123,
    "note_count": 45
  }
}
```

### 统计数据字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `follower_count` | integer | 粉丝总数 |
| `follow_count` | integer | 关注总数 |
| `liked_count` | integer | 获赞与收藏总数 |
| `exposure_count` | integer | 曝光数（7日） |
| `view_count` | integer | 观看数（7日） |
| `like_count_7d` | integer | 点赞数（7日） |
| `comment_count_7d` | integer | 评论数（7日） |
| `share_count_7d` | integer | 分享数（7日） |
| `favorite_count_7d` | integer | 收藏数（7日） |
| `new_fans_7d` | integer | 新增粉丝（7日） |
| `note_count` | integer | 笔记总数 |

### 2. 粉丝分析响应

```json
{
  "success": true,
  "analytics": {
    "period": "7d",
    "new_fans": 123,
    "lost_fans": 12,
    "net_growth": 111,
    "demographics": {
      "gender": {
        "male": 35,
        "female": 65
      },
      "age": {
        "18-24": 45,
        "25-30": 35,
        "31-35": 15,
        "36+": 5
      },
      "region": {
        "广东": 30,
        "北京": 15,
        "上海": 12,
        "其他": 43
      }
    },
    "interests": [
      {"name": "美食", "percentage": 68},
      {"name": "旅游", "percentage": 45},
      {"name": "时尚", "percentage": 32}
    ]
  }
}
```

### 粉丝分析字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `period` | string | 分析周期 |
| `new_fans` | integer | 新增粉丝数 |
| `lost_fans` | integer | 流失粉丝数 |
| `net_growth` | integer | 净增长（新增-流失） |
| `demographics` | object | 人口统计数据 |
| `interests` | array | 兴趣分布 |

### 3. 内容分析响应

```json
{
  "success": true,
  "content": {
    "total_notes": 45,
    "notes": [
      {
        "id": "65f8a3b2c4d1e5f6a7b8c9d0",
        "title": "深圳美食探店",
        "type": "normal",
        "publish_time": 1706169600000,
        "metrics": {
          "exposure": 12500,
          "views": 8900,
          "likes": 1234,
          "comments": 89,
          "shares": 45,
          "favorites": 567,
          "engagement_rate": 15.6
        },
        "performance": "优秀"
      }
    ],
    "summary": {
      "avg_exposure": 2500,
      "avg_likes": 450,
      "avg_engagement_rate": 12.3,
      "best_performing": "65f8a3b2c4d1e5f6a7b8c9d0"
    }
  }
}
```

### 内容分析字段说明

#### Metrics（指标）

| 字段 | 类型 | 说明 |
|------|------|------|
| `exposure` | integer | 曝光数 |
| `views` | integer | 浏览数 |
| `likes` | integer | 点赞数 |
| `comments` | integer | 评论数 |
| `shares` | integer | 分享数 |
| `favorites` | integer | 收藏数 |
| `engagement_rate` | float | 互动率（%） |

#### Summary（汇总）

| 字段 | 类型 | 说明 |
|------|------|------|
| `avg_exposure` | integer | 平均曝光数 |
| `avg_likes` | integer | 平均点赞数 |
| `avg_engagement_rate` | float | 平均互动率 |
| `best_performing` | string | 表现最好的笔记ID |

## 排序字段说明

| 字段值 | 说明 |
|--------|------|
| `exposure` | 曝光数 |
| `likes` | 点赞数 |
| `comments` | 评论数 |
| `shares` | 分享数 |
| `favorites` | 收藏数 |

## 核心指标解释

### 1. 曝光与浏览

- **曝光数**: 内容被展示的次数
- **浏览数**: 用户实际点击查看的次数
- **点击率**: 浏览数 / 曝光数

### 2. 互动指标

- **点赞数**: 用户点赞次数
- **评论数**: 用户评论次数
- **分享数**: 用户分享次数
- **收藏数**: 用户收藏次数

### 3. 综合指标

- **互动率**: (点赞+评论+分享+收藏) / 浏览数
- **粉丝增长率**: 新增粉丝 / 总粉丝数
- **内容产出**: 一定周期内发布的笔记数

## 数据分析维度

### 时间维度

- 7日数据：短期趋势
- 30日数据：中期趋势
- 历史对比：长期趋势

### 内容维度

- 笔记类型：图文vs视频
- 话题分类：不同类别表现
- 发布时间：最佳发布时段

### 用户维度

- 性别分布
- 年龄分布
- 地域分布
- 兴趣偏好

## 使用场景

### 创作者运营

- 了解账号增长趋势
- 分析内容表现
- 优化发布策略
- 了解粉丝画像

### 数据分析

- 追踪关键指标
- 发现增长机会
- 识别优质内容
- 调整内容方向

### 商业合作

- 展示账号价值
- 证明影响力
- 洽谈合作报价
- 制定营销策略

## 数据应用建议

### 内容优化

1. **分析高表现内容**
   - 找出共同特点
   - 复制成功经验
   - 优化内容方向

2. **改进低表现内容**
   - 分析失败原因
   - 调整创作策略
   - 避免重复错误

### 发布策略

1. **最佳发布时间**
   - 分析历史数据
   - 找出高峰时段
   - 优化发布时间

2. **发布频率**
   - 保持稳定产出
   - 避免过度发布
   - 保证内容质量

### 粉丝运营

1. **了解粉丝画像**
   - 性别、年龄分布
   - 地域分布
   - 兴趣偏好

2. **精准内容定位**
   - 匹配粉丝需求
   - 提供有价值内容
   - 增强粉丝粘性

## 关键指标基准

以下为不同粉丝量级的参考标准：

### 新手账号（<1000粉丝）

- 互动率：5-10%
- 7日新增粉丝：10-50
- 平均曝光：500-2000

### 成长账号（1000-1万粉丝）

- 互动率：10-15%
- 7日新增粉丝：50-200
- 平均曝光：2000-1万

### 头部账号（1万-10万粉丝）

- 互动率：8-12%
- 7日新增粉丝：200-1000
- 平均曝光：1万-10万

### 顶级账号（>10万粉丝）

- 互动率：5-10%
- 7日新增粉丝：1000+
- 平均曝光：10万+

## 注意事项

1. **数据延迟**: 数据可能有1-2小时延迟
2. **数据波动**: 正常现象，关注趋势而非单点
3. **数据隐私**: 仅自己可见，不可分享
4. **合理预期**: 增长需要时间和优质内容
5. **综合分析**: 多维度综合判断

## 相关文档

- [获取笔记详情 API](./05-feed-detail.md)
- [用户资料 API](./09-user-profile.md)
- [获取Feed列表 API](./07-feeds.md)
