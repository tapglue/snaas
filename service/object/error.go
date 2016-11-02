package object

import (
	"errors"
	"fmt"
)

const errFmt = "%s: %s"

// Common errors for Object validation and Service.
var (
	ErrEmptySource       = errors.New("empty source")
	ErrInvalidAttachment = errors.New("invalid attachment")
	ErrInvalidObject     = errors.New("invalid object")
	ErrMissingReference  = errors.New("referenced object missing")
	ErrNamespaceNotFound = errors.New("namespace not found")
	ErrNotFound          = errors.New("object not found")
)

// Error wraps common Object errors.
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

// IsInvalidAttachment indicates if err is ErrInvalidAttachment.
func IsInvalidAttachment(err error) bool {
	return unwrapError(err) == ErrInvalidAttachment
}

// IsInvalidObject indicates if err is ErrInvalidObject.
func IsInvalidObject(err error) bool {
	return unwrapError(err) == ErrInvalidObject
}

// IsMissingReference indicates if err is ErrMissingReference.
func IsMissingReference(err error) bool {
	return unwrapError(err) == ErrMissingReference
}

// IsNamespaceNotFound indicates if err is ErrNamespaceNotFound.
func IsNamespaceNotFound(err error) bool {
	return unwrapError(err) == ErrNamespaceNotFound
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
