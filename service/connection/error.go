package connection

import (
	"errors"
	"fmt"
)

const errFmt = "%s: %s"

// Common errors for Connection service implementations and validations.
var (
	ErrEmptySource       = errors.New("empty source")
	ErrInvalidConnection = errors.New("invalid connection")
)

type Error struct {
	err error
	msg string
}

func (e Error) Error() string {
	return e.msg
}

// IsEmptySource indicates if err is ErrEmptySource
func IsEmptySource(err error) bool {
	return unwrapError(err) == ErrEmptySource
}

// IsInvalidConnection indicates if err is ErrInvalidConnection.
func IsInvalidConnection(err error) bool {
	return unwrapError(err) == ErrInvalidConnection
}

func unwrapError(err error) error {
	switch e := err.(type) {
	case *Error:
		return e.err
	}

	return err
}

func wrapError(err error, format string, args ...interface{}) error {
	return &Error{
		err: err,
		msg: fmt.Sprintf(
			errFmt,
			err.Error(),
			fmt.Sprintf(format, args...),
		),
	}
}
