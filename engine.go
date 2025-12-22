package sgin

import (
	"errors"
	"net"
	"net/http"
	"reflect"
	"strings"

	"github.com/baagod/sgin/oa"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"golang.org/x/text/language"
)

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
	// 回调 [纯文本] 和 [JSON] 日志，返回 true 输出默认日志到控制台。
	Logger  func(c *Ctx, out string, s string) bool
	OpenAPI *oa.OpenAPI
	Locales []language.Tag // 绑定验证错误所使用的多语言支持
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
		op: oa.Operation{
			Responses: map[string]*oa.Response{},
			Security:  []oa.Requirement{{}},
		},
	}

	e.Use(Recovery, Logger) // 注册 [恢复] 和 [日志] 中间件
	e.useTranslator()       // 注册校验翻译器

	// gin.engine 配置
	if err := e.engine.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		debugWarning(err.Error())
	}

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

func (e *Engine) Gin() *gin.Engine {
	return e.engine
}

// localeMiddleware 语言检测中间件
func localeMiddleware(c *Ctx) error {
	// 1. 优先检查查询参数 ?lang=zh-CN
	if lang := c.ctx.Query("lang"); lang != "" {
		if tag, err := language.Parse(lang); err == nil {
			c.SetLocale(tag)
			return c.Next()
		}
	}

	// 2. 解析 Accept-Language 头（支持权重）
	if lang := c.GetHeader(HeaderAcceptLanguage); lang != "" {
		if tags, _, _ := language.ParseAcceptLanguage(lang); len(tags) > 0 {
			// 如果有匹配器，使用匹配器选择最合适的语言
			if matcher := c.engine.languageMatcher; matcher != nil {
				tag, _, _ := matcher.Match(tags...)
				c.SetLocale(tag)
				return c.Next()
			}
		}
	}

	c.SetLocale(c.engine.defaultLang)
	return c.Next()
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

// useTranslator 使用绑定校验翻译器
func (e *Engine) useTranslator() {
	validate, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		panic("validator engine is not *validator.Validate")
	}

	validate.RegisterTagNameFunc(func(f reflect.StructField) string {
		// 优先使用 doc 标签
		if label, found := f.Tag.Lookup("doc"); found && label != "-" {
			return label
		}

		// 依次检查其他标签
		for _, tag := range []string{"json", "form", "header", "uri"} {
			if label := f.Tag.Get(tag); label != "" && label != "-" {
				return strings.Split(label, ",")[0] // "" => f.Name
			}
		}

		return f.Name
	})

	// 使用 Locales 配置
	e.defaultLang, e.languageMatcher, e.translator = localeComponents(validate, e.cfg.Locales...)
	if e.defaultLang != language.Und {
		e.Use(localeMiddleware)
	}
}
