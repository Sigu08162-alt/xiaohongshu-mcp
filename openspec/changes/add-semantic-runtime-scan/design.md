## Context
采集链路在 SPA 菜单场景下依赖 URL 与静态 selector，导致点击失败与页面误判。需要引入运行时语义驱动与 DOM 指纹验证。

## Goals / Non-Goals
- Goals: 提高页面进入成功率；完全配置驱动；可观测、可回退
- Non-Goals: 100% 绝对成功率；依赖硬编码 URL/关键词

## Decisions
- Decision: 使用 DOM 指纹作为主验证；语义锚点为辅助定位
- Alternatives considered: 仅网络请求驱动（不稳定且难泛化）

## Risks / Trade-offs
- 风险: 指纹误判导致误报成功 → 通过阈值与多指标降低风险
- 权衡: 增加复杂度换取稳定性

## Migration Plan
- 先引入配置与指纹模块
- 再接入 refresh_selectors/collect_metadata
- 最后接入 discover_pages

## Open Questions
- 指纹阈值默认值如何设定
- 是否需要针对不同页面类型配置不同锚点集
