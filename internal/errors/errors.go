package errors

import "fmt"

type LibError struct {
	Reason string
	Cause  error
}

func (e *LibError) Error() string {
	return e.Reason
}

func (e *LibError) Unwrap() error {
	return e.Cause
}

func CustomError(template string, params ...interface{}) error {
	return &LibError{
		Reason: fmt.Sprintf(template, params...),
	}
}

func CustomErrorWrap(template string, cause error, params ...interface{}) error {
	return &LibError{
		Reason: fmt.Sprintf(template, params...),
		Cause:  cause,
	}
}
