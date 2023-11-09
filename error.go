package sgin

import (
	"github.com/baagod/sgin/utils"
)

// Error represents an error that occurred while handling a request.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewError creates a new Error instance with an optional message
func NewError(code int, message ...string) *Error {
	err := &Error{Code: code, Message: utils.StatusMessage(code)}
	if len(message) > 0 {
		err.Message = message[0]
	}
	return err
}

func (e *Error) Error() string {
	return e.Message
}
