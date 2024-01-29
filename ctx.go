package sgin

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
)

const (
	FmtJSON   = "JSON"
	FmtXML    = "XML"
	FmtFile   = "File"
	FmtDown   = "Download"
	FmtStatus = "Status"
)

type Ctx struct {
	Request *http.Request
	Writer  gin.ResponseWriter
	Params  gin.Params
	Keys    map[string]any

	c      *gin.Context
	engine *Engine
	args   any
}

func newCtx(c *gin.Context, app *Engine) *Ctx {
	return &Ctx{
		c:       c,
		engine:  app,
		Request: c.Request,
		Writer:  c.Writer,
		Params:  c.Params,
		Keys:    c.Keys,
	}
}

// ------ ARGS 参数 ------

func (c *Ctx) Args() (args map[string]any) {
	if args, _ = c.args.(map[string]any); args != nil {
		return
	}

	ct := c.Header(HeaderContentType)
	if ct != "" && ct != gin.MIMEPOSTForm &&
		ct != gin.MIMEMultipartPOSTForm && ct != gin.MIMEJSON {
		return map[string]any{}
	}

	args = map[string]any{}
	_ = c.Request.ParseMultipartForm(32 << 20)
	for k, v := range c.Request.Form {
		args[k] = v[0]
	}

	switch ct {
	case gin.MIMEJSON:
		body, ok := c.Set(gin.BodyBytesKey).([]byte)
		if !ok {
			if body, _ = io.ReadAll(c.Request.Body); body != nil {
				c.Set(gin.BodyBytesKey, body)
			}
		}
		if body != nil {
			dec := sonic.ConfigDefault.NewDecoder(bytes.NewReader(body))
			dec.UseNumber()
			_ = dec.Decode(&args)
			c.args = args
		}
	}

	return args
}

func (c *Ctx) Arg(key string, e ...string) string {
	if v, ok := c.Args()[key]; ok {
		return fmt.Sprint(v)
	}
	return append(e, "")[0]
}

func (c *Ctx) ArgInt(key string, e ...int) int {
	v, err := strconv.Atoi(c.Arg(key))
	if err != nil && e != nil {
		v = e[0]
	}
	return v
}

func (c *Ctx) ArgInt64(key string, e ...int64) int64 {
	v, err := strconv.ParseInt(c.Arg(key), 10, 64)
	if err != nil && e != nil {
		v = e[0]
	}
	return v
}

func (c *Ctx) ArgBool(key string) bool {
	v := c.Arg(key)
	return v != "" && v != "0"
}

func (c *Ctx) MultipartForm() (*multipart.Form, error) {
	return c.c.MultipartForm()
}

// ------ RESPONSE 响应 ------

func (c *Ctx) Send(body any, format ...string) error {
	c.format(body, format...)
	return nil
}

func (c *Ctx) SendHTML(name string, data any) error {
	c.c.Abort()
	c.c.HTML(c.StatusCode(), name, data)
	return nil
}

func (c *Ctx) Next() error {
	c.c.Next()
	return nil
}

// ------ SET 设置 ------

func (c *Ctx) Status(code int) *Ctx {
	c.Writer.WriteHeader(code)
	return c
}

func (c *Ctx) Set(key string, value ...any) any {
	if value != nil {
		c.c.Set(key, value[0])
		return nil
	}
	v, _ := c.c.Get(key)
	return v
}

// ------ GET 获取 ------

func (c *Ctx) Method() string {
	return c.Request.Method
}

func (c *Ctx) Header(key string, value ...string) string {
	if value != nil {
		c.c.Header(key, value[0])
		return ""
	}
	return c.c.GetHeader(key)
}

func (c *Ctx) HeaderOrQuery(key string) (value string) {
	if value = c.c.GetHeader(key); value == "" {
		value = c.c.Query(key)
	}
	return value
}

func (c *Ctx) RawBody() []byte {
	body, ok := c.Set(gin.BodyBytesKey).([]byte)
	if !ok {
		if body, _ = io.ReadAll(c.Request.Body); body != nil {
			c.Set(gin.BodyBytesKey, body)
		}
	}
	return body
}

func (c *Ctx) GinCtx() *gin.Context {
	return c.c
}

func (c *Ctx) StatusCode() int {
	return c.c.Writer.Status()
}

func (c *Ctx) Path(full ...bool) string {
	if full != nil {
		return c.c.FullPath()
	}
	return c.c.Request.URL.Path
}

func (c *Ctx) Param(key string) string {
	return c.Params.ByName(key)
}

func (c *Ctx) IP() (ip string) {
	if ip = c.c.ClientIP(); ip == "::1" {
		ip = "127.0.0.1"
	}
	return ip
}

func (c *Ctx) Cookie(name string) (string, error) {
	return c.c.Cookie(name)
}

func (c *Ctx) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	c.c.SetCookie(name, value, maxAge, path, domain, secure, httpOnly)
}

func (c *Ctx) SaveFile(file *multipart.FileHeader, dst string) error {
	return c.c.SaveUploadedFile(file, dst)
}

func (c *Ctx) format(body any, format ...string) {
	gc := c.c
	if gc.Abort(); body == nil { // 停止继续处理
		return
	}

	if st, ok := body.(int); ok {
		gc.Status(st)
		gc.Writer.WriteHeaderNow()
		return
	}

	fmtStr := append(format, "")[0]
	if fmtStr == FmtFile || fmtStr == FmtDown {
		file := fmt.Sprint(body)
		filename := filepath.Base(file)
		if fmtStr == FmtDown {
			c.Header(HeaderContentDisposition, `attachment; filename*=UTF-8''`+url.QueryEscape(filename))
		}
		gc.File(file)
		return
	}

	switch b := body.(type) {
	case string:
		gc.String(c.StatusCode(), b)
		return
	case error:
		_ = c.engine.errHandler(c, b)
		return
	}

	status := c.StatusCode()
	accept := c.Header(HeaderAccept)

	if fmtStr == FmtJSON || strings.Contains(accept, gin.MIMEJSON) { // 优先返回 JSON
		gc.JSON(status, body)
		return
	} else if fmtStr == FmtXML || strings.Contains(accept, gin.MIMEXML) || strings.Contains(accept, gin.MIMEXML2) {
		gc.XML(status, body)
		return
	} else if strings.Contains(accept, gin.MIMEHTML) || strings.Contains(accept, gin.MIMEPlain) {
		gc.String(status, fmt.Sprint(body))
		return
	}

	gc.JSON(status, body) // 默认返回 JSON
}
