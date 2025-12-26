refactor(core): 深度架构重构与 OpenAPI 迁移完成

1. 核心架构与命名空间：将 oa 子包完整合并至 sgin 根目录，统一 package sgin 声明，彻底消除了跨包循环引用隐患。
2. Handler 与元数据管理：实现泛型强类型处理器 H[I, R]，通过反射 bindV3 完美支持值类型与指针类型自动绑定；hMeta 采用
   Pop/Delete 机制确保元数据立即清理，保障内存安全。
3. 路由系统：升级 IRouter 接口支持 Any、Match 及全量静态文件服务方法；彻底移除 lastOp 状态字段，通过函数式选项模式实现配置原子化，消除了并发注册风险。
4. 响应结果逻辑：利用 Go 赋值语法特性优化 result.go (if r.Status = ...; len(code) > 0)，实现“总是赋值、按需更新”逻辑；完善
   NewXX 与 SetXX 组合模式，支持高效链式调用。
5. OpenAPI 生成：重构 Operation.Clone() 深度拷贝逻辑，确保 Responses map 安全初始化；Registry 与 Schema 逻辑保持高度兼容，精准映射
   Go 复杂类型。
