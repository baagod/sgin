package sgin

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net"
	"net/http"
)

type App struct {
	RouterGroup
	engine     *gin.Engine
	errHandler func(*Ctx, error) error
}

type Config struct {
	Mode         string   // gin.DebugMode | gin.ReleaseMode
	Views        []string // filepath.Glob pattern | []file
	ErrorHandler func(*Ctx, error) error
}

func defaultConfig(f ...Config) Config {
	cfg := append(f, Config{})[0]
	if cfg.Mode == "" {
		cfg.Mode = gin.DebugMode
	}
	return cfg
}

func New(f ...Config) *App {
	cfg := defaultConfig(f...)
	gin.SetMode(cfg.Mode)

	var engine *gin.Engine
	if gin.SetMode(cfg.Mode); cfg.Mode == gin.DebugMode {
		engine = gin.Default()
	} else {
		engine = gin.New()
	}

	if views := len(cfg.Views); views > 0 {
		if views == 1 {
			engine.LoadHTMLGlob(cfg.Views[0])
		} else {
			engine.LoadHTMLFiles(cfg.Views...)
		}
	}

	app := &App{
		engine:      engine,
		RouterGroup: RouterGroup{grp: &engine.RouterGroup, root: true},
		errHandler: func(c *Ctx, err error) error {
			code := http.StatusInternalServerError

			var e *Error
			if errors.As(err, &e) && e.Code != 0 {
				code = e.Code
			}

			return c.Status(code).Send(err)
		},
	}

	app.RouterGroup.app = app
	if cfg.ErrorHandler != nil {
		app.errHandler = cfg.ErrorHandler
	}

	return app
}

func (app *App) Routes() gin.RoutesInfo {
	return app.engine.Routes()
}

func (app *App) Run(addr string, certAndKeyFile ...string) error {
	if certAndKeyFile != nil {
		return app.engine.RunTLS(addr, certAndKeyFile[0], certAndKeyFile[1])
	}
	return app.engine.Run(addr)
}

func (app *App) RunListener(listener net.Listener) error {
	return app.engine.RunListener(listener)
}
