feat: 新增 YAML/TOML 支持并标准化 MIME 类型

- 在 `helpers.go` 中删除所有未使用的 `MIME...UTF8` 常量（现代 Gin 已自动处理 UTF-8）
- 将 `MIMETextYAML` 从 `text/yaml` 改为标准的 `application/yaml`
- 新增 `MIMETextYAMLX` (`application/x-yaml`) 以兼容旧客户端
- 新增 `MIMETOML` (`application/toml`) 支持
- 在 `ctx.go` 中新增 `SendYAML` 和 `SendTOML` 方法
- 扩展 `autoFormat` 内容协商范围，支持 YAML 和 TOML
- 在 `engine.go` 中使用标准化的 MIME 类型常量
