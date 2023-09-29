package sgin

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"io"
	"net/http"
	"reflect"
	"strings"
)

var ginCtxType = reflect.TypeOf(&gin.Context{})

type (
	AnyHandler any
	Handler    struct {
		Binding []binding.Binding
		Fn      AnyHandler
		Error   func(*gin.Context, error)
	}
)

func handle(r *RouterGroup, a ...AnyHandler) (handlers []gin.HandlerFunc) {
	for _, f := range a {
		if v, ok := f.(gin.HandlerFunc); ok {
			handlers = append(handlers, v)
			continue
		}

		var h *Handler
		if h, _ = f.(*Handler); h == nil {
			h = &Handler{Fn: f}
		}

		fn := reflect.ValueOf(h.Fn)
		fnT := fn.Type()

		handlers = append(handlers, func(c *gin.Context) {
			var in []reflect.Value       // 输入参数 | 0: 上下文, 1: 输入参数
			if fnT.In(0) == ginCtxType { // 如果第一个参数是 *gin.Context
				in = append(in, reflect.ValueOf(c))
			} else { // 如果不是，则第一个参数必须是 *sgin.Ctx，否则会出错。
				ctx, _ := c.Keys["_sgin/ctxkey"].(*Ctx)
				if ctx == nil {
					ctx = &Ctx{c: c, Request: c.Request, Writer: c.Writer, Params: c.Params}
					c.Set("_sgin/ctxkey", ctx)
				}
				in = append(in, reflect.ValueOf(ctx))
			}

			if fnT.NumIn() == 2 { // 有绑定参数
				val, err := bindIn(c, h.Binding, fnT.In(1)) // 绑定输入参数
				if err != nil {                             // 处理错误
					if c.Abort(); h.Error != nil {
						h.Error(c, err)
					} else if r.error != nil {
						r.error(c, err)
					} else {
						c.String(http.StatusInternalServerError, err.Error())
					}
					return
				}
				in = append(in, val)
			}

			retResponse(c, fn.Call(in)...)
		})
	}

	return handlers
}

func bindIn(c *gin.Context, bindings []binding.Binding, T reflect.Type) (v reflect.Value, err error) {
	v = reflect.New(T.Elem()) // *T-value
	ptr := v.Interface()
	var names = map[string]bool{}

	if bindings != nil {
		for _, b := range bindings {
			name := b.Name()
			names[name] = true
			if vu, ok := b.(binding.BindingUri); ok {
				m := make(map[string][]string)
				for _, x := range c.Params {
					m[x.Key] = []string{x.Value}
				}
				if err = vu.BindUri(m, ptr); err != nil {
					return
				}
				continue
			}
			if bb, ok := b.(binding.BindingBody); ok {
				if err = c.ShouldBindBodyWith(ptr, bb); err != nil {
					if err == io.EOF {
						text := fmt.Sprintf("bind %s error: %v", name, err)
						err = errors.New(text)
					}
					return
				}
				continue
			}
			if err = c.ShouldBindWith(ptr, b); err != nil {
				return
			}
		}
	}

	ct := c.ContentType()
	if _, ok := names["form"]; !ok && ct == "" {
		err = c.ShouldBindWith(ptr, binding.Form)
	}

	if _, ok := names["json"]; !ok && ct == gin.MIMEJSON {
		err = c.ShouldBindBodyWith(ptr, binding.JSON)
	} else if _, ok = names["xml"]; !ok && ct == gin.MIMEXML || ct == gin.MIMEXML2 {
		err = c.ShouldBindBodyWith(ptr, binding.XML)
	} else if _, ok = names["toml"]; !ok && ct == gin.MIMETOML {
		err = c.ShouldBindBodyWith(ptr, binding.TOML)
	} else if _, ok = names["yaml"]; !ok && ct == gin.MIMEYAML {
		err = c.ShouldBindBodyWith(ptr, binding.YAML)
	} else if _, ok = names["form"]; !ok && ct == gin.MIMEPOSTForm || ct == gin.MIMEMultipartPOSTForm {
		err = c.ShouldBindWith(ptr, binding.Form)
	}

	return
}

func retResponse(c *gin.Context, out ...reflect.Value) {
	if out == nil {
		return
	}

	body := out[0].Interface()
	if body == nil {
		return
	}

	if st, ok := body.(int); ok { // 如果第一个输出参数是 int 类型
		if len(out) == 1 { // 如果只有一个输出参数
			c.AbortWithStatus(st)
			return
		}
		c.Status(st) // 设置状态码为 st
	}

	if len(out) == 2 { // 如果有两个输入参数，则第一个 st (状态码)，第二个为 body。
		body = out[1].Interface() // 获取第二个输入参数
	}

	_format(c, body)
}

func _format(c *gin.Context, body any, format ...string) {
	if c.Abort(); body == nil { // 停止继续处理
		return
	}

	status := c.Writer.Status()
	accept := c.GetHeader(HeaderAccept)

	switch body.(type) {
	case string, error:
		c.String(status, fmt.Sprint(body))
		return
	}

	f := append(format, "")[0]
	if f == FormatJSON || strings.Contains(accept, gin.MIMEJSON) { // 优先返回 JSON
		c.JSON(status, body)
		return
	} else if f == FormatXML || strings.Contains(accept, gin.MIMEXML) || strings.Contains(accept, gin.MIMEXML2) {
		c.XML(status, body)
		return
	} else if strings.Contains(accept, gin.MIMEHTML) || strings.Contains(accept, gin.MIMEPlain) {
		c.String(status, fmt.Sprint(body))
		return
	}

	c.JSON(status, body) // 默认返回 JSON
}
