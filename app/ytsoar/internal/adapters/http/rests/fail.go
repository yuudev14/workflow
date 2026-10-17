package rest

import (
	"net/http"

	"github.com/yuudev14/ytsoar/internal/domain/apperr"
	"github.com/yuudev14/ytsoar/internal/logger"
)

var statusByKind = map[apperr.Kind]int{
	apperr.Invalid:      http.StatusBadRequest,
	apperr.Unauthorized: http.StatusUnauthorized,
	apperr.Forbidden:    http.StatusForbidden,
	apperr.NotFound:     http.StatusNotFound,
	apperr.Conflict:     http.StatusConflict,
	apperr.Unavailable:  http.StatusBadGateway,
}

// Fail is the single exit for an error from a service or repository. A classified error
// becomes its status and its own message. Anything else is a 500, with the detail in the
// log and never sent to the client.
//
// It logs once: layers below must return errors, not log them, or one failure shows up
// several times and nobody knows it is one event.
func (r *Response) Fail(log logger.Logger, err error) {
	kind, msg := apperr.KindOf(err)

	status, ok := statusByKind[kind]
	if !ok {
		log.Errorw("unhandled request error",
			"err", err,
			"method", r.C.Request.Method,
			"path", r.C.FullPath(),
		)
		r.ResponseError(http.StatusInternalServerError, "internal server error")
		return
	}

	log.Debugw("request rejected",
		"err", err,
		"status", status,
		"method", r.C.Request.Method,
		"path", r.C.FullPath(),
	)
	r.ResponseError(status, msg)
}
