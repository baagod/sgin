# sgin

这是一个 [gin](https://github.com/gin-gonic/gin) 的封装版本，旨在提供更加智能、简洁的 API 开发体验。它通过增强的 Handler 签名、统一的参数绑定、自动化的 OpenAPI 文档生成和多语言校验错误支持，让开发者专注于业务逻辑。

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
    r := sgin.New(sgin.Config{})

    // 2. 定义路由
    r.GET("/", func(c *sgin.Ctx) string {
        return "Hello sgin!"
    })

    // 3. 启动服务
    r.Run(":8080")
}
```

## 核心功能

`sgin` 的核心价值在于提供更加智能、简洁的 API 开发体验。以下是你需要了解的核心功能：

### 智能 Handler 签名

`sgin` 支持多种灵活的 Handler 签名，自动处理参数绑定和响应发送。

**支持的签名示例：**

- `func(*gin.Context)`: 兼容 gin
- `func(*sgin.Ctx) error`
- `func(*sgin.Ctx) (any, error)`
- `func(*sgin.Ctx, input Struct) (any, error)`
- `func(*sgin.Ctx, input Struct) (any)`
- `func(*sgin.Ctx, input *Struct)`: 指针结构体也支持

### 请求参数绑定

只需在 Handler 的第二个参数定义结构体，`sgin` 会自动将 **URI**、**Header**、**Query**、**Form** 和 **Body (JSON/XML)** 的数据绑定到该结构体上。

```go
type UserReq struct {
    ID    int    `uri:"id" binding:"required"`
	Name  string `form:"name" binding:"required" label:"姓名"`
    Age   int    `form:"age" default:"18"`
    Token string `header:"Authorization"`
}

