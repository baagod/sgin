package sgin

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler any // Gin原生 | sgin V2 智能 Handler

// handler 是核心适配器，负责将用户传入的任意 Handler 转换为 Gin 的 HandlerFunc
func handler(r *Route, a ...Handler) (handlers []gin.HandlerFunc) {
	for _, f := range a {
		// 1. L0: 优先识别 Gin 原生 Handler
		switch ginHandler := f.(type) {
		case gin.HandlerFunc:
			handlers = append(handlers, ginHandler)
			continue
		case func(*gin.Context):
			handlers = append(handlers, ginHandler)
			continue
		}

		// 2. L2: 智能反射适配器
		hValue := reflect.ValueOf(f)
		hType := hValue.Type()

		// --- 启动时自检 (Fail Fast) ---
		if hType.Kind() != reflect.Func {
			panic(fmt.Sprintf("Handler must be a function, got %T", f))
		}

		numIn := hType.NumIn()
		if numIn < 1 || numIn > 2 {
			panic(fmt.Sprintf("Handler accepts 1 or 2 arguments, got %d. Function: %T", numIn, f))
		}

		// 检查第一个参数必须是 *Ctx
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
		handlers = append(handlers, func(ginCtx *gin.Context) {
			// 获取或创建 sgin.Ctx
			ctx, _ := ginCtx.Keys[CtxKey].(*Ctx)
			if ctx == nil {
				ctx = newCtx(ginCtx, r.engine)
				ginCtx.Set(CtxKey, ctx)
			}

			// 准备参数列表
			args := make([]reflect.Value, numIn)
			args[0] = reflect.ValueOf(ctx)

			// 如果有请求参数，执行智能绑定
			if numIn == 2 {
				val, err := bindV2(ginCtx, reqType)
				if err != nil {
					// 绑定失败，统一处理错误
					_ = r.engine.config.ErrorHandler(ctx, &Error{Message: err.Error(), Code: 400})
					ginCtx.Abort()
					return
				}
				args[1] = val
			}

			// 反射调用业务逻辑
			results := hValue.Call(args)

			// 结果归一化并发送
			res := convertToResult(results)
			ctx.sendResult(res)
		})
	}

	return
}

// bindV2 实现了 V2 架构的智能复合绑定
func bindV2(c *gin.Context, T reflect.Type) (_ reflect.Value, err error) {
	// 确保我们操作的是具体类型（非指针）来创建实例，但绑定时可能需要指针
	isPtr := T.Kind() == reflect.Ptr
	baseType := T
	if isPtr {
		baseType = T.Elem()
	}

	// 创建一个新的结构体实例
	// valPtr 是指向该结构体的指针 (例如 *UserReq)
	valPtr := reflect.New(baseType)
	ptrInterface := valPtr.Interface()

	// --- 1. URI 绑定 (Path) ---
	// 只要有 uri tag，Gin 就会尝试绑定
	if err = c.ShouldBindUri(ptrInterface); err != nil {
		return
	}

	// --- 2. Header 绑定 ---
	if err = c.ShouldBindHeader(ptrInterface); err != nil {
		return
	}

	// --- 3. Query 绑定 (URL Params) ---
	// 显式绑定 Query，即使是 POST 请求也可以有 Query 参数
	if err = c.ShouldBindQuery(ptrInterface); err != nil {
		return
	}

	// --- 4. Body 绑定 (互斥) ---
	// GET 请求通常没有 Body，跳过
	if ct := c.ContentType(); c.Request.Method != "GET" {
		// 根据 Content-Type 智能选择
		if ct == gin.MIMEJSON {
			err = c.ShouldBindJSON(ptrInterface)
		} else if ct == gin.MIMEXML {
			err = c.ShouldBindXML(ptrInterface)
		} else if ct == gin.MIMEPOSTForm || strings.HasPrefix(ct, gin.MIMEMultipartPOSTForm) {
			// 对于 Form，Gin 的 ShouldBind 已经涵盖了 Query，但我们为了保险（和拿到 Query 里的 form tag）
			// 可能会重复绑定，但这是安全的。
			// 不过，ShouldBind 本身就会做 Query + Form 的混合绑定。
			// 这里我们显式调用 ShouldBind 用于处理 PostForm
			err = c.ShouldBind(ptrInterface)
		}
	}

	if err != nil {
		// 处理校验错误
		var vErrs validator.ValidationErrors
		if errors.As(err, &vErrs) {
			// 尝试获取 failtip 自定义错误消息
			for _, e := range vErrs {
				if field, ok := baseType.FieldByName(e.Field()); ok {
					if failtip := field.Tag.Get("failtip"); failtip != "" {
						return reflect.Value{}, errors.New(failtip)
					}
				}
			}
		}
		return
	}

	// 返回结果
	if isPtr {
		return valPtr, nil // 用户要 *T，返回 *T
	}

	return valPtr.Elem(), nil // 用户要 T，返回 T
}