package sgin

import (
	"fmt"
)

type Response struct {
	Event   string `json:"event"`
	Status  int    `json:"status"`
	Code    int    `json:"code"`
	Count   int    `json:"count"`
	Message string `json:"msg"`
	Data    any    `json:"data"`
}

func (r Response) SetStatus(status int) Response {
	r.Status = status
	return r
}

func (r Response) SetCode(code int) Response {
	r.Code = code
	return r
}

func (r Response) SetCount(count int) Response {
	r.Count = count
	return r
}

func (r Response) OK(data ...any) Response {
	if r.Status = 1; data != nil {
		r.Data = data[0]
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

func (r Response) Error(format any, a ...any) Response {
	if format != nil {
		r.Status, r.Data = 0, nil
		if a != nil {
			r.Message = fmt.Sprintf(format.(string), a...)
		} else {
			r.Message = fmt.Sprint(format)
		}
	}
	return r
}
