package sgin

import (
	"errors"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler any // func(*Ctx[, T]) -> T | (int, T) | (T, error)

func handler(r *Routers, a ...Handler) (handlers []gin.HandlerFunc) {
	for _, f := range a {
		switch ginHandler := f.(type) {
		case gin.HandlerFunc:
			handlers = append(handlers, ginHandler)
			continue
		case func(*gin.Context):
			handlers = append(handlers, ginHandler)
			continue
		}

		handler := reflect.ValueOf(f)
		handlerType := handler.Type()

		handlers = append(handlers, func(ginCtx *gin.Context) {
			ctx, _ := ginCtx.Keys["_baa/sgin/ctxkey"].(*Ctx)
			if ctx == nil {
				ctx = newCtx(ginCtx, r.engine)      // 创建 *sgin.Ctx
				ginCtx.Set("_baa/sgin/ctxkey", ctx) // 保存 *sgin.Ctx
			}

			inputParam := []reflect.Value{reflect.ValueOf(ctx)} // *Ctx[, T]
			if handlerType.NumIn() == 2 {                       // 如果处理函数有两个参数
				value, err := bind(ginCtx, handlerType.In(1)) // 创建并绑定请求结构体
				if err != nil {                               // 处理错误
					ginCtx.Abort()                                                      // 停止请求链
					_ = r.engine.config.ErrorHandler(ctx, &Error{Message: err.Error()}) // 返回错误
					return
				}
				inputParam = append(inputParam, value)
			}

			response(ctx, handler.Call(inputParam))
		})
	}

	return
}

// bind 绑定请求结构体
func bind(c *gin.Context, T reflect.Type) (value reflect.Value, err error) {
	isStructPtr := T.Kind() == reflect.Ptr
	if isStructPtr {
		T = T.Elem()
	}

	value = reflect.New(T)   // 创建结构体 *T 的 reflect.Value
	ptr := value.Interface() // 结构体指针
	ct := c.ContentType()

	if c.Request.Method == "GET" || ct == gin.MIMEPOSTForm || strings.HasPrefix(ct, gin.MIMEMultipartPOSTForm) {
		err = c.ShouldBind(ptr)
	} else if ct == gin.MIMEJSON {
		err = c.ShouldBindJSON(ptr)
	} else if ct == gin.MIMEXML {
		err = c.ShouldBindXML(ptr)
	}

	var vErrs validator.ValidationErrors
	if errors.As(err, &vErrs) {
		for _, e := range vErrs {
			if field, ok := T.FieldByName(e.Field()); ok {
				if failtip := field.Tag.Get("failtip"); failtip != "" {
					err = errors.New(failtip)
					break
				}
			}
		}
	}

	if !isStructPtr {
		value = value.Elem()
	}

	return
}

// response 返回响应 (处理函数的给定返回值)
// 返回值可以为：T | (int, T) | (T, error)
func response(c *Ctx, result []reflect.Value) {
	if result == nil { // 没有返回值
		return
	}

	first := result[0].Interface() // 第一个返回值
	if len(result) == 1 {
		c.format(first)
		return
	}

	second := result[1].Interface()        // 第二个返回值
	if statusCode, ok := first.(int); ok { // (int, T)
		c.Status(statusCode).format(second)
		return
	}

	if err, ok := second.(error); ok { // (T, error)
		c.format(err)
		return
	}

	c.format(first) // 没有错误，返回 T
}
