# Change: 添加运行时语义驱动的页面采集

## Why
当前创作者中心为 SPA + 前端路由，菜单无 URL，静态选择器易失效，导致采集成功率低。需要运行时语义驱动与 DOM 指纹验证来提高稳定性。

## What Changes
- 新增运行时语义驱动引擎（DOM 指纹 + 可点击性驱动）
- 新增语义采集配置（锚点、指纹阈值、点击计划、回退策略）
- 采集链路改为“点击 → 指纹验证 → 失败回退”
- 增加可观测输出（语义轨迹与失败报告）

## Impact
- Affected specs: semantic-scan（新增）
- Affected code: discover_pages / refresh_selectors / collect_metadata / configs
