feat(core): 引入 `CORS` 支持并重构 `Engine` 中间件加载逻辑

- 在 `Config` 结构体中新增 `Cors` 成员，支持集成 `github.com/gin-contrib/cors`。
- 将中间件初始化逻辑重构至 `useMiddleware` 私有方法。
- 将 `defaultConfig` 重命名并导出为 `DefaultConfig`。
- 更新 `README.md` 以包含 `CORS` 配置示例及 `RemoteIP` 方法说明。
- 优化 `AGENTS.md` 的角色定义并精简 `Rules.md`。
