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

	"github.com/bytedance/sonic"
	"github.com/clbanning/mxj/v2"
	"github.com/gin-gonic/gin"
	"github.com/rs/xid"
	"github.com/spf13/cast"
	"golang.org/x/text/language"
)

const (
	CtxKey    = "_baa/sgin/ctx"
	localeKey = "_baa/sgin/locale"
)

type Ctx struct {
	Request *http.Request
	Writer  gin.ResponseWriter
	Params  gin.Params
	Keys    map[string]any

	engine  *Engine
	ctx     *gin.Context
	cache   map[string]any // 缓存所有请求参数的键值
	traceid string         // 请求的 [跟踪ID]
}

func newCtx(ctx *gin.Context, e *Engine) *Ctx {
	c := &Ctx{
		engine:  e,
		ctx:     ctx,
		traceid: ctx.GetHeader(HeaderXRequestID),
		Request: ctx.Request,
		Writer:  ctx.Writer,
		Params:  ctx.Params,
		Keys:    ctx.Keys,
	}

	if c.traceid == "" {
		c.traceid = xid.New().String()
		c.Header(HeaderXRequestID, c.traceid)
	}

	return c
}

// ------ 参数获取 ------

// Values 获取所有参数 (Body 覆盖 Query)
func (c *Ctx) Values() map[string]any {
	if c.cache != nil {
		return c.cache
	}

	c.cache = map[string]any{}
	ct := c.GetHeader(HeaderContentType)

	_ = c.Request.ParseForm()
	for k, v := range c.Request.Form {
		if len(v) > 0 {
			c.cache[k] = v[0]
		}
	}

	// 处理 JSON
	if strings.HasPrefix(ct, MIMEJSON) {
		r := bytes.NewReader(c.RawBody())
		dec := sonic.ConfigDefault.NewDecoder(r)
		dec.UseNumber()
		_ = dec.Decode(&c.cache)
	} else if strings.HasPrefix(ct, MIMEXML) || strings.HasPrefix(ct, MIMETextXML) {
		// 处理 XML
		if m, _ := mxj.NewMapXml(c.RawBody()); m != nil {
			c.cache = m
		}
	} else if strings.HasPrefix(ct, MIMEMultipartForm) {
		// 处理 Multipart Form
		if form, err := c.ctx.MultipartForm(); err == nil {
			for k, v := range form.Value {
				if len(v) > 0 {
					c.cache[k] = v[0]
				}
			}
			for k, v := range form.File {
				if len(v) > 0 {
					c.cache[k] = v[0] // *multipart.FileHeader
				}
			}
		}
	}

	return c.cache
}

// Value 获取请求参数
func (c *Ctx) Value(key string, def ...string) string {
	if len(def) == 0 {
		return cast.ToString(c.ValueAny(key))
	}
	return cast.ToString(c.ValueAny(key, def[0]))
}

// ValueAny 获取原始类型的参数值
func (c *Ctx) ValueAny(key string, def ...any) any {
	v, ok := c.Values()[key]
	if !ok && len(def) > 0 {
		return def[0]
	}
	return v
}

func (c *Ctx) ValueInt(key string, def ...any) int {
	return cast.ToInt(c.ValueAny(key, def...))
}

func (c *Ctx) ValueInt64(key string, def ...any) int64 {
	return cast.ToInt64(c.ValueAny(key, def...))
}

func (c *Ctx) ValueFloat64(key string, def ...any) float64 {
	return cast.ToFloat64(c.ValueAny(key, def...))
}

func (c *Ctx) ValueBool(key string, def ...any) bool {
	return cast.ToBool(c.ValueAny(key, def...))
}

// ValueFile 获取上传的文件
func (c *Ctx) ValueFile(key string) (*multipart.FileHeader, error) {
	return c.ctx.FormFile(key)
}

func (c *Ctx) SaveFile(file *multipart.FileHeader, dst string) error {
	return c.ctx.SaveUploadedFile(file, dst)
}

// ------ 请求信息 ------

func (c *Ctx) Method() string {
	return c.Request.Method
}

func (c *Ctx) IP() string {
	return c.ctx.ClientIP()
}

// Path 返回请求路径
// 假设请求: /users/123/profile?view=full&name=John%20Doe
// Path() 返回: "/users/123/profile"
// Path(true) 返回: "/users/:id/profile"
func (c *Ctx) Path(full ...bool) string {
	if full != nil {
		return c.ctx.FullPath()
	}
	return c.ctx.Request.URL.Path
}

func (c *Ctx) Param(key string) string {
	return c.Params.ByName(key)
}

// GetHeader 获取 HTTP 请求头的值，如果不存在则返回可选的默认值。
func (c *Ctx) GetHeader(key string, value ...string) string {
	header := c.ctx.GetHeader(key)
	if header == "" && len(value) > 0 {
		return value[0]
	}
	return header
}

