package sgin

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type IRouter interface {
	Use(...Handler) IRouter
	GET(string, Handler, ...AddOperation) IRouter
	POST(string, Handler, ...AddOperation) IRouter
	PUT(string, Handler, ...AddOperation) IRouter
	DELETE(string, Handler, ...AddOperation) IRouter
	HEAD(string, Handler, ...AddOperation) IRouter
	PATCH(string, Handler, ...AddOperation) IRouter
	OPTIONS(string, Handler, ...AddOperation) IRouter
	Handle(string, string, Handler, ...AddOperation) IRouter
	Any(string, Handler, ...AddOperation) IRouter
	Match([]string, string, Handler, ...AddOperation) IRouter
	Group(string, ...AddOperation) IRouter
	Static(string, string) IRouter
	StaticFile(string, string) IRouter
	StaticFileFS(string, string, http.FileSystem) IRouter
	StaticFS(string, http.FileSystem) IRouter
}

var anyMethods = []string{
	http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch,
	http.MethodHead, http.MethodOptions, http.MethodDelete, http.MethodConnect,
	http.MethodTrace,
}

type Router struct {
	i    gin.IRouter
	e    *Engine
	base string // 基础路径
	api  *API
	op   Operation // 该路由的基础 Operation
}

func (r *Router) Use(handlers ...Handler) IRouter {
	r.i.Use(handlers...)
	return r
}

func (r *Router) GET(path string, h Handler, ops ...AddOperation) IRouter {
	return r.Handle(http.MethodGet, path, h, ops...)
}

func (r *Router) POST(path string, h Handler, ops ...AddOperation) IRouter {
	return r.Handle(http.MethodPost, path, h, ops...)
}

func (r *Router) PUT(path string, h Handler, ops ...AddOperation) IRouter {
	return r.Handle(http.MethodPut, path, h, ops...)
}

func (r *Router) DELETE(path string, h Handler, ops ...AddOperation) IRouter {
	return r.Handle(http.MethodDelete, path, h, ops...)
}

func (r *Router) HEAD(path string, h Handler, ops ...AddOperation) IRouter {
	return r.Handle(http.MethodHead, path, h, ops...)
}

func (r *Router) PATCH(path string, h Handler, ops ...AddOperation) IRouter {
	return r.Handle(http.MethodPatch, path, h, ops...)
}

func (r *Router) OPTIONS(path string, h Handler, ops ...AddOperation) IRouter {
	return r.Handle(http.MethodOptions, path, h, ops...)
}

func (r *Router) Any(path string, h Handler, ops ...AddOperation) IRouter {
	return r.Match(anyMethods, path, h, ops...)
}

func (r *Router) Handle(method, path string, h Handler, ops ...AddOperation) IRouter {
	if r.e.cfg.OpenAPI != nil {
		if meta, ok := hMeta.Pop(h); ok {
			op := r.op.Clone()
			for _, f := range ops {
				f(op)
			}
			r.api.Register(op, r.fullPath(path), method, meta)
		}
	}
	r.i.Handle(method, path, h)
	return r
}

func (r *Router) Match(methods []string, path string, h Handler, ops ...AddOperation) IRouter {
	if r.e.cfg.OpenAPI != nil {
		if meta, ok := hMeta.Get(h); ok {
			for _, method := range methods {
				op := r.op.Clone()
				for _, f := range ops {
					f(op)
				}
				r.api.Register(op, r.fullPath(path), method, meta)
			}
			hMeta.Delete(h)
		}
	}
	r.i.Match(methods, path, h)
	return r
}

func (r *Router) Group(path string, ops ...AddOperation) IRouter {
	op := r.op.Clone()
	for _, f := range ops {
		f(op)
	}
	return &Router{
		i:    r.i.Group(path),
		e:    r.e,
		base: r.fullPath(path),
		api:  r.api,
		op:   *op,
	}
}

func (r *Router) Static(path, root string) IRouter {
	r.i.Static(path, root)
	return r
}

func (r *Router) StaticFile(name, root string) IRouter {
	r.i.StaticFile(name, root)
	return r
}

func (r *Router) StaticFileFS(name, root string, fs http.FileSystem) IRouter {
	r.i.StaticFileFS(name, root, fs)
	return r
}

func (r *Router) StaticFS(name string, root http.FileSystem) IRouter {
	r.i.StaticFS(name, root)
	return r
}

func (r *Router) fullPath(path string) string {
	return strings.ReplaceAll(r.base+path, "//", "/")
}
