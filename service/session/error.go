package session

import (
	"errors"
	"fmt"
)

const errFmt = "%s: %s"

// Common errors for Session service implementations and validations.
var (
	ErrInvalidSession = errors.New("invalid session")
	ErrNotFound       = errors.New("session not found")
)

// Error wraps common Session errors.
type Error struct {
	err error
	msg string
}

func (e Error) Error() string {
	return e.msg
}

// IsInvalidSession indicates if err is ErrInvalidSession.
func IsInvalidSession(err error) bool {
	return unwrapError(err) == ErrInvalidSession
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
