# 发布视频笔记 API

## 概述

发布视频笔记到小红书平台。

## 接口信息

**模块**: `publish_video.go`
**功能**: 上传视频并发布视频笔记

## JSON Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "PublishVideoNote",
  "description": "发布小红书视频笔记",
  "type": "object",
  "required": ["title", "content", "video_path"],
  "properties": {
    "title": {
      "type": "string",
      "description": "视频标题（1-20字符）",
      "minLength": 1,
      "maxLength": 20,
      "examples": ["深圳美食探店", "周末Vlog"]
    },
    "content": {
      "type": "string",
      "description": "视频正文内容（1-1000字符）",
      "minLength": 1,
      "maxLength": 1000,
      "examples": ["今天带大家探店深圳湾附近的网红餐厅~"]
    },
    "video_path": {
      "type": "string",
      "description": "视频文件路径（支持mp4、mov等格式）",
      "examples": ["/path/to/video.mp4"]
    },
    "tags": {
      "type": "array",
      "description": "话题标签列表（可选，最多10个）",
      "items": {
        "type": "string"
      },
      "maxItems": 10,
      "examples": [["深圳美食", "探店", "周末vlog"]]
    },
    "schedule_time": {
      "type": "string",
      "description": "定时发布时间（可选，ISO 8601格式）",
      "format": "date-time",
      "examples": ["2026-01-27T10:30:00+08:00", null]
    }
  }
}
```

## 字段说明

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `title` | string | ✅ | - | 视频标题，1-20字符 |
| `content` | string | ✅ | - | 视频正文，1-1000字符 |
| `video_path` | string | ✅ | - | 视频文件的本地路径 |
| `tags` | array | ❌ | [] | 话题标签，最多10个 |
| `schedule_time` | datetime | ❌ | null | 定时发布时间，null表示立即发布 |

## 调用示例

### 示例 1: 立即发布视频

```json
{
  "title": "深圳美食探店",
  "content": "今天带大家探店深圳湾附近的网红餐厅，味道真的绝了！",
  "video_path": "/Users/username/Videos/shenzhen_food.mp4",
  "tags": ["深圳美食", "探店", "美食推荐"]
}
```

### 示例 2: 定时发布

```json
{
  "title": "周末Vlog",
  "content": "记录这个周末的美好时光~",
  "video_path": "/Users/username/Videos/weekend_vlog.mp4",
  "tags": ["周末vlog", "生活记录"],
  "schedule_time": "2026-01-27T09:00:00+08:00"
}
```

### 示例 3: 无标签发布

```json
{
  "title": "随手拍",
  "content": "随便拍拍，记录生活",
  "video_path": "/path/to/casual_video.mp4"
}
```

## 响应格式

### 成功响应

```json
{
  "success": true,
  "note_id": "65f8a3b2c4d1e5f6a7b8c9d0",
  "url": "https://www.xiaohongshu.com/explore/65f8a3b2c4d1e5f6a7b8c9d0",
  "message": "视频发布成功"
}
```

### 失败响应

```json
{
  "success": false,
  "error_code": "VIDEO_UPLOAD_FAILED",
  "message": "视频上传失败",
  "details": {
    "reason": "视频格式不支持"
  }
}
```

## 错误代码

| 错误代码 | 说明 |
|----------|------|
| `INVALID_TITLE_LENGTH` | 标题长度不符合要求 |
| `INVALID_CONTENT_LENGTH` | 正文长度不符合要求 |
| `VIDEO_FILE_NOT_FOUND` | 视频文件不存在 |
| `VIDEO_FORMAT_NOT_SUPPORTED` | 视频格式不支持 |
| `VIDEO_TOO_LARGE` | 视频文件过大 |
| `VIDEO_UPLOAD_FAILED` | 视频上传失败 |
| `INVALID_SCHEDULE_TIME` | 定时发布时间无效 |

## 使用限制

1. **视频格式**: 支持 MP4、MOV 等常见格式
2. **视频大小**: 建议不超过 500MB
3. **视频时长**: 建议 15秒-5分钟
4. **上传时间**: 视频上传可能需要较长时间，请耐心等待

## 注意事项

1. 视频文件必须存在于本地文件系统
2. 视频上传过程中请勿关闭程序
3. 定时发布时间必须是未来时间
4. 标签不需要添加 `#` 号，系统会自动添加

## 相关文档

- [发布图文 API](./00-publish-image.md)
- [数据分析 API](./10-data-analytics.md)
