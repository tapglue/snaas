package user

import (
	"errors"
	"fmt"
)

const errFmt = "%s: %s"

// Common errors for User service implementations and validations.
var (
	ErrInvalidUser = errors.New("invalid user")
	ErrNotFound    = errors.New("user not found")
)

// Error wraps common User errors.
type Error struct {
	err error
	msg string
}

func (e Error) Error() string {
	return e.msg
}

// IsInvalidUser indicates if err is ErrInvalidUser.
func IsInvalidUser(err error) bool {
	return unwrapError(err) == ErrInvalidUser
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
