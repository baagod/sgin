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
	"time"

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
	Uris    gin.Params
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
		Uris:    ctx.Params,
		Keys:    ctx.Keys,
	}

	if c.traceid == "" {
		c.traceid = xid.New().String()
		c.Header(HeaderXRequestID, c.traceid)
	}

	return c
}

// ------ 参数获取 ------

// Params 获取所有 请求 (含 Body) 参数
func (c *Ctx) Params() map[string]any {
	if c.cache != nil {
		return c.cache
	}

	c.cache = map[string]any{}
	ct := c.GetHeader(HeaderContentType)

	if strings.HasPrefix(ct, MIMEJSON) {
		r := bytes.NewReader(c.RawBody())
		dec := sonic.ConfigDefault.NewDecoder(r)
		dec.UseNumber()
		_ = dec.Decode(&c.cache)
	} else if strings.HasPrefix(ct, MIMEXML) || strings.HasPrefix(ct, MIMETextXML) {
		if m, _ := mxj.NewMapXml(c.RawBody()); m != nil {
			c.cache = m
		}
	}

	partForm, err := c.ctx.MultipartForm()

	for k, v := range c.Request.Form {
		if vLen := len(v); vLen == 1 {
			c.cache[k] = v[0]
		} else if vLen > 1 {
			c.cache[k] = v
		}
	}

	if err == nil && strings.HasPrefix(ct, MIMEMultipartForm) {
		for k, v := range partForm.Value {
			if vLen := len(v); vLen == 1 {
				c.cache[k] = v[0]
			} else if vLen > 1 {
				c.cache[k] = v
			}
		}
		for k, v := range partForm.File {
			if vLen := len(v); vLen == 1 {
				c.cache[k] = v[0]
			} else if vLen > 1 {
				c.cache[k] = v
			}
		}
	}

	return c.cache
}

// Param 获取请求参数
func (c *Ctx) Param(key string, def ...any) string {
	if len(def) == 0 {
		return cast.ToString(c.ParamAny(key))
	}
	return cast.ToString(c.ParamAny(key, def[0]))
}

// ParamAny 获取原始类型的参数值
func (c *Ctx) ParamAny(key string, def ...any) any {
	v, ok := c.Params()[key]
	if !ok && len(def) > 0 {
		return def[0]
	}
	return v
}

func (c *Ctx) ParamInt(key string, def ...any) int {
	return cast.ToInt(c.ParamAny(key, def...))
}

func (c *Ctx) ParamInt8(key string, def ...any) int8 {
	return cast.ToInt8(c.ParamAny(key, def...))
}

func (c *Ctx) ParamInt16(key string, def ...any) int16 {
	return cast.ToInt16(c.ParamAny(key, def...))
}

func (c *Ctx) ParamInt32(key string, def ...any) int32 {
	return cast.ToInt32(c.ParamAny(key, def...))
}

func (c *Ctx) ParamInt64(key string, def ...any) int64 {
	return cast.ToInt64(c.ParamAny(key, def...))
}

func (c *Ctx) ParamUint(key string, def ...any) uint {
	return cast.ToUint(c.ParamAny(key, def...))
}

func (c *Ctx) ParamUint8(key string, def ...any) uint8 {
	return cast.ToUint8(c.ParamAny(key, def...))
}

func (c *Ctx) ParamUint16(key string, def ...any) uint16 {
	return cast.ToUint16(c.ParamAny(key, def...))
}

func (c *Ctx) ParamUint32(key string, def ...any) uint32 {
	return cast.ToUint32(c.ParamAny(key, def...))
}

func (c *Ctx) ParamUint64(key string, def ...any) uint64 {
	return cast.ToUint64(c.ParamAny(key, def...))
}

func (c *Ctx) ParamFloat32(key string, def ...any) float32 {
	return cast.ToFloat32(c.ParamAny(key, def...))
}

func (c *Ctx) ParamFloat64(key string, def ...any) float64 {
	return cast.ToFloat64(c.ParamAny(key, def...))
}

func (c *Ctx) ParamBool(key string, def ...any) bool {
	return cast.ToBool(c.ParamAny(key, def...))
}

func (c *Ctx) ParamTime(key string, def ...any) time.Time {
	return cast.ToTime(c.ParamAny(key, def...))
}

func (c *Ctx) ParamDuration(key string, def ...any) time.Duration {
	return cast.ToDuration(c.ParamAny(key, def...))
}

// ParamFile 获取上传的文件
func (c *Ctx) ParamFile(key string) (*multipart.FileHeader, error) {
	return c.ctx.FormFile(key)
}

func (c *Ctx) ParamArray(key string) []string {
	return cast.ToStringSlice(c.ParamAny(key))
}

