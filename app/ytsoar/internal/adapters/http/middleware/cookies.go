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

// SetRefreshCookie writes the refresh token.
//
// Path is "/" rather than the auth route because nginx strips the /auth-api/
// prefix: a path-scoped cookie would be set for a path the browser never
// requests again, and would never be sent back. SameSite=Lax is what stops a
// cross-site page from driving /refresh or the websocket handshake.
func (w *CookieWriter) SetRefreshCookie(c *gin.Context, token string, expiresAt time.Time) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(RefreshCookieName, token, maxAge, "/", "", w.Secure, true)
}

func (w *CookieWriter) ClearRefreshCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(RefreshCookieName, "", -1, "/", "", w.Secure, true)
}

func ReadRefreshCookie(c *gin.Context) string {
	value, err := c.Cookie(RefreshCookieName)
	if err != nil {
		return ""
	}
	return value
}