package core

import (
	"errors"
	"fmt"
)

const errFmt = "%s: %s"

// Common errors
var (
	ErrInvalidEntity = errors.New("invalid entity")
	ErrNotFound      = errors.New("resource not found")
	ErrUnauthorized  = errors.New("origin unauthorized")
)

// Error is a wraper used to transport core specific errors.
type Error struct {
	Err error
	Msg string
}

func (e *Error) Error() string {
	return e.Msg
}

// IsInvalidEntity indciates if err is ErrInvalidEntity.
func IsInvalidEntity(err error) bool {
	return unwrapError(err) == ErrInvalidEntity
}

// IsNotFound indicates if err is ErrNotFound.
func IsNotFound(err error) bool {
	return unwrapError(err) == ErrNotFound
}

// IsUnauthorized indicates if err is ErrUnauthorized.
func IsUnauthorized(err error) bool {
	return unwrapError(err) == ErrUnauthorized
}

func unwrapError(err error) error {
	switch e := err.(type) {
	case *Error:
		return e.Err
	}

	return err
}

func wrapError(err error, format string, args ...interface{}) error {
	return &Error{
		Err: err,
		Msg: fmt.Sprintf(
			errFmt,
			err.Error(),
			fmt.Sprintf(format, args...),
		),
	}
}
