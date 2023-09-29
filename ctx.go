package sgin

import (
	"bytes"
	"fmt"
	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
)

const (
	FormatJSON = "JSON"
	FormatXML  = "XML"
)

type Ctx struct {
	Request *http.Request
	Writer  http.ResponseWriter
	Params  gin.Params

	c    *gin.Context
	args any
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
			_ = dec.Decode(&c.args)
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

// ------ RESPONSE 响应 ------

func (c *Ctx) Send(body any, format ...string) error {
	_format(c.c, body, format...)
	return nil
}

func (c *Ctx) SendStatus(code int) error {
	c.c.AbortWithStatus(code)
	return nil
}

func (c *Ctx) SendFile(file string, attachment ...bool) error {
	filename := filepath.Base(file)
	if append(attachment, false)[0] {
		c.Header(HeaderContentDisposition, `attachment; filename*=UTF-8''`+url.QueryEscape(filename))
	}
	c.c.File(file)
	return nil
}

func (c *Ctx) Next() error {
	c.c.Next()
	return nil
}

// ------ SET 设置 ------

func (c *Ctx) Status(code int) *Ctx {
	c.c.Status(code)
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
	return c.c.Request.Method
}

func (c *Ctx) Header(key string, value ...string) string {
	if value != nil {
		c.c.Header(key, value[0])
		return ""
	}
	return c.c.GetHeader(key)
}

func (c *Ctx) HeaderOrQuery(key string) (value string) {
	if value = c.Header(key); value == "" {
		value = c.c.Query(key)
	}
	return
}

func (c *Ctx) StatusCode() int {
	return c.c.Writer.Status()
}

func (c *Ctx) Path() string {
	return c.c.Request.URL.Path
}

func (c *Ctx) IP() (ip string) {
	if ip = c.c.ClientIP(); ip == "::1" {
		ip = "127.0.0.1"
	}
	return ip
}
