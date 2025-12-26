package sgin

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type IRouter interface {
	Use(...Handler) IRouter
	GET(string, Handler) IRouter
	POST(string, Handler) IRouter
	PUT(string, Handler) IRouter
	DELETE(string, Handler) IRouter
	Group(string, Handler) IRouter
	Handle(method, path string, h Handler) IRouter
	Static(path, root string) IRouter
	Operation(func(*Operation)) IRouter
}

type Router struct {
	i    gin.IRouter
	e    *Engine
	base string // 基础路径
	api  *API

	op     Operation  // 当前路由 Operation
	lastOp *Operation // 每个处理方法都 Operation
}

func (r *Router) Use(handlers ...Handler) IRouter {
	r.i.Use(handlers...)
	return r
}

func (r *Router) Operation(f func(*Operation)) IRouter {
	if f != nil && r.e.cfg.OpenAPI != nil {
		if r.lastOp != nil {
			f(r.lastOp)
		} else {
			f(&r.op)
		}
	}
	return r
}

func (r *Router) GET(path string, h Handler) IRouter {
	return r.Handle(http.MethodGet, path, h)
}

func (r *Router) POST(path string, h Handler) IRouter {
	return r.Handle(http.MethodPost, path, h)
}

func (r *Router) PUT(path string, h Handler) IRouter {
	return r.Handle(http.MethodPut, path, h)
}

func (r *Router) DELETE(path string, h Handler) IRouter {
	return r.Handle(http.MethodDelete, path, h)
}

func (r *Router) Group(path string, h Handler) IRouter {
	return &Router{
		i:    r.i.Group(path, h),
		e:    r.e,
		base: r.fullPath(path),
		api:  r.api,
		op:   Operation{Responses: map[string]*ResponseBody{}},
	}
}

func (r *Router) Handle(method, path string, h Handler) IRouter {
	if r.e.cfg.OpenAPI != nil {
		if meta, ok := hMeta.Get(h); ok {
			r.lastOp = r.op.Clone()
			r.api.Register(r.lastOp, r.fullPath(path), method, meta)
			hMeta.Delete(h)
		}
	}
	r.i.Handle(method, path, h)
	return r
}

func (r *Router) Static(path, root string) IRouter {
	r.i.Static(path, root)
	return r
}

func (r *Router) fullPath(path string) string {
	return strings.ReplaceAll(r.base+path, "//", "/")
}
