package apperror

import (
	"errors"
	"fmt"
)

type Code string

const (
	CodeInvalidArgument   Code = "INVALID_ARGUMENT"
	CodeNotFound          Code = "NOT_FOUND"
	CodeInsufficientFunds Code = "INSUFFICIENT_FUNDS"
	CodeInternal          Code = "INTERNAL"
)

type AppError struct {
	Code Code
	Msg  string
	Err  error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	}
	return e.Msg
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func New(code Code, msg string) *AppError {
	return &AppError{Code: code, Msg: msg}
}

func Wrap(code Code, msg string, err error) *AppError {
	if err == nil {
		return New(code, msg)
	}
	return &AppError{Code: code, Msg: msg, Err: err}
}

func CodeOf(err error) Code {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return CodeInternal
}
