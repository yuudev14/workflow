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

// Fail is the single exit for an error returned by a service or repository.
// A classified error becomes its status and its own message; anything else is
// a 500 whose detail goes to the log and never to the client.
//
// It logs once — layers below must return errors rather than logging them, or
// one failure appears several times with no way to tell they are one event.
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
