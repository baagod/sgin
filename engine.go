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
	Mode           string // gin.DebugMode | gin.ReleaseMode | gin.TestMode
	Run            string // address, e.g ":8080"
	RunTLS         string // address, certFile, keyFile
	RunListener    net.Listener
	RunTLSListener func() (listener net.Listener, certFile string, keyFile string)
	Recovery       func(*Ctx, string)
	ErrorHandler   func(*Ctx, error) error
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

func (e *Engine) Run() (err error) {
	if tlsFn := e.config.RunTLSListener; tlsFn != nil {
		listener, certFile, keyFile := tlsFn()
		if gin.IsDebugging() {
			_, _ = fmt.Fprintf(gin.DefaultWriter, "[GIN-debug] Listening and serving HTTPS on listener what's bind with address@%s", listener.Addr())
			defer func() {
				if err != nil {
					_, _ = fmt.Fprintf(gin.DefaultErrorWriter, "[GIN-debug] [ERROR] %v\n", err)
				}
			}()
		}
		return http.ServeTLS(listener, e.engine.Handler(), certFile, keyFile)

	} else if listener := e.config.RunListener; listener != nil {
		return e.engine.RunListener(listener)

	} else if tls := e.config.RunTLS; tls != "" {
		tls = strings.ReplaceAll(tls, " ", "")
		certs := strings.Split(tls, ",")
		return e.engine.RunTLS(certs[0], certs[0], certs[1])
	}

	return e.engine.Run(e.config.Run)
}
