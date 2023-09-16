package sgin

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type Group struct {
	r *gin.RouterGroup
}

func (grp *Group) Use(args ...Handler) Router {
	grp.r.Use(HandlerFunc(args)...)
	return grp
}

func (grp *Group) Get(path string, handlers ...Handler) Router {
	return grp.Add(http.MethodGet, path, handlers...)
}

func (grp *Group) Post(path string, handlers ...Handler) Router {
	return grp.Add(http.MethodPost, path, handlers...)
}

func (grp *Group) Group(path string, handlers ...Handler) Router {
	r := grp.r.Group(path, HandlerFunc(handlers...)...)
	return &Group{r}
}

func (grp *Group) Add(method string, path string, handlers ...Handler) Router {
	grp.r.Handle(method, path, HandlerFunc(handlers...)...)
	return grp
}
