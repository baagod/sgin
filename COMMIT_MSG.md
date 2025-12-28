feat: 为 HTTP 方法添加颜色高亮并修复类型兼容性

- 修复 `Ctx.Keys` 类型从 `map[string]any` 改为 `map[any]any` 以兼容新版 Gin
- 修复 `Ctx.Get` 方法的 `key` 参数类型为 `any`
- 新增 HTTP 方法颜色映射，支持 `GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS/TRACE`
- 在 `anyMethods` 中移除 `http.MethodConnect`
- 优化 `Logger` 日志格式，统一使用 `cyan` 颜色标识各字段
- 简化 OpenAPI 调试提示信息
