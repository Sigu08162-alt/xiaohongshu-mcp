# Change: 工具链扫描/采集配置驱动化（去除硬编码URL）

## Why
当前扫描/采集工具在发现链接与采集页面时仍包含硬编码URL与默认页面列表，违背“全部由采集结果驱动”的目标，导致输出易失真且难以维护。

## What Changes
- **BREAKING**: 扫描工具不再使用 fallback URL 映射；仅使用页面真实链接。
- **BREAKING**: 采集工具不再使用默认页面列表；必须显式提供发现文件或配置来源。
- 交互元素采集保持为默认策略（与现有采集逻辑一致）。
- 对缺失输入进行严格报错（无发现文件即失败）。

## Impact
- Affected specs: toolchain-collector
- Affected code: cmd/discover_pages, cmd/refresh_selectors, collect_all.sh, toolchain_v2.sh
