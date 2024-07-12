package sgin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Router interface {
	Use(args ...Handler) Router
	GET(path string, handlers ...Handler) Router
	POST(path string, handlers ...Handler) Router
	Group(path string, handlers ...Handler) Router
	Handle(method, path string, handlers ...Handler) Router
	Static(path, root string) Router
}

type Routers struct {
	grp    *gin.RouterGroup
	engine *Engine
	root   bool
}

func (r *Routers) Use(args ...Handler) Router {
	r.iRouter().Use(handle(r, args...)...)
	return r.router()
}

func (r *Routers) GET(path string, handlers ...Handler) Router {
	return r.Handle(http.MethodGet, path, handlers...)
}

func (r *Routers) POST(path string, handlers ...Handler) Router {
	return r.Handle(http.MethodPost, path, handlers...)
}

func (r *Routers) Group(path string, handlers ...Handler) Router {
	grp := r.grp.Group(path, handle(r, handlers...)...)
	return &Routers{grp: grp, engine: r.engine}
}

func (r *Routers) Handle(method string, path string, handlers ...Handler) Router {
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
