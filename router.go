package sgin

import (
    "net/http"
    "reflect"
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
}

type Router struct {
    i    gin.IRouter
    e    *Engine
    base string
    op   OAOperation
}

func (r *Router) Use(args ...Handler) IRouter {
    r.i.Use(handler(r.e, args...)...)
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
    realHandlers, opOption := separateHandlers(handlers)
    grp := r.i.Group(path, handler(r.e, realHandlers...)...)

    op := OAOperation{Responses: map[string]OAResponse{}}
    if opOption != nil {
        opOption(&op)
    }

    return &Router{i: grp, e: r.e, base: r.fullPath(path), op: op}
}

func (r *Router) Handle(method, path string, handlers ...Handler) IRouter {
    realHandlers, operation := separateHandlers(handlers)

    if r.e.cfg.OpenAPI {
        // 基于当前 Router 的 op 原型克隆一个新的 OAOperation，用于当前路由
        opForThisRoute := r.op.Clone()

        // 应用路由级别传入的 AddOperation 到这个克隆的 op 上
        if operation != nil {
            operation(r.op.Clone())
        }

        // 将克隆并应用了选项的 op 传递给 AnalyzeAndRegister
        // AnalyzeAndRegister 只需要知道最终的 OAOperation
        if len(realHandlers) > 0 {
            fullPath := r.fullPath(path)
            AnalyzeAndRegister(fullPath, method, realHandlers[len(realHandlers)-1], opForThisRoute)
        }
    }

    r.i.Handle(method, path, handler(r.e, realHandlers...)...)
    return r
}

func (r *Router) Static(path, root string) IRouter {
    r.i.Static(path, root)
    return r
}

func (r *Router) fullPath(path string) string {
    return strings.ReplaceAll(r.base+path, "//", "/")
}

func separateHandlers(handlers []Handler) ([]Handler, AddOperation) {
    if len(handlers) == 0 {
        return handlers, nil
    }

    h := handlers[0]
    opType := reflect.TypeOf((AddOperation)(nil))

    if h != nil && reflect.TypeOf(h).ConvertibleTo(opType) {
        opFunc := reflect.ValueOf(h).Convert(opType).Interface().(AddOperation)
        return handlers[1:], opFunc
    }

    return handlers, nil
}
