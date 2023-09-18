package sgin

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type Handler any

type Router interface {
	Use(args ...Handler) Router
	Get(path string, handlers ...Handler) Router
	Post(path string, handlers ...Handler) Router
	Group(path string, handlers ...Handler) Router
	Add(method, path string, handlers ...Handler) Router
	Static(path, root string) Router
}

type RouterGroup struct {
	r    *gin.RouterGroup
	app  *App
	root bool
}

func (grp *RouterGroup) Use(args ...Handler) Router {
	grp.iRouter().Use(HandlerFunc(args...)...)
	return grp.router()
}

func (grp *RouterGroup) Get(path string, handlers ...Handler) Router {
	return grp.Add(http.MethodGet, path, handlers...)
}

func (grp *RouterGroup) Post(path string, handlers ...Handler) Router {
	return grp.Add(http.MethodPost, path, handlers...)
}

func (grp *RouterGroup) Group(path string, handlers ...Handler) Router {
	r := grp.r.Group(path, HandlerFunc(handlers...)...)
	return &RouterGroup{r: r, app: grp.app}
}

func (grp *RouterGroup) Add(method string, path string, handlers ...Handler) Router {
	grp.iRouter().Handle(method, path, HandlerFunc(handlers...)...)
	return grp.router()
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
