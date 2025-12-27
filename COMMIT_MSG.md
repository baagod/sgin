feat(openapi): 支持隐藏特定路由且默认隐藏内置文档路由

- 在 Operation 中增加 Hidden 标记位
- 新增 APIHidden 装饰器函数用于标记隐藏路由
- 在 api.Register 中实现隐藏逻辑
- 默认隐藏 /openapi.yaml 和 /docs 路由
- 优化 registry.go 中的 Schema 解析逻辑
