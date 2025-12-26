docs(openapi): 优化 OpenAPI 生成逻辑并完善中文注释

1. 逻辑文档化：为 `api.go` 中的核心解析函数（Register, parseRequestParams 等）添加了详尽的中文注释，明确了标签映射与解析流程。
2. 标签处理优化：增强了 `form` 标签在 OpenAPI query 参数中的映射逻辑，并实现了标签向顶层 `OpenAPI.Tags` 的自动同步。
3. 智能响应注入：改进 `parseResponseBody` 逻辑，仅在未定义 200 响应时自动注入，支持多状态码共存。
4. 示例与辅助：更新 `example/main.go` 以适配最新的路由配置接口；在 `openapi.go` 中新增 `Schema` 辅助方法并统一接收者命名。