r.POST("/users/:id", func(c *sgin.Ctx, p UserReq) (map[string]any, error) {
    // req 已自动绑定并校验通过
    return map[string]any{"id": p.ID, "name": p.Name, "age": p.Age}, nil
})
```

### 统一响应处理

Handler 的返回值会被自动处理：

- `error`: 自动调用配置的 `ErrorHandler`。
- `data`: 自动根据请求头 `Accept` 格式化为 JSON, XML 或 Text。

你也可以使用 `c.Send()` 手动发送：

```go
c.Send("Hello")                 // Text
c.Send(User{}, sgin.FormatJSON) // JSON
c.Send(User{}, sgin.FormatXML)  // 手动指定格式
c.Send(err)                     // Error
```

### 增强的 Context (`sgin.Ctx`)

`sgin.Ctx` 封装了 `gin.Context`，提供了更便捷、类型安全的 API。以下是所有可用方法的完整指南：

#### 参数获取与类型转换

`sgin` 统一处理来自不同来源的参数（Query、Form、JSON Body、XML、Multipart），并提供类型安全的访问方法。

- `Values() map[string]any`: 获取所有请求参数的键值对（Body 覆盖 Query）
- `Value(string, ...string) string`: 获取字符串参数，支持默认值
- `ValueAny(string, ...any) any`: 获取原始类型的参数值
- `ValueInt(string, ...int), ValueBool, ...`: 获取查询或请求体参数
- `ValueFile(string) (*multipart.FileHeader, error)`: 获取上传的文件
- `SaveFile(*multipart.FileHeader, string) error`: 保存上传的文件到指定路径

#### 请求信息与元数据

- `Method() string`: 获取 HTTP 方法（GET、POST 等）
- `IP() string`: 获取客户端 IP 地址
- `Path(full ...bool) string`: 获取请求路径（`full=true` 返回路由定义路径）
- `Param(key string) string`: 获取路径参数（如 `/users/:id` 中的 `id`）
- `GetHeader(key string, value ...string) string`: 获取请求头，支持默认值
- `RawBody() []byte`: 获取原始请求体（支持多次读取）
- `StatusCode() int`: 获取当前响应状态码

#### 响应控制

- `Send(body any, format ...string) error`: 发送响应，自动根据 Accept 头协商格式
- `SendHTML(name string, data any) error`: 发送 HTML 响应
- `Status(code int) *Ctx`: 设置响应状态码（链式调用）
- `Header(key string, value string) *Ctx`: 设置响应头（链式调用）
- `Content(value string) *Ctx`: 设置 Content-Type 头（链式调用）

**响应格式常量**：
- `sgin.FormatJSON` - 强制返回 JSON 格式
- `sgin.FormatXML` - 强制返回 XML 格式
- `sgin.FormatText` - 强制返回纯文本格式
- `sgin.FormatUpload` - 文件上传
- `sgin.FormatDownload` - 文件下载

#### 上下文存储与中间件

- `Get(key string, value ...any) any`: 获取或设置上下文值，不会发生 `panic`。
- `Next() error`: 执行下一个中间件或处理器

#### Cookie 操作

- `Cookie(name string) (string, error)`: 获取 Cookie 值
- `SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool)`: 设置 Cookie

#### 追踪与调试

- `TraceID() string`: 获取当前请求的跟踪 ID（自动生成或从 `X-Request-ID` 头读取）
- `Gin() *gin.Context`: 返回底层的 `*gin.Context`（用于访问原生 gin 功能）

#### 多语言支持

- `Locale() language.Tag`: 获取当前请求的语言设置
- `SetLocale(locale language.Tag)`: 手动设置请求语言（覆盖自动检测）

#### 使用示例

```go
func(c *sgin.Ctx) {
    // 参数获取与类型转换
    id := c.ValueInt("id")          // 获取整数参数，默认值为 0
    name := c.Value("name", "匿名")  // 获取字符串参数，默认值为 "匿名"
    isAdmin := c.ValueBool("admin") // 获取布尔值参数
    
    // 请求信息
    method := c.Method()    // "GET", "POST" 等
    clientIP := c.IP()      // 客户端 IP
    traceID := c.TraceID()  // 请求跟踪 ID
    
    // 响应控制
    c.Header("X-Custom-Header", "value")
    c.Status(200).Send(map[string]any{
        "id": id,
        "name": name,
        "trace_id": traceID,
    })
    
    // 文件上传处理
    if file, err := c.ValueFile("avatar"); err == nil {
        c.SaveFile(file, "./uploads/avatar.jpg")
    }
    
    // 多语言支持
    locale := c.Locale()
    fmt.Printf("当前请求语言: %v\n", locale)
}
```

### Engine API

`sgin.Engine` 是框架的核心入口，它封装了 `gin.Engine` 并提供了更简洁、一致的 API 设计。以下是 `Engine` 的主要方法：

#### 核心方法

- `New(config ...sgin.Config) *sgin.Engine`: 创建新的 `sgin` 实例，支持可选配置
- `Run(addr string, certfile ...string) error`: 启动 HTTP(S) 服务器，通过可选参数支持 HTTPS
- `RunListener(listener net.Listener) error`: 通过指定的监听器启动服务器
- `Gin() *gin.Engine`: 获取底层的 `gin.Engine` 实例（用于访问原生功能）

#### 使用示例

```go
package main

import (
    "github.com/baagod/sgin"
    "net"
)

