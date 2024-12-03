package sgin

import (
	"fmt"

	"github.com/spf13/cast"
)

type Response struct {
	Event   string `json:"event"`
	Status  int    `json:"status"`
	Code    int    `json:"code"`
	Count   int    `json:"count"`
	Message string `json:"msg"`
	Data    any    `json:"data"`
}

// SetStatus 设置 status, code
func (r *Response) SetStatus(status any, code ...any) *Response {
	newcode := 0
	st := cast.ToInt(status)

	if code != nil {
		newcode = cast.ToInt(code[0])
		if r != nil {
			r.Code = newcode
		}
	}

	if r == nil {
		return &Response{Status: st, Code: newcode}
	}

	r.Status = st
	return r
}

// SetCode 设置 code
func (r *Response) SetCode(code any) *Response {
	if r == nil {
		return &Response{Code: cast.ToInt(code)}
	}
	r.Code = cast.ToInt(code)
	return r
}

// SetEvent 设置事件
func (r *Response) SetEvent(event string) *Response {
	if r == nil {
		return &Response{Message: event}
	}
	r.Event = event
	return r
}

// SetMessage 设置消息
func (r *Response) SetMessage(msg any) *Response {
	if r == nil {
		return &Response{Message: fmt.Sprint(msg)}
	}
	r.Message = fmt.Sprint(msg)
	return r
}

// SetData 设置数据
func (r *Response) SetData(data any) *Response {
	if r == nil {
		return &Response{Data: data}
	}
	r.Data = data
	r.Status = 1
	return r
}

// SetFailData 设置失败数据
func (r *Response) SetFailData(data any) *Response {
	if r == nil {
		return &Response{Data: data}
	}
	r.Data = data
	r.Status = 0
	return r
}
