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

type RouterGroup struct {
	grp  *gin.RouterGroup
	app  *App
	root bool
}

func (r *RouterGroup) Use(args ...AnyHandler) Router {
	r.iRouter().Use(handle(r, args...)...)
	return r.router()
}

func (r *RouterGroup) GET(path string, handlers ...AnyHandler) Router {
	return r.Handle(http.MethodGet, path, handlers...)
}

func (r *RouterGroup) POST(path string, handlers ...AnyHandler) Router {
	return r.Handle(http.MethodPost, path, handlers...)
}

func (r *RouterGroup) Group(path string, handlers ...AnyHandler) Router {
	grp := r.grp.Group(path, handle(r, handlers...)...)
	return &RouterGroup{grp: grp, app: r.app}
}

func (r *RouterGroup) Handle(method string, path string, handlers ...AnyHandler) Router {
	r.iRouter().Handle(method, path, handle(r, handlers...)...)
	return r.router()
}

func (r *RouterGroup) Static(path, root string) Router {
	r.iRouter().Static(path, root)
	return r.router()
}

func (r *RouterGroup) iRouter() gin.IRouter {
	if r.root {
		return r.app.engine
	}
	return r.grp
}

func (r *RouterGroup) router() Router {
	if r.root {
		return r.app
	}
	return r
}
