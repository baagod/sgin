package sgin

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

type (
	ginHandler = func(*gin.Context)
	AnyHandler = any // func(<*gin.Context | *Ctx>[, *T]) <error | T> | (int, T) | (T, error)
	Handler    struct {
		Binding []binding.Binding
		Fn      AnyHandler
	}
)

func handle(r *Routers, a ...AnyHandler) (handlers []gin.HandlerFunc) {
	for _, f := range a {
		switch fn := f.(type) {
		case gin.HandlerFunc:
			handlers = append(handlers, fn)
			continue
		case ginHandler:
			handlers = append(handlers, fn)
			continue
		}

		var h *Handler
		if h, _ = f.(*Handler); h == nil {
			h = &Handler{Fn: f}
		}

		fn := reflect.ValueOf(h.Fn)
		fnT := fn.Type()

		handlers = append(handlers, func(gc *gin.Context) {
			c, _ := gc.Keys["_baa/sgin/ctxkey"].(*Ctx)
			if c == nil {
				c = newCtx(gc, r.engine)
				gc.Set("_baa/sgin/ctxkey", c)
			}
			in := []reflect.Value{reflect.ValueOf(c)}
			if fnT.NumIn() == 2 {
				v, err := bindIn(gc, h.Binding, fnT.In(1))
				if err != nil { // 处理错误
					gc.Abort()
					_ = r.engine.errHandler(c, &Error{Message: err.Error()})
					return
				}
				in = append(in, v)
			}

			if err := response(c, fn.Call(in)); err != nil {
				_ = r.engine.errHandler(c, err)
			}
		})
	}

	return
}

func bindIn(c *gin.Context, bindings []binding.Binding, T reflect.Type) (v reflect.Value, err error) {
	elem := T.Elem()      //  T: struct
	v = reflect.New(elem) // *T: object
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
			} else if bb, ok := b.(binding.BindingBody); ok {
				if err = c.ShouldBindBodyWith(ptr, bb); err != nil {
					if err == io.EOF {
						text := fmt.Sprintf("bind %s error: %v", name, err)
						err = errors.New(text)
					}
					return
				}
			} else if err = c.ShouldBindWith(ptr, b); err != nil {
				return
			}
		}
	}

	ct := c.ContentType()
	if _, ok := names["form"]; !ok &&
		c.Request.Method == "GET" ||
		ct == gin.MIMEPOSTForm ||
		strings.HasPrefix(ct, gin.MIMEMultipartPOSTForm) {
		err = c.ShouldBind(ptr)
	} else {
		if _, ok := names["json"]; !ok && ct == gin.MIMEJSON {
			err = c.ShouldBindBodyWith(ptr, binding.JSON)
		} else if _, ok = names["xml"]; !ok && ct == gin.MIMEXML || ct == gin.MIMEXML2 {
			err = c.ShouldBindBodyWith(ptr, binding.XML)
		} else if _, ok = names["toml"]; !ok && ct == gin.MIMETOML {
			err = c.ShouldBindBodyWith(ptr, binding.TOML)
		} else if _, ok = names["yaml"]; !ok && ct == gin.MIMEYAML {
			err = c.ShouldBindBodyWith(ptr, binding.YAML)
		}
	}

	var vErrs validator.ValidationErrors
	if errors.As(err, &vErrs) {
		for _, e := range vErrs {
			if field, ok := elem.FieldByName(e.Field()); ok {
				if failtip := field.Tag.Get("failtip"); failtip != "" {
					err = errors.New(failtip)
					break
				}
			}
		}
	}

	return
}

func response(c *Ctx, ret []reflect.Value) (err error) {
	if ret == nil {
		return
	}

	first := ret[0].Interface()
	if first == nil {
		return
	}

	if len(ret) == 1 { // [error | T]
		//goland:noinspection GoTypeAssertionOnErrors
		if err, _ = first.(error); err != nil {
			return
		}

		c.format(first)
		return
	} else if first != nil { // (int, T) or (T, error)
		if st, ok := first.(int); ok { // (int, T)
			c.Status(st).format(ret[1].Interface())
		} else { // (T, error)
			c.format(first)
		}
		return
	}

	return ret[1].Interface().(error) // (T, error)
}
