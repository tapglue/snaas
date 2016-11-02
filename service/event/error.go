package event

import (
	"errors"
	"fmt"
)

const errFmt = "%s: %s"

// Common errors for Event implementations.
var (
	ErrEmptySource       = errors.New("empty source")
	ErrInvalidEvent      = errors.New("invalid event")
	ErrNamespaceNotFound = errors.New("namespace not found")
	ErrNotFound          = errors.New("event not found")
)

// Error wraps common Event errors.
type Error struct {
	err error
	msg string
}

func (e Error) Error() string {
	return e.msg
}

// IsEmptySource indicates if err is ErrEmptySource.
func IsEmptySource(err error) bool {
	return unwrapError(err) == ErrEmptySource
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
