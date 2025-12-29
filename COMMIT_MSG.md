refactor(ctx): 重构 `Path` 逻辑与 `URI` 参数处理

- `Path()`: 逻辑行为调整为默认返回真实请求路径，传递 `full=true` 获取路由定义模式
- `URI` / `AddURI`: 统一大写命名规范，内部实现切换至底层的 `Param` 与 `AddParam` 方法
- `SetCookie`: 调整参数签名顺序，将 `path` 和 `domain` 置于 `maxAge` 之前
- `README.md`: 同步更新相关 API 描述、安装建议及代码示例
- 规范化 `ctx.go` 中的内部注释章节划分
