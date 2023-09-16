package sgin

import (
	"github.com/gin-gonic/gin"
	"net"
)

type App struct {
	e *gin.Engine
	RouterGroup
}

type Config struct {
	Mode  string   // gin.DebugMode | gin.ReleaseMode
	Views []string // filepath.Glob pattern | []file
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

	var e *gin.Engine
	if cfg.Mode == gin.DebugMode {
		e = gin.Default()
	} else {
		e = gin.New()
	}

	if len(cfg.Views) == 1 {
		e.LoadHTMLGlob(cfg.Views[0])
	} else {
		e.LoadHTMLFiles(cfg.Views...)
	}

	return &App{e: e, RouterGroup: RouterGroup{&e.RouterGroup}}
}

func (app *App) Routes() gin.RoutesInfo {
	return app.e.Routes()
}

func (app *App) Run(addr string, certAndKeyFile ...string) error {
	if certAndKeyFile != nil {
		return app.e.RunTLS(addr, certAndKeyFile[0], certAndKeyFile[1])
	}
	return app.e.Run(addr)
}

func (app *App) RunListener(listener net.Listener) error {
	return app.e.RunListener(listener)
}
