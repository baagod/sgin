package sgin

import (
	"errors"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
	"golang.org/x/text/language"
)

const EngineKey = "_baa/sgin/engine"

type Engine struct {
	Router
	cfg             Config
	engine          *gin.Engine
	languageMatcher language.Matcher
	translator      *ut.UniversalTranslator
	defaultLang     language.Tag
}

type Config struct {
	Mode           string                      // gin.DebugMode | gin.ReleaseMode | gin.TestMode
	TrustedProxies []string                    // gin.SetTrustedProxies
	Recovery       func(c *Ctx, out, s string) // 回调 [带颜色的输出] 和 [结构化日志]
	ErrorHandler   func(c *Ctx, err error) error
	Logger         func(c *Ctx, out string, s string) // 回调 [纯文本] 和 [JSON] 日志
	OpenAPI        *API
	Locales        []language.Tag // 绑定验证错误所使用的多语言支持
}

// DefaultErrorHandler 默认的错误处理器
func DefaultErrorHandler(c *Ctx, err error) error {
	code := http.StatusInternalServerError

	var e *Error
	if errors.As(err, &e) && e.Code > 0 { // 如果是 *Error 错误
		code = e.Code
	} else if stc := c.StatusCode(); stc != 200 && stc != 0 {
		code = stc
	}

	return c.Content(MIMETextPlainUTF8).Status(code).Send(err.Error())
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
		op:   Operation{Responses: map[string]*ResponseBody{}},
	}

	// 注册 Engine 注入中间件 (必须在最前)
	e.Use(func(c *gin.Context) {
		c.Set(EngineKey, e)
		c.Next()
	})

	e.Use(Recovery, Logger) // 注册 [恢复] 和 [日志] 中间件
	if tr := useTranslator(e); tr != nil {
		e.Use(tr) // 注册校验翻译器
	}

	// gin.engine 配置
	if err := e.engine.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		debugWarning(err.Error())
	}

	// OpenAPI 文档中间件
	if cfg.OpenAPI != nil && cfg.Mode != gin.ReleaseMode {
		e.GET("/openapi.yaml", He(func(c *Ctx) error {
			if specYAML, err := cfg.OpenAPI.YAML(); err == nil {
				return c.Content(MIMETextYAMLUTF8).Send(string(specYAML))
			}
			return c.Send(ErrInternalServerError())
		}), APIHidden)

		e.GET("/docs", He(func(c *Ctx) error {
			return c.Content(MIMETextHTMLUTF8).Send(DocsHTML)
		}), APIHidden)

		debugInfo("OpenAPI 已开启，访问 /docs 或 /openapi.yaml 查看文档。")
	}

	return e
}

// Routes 返回注册的路由信息切片
func (e *Engine) Routes() gin.RoutesInfo {
	return e.engine.Routes()
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

func (e *Engine) Gin() *gin.Engine {
	return e.engine
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
