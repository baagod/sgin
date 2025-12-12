package sgin

import (
    "net/http"
    "reflect"
    "strings"

    "github.com/baagod/sgin/oa"
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
    op   oa.Operation
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
    realHandlers, operation := separateHandlers(handlers)
    grp := r.i.Group(path, handler(r.e, realHandlers...)...)

    op := oa.Operation{
        Responses: map[string]oa.Response{},
        Security:  []oa.Requirement{{}},
    }

    if operation != nil {
        operation(&op)
    }

    return &Router{i: grp, e: r.e, base: r.fullPath(path), op: op}
}

func (r *Router) Handle(method, path string, handlers ...Handler) IRouter {
    realHandlers, operation := separateHandlers(handlers) // 在这里声明并赋值

    if r.e.cfg.OpenAPI != nil {
        fullPath := r.fullPath(path)
        cloneOp := r.op.Clone()

        if operation != nil {
            operation(cloneOp)
        }

        if len(realHandlers) > 0 {
            oa.Register(fullPath, method, realHandlers[len(realHandlers)-1], cloneOp)
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

func separateHandlers(handlers []Handler) ([]Handler, oa.AddOperation) {
    if len(handlers) == 0 {
        return handlers, nil
    }

    h := handlers[0]
    opType := reflect.TypeOf((oa.AddOperation)(nil))

    if h != nil && reflect.TypeOf(h).ConvertibleTo(opType) {
        opFunc := reflect.ValueOf(h).Convert(opType).Interface().(oa.AddOperation)
        return handlers[1:], opFunc
    }

    return handlers, nil
}
