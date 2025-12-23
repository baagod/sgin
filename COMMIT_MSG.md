enhance(ctx): 扩展Get方法支持多级查找并更新示例

- 增强`Get()`方法：支持两级查找（先 `gin.Get`，再 `gin.Value`），提升上下文值访问能力
- 保持API兼容：`Get` 方法签名不变，增强功能而不破坏现有代码

此次修改使 `sgin.Ctx.Get()` 能够访问到通过 `gin.Context.Value()` 设置的值，
同时示例代码展示了如何实现类型安全的参数绑定高阶函数。
