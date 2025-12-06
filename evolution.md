# sgin Evolution: V2 架构演进蓝图

## 1. 核心愿景与设计哲学

**目标**：打造 **"Gin 的实用主义类型安全外骨骼"**。
**口号**：*Best of Both Worlds: Gin Performance & Ecosystem, Sgin DX & Auto-Doc.*

在保留 `sgin` V1 (基于反射) 易用性的基础上，引入 V2 (智能反射) 核心，解决以下痛点：
1.  **极致开发体验 (DX)**：采用 `r.Get(path, handler any)` 风格，让开发者用最自然的 Go 语法编写 Handler。
2.  **代码即文档**：自动从结构体 Tag 生成符合 **OpenAPI Specification v3.2.0** 标准的文档。
3.  **高效运行时**：反射仅在启动时进行签名分析和适配器生成，运行时性能接近原生 Gin。
4.  **编译期辅助**：虽不对 `any` 进行编译期类型检查，但提供强大的**启动时签名验证**，实现“快速失败 (Fail Fast)”。
5.  **完全兼容**：原生 `gin.HandlerFunc`、`sgin` V1 反射 Handler、`sgin` V2 智能 Handler 可在同一个路由组中无缝共存。

---

## 2. 架构分层 (The "Sgin Stack")

新架构将分为三层，确保渐进式迁移：

| 层级 | 名称 | 职责 | 关键技术 |
| :--- | :--- | :--- | :--- |
| **L0** | **Native Core** | Gin 原始引擎，处理 HTTP 调度、原生中间件。 | `gin.Engine`, `http.Handler` |
| **L1** | **Legacy Adapter** | 兼容现有的 `sgin` 反射逻辑，处理旧业务。 | `reflect.Value`, `interface{}` |
| **L2** | **Smart Reflection Core** | **(新增)** 智能反射分析 Handler 签名、参数绑定、返回值归一化。 | `reflect`, `Tag Parsing` |

---

## 3. 详细改造方案

### 3.1. 阶段一：智能 Handler 适配器与统一结果处理

**目标**：实现 `r.Get(path string, handler any)` 风格，支持多样的 Handler 签名，并将其返回值统一为内部标准结构。

#### 3.1.1. Handler 签名规范

`sgin` V2 将支持以下 Handler 签名，以兼顾灵活性和规范性：

**L0: Gin 原生 Handler (最高优先级)**
*   `func(*gin.Context)`: 纯 Gin 风格，`sgin` 不做任何介入，直接传递给 Gin。

**L2: Sgin V2 智能 Handler**
*   **入参 (Parameter Options)**:
    1.  `func(c *Ctx)`: 仅传入 `sgin` 上下文。
    2.  `func(c *Ctx, req T)`: 传入 `sgin` 上下文和请求结构体 `T`。`T` 必须是结构体或其指针。
*   **出参 (Return Value Options)**:
    1.  `(T, error)`: **推荐**。最符合 Go 惯例，`T` 为数据体，`error` 为错误。
    2.  `(int, T)`: 支持显式指定 HTTP 状态码和数据体 `T`。
    3.  `T`: 单一返回值，作为数据体。
    4.  `error`: 单一错误返回值。

#### 3.1.2. 统一输入/输出定义 (Single Input Struct)

废弃 `c.Query()`, `c.PostForm()` 的分散调用。
请求参数结构体 `T` 支持以下 Tag 组合进行智能绑定：

```go
// 示例：一个结构体搞定所有请求参数来源
type GetUserReq struct {
    ID        int    `uri:"id" binding:"required" doc:"用户ID"`       // 路径参数 (e.g., /users/:id)
    ShowDetail bool   `form:"detail" default:"false"`    // Query 参数 (e.g., /users?detail=true)
    AuthToken string `header:"Authorization"`           // Header 参数 (e.g., Header["Authorization"])
    ForceUpdate bool `form:"forceUpdate"`               // Body Form 参数 (for POST/PUT form-urlencoded)
    Name      string `json:"name"`                      // Body JSON 参数 (for POST/PUT application/json)
}
```

**智能绑定逻辑 (Smart Binder)**：
框架内部的绑定器将按以下优先级和逻辑执行：
1.  **URI 参数绑定** (`uri` tag): `c.ShouldBindUri(ptr)`。
2.  **Header 参数绑定** (`header` tag): `c.ShouldBindHeader(ptr)`。
3.  **Query 参数绑定** (`form` tag for GET/POST query): `c.ShouldBindQuery(ptr)`。
4.  **Body 参数绑定** (`json`, `xml`, `form` for Body): 根据 `Content-Type` 头，智能调用 `c.ShouldBindJSON`, `c.ShouldBindXML`, `c.ShouldBind` (PostForm)。
    *   **互斥处理**: Body 类型绑定是互斥的，只会执行其中一个。
    *   **GET 请求无 Body**: GET 方法跳过 Body 绑定。

#### 3.1.3. 统一响应结果结构 (Internal Result Normalization)

为了简化后续响应处理，所有 Handler 的返回值（无论是 `(T, error)` 还是 `(int, T)` 等），都将在反射调用后，统一转换为内部 `*result` 结构体。

