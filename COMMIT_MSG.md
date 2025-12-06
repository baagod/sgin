feat(core): 实现 V2 智能 Handler 架构

- 引入 result.go 用于统一响应归一化
- 重构 handler.go 以支持智能反射、启动自检及复合绑定 (bindV2)
- 更新 ctx.go 以通过 sendResult 消费归一化结果
- 在 Engine 中新增 OpenAPI 配置项
- 新增 sgin_test.go 单元测试并更新 test/main.go 示例
- 在 evolution.md 中记录架构演进

此提交标志着 sgin 向“实用主义 V2”架构的转型，支持 r.Get(path, handler any) 语法、复合绑定（URI/Header/Query/Body）及统一错误处理。