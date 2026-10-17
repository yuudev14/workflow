package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/yuudev14/ytsoar/internal/adapters/http/middleware"
	rest "github.com/yuudev14/ytsoar/internal/adapters/http/rests"
)

const oidcStateCookieTTL = 600 * time.Second

// Providers lists the enabled identity providers for the login screen. Public, because
// the browser needs it before it has a session.
func (h *AuthHandler) Providers(c *gin.Context) {
	response := rest.Response{C: c}

	providers, err := h.authService.ListProviders(c.Request.Context())
	if err != nil {
		h.logger.Error(err)
		response.ResponseError(http.StatusInternalServerError, "could not load providers")
		return
	}
	response.ResponseSuccess(providers)
}

// OIDCStart puts the state and PKCE verifier in a short-lived cookie and redirects (302)
// the browser to the identity provider.
func (h *AuthHandler) OIDCStart(c *gin.Context) {
	response := rest.Response{C: c}

	providerID, err := uuid.Parse(c.Param("provider_id"))
	if err != nil {
		response.ResponseError(http.StatusBadRequest, "provider_id must be a uuid")
		return
	}

	redirectURL, stateCookie, err := h.authService.StartOIDC(c.Request.Context(), providerID)
	if err != nil {
		h.logger.Error(err)
		c.Redirect(http.StatusFound, h.authService.FrontendURL()+"/login?error=sso")
		return
	}

	h.cookies.SetOIDCStateCookie(c, stateCookie, oidcStateCookieTTL)
	c.Redirect(http.StatusFound, redirectURL)
}

// OIDCCallback checks the state, exchanges the code, creates or syncs the user, sets the
// session cookies and returns the browser to the app. On any failure it goes to
// /login?error=sso and never shows an error body, because the user sees this top-level
// navigation.
func (h *AuthHandler) OIDCCallback(c *gin.Context) {
	frontend := h.authService.FrontendURL()

	providerID, err := uuid.Parse(c.Param("provider_id"))
	if err != nil {
		c.Redirect(http.StatusFound, frontend+"/login?error=sso")
		return
	}

	// the state cookie is single-use, whatever the result
	stateCookie := middleware.ReadOIDCStateCookie(c)
	h.cookies.ClearOIDCStateCookie(c)

	if errParam := c.Query("error"); errParam != "" {
		h.logger.Warnf("oidc provider returned error: %s", errParam)
		c.Redirect(http.StatusFound, frontend+"/login?error=sso")
		return
	}

	pair, err := h.authService.CompleteOIDC(
		c.Request.Context(),
		providerID,
		c.Query("code"),
		c.Query("state"),
		stateCookie,
	)
	if err != nil {
		h.logger.Warnf("oidc callback failed: %v", err)
		c.Redirect(http.StatusFound, frontend+"/login?error=sso")
		return
	}

	h.cookies.SetAccessCookie(c, pair.AccessToken, pair.AccessExpiresAt)
	h.cookies.SetRefreshCookie(c, pair.RefreshToken, pair.RefreshExpires)
	c.Redirect(http.StatusFound, frontend)
}