```go
// 内部使用，不对外暴露
type result struct {
    Status int   // HTTP 状态码，0 表示未设置
    Data   any   // 响应数据 (T)
    Err    error // 错误
}
```

*   **转换逻辑**: 在 Handler 被调用后，将 `[]reflect.Value` 转换为 `*result` 实例。
    *   `[]reflect.Value` -> `*result`
    *   例如：`values = [int(200), UserResp{...}]` -> `result{Status: 200, Data: UserResp{...}, Err: nil}`
    *   `values = [UserResp{}, error]` -> `result{Status: 0, Data: UserResp{}, Err: error}`

*   **消费逻辑**: `Ctx.sendResult(r *result)` 将根据 `r.Status`、`r.Err`、`r.Data` 统一发送 HTTP 响应。

#### 3.1.4. 错误处理标准化

定义统一的 `APIError` 接口，参考 RFC 7807。

```go
type APIError interface {
    error
    Status() int // 获取 HTTP 状态码
    Payload() any // 获取详细错误载荷 (可选)
}
// 开发者只需返回 error 接口的实现
return nil, sgin.ErrNotFound("用户不存在") // -> 框架自动转为 HTTP 404，Body: {"code": 1001, "message": "用户不存在"}
```

---

### 3.2. 阶段二：自动文档引擎 (OpenAPI Generator)

