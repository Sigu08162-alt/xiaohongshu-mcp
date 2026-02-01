## ADDED Requirements
### Requirement: 页面发现必须基于真实链接
扫描工具 MUST 仅输出页面中真实存在的链接，不得通过硬编码/推断构造URL。

#### Scenario: 仅保留真实链接
- **WHEN** 页面发现器运行
- **THEN** 输出链接 MUST 来自页面DOM中的真实链接
- **AND THEN** 不得输出 fallback/推断URL

### Requirement: 采集器必须显式指定页面来源
组件采集器 MUST 要求传入发现页面列表文件；缺失输入 MUST 报错。

#### Scenario: 未提供发现文件
- **WHEN** 未提供发现页面文件
- **THEN** 工具 MUST 失败并提示先运行 discover_pages

### Requirement: 采集器必须显式指定输出文件
组件采集器与元数据采集器 MUST 要求传入输出文件路径；缺失输出 MUST 报错。

#### Scenario: 未提供输出文件
- **WHEN** 未提供输出文件参数
- **THEN** 工具 MUST 失败并提示指定输出文件

### Requirement: 采集交互元素为默认策略
元数据采集 MUST 以交互/语义元素为默认采集范围。

#### Scenario: 采集交互元素
- **WHEN** 采集器遍历页面
- **THEN** 输出 SHOULD 以输入、按钮、可编辑、语义元素为主
