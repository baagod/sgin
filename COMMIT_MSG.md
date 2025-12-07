fix(docs): 移除 openapi.go 中不准确的 RequestBody 解析逻辑

- 在 parseRequest 函数中，移除了基于 hasJSON 标记的简单 Body 判断。
- 删除了自动设置 op.RequestBody 的代码块，以避免将包含 json 标签的 Query/Form 参数错误地识别为 JSON Body。
- Body 解析逻辑将在后续版本中进一步精细化。