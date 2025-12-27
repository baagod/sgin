# sgin

`sgin` 是一个基于 [Gin](https://github.com/gin-gonic/gin) 的**实用主义** Web 框架，旨在提供简洁易用的 API 开发体验。

它通过增强 **处理器方法**、**自动化参数绑定**、**统一错误处理** 以及 **代码即文档** 的核心能力，**并且兼容原生 `gin`、`gin.HandlerFunc` (包括中间件)**。

## 核心特性

- 🚀 **强类型处理器**: 告别 `c.ShouldBind`，使用泛型包装器 `sgin.H` 自动处理输入输出。
- 📚 **代码即文档**: 定义好结构体，OpenAPI 3.1 文档自动生成，无需手写 YAML。
- 🛡️ **统一错误处理**: 内置错误规范与标准化响应封装。
- 🌍 **国际化支持**: 基于 `langeuge.tag` 的参数校验错误自动翻译。
- ⚡  **开箱即用**: 内置结构化日志、`Panic` 堆栈追踪、跨域处理等工程化组件。

## 安装

```bash
go get github.com/baagod/sgin
```

## 快速开始

```go
package main

import "github.com/baagod/sgin"

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
// 1. 标准写法：自动绑定 JSON/Form 到 User 结构体
r.POST("/users", sgin.H(func(c *sgin.Ctx, user User) (User, error) {
    if err := db.Create(&user); err != nil {
        return User{}, err // 自动处理错误响应
    }
    return user, nil // 自动序列化为 JSON
}))

// 2. 仅输出：适合查询类接口
r.GET("/version", sgin.Ho(func(c *sgin.Ctx, _ struct{}) string {
    return "v1.0.0"
}))

// 3. 仅错误：适合文件下载或不返回数据的中间件处理操作
r.GET("/download", sgin.Hn(func(c *sgin.Ctx) error {
    c.Send(sgin.BodyFile("report.pdf"))
    return nil
}))
```

### 统一响应处理 

`Handler` 的返回值会被自动处理：

- `error`: 调用配置的 `ErrorHandler` 将 `error.Error()` 返回。
- `data`: 根据请求头 `Accept` 格式化为 `JSON`, `XML` 或 `Text`。
  - 若 `Accept` 包含 `application/xml` 且不包含 `text/html`，返回 XML。
  - 若是字符串，返回 `text/plain`。
  - 其他情况默认返回 `application/json`。

你还可以使用 `c.Send()` 发送指定格式的数据：

```go
c.Send("Hello") // 返回文本消息
c.Send(User{})  // 根据请求头 `Accept` 返回对应格式的数据
c.Send(sgin.BodyXML(User{}))       // 手动指定格式
c.Send(sgin.ErrBadRequest("bad"))  // 返回指定的错误状态和可选消息
c.Header(sgin.HeaderAcceptLanguage, "zh-cn").Send("") // 设置请求头并发送数据
```

### 增强的 Context

`sgin.Ctx` 封装了 `gin.Context`，提供了更符合人体工程学的 API：

#### 参数获取

`sgin` 统一处理来自不同来源的参数（`Query`, `Form`, `JSON`, `XML`, `Multipart`），并提供类型安全的访问方法。

- `Values() map[string]any`: 获取所有请求参数的键值对（Body 覆盖 Query）
- `Value(string, ...string) string`: 获取字符串参数，支持默认值
- `ValueAny(string, ...any) any, ValueInt, ...`: 获取查询或请求体参数
- `ValueFile(string) (*multipart.FileHeader, error)`: 获取上传的文件
- `SaveFile(*multipart.FileHeader, string) error`: 保存上传的文件到指定路径

#### 请求信息

- `Method() string`: 获取 HTTP 方法
- `IP() string`: 获取客户端 IP 地址
- `Path(full ...bool) string`: 获取请求路径（`full=true` 返回路由定义路径）
- `Param(key string) string`: 获取路径参数（如 `/users/:id` 中的 `id`）
- `GetHeader(key string, value ...string) string`: 获取支持默认值的请求头
- `RawBody() []byte`: 获取原始请求体 (支持多次读取)
- `StatusCode() int`: 获取响应状态码
- `Cookie(name string) (string, error)`: 获取 Cookie 值
- `SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool)`: 设置 Cookie

#### 响应控制

- `Send(body any) error`: 发送响应，自动根据 `Accept` 头协商格式。
- `Status(code int) *Ctx`: 设置响应状态码
- `Header(key string, value string) *Ctx`: 设置响应头
- `Content(value string) *Ctx`: 设置 `Content-Type` 头

**支持的响应体格式：**

- `sgin.BodyJSON(any)`: 返回 JSON 
- `sgin.BodyXML(any)`: 返回 XML 
- `sgin.BodyText(any)`: 返回纯文本
- `sgin.BodyUpload(any)`: 文件上传
- `sgin.BodyDownload(any)`: 文件下载
- `sgin.BodyHTML(name string, data any)`: 返回 HTML

#### 上下文存储与中间件

- `Get(key string, value ...any) any`: 获取或设置上下文值，不会发生 `panic`。
- `Next() error`: 执行下一个中间件或处理器

#### 追踪与调试

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
    Logger: func(c *sgin.Ctx, out, stru string) bool {
        fmt.Print(out) // 控制台日志
        log.Info(stru) // JSON 日志
        return false   // 拦截默认日志输出
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
 
File: example/main.go:86 HandleAPI()
  83   // HandleAPI API 层处理函数
  84   func HandleAPI(c *sgin.Ctx) {
  85       userID := c.Param("id")
  86 >     profile, err := GetUserProfile(userID)
  87       if err != nil {
  88           c.Send(err)
  89           return
 
File: reflect/value.go:586 call()
  583   }
  584   
  585   // Call.
  586 > call(frametype, fn, stackArgs, uint32(frametype.size), uint32(abid.retOffset), uint32(frameSize), &regArgs)
  587   
  588   // For testing; see TestCallMethodJump.
  589   if callGC {
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
    {
      "file": "example/main.go",
      "line": 88,
      "func": "HandleAPI",
      "source": "85   // HandleAPI API 层处理函数\n86   func HandleAPI(c *sgin.Ctx) {\n87       userID := c.Param(\"id\")\n88 >     profile, err := GetUserProfile(userID)\n89       if err != nil {\n90           c.Send(err)\n91           return\n"
    },
    {
      "file": "reflect/value.go",
      "line": 586,
      "func": "call",
      "source": "583   }\n584   \n585   // Call.\n586 > call(frametype, fn, stackArgs, uint32(frametype.size), uint32(abid.retOffset), uint32(frameSize), &regArgs)\n587   \n588   // For testing; see TestCallMethodJump.\n589   if callGC {\n"
    }
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

## 许可证

MIT
