package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yuudev14/ytsoar/internal/adapters/http/middleware"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

var errRejected = errors.New("rejected")

// stubVerifier stands in for the auth service: it accepts exactly one token
// per channel and rejects everything else.
type stubVerifier struct {
	validAccessToken  string
	validRefreshToken string
	user              domain.AuthUser
}

func (s *stubVerifier) VerifyAccessToken(tokenString string) (domain.AuthUser, error) {
	if s.validAccessToken != "" && tokenString == s.validAccessToken {
		return s.user, nil
	}
	return domain.AuthUser{}, errRejected
}

func (s *stubVerifier) VerifyRefreshTokenForWS(_ context.Context, refreshToken string) (domain.AuthUser, error) {
	if s.validRefreshToken != "" && refreshToken == s.validRefreshToken {
		return s.user, nil
	}
	return domain.AuthUser{}, errRejected
}

func newStub() *stubVerifier {
	return &stubVerifier{
		user: domain.AuthUser{ID: uuid.New(), Username: "alice"},
	}
}

// runRequest mounts mw on a route that reports whoever the middleware
// authenticated, then plays req through it.
func runRequest(t *testing.T, mw gin.HandlerFunc, req *http.Request) (*httptest.ResponseRecorder, *domain.AuthUser) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	var seen *domain.AuthUser
	router := gin.New()
	router.GET("/probe", mw, func(c *gin.Context) {
		if user, ok := middleware.CurrentUser(c); ok {
			seen = &user
		}
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec, seen
}

func TestAuthAcceptsBearerToken(t *testing.T) {
	stub := newStub()
	stub.validAccessToken = "good-access-token"

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer good-access-token")

	rec, seen := runRequest(t, middleware.Auth(logger.NewNop(), stub), req)

	assert.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, seen, "CurrentUser should be populated")
	assert.Equal(t, stub.user.ID, seen.ID)
	assert.Equal(t, "alice", seen.Username)
}

func TestAuthRejectsMissingHeader(t *testing.T) {
	stub := newStub()
	stub.validAccessToken = "good-access-token"

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)

	rec, seen := runRequest(t, middleware.Auth(logger.NewNop(), stub), req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Nil(t, seen)
}

func TestAuthRejectsBadToken(t *testing.T) {
	stub := newStub()
	stub.validAccessToken = "good-access-token"

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer tampered")

	rec, _ := runRequest(t, middleware.Auth(logger.NewNop(), stub), req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// Anything that is not a Bearer scheme is not a credential we understand.
func TestAuthRejectsNonBearerSchemes(t *testing.T) {
	stub := newStub()
	stub.validAccessToken = "good-access-token"

	for _, header := range []string{
		"good-access-token",        // no scheme
		"Basic good-access-token",  // wrong scheme
		"Bearer",                   // scheme only
	} {
		req := httptest.NewRequest(http.MethodGet, "/probe", nil)
		req.Header.Set("Authorization", header)

		rec, _ := runRequest(t, middleware.Auth(logger.NewNop(), stub), req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code, "header %q must not authenticate", header)
	}
}

// The scheme is case-insensitive per RFC 7235.
func TestAuthAcceptsLowercaseBearer(t *testing.T) {
	stub := newStub()
	stub.validAccessToken = "good-access-token"

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "bearer good-access-token")

	rec, seen := runRequest(t, middleware.Auth(logger.NewNop(), stub), req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotNil(t, seen)
}


func TestAuthIgnoresRefreshCookie(t *testing.T) {
	stub := newStub()
	stub.validAccessToken = "good-access-token"
	stub.validRefreshToken = "good-refresh-token"

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.AddCookie(&http.Cookie{Name: middleware.RefreshCookieName, Value: "good-refresh-token"})

	rec, _ := runRequest(t, middleware.Auth(logger.NewNop(), stub), req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthAcceptsAccessCookie(t *testing.T) {
	stub := newStub()
	stub.validAccessToken = "good-access-token"

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.AddCookie(&http.Cookie{Name: middleware.AccessCookieName, Value: "good-access-token"})

	rec, seen := runRequest(t, middleware.Auth(logger.NewNop(), stub), req)

	assert.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, seen, "CurrentUser should be populated from the cookie")
	assert.Equal(t, stub.user.ID, seen.ID)
}

func TestAuthRejectsBadAccessCookie(t *testing.T) {
	stub := newStub()
	stub.validAccessToken = "good-access-token"

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.AddCookie(&http.Cookie{Name: middleware.AccessCookieName, Value: "tampered"})

	rec, _ := runRequest(t, middleware.Auth(logger.NewNop(), stub), req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}


func TestAuthPrefersHeaderOverCookie(t *testing.T) {
	stub := newStub()
	stub.validAccessToken = "good-access-token"

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer good-access-token")
	req.AddCookie(&http.Cookie{Name: middleware.AccessCookieName, Value: "stale-cookie"})

	rec, seen := runRequest(t, middleware.Auth(logger.NewNop(), stub), req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotNil(t, seen)
}

func TestAuthFromRefreshCookieAcceptsCookie(t *testing.T) {
	stub := newStub()
	stub.validRefreshToken = "good-refresh-token"

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.AddCookie(&http.Cookie{Name: middleware.RefreshCookieName, Value: "good-refresh-token"})

	rec, seen := runRequest(t, middleware.AuthFromRefreshCookie(logger.NewNop(), stub), req)

	assert.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, seen)
	assert.Equal(t, stub.user.ID, seen.ID)
}

func TestAuthFromRefreshCookieRejectsMissingCookie(t *testing.T) {
	stub := newStub()
	stub.validRefreshToken = "good-refresh-token"

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)

	rec, _ := runRequest(t, middleware.AuthFromRefreshCookie(logger.NewNop(), stub), req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthFromRefreshCookieRejectsRevokedToken(t *testing.T) {
	stub := newStub()
	stub.validRefreshToken = "good-refresh-token"

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.AddCookie(&http.Cookie{Name: middleware.RefreshCookieName, Value: "revoked-token"})

	rec, _ := runRequest(t, middleware.AuthFromRefreshCookie(logger.NewNop(), stub), req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// CurrentUser must not report a user on a route with no auth middleware.
func TestCurrentUserAbsentWithoutMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var ok bool
	router := gin.New()
	router.GET("/open", func(c *gin.Context) {
		_, ok = middleware.CurrentUser(c)
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/open", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.False(t, ok)
}
