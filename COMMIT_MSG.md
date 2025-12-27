feat(ctx): 重构响应发送机制，统一使用 SendXX 前缀方法

- 在 Ctx 中新增 SendJSON, SendXML, SendText, SendHTML, SendFile, SendDownload 方法
- 移除冗余的 Body 包装器及 body.go 文件
- 简化 autoFormat 逻辑，实现更直观的响应分发
- 同步更新 README.md 文档及示例代码
