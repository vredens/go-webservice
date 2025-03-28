package webservice

import (
	"errors"
	"fmt"
)

type Error struct {
	internal error
	Code     int
}

func NewError(code int, err error) Error {
	return Error{internal: err, Code: code}
}

func (err Error) Error() string {
	if underlying := errors.Unwrap(err.internal); underlying == nil {
		return fmt.Sprintf("code=%d, message=%s", err.Code, err.internal.Error())
	}
	return fmt.Sprintf("code=%d, message=%s, internal=%v", err.Code, err.internal.Error(), errors.Unwrap(err.internal))
}

func (err Error) Unwrap() error {
	return errors.Unwrap(err.internal)
}

func (err Error) JSONFormatter() string {
	return fmt.Sprintf("{\"code\":%d,\"message\":%q}", err.Code, err.internal.Error())
}
