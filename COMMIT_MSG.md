feat(ctx): 重构参数方法命名并新增流控制与响应方法

- 重构 URI 参数获取：`Param()` → `Uri()`，字段名 `Params` → `Uris`
- 重构请求参数获取：`Value()` 系列方法 → `Param()` 系列方法
  - `ParamAny()`, `ParamInt()`, `ParamBool()` 等类型安全方法
  - 新增 `ParamArray()`, `ParamMap()` 支持多值和映射参数
- 优化 `Params()` 方法：统一使用 `MultipartForm()` 解析，支持 Body 覆盖 Query
- 新增请求信息方法：`AddUri()`, `RemoteIP()`
- 增强 Cookie 控制：`SetCookie()` 支持 `SameSite` 可选参数
- 新增响应方法：
  - `SendSSEvent()`: 服务器发送事件
  - `SendReader()`: 从 `io.Reader` 发送数据
  - `Redirect()`: HTTP 重定向
  - `SendBinary()`: 发送二进制数据
- 实现 `context.Context` 接口：`Deadline()`, `Done()`, `Err()`, `Value(key any)`
- 新增 `Content()` 辅助方法：设置 `Content-Type` 响应头
- 同步 README 文档：更新所有示例代码和 API 说明
