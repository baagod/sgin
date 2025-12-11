package sgin

import (
    "errors"
    "fmt"
    "net"
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
)

type Engine struct {
    Router
    cfg    Config
    engine *gin.Engine
}

type Config struct {
    Mode           string   // gin.DebugMode | gin.ReleaseMode | gin.TestMode
    TrustedProxies []string // gin.SetTrustedProxies
    Recovery       func(*Ctx, string)
    ErrorHandler   func(*Ctx, error) error
    // 日志记录器回调，参数为 [文本] 和 [JSON] 消息，返回 true 输出默认日志。
    Logger  func(*Ctx, string, string) bool
    OpenAPI bool // 是否开启 OpenAPI 文档生成
}

// DefaultErrorHandler 默认的错误处理器
func DefaultErrorHandler(c *Ctx, err error) error {
    // Status code default is 500
    code := http.StatusInternalServerError

    var e *Error
    if errors.As(err, &e) && e.Code > 0 { // 如果是 *Error 错误
        code = e.Code
    } else if stc := c.StatusCode(); stc != 200 && stc != 0 {
        code = stc
    }

    return c.Content(MIMETextPlainUTF8).Status(code).Send(err.Error())
}

func defaultConfig(config ...Config) (cfg Config) {
    if len(config) > 0 {
        cfg = config[0]
    }
    if cfg.ErrorHandler == nil {
        cfg.ErrorHandler = DefaultErrorHandler
    }
    return cfg
}

func New(config ...Config) *Engine {
    cfg := defaultConfig(config...)
    gin.SetMode(cfg.Mode)

    e := &Engine{engine: gin.New(), cfg: cfg}
    e.Router = Router{
        i:    e.engine,
        e:    e,
        base: "/",
        op: OAOperation{
            Responses: map[string]OAResponse{},
            Security:  []OARequirement{{}},
        },
    }

    // gin.engine 配置
    if err := e.engine.SetTrustedProxies(cfg.TrustedProxies); err != nil {
        _, _ = fmt.Fprintf(gin.DefaultErrorWriter, "[GIN] [ERROR] %v\n", err)
    }

    // 注册 [日志] 和 [恢复] 中间件
    e.Use(Logger, Recovery)

    // OpenAPI 文档中间件
    if cfg.OpenAPI && cfg.Mode != gin.ReleaseMode {
        e.GET("/openapi.yaml", func(c *Ctx) error {
            if specYAML, err := globalSpec.YAML(); err == nil {
                return c.Content(MIMETextYAMLUTF8).Send(string(specYAML))
            }
            return c.Send(ErrInternalServerError())
        })

        e.GET("/docs", func(c *Ctx) error {
            return c.Content(MIMETextHTMLUTF8).Send(swaggerHTML)
        })
    }

    return e
}

func (e *Engine) Routes() gin.RoutesInfo {
    return e.engine.Routes()
}

func (e *Engine) Handler() http.Handler {
    return e.engine.Handler()
}

// Run 启动 HTTP 或 HTTPS 服务器。
// 参数 addr 指定服务器监听的地址，为空则使用 ":8080"。
// 参数 cert 为可选参数，包含证书和私钥路径，如果提供则启动 HTTPS 服务器；否则启动 HTTP 服务器。
// 返回值 err 表示启动服务器时可能发生的错误。
func (e *Engine) Run(addr string, cert ...string) (err error) {
    defer func() { debugError(err) }()
    if addr == "" {
        addr = ":8080"
    }

    if cert != nil {
        debug("Listening and serving HTTPS on %s\n", addr)
        return http.ListenAndServeTLS(addr, cert[0], cert[1], e.Handler())
    }

    debug("Listening and serving HTTP on %s\n", addr)
    return http.ListenAndServe(addr, e.Handler())
}

// RunServer 使用提供的 listener 启动 HTTP 或 HTTPS 服务器。
// 参数 listener 是一个 net.Listener 接口，用于指定服务器监听的网络连接。
// 参数 cert 为可选参数，包含证书和私钥路径，如果提供则启动 HTTPS 服务器；否则启动 HTTP 服务器。
// 返回值 err 表示启动服务器时可能发生的错误。
func (e *Engine) RunServer(listener net.Listener, cert ...string) (err error) {
    defer func() { debugError(err) }()

    if cert != nil {
        debug("Listening and serving HTTPS on %s", listener.Addr())
        return http.ServeTLS(listener, e.Handler(), cert[0], cert[1])
    }

    debug("Listening and serving HTTP on %s", listener.Addr())
    return http.Serve(listener, e.Handler())
}

func debug(format string, values ...any) {
    if !gin.IsDebugging() {
        return
    }
    if !strings.HasSuffix(format, "\n") {
        format += "\n"
    }
    _, _ = fmt.Fprintf(gin.DefaultWriter, "[GIN-debug] "+format, values...)
}

func debugError(err error) {
    if err != nil && gin.IsDebugging() {
        _, _ = fmt.Fprintf(gin.DefaultErrorWriter, "[GIN-debug] [ERROR] %v\n", err)
    }
}
