# 小红书图文发布 API 文档

## 概述

本文档描述了小红书图文笔记发布功能的 JSON API 参数结构，适用于 MCP 工具调用。

## JSON Schema 定义

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "PublishImageNote",
  "description": "发布小红书图文笔记",
  "type": "object",
  "required": ["title", "content", "image_paths"],
  "properties": {
    "title": {
      "type": "string",
      "description": "笔记标题（1-20字符）",
      "minLength": 1,
      "maxLength": 20,
      "examples": ["深圳周末打卡", "美食分享"]
    },
    "content": {
      "type": "string",
      "description": "笔记正文内容（1-1000字符）",
      "minLength": 1,
      "maxLength": 1000,
      "examples": ["今天去了深圳湾公园，风景真美！推荐大家周末来打卡~"]
    },
    "image_paths": {
      "type": "array",
      "description": "图片文件路径列表（1-18张）",
      "items": {
        "type": "string",
        "description": "图片文件的绝对路径或相对路径"
      },
      "minItems": 1,
      "maxItems": 18,
      "examples": [["/path/to/image1.jpg", "/path/to/image2.jpg"]]
    },
    "tags": {
      "type": "array",
      "description": "话题标签列表（可选）",
      "items": {
        "type": "string",
        "description": "话题名称，不需要带#号"
      },
      "examples": [["深圳旅游", "周末打卡", "美食分享"]]
    },
    "location": {
      "type": "string",
      "description": "地点名称（可选），支持城市、商圈、具体POI",
      "examples": ["深圳湾公园", "深圳市", "北京三里屯"]
    },
    "collection": {
      "type": "string",
      "description": "合集名称（可选），需要先在APP端创建合集",
      "examples": ["深圳探店", "美食合集"]
    },
    "group_chat": {
      "type": "string",
      "description": "关联群聊名称（可选）",
      "examples": ["深圳吃喝玩乐群"]
    },
    "marker_tags": {
      "type": "array",
      "description": "标记列表（可选），可以标记地点或用户昵称，系统会自动智能匹配",
      "items": {
        "type": "string",
        "description": "地点名称或用户昵称"
      },
      "examples": [["深圳湾公园", "张三", "某美食博主"]]
    },
    "original_claim": {
      "type": "boolean",
      "description": "是否声明原创（可选，默认false）",
      "default": false,
      "examples": [true, false]
    },
    "content_type": {
      "type": "string",
      "description": "内容类型声明（可选）",
      "enum": ["虚构演绎", "AI合成", "来源声明"],
      "examples": ["AI合成"]
    },
    "visible_scope": {
      "type": "string",
      "description": "可见范围（可选，默认公开可见）",
      "enum": ["公开可见", "仅自己可见", "仅互关好友可见", "只给谁看", "不给谁看"],
      "default": "公开可见",
      "examples": ["公开可见", "仅互关好友可见"]
    },
    "allow_duet": {
      "type": "boolean",
      "description": "是否允许合拍（可选，null表示使用默认值）",
      "examples": [true, false, null]
    },
    "allow_copy": {
      "type": "boolean",
      "description": "是否允许正文复制（可选，null表示使用默认值）",
      "examples": [true, false, null]
    },
    "schedule_time": {
      "type": "string",
      "description": "定时发布时间（可选，ISO 8601格式，null表示立即发布）",
      "format": "date-time",
      "examples": ["2026-01-26T10:30:00+08:00", null]
    }
  }
}
```

## 字段说明

### 必填字段

| 字段 | 类型 | 限制 | 说明 |
|------|------|------|------|
| `title` | string | 1-20字符 | 笔记标题 |
| `content` | string | 1-1000字符 | 笔记正文内容 |
| `image_paths` | array | 1-18个元素 | 图片文件路径列表 |

### 可选字段

#### 基础内容

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `tags` | array | `[]` | 话题标签列表，不需要带#号 |

#### 位置与社交

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `location` | string | `""` | 地点名称，支持城市、商圈、POI |
| `collection` | string | `""` | 合集名称（需先在APP端创建） |
| `group_chat` | string | `""` | 关联群聊名称 |
| `marker_tags` | array | `[]` | 标记的地点或用户，支持智能匹配 |

#### 权益与声明

| 字段 | 类型 | 默认值 | 可选值 |
|------|------|--------|--------|
| `original_claim` | boolean | `false` | `true` / `false` |
| `content_type` | string | `""` | `"虚构演绎"` / `"AI合成"` / `"来源声明"` |
| `visible_scope` | string | `"公开可见"` | `"公开可见"` / `"仅自己可见"` / `"仅互关好友可见"` / `"只给谁看"` / `"不给谁看"` |

#### 互动权限

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `allow_duet` | boolean | `null` | 是否允许合拍，`null`表示使用平台默认值 |
| `allow_copy` | boolean | `null` | 是否允许正文复制，`null`表示使用平台默认值 |

#### 发布控制

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `schedule_time` | string | `null` | 定时发布时间（ISO 8601格式），`null`表示立即发布 |

## 调用示例

### 示例 1: 简单发布（仅必填字段）

```json
{
  "title": "深圳周末打卡",
  "content": "今天去了深圳湾公园，风景真美！",
  "image_paths": [
    "/Users/username/Pictures/shenzhen1.jpg",
    "/Users/username/Pictures/shenzhen2.jpg"
  ]
}
```

### 示例 2: 完整配置发布

```json
{
  "title": "深圳湾公园游玩攻略",
  "content": "周末带家人来深圳湾公园，阳光正好，微风不燥。推荐傍晚来，可以看日落🌅",
  "image_paths": [
    "/Users/username/Photos/sunset1.jpg",
    "/Users/username/Photos/sunset2.jpg",
    "/Users/username/Photos/sunset3.jpg"
  ],
  "tags": ["深圳旅游", "周末打卡", "深圳湾公园", "亲子游"],
  "location": "深圳湾公园",
  "marker_tags": ["深圳湾公园", "深圳海滨栈道"],
  "collection": "深圳周末好去处",
  "original_claim": true,
  "visible_scope": "公开可见",
  "allow_duet": true,
  "allow_copy": true
}
```

### 示例 3: AI生成内容声明

```json
{
  "title": "AI绘画作品分享",
  "content": "使用AI工具生成的赛博朋克风格城市夜景，欢迎交流学习！",
  "image_paths": ["/Users/username/ai_art/cyberpunk1.png"],
  "tags": ["AI绘画", "赛博朋克", "数字艺术"],
  "content_type": "AI合成",
  "original_claim": false,
  "visible_scope": "公开可见"
}
```

### 示例 4: 定时发布

```json
{
  "title": "早安深圳",
  "content": "新的一天，从美好的早餐开始☀️",
  "image_paths": ["/path/to/breakfast.jpg"],
  "tags": ["早餐", "深圳美食"],
  "schedule_time": "2026-01-27T07:00:00+08:00"
}
```

### 示例 5: 私密笔记

```json
{
  "title": "个人生活记录",
  "content": "今天的心情和感悟...",
  "image_paths": ["/path/to/private_photo.jpg"],
  "visible_scope": "仅自己可见",
  "allow_copy": false
}
```

### 示例 6: 标记地点和用户

```json
{
  "title": "和朋友的聚会",
  "content": "周末和好友们在咖啡厅度过愉快的下午时光☕",
  "image_paths": ["/path/to/cafe.jpg"],
  "tags": ["咖啡", "周末"],
  "location": "星巴克(深圳湾店)",
  "marker_tags": ["星巴克(深圳湾店)", "张三", "李四"],
  "visible_scope": "公开可见"
}
```

## 响应格式

### 成功响应

```json
{
  "success": true,
  "note_id": "65f8a3b2c4d1e5f6a7b8c9d0",
  "url": "https://www.xiaohongshu.com/explore/65f8a3b2c4d1e5f6a7b8c9d0",
  "message": "发布成功"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `success` | boolean | 是否成功 |
| `note_id` | string | 笔记ID |
| `url` | string | 笔记详情页URL |
| `message` | string | 成功消息 |

### 失败响应

```json
{
  "success": false,
  "error_code": "INVALID_TITLE_LENGTH",
  "message": "标题长度超出限制（最多20字符）",
  "details": {
    "field": "title",
    "current_length": 25,
    "max_length": 20
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `success` | boolean | 是否成功（固定为false） |
| `error_code` | string | 错误代码 |
| `message` | string | 错误消息 |
| `details` | object | 错误详情（可选） |

### 常见错误代码

| 错误代码 | 说明 |
|----------|------|
| `INVALID_TITLE_LENGTH` | 标题长度不符合要求 |
| `INVALID_CONTENT_LENGTH` | 正文长度不符合要求 |
| `INVALID_IMAGE_COUNT` | 图片数量不符合要求（1-18张） |
| `IMAGE_FILE_NOT_FOUND` | 图片文件不存在 |
| `INVALID_IMAGE_FORMAT` | 图片格式不支持 |
| `COLLECTION_NOT_FOUND` | 合集不存在 |
| `INVALID_SCHEDULE_TIME` | 定时发布时间无效 |
| `INVALID_CONTENT_TYPE` | 内容类型声明值无效 |
| `INVALID_VISIBLE_SCOPE` | 可见范围值无效 |

## 功能特性

### 1. 智能标记匹配

`marker_tags` 字段支持智能匹配：
- 系统会自动在"地点"和"用户"两个选项卡中搜索
- 先搜索地点，如果未找到再搜索用户
- 支持模糊匹配，会选择第一个匹配的结果

### 2. 话题标签自动格式化

`tags` 字段不需要手动添加`#`号：
- 输入：`["深圳旅游", "周末打卡"]`
- 系统会自动转换为：`#深圳旅游 #周末打卡`

### 3. 定时发布

`schedule_time` 支持 ISO 8601 格式：
- 完整格式：`2026-01-27T10:30:00+08:00`
- 简化格式：`2026-01-27T10:30:00Z`
- 必须是未来时间

### 4. 可见范围控制

支持细粒度的可见范围控制：
- **公开可见**：所有人可见
- **仅自己可见**：私密笔记
- **仅互关好友可见**：互相关注的好友可见
- **只给谁看**：指定用户可见（需要额外参数）
- **不给谁看**：屏蔽指定用户（需要额外参数）

## 使用注意事项

1. **图片路径**：必须是有效的文件路径，支持绝对路径和相对路径
2. **合集功能**：必须先在小红书APP端创建合集，才能在发布时关联
3. **定时发布**：定时时间必须是未来时间，且不能超过平台限制
4. **标记功能**：标记的用户必须是小红书平台上存在的用户
5. **原创声明**：声明原创后将获得原创标记和平台保护
6. **内容类型**：AI生成内容建议声明为"AI合成"

## 版本历史

- **v1.0.0** (2026-01-25): 初始版本，支持完整的图文发布功能

## 相关文档

- [功能清单](./features.md)
- [开发文档](../README.md)
