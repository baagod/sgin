# sgin

这是一个 [gin](https://github.com/gin-gonic/gin) 的魔改版本，旨在让其更加简单。

> 这个库目前只有我一个人在用，可能并没有考虑到大众的需求，或有些不足的地方和 Bug，如果你也在使用这个库，遇到任何问题都可以提交 Issue。

## 使用

```go
r := sgin.New(sgin.Config{
    Mode: gin.DebugMode, // 默认值
    // 相当于 LoadHTMLGlob() 或 LoadHTMLFiles()
    Views: []string{"./views/index.tmpl"},
})

r.Run(":8080")
```

配置是可选的，可传可不传。

### 处理方法

处理方法的签名是 `func(<*sin.Ctx | *gin.Context>[input])`。第一个参数必须是 `*sin.Ctx` 或 `*gin.Context`，第二个参数 `input` 是可选的，它接收任意请求传来的数据。用法如下：

```go
type Request struct {
    Name string `form:"name"`
    Age  string `form:"age" binding:"required"`
}

r.Get("/index", func(c *sgin.Ctx, r Request) {
    _ = c.Send(r) // 将 r 当做响应体发送回去
})
```

`Send(any, ...string)` 方法会自动根据请求头 `Accept` 返回对应的数据类型，也可以将想要发送的其他格式传递给第二个参数 `format`，如 `sgin.FormatJSON` 或 `sgin.FormatXML`。

但是如果发送的数据类型是 `Error` 或 `String`，则忽略上面的条件直接返回 `String` 响应。

此外，处理方法还可以有返回值，可以返回如下类型：

- `int` 表示状态码；
- `response` 任意响应体 (包括 `Error`)；
- `(int, response)` 状态码, 任意响应体；
- `(response, Error)` 任意响应体，错误。

```go
r.Get("/index", func(c *sgin.Ctx) (r *sgin.Response) {
    // 此处返回响应的工作与 `Send()` 方法相同
    return r.WithMessage("OK")
})
```

### `Ctx` Api

- `Args() map[string]any`：该方法根据不同的请求方式返回请求参数的集合；

    例如有一个查询请求为 `name=xx&age=10`，调用 `Args()` 方法将返回 `{"name": "xx", "age": 10}`，同样的 POST 或 JSON 请求也会返回该 `map`。
    
    这样我们就可以不再用区分是何种请求方式，直接调用 `Args()` 就能拿到你想要的数据。其他如 `ArgInt()`、`ArgBool()...` 方法都是该方法的快捷方式。

- `Set(key string, ...any) any`：如果给定第二个参数，则将值设置在上下文中，否则返回 `key` 的值；
- `Header(key string, ...any) any`：该方法返回或设置请求头的值。此外 `sgin` 定义了许多枚举来帮助你快速找到某个请求头，例如要获取内容类型：`Header(sgin.HeaderContentType)`；
- `Bind(any) error`：将请求数据绑定到一个结构体中，该方法遇到错误不会终止请求；
- `SendStatus(int) error`：发送状态码响应；
- `SendFile(string, attachment ...bool) error`：发送文件。`attachment` 表示是否要将其作为下载内容；
- `Status(int) *Ctx`：设置响应状态码；
- `StatusCode() int`：获取响应状态码；
- `Method() string`：获取请求方法；
- `HeaderOrQuery(string) string`：返回请求头 (优先) 或查询参数；
- `Path() string`：返回 `Request.URL.Path`；
- `IP() string`：返回远程客户端 IP，如果是本机则返回 127.0.0.1；

[视频教程](https://www.bilibili.com/video/BV1Nh4y1e7kk/?vd_source=7ae7a1bdbc2bfacc227a70634fc5d2c2#reply186203730000)
