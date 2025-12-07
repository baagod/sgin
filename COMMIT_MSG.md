refactor(core): 简化错误处理机制，移除 APIError 接口

- 移除 `APIError` 接口，统一使用 `*sgin.Error` 结构体，以保持与主流框架（如 Fiber）的一致性和简洁性。
- 移除 `Error` 结构体的 `Status()` 方法，直接使用 `Code` 字段。
- 将所有预定义错误（如 `ErrBadRequest`）转换为函数形式，支持自定义错误消息。
- 修正 `example/main.go` 中的错误处理逻辑，适配新的错误结构。
- 修正 `NewError` 的默认消息逻辑，使用 `http.StatusText(code)`。