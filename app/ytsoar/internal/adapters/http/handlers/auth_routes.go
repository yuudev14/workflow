package handlers

import "github.com/gin-gonic/gin"

// RegisterPublicRoutes mounts the endpoints that run before a caller has an
// access token. Refresh and logout authenticate with the refresh cookie
// instead, so they must not sit behind the bearer middleware.
func (h *AuthHandler) RegisterPublicRoutes(route *gin.RouterGroup) {
	group := route.Group("auth/v1")
	{
		group.POST("/login", h.Login)
		group.POST("/refresh", h.Refresh)
		group.POST("/logout", h.Logout)
	}
}

func (h *AuthHandler) RegisterProtectedRoutes(route *gin.RouterGroup) {
	group := route.Group("auth/v1")
	{
		group.GET("/me", h.Me)
	}
}
