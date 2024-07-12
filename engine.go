package sgin

import (
	"errors"
	"net"

	"github.com/gin-gonic/gin"
	"github.com/ztrue/tracerr"
)

type Engine struct {
	Routers
	engine     *gin.Engine
	errHandler func(*Ctx, error) error
}

type Config struct {
	Mode         string   // gin.DebugMode | gin.ReleaseMode
	Views        []string // filepath.Glob pattern | []file
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
	f := append(config, Config{})[0]
	gin.SetMode(f.Mode)

	e := &Engine{engine: gin.New()}
	e.Routers = Routers{engine: e, grp: &e.engine.RouterGroup, root: true}

	if views := len(f.Views); views > 0 {
		if views == 1 {
			e.engine.LoadHTMLGlob(f.Views[0])
		} else {
			e.engine.LoadHTMLFiles(f.Views...)
		}
	}

	if e.errHandler = f.ErrorHandler; e.errHandler == nil {
		e.errHandler = DefaultErrorHandler
	}

	if f.Recovery != nil {
		e.Use(func(ctx *Ctx) error {
			defer func() {
				if err := recover(); err != nil {
					_ = ctx.Send(ErrInternalServerError)
					f.Recovery(ctx, tracerr.Sprint(tracerr.Wrap(err.(error))))
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

func (e *Engine) Run(addr string, certAndKeyFile ...string) error {
	if certAndKeyFile != nil {
		return e.engine.RunTLS(addr, certAndKeyFile[0], certAndKeyFile[1])
	}
	return e.engine.Run(addr)
}

func (e *Engine) RunListener(listener net.Listener) error {
	return e.engine.RunListener(listener)
}
