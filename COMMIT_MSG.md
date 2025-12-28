feat: 正式发布 v2 版本，全面拥抱 Go 1.24+。

- 升级模块路径为 `github.com/baagod/sgin/v2` 以支持语义化导入。
- 提升最低 Go 版本要求至 1.24.0。
- 更新所有核心依赖（`Gin`, `Cast`, `Validator` 等）至最新版本。
- 同步重构全量内部导入路径及 `README` 示例。
- 执行 `go mod tidy` 深度清理依赖环境。
