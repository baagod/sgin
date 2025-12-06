package sgin

import (
    "reflect"
)

// result 是内部使用的统一响应结构体
// 它负责将各种 Handler 的返回值归一化，以便统一处理
type result struct {
    Status int   // HTTP 状态码，0 表示未设置
    Data   any   // 响应数据
    Err    error // 错误
}

// convertToResult 将反射调用的返回值切片转换为统一的 result 结构
func convertToResult(values []reflect.Value) *result {
    res := &result{}

    if len(values) == 0 {
        return res
    }

    // 获取第一个返回值
    v1 := values[0].Interface()

    // 单返回值情况
    if len(values) == 1 {
        switch v := v1.(type) {
        case error:
            res.Err = v // 返回的是 error
        default:
            // 如果Handler只返回一个int，我们默认它就是数据而不是状态码。
            // 状态码应该通过 (int, T) 或者 c.Status() 来设置。
            res.Data = v // 返回的是 T (Data)
        }
        return res
    }

    // 双返回值情况
    v2 := values[1].Interface()

    // 情况 A: (int, T) -> Status, Data
    // 注意：这里假设第一个 int 是状态码
    if code, ok := v1.(int); ok {
        res.Status = code
        res.Data = v2
        return res
    }

    // 情况 B: (T, error) -> Data, Err
    // 这是最标准的 Go 风格
    if res.Data = v1; v2 != nil {
        res.Err = v2.(error)
    }

    return res
}
