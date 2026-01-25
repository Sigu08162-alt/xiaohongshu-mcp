# 用户资料 API

## 概述

获取小红书用户的主页信息和资料数据。

## 接口信息

**模块**: `user_profile.go`
**功能**: 获取用户主页、获取自己的资料

## JSON Schema

### 1. 获取用户主页

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "GetUserProfile",
  "description": "获取用户主页信息",
  "type": "object",
  "required": ["user_id", "xsec_token"],
  "properties": {
    "user_id": {
      "type": "string",
      "description": "用户ID",
      "examples": ["5f7e1234567890abcdef"]
    },
    "xsec_token": {
      "type": "string",
      "description": "安全令牌",
      "examples": ["abc123def456"]
    }
  }
}
```

### 2. 获取我的资料

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "GetMyProfile",
  "description": "获取自己的主页信息",
  "type": "object",
  "properties": {}
}
```

## 字段说明

### 获取用户主页

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `user_id` | string | ✅ | 用户ID |
| `xsec_token` | string | ✅ | 安全令牌 |

### 获取我的资料

无需参数，直接调用。

## 调用示例

### 示例 1: 获取指定用户主页

```json
{
  "user_id": "5f7e1234567890abcdef",
  "xsec_token": "abc123def456"
}
```

### 示例 2: 获取我的资料

```json
{}
```

## 响应格式

### 成功响应

```json
{
  "success": true,
  "user_basic_info": {
    "user_id": "5f7e1234567890abcdef",
    "nickname": "美食探店小王",
    "avatar": "https://...",
    "desc": "深圳美食博主 | 探店达人",
    "gender": 1,
    "ip_location": "广东",
    "red_official_verified": false,
    "verified_content": "",
    "follows": "256",
    "fans": "1.2w",
    "interaction": "5.6w"
  },
  "interactions": [
    {
      "type": "follows",
      "count": "256",
      "name": "关注"
    },
    {
      "type": "fans",
      "count": "1.2w",
      "name": "粉丝"
    },
    {
      "type": "interaction",
      "count": "5.6w",
      "name": "获赞与收藏"
    }
  ],
  "feeds": [
    {
      "id": "65f8a3b2c4d1e5f6a7b8c9d0",
      "title": "深圳美食探店",
      "type": "normal",
      "cover": {
        "url": "https://...",
        "width": 1080,
        "height": 1440
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

### 响应字段说明

#### user_basic_info（用户基本信息）

| 字段 | 类型 | 说明 |
|------|------|------|
| `user_id` | string | 用户ID |
| `nickname` | string | 昵称 |
| `avatar` | string | 头像URL |
| `desc` | string | 个人简介 |
| `gender` | integer | 性别（0=未知，1=男，2=女） |
| `ip_location` | string | IP归属地 |
| `red_official_verified` | boolean | 是否官方认证 |
| `verified_content` | string | 认证信息 |
| `follows` | string | 关注数 |
| `fans` | string | 粉丝数 |
| `interaction` | string | 获赞与收藏数 |

#### interactions（互动数据）

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | string | 数据类型（follows/fans/interaction） |
| `count` | string | 数量 |
| `name` | string | 显示名称 |

#### feeds（笔记列表）

用户发布的笔记列表，字段同 [Feed列表 API](./07-feeds.md)。

### 失败响应

```json
{
  "success": false,
  "error_code": "USER_NOT_FOUND",
  "message": "用户不存在"
}
```

## 错误代码

| 错误代码 | 说明 |
|----------|------|
| `USER_NOT_FOUND` | 用户不存在 |
| `INVALID_XSEC_TOKEN` | 安全令牌无效 |
| `USER_PRIVATE` | 用户设置为私密 |
| `USER_BLOCKED` | 用户已被封禁 |
| `PAGE_LOAD_FAILED` | 页面加载失败 |

## 功能特性

### 1. 完整资料

获取的信息包括：
- 基本信息（昵称、头像、简介）
- 互动数据（关注、粉丝、获赞）
- 认证信息
- 已发布的笔记

### 2. 隐私保护

根据用户隐私设置：
- 公开账号：任何人可查看
- 私密账号：需要关注才能查看
- 部分信息可能隐藏

### 3. 实时数据

- 数据实时更新
- 反映最新状态
- 包含最新发布的笔记

## 数据格式

### 数字缩写

小红书对大数字采用缩写：
- `1.2w` = 12000
- `5000+` = 超过5000
- `10w+` = 超过100000

### 性别编码

| 值 | 含义 |
|----|------|
| 0 | 未知/未设置 |
| 1 | 男 |
| 2 | 女 |

## 使用场景

### 查看用户资料

- 了解创作者背景
- 查看发布内容
- 评估账号影响力
- 决定是否关注

### 获取自己资料

- 查看账号数据
- 管理个人信息
- 检查笔记状态
- 数据分析

## 认证类型

小红书支持多种认证：

| 认证类型 | 说明 | 标识 |
|----------|------|------|
| **官方认证** | 品牌、机构官方账号 | 红色V |
| **达人认证** | 优质创作者 | 黄色V |
| **职业认证** | 特定职业身份 | 蓝色V |
| **企业认证** | 企业账号 | 企业标识 |

## 隐私设置

用户可以设置：

1. **账号隐私**
   - 公开账号
   - 私密账号（需审核关注）

2. **内容可见性**
   - 公开可见
   - 仅粉丝可见
   - 仅互关可见

3. **互动权限**
   - 允许评论
   - 允许私信
   - 允许@

## 数据分析价值

从用户资料可以分析：

### 账号质量指标

- **粉丝数**: 账号影响力
- **获赞数**: 内容质量
- **关注/粉丝比**: 账号定位
- **笔记数量**: 活跃度
- **更新频率**: 运营能力

### 内容方向

- **笔记类型**: 图文/视频比例
- **话题分布**: 内容领域
- **互动数据**: 受欢迎程度
- **发布规律**: 运营策略

## 最佳实践

### 查看他人资料

1. **尊重隐私**: 不要过度查看
2. **合理使用**: 仅用于正当目的
3. **真实互动**: 建立真实联系

### 管理自己资料

1. **完善信息**: 填写完整资料
2. **优质头像**: 使用清晰头像
3. **精炼简介**: 突出个人特色
4. **定期更新**: 保持资料时效性

## 注意事项

1. **隐私保护**: 不泄露他人隐私信息
2. **数据时效**: 数据会实时变化
3. **权限限制**: 私密账号需要权限
4. **合规使用**: 遵守平台规则

## 相关文档

- [关注 API](./06-follow.md)
- [获取Feed列表 API](./07-feeds.md)
- [数据分析 API](./10-data-analytics.md)
