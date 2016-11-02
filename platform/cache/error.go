package cache

import (
	"errors"
	"fmt"
)

const errFmt = "%s: %s"

// Common errors for Event implementations.
var (
	ErrKeyNotFound = errors.New("key not found")
)

// Error wraps common Event errors.
type Error struct {
	err error
	msg string
}

func (e Error) Error() string {
	return e.msg
}

// IsKeyNotFound checks if err is ErrKeyNotFound.
func IsKeyNotFound(err error) bool {
	return unwrapError(err) == ErrKeyNotFound
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
