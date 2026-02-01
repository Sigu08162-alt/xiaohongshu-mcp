## ADDED Requirements

### Requirement: 配置驱动的轮询与等待
MCP 工具链所有轮询与等待行为 MUST 由配置驱动，且按模块拆分配置。

#### Scenario: 发布模块等待
- **WHEN** 发布模块执行上传完成等待
- **THEN** 使用发布模块配置的超时与轮询间隔

#### Scenario: 互动模块等待
- **WHEN** 互动模块执行状态轮询
- **THEN** 使用互动模块配置的超时与轮询间隔

### Requirement: 配置缺失即失败
系统 MUST 在启动时校验轮询/等待配置完整性，缺失即报错并拒绝启动。

#### Scenario: 配置缺失
- **WHEN** 任何模块的必填轮询配置缺失
- **THEN** 启动失败并返回明确错误信息

## MODIFIED Requirements

### Requirement: 运行时默认值策略
系统 MUST 禁止使用硬编码时间或默认兜底值作为轮询/等待策略。

#### Scenario: 未配置默认值
- **WHEN** 轮询/等待配置未提供
- **THEN** 运行时不得使用代码内默认值
