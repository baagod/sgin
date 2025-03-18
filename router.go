package sgin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Router interface {
	Use(args ...Handler) Router
	GET(path string, handlers ...Handler) Router
	POST(path string, handlers ...Handler) Router
	PUT(path string, handlers ...Handler) Router
	DELETE(path string, handlers ...Handler) Router
	Group(path string, handlers ...Handler) Router
	Handle(method, path string, handlers ...Handler) Router
	Static(path, root string) Router
}

type Route struct {
	group  *gin.RouterGroup
	engine *Engine
	root   bool
}

func (r *Route) Use(args ...Handler) Router {
	handlers := handler(r, args...)
	if r.root {
		r.engine.engine.Use(handlers...)
	} else {
		r.group.Use(handlers...)
	}
	return r.router()
}

func (r *Route) GET(path string, handlers ...Handler) Router {
	return r.Handle(http.MethodGet, path, handlers...)
}

func (r *Route) POST(path string, handlers ...Handler) Router {
	return r.Handle(http.MethodPost, path, handlers...)
}

func (r *Route) PUT(path string, handlers ...Handler) Router {
	return r.Handle(http.MethodPut, path, handlers...)
}

func (r *Route) DELETE(path string, handlers ...Handler) Router {
	return r.Handle(http.MethodDelete, path, handlers...)
}

func (r *Route) Group(path string, handlers ...Handler) Router {
	grp := r.group.Group(path, handler(r, handlers...)...)
	return &Route{group: grp, engine: r.engine}
}

func (r *Route) Handle(method string, path string, handlers ...Handler) Router {
	r.group.Handle(method, path, handler(r, handlers...)...)
	return r.router()
}

func (r *Route) Static(path, root string) Router {
	r.group.Static(path, root)
	return r.router()
}

func (r *Route) router() Router {
	if r.root {
		return r.engine
	}
	return r
}
