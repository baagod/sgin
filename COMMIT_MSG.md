feat: 集成泛型 JWT 认证组件

- 新增 `jwt.go` 实现基于 `golang-jwt/jwt/v5` 的泛型认证管理器 `JWT[T]`。
- 引入 `ClaimsValidator` 接口，支持“全上下文”业务校验（访问 `RegisteredClaims`）。
- 重构 `Issue` 链路为 `IssueWithSetup`，采用手动构造 Token 模式，彻底解决 Claims 初始化时序问题。
- 重构 `Parse` 方法，深度集成 `jwt.v5` 错误模型，移除冗余验证，提升健壮性。
- `NewJWT` 采用 Functional Options 模式，强制核心参数（Key/Secret/Timeout）并提供无限扩展性。
- 更新 `README.md`，添加完整的 JWT 认证使用文档。
- 更新 `example/main.go`，演示最新的泛型 API 调用及自定义校验用法。