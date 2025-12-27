feat: 完善处理器包装器分类 (`He/Hn`) 并修复日志中间件 Bug

- 将 `sgin.Hn` 重命名为 `sgin.He` (Handler Error-only)，并新增无返回值的 `sgin.Hn` (Handler-None)
- 结合 `Hn` 包装器彻底修复 `Logger` 中间件重复触发 `Next()` 的严重 Bug。
- 优化 `hMeta.Pop` 逻辑，在 `router.Use` 中显式清理元数据，增强中间件安全性。
- 将 `Result` 结构体字段 `Message` 统一重命名为 `Msg` 并同步更新配套方法 `SetMsg`。
- 同步更新 `README.md` 文档，确保示例代码与新 API 保持一致。
