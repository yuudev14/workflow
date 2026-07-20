// Path is "/" rather than the auth route because nginx strips the /auth-api/
// prefix: a path-scoped cookie would be set for a path the browser never
// requests again, and would never be sent back. SameSite=Lax is what stops a
// cross-site page from driving /refresh or the websocket handshake.
package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type CookieWriter struct {
	Secure bool
}

func NewCookieWriter(secure bool) *CookieWriter {
	return &CookieWriter{Secure: secure}
}

func maxAgeUntil(expiresAt time.Time) int {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		return 0
	}
	return maxAge
}

func (w *CookieWriter) SetAccessCookie(c *gin.Context, token string, expiresAt time.Time) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(AccessCookieName, token, maxAgeUntil(expiresAt), "/", "", w.Secure, true)
}

func (w *CookieWriter) ClearAccessCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(AccessCookieName, "", -1, "/", "", w.Secure, true)
}

// SetRefreshCookie writes the refresh token.
func (w *CookieWriter) SetRefreshCookie(c *gin.Context, token string, expiresAt time.Time) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(RefreshCookieName, token, maxAgeUntil(expiresAt), "/", "", w.Secure, true)
}

func (w *CookieWriter) ClearRefreshCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(RefreshCookieName, "", -1, "/", "", w.Secure, true)
}

// ClearSession removes both cookies. Used on logout and whenever a refresh is
// rejected, so a dead session leaves nothing behind.
func (w *CookieWriter) ClearSession(c *gin.Context) {
	w.ClearAccessCookie(c)
	w.ClearRefreshCookie(c)
}

func ReadAccessCookie(c *gin.Context) string {
	value, err := c.Cookie(AccessCookieName)
	if err != nil {
		return ""
	}
	return value
}

func ReadRefreshCookie(c *gin.Context) string {
	value, err := c.Cookie(RefreshCookieName)
	if err != nil {
		return ""
	}
	return value
}
