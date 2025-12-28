refactor: 采用 Go 1.24/1.23 特性进行现代化重构

- 在 `locale.go` 中使用 `maps.Keys` 和 `slices.Collect` 简化语言列表获取
- 在 `schema.go` 中引入 `iter.Seq` 实现 `getFields` 迭代器，提升 `registry.go` 的可读性
- 使用 `reflect.TypeFor` (Go 1.22+) 替换旧的反射类型获取方式
- 在 `openapi.go` 中使用 `slices.Clone` 和 `maps.Clone` 优化 `Operation.Clone` 性能
- 修改 `ctx.go` 中 `autoFormat` 逻辑，浏览器请求默认返回 JSON
- 将 `for i := 0; i < n; i++` 替换为 `for i := range n` (api.go、schema.go、helper/helper.go)
- 优化 `AGENTS.md` 和 `Rules.md` 中的系统指令及提交流程规范
