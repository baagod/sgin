package sgin

import (
	"errors"
	"net"
	"strings"

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
	code := StatusInternalServerError
	if errors.As(err, &e) && e.Code > 0 {
		code = e.Code
	}
	c.Header(HeaderContentType, MIMETextPlainCharsetUTF8)
	return c.Status(code).Send(err.Error())
}

func New(config ...Config) *Engine {
	cfg := append(config, Config{})[0]
	gin.SetMode(cfg.Mode)

	e := &Engine{engine: gin.New()}
	e.Routers = Routers{engine: e, grp: &e.engine.RouterGroup, root: true}

	if views := len(cfg.Views); views > 0 {
		if views == 1 {
			e.engine.LoadHTMLGlob(cfg.Views[0])
		} else {
			e.engine.LoadHTMLFiles(cfg.Views...)
		}
	}

	if e.errHandler = cfg.ErrorHandler; e.errHandler == nil {
		e.errHandler = DefaultErrorHandler
	}

	if cfg.Recovery != nil {
		e.Use(func(c *Ctx) error {
			defer func() {
				if err := recover(); err != nil {
					_ = c.Send(ErrInternalServerError)
					trace := tracerr.SprintSource(tracerr.Wrap(err.(error)), 3)
					cfg.Recovery(c, strings.ReplaceAll(trace, "\t", "    "))
				}
			}()
			return c.Next()
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
