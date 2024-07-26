package sgin

import (
	"bytes"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/spf13/cast"
)

type response struct {
	Event   string `json:"event"`
	Status  int    `json:"status"`
	Code    int    `json:"code"`
	Count   int    `json:"count"`
	Message string `json:"msg"`
	Data    any    `json:"data"`
}

type Response struct {
	event   string
	status  int
	code    int
	count   int
	message string
	data    any
	err     bool // 只记录内部是否有错误，不返回。
}

func (r *Response) MarshalJSON() ([]byte, error) {
	return sonic.Marshal(&response{
		Event:   r.event,
		Status:  r.status,
		Code:    r.code,
		Count:   r.count,
		Message: r.message,
		Data:    r.data,
	})
}

func (r *Response) UnmarshalJSON(data []byte) (err error) {
	aux := &response{}
	dec := sonic.ConfigDefault.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	if err = dec.Decode(aux); err == nil {
		r.event = aux.Event
		r.status = aux.Status
		r.code = aux.Code
		r.count = aux.Count
		r.message = aux.Message
		r.data = aux.Data
	}

	return
}

func (r *Response) Event(event string) *Response {
	if r == nil {
		return &Response{event: event}
	}
	r.event = event
	return r
}

func (r *Response) Status(status int) *Response {
	if r == nil {
		return &Response{status: status}
	}
	r.status = status
	return r
}

// Code 设置响应代码。可以是数字、布尔值或可转为数字的字符串。
func (r *Response) Code(code any) *Response {
	if r == nil {
		return &Response{code: cast.ToInt(code)}
	}
	r.code = cast.ToInt(code)
	return r
}

func (r *Response) Count(count int) *Response {
	if r == nil {
		return &Response{count: count}
	}
	r.count = count
	return r
}

func (r *Response) Message(format any, a ...any) *Response {
	var message string
	if a != nil {
		message = fmt.Sprintf(format.(string), a...)
	} else {
		message = fmt.Sprint(format)
	}

	if r == nil {
		return &Response{message: message}
	}

	r.message = message
	return r
}

// Data 设置响应数据。该方法同时将 r.status 设置为 1。
func (r *Response) Data(data ...any) *Response {
	if r == nil {
		return &Response{status: 1, data: append(data, nil)[0]}
	}

	if r.status = cast.ToInt(!r.err); r.err {
		r.data = nil
	} else {
		r.data = append(data, nil)[0]
	}

	return r
}

func (r *Response) Error(format any, a ...any) *Response {
	status, message, data, err := 0, "", any(nil), false

	if r != nil {
		status = r.status
		message = r.message
		data = r.data
		err = r.err
	}

	if format != nil { // 有错误
		if status, data = 0, nil; a != nil {
			message = fmt.Sprintf(format.(string), a...)
		} else {
			message = fmt.Sprint(format)
		}
		err = true
	}

	if r == nil {
		return &Response{status: status, message: message, data: data, err: err}
	}

	r.status = status
	r.message = message
	r.data = data
	r.err = err
	return r
}
