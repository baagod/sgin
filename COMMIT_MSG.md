feat(core): 重构 OpenAPI 架构并迁移至 sgin 包

- 将 OpenAPI 子包迁移至 sgin 根目录，消除循环引用
- 重构 Handler 系统，从基于反射的动态适配器改为泛型强类型处理器（H/Ho/Hn）
- 新增 HandleMeta 管理元数据，使用局部 map + 立即清理机制
- 路由注册改为只接受单个 Handler，强制区分中间件和处理函数
- 移除 unsafe 操作和全局 handlers map，提升类型安全性
