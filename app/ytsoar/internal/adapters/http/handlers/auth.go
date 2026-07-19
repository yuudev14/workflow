package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yuudev14/ytsoar/internal/adapters/http/middleware"
	rest "github.com/yuudev14/ytsoar/internal/adapters/http/rests"
	"github.com/yuudev14/ytsoar/internal/application/auth"
	"github.com/yuudev14/ytsoar/internal/logger"
)

type AuthHandler struct {
	logger      logger.Logger
	authService *auth.Service
	cookies     *middleware.CookieWriter
}

func NewAuthHandler(
	log logger.Logger,
	authService *auth.Service,
	cookies *middleware.CookieWriter,
) *AuthHandler {
	return &AuthHandler{logger: log, authService: authService, cookies: cookies}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// TokenResponse is the login/refresh body. The refresh token is absent on
// purpose — it goes out as an httpOnly cookie the frontend never reads.
type TokenResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// Login godoc
// @Summary Authenticate with a username and password
// @Tags auth
// @Router /api/auth/v1/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	response := rest.Response{C: c}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	pair, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			response.ResponseError(http.StatusUnauthorized, "invalid username or password")
			return
		}
		h.logger.Error(err)
		response.ResponseError(http.StatusInternalServerError, "could not sign in")
		return
	}

	h.cookies.SetRefreshCookie(c, pair.RefreshToken, pair.RefreshExpires)
	response.ResponseSuccess(TokenResponse{
		AccessToken: pair.AccessToken,
		ExpiresAt:   pair.AccessExpiresAt,
	})
}

// Refresh godoc
// @Summary Exchange the refresh cookie for a new access token
// @Tags auth
// @Router /api/auth/v1/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	response := rest.Response{C: c}

	refreshToken := middleware.ReadRefreshCookie(c)
	if refreshToken == "" {
		response.ResponseError(http.StatusUnauthorized, "no session")
		return
	}

	pair, err := h.authService.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			// The cookie is worthless now; clearing it stops the frontend
			// from retrying with it forever.
			h.cookies.ClearRefreshCookie(c)
			response.ResponseError(http.StatusUnauthorized, "session expired")
			return
		}
		h.logger.Error(err)
		response.ResponseError(http.StatusInternalServerError, "could not refresh session")
		return
	}

	h.cookies.SetRefreshCookie(c, pair.RefreshToken, pair.RefreshExpires)
	response.ResponseSuccess(TokenResponse{
		AccessToken: pair.AccessToken,
		ExpiresAt:   pair.AccessExpiresAt,
	})
}

// Logout godoc
// @Summary Revoke the current session
// @Tags auth
// @Router /api/auth/v1/logout [post]
//
// Deliberately unauthenticated: logging out has to work when the access token
// has already expired, and it can only ever revoke the caller's own cookie.
func (h *AuthHandler) Logout(c *gin.Context) {
	response := rest.Response{C: c}

	if err := h.authService.Logout(c.Request.Context(), middleware.ReadRefreshCookie(c)); err != nil {
		h.logger.Error(err)
	}

	h.cookies.ClearRefreshCookie(c)
	response.ResponseSuccess(gin.H{"message": "signed out"})
}

// Me godoc
// @Summary Current user, roles and permissions
// @Tags auth
// @Security BearerAuth
// @Router /api/auth/v1/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	response := rest.Response{C: c}

	authUser, ok := middleware.CurrentUser(c)
	if !ok {
		response.ResponseError(http.StatusUnauthorized, "unauthorized")
		return
	}

	me, err := h.authService.Me(c.Request.Context(), authUser.ID)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			response.ResponseError(http.StatusNotFound, "user not found")
			return
		}
		h.logger.Error(err)
		response.ResponseError(http.StatusInternalServerError, "could not load profile")
		return
	}

	response.ResponseSuccess(me)
}
