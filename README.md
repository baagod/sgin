# sgin

这是一个 [gin](https://github.com/gin-gonic/gin) 的封装版本，旨在提供更加智能、简洁的 API 开发体验，并且完美兼容原生 `gin`, `gin.HandlerFunc` (包括中间件处理) 。

它通过增强的 `Handler` 签名、参数绑定、统一的响应处理、错误处理、自动化 OpenAPI 文档生成和多语言校验错误等支持，让开发者专注于业务逻辑。

## 安装

```bash
go get github.com/baagod/sgin
```

## 快速开始

```go
 r := sgin.New(sgin.Config{})
 r.GET("/", func(c *sgin.Ctx) string {
     return "Hello sgin!"
 })
 r.Run(":8080")
```

## 核心功能

`sgin` 的核心价值在于提供更加智能、简洁的 API 开发体验。以下是你需要了解的核心功能。

### 智能 Handler 签名

`sgin` 支持多种灵活的 `Handler` 签名，自动处理参数绑定和响应发送。

**支持的签名示例：**

- `func(*gin.Context)`: 兼容 `gin.HandlerFunc`
- `func(*sgin.Ctx) error`
- `func(*sgin.Ctx) (any, error)`
- `func(*sgin.Ctx, input Struct) (any, error)`
- `func(*sgin.Ctx, input Struct) (any)`
- `func(*sgin.Ctx, input *Struct)`: 支持绑定指针结构体

### 请求参数绑定

只需在 `Handler` 的第二个参数定义结构体，`sgin` 会自动将其与 `URI`、`Header`、`Query`、`Form` 和 `Body` (JSON/XML) 的数据绑定。如下：

```go
type User struct {
    ID    int    `uri:"id" binding:"required"`
	Name  string `form:"name" binding:"required" label:"姓名"`
    Age   int    `form:"age" default:"18"`
    Token string `header:"Authorization"`
}

r.POST("/users/:id", func(c *sgin.Ctx, p User) (map[string]any, error) {
    return map[string]any{"id": p.ID, "name": p.Name, "age": p.Age}, nil
})
```

### 统一响应处理

`Handler` 的返回值会被自动处理：

- `error`: 调用配置的 `ErrorHandler` 将 `error.Error()` 返回。
- `data`: 根据请求头 `Accept` 格式化为 `JSON`, `XML` 或 `Text`。

你也可以使用 `c.Send()` 发送指定数据：

```go
c.Send("Hello") // Text
c.Send(User{})  // 根据请求头 `Accept` 返回对应格式的数据
c.Send(sgin.BodyXML(User{}))  // 手动指定格式
c.Send(err)                   // Error
```

### 增强的 `sgin.Ctx`

`sgin.Ctx` 封装了 `gin.Context`，提供更便捷 API。以下是所有可用方法的完整指南：

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

- `Send(body any, format ...string) error`: 发送响应，自动根据 Accept 头协商格式
- `Status(code int) *Ctx`: 设置响应状态码（链式调用）
- `Header(key string, value string) *Ctx`: 设置响应头（链式调用）
- `Content(value string) *Ctx`: 设置 Content-Type 头（链式调用）

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

#### 多语言支持

- `Locale() language.Tag`: 获取当前请求的语言设置
- `SetLocale(locale language.Tag)`: 手动设置请求语言（覆盖自动检测）

### Engine API

`sgin.Engine` 是框架的核心入口，它封装了 `gin.Engine` 并提供了更简洁、一致的 API 设计。以下是 `Engine` 的主要方法：

- `New(config ...sgin.Config) *sgin.Engine`: 创建新的 `sgin` 实例，支持可选配置
- `Run(addr string, certfile ...string) error`: 启动 HTTP(S) 服务器，通过可选参数支持 HTTPS
- `RunListener(listener net.Listener) error`: 通过指定的监听器启动服务器
- `Gin() *gin.Engine`: 获取底层的 `gin.Engine` 实例 (用于访问原生功能) 。

#### 使用示例

```go
// 1. 极简初始化
app := sgin.New()

// 2. 链式路由定义（继承自 Router）
app.GET("/", func(c *sgin.Ctx) string {
  return "Hello sgin!"
})

// 3. 启动 HTTP 服务
go app.Run(":8080")

// 4. 启动 HTTPS 服务
go app.Run(":8443", "cert.pem", "cert.key")

// 5. 通过监听器启动（灵活部署）
listener, _ := net.Listen("tcp", ":9090")
app.RunListener(listener)

// 6. 访问底层 gin（逃生舱模式）
ginEngine := app.Gin()
ginEngine.Static("/static", "./assets")
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

#### **结构化 JSON 输出**

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

**`sgin` 目前支持以下语言：**

- 🇨🇳 中文 (Chinese, SimplifiedChinese)
- 🇺🇸 英文 (English)
- 🇯🇵 日文 (Japanese)
- 🇰🇷 韩文 (Korean)
- 🇫🇷 法文 (French)
- 🇷🇺 俄文 (Russian)
- 🇩🇪 德文 (German)
- 🇪🇸 西班牙文 (Spanish)

可以通过 `sgin.SupportedLanguages()` 函数获取框架支持的所有语言标签。
