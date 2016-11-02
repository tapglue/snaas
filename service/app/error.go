package app

import (
	"errors"
	"fmt"
)

const errFmt = "%s: %s"

// Common errors for App services.
var (
	ErrNotFound = errors.New("app not found")
)

// Error wraps common App errors.
type Error struct {
	err error
	msg string
}

func (e Error) Error() string {
	return e.msg
}

// IsNotFound indicates if err is ErrNotFound.
func IsNotFound(err error) bool {
	return unwrapError(err) == ErrNotFound
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
			err,
			fmt.Sprintf(format, args...),
		),
	}
}
