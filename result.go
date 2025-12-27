package sgin

import (
	"fmt"

	"github.com/spf13/cast"
)

type Result struct {
	Event  string `json:"event"`  // 事件标识
	Status int    `json:"status"` // 自定义状态码，经常用于定义请求成功或失败等错误状态 (非 HTTP 状态码)
	Code   int    `json:"code"`   // 自定义代码，经常与 Status 关联。例如: Status=0 时，Code=N。
	Count  int    `json:"count"`  // 如果 Data 返回列表，可以在这里设置列表长度。
	Msg    string `json:"msg"`    // 结果消息
	Data   any    `json:"data"`   // 结果数据
}

func (r *Result) newStatus(status any, code ...any) *Result {
	st := cast.ToInt(status)
	if len(code) == 0 {
		return &Result{Status: st}
	}
	return &Result{Status: st, Code: cast.ToInt(code[0])}
}

func (r *Result) newOK(data ...any) *Result {
	if len(data) == 0 {
		return &Result{Status: 1}
	}
	return &Result{Data: data[0], Status: 1}
}

func (r *Result) NewFailed(data ...any) *Result {
	if len(data) == 0 {
		return &Result{}
	}
	return &Result{Data: data[0]}
}

// SetStatus 设置 status, code
func (r *Result) SetStatus(status any, code ...any) (res *Result) {
	if res = r; res == nil {
		res = r.newStatus(status, code...)
	} else if r.Status = cast.ToInt(status); len(code) > 0 {
		r.Code = cast.ToInt(code[0])
	}
	return res
}

// SetCode 设置 code
func (r *Result) SetCode(code any) *Result {
	if r == nil {
		return &Result{Code: cast.ToInt(code)}
	}
	r.Code = cast.ToInt(code)
	return r
}

// SetEvent 设置事件
func (r *Result) SetEvent(e string) *Result {
	if r == nil {
		return &Result{Event: e}
	}
	r.Event = e
	return r
}

// SetMsg 设置消息
func (r *Result) SetMsg(format any, a ...any) *Result {
	m := fmt.Sprintf(fmt.Sprint(format), a...)
	if r == nil {
		return &Result{Msg: m}
	}
	r.Msg = m
	return r
}

// OK 设置成功 (status=1) 数据
func (r *Result) OK(data ...any) (res *Result) {
	if res = r; res == nil {
		res = r.newOK(data...)
	} else if r.Status = 1; len(data) > 0 {
		r.Data = data[0]
	}
	return res
}

// Failed 设置失败 (status=0) 数据
func (r *Result) Failed(data ...any) (res *Result) {
	if res = r; res == nil {
		res = r.NewFailed(data...)
	} else if r.Status = 0; len(data) > 0 {
		r.Data = data[0]
	}
	return res
}
