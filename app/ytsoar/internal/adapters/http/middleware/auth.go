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

const (
	AccessCookieName  = "ytsoar_at"
	RefreshCookieName = "ytsoar_rt"
)

type TokenVerifier interface {
	VerifyAccessToken(tokenString string) (domain.AuthUser, error)
	VerifyRefreshTokenForWS(ctx context.Context, refreshToken string) (domain.AuthUser, error)
}

// Auth authenticates ordinary API requests.
//
// Browsers send the access token as an httpOnly cookie, which they attach
// automatically. the frontend never handles the token. Service clients (the
// agent service, the worker, the CLI) have no cookie jar, so an Authorization
// header is accepted too; the header wins when both are present.
//
// Accepting cookies means CSRF protection rests on SameSite=Lax plus the
// exact-origin CORS allow-list rather than on headers being unforgeable. Lax
// keeps cookies off cross-site POST/PUT/DELETE, so mutations stay safe as long
// as they never hide behind GET.
func Auth(log logger.Logger, verifier TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := bearerToken(c.GetHeader("Authorization"))
		if raw == "" {
			raw = ReadAccessCookie(c)
		}
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

		SetCurrentUser(c, user)
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

		SetCurrentUser(c, user)
		c.Next()
	}
}

// SetCurrentUser records the authenticated caller. The context key stays
// unexported so identity can only be set through here — nothing outside this
// package can forge one by writing the raw key.
func SetCurrentUser(c *gin.Context, user domain.AuthUser) {
	c.Set(authUserKey, user)
}

// CurrentUser returns the authenticated caller.
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
