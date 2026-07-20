// Package apperr classifies application errors so the HTTP layer can choose a
// status code without importing every application package, and so no error
// from below a handler reaches the client verbatim.
package apperr

import "errors"

type Kind int

// Internal is the zero value on purpose: an unclassified error is a bug or an
// infrastructure failure, and both are 500s.
const (
	Internal Kind = iota
	Invalid
	Unauthorized
	Forbidden
	NotFound
	Conflict
	Unavailable
)

// Error carries a client-safe message. Anything unsafe belongs in cause, which
// is logged and never rendered.
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

// KindOf returns the classification of the outermost classified error and the
// message safe to send. Only that error's own msg is returned, never the
// chain, so wrapping a driver error cannot push its text out to a client.
//
// To attach detail to a sentinel, wrap it rather than formatting around it:
// Wrap(Invalid, "role_ids must be uuids", ErrValidation) keeps errors.Is
// working and still yields a message you wrote.
func KindOf(err error) (Kind, string) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.kind, appErr.msg
	}
	return Internal, ""
}
