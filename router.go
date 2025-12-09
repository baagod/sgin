package sgin

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
)

type IRouter interface {
    Use(args ...Handler) IRouter
    GET(path string, handlers ...Handler) IRouter
    POST(path string, handlers ...Handler) IRouter
    PUT(path string, handlers ...Handler) IRouter
    DELETE(path string, handlers ...Handler) IRouter
    Group(path string, handlers ...Handler) IRouter
    Handle(method, path string, handlers ...Handler) IRouter
    Static(path, root string) IRouter
}

type Router struct {
    group  *gin.RouterGroup
    engine *Engine
    root   bool
}

func (r *Router) Use(args ...Handler) IRouter {
    handlers := handler(r, args...)
    if r.root {
        r.engine.engine.Use(handlers...)
    } else {
        r.group.Use(handlers...)
    }
    return r.router()
}

func (r *Router) GET(path string, handlers ...Handler) IRouter {
    return r.Handle(http.MethodGet, path, handlers...)
}

func (r *Router) POST(path string, handlers ...Handler) IRouter {
    return r.Handle(http.MethodPost, path, handlers...)
}

func (r *Router) PUT(path string, handlers ...Handler) IRouter {
    return r.Handle(http.MethodPut, path, handlers...)
}

func (r *Router) DELETE(path string, handlers ...Handler) IRouter {
    return r.Handle(http.MethodDelete, path, handlers...)
}

func (r *Router) Group(path string, handlers ...Handler) IRouter {
    grp := r.group.Group(path, handler(r, handlers...)...)
    return &Router{group: grp, engine: r.engine}
}

func (r *Router) Handle(method string, path string, handlers ...Handler) IRouter {
    if r.engine.config.OpenAPI {
        fullPath := r.group.BasePath() + path
        // 移除可能重复的斜杠
        if strings.Contains(fullPath, "//") {
            fullPath = strings.ReplaceAll(fullPath, "//", "/")
        }
        for _, h := range handlers {
            AnalyzeAndRegister(fullPath, method, h)
        }
    }
    r.group.Handle(method, path, handler(r, handlers...)...)
    return r.router()
}

func (r *Router) Static(path, root string) IRouter {
    r.group.Static(path, root)
    return r.router()
}

func (r *Router) router() IRouter {
    if r.root {
        return r.engine
    }
    return r
}
