package error

import (
	"errors"
	"fmt"
)

const errFmt = "%s: %s"

// General-purpose errors.
var (
	ErrNotFound = errors.New("not found")
)

// Platform errors.
var (
	ErrDeviceDisabled   = errors.New("device disabled")
	ErrInvalidPlatform  = errors.New("invalid platform")
	ErrInvalidReaction  = errors.New("invalid reaction")
	ErrReactionNotFound = errors.New("reaction not found")
)

// Error wrapper.
type Error struct {
	err error
	msg string
}

func (e Error) Error() string {
	return e.msg
}

// IsDeviceDisabled indicates if err is ErrDeviceDisabled.
func IsDeviceDisabled(err error) bool {
	return unwrapError(err) == ErrDeviceDisabled
}

// IsInvalidPlatform indicates if err is ErrInvalidPlatform.
func IsInvalidPlatform(err error) bool {
	return unwrapError(err) == ErrInvalidPlatform
}

// IsInvalidReaction indicates if err is ErrInvalidReaction.
func IsInvalidReaction(err error) bool {
	return unwrapError(err) == ErrInvalidReaction
}

// IsReactionNotFound indicates if err is ErrReactionNotFound.
func IsReactionNotFound(err error) bool {
	return unwrapError(err) == ErrReactionNotFound
}

// IsNotFound indicates if err is ErrNotFouund.
func IsNotFound(err error) bool {
	return unwrapError(err) == ErrNotFound
}

// Wrap constructs an Error with proper messaaging.
func Wrap(err error, format string, args ...interface{}) error {
	return &Error{
		err: err,
		msg: fmt.Sprintf(
			errFmt,
			err, fmt.Sprintf(format, args...),
		),
	}
}

func unwrapError(err error) error {
	switch e := err.(type) {
	case *Error:
		return e.err
	}

	return err
}
