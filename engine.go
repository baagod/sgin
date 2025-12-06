package sgin

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ztrue/tracerr"
)

type Engine struct {
	Route
	config Config
	engine *gin.Engine
}

type Config struct {
	Mode           string // gin.DebugMode | gin.ReleaseMode | gin.TestMode
	TrustedProxies []string
	Recovery       func(*Ctx, string)
	ErrorHandler   func(*Ctx, error) error
	OpenAPI        bool // 是否开启 OpenAPI 文档生成
}

// DefaultErrorHandler 该进程从处理程序返回错误
func DefaultErrorHandler(c *Ctx, err error) error {
	var e *Error
	statusCode := StatusInternalServerError

	if errors.As(err, &e) && e.Code > 0 { // 如果是 *Error 错误
		statusCode = e.Code
	} else if stc := c.StatusCode(); stc != 200 && stc != 0 {
		statusCode = stc
	}

	c.Header(HeaderContentType, MIMETextPlainCharsetUTF8)
	return c.Status(statusCode).Send(err.Error())
}

func defaultConfig(config ...Config) Config {
	cfg := append(config, Config{})[0]
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = DefaultErrorHandler
	}
	return cfg
}

func New(config ...Config) *Engine {
	cfg := defaultConfig(config...)
	gin.SetMode(cfg.Mode)

	e := &Engine{engine: gin.New(), config: cfg}
	e.Route = Route{engine: e, group: &e.engine.RouterGroup, root: true}

	if err := e.engine.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		debugError(err)
	}

	if cfg.Recovery != nil {
		e.Use(func(ctx *Ctx) error {
			defer func() {
				if err := recover(); err != nil {
					_ = ctx.Send(ErrInternalServerError)
					cfg.Recovery(ctx, tracerr.Sprint(tracerr.Wrap(err.(error))))
				}
			}()
			return ctx.Next()
		})
	} else {
		e.engine.Use(gin.Recovery())
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
