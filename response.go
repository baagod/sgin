package sgin

import (
	"bytes"
	"fmt"

	"github.com/bytedance/sonic"
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

func (r *Response) Code(code int) *Response {
	if r == nil {
		return &Response{code: code}
	}
	r.code = code
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

func (r *Response) Data(data ...any) *Response {
	if r == nil {
		return &Response{status: 1, data: append(data, nil)[0]}
	}

	if r.status = 1; data != nil {
		r.data = data[0]
	}

	return r
}

func (r *Response) Error(format any, a ...any) *Response {
	status, message, data := 0, "", any(nil)
	if r != nil {
		status = r.status
		message = r.message
		data = r.data
	}

	if format != nil { // 有错误
		if status, data = 0, nil; a != nil {
			message = fmt.Sprintf(format.(string), a...)
		} else {
			message = fmt.Sprint(format)
		}
	}

	if r == nil {
		return &Response{status: status, message: message, data: data}
	}

	r.status = status
	r.message = message
	r.data = data
	return r
}