func (c *Ctx) ParamMap(key string) map[string]string {
	return cast.ToStringMapString(c.ParamAny(key))
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

func (c *Ctx) RemoteIP() string {
	return c.ctx.RemoteIP()
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

func (c *Ctx) Uri(key string) string {
	return c.ctx.Param(key)
}

func (c *Ctx) AddUri(key, value string) *Ctx {
	c.ctx.AddParam(key, value)
	return c
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

func (c *Ctx) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool, samesite ...http.SameSite) *Ctx {
	if len(samesite) > 0 {
		c.ctx.SetSameSite(samesite[0])
	}
	c.ctx.SetCookie(name, value, maxAge, path, domain, secure, httpOnly)
	return c
}

// ------ 响应控制 ------

func (c *Ctx) Send(body any) error {
	c.autoFormat(body)
	return nil
}

func (c *Ctx) SendJSON(data any) error {
	c.ctx.Abort()
	c.ctx.JSON(c.StatusCode(), data)
	return nil
}

func (c *Ctx) SendXML(data any) error {
	c.ctx.Abort()
	c.ctx.XML(c.StatusCode(), data)
	return nil
}

func (c *Ctx) SendText(data any) error {
	c.ctx.Abort()
	c.ctx.String(c.StatusCode(), fmt.Sprint(data))
	return nil
}

func (c *Ctx) SendYAML(data any) error {
	c.ctx.Abort()
	c.ctx.YAML(c.StatusCode(), data)
	return nil
}

func (c *Ctx) SendTOML(data any) error {
	c.ctx.Abort()
	c.ctx.TOML(c.StatusCode(), data)
	return nil
}

func (c *Ctx) SendHTML(name string, data any) error {
	c.ctx.Abort()
	c.ctx.HTML(c.StatusCode(), name, data)
	return nil
}

func (c *Ctx) SendBinary(data []byte) error {
	c.ctx.Abort()
	c.ctx.Data(c.StatusCode(), c.GetHeader(HeaderContentType), data)
	return nil
}

// SendFile 发送文件，传递 name 为下载流。
func (c *Ctx) SendFile(file string, name ...string) error {
	if c.ctx.Abort(); len(name) > 0 {
		filename := filepath.Base(file)
		if name[0] != "" {
			filename = name[0]
		}
		disposition := `attachment; filename*=UTF-8''` + url.QueryEscape(filename)
		c.Header(HeaderContentDisposition, disposition)
	}
	c.ctx.File(file)
	return nil
}

// SendStream 发送流响应并返回布尔值，标识 “客户端是否在流中间断开连接”。
func (c *Ctx) SendStream(step func(w io.Writer) bool) bool {
	return c.ctx.Stream(step)
}

// SendReader 从 io.Reader 发送数据
// size: 数据长度 (如果未知传 -1)
// contentType: 显式指定类型，为空则尝试自动探测或使用 header。
func (c *Ctx) SendReader(reader io.Reader, size int64, extraHeaders ...map[string]string) error {
	c.ctx.Abort()
	ct := c.GetHeader(HeaderContentType)
	var headers map[string]string

	if len(extraHeaders) > 0 {
		headers = extraHeaders[0]
	}

	c.ctx.DataFromReader(c.StatusCode(), size, ct, reader, headers)
	return nil
}

// SendSSEvent 将服务器发送事件写入正体流
func (c *Ctx) SendSSEvent(name string, message any) error {
	c.ctx.Abort()
	c.ctx.SSEvent(name, message)
	return nil
}

// Redirect 返回到特定位置的 HTTP 重定向
func (c *Ctx) Redirect(loc string) error {
	c.ctx.Abort()
	c.ctx.Redirect(c.StatusCode(), loc)
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

// Get 设置或将值存储到上下文，不会发生 panic。
func (c *Ctx) Get(key string, value ...any) any {
	if len(value) > 0 {
		c.ctx.Set(key, value[0])
		return value[0]
	}
	if v, ok := c.ctx.Get(key); ok {
		return v
	}
	return c.ctx.Value(key)
}

func (c *Ctx) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

func (c *Ctx) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c *Ctx) Err() error {
	return c.ctx.Err()
}

func (c *Ctx) Value(key any) any {
	return c.ctx.Value(key)
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

// send 发送响应结果
func (c *Ctx) send(data any, err error) {
	if err != nil { // 先处理错误
		_ = c.engine.cfg.ErrorHandler(c, err)
		return
	}

	if data != nil {
		_ = c.Send(data) // 发送数据
	}
}

// autoFormat 自动根据 Accept 头返回对应类型的数据
func (c *Ctx) autoFormat(body any) {
	gc := c.ctx
	if gc.Abort(); body == nil { // 停止后续请求链的执行
		return
	}

	// 处理错误
	if err, ok := body.(error); ok {
		_ = c.engine.cfg.ErrorHandler(c, err)
		return
	}

	// Accept 前缀是 "text/html" 为浏览器直接访问，直接返回 JSON。
	if strings.HasPrefix(c.GetHeader(HeaderAccept), MIMETextHTML) {
		_ = c.SendJSON(body)
		return
	}

	// 其他情况，使用 Gin Negotiate 进行正常内容协商。
	gc.Negotiate(c.StatusCode(), gin.Negotiate{
		Offered: []string{
			MIMEJSON, MIMETextHTML, MIMEXML,
			MIMETextXML, MIMEYAML, MIMEYAMLX, MIMETOML,
		},
		Data: body,
	})
}
