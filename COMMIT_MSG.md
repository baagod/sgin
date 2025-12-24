feat(oa): 重构 Schema 可空类型处理机制

- 在 `Schema` 结构体中添加 `Nullable bool` 字段，标签为 `yaml:"-"`
- 实现 `MarshalYAML()` 方法，当 `Nullable == true` 时将 `Type` 序列化为 `[type, "null"]` 数组
- 简化 `config.go` 中的类型处理逻辑，直接为指针类型设置 `Nullable: true`
- 移除复杂的 `TypeNullable()` 函数，统一使用 Schema.Nullable 字段
- 符合 OpenAPI 3.1 规范，使用数组格式表示可空类型而非单独的 nullable 字段

此次重构使可空类型处理更加简洁、统一，并符合最新的 OpenAPI 规范标准。