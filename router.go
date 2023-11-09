package sgin

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type Router interface {
	Use(args ...AnyHandler) Router
	GET(path string, handlers ...AnyHandler) Router
	POST(path string, handlers ...AnyHandler) Router
	Group(path string, handlers ...AnyHandler) Router
	Handle(method, path string, handlers ...AnyHandler) Router
	Static(path, root string) Router
}

type Routers struct {
	grp    *gin.RouterGroup
	engine *Engine
	root   bool
}

func (r *Routers) Use(args ...AnyHandler) Router {
	r.iRouter().Use(handle(r, args...)...)
	return r.router()
}

func (r *Routers) GET(path string, handlers ...AnyHandler) Router {
	return r.Handle(http.MethodGet, path, handlers...)
}

func (r *Routers) POST(path string, handlers ...AnyHandler) Router {
	return r.Handle(http.MethodPost, path, handlers...)
}

func (r *Routers) Group(path string, handlers ...AnyHandler) Router {
	grp := r.grp.Group(path, handle(r, handlers...)...)
	return &Routers{grp: grp, engine: r.engine}
}

func (r *Routers) Handle(method string, path string, handlers ...AnyHandler) Router {
	r.iRouter().Handle(method, path, handle(r, handlers...)...)
	return r.router()
}

func (r *Routers) Static(path, root string) Router {
	r.iRouter().Static(path, root)
	return r.router()
}

func (r *Routers) iRouter() gin.IRouter {
	if r.root {
		return r.engine.engine
	}
	return r.grp
}

func (r *Routers) router() Router {
	if r.root {
		return r.engine
	}
	return r
}
