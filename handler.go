package sgin

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"reflect"
)

var ginCtxType = reflect.TypeOf(&gin.Context{})

func HandlerFunc(a ...Handler) []gin.HandlerFunc {
	var handlers []gin.HandlerFunc

	for _, f := range a {
		if v, ok := f.(gin.HandlerFunc); ok {
			handlers = append(handlers, v)
			continue
		}

		fv := reflect.ValueOf(f) // 处理函数的 reflect.Value
		args := fv.Type()        // 处理函数的 reflect.Type

		handlers = append(handlers, func(c *gin.Context) {
			var in []reflect.Value                        // 输入参数 | 0: 上下文, 1: 输入参数
			if first := args.In(0); first == ginCtxType { // 如果第一个参数是 *gin.Context
				in = append(in, reflect.ValueOf(c))
			} else { // 如果不是，则第一个参数必须是 *sgin.Ctx，否则会出错。
				ctx, _ := c.Keys["_sgin/ctxkey"].(*Ctx)
				if ctx == nil {
					ctx = &Ctx{
						c:       c,
						Request: c.Request,
						Writer:  c.Writer,
						Params:  c.Params,
					}
					c.Set("_sgin/ctxkey", ctx)
				}
				in = append(in, reflect.ValueOf(ctx))
			}

			if args.NumIn() == 2 { // 如果有第二个参数，则被视为要绑定的输入参数。如查询参数、表单参数。
				var err error
				p := reflect.New(args.In(1)) // 输入参数的 reflect.Value

				switch c.ContentType() {
				case gin.MIMEJSON:
					err = c.ShouldBindBodyWith(p.Interface(), binding.JSON)
				case gin.MIMEXML, gin.MIMEXML2:
					err = c.ShouldBindBodyWith(p.Interface(), binding.XML)
				default:
					err = c.ShouldBind(p.Interface())
				}

				if err != nil {
					c.Abort()
					c.String(500, err.Error())
					return
				}

				in = append(in, p.Elem())
			}

			handleResult(c, fv.Call(in)...)
		})
	}

	return handlers
}

func handleResult(c *gin.Context, out ...reflect.Value) {
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
