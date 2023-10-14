# sgin

这是一个 [gin](https://github.com/gin-gonic/gin) 的魔改版本，旨在让其更加简单。

## 介绍

```go
r := sgin.New(sgin.Config{
    Mode: gin.DebugMode, // 默认值
    // 相当于 LoadHTMLGlob() 或 LoadHTMLFiles()
    Views: []string{"./views/index.tmpl"},
    ErrorHandler: func(c *sgin.Ctx, err error) { // 默认
      code := http.StatusInternalServerError
      var e *Error
      if errors.As(err, &e) && e.Code != 0 {
          code = e.Code
      }
      return c.Status(code).Send(err)
    }
})

r.Run(":8080")
```

配置是可选的，可传可不传。

### 处理方法

处理方法的签名是：`gin.HandlerFunc, func(*Ctx[, *T]) <error | T> | (int, T) | (T, error)`。
处理方法的第二个参数是可选的，它接收任意请求传来的数据。用法如下：

```go
type Request struct {
    Name   string `form:"name" json:"name"`
    Age    string `form:"age" json:"age" binding:"required"`
    Milli  string `header:"milli" json:"milli"`
    uid    string `uri:"uid" json:"uri"`
}

r.Get("/index", func(c *sgin.Ctx, r *Request) error {
    return c.Send(r) // 将 r 当做响应体发送
})

r.Get("/index/v2", func(c *sgin.Ctx) (r *sgin.Response) {
    // 返回原理与 `Send()` 方法相同
    return &sgin.Response{Message: "OK"}
})
```

`Send(any, format ...string)` 方法会自动根据请求头 `Accept` 返回对应类型的数据，也可以指定其他类型发送，如 `sgin.FmtJSON`。

当处理方法返回响应发生错误时，就会调用全局错误处理函数。以下是处理方法支持的返回类型：

- `error | T`：返回一个错误或具体的响应；
- `T`：返回任意的响应体；
- `(int, T)`：返回状态码和响应体；
- `(T, error)`：任意响应体，错误。

此外，如果你还想要为输入绑定更多的额外参数，或是想为某方法单独处理错误：

```go
r.GET("/index", &sgin.Handler{
    Binding: []binding.Binding{sgin.Uri, sgin.Header},
    Fn: func(c *sgin.Ctx, req *Request) error {
        return c.Send(r)
    },
    Error: func(c *sgin.Ctx, err error) error {
        return c.Status(500).Send(err)
    },
})
```

该处理将添加 `uri` 和 `header` 参数到你的输入 ( `req` ) 中，并且在发生错误时，错误将回调到该单独错误处理中，而不会再在其他地方调用。

## Api

### `Ctx`

- `Args() map[string]any`：该方法根据不同的请求方式返回请求参数的集合；

  例如有一个查询请求为 `name=xx&age=10`，调用 `Args()` 方法将返回 `{"name": "xx", "age": 10}`，同样的 POST 或 JSON
  请求也会返回该 `map`。

  这样我们就可以不再用区分是何种请求方式，直接调用 `Args()` 就能拿到你想要的数据。其他如 `ArgInt()`、`ArgBool()...`
  方法都是该方法的快捷方式。
- `Send(any, format ...string)` 发送响应。
- `SendHTML(string, any)` 以 `HTML` 模板作为响应发送。
- `Set(key string, ...any) any`：设置或将值存储在上下文中。
- `Header(key string, ...any) any`：设置或将值写入响应头中。此外 `sgin`
  定义了许多枚举来帮助你快速找到某个请求头，例如要获取内容类型：`Header(sgin.HeaderContentType)`；
- `Status(int) *Ctx`：设置响应状态码；
- `StatusCode() int`：获取响应状态码；
- `Method() string`：获取请求方法；
- `Path() string`：返回 `Request.URL.Path`；
- `IP() string`：返回远程客户端 IP，如果是本机则返回 127.0.0.1；
- 其他继承自 `gin.Context` 的方法。 
