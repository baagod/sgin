feat(ctx): 扩展 Ctx 参数获取方法并增强默认值支持

- 在 `Ctx` 中增加 `ValueInt8/16/32/64`, `ValueUint/8/16/32/64`, `ValueTime`, `ValueDuration` 等类型安全的获取方法
- 将 `Ctx.Value` 的默认值参数类型从 `string` 修改为 `any`，提升灵活性。
- 优化 `README.md` 中的示例代码和描述
