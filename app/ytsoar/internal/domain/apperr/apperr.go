// Package apperr classifies application errors, so the HTTP layer can pick a status code
// without importing every application package, and no error from below a handler reaches
// the client as it is.
package apperr

import "errors"

type Kind int

// Internal is the zero value on purpose: an unclassified error is a bug or an
// infrastructure failure, and both are 500.
const (
	Internal Kind = iota
	Invalid
	Unauthorized
	Forbidden
	NotFound
	Conflict
	Unavailable
)

// Error has a client-safe message. Anything unsafe goes in cause, which is logged and
// never shown.
type Error struct {
	kind  Kind
	msg   string
	cause error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return e.msg + ": " + e.cause.Error()
	}
	return e.msg
}

func (e *Error) Unwrap() error { return e.cause }

func (e *Error) Kind() Kind { return e.kind }

func New(kind Kind, msg string) *Error { return &Error{kind: kind, msg: msg} }

func Wrap(kind Kind, msg string, cause error) *Error {
	return &Error{kind: kind, msg: msg, cause: cause}
}

// KindOf returns the kind of the outermost classified error and the safe message. Only
// the msg of that error is returned, never the chain, so wrapping a driver error cannot
// send its text to a client.
//
// To add detail to a sentinel, wrap it, do not format around it:
// Wrap(Invalid, "role_ids must be uuids", ErrValidation) keeps errors.Is working and gives
// a message you wrote.
func KindOf(err error) (Kind, string) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.kind, appErr.msg
	}
	return Internal, ""
}
