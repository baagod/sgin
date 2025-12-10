package sgin

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
)

type IRouter interface {
    Use(...Handler) IRouter
    GET(string, ...Handler) IRouter
    POST(string, ...Handler) IRouter
    PUT(string, ...Handler) IRouter
    DELETE(string, ...Handler) IRouter
    Group(string, ...Handler) IRouter
    Handle(method, path string, handlers ...Handler) IRouter
    Static(path, root string) IRouter
    Security(schemes ...string) IRouter
}

type Router struct {
    i        gin.IRouter
    e        *Engine
    base     string                  // 基础路径
    security []OASecurityRequirement // 路由组安全配置
}

func (r *Router) Use(args ...Handler) IRouter {
    r.i.Use(handler(r.e, args...)...)
    return r
}

func (r *Router) Security(schemes ...string) IRouter {
    for _, scheme := range schemes {
        r.security = append(r.security, OASecurityRequirement{
            scheme: {},
        })
    }
    return r
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
    grp := r.i.Group(path, handler(r.e, handlers...)...)
    return &Router{i: grp, e: r.e, base: r.fullPath(path), security: r.security}
}

func (r *Router) Handle(method, path string, handlers ...Handler) IRouter {
    if r.e.cfg.OpenAPI {
        fullPath := r.fullPath(path)
        for _, h := range handlers {
            AnalyzeAndRegister(fullPath, method, h, r.security)
        }
    }
    r.i.Handle(method, path, handler(r.e, handlers...)...)
    return r
}

func (r *Router) Static(path, root string) IRouter {
    r.i.Static(path, root)
    return r
}

func (r *Router) fullPath(path string) string {
    return strings.ReplaceAll(r.base+path, "//", "/")
}
