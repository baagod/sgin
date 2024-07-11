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
	Err     error  `json:"-"`
}

func (r *Response) WithStatus(status int) *Response {
	if r == nil {
		return &Response{Status: status}
	}
	r.Status = status
	return r
}

func (r *Response) WithCode(code any) *Response {
	if r == nil {
		return &Response{Code: cast.ToInt(code)}
	}
	r.Code = cast.ToInt(code)
	return r
}

func (r *Response) WithMsg(message any) *Response {
	if r == nil {
		return &Response{Message: fmt.Sprint(message)}
	}
	r.Message = fmt.Sprint(message)
	return r
}

func (r *Response) WithData(data any) *Response {
	if r == nil {
		return &Response{Data: data}
	}
	r.Data = data
	return r
}

func (r *Response) WithCount(count int) *Response {
	if r == nil {
		return &Response{Count: count}
	}
	r.Count = count
	return r
}

func (r *Response) OK(data ...any) *Response {
	data = append(data, nil)
	if r == nil {
		return &Response{Status: 1, Data: data[0]}
	}
	r.Status = 1
	r.Data = data[0]
	return r
}

func (r Response) SetData(data ...any) Response {
	if r.Err != nil {
		r.Message = r.Err.Error()
	} else {
		if r.Status = 1; data != nil {
			r.Data = data[0]
		}
	}

	return r
}

func (r Response) Error(format any, a ...any) Response {
	if format != nil {
		if r.Status = 0; a != nil {
			r.Message = fmt.Sprintf(format.(string), a...)
		} else {
			r.Message = fmt.Sprint(format)
		}
	}
	return r
}

func (r Response) Msg(format any, a ...any) Response {
	if format != nil {
		if a != nil {
			r.Message = fmt.Sprintf(format.(string), a...)
		} else {
			r.Message = fmt.Sprint(format)
		}
	}
	return r
}

func (r Response) SetCount(count int) Response {
	r.Count = count
	return r
}

func (r Response) SetCode(code int) Response {
	r.Code = code
	return r
}
