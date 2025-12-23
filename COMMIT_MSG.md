refactor(openapi): 重构模块架构并简化Logger回调

- 重构 OpenAPI 模块：将配置和业务逻辑迁移到 Config 结构体，保持 OpenAPI 为纯净规范字段
- 修改 Logger 回调签名：从返回 `bool` 改为无返回值，简化接口设计
- 更新 `router` 和 `engine`：适应 `Config` 类型变更，确保注册功能正常工作
- 优化文档和示例：更新 `README` 和示例代码，展示新 API 使用方式
- 修复工具调用：修正 `AGENTS.md` 中的 git 命令引用

BREAKING CHANGE: `Logger` 回调函数签名变更，从`func(c *Ctx, out, s string) bool` 改为
`func(c *Ctx, out, s string)`，现有实现需要移除返回值。
