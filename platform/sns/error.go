package sns

import (
	"errors"
	"fmt"
)

const errFmt = "%s: %s"

// Common errors for Device serive implementations and validations.
var (
	ErrDeliveryFailure  = errors.New("delivery failed")
	ErrEndpointDisabled = errors.New("endppint disabled")
	ErrEndpointNotFound = errors.New("endpoint not found")
)

// Error wraps common Device errors.
type Error struct {
	err error
	msg string
}

func (e Error) Error() string {
	return e.msg
}

// IsDeliveryFailure indicates if err is ErrDeliveryFailure
func IsDeliveryFailure(err error) bool {
	return unwrapError(err) == ErrDeliveryFailure
}

// IsEndpointDisabled indicates if err is ErrEndpointDisabled.
func IsEndpointDisabled(err error) bool {
	return unwrapError(err) == ErrEndpointDisabled
}

// IsEndpointNotFound indicates if err is ErrEndpointNotFound.
func IsEndpointNotFound(err error) bool {
	return unwrapError(err) == ErrEndpointNotFound
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
