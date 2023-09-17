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
	r *gin.RouterGroup
}

func (grp *RouterGroup) Use(args ...Handler) Router {
	grp.r.Use(HandlerFunc(args)...)
	return grp
}

func (grp *RouterGroup) Get(path string, handlers ...Handler) Router {
	return grp.Add(http.MethodGet, path, handlers...)
}

func (grp *RouterGroup) Post(path string, handlers ...Handler) Router {
	return grp.Add(http.MethodPost, path, handlers...)
}

func (grp *RouterGroup) Group(path string, handlers ...Handler) Router {
	r := grp.r.Group(path, HandlerFunc(handlers...)...)
	return &RouterGroup{r}
}

func (grp *RouterGroup) Add(method string, path string, handlers ...Handler) Router {
	grp.r.Handle(method, path, HandlerFunc(handlers...)...)
	return grp
}

func (grp *RouterGroup) Static(path, root string) Router {
	grp.r.Static(path, root)
	return grp
}
