package sgin

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"reflect"
)

type Router interface {
	Use(args ...AnyHandler) Router
	Get(path string, handlers ...AnyHandler) Router
	Post(path string, handlers ...AnyHandler) Router
	Group(path string, handlers ...AnyHandler) Router
	Add(method, path string, handlers ...AnyHandler) Router
	Static(path, root string) Router
}

type RouterGroup struct {
	r     *gin.RouterGroup
	app   *App
	root  bool
	error func(*gin.Context, error)
}

func (grp *RouterGroup) Use(args ...AnyHandler) Router {
	grp.iRouter().Use(grp.handle(args...)...)
	return grp.router()
}

func (grp *RouterGroup) Get(path string, handlers ...AnyHandler) Router {
	return grp.Add(http.MethodGet, path, handlers...)
}

func (grp *RouterGroup) Post(path string, handlers ...AnyHandler) Router {
	return grp.Add(http.MethodPost, path, handlers...)
}

func (grp *RouterGroup) Group(path string, handlers ...AnyHandler) Router {
	r := grp.r.Group(path, grp.handle(handlers...)...)
	return &RouterGroup{r: r, app: grp.app}
}

func (grp *RouterGroup) Add(method string, path string, handlers ...AnyHandler) Router {
	grp.iRouter().Handle(method, path, grp.handle(handlers...)...)
	return grp.router()
}

func (grp *RouterGroup) Error(f func(*gin.Context, error)) {
	grp.error = f
}

func (grp *RouterGroup) handle(a ...AnyHandler) (handlers []gin.HandlerFunc) {
	for _, f := range a {
		if v, ok := f.(gin.HandlerFunc); ok {
			handlers = append(handlers, v)
			continue
		}

		var h *Handler
		if h, _ = f.(*Handler); h == nil {
			h = &Handler{Fn: f}
		}

		fnV := reflect.ValueOf(h.Fn)
		fnT := fnV.Type()

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

			if fnT.NumIn() == 2 { // 第二个参数被视为要绑定的输入参数
				val, err := bindIn(c, h.Bindings, fnT.In(1)) // 绑定输入参数
				if err != nil {                              // 处理错误
					if c.Abort(); h.Error != nil {
						h.Error(c, err)
					} else if grp.error != nil {
						grp.error(c, err)
					} else {
						c.String(http.StatusInternalServerError, err.Error())
					}
					return
				}
				in = append(in, val)
			}

			retResponse(c, fnV.Call(in)...)
		})
	}

	return handlers
}

func (grp *RouterGroup) Static(path, root string) Router {
	grp.iRouter().Static(path, root)
	return grp.router()
}

func (grp *RouterGroup) iRouter() gin.IRouter {
	if grp.root {
		return grp.app.e
	}
	return grp.r
}

func (grp *RouterGroup) router() Router {
	if grp.root {
		return grp.app
	}
	return grp
}
