package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yuudev14/ytsoar/internal/adapters/http/middleware"
)

// captureCookie runs fn inside a handler and returns the cookie it wrote.
func captureCookie(t *testing.T, fn func(c *gin.Context)) *http.Cookie {
	t.Helper()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/probe", func(c *gin.Context) {
		fn(c)
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/probe", nil))

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	return cookies[0]
}

// HttpOnly is the whole reason the access token moved into a cookie: it is
// what stops an injected script reading the token and replaying it elsewhere.
func TestSetAccessCookieAttributes(t *testing.T) {
	writer := middleware.NewCookieWriter(false)

	cookie := captureCookie(t, func(c *gin.Context) {
		writer.SetAccessCookie(c, "the-access-token", time.Now().Add(15*time.Minute))
	})

	assert.Equal(t, middleware.AccessCookieName, cookie.Name)
	assert.Equal(t, "the-access-token", cookie.Value)
	assert.True(t, cookie.HttpOnly, "JS must never be able to read the access token")
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite,
		"Lax is what keeps the cookie off cross-site mutations now that headers are not required")
	assert.Equal(t, "/", cookie.Path)
	assert.Greater(t, cookie.MaxAge, 0)
}

func TestClearAccessCookie(t *testing.T) {
	writer := middleware.NewCookieWriter(false)

	cookie := captureCookie(t, func(c *gin.Context) {
		writer.ClearAccessCookie(c)
	})

	assert.Equal(t, middleware.AccessCookieName, cookie.Name)
	assert.Empty(t, cookie.Value)
	assert.Less(t, cookie.MaxAge, 0)
	assert.Equal(t, "/", cookie.Path)
}

// Logout and a rejected refresh must leave nothing behind, or the browser
// keeps replaying a dead session.
func TestClearSessionClearsBothCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := middleware.NewCookieWriter(false)

	router := gin.New()
	router.GET("/probe", func(c *gin.Context) {
		writer.ClearSession(c)
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/probe", nil))

	cleared := map[string]bool{}
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Value == "" && cookie.MaxAge < 0 {
			cleared[cookie.Name] = true
		}
	}

	assert.True(t, cleared[middleware.AccessCookieName], "access cookie must be cleared")
	assert.True(t, cleared[middleware.RefreshCookieName], "refresh cookie must be cleared")
}

// These attributes are the entire CSRF and XSS story for the refresh token,
// so they are asserted rather than assumed.
func TestSetRefreshCookieAttributes(t *testing.T) {
	writer := middleware.NewCookieWriter(false)

	cookie := captureCookie(t, func(c *gin.Context) {
		writer.SetRefreshCookie(c, "the-refresh-token", time.Now().Add(time.Hour))
	})

	assert.Equal(t, middleware.RefreshCookieName, cookie.Name)
	assert.Equal(t, "the-refresh-token", cookie.Value)
	assert.True(t, cookie.HttpOnly, "JS must never be able to read the refresh token")
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite,
		"Lax is what stops a cross-site page driving /refresh or the WS handshake")
	assert.Equal(t, "/", cookie.Path,
		"nginx strips the /auth-api prefix, so a scoped path would never be sent back")
	assert.False(t, cookie.Secure)
	assert.Greater(t, cookie.MaxAge, 0)
}

func TestSetRefreshCookieHonoursSecureFlag(t *testing.T) {
	writer := middleware.NewCookieWriter(true)

	cookie := captureCookie(t, func(c *gin.Context) {
		writer.SetRefreshCookie(c, "the-refresh-token", time.Now().Add(time.Hour))
	})

	assert.True(t, cookie.Secure, "COOKIE_SECURE=true must reach the browser")
}

// An already-expired token must not produce a negative MaxAge, which a browser
// reads as "delete this cookie" rather than "this is expired".
func TestSetRefreshCookieClampsPastExpiry(t *testing.T) {
	writer := middleware.NewCookieWriter(false)

	cookie := captureCookie(t, func(c *gin.Context) {
		writer.SetRefreshCookie(c, "stale", time.Now().Add(-time.Hour))
	})

	assert.GreaterOrEqual(t, cookie.MaxAge, 0)
}

func TestClearRefreshCookie(t *testing.T) {
	writer := middleware.NewCookieWriter(false)

	cookie := captureCookie(t, func(c *gin.Context) {
		writer.ClearRefreshCookie(c)
	})

	assert.Equal(t, middleware.RefreshCookieName, cookie.Name)
	assert.Empty(t, cookie.Value)
	assert.Less(t, cookie.MaxAge, 0, "a negative MaxAge is what deletes the cookie")
	assert.True(t, cookie.HttpOnly)
	assert.Equal(t, "/", cookie.Path,
		"must match the path it was set with, or the browser keeps the old cookie")
}

func TestReadRefreshCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var got string
	router := gin.New()
	router.GET("/probe", func(c *gin.Context) {
		got = middleware.ReadRefreshCookie(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.AddCookie(&http.Cookie{Name: middleware.RefreshCookieName, Value: "cookie-value"})
	router.ServeHTTP(httptest.NewRecorder(), req)

	assert.Equal(t, "cookie-value", got)
}

// No cookie is an ordinary case (a signed-out visitor), not an error.
func TestReadRefreshCookieAbsent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var got string
	router := gin.New()
	router.GET("/probe", func(c *gin.Context) {
		got = middleware.ReadRefreshCookie(c)
		c.Status(http.StatusOK)
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/probe", nil))

	assert.Empty(t, got)
}
