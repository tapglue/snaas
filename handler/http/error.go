package http

import (
	"errors"
	"fmt"

	"github.com/tapglue/snaas/core"
)

// Errors used for protocol control flow.
var (
	ErrBadRequest    = errors.New("bad request")
	ErrLimitExceeded = errors.New("limit")
	ErrUnauthorized  = errors.New("unauthorized")
)

// Error is used to carry additional error informaiton reported back to clients.
type Error struct {
	Err     error
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func wrapError(err error, msg string) *Error {
	return &Error{
		Err:     err,
		Message: fmt.Sprintf("%s: %s", err.Error(), msg),
	}
}

func unwrapError(err error) error {
	switch e := err.(type) {
	case *Error:
		return e.Err
	case *core.Error:
		return e.Err
	}

	return err
}
