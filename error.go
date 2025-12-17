package sgin

import (
    "net/http"
)

// Error 是 APIError 的默认实现
type Error struct {
    Code    int
    Message string
}

func (e *Error) Error() string {
    return e.Message
}

// NewError 创建一个新的 Error
// 如果没有提供消息，将使用 http.StatusText(code) 作为默认消息
func NewError(code int, msg ...string) *Error {
    if len(msg) == 0 {
        return &Error{Code: code, Message: http.StatusText(code)}
    }
    return &Error{Code: code, Message: msg[0]}
}

// --- 常用预定义错误 (函数形式，支持自定义消息) ---

// StatusNotModified 304
func StatusNotModified(msg ...string) *Error {
    return NewError(http.StatusNotModified, msg...)
}

// ErrBadRequest 400
func ErrBadRequest(msg ...string) *Error {
    return NewError(http.StatusBadRequest, msg...)
}

// ErrUnauthorized 401
func ErrUnauthorized(msg ...string) *Error {
    return NewError(http.StatusUnauthorized, msg...)
}

// ErrPaymentRequired 402
func ErrPaymentRequired(msg ...string) *Error {
    return NewError(http.StatusPaymentRequired, msg...)
}

// ErrForbidden 403
func ErrForbidden(msg ...string) *Error {
    return NewError(http.StatusForbidden, msg...)
}

// ErrNotFound 404
func ErrNotFound(msg ...string) *Error {
    return NewError(http.StatusNotFound, msg...)
}

// ErrMethodNotAllowed 405
func ErrMethodNotAllowed(msg ...string) *Error {
    return NewError(http.StatusMethodNotAllowed, msg...)
}

// ErrNotAcceptable 406
func ErrNotAcceptable(msg ...string) *Error {
    return NewError(http.StatusNotAcceptable, msg...)
}

// ErrProxyAuthRequired 407
func ErrProxyAuthRequired(msg ...string) *Error {
    return NewError(http.StatusProxyAuthRequired, msg...)
}

// ErrRequestTimeout 408
func ErrRequestTimeout(msg ...string) *Error {
    return NewError(http.StatusRequestTimeout, msg...)
}

// ErrConflict 409
func ErrConflict(msg ...string) *Error {
    return NewError(http.StatusConflict, msg...)
}

// ErrGone 410
func ErrGone(msg ...string) *Error {
    return NewError(http.StatusGone, msg...)
}

// ErrLengthRequired 411
func ErrLengthRequired(msg ...string) *Error {
    return NewError(http.StatusLengthRequired, msg...)
}

// ErrPreconditionFailed 412
func ErrPreconditionFailed(msg ...string) *Error {
    return NewError(http.StatusPreconditionFailed, msg...)
}

// ErrRequestEntityTooLarge 413
func ErrRequestEntityTooLarge(msg ...string) *Error {
    return NewError(http.StatusRequestEntityTooLarge, msg...)
}

// ErrRequestURITooLong 414
func ErrRequestURITooLong(msg ...string) *Error {
    return NewError(http.StatusRequestURITooLong, msg...)
}

// ErrUnsupportedMediaType 415
func ErrUnsupportedMediaType(msg ...string) *Error {
    return NewError(http.StatusUnsupportedMediaType, msg...)
}

// ErrRequestedRangeNotSatisfiable 416
func ErrRequestedRangeNotSatisfiable(msg ...string) *Error {
    return NewError(http.StatusRequestedRangeNotSatisfiable, msg...)
}

// ErrExpectationFailed 417
func ErrExpectationFailed(msg ...string) *Error {
    return NewError(http.StatusExpectationFailed, msg...)
}

// ErrTeapot 418
func ErrTeapot(msg ...string) *Error {
    return NewError(http.StatusTeapot, msg...)
}

// ErrMisdirectedRequest 421
func ErrMisdirectedRequest(msg ...string) *Error {
    return NewError(http.StatusMisdirectedRequest, msg...)
}

// ErrUnprocessableEntity 422
func ErrUnprocessableEntity(msg ...string) *Error {
    return NewError(http.StatusUnprocessableEntity, msg...)
}

// ErrLocked 423
func ErrLocked(msg ...string) *Error {
    return NewError(http.StatusLocked, msg...)
}

// ErrFailedDependency 424
func ErrFailedDependency(msg ...string) *Error {
    return NewError(http.StatusFailedDependency, msg...)
}

// ErrTooEarly 425
func ErrTooEarly(msg ...string) *Error {
    return NewError(http.StatusTooEarly, msg...)
}

// ErrUpgradeRequired 426
func ErrUpgradeRequired(msg ...string) *Error {
    return NewError(http.StatusUpgradeRequired, msg...)
}

// ErrPreconditionRequired 428
func ErrPreconditionRequired(msg ...string) *Error {
    return NewError(http.StatusPreconditionRequired, msg...)
}

// ErrTooManyRequests 429
func ErrTooManyRequests(msg ...string) *Error {
    return NewError(http.StatusTooManyRequests, msg...)
}

// ErrRequestHeaderFieldsTooLarge 431
func ErrRequestHeaderFieldsTooLarge(msg ...string) *Error {
    return NewError(http.StatusRequestHeaderFieldsTooLarge, msg...)
}

// ErrUnavailableForLegalReasons 451
func ErrUnavailableForLegalReasons(msg ...string) *Error {
    return NewError(http.StatusUnavailableForLegalReasons, msg...)
}

// ErrInternalServerError 500
func ErrInternalServerError(msg ...string) *Error {
    return NewError(http.StatusInternalServerError, msg...)
}

// ErrNotImplemented 501
func ErrNotImplemented(msg ...string) *Error {
    return NewError(http.StatusNotImplemented, msg...)
}

// ErrBadGateway 502
func ErrBadGateway(msg ...string) *Error {
    return NewError(http.StatusBadGateway, msg...)
}

// ErrServiceUnavailable 503
func ErrServiceUnavailable(msg ...string) *Error {
    return NewError(http.StatusServiceUnavailable, msg...)
}

// ErrGatewayTimeout 504
func ErrGatewayTimeout(msg ...string) *Error {
    return NewError(http.StatusGatewayTimeout, msg...)
}

// ErrHTTPVersionNotSupported 505
func ErrHTTPVersionNotSupported(msg ...string) *Error {
    return NewError(http.StatusHTTPVersionNotSupported, msg...)
}

// ErrVariantAlsoNegotiates 506
func ErrVariantAlsoNegotiates(msg ...string) *Error {
    return NewError(http.StatusVariantAlsoNegotiates, msg...)
}

// ErrInsufficientStorage 507
func ErrInsufficientStorage(msg ...string) *Error {
    return NewError(http.StatusInsufficientStorage, msg...)
}

// ErrLoopDetected 508
func ErrLoopDetected(msg ...string) *Error {
    return NewError(http.StatusLoopDetected, msg...)
}

// ErrNotExtended 510
func ErrNotExtended(msg ...string) *Error {
    return NewError(http.StatusNotExtended, msg...)
}

// ErrNetworkAuthenticationRequired 511
func ErrNetworkAuthenticationRequired(msg ...string) *Error {
    return NewError(http.StatusNetworkAuthenticationRequired, msg...)
}
