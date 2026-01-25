# Change: 发布图文接口新增地点与标记字段

## Why
当前发布图文接口未暴露地点与标记能力，导致发布后仍需手工补全地点标签，影响效率与一致性。

## What Changes
- 为 `POST /publish` 新增 `location` 与 `marker_tags` 两个可选字段。
- 领域模型与用例层透传新字段，发布时设置地点与标记。
- 更新 Swagger 与 API 文档示例。

## Impact
- Affected specs: publish
- Affected code: `service.go`, `internal/domain/publish/content.go`, `internal/app/publish`, `docs/swagger.json`, `docs/swagger.yaml`, `docs/API.md`, `docs/api/00-publish-image.md`
- Backward compatibility: 非破坏性，新增可选字段
