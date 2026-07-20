package apperr_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yuudev14/ytsoar/internal/domain/apperr"
)

var errSentinel = apperr.New(apperr.NotFound, "user not found")

func TestUnclassifiedErrorIsInternalWithNoMessage(t *testing.T) {
	kind, msg := apperr.KindOf(errors.New("pq: connection refused"))

	assert.Equal(t, apperr.Internal, kind)
	assert.Empty(t, msg, "an unclassified message must never be offered to the client")
}

func TestNilErrorIsInternal(t *testing.T) {
	kind, msg := apperr.KindOf(nil)

	assert.Equal(t, apperr.Internal, kind)
	assert.Empty(t, msg)
}

func TestSentinelKeepsKindAndMessage(t *testing.T) {
	kind, msg := apperr.KindOf(errSentinel)

	assert.Equal(t, apperr.NotFound, kind)
	assert.Equal(t, "user not found", msg)
}

// The point of the whole package: a classified error carrying a driver error
// must surface its own message, never the cause's.
func TestWrappedCauseNeverReachesTheMessage(t *testing.T) {
	err := apperr.Wrap(apperr.Invalid, "role_ids must be uuids",
		errors.New("pq: invalid input syntax for type uuid"))

	kind, msg := apperr.KindOf(err)

	assert.Equal(t, apperr.Invalid, kind)
	assert.Equal(t, "role_ids must be uuids", msg)
	assert.NotContains(t, msg, "pq:")
}

// Wrapping preserves errors.Is, which every handler branch depends on.
func TestWrapKeepsSentinelIdentity(t *testing.T) {
	err := apperr.Wrap(apperr.NotFound, "no such user", errSentinel)

	assert.True(t, errors.Is(err, errSentinel))
}

// An unclassified error wrapped around a classified one still resolves to the
// inner classification, so an op-name wrap does not turn a 404 into a 500.
func TestOuterFormattingKeepsInnerKind(t *testing.T) {
	err := fmt.Errorf("auth.GetUser: %w", errSentinel)

	kind, msg := apperr.KindOf(err)

	assert.Equal(t, apperr.NotFound, kind)
	assert.Equal(t, "user not found", msg, "the op prefix is for logs, not for the client")
}

// The full chain stays available for logging even though it is never returned.
func TestErrorStringRetainsCauseForLogging(t *testing.T) {
	err := apperr.Wrap(apperr.Internal, "could not save user",
		errors.New("pq: duplicate key"))

	assert.Contains(t, err.Error(), "pq: duplicate key")
}
