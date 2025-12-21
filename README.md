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

### 3.1 智能 Handler 签名

`sgin` 支持多种灵活的 Handler 签名，自动处理参数绑定和响应发送。

**支持的签名示例：**

- `func(*gin.Context)` 兼容 gin
- `func(*sgin.Ctx) error`
- `func(*sgin.Ctx) (any, error)`
- `func(*sgin.Ctx, input Struct) (any, error)`
- `func(*sgin.Ctx, input Struct) (any)`

### 3.2 请求参数绑定

只需在 Handler 的第二个参数定义结构体，`sgin` 会自动将 **URI**、**Header**、**Query**、**Form** 和 **Body (JSON/XML)** 的数据绑定到该结构体上。

```go
type UserReq struct {
    ID    int    `uri:"id" binding:"required"`
	Name  string `form:"name" binding:"required" label:"姓名"`
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

### 3.3 统一响应处理

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

### 3.4 增强的 Context (`sgin.Ctx`)

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

## 配置详解

`sgin` 提供了灵活的配置选项，所有配置都在 `sgin.Config` 结构体中设置。以下是所有可用配置的详细说明：

### 4.1 基础配置

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

### 4.2 多语言配置

`sgin` 提供了完整的校验错误多语言本地化支持，基于 `validator/v10` 和 `universal-translator`。

```go
import (
    "github.com/baagod/sgin"
    "golang.org/x/text/language"
)

r := sgin.New(sgin.Config{
    Locales: []language.Tag{
        // 第一个语言为默认语言
        language.Chinese,          // 中文
        // 可配置多种语言
        language.English,          // 英文
        language.Japanese,         // 日文
        language.Korean,           // 韩文
        language.French,           // 法文
        language.Russian,          // 俄文
        language.German,           // 德文
        language.Spanish,          // 西班牙文
    },
})
```

**三层回退逻辑**：当校验失败时，错误消息中的字段名按以下顺序确定：
1. **`label` 标签**：用户友好的字段名（推荐）
2. **`json` 标签**：API 字段名
3. **结构体字段名**：最后的回退

**设计原则**：
- **零魔法原则**：不配置 `Locales` = 无翻译，返回原始英文错误
- **显式配置**：用户只需提供标准库语言标签，框架自动创建对应翻译器
- **类型安全**：使用 `language.Tag` 而非字符串，编译时检查语言标签有效性
- **自动映射**：框架内部处理翻译器注册和语言匹配，用户无需关心底层细节

### 4.3 OpenAPI 配置

启用 OpenAPI 文档生成功能：

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

### 4.4 Panic 恢复配置

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

**功能特性**：
- **多级调用栈追溯**：自动定位业务代码中的错误位置，跳过框架和标准库的干扰。
- **源码上下文展示**：在控制台直接打印报错行及其前后的源代码片段，并高亮显示。
- **路径自动简化**：智能缩短文件路径（如简化 `GOROOT`、`GOPATH` 或项目根目录路径）。
- **双流输出**：同时提供美观的控制台日志和结构化的 JSON 日志，方便接入日志系统。

## 增强特性

在了解核心功能和配置之后，以下是 `sgin` 提供的增强特性，可以帮助你构建更加强大、易维护的API。

### 5.1 OpenAPI 文档生成与使用

`sgin` 可以通过分析 Handler 的输入输出结构体，自动生成 OpenAPI 3.1 文档。

**启用方法**：
在 `sgin.Config` 中配置 `OpenAPI` 字段（见 4.3 OpenAPI 配置）。

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

### 5.2 Panic 恢复与调试

`sgin` 的 Panic 恢复功能在 4.4 章节已配置。以下是具体使用场景和最佳实践：

#### 使用场景
- **开发环境**：使用彩色控制台输出，快速定位错误位置
- **生产环境**：将结构化 JSON 日志写入文件或发送到日志收集系统
- **调试复杂错误**：源码上下文展示功能帮助理解错误的调用链

#### 最佳实践
```go
// 生产环境配置示例
r := sgin.New(sgin.Config{
    Recovery: func(c *sgin.Ctx, logStr, jsonStr string) {
        // 开发环境：输出彩色日志到控制台
        if os.Getenv("ENV") == "development" {
            fmt.Print(logStr)
        }
        
        // 生产环境：记录结构化日志
        logEntry := map[string]any{}
        json.Unmarshal([]byte(jsonStr), &logEntry)
        log.Error("panic recovered", "details", logEntry)
    },
})
```

### 5.3 多语言校验错误详细使用

#### 字段标签与错误消息

使用 `label` 标签为字段指定用户友好的名称，校验错误时会自动使用：

```go
type LoginReq struct {
    Username string `json:"username" label:"用户名" binding:"required,min=3"`
    Password string `json:"password" label:"密码" binding:"required,min=6"`
    Email    string `json:"email" label:"邮箱" binding:"required,email"`
}
```

#### 语言检测与匹配

`sgin` 支持多种语言检测方式，优先级如下：

1. **查询参数**：`?lang=zh-CN`
2. **Accept-Language 头**：支持权重解析（如 `Accept-Language: zh-CN,zh;q=0.9,en;q=0.8`）
3. **默认语言**：配置的第一个语言

**智能匹配机制**：
- 使用 Go 标准库 `golang.org/x/text/language` 进行语言匹配
- 支持语言变体智能匹配（如 `zh-CN` ↔ `zh`）
- 匹配失败时自动回退到默认语言，确保总有翻译可用

#### 使用示例

```go
import "golang.org/x/text/language"

// 配置支持的语言
r := sgin.New(sgin.Config{
    Locales: []language.Tag{
        language.Chinese,  // 默认语言
        language.English,  // 备用语言
    },
})

// 注册路由
r.POST("/login", func(c *sgin.Ctx, req LoginReq) error {
    // 业务逻辑...
    return nil
})
```

**客户端请求示例**：
```bash
# 使用查询参数指定语言
POST /login?lang=zh-CN

# 使用 Accept-Language 头
POST /login
Accept-Language: zh-CN

# 无语言信息时，使用默认语言（中文）
POST /login
```

校验失败时将返回对应语言的错误消息，如中文错误："用户名不能为空"。

#### 语言检测中间件

`sgin` 自动注册语言检测中间件（当配置了 `Locales` 时），你可以在业务代码中获取当前语言：

```go
func(c *sgin.Ctx) {
    // 获取当前请求的语言设置
    locale := c.Locale()
    fmt.Printf("当前语言: %v\n", locale)
    
    // 手动设置语言（覆盖自动检测）
    c.SetLocale(language.English)
}
```

#### 支持的语言列表

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