func main() {
    // 1. 极简初始化
    app := sgin.New()
    
    // 2. 链式路由定义（继承自 Router）
    app.GET("/", func(c *sgin.Ctx) string {
        return "Hello sgin!"
    })
    
    // 3. 启动 HTTP 服务
    go app.Run(":8080")
    
    // 4. 启动 HTTPS 服务（通过可选参数）
    go app.Run(":8443", "cert.pem", "cert.key")
    
    // 5. 通过监听器启动（灵活部署）
    listener, _ := net.Listen("tcp", ":9090")
    app.RunListener(listener)
    
    // 6. 访问底层 gin（逃生舱模式）
    ginEngine := app.Gin()
    ginEngine.Static("/static", "./assets")
}
```

## 配置详解

`sgin` 提供了灵活的配置选项，所有配置都在 `sgin.Config` 结构体中设置。以下是所有可用配置的详细说明：

### 基础配置

```go
r := sgin.New(sgin.Config{
    // 运行模式 (可选: gin.DebugMode, gin.ReleaseMode, gin.TestMode)
    Mode: gin.ReleaseMode,
    
    // 受信任的代理IP列表，用于获取真实客户端IP
    TrustedProxies: []string{"192.168.1.0/24"},
    
    // 自定义错误处理器
    ErrorHandler: func(c *sgin.Ctx, err error) error {
        // 可以根据错误类型返回不同的状态码和消息
        return c.Status(500).Send(map[string]any{
            "error": err.Error(),
            "code":  500,
        })
    },
    
    // 自定义日志处理器
    // text: 控制台友好格式，json: 结构化JSON格式
    // 返回 true 继续输出默认日志，false 拦截日志输出
    Logger: func(c *sgin.Ctx, text string, json string) bool {
        // 开发环境输出彩色日志
        fmt.Print(text)
        // 生产环境可以记录JSON日志
        // log.Info(json)
        return false // 拦截默认日志输出
    },
})
```

### OpenAPI 配置

`sgin` 可以通过分析 Handler 的输入输出结构体，自动生成 OpenAPI 3.1 文档。启用后，框架会自动生成规范文件和交互式文档页面。

**启用方法**：
```go
import "github.com/baagod/sgin/oa"

r := sgin.New(sgin.Config{
    OpenAPI: oa.New(oa.Config{
        // OpenAPI 规范基本信息
        Info: oa.Info{
            Title:       "我的API",
            Description: "这是一个示例API",
            Version:     "1.0.0",
        },
    }),
})
```

**文档自定义**：
在路由定义的第一个参数传入 `func(*oa.Operation)` 来补充文档信息。

```go
import "github.com/baagod/sgin/oa"

type LoginReq struct {
    Username string `json:"username" doc:"用户名"`
    Password string `json:"password" doc:"密码"`
}

