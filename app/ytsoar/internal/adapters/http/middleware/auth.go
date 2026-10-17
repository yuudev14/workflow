// Package middleware has the gin handlers that run before the route handlers:
// authentication, and permission checks next to it.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// authUserKey is unexported, so only CurrentUser can read the context value
const authUserKey = "auth_user"

const (
	AccessCookieName    = "ytsoar_at"
	RefreshCookieName   = "ytsoar_rt"
	OIDCStateCookieName = "ytsoar_oidc"
)

type TokenVerifier interface {
	VerifyAccessToken(tokenString string) (domain.AuthUser, error)
	VerifyRefreshTokenForWS(ctx context.Context, refreshToken string) (domain.AuthUser, error)
	VerifyAPIKey(ctx context.Context, key string) (domain.AuthUser, error)
}

// Auth authenticates normal API requests.
//
// Browsers send the access token as an httpOnly cookie, so the frontend never handles the
// token. Service clients (agent service, worker, CLI) have no cookie jar, so an
// Authorization header is accepted too. The header wins when both exist. A value
// with domain.APIKeyPrefix is an API key, anything else is an access token.
//
// Because cookies are accepted, CSRF protection depends on SameSite=Lax and the exact-origin
// CORS allow-list, not on headers that cannot be forged. Lax keeps cookies off cross-site
// POST/PUT/DELETE, so mutations stay safe as long as they never hide behind GET.
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

		var user domain.AuthUser
		var err error
		if strings.HasPrefix(raw, domain.APIKeyPrefix) {
			user, err = verifier.VerifyAPIKey(c.Request.Context(), raw)
		} else {
			user, err = verifier.VerifyAccessToken(raw)
		}
		if err != nil {
			log.Debugw("rejected access token", "path", c.Request.URL.Path)
			unauthorized(c)
			return
		}

		SetCurrentUser(c, user)
		c.Next()
	}
}

// AuthFromRefreshCookie authenticates the websocket handshake. A browser cannot set
// headers on a WebSocket, but the handshake is a normal GET, so the refresh cookie comes
// with it. The token is read, never rotated, and checked against the database, so a logout
// stops reconnects at once.
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

// SetCurrentUser saves the authenticated caller. The context key stays unexported, so
// nothing outside this package can fake an identity.
func SetCurrentUser(c *gin.Context, user domain.AuthUser) {
	c.Set(authUserKey, user)
}

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
