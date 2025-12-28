refactor: 优化 `autoFormat` 内容协商逻辑，简化判断条件

- 改用 `strings.HasPrefix` 检查 `Accept` 头前缀是否为 `text/html`，精准识别浏览器直接访问。
- 删除手动的 XML 和类型判断逻辑，统一使用 Gin `Negotiate` 进行标准内容协商。
- 浏览器直接访问时返回 JSON 便于调试，其他情况按 `Accept` 头偏好响应。
