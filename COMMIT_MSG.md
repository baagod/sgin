refactor(oa): 重构 Schema 注册机制并优化 OpenAPI 命名与转换逻辑

- 引入 `Registry` 注册表，采用“先占位后填充”策略解决结构体递归引用导致的死循环问题。
- 升级 `DefaultSchemaNamer`：支持自动识别并修剪主模块及 `main` 包路径前缀，移除冗余后缀。
- 增强 `helper.Convert`：引入 `ConvertibleTo` 性能优化，统一多级指针与切片递归转换路径。
- 优化 OpenAPI 规范适配：引入基于 `omitempty` 的智能可空性推断，修复序列化副作用。
- 改进工具函数：重构 `UpperFirst` 以支持 UTF-8 安全处理，新增注册表单元测试。

此次变更显著提升了文档生成的精准度与系统健壮性，使生成的 Schema 命名更符合业务直觉。
