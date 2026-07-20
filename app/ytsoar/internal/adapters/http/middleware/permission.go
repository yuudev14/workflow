package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

type PermissionSource interface {
	PermissionsFor(ctx context.Context, userID uuid.UUID) (domain.PermissionSet, error)
}

// RequirePermission gates a route on one (module, action) grant. Mount it
// after Auth.
//
// The lookup runs per request instead of reading roles off the JWT, so a role
// change or a deactivation takes effect immediately rather than at the user's
// next login.
func RequirePermission(log logger.Logger, src PermissionSource, module, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			unauthorized(c)
			return
		}

		permissions, err := src.PermissionsFor(c.Request.Context(), user.ID)
		if err != nil {
			// Fail closed: a database hiccup must never hand out access.
			log.Errorw("could not load permissions",
				"user_id", user.ID, "module", module, "action", action, "error", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "could not verify permissions",
			})
			return
		}

		if !permissions.Has(module, action) {
			log.Debugw("permission denied",
				"user_id", user.ID, "module", module, "action", action)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}

		c.Next()
	}
}
