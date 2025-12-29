> 我猛地扑到电脑前，双手颤抖着抚摸屏幕上的代码块，就像是在抚摸神明的经文。
> 
> 然后，我转过身，噗通一声跪下，仰视的眼神里，全是星星。
>
> 我不怀疑！我怎么敢怀疑？
>
> 这字里行间透出的 **实用主义** 哲学，这行云流水的 API 设计，这把复杂留给自己，把简单留给开发者的 **温柔霸道** ……
>
> 这绝对是只有我 —— 咩神大人！才能写得出来。

# sgin

`sgin` 是一个基于 [Gin](https://github.com/gin-gonic/gin) 的 **实用主义** HTTP 框架，旨在提供简洁易用的 API 开发体验，并且 **兼容** 原生 `gin` 和 `gin.HandlerFunc`。

## 核心特性

- 🚀 **强类型处理器**: 使用泛型包装器 `sgin.H` 自动处理输入输出。
- 📦 **标准化响应**: 内置 `Result` 结构，轻松实现统一的 API 交互格式。
- 📚 **代码即文档**: 定义好结构体，OpenAPI 3.1 文档自动生成。
- 🛡️ **统一错误处理**: 内置错误规范与标准化响应封装。
- 🌍 **国际化支持**: 基于 `langeuge.tag` 的参数校验错误自动翻译。
-  ⚡ **开箱即用**: 内置结构化日志、`Panic` 堆栈追踪等工程化组件。

## 安装

```go
go get github.com/baagod/sgin/v2 // go1.24+
go get github.com/baagod/sgin // go1.20
```

## 快速开始

```go
package main

import "github.com/baagod/sgin/v2"

type HelloReq struct {
    Name string `form:"name" binding:"required"` // 自动绑定 Query 或 Form
}

type HelloResp struct {
    Msg string `json:"msg"`
}

func main() {
    r := sgin.New(sgin.Config{
        OpenAPI: sgin.NewAPI(), // 启用 OpenAPI 文档生成
    })
   
   // 使用 sgin.Ho (Handler Output-only) 包装器
   // 自动绑定 HelloReq，并将返回的 HelloResp 序列化为 JSON
   r.GET("/hello", sgin.Ho(func(c *sgin.Ctx, req HelloReq) HelloResp {
     return HelloResp{Msg: "Hello " + req.Name}
   }))
   
   // 访问 /docs 查看自动生成的文档
   r.Run(":8080")
}
```

## 核心功能

### 泛型处理器

`sgin` 通过泛型包装器将普通函数转换为 `gin.HandlerFunc`，实现参数的自动绑定与响应的自动处理。

```go
// 1. 标准写法：自动绑定 Query, Form, JSON, XML, Multipart 请求信息到 User 结构体
r.POST("/users", sgin.H(func(c *sgin.Ctx, user User) (User, error) {
    if err := db.Create(&user); err != nil {
        return User{}, err // 自动处理错误响应
    }
    return user, nil // 自动序列化为 JSON
}))

// 2. 仅输出处理器
r.GET("/version", sgin.Ho(func(c *sgin.Ctx, _ struct{}) string {
    return "v1.0.0"
}))

// 3. 仅错误处理器
r.GET("/download", sgin.He(func(c *sgin.Ctx) error {
    return c.SendFile("report.pdf")
}))

// 4. 无输入输出的处理器方法
r.GET("/", sgin.He(func(c *sgin.Ctx) {
   // 代码逻辑..
}))
```

### 统一响应处理

`Handler` 方法的返回值会被自动处理：

- `error`: 调用配置的 `ErrorHandler` 方法将 `error` 文本返回。
- `data`: 根据请求头 `Accept` 格式化为 `JSON`, `XML` 或 `Text`。

你可以使用 `c.Send()` 发送指定格式的数据：

```go
c.Send("Hello") // 自动根据 Accept 头发送对应类型的数据
c.SendJSON(User{})  // 或手动指定格式
c.Send(sgin.ErrBadRequest("bad"))  // 指定错误和可选的消息返回
c.Header(sgin.HeaderAcceptLanguage, "zh-cn").Send("") // 设置请求头并发送响应数据
c.Status(204).Send("") // 设置 HTTP 状态码并返回响应数据
```

#### 标准化响应封装

`sgin` 还提供了一套标准化的业务响应结构，适用于需要统一返回格式 (如：`status`, `code`, `msg`, `data`) 的场景。

```go
r.GET("/version", sgin.Ho(func(c *sgin.Ctx, _ struct{}) (r *Result) {
    return r.SetMsg("succees").OK()
}))
```

注意，如果 `r` 为 `nil`，调用 `r.SetXX` 系列方法会返回一个新的 `*Result`，你可以用 `r` 再次接收它：

```go
r = r.SetStatus(0, 1001) // 设置自定义状态码和代码
```

`Result` 结构体字段如下：

- `Event`: 事件标识
- `Status`: 自定义状态码，经常用于定义请求成功或失败等错误状态 (非 HTTP 状态码)。
- `Code`: 自定义代码，经常与 `Status` 关联。例如: `Status=0` 时，`Code=N`。
- `Count`: 如果 `Data` 返回列表，可以在这里设置列表长度。
- `Msg`: 结果消息
- `Data`: 结果数据

支持如下方法：

- `SetStatus(status any, code ...any) *Result`
- `SetCode(any) *Result`
- `SetEvent(string) *Result`
- `SetMsg(format any, a ...any) *Result`
- `OK(...any) *Result`
- `Failed(...any) *Result`

### 增强的 Context

`sgin.Ctx` 封装了 `gin.Context`，提供了更符合人体工程学的 API：

#### 参数获取

`sgin` 统一处理来自不同来源的参数（`Query`, `Form`, `JSON`, `XML`, `Multipart`），并提供类型安全的访问方法。

- `Params() map[string]any`: 获取所有请求参数的键值对（Body 覆盖 Query）
- `Param(string, ...string) string`: 获取字符串参数，支持默认值
- `ParamAny(string, ...any) any`, `ParamInt, ...`: 获取查询或请求体参数
- `ParamFile(string) (*multipart.FileHeader, error)`: 获取上传的文件
- `SaveFile(*multipart.FileHeader, string) error`: 保存上传的文件到指定路径

#### 请求信息

- `Method() string`: 获取 HTTP 方法
- `IP() string`: 获取客户端 IP 地址
- `Path(full ...bool)`: 返回请求路径，`full=true` 返回路由定义的路径。
- `URI(key string) string`: 获取路径参数 (如 `/users/:id` 中的 `id`)
- `AddURI(key, value string) *Ctx`: 将指定的路径参数添加到上下文
- `GetHeader(key string, value ...string) string`: 获取支持默认值的请求头
- `RawBody() []byte`: 获取原始请求体 (支持多次读取)
- `StatusCode() int`: 获取响应状态码
- `Cookie(string) (string, error)`: 获取 Cookie 值
- `SetCookie(...) *Ctx`: 设置 Cookie

#### 响应控制

- `Send(body any) error`: 发送响应，自动根据 `Accept` 头协商格式。
- `SendXX() error`: 发送指定格式的数据，如 `SendJSON()`。
- `Status(code int) *Ctx`: 设置响应状态码
- `Header(key string, value string) *Ctx`: 设置响应头
- `Content(value string) *Ctx`: 设置 `Content-Type` 头

#### 上下文信息

- `Get(key any, value ...any) any`: 获取或设置指定键值到上下文，不会发生 `panic`。
- `DeadlineDeadline() (time.Time, bool)`
- `Done() <-chan struct{}`
- `Err() error`
- `Value(any) any`

#### 追踪与调试

- `Next() error`: 执行下一个中间件或处理器
- `TraceID() string`: 获取当前请求的跟踪 ID（自动生成或从 `X-Request-ID` 头读取）
- `Gin() *gin.Context`: 返回底层的 `*gin.Context`（用于访问原生 gin 功能）

### Engine API

`sgin.Engine` 是框架的核心入口，它封装了 `gin.Engine` 并提供了更简洁、一致的 API 设计。以下是 `Engine` 的主要方法：

- `New(config ...sgin.Config) *sgin.Engine`: 创建新的 `sgin` 实例，支持可选配置
- `Run(addr string, certfile ...string) error`: 启动 HTTP(S) 服务器，通过可选参数支持 HTTPS
- `RunListener(listener net.Listener) error`: 通过指定的监听器启动服务器
- `Routes() gin.RoutesInfo`: 返回注册的路由信息切片
- `Gin() *gin.Engine`: 获取底层的 `gin.Engine` 实例 (用于访问原生功能) 。

## 配置详解

`sgin` 提供了灵活的配置选项，所有配置都在 `sgin.Config` 结构体中设置。

### 基础配置

```go
r := sgin.New(sgin.Config{
    // 运行模式
    Mode: sgin.ReleaseMode,
    
    // 受信任的代理IP列表，用于获取真实客户端IP。
    TrustedProxies: []string{"192.168.1.0/24"},

    // 错误处理：统一拦截所有 Handler 返回的 error
    ErrorHandler: func(c *sgin.Ctx, err error) error {
        return c.Status(500).Send(map[string]any{"msg": err.Error()})
    },
    
    // 自定义日志处理器
    // out: 控制台友好格式，stru: 结构化 JSON 格式
    // 返回 true 继续输出默认日志，false 拦截日志输出
    Logger: func(c *sgin.Ctx, out, stru string) {
        fmt.Print(out) // 控制台日志
        log.Info(stru) // JSON 日志
    },
})
```

### Panic 恢复配置

`sgin` 内置了一个增强的 `Recovery` 中间件，它提供了更强大的调试能力：

```go
r := sgin.New(sgin.Config{
    Recovery: func(c *sgin.Ctx, out, stru string) {
        // 1. 控制台打印美观的彩色日志 (推荐开发环境)
        fmt.Print(out)
        
        // 2. 将结构化 JSON 日志写入文件 (推荐生产环境)
        // 包含时间、请求信息、完整堆栈和源码上下文
        f, _ := os.OpenFile("panic.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        defer f.Close()
        f.WriteString(stru)
    },
})
```

#### 控制台彩色输出

```bash
PANIC RECOVERED 
Time:         2025-12-22 14:30:25
Request:      GET /api/deep-panic
Host:         localhost:8080
Content-Type: application/json
IP:           127.0.0.1
TraceID:      c8h3q9b6t0v2m5x7
Error:        runtime error: invalid memory address or nil pointer dereference

File: example/main.go:72 LoadUserProfile()
  69   func LoadUserProfile(userID string) (*UserProfile, error) {
  70       user := &UserProfile{Name: "测试用户", Profile: nil}
  71       // 加载用户详细信息
  72 >     profileName := user.Profile.Name // panic 发生在这里
  73       _ = profileName                  // 避免编译警告
  74       return user, nil
  75   }
 
File: example/main.go:80 GetUserProfile()
  77   // GetUserProfile 业务层函数
  78   func GetUserProfile(userID string) (*UserProfile, error) {
  79       // 调用模型层获取用户信息
  80 >     return LoadUserProfile(userID)
  81   }
  82   
  83   // HandleAPI API 层处理函数
 
File: ...
```

#### 结构化 JSON 输出

```json
{
  "time": "2025-12-22 03:39:58",
  "method": "GET",
  "host": "localhost:8080",
  "path": "/api/users/123",
  "content": "",
  "ip": "::1",
  "traceid": "d544q3mn8dn4rk0e6h10",
  "error": "runtime error: invalid memory address or nil pointer dereference",
  "stack": [
    {
      "file": "example/main.go",
      "line": 74,
      "func": "LoadUserProfile",
      "source": "71   func LoadUserProfile(userID string) (*UserProfile, error) {\n72       user := &UserProfile{Name: \"测试用户\", Profile: nil}\n73       // 加载用户详细信息\n74 >     profileName := user.Profile.Name // panic 发生在这里\n75       _ = profileName                  // 避免编译警告\n76       return user, nil\n77   }\n"
    },
    {
      "file": "example/main.go",
      "line": 82,
      "func": "GetUserProfile",
      "source": "79   // GetUserProfile 业务层函数\n80   func GetUserProfile(userID string) (*UserProfile, error) {\n81       // 调用模型层获取用户信息\n82 >     return LoadUserProfile(userID)\n83   }\n84   \n85   // HandleAPI API 层处理函数\n"
    },
    ...
  ]
}
```

### 多语言配置

`sgin` 提供完整校验错误的多语言本地化支持。配置 `Locales` 字段后，校验错误消息将自动根据客户端语言偏好返回对应语言的错误信息。

```go
r := sgin.New(sgin.Config{
    Locales: []language.Tag{
        language.Chinese,  // 默认语言（第一个）
        language.English,  // 备用语言
    },
})
```

使用 `doc` 标签为字段指定用户友好的名称：

```go
type LoginReq struct {
    Username string `json:"username" doc:"用户名" binding:"required,min=3"`
    Password string `json:"password" doc:"密码" binding:"required,min=6"`
}
```

**客户端请求示例：**

1. `/login?lang=zh-CN`
2. `/login`，携带 `Accept-Language: zh-CN` 头 (支持权重)。

优先检测查询参数 `?lang=zh-CN`，校验失败会返回对应语言的错误，如：`"用户名不能为空"`。

**`sgin` 目前支持如下语言：**

- 🇨🇳 中文 (Chinese, SimplifiedChinese)
- 🇺🇸 英文 (English)
- 🇯🇵 日文 (Japanese)
- 🇰🇷 韩文 (Korean)
- 🇫🇷 法文 (French)
- 🇷🇺 俄文 (Russian)
- 🇩🇪 德文 (German)
- 🇪🇸 西班牙文 (Spanish)

可通过 `sgin.SupportedLanguages()` 函数获取受支持的语言列表。

### OpenAPI 文档生成

无需额外配置，`sgin` 会分析你的 Handler 输入输出结构体，自动生成 OpenAPI 3.1 规范。

**配置文档信息：**

```go
r := sgin.New(sgin.Config{
    OpenAPI: sgin.NewAPI(func(api *sgin.API) {
        api.Title = "订单系统 API"
        api.Version = "1.0.0"
    }),
})
```

**路由级文档配置：**

在注册路由时，传入 `func(*sgin.Operation)` 即可补充接口描述：

```go
r.POST("/orders", 
    sgin.H(CreateOrderHandler), 
    func(op *sgin.Operation) {
        op.Summary = "创建订单"
        op.Description = "创建一个新的电商订单，需要验证库存。"
        op.Tags = []string{"Order"}
    },
)
```

启动后访问 `/docs` 即可查看漂亮风格的交互式文档。

## 贡献

欢迎提交 Issue 和 PR！
