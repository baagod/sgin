package sgin

import (
    "errors"
    "reflect"
    "sync"
    "unsafe"

    "github.com/baagod/sgin/v2/helper"
    "github.com/gin-gonic/gin"
    "github.com/gin-gonic/gin/binding"
    "github.com/go-playground/validator/v10"
)

var (
    hMeta      HandleMeta
    structType = reflect.TypeFor[struct{}]()
)

type HandleArg struct {
    In, Out reflect.Type
}

type HandleMeta struct {
    m sync.Map
}

func handlerKey(h Handler) uintptr {
    return *(*uintptr)(unsafe.Pointer(&h))
}

func (m *HandleMeta) Get(h Handler) (*HandleArg, bool) {
    if v, ok := m.m.Load(handlerKey(h)); ok {
        return v.(*HandleArg), true
    }
    return nil, false
}

func (m *HandleMeta) Set(h Handler, meta *HandleArg) {
    m.m.Store(handlerKey(h), meta)
}

func (m *HandleMeta) Delete(h Handler) {
    m.m.Delete(handlerKey(h))
}

func (m *HandleMeta) Pop(h Handler) (a *HandleArg) {
    key := handlerKey(h)
    if v, exist := m.m.Load(key); exist {
        a, _ = v.(*HandleArg)
    }
    m.m.Delete(key)
    return
}

type Handler = gin.HandlerFunc

// H 创建一个带有 [输入] 和 [输出] 的强类型处理器 (支持 OpenAPI)
func H[I any, R any](f func(*Ctx, I) (R, error)) Handler {
    // 预先计算类型
    tIn := reflect.TypeFor[I]()
    ptrIn := tIn.Kind() == reflect.Ptr
    if ptrIn { // 如果传递指针，会变成 **I，需要再次解引用。
        tIn = tIn.Elem()
    }

    tOut := helper.Deref(reflect.TypeFor[R]())

    // 构造原生 Gin 闭包
    h := func(gc *gin.Context) {
        e := gc.MustGet(EngineKey).(*Engine)
        c, _ := gc.Keys[CtxKey].(*Ctx)

        if c == nil {
            c = newCtx(gc, e)
            gc.Set(CtxKey, c)
        }

        var in I               // 初始化输入参数。注意，如果 I 是指针结构体，这里是 nil。
        if tIn != structType { // 非空结构体时才绑定参数
            result, err := bindV3(c, tIn, ptrIn)
            if err != nil {
                gc.Abort()
                _ = e.cfg.ErrorHandler(c, ErrBadRequest(err.Error()))
                return
            }
            in = result.(I)
        }

        c.send(f(c, in))
    }

    hMeta.Set(h, &HandleArg{In: tIn, Out: tOut}) // 注册元数据
    return h
}

// Ho 创建一个仅有 [输出] 的强类型处理器 (支持 OpenAPI)
func Ho[I any, R any](f func(*Ctx, I) R) Handler {
    return H(func(c *Ctx, in I) (R, error) {
        return f(c, in), nil
    })
}

// He 创建一个无输入且仅返回 error 的处理器方法
func He(f func(*Ctx) error) Handler {
    return H(func(c *Ctx, _ struct{}) (any, error) {
        return nil, f(c)
    })
}

// Hn 创建一个无输入输出的处理器方法
func Hn(f func(*Ctx)) Handler {
    return H(func(c *Ctx, _ struct{}) (any, error) {
        f(c)
        return nil, nil
    })
}

func bindV3(c *Ctx, t reflect.Type, ptr bool) (_ any, err error) {
    gc := c.ctx
    v := reflect.New(t) // v = *in
    value := v.Interface()

    // 绑定 URI, Header, Query 和 Body 参数，忽略效验错误。
    for _, f := range []func(any) error{
        gc.ShouldBindUri,
        gc.ShouldBindHeader,
        gc.ShouldBindQuery,
        func(o any) error {
            b := binding.Default(gc.Request.Method, gc.ContentType())

            // 针对会消耗 Body 的格式使用 BindingBody 进行缓存式绑定
            if b == binding.JSON || b == binding.XML || b == binding.YAML {
                if bb, ok := b.(binding.BindingBody); ok {
                    return gc.ShouldBindBodyWith(o, bb)
                }
            }

            // 其他情况（如 Form, Query 或断言失败）使用标准绑定
            return gc.ShouldBindWith(o, b)
        },
    } {
        if err = tryBind(f, value); err != nil {
            return
        }
    }

    // 所有数据来源都尝试绑定后，手动触发一次完整校验。
    // 这是为了捕获之前被 tryBind 忽略的校验错误（如果最终还是缺字段）。
    if err = binding.Validator.ValidateStruct(value); err != nil {
        var errs validator.ValidationErrors
        if !errors.As(err, &errs) || len(errs) == 0 {
            return
        }

        if tr := c.engine.translator; tr != nil {
            locale := c.locale().String() // 获取当前请求的语言
            if trans, found := tr.GetTranslator(locale); found {
                err = errors.New(errs[0].Translate(trans)) // 翻译首个校验错误
            }
        }

        return
    }

    if ptr { // 用户要 *t
        return v.Interface(), nil
    }

    return v.Elem().Interface(), nil // 返回 t
}

// tryBind 执行绑定操作。
// 如果是校验错误（validator.ValidationErrors），则忽略并返回 nil，允许从其他来源继续绑定。
// 如果是其他错误（如解析错误），则直接返回该错误。
func tryBind(binder func(any) error, ptr any) (err error) {
    if err = binder(ptr); err != nil {
        var errs validator.ValidationErrors
        if errors.As(err, &errs) {
            return nil
        }
    }
    return
}
