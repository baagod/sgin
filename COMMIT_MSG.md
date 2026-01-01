feat: 集成泛型 JWT 认证组件

- 新增 `jwt.go` 实现基于 `golang-jwt/jwt/v5` 的泛型认证管理器 `JWT[T]`。
- 引入 `ClaimsValidator` 接口，支持“全上下文”业务校验（访问 `RegisteredClaims`）。
- `NewJWT` 采用 Functional Options 模式，强制核心参数（Key/Secret/Timeout）并提供无限扩展性。
- 更新 `README.md`，添加完整的 JWT 认证使用文档。
