package sgin

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"reflect"
	"strings"
)

type AnyHandler any

var ginCtxType = reflect.TypeOf(&gin.Context{})

// Handler 处理器
type Handler struct {
	Bindings *BindingOption
	Fn       AnyHandler
	Error    func(*gin.Context, error)
}

func bindIn(c *gin.Context, bindOpts *BindingOption, T reflect.Type) (v reflect.Value, err error) {
	v = reflect.New(T.Elem()) // *T-value
	ptr := v.Interface()
	var names = map[string]bool{"form": false, "xml": false, "toml": false, "yaml": false, "json": false}

	if bindOpts != nil {
		if bindOpts.Uri != nil {
			if err = c.ShouldBindUri(ptr); err != nil {
				return
			}
		}

		for _, x := range bindOpts.Bindings {
			names[x.Name()] = true
			if bb, ok := x.(binding.BindingBody); ok {
				if err = c.ShouldBindBodyWith(ptr, bb); err != nil {
					return
				}
				continue
			}
			if err = c.ShouldBindWith(ptr, x); err != nil {
				return
			}
		}
	}

	if ct := c.ContentType(); ct == gin.MIMEJSON && !names["json"] {
		err = c.ShouldBindBodyWith(ptr, binding.JSON)
	} else if (ct == gin.MIMEXML || ct == gin.MIMEXML2) && !names["xml"] {
		err = c.ShouldBindBodyWith(ptr, binding.XML)
	} else if ct == gin.MIMETOML && !names["toml"] {
		err = c.ShouldBindBodyWith(ptr, binding.TOML)
	} else if ct == gin.MIMEYAML && !names["yaml"] {
		err = c.ShouldBindBodyWith(ptr, binding.YAML)
	} else if ct == "" || ct == gin.MIMEPOSTForm || ct == gin.MIMEMultipartPOSTForm {
		if !names["form"] {
			err = c.ShouldBindWith(ptr, binding.Form)
		}
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

	format(c, body)
}

func format(c *gin.Context, body any, format ...string) {
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
