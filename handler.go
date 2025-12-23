package sgin

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

type Handler any // gin.HandlerFunc, func(*sgin.Ctx[, Input]) T | (T, error)

// handler 是核心适配器，负责将用户传入的任意 Handler 转换为 Gin 的 HandlerFunc
func handler(engine *Engine, a ...Handler) (handlers []gin.HandlerFunc) {
	for _, f := range a {
		// 1. 优先识别 Gin 原生 Handler
		switch ginHandler := f.(type) {
		case gin.HandlerFunc:
			handlers = append(handlers, ginHandler)
			continue
		case func(*gin.Context):
			handlers = append(handlers, ginHandler)
			continue
		}

		// 2. 智能反射适配器
		h := reflect.ValueOf(f)
		hType := h.Type()

		// --- 启动时自检 (Fail Fast) ---
		if hType.Kind() != reflect.Func {
			panic(fmt.Sprintf("Handler must be a function, got %T", f))
		}

		numIn := hType.NumIn()
		if numIn < 1 || numIn > 2 {
			panic(fmt.Sprintf("Handler accepts 1 or 2 arguments, got %d. Function: %T", numIn, f))
		}

		// 检查第一个参数必须是 *sgin.Ctx
		if hType.In(0) != reflect.TypeOf(&Ctx{}) {
			panic(fmt.Sprintf("Handler's first argument must be *sgin.Ctx. Function: %T", f))
		}

		// 预先计算是否有第二个参数（请求结构体）
		var reqType reflect.Type
		if numIn == 2 {
			// 允许是指针或结构体，bindV2 会处理
			reqType = hType.In(1)
		}

		// --- 生成运行时闭包 ---
		handlers = append(handlers, func(gc *gin.Context) {
			// 获取或创建 sgin.Ctx
			ctx, _ := gc.Keys[CtxKey].(*Ctx)
			if ctx == nil {
				ctx = newCtx(gc, engine)
				gc.Set(CtxKey, ctx)
			}

			// 准备参数列表
			args := make([]reflect.Value, numIn)
			args[0] = reflect.ValueOf(ctx)

			// 如果有请求参数，执行智能绑定
			if numIn == 2 {
				val, err := bindV2(ctx, reqType)
				if err != nil { // 绑定失败，统一处理错误
					gc.Abort()
					_ = engine.cfg.ErrorHandler(ctx, ErrBadRequest(err.Error()))
					return
				}
				args[1] = val
			}

			// 反射调用业务逻辑
			results := h.Call(args)

			// 结果归一化并发送
			res := convertToResult(results)
			ctx.sendResult(res)
		})
	}

	return
}

// bindV2 实现了 V2 架构的智能复合绑定
func bindV2(c *Ctx, t reflect.Type) (_ reflect.Value, err error) {
	gc := c.ctx

	// 确保我们操作的是具体类型（非指针）来创建实例，但绑定时可能需要指针
	isPtr := t.Kind() == reflect.Ptr
	if isPtr {
		t = t.Elem()
	}

	// 创建一个新的结构体实例
	valPtr := reflect.New(t) // valPtr 是指向该结构体的指针 (例如 *UserReq)
	ptr := valPtr.Interface()

	// 绑定 URI, Header, Query 和 Body 参数，忽略效验错误。
	if err = tryBind(gc.ShouldBindUri, ptr); err == nil {
		if err = tryBind(gc.ShouldBindHeader, ptr); err == nil {
			err = tryBind(gc.ShouldBind, ptr)
		}
	}

	if err != nil {
		return
	}

	// 所有数据来源都尝试绑定后，手动触发一次完整校验。
	// 这是为了捕获之前被 tryBind 忽略的校验错误（如果最终还是缺字段）。
	if err = binding.Validator.ValidateStruct(ptr); err != nil {
		var errs validator.ValidationErrors
		if errors.As(err, &errs) && len(errs) > 0 {
			if tr := c.engine.translator; tr != nil {
				locale := c.locale().String() // 获取当前请求的语言
				if trans, found := tr.GetTranslator(locale); found {
					err = errors.New(errs[0].Translate(trans)) // 翻译首个校验错误
				}
			}
		}
		return
	}

	if isPtr { // 用户要 *t
		return valPtr, nil // 返回 *t
	}

	return valPtr.Elem(), nil // 否则返回 t
}

// tryBind 执行绑定操作。
// 如果是校验错误（validator.ValidationErrors），则忽略并返回 nil，允许从其他来源继续绑定。
// 如果是其他错误（如解析错误），则直接返回该错误。
func tryBind(binder func(any) error, ptr any) (err error) {
	if err = binder(ptr); err != nil {
		var vErrors validator.ValidationErrors
		if errors.As(err, &vErrors) {
			return nil
		}
	}
	return
}