func (c *Ctx) StatusCode() int {
	return c.ctx.Writer.Status()
}

func (c *Ctx) RawBody() (body []byte) {
	if body, _ = c.Get(gin.BodyBytesKey).([]byte); body == nil {
		if body, _ = io.ReadAll(c.Request.Body); body != nil {
			c.Get(gin.BodyBytesKey, body)
		}
	}
	return body
}

func (c *Ctx) Cookie(name string) (string, error) {
	return c.ctx.Cookie(name)
}

func (c *Ctx) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	c.ctx.SetCookie(name, value, maxAge, path, domain, secure, httpOnly)
}

// ------ 响应控制 ------

func (c *Ctx) Send(body any) error {
	c.autoFormat(body)
	return nil
}

func (c *Ctx) Status(code int) *Ctx {
	c.Writer.WriteHeader(code)
	return c
}

func (c *Ctx) Header(key string, value string) *Ctx {
	c.ctx.Header(key, value)
	return c
}

func (c *Ctx) Content(value string) *Ctx {
	c.ctx.Header(HeaderContentType, value)
	return c
}

// ------ 上下文存储与中间件 ------

func (c *Ctx) Next() error {
	c.ctx.Next()
	return nil
}

// Get 设置或将值存储到上下文
func (c *Ctx) Get(key string, value ...any) any {
	if len(value) > 0 {
		c.ctx.Set(key, value[0])
		return value[0]
	}
	v, _ := c.ctx.Get(key)
	return v
}

// ------ 追踪与调试 ------

// TraceID 获取请求的 [跟踪ID]
func (c *Ctx) TraceID() string {
	return c.traceid
}

// Gin 返回底层的 *gin.Context
func (c *Ctx) Gin() *gin.Context {
	return c.ctx
}

// ------ 私有方法 ------

// locale 获取或设置验证器翻译语言
func (c *Ctx) locale(tag ...language.Tag) language.Tag {
	if len(tag) > 0 {
		c.ctx.Set(localeKey, tag[0])
		return tag[0]
	}
	lang, _ := c.Get(localeKey).(language.Tag)
	return lang
}

// send 消费内部归一化的响应结果
func (c *Ctx) send(r *result) {
	// 1. 处理状态码 (如果有显式设置)
	if r.Status > 0 {
		c.Status(r.Status)
	}

	// 2. 先处理错误
	if r.Err != nil {
		_ = c.engine.cfg.ErrorHandler(c, r.Err)
		return
	}

	// 3. 发送数据
	if r.Data != nil {
		_ = c.Send(r.Data)
	}
}

// autoFormat 自动根据 Accept 头返回对应类型的数据
func (c *Ctx) autoFormat(body any) {
	gc := c.ctx
	if gc.Abort(); body == nil { // 停止后续请求链的执行
		return
	}

	status := c.StatusCode() // HTTP 状态码

	// 1. 如果指定格式
	if f, ok := body.(Body); ok && f.format != "" {
		switch f.format {
		case FmtJSON:
			gc.JSON(status, f.data)
		case FmtXML:
			gc.XML(status, f.data)
		case FmtText:
			gc.String(status, fmt.Sprint(f.data))
		case FmtUpload, FmtDownload:
			file := fmt.Sprint(f.data)
			if f.format == FmtDownload {
				filename := filepath.Base(file)
				c.Header(HeaderContentDisposition, `attachment; filename*=UTF-8''`+url.QueryEscape(filename))
			}
			gc.File(file) // 将指定文件写入正体流
		case FmtHTML: // 发送 HTML
			gc.HTML(status, f.name, f.data)
		default:
			err := fmt.Errorf("unsupported response body format: '%s'", f.format)
			_ = c.engine.cfg.ErrorHandler(c, err)
		}

		return
	}

	// 2. 特殊类型处理
	if err, ok := body.(error); ok {
		_ = c.engine.cfg.ErrorHandler(c, err)
		return
	}

	// 3. 按 Accept 头协商
	// 如果明确只想要 XML 且不包含 HTML (排除浏览器默认行为)，才返回 XML。
	// 浏览器通常 Accept 包含 text/html 和 application/xml，这里避免误判。
	accept := c.GetHeader(HeaderAccept)
	if (strings.Contains(accept, MIMEXML) || strings.Contains(accept, MIMETextXML)) &&
		!strings.Contains(accept, MIMETextHTML) {
		gc.XML(status, body)
		return
	}

	// 4. 默认策略 (按类型推断)
	switch v := body.(type) {
	case string:
		// 如果是字符串，默认为 text/plain
		gc.String(status, v)
	default:
		// 其他类型（包括 struct, map, []byte, int 等），默认为 JSON
		gc.JSON(status, body)
	}
}