**目标**：应用启动时扫描路由，生成符合 [OpenAPI Specification v3.2.0](https://spec.openapis.org/oas/v3.2.0.html) 标准的 Swagger 文档，无需手写 YAML。

#### 3.2.1. 元数据捕获机制

由于我们使用 `any` 和反射，所有 Handler 的元数据（Input Struct `T` 和 Output Struct `R` 的 `reflect.Type`，以及 `doc` 标签）将在 **应用启动时** 集中提取和存储。

#### 3.2.2. 文档生成流程

1.  在 `Engine.Run()` 之前，遍历所有已注册的 `sgin` V2 智能 Handler。
2.  通过反射获取 Handler 的函数签名、输入结构体 `T` 和输出结构体 `R` 的 `reflect.Type`。
3.  解析 `T` 和 `R` 结构体字段上的 `json`, `uri`, `form`, `header`, `doc` 等 Tag。
4.  将解析结果转换为 OpenAPI v3.2.0 的 Path Item 和 Schema Object。
5.  在 `Engine` 实例中存储生成的 OpenAPI Spec 对象。
6.  提供 `/docs` 和 `/openapi.json` 路由，分别用于渲染 Swagger UI 和暴露原始 Spec 文件。

---

### 3.3. 阶段三：增强型 Context 与可观测性

**目标**：吸收 `gofr` 的优点，但保持轻量。

#### 3.3.1. 瘦身 `Ctx`

*   V2 Handler 中，请求数据都在 `req T` 中，`Ctx` 主要用于获取底层 `gin.Context`、TraceID、Logger、设置响应状态码和头等协议层操作。
*   `ArgInt` 等方法将保留，作为兼容 V1 和快速取参的便利方法。

#### 3.3.2. 默认中间件集成

提供开箱即用的中间件组合（可关闭），自动注入到 `Engine` 级别：
*   **RequestLogger**: 结构化日志 (JSON)，包含 TraceID、耗时、Status、ClientIP。
*   **Recovery**: 增强版 Recovery，捕获 Panic，转换为统一的 `APIError` 响应。
*   **OpenAPI UI**: 默认挂载 `/docs` 和 `/openapi.json` 路由。

---

## 4. 兼容性与迁移策略

这是本次改造的最关键约束。

### 4.1. 混合运行模式

用户可以在同一个项目中混合使用三种风格的 Handler：

```go
func main() {
    r := sgin.New()

    // 风格 1: 原生 Gin (L0) - 完全支持
    r.GET("/native", func(c *gin.Context) {
        c.String(200, "Native Gin Handler")
    })

    // 风格 2: sgin V1 传统反射 (L1) - 兼容支持 (将被 V2 的智能反射优化取代)
    r.GET("/v1-legacy", func(c *sgin.Ctx) string {
        return "V1 Legacy Handler"
    })

    // 风格 3: sgin V2 智能 Handler (L2) - 推荐
    r.GET("/v2-smart", func(c *sgin.Ctx, req GetUserReq) (UserResp, error) {
        // req 会被自动绑定
        // 返回值会被统一处理
        return UserResp{ID: req.ID, Name: "V2 Smart Handler"}, nil
    })
    
    // 甚至可以在一个路由链中混用
    r.GET("/mix", middleware.Auth /*原生 Gin 中间件*/, func(c *sgin.Ctx, req MixReq) (MixResp, error) { /*V2 智能 Handler*/ })
}
```

### 4.2. 对 `handler.go` 和 `router.go` 的修改计划

*   **`router.go`**:
    *   `Router` 接口的 `GET`, `POST`, `PUT`, `DELETE`, `Handle` 等方法签名将保持不变，仍接受 `...Handler` (即 `...any`)。
    *   在这些方法内部，将调用新的适配逻辑。

*   **`handler.go` (核心适配器)**:
    *   原 `handler()` 函数将重构为 V2 核心适配器，负责：
        1.  **Handler 类型识别**:
            *   `func(*gin.Context)`: 识别为 Gin 原生 Handler。
            *   `func(*Ctx[, T]) [RetVals...]`: 识别为 Sgin V2 智能 Handler。
        2.  **启动时签名验证**: 对 Sgin V2 智能 Handler 进行严格的入参/出参数量和类型检查，不符合规范立即 `panic`，并给出清晰的错误信息。
        3.  **适配器生成**: 为每个 Sgin V2 智能 Handler 生成一个**闭包**，这个闭包就是真正的 `gin.HandlerFunc`。这个闭包内部负责：
            *   创建 `sgin.Ctx`。
            *   **智能绑定 (Smart Binder)**：根据 Handler 签名和 `req T` 的 Tag 进行多源绑定。
            *   调用用户 Handler。
            *   **结果归一化**: 将用户 Handler 的返回值转换为内部 `*result` 结构。
            *   调用 `Ctx.sendResult` 发送统一响应。

---

## 5. 实施路线图 (Milestones)

1.  **M1 - Foundation & Adapter**:
    *   实现 `pkg/binding`：统一 Tag 解析器和智能复合绑定逻辑。
    *   实现 `pkg/response`：定义 `result` 结构体和 `convertToResult` 归一化逻辑，以及 `Ctx.sendResult`。
    *   重构 `handler.go`：实现 V2 智能 Handler 识别、启动时签名验证、闭包适配器生成、智能绑定和结果归一化。
2.  **M2 - Integration & Router**:
    *   修改 `router.go` 使其调用新的 `handler.go` 适配器。
    *   确保所有风格的 Handler (L0, L1, L2) 能混合注册并正常工作。
3.  **M3 - Documentation**:
    *   实现 `pkg/openapi`：定义 OpenAPI 结构、实现从 `reflect.Type` 和 `doc` Tag 到 OpenAPI v3.2.0 Schema 的转换器。
    *   在 `Engine` 初始化时触发文档生成和注册 `/docs`, `/openapi.json` 路由。
4.  **M4 - Polish & Observability**:
    *   错误处理标准化：实现 `APIError` 接口和默认错误处理。
    *   增强 `Ctx`：集成 TraceID、Logger 等。
    *   默认中间件：实现 RequestLogger, Recovery 等。

---

## 6. 示例代码 (Future User Experience)

```go
package main

import (
    "github.com/baagod/sgin"
    "github.com/baagod/sgin/middleware" // 假设中间件在 pkg/middleware
)

// 1. 定义输入 (支持多源绑定，自动校验 + 文档)
type GetUserReq struct {
    ID        int    `uri:"id" binding:"required" doc:"用户ID"`
    ShowDetail bool   `form:"detail" default:"false" doc:"是否显示详情"`
    AuthToken string `header:"Authorization" doc:"认证Token"`
    ClientIP  string `json:"clientIP" doc:"客户端IP (仅用于示例，实际不应从Body获取)"` // 也可以有 Body
}

// 2. 定义输出 (自动文档)
type UserResp struct {
    ID      int    `json:"id"`
    Name    string `json:"name"`
    Role    string `json:"role"`
    Message string `json:"message,omitempty"` // 示例：可选字段
}

func main() {
    r := sgin.New(sgin.Config{OpenAPI: true}) // 开启文档，可在 Config 中配置更多选项

    // 3. 注册路由 (V2 风格，Handler 签名灵活多样)
    // 示例 1: 标准 (T, error) 返回
    r.GET("/users/:id", func(c *sgin.Ctx, req GetUserReq) (UserResp, error) {
        if req.ID <= 0 {
            return UserResp{}, sgin.ErrBadRequest("用户ID无效")
        }
        // 假设从 DB 获取用户
        user := UserResp{ID: req.ID, Name: "初见用户", Role: "Admin"}
        return user, nil
    })

    // 示例 2: (int, T) 返回，显式指定状态码 (如 201 Created)
    type CreateUserReq struct {
        Name string `json:"name" binding:"required"`
    }
    r.POST("/users", func(c *sgin.Ctx, req CreateUserReq) (int, UserResp) {
        // 创建用户逻辑
        newUser := UserResp{ID: 101, Name: req.Name, Role: "User"}
        return 201, newUser // 返回 201 Created
    })

    // 示例 3: 只有 Ctx 入参，只有 error 出参 (如删除失败)
    r.DELETE("/users/:id", func(c *sgin.Ctx) error {
        userID := c.Param("id") // 直接从 Ctx 取参数，也兼容 V1 风格
        if userID == "admin" {
            return sgin.ErrForbidden("禁止删除管理员")
        }
        // 删除逻辑
        return nil // 删除成功
    })

    // 示例 4: 只有 Ctx 入参，只有 T 出参 (如健康检查)
    r.GET("/health", func(c *sgin.Ctx) string {
        return "OK"
    })

    r.Run(":8080")
}
