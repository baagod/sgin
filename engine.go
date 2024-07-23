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
	config Config
	engine *gin.Engine
	Routers
}

type Config struct {
	Mode         string // gin.DebugMode | gin.ReleaseMode | gin.TestMode
	Recovery     func(*Ctx, string)
	ErrorHandler func(*Ctx, error) error
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

func New(config ...Config) *Engine {
	cfg := append(config, Config{})[0]
	if cfg.ErrorHandler == nil {
		cfg.ErrorHandler = DefaultErrorHandler
	}
	gin.SetMode(cfg.Mode)

	e := &Engine{engine: gin.New()}
	e.Routers = Routers{engine: e, grp: &e.engine.RouterGroup, root: true}

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

func (e *Engine) Run(addr ...string) (err error) {
	address := append(addr, ":8080")[0]
	debug("Listening and serving HTTP on %s\n", address)
	defer func() { debugError(err) }()
	return http.ListenAndServe(address, e.Handler())
}

func (e *Engine) RunTLS(addr, certFile, keyFile string) (err error) {
	debug("Listening and serving HTTPS on %s\n", addr)
	defer func() { debugError(err) }()
	return http.ListenAndServeTLS(addr, certFile, keyFile, e.Handler())
}

func (e *Engine) RunServer(listener net.Listener) (err error) {
	debug("Listening and serving HTTP on %s", listener.Addr())
	defer func() { debugError(err) }()
	return http.Serve(listener, e.Handler())
}

func (e *Engine) RunServeTLS(listener net.Listener, certFile string, keyFile string) (err error) {
	debug("Listening and serving HTTPS on %s", listener.Addr())
	defer func() { debugError(err) }()
	return http.ServeTLS(listener, e.Handler(), certFile, keyFile)
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
