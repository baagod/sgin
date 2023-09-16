package sgin

import (
	"fmt"
	"github.com/spf13/cast"
)

type Response struct {
	Event   string `json:"event"`
	Status  int    `json:"status"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
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

func (r *Response) WithMessage(message any) *Response {
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

func (r *Response) OK(data ...any) *Response {
	data = append(data, nil)
	if r == nil {
		return &Response{Status: 1, Data: data[0]}
	}
	r.Status = 1
	r.Data = data[0]
	return r
}
