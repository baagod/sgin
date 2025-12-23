perf(handler): 预计算反射类型信息并重构多语言模块

- 优化 `bindV2` 性能：在路由注册时预计算 `isPtr` 和 `baseType`，消除重复的 `t.Kind()` 和 `t.Elem()` 调用
- 重构多语言支持：将 `useTranslator` 从 engine.go 移至 locale.go，提高模块内聚性
- 函数重命名：`handler` → `ginHandlers`，`convertToResult` → `convertResult`，`sendResult` → `send` 以提升可读性
- 提取常量：`validateTags` 数组统一验证标签优先级顺序
- 更新：`.gitignore` 添加 `*.bat` 文件忽略
