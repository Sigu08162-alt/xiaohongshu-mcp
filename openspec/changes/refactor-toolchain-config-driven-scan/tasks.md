## 1. Implementation
- [x] 1.1 移除 discover_pages 中 fallback URL 逻辑，确保仅保留真实链接
- [x] 1.2 移除 refresh_selectors 默认页面列表，要求 --pages 输入（缺失即报错）
- [x] 1.3 更新 collect_all.sh 与 toolchain_v2.sh，确保传入发现文件
- [x] 1.4 更新相关文档/提示信息，强调必须先 discover_pages
- [x] 1.5 运行基础构建或局部测试验证
