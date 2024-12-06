package sgin

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/clbanning/mxj/v2"
	"github.com/spf13/cast"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
)

const (
	FormatXML      = "XML"
	FormatJSON     = "JSON"
	FormatString   = "String"
	FormatUpload   = "Upload"
	FormatDownload = "Download"
)

type Ctx struct {
	Request *http.Request
	Writer  gin.ResponseWriter
	Params  gin.Params
	Keys    map[string]any

	ctx    *gin.Context
	engine *Engine
	args   any
}

func newCtx(ctx *gin.Context, e *Engine) *Ctx {
	return &Ctx{
		engine:  e,
		ctx:     ctx,
		Request: ctx.Request,
		Writer:  ctx.Writer,
		Params:  ctx.Params,
		Keys:    ctx.Keys,
	}
}

// ------ 请求参数 ------

func (c *Ctx) Args() (args map[string]any) {
	// 已经解析过请求数据
	if args, _ = c.args.(map[string]any); args != nil {
		return
	}

	args = map[string]any{}
	ct := c.Header(HeaderContentType)

	if ct == "" || c.Request.Method == "GET" || ct == gin.MIMEPOSTForm {
		_ = c.Request.ParseForm()
		for k, v := range c.Request.Form {
			args[k] = v[0]
		}
	} else if strings.HasPrefix(ct, gin.MIMEMultipartPOSTForm) {
		if form, err := c.ctx.MultipartForm(); err == nil {
			for k, v := range form.Value {
				args[k] = v[0]
			}
			for k, v := range form.File {
				args[k] = v[0]
			}
		}
	}

	switch ct {
	case gin.MIMEJSON:
		r := bytes.NewReader(c.RawBody())
		dec := sonic.ConfigDefault.NewDecoder(r)
		dec.UseNumber()
		_ = dec.Decode(&args)
	case gin.MIMEXML:
		if m, _ := mxj.NewMapXml(c.RawBody()); m != nil {
			args = m
		}
	}

	c.args = args
	return args
}

func (c *Ctx) Arg(key string, or ...string) string {
	if v, ok := c.Args()[key]; ok {
		return fmt.Sprint(v)
	}
	return append(or, "")[0]
}

func (c *Ctx) ArgInt(key string, or ...int) int {
	v, err := cast.ToIntE(c.Arg(key))
	if err != nil && or != nil {
		return or[0]
	}
	return v
}

func (c *Ctx) ArgInt64(key string, or ...int64) int64 {
	v, err := cast.ToInt64E(c.Arg(key))
	if err != nil && or != nil {
		return or[0]
	}
	return v
}

func (c *Ctx) ArgFloat64(key string, or ...float64) float64 {
	v, err := cast.ToFloat64E(c.Arg(key))
	if err != nil && or != nil {
		return or[0]
	}
	return v
}

func (c *Ctx) ArgBool(key string) bool {
	return cast.ToBool(c.Arg(key))
}

// ------ 响应 ------

func (c *Ctx) Send(body any, format ...string) error {
	c.autoFormat(body, format...)
	return nil
}

func (c *Ctx) SendHTML(name string, data any) error {
	c.ctx.Abort()
	c.ctx.HTML(c.StatusCode(), name, data)
	return nil
}

func (c *Ctx) Next() error {
	c.ctx.Next()
	return nil
}

// ------ SET 设置 ------

func (c *Ctx) Status(code int) *Ctx {
	c.Writer.WriteHeader(code)
	return c
}

// Locals 设置或将值存储到上下文
func (c *Ctx) Locals(key string, value ...any) any {
	if value != nil {
		c.ctx.Set(key, value[0])
		return nil
	}
	v, _ := c.ctx.Get(key)
	return v
}

// ------ GET 获取 ------

func (c *Ctx) Method() string {
	return c.Request.Method
}

// Header 获取 HTTP 请求头的值，如果不存在则返回可选的默认值。
func (c *Ctx) Header(key string, value ...string) string {
	header := c.ctx.GetHeader(key)
	if header == "" && value != nil {
		return value[0]
	}
	return header
}

func (c *Ctx) SetHeader(key string, value string) {
	c.ctx.Header(key, value)
}

func (c *Ctx) RawBody() (body []byte) {
	if body, _ = c.Locals(gin.BodyBytesKey).([]byte); body == nil {
		if body, _ = io.ReadAll(c.Request.Body); body != nil {
			c.Locals(gin.BodyBytesKey, body)
		}
	}
	return body
}

func (c *Ctx) StatusCode() int {
	return c.ctx.Writer.Status()
}

func (c *Ctx) Path(full ...bool) string {
	if full != nil {
		return c.ctx.FullPath()
	}
	return c.ctx.Request.URL.Path
}

func (c *Ctx) Param(key string) string {
	return c.Params.ByName(key)
}

func (c *Ctx) IP() (ip string) {
	if ip = c.ctx.ClientIP(); ip == "::1" {
		ip = "127.0.0.1"
	}
	return ip
}

func (c *Ctx) Cookie(name string) (string, error) {
	return c.ctx.Cookie(name)
}

func (c *Ctx) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	c.ctx.SetCookie(name, value, maxAge, path, domain, secure, httpOnly)
}

func (c *Ctx) SaveFile(file *multipart.FileHeader, dst string) error {
	return c.ctx.SaveUploadedFile(file, dst)
}

// autoFormat 自动根据 Accept 头返回对应类型的数据
func (c *Ctx) autoFormat(body any, format ...string) {
	ginCtx := c.ctx
	if ginCtx.Abort(); body == nil { // 停止请求链
		return
	}

	// 上传或下载文件
	responseType := append(format, "")[0]
	if responseType == FormatUpload || responseType == FormatDownload {
		file := fmt.Sprint(body)
		filename := filepath.Base(file)
		if responseType == FormatDownload {
			c.Header(HeaderContentDisposition, `attachment; filename*=UTF-8''`+url.QueryEscape(filename))
		}
		ginCtx.File(file)
		return
	}

	// 返回状态码、字符串、错误
	switch b := body.(type) {
	case int:
		ginCtx.Status(b)
		ginCtx.Writer.WriteHeaderNow()
		return
	case error:
		_ = c.engine.config.ErrorHandler(c, b)
		return
	}

	// 返回响应体
	status := c.StatusCode()
	accept := c.Header(HeaderAccept)

	if responseType == FormatJSON || strings.Contains(accept, gin.MIMEJSON) {
		ginCtx.JSON(status, body)
		return
	}

	if responseType == FormatXML || strings.Contains(accept, gin.MIMEXML) {
		if b, ok := body.([]byte); ok { // 返回 []byte
			body = string(b)
		}
		if s, ok := body.(string); ok { // 返回 string
			ginCtx.String(status, s)
			return
		}
		ginCtx.XML(status, body) // 返回其他
		return
	}

	bodyType := fmt.Sprintf("%T", body)
	if responseType == FormatString ||
		strings.Contains(accept, gin.MIMEHTML) ||
		strings.Contains(accept, gin.MIMEPlain) ||
		bodyType == "string" ||
		bodyType == "[]uint8" /* []byte */ {
		ginCtx.String(status, fmt.Sprintf("%s", body))
		return
	}

	ginCtx.JSON(status, body) // 默认返回 JSON
}