// 注册路由时添加文档描述
r.POST("/login", func(op *oa.Operation) {
    op.Summary = "用户登录"
    op.Tags = []string{"Auth"}
    op.Description = "用户登录接口，返回认证令牌"
}, func(c *sgin.Ctx, req LoginReq) (string, error) {
    // 业务逻辑...
    return "token-xxx", nil
})
```

启动后访问以下URL查看生成的文档：
- `/openapi.yaml` - OpenAPI 规范文件
- `/docs` - 交互式API文档页面

### Panic 恢复配置

`sgin` 内置了一个增强的 Recovery 中间件，相比原生 gin，它提供了更强大的调试能力：

```go
r := sgin.New(sgin.Config{
    // Panic 恢复回调
    Recovery: func(c *sgin.Ctx, logStr, jsonStr string) {
        // 1. 控制台打印美观的彩色日志 (推荐开发环境)
        fmt.Print(logStr)
        
        // 2. 将结构化 JSON 日志写入文件 (推荐生产环境)
        // 包含时间、请求信息、完整堆栈和源码上下文
        f, _ := os.OpenFile("panic.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        defer f.Close()
        f.WriteString(jsonStr + "\n")
    },
})
```

#### **控制台彩色输出**

   ```
   PANIC RECOVERED 
   Time:         2025-12-22 14:30:25
   Request:      GET /api/users/123
   Host:         localhost:8080
   Content-Type: application/json
   IP:           127.0.0.1
   TraceID:      c8h3q9b6t0v2m5x7
   Error:        runtime error: invalid memory address or nil pointer dereference
   
   File: models/user.go:45 LoadUserProfile()
     44        // 加载用户详细信息
     45 >     profile := user.Profile.Name // panic 发生在这里
     46        return &profile, nil
     47      }
   
   File: handlers/user.go:78 GetUserProfile()
     77        // 调用模型层获取用户信息
     78        profile, err := models.LoadUserProfile(userID)
     79        if err != nil {
     80            return nil, err
     81        }
   
   File: handlers/api.go:32 HandleAPI()
     31        // 处理用户API请求
     32        profile := GetUserProfile(userID)
     33        return c.JSON(profile)
     34      }
   
   File: main.go:15 main()
     14        // 启动HTTP服务器
     15        r := sgin.New()
     16        r.GET("/api/users/:id", HandleAPI)
   ```

#### **结构化 JSON 输出**

   ```json
   {
     "time": "2025-12-22 14:30:25",
     "method": "GET",
     "host": "localhost:8080",
     "path": "/api/users/123",
     "content": "application/json",
     "ip": "127.0.0.1",
     "traceid": "c8h3q9b6t0v2m5x7",
     "error": "runtime error: invalid memory address or nil pointer dereference",
     "stack": [
       {
         "file": "models/user.go",
         "line": 45,
         "func": "LoadUserProfile",
         "source": "44        // 加载用户详细信息\n45 >     profile := user.Profile.Name // panic 发生在这里\n46        return &profile, nil\n47      }"
       },
       {
         "file": "handlers/user.go",
         "line": 78,
         "func": "GetUserProfile",
         "source": "77        // 调用模型层获取用户信息\n78        profile, err := models.LoadUserProfile(userID)\n79        if err != nil {\n80            return nil, err\n81        }"
       },
       {
         "file": "handlers/api.go",
         "line": 32,
         "func": "HandleAPI",
         "source": "31        // 处理用户API请求\n32        profile := GetUserProfile(userID)\n33        return c.JSON(profile)\n34      }"
       },
       {
         "file": "main.go",
         "line": 15,
         "func": "main",
         "source": "14        // 启动HTTP服务器\n15        r := sgin.New()\n16        r.GET(\"/api/users/:id\", HandleAPI)"
       }
     ]
   }
   ```

### 多语言配置

`sgin` 提供完整的校验错误多语言本地化支持。配置 `Locales` 字段后，校验错误消息将自动根据客户端语言偏好返回对应语言的错误信息。

**基础配置**：
```go
import (
    "github.com/baagod/sgin"
    "golang.org/x/text/language"
)

r := sgin.New(sgin.Config{
    Locales: []language.Tag{
        language.Chinese,  // 默认语言（第一个）
        language.English,  // 备用语言
        // 可选：language.Japanese, language.Korean, language.French, 
        // language.Russian, language.German, language.Spanish
    },
})
```

**字段标签**：使用 `label` 标签为字段指定用户友好的名称。
```go
type LoginReq struct {
    Username string `json:"username" label:"用户名" binding:"required,min=3"`
    Password string `json:"password" label:"密码" binding:"required,min=6"`
}
```

**语言检测优先级**：
1. 查询参数 `?lang=zh-CN`
2. `Accept-Language` 请求头（支持权重）
3. 配置的第一个语言（默认）

**完整示例**：
```go
r.POST("/login", func(c *sgin.Ctx, req LoginReq) error {
    // 业务逻辑...
    return nil
})

// 客户端请求示例：
// POST /login?lang=zh-CN
// POST /login (携带 Accept-Language: zh-CN 头)
// 校验失败返回对应语言错误，如："用户名不能为空"
```

**支持的语言**：
`sgin` 目前支持以下语言：
- 🇨🇳 中文 (Chinese, SimplifiedChinese)
- 🇺🇸 英文 (English)
- 🇯🇵 日文 (Japanese)
- 🇰🇷 韩文 (Korean)
- 🇫🇷 法文 (French)
- 🇷🇺 俄文 (Russian)
- 🇩🇪 德文 (German)
- 🇪🇸 西班牙文 (Spanish)

可以通过 `sgin.SupportedLanguages()` 函数获取框架支持的所有语言标签。
