// Package middleware holds the gin handlers that run before route handlers:
// authentication today, permission checks alongside it.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// authUserKey is unexported so only CurrentUser can read the context value.
const authUserKey = "auth_user"

// RefreshCookieName is the httpOnly cookie holding the refresh token. It is
// the only credential the browser stores; access tokens live in JS memory.
const RefreshCookieName = "ytsoar_rt"

// TokenVerifier is the slice of the auth service the middleware needs.
type TokenVerifier interface {
	VerifyAccessToken(tokenString string) (domain.AuthUser, error)
	VerifyRefreshTokenForWS(ctx context.Context, refreshToken string) (domain.AuthUser, error)
}

// Auth authenticates ordinary API requests from the Authorization header.
// Header-only is deliberate: a cross-site page can make the browser send
// cookies but cannot make it send headers, so these routes cannot be forged.
func Auth(log logger.Logger, verifier TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := bearerToken(c.GetHeader("Authorization"))
		if raw == "" {
			unauthorized(c)
			return
		}

		user, err := verifier.VerifyAccessToken(raw)
		if err != nil {
			log.Debugw("rejected access token", "path", c.Request.URL.Path)
			unauthorized(c)
			return
		}

		c.Set(authUserKey, user)
		c.Next()
	}
}

// AuthFromRefreshCookie authenticates the websocket handshake. A browser
// cannot attach headers to a WebSocket, but the handshake is a normal GET, so
// the refresh cookie rides along. The token is read only — never rotated —
// and verified against the database so a logout stops reconnects at once.
func AuthFromRefreshCookie(log logger.Logger, verifier TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := c.Cookie(RefreshCookieName)
		if err != nil || raw == "" {
			unauthorized(c)
			return
		}

		user, err := verifier.VerifyRefreshTokenForWS(c.Request.Context(), raw)
		if err != nil {
			log.Debugw("rejected websocket handshake", "path", c.Request.URL.Path)
			unauthorized(c)
			return
		}

		c.Set(authUserKey, user)
		c.Next()
	}
}

// CurrentUser returns the authenticated caller. The second result is false on
// routes that did not run an auth middleware.
func CurrentUser(c *gin.Context) (domain.AuthUser, bool) {
	value, exists := c.Get(authUserKey)
	if !exists {
		return domain.AuthUser{}, false
	}
	user, ok := value.(domain.AuthUser)
	return user, ok
}

func bearerToken(header string) string {
	scheme, value, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(value)
}

func unauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
}
