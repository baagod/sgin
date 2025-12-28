refactor(ctx): 统一响应方法命名规范

- 重命名 `SendBinary` → `SendBytes`: 语义更精准，与 Go `[]byte` 类型对应
- 重命名 `Redirect` → `SendRedirect`: 保持 `Send*` 系列方法命名一致性
