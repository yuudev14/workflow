// Path is "/" and not the auth route, because nginx strips the /auth-api/ prefix. A cookie
// with that path would be set for a path the browser never requests, so it would never be
// sent back. SameSite=Lax stops a cross-site page from calling /refresh or the websocket
// handshake.
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

func (w *CookieWriter) SetRefreshCookie(c *gin.Context, token string, expiresAt time.Time) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(RefreshCookieName, token, maxAgeUntil(expiresAt), "/", "", w.Secure, true)
}

func (w *CookieWriter) ClearRefreshCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(RefreshCookieName, "", -1, "/", "", w.Secure, true)
}

// ClearSession removes both cookies. Used on logout and when a refresh is rejected, so a
// dead session leaves nothing behind.
func (w *CookieWriter) ClearSession(c *gin.Context) {
	w.ClearAccessCookie(c)
	w.ClearRefreshCookie(c)
}

// SetOIDCStateCookie keeps the signed state and PKCE verifier during the trip to the IdP.
// It is short-lived and single-use, the callback clears it. Lax lets it come back with the
// top-level redirect from the provider.
func (w *CookieWriter) SetOIDCStateCookie(c *gin.Context, value string, ttl time.Duration) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(OIDCStateCookieName, value, int(ttl.Seconds()), "/", "", w.Secure, true)
}

func (w *CookieWriter) ClearOIDCStateCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(OIDCStateCookieName, "", -1, "/", "", w.Secure, true)
}

func ReadOIDCStateCookie(c *gin.Context) string {
	value, err := c.Cookie(OIDCStateCookieName)
	if err != nil {
		return ""
	}
	return value
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
