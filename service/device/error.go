package device

import (
	"errors"
	"fmt"
)

const errFmt = "%s: %s"

// Common errors for Device serive implementations and validations.
var (
	ErrInvalidDevice = errors.New("invalid device")
)

// Error wraps common Device errors.
type Error struct {
	err error
	msg string
}

func (e Error) Error() string {
	return e.msg
}

// IsInvalidDevice indicates if err is ErrInvalidDevice.
func IsInvalidDevice(err error) bool {
	return unwrapError(err) == ErrInvalidDevice
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
