package sgin

import (
    "errors"
    "net"
    "net/http"

    "github.com/baagod/sgin/oa"
    "github.com/gin-gonic/gin"
)

type Engine struct {
    Router
    cfg    Config
    engine *gin.Engine
}

type Config struct {
    Mode           string                      // gin.DebugMode | gin.ReleaseMode | gin.TestMode
    TrustedProxies []string                    // gin.SetTrustedProxies
    Recovery       func(c *Ctx, out, s string) // 回调 [带颜色的输出] 和 [结构化日志]
    ErrorHandler   func(c *Ctx, err error) error
    // 回调 [纯文本] 和 [JSON] 日志，返回 true 输出默认日志。
    Logger  func(c *Ctx, text string, s string) bool
    OpenAPI *oa.OpenAPI
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
        api:  cfg.OpenAPI,
        op: oa.Operation{
            Responses: map[string]*oa.Response{},
            Security:  []oa.Requirement{{}},
        },
    }

    // gin.engine 配置
    if err := e.engine.SetTrustedProxies(cfg.TrustedProxies); err != nil {
        debugWarning(err.Error())
    }

    // 注册 [日志] 和 [恢复] 中间件
    e.Use(Logger, Recovery)

    // OpenAPI 文档中间件
    if cfg.OpenAPI != nil && cfg.Mode != gin.ReleaseMode {
        e.GET("/openapi.yaml", func(c *Ctx) error {
            if specYAML, err := cfg.OpenAPI.YAML(); err == nil {
                return c.Content(MIMETextYAMLUTF8).Send(string(specYAML))
            }
            return c.Send(ErrInternalServerError())
        })

        e.GET("/docs", func(c *Ctx) error {
            return c.Content(MIMETextHTMLUTF8).Send(oa.DocsHTML)
        })
    }

    return e
}

// Run 将路由器连接到 http.Server 并开始监听和提供 HTTP[S] 请求
//
// 提供 cert 和 key 文件路径作为 certfile 参数，可开启 HTTPS 服务。
func (e *Engine) Run(addr string, certfile ...string) (err error) {
    if len(certfile) > 0 {
        return e.engine.RunTLS(addr, certfile[0], certfile[1])
    }
    return e.engine.Run(addr)
}

// RunListener 将路由器连接到 http.Server, 并开始通过指定的 listener 监听和提供 HTTP 请求。
func (e *Engine) RunListener(listener net.Listener) (err error) {
    return e.engine.RunListener(listener)
}
