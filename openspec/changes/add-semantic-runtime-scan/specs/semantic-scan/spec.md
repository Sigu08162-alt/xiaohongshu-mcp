## ADDED Requirements
### Requirement: 运行时语义驱动采集
系统 SHALL 基于运行时 DOM 指纹与语义点击计划完成页面进入验证，且所有策略由配置驱动。

#### Scenario: 指纹验证成功
- **WHEN** 点击候选目标后 DOM 指纹变化满足阈值
- **THEN** 系统判定进入成功并记录语义轨迹

#### Scenario: 指纹验证失败
- **WHEN** 点击候选目标后 DOM 指纹变化不满足阈值
- **THEN** 系统执行回退策略并记录失败报告

### Requirement: 可点击性检测
系统 SHALL 在点击前检测目标元素可见性与可点击性。

#### Scenario: 元素不可点击
- **WHEN** 元素不可见或被遮挡
- **THEN** 系统跳过该目标并尝试下一个候选

### Requirement: 可观测输出
系统 SHALL 输出语义轨迹与失败报告，包含点击目标与指纹变化摘要。

#### Scenario: 失败报告记录
- **WHEN** 所有候选目标均失败
- **THEN** 系统记录失败原因与指纹差异摘要
