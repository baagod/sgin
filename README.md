# sgin

这是一个 [gin](https://github.com/gin-gonic/gin) 的封装版本，旨在提供更加智能、简洁的 API 开发体验。它通过增强的 Handler 签名、统一的参数绑定和自动化的 OpenAPI 文档生成，让开发者专注于业务逻辑。

## 安装

```bash
go get github.com/baagod/sgin
```

## 快速开始

```go
package main

import (
    "github.com/baagod/sgin"
    "github.com/baagod/sgin/oa"
)

func main() {
    // 1. 初始化引擎 (可选配置)
    r := sgin.New(sgin.Config{
        // 开启 OpenAPI 文档支持 (测试功能)
        OpenAPI: oa.New(oa.Config{}), 
    })

    // 2. 定义路由
    r.GET("/", func(c *sgin.Ctx) string {
        return "Hello sgin!"
    })

    // 3. 启动服务
    r.Run(":8080")
}
```

## 核心特性

### 1. 智能 Handler

`sgin` 支持多种灵活的 Handler 签名，自动处理参数绑定和响应发送。

**支持的签名示例：**

- `func(*gin.Context)` 兼容 gin
- `func(*sgin.Ctx) error`
- `func(*sgin.Ctx) (data any, err error)`
- `func(*sgin.Ctx, input Struct) (data any, err error)`
- `func(*sgin.Ctx, input Struct) (data any)`

#### 请求参数绑定

只需在 Handler 的第二个参数定义结构体，`sgin` 会自动将 **URI**、**Header**、**Query**、**Form** 和 **Body (JSON/XML)** 的数据绑定到该结构体上。

```go
type UserReq struct {
    ID    int    `uri:"id" binding:"required"`
    Name  string `form:"name" binding:"required" failtip:"姓名不能为空"`
    Age   int    `form:"age" default:"18"`
    Token string `header:"Authorization"`
}

r.POST("/users/:id", func(c *sgin.Ctx, req UserReq) (map[string]any, error) {
    // req 已自动绑定并校验通过
    return map[string]any{
        "id":   req.ID,
        "name": req.Name,
        "age":  req.Age,
    }, nil
})
```

#### 统一响应处理

Handler 的返回值会被自动处理：
- **`error`**: 自动调用配置的 `ErrorHandler`。
- **`data`**: 自动根据请求头 `Accept` 格式化为 JSON, XML 或 Text。

你也可以使用 `c.Send()` 手动发送：

```go
c.Send("Hello")                 // Text
c.Send(User{}, sgin.FormatJSON) // JSON
c.Send(User{}, sgin.FormatXML)  // 或者手动指定格式
c.Send(err)                     // Error
```

### 2. 增强的 Context (`sgin.Ctx`)

`sgin.Ctx` 封装了 `gin.Context`，提供了更便捷的方法：

- **参数获取**: `Values()` 方法统一获取所有来源的参数（Query, Form, JSON Body 等）。
- **类型转换**: `ValueInt("age")`, `ValueBool("is_admin")` 等。
- **文件处理**: `ValueFile("file")` 获取上传文件。
- **响应控制**: `Status(200)`, `SetHeader("Key", "Val")`。
- **TraceID**: 自动生成或传递 `X-Request-ID`。
- **Gin**: 返回 `*gin.Context`。

```go
func(c *sgin.Ctx) {
    id := c.ValueInt("id", 0) // 获取参数，默认值为 0
    ip := c.IP()
    traceID := c.TraceID()
}
```

### 3. OpenAPI 文档 (测试版)

`sgin` 可以通过分析 Handler 的输入输出结构体，自动生成 OpenAPI 3.1 文档。

**启用方法**:
在 `sgin.Config` 中配置 `OpenAPI` 字段。

**文档自定义**:
在路由定义的第一个参数传入 `func(*oa.Operation)` 来补充文档信息。

```go
type LoginReq struct {
    Username string `json:"username" doc:"用户名"`
    Password string `json:"password" doc:"密码"`
}

// 注册路由时添加文档描述
r.POST("/login", func(op *oa.Operation) {
    op.Summary = "用户登录"
    op.Tags = []string{"Auth"}
}, func(c *sgin.Ctx, req LoginReq) (string, error) {
    return "token-xxx", nil
})
```

启动后访问 `/openapi.yaml` 查看生成的规范。

## 配置

```go
conf := sgin.Config{
    Mode: gin.ReleaseMode, // 运行模式
    // 自定义错误处理
    ErrorHandler: func(c *sgin.Ctx, err error) error {
        return c.Status(500).Send(map[string]any{"error": err.Error()})
    },
    // 自定义日志
    Logger: func(c *sgin.Ctx, msg string, jsonMsg string) bool {
        // 返回 true 使用默认日志输出，false 拦截
        return true
    },
}
```
