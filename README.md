# sgin

这是一个 [gin](https://github.com/gin-gonic/gin) 的魔改版本，旨在让其更加简单。

`sgin` 拥有一个可选的默认配置。

```go
r := sgin.New(sgin.Config{
    Mode: gin.DebugMode, // 默认值
    ErrorHandler: func(c *sgin.Ctx, err error) { // 默认错误处理
        var e *Error
        code := StatusInternalServerError
    
        if errors.As(err, &e) && e.Code > 0 { // 如果是 *Error
            code = e.Code
        } else if stc := c.StatusCode(); stc != 200 && stc != 0 {
            code = stc
        }
    
        return c.Status(code).Send(err.Error())
    }
})

r.Run(":8080")
```

## 处理方法

`sgin` 主要修改了原生处理函数，处理函数的签名变成了：`func(*sgin.Ctx[, T]) T | (T, error)` 。

### 接收请求参数

其中跟在 `*sgin.Ctx` 后面的 `T` 是可选的，它接收来自请求传递的数据。例如请求 `index?name=p1&age=10`
，这会将查询参数绑定到结构体 `p` 中。

```go
r.Get("/index", func(c *sgin.Ctx, p struct{
    Name string `form:"name"`
    Age  int    `form:"age"`
}) {
    // p => {"p1", 10}
})
```

接收 JSON 参数，只需将标签改为 `json` (`xml` 同理)：

```go
r.Get("/index", func(c *sgin.Ctx, p struct{
    Name string `json:"name"`
    Age  int    `json:"age"`
}) {
    // p => {"p1", 10}
})
```

### 返回响应

要返回响应数据，你可以使用 `c.Send()`，或在处理函数中定义返回值（稍后介绍）。

```go
r.Get("/index", func(c *sgin.Ctx) {
    c.Sned("Very OK")
})
```

`Send(any, format ...string)` 方法会自动根据请求头 `Accept` 返回对应格式的数据，也可以手动指定。

`Send()` 细节：

- 如果发送数字，将被仅当做状态码返回。
- 如果发送 `string` 或 `error` 将返回原字符串或 `error.Error()`。
- 将格式传递给 `format` 参数，将返回对应类型的数据。例如 `c.Send(map[string]any{}, sgin.FormatJSON)`。
- 如果你有状态码和错误一起返回：`c.Send(sgin.NewError(statusCode, msg))` 。
- 当然你可以先设置状态码后发送你的数据：`c.Status(200).Send(...)`。

----

除了可以使用 `Send()` 返回响应数据外，还可以将其定义为处理函数的返回值，返回值类型可以为：

- `T`: 任意响应体；
- `(int, T)`: 状态码和任意响应体；
- `(T, error)`: 响应体和错误。

注意，`T` 是任意类型，通常为 `int`, `error`, `string`, `Any`，处理函数的返回值类型和使用 `Send()` 返回的数据基本是一致的。

```go
r.Get("/index", func(c *sgin.Ctx) int {
    return 401 // 只返回状态码
})

r.Get("/index", func(c *sgin.Ctx) error {
    return errors.New("error") // 在不指定错误状态码的情况下，状态码默认为 500。
})

r.Get("/index", func(c *sgin.Ctx) (r sgin.Response) {
    return r.OK() // 返回任意响应
})

r.Get("/index", func(c *sgin.Ctx) (r *map[string]any, error) {
    return nil, errors.new("test error") // 返回响应或错误
})

r.Get("/index", func(c *sgin.Ctx) (int, map[string]any) {
    return nil, map[string]any{"msg": "test"} // 返回状态码和响应
})
```

## Api

### `Ctx`

- `Args() map[string]any`：返回请求参数集合，无论是 GET、POST 还是 JSON 等请求；
- `ArgInt(key string, e ...string)` 等：将请求参数转为对应的类型；
- `Send(any, format ...string)` 发送响应。
- `SendHTML(string, any)` 以 `HTML` 模板作为响应发送。
- `Locals(key string, ...any) any`：设置或将值存储到上下文。
- `Header(key string, ...any) any`：设置或写入响应头。此外 `sgin`
  定义了许多枚举来帮助你快速找到某个请求头，例如要获取内容类型：`Header(sgin.HeaderContentType)`；
- `Status(int) *Ctx`：设置响应状态码；
- `StatusCode() int`：获取响应状态码；
- `Method() string`：获取请求方法；
- `Path(full ...bool) string`：返回部分或全部请求路径；
- `IP() string`：返回远程客户端 IP，如果是本机则返回 127.0.0.1；
- ... 其他来自 `gin.Context` 的方法。
