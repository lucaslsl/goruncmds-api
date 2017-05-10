package app

import (
	"errors"
)

type Error interface {
	error
	StatusCode() int
	Data() []interface{}
}

type ErrorResponseDetail map[string]interface{}

type ErrorResponse struct {
	Code    int                   `json:"code"`
	Err     error                 `json:"error"`
	Details []ErrorResponseDetail `json:"details"`
}

func (e ErrorResponse) StatusCode() int {
	return e.Code
}

func (e ErrorResponse) Error() string {
	return e.Err.Error()
}

func (e ErrorResponse) Data() []interface{} {
	s := make([]interface{}, len(e.Details))
	for i, v := range e.Details {
		s[i] = v
	}
	return s
}

var ErrUnprocessableEntity = errors.New("Validation Failed")
var ErrNotFound = errors.New("Not Found")
var ErrRequestEntityTooLarge = errors.New("Request Body Too Large")
var ErrInternalServerError = errors.New("Internal Error")
var ErrInvalidRequest = errors.New("Invalid Request")
var ErrUnauthorized = errors.New("Unauthorized")
