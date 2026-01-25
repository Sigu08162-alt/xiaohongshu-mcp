## ADDED Requirements

### Requirement: 支持发布图文地点与标记
系统 SHALL 在 `POST /publish` 请求中支持可选字段 `location` 与 `marker_tags`，并在发布时应用到图文内容。

#### Scenario: 传入地点与标记成功发布
- **WHEN** 用户在发布图文请求中提供 `location` 与 `marker_tags`
- **THEN** 系统应尝试设置地点与标记并完成发布

#### Scenario: 未传入地点与标记
- **WHEN** 用户未提供 `location` 或 `marker_tags`
- **THEN** 系统应保持现有发布流程不变
