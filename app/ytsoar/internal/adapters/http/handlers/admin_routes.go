package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/yuudev14/ytsoar/internal/adapters/http/middleware"
	"github.com/yuudev14/ytsoar/internal/domain"
)

// RegisterRoutes mounts the settings module. Everything here administers who
// may do what, so it all maps to `settings.*` — §10 has no separate users or
// roles module.
func (h *AdminHandler) RegisterRoutes(
	route *gin.RouterGroup,
	requirePermission middleware.PermissionMiddleware,
) {
	read := requirePermission(domain.ModuleSettings, domain.ActionRead)
	create := requirePermission(domain.ModuleSettings, domain.ActionCreate)
	update := requirePermission(domain.ModuleSettings, domain.ActionUpdate)
	remove := requirePermission(domain.ModuleSettings, domain.ActionDelete)

	users := route.Group("users/v1")
	{
		users.GET("", read, h.ListUsers)
		users.GET("/:user_id", read, h.GetUser)
		users.POST("", create, h.CreateUser)
		users.PUT("/:user_id", update, h.UpdateUser)
		users.PUT("/:user_id/roles", update, h.SetUserRoles)
		users.PUT("/:user_id/password", update, h.SetUserPassword)
		// Deactivation, not deletion — see the handler.
		users.DELETE("/:user_id", remove, h.DeactivateUser)
	}

	roles := route.Group("roles/v1")
	{
		roles.GET("", read, h.ListRoles)
		roles.GET("/:role_id", read, h.GetRole)
		roles.POST("", create, h.CreateRole)
		roles.PUT("/:role_id", update, h.UpdateRole)
		roles.PUT("/:role_id/permissions", update, h.SetRolePermissions)
		roles.DELETE("/:role_id", remove, h.DeleteRole)
	}

	teams := route.Group("teams/v1")
	{
		teams.GET("", read, h.ListTeams)
		teams.GET("/:team_id", read, h.GetTeam)
		teams.POST("", create, h.CreateTeam)
		teams.PUT("/:team_id", update, h.UpdateTeam)
		teams.PUT("/:team_id/members", update, h.SetTeamMembers)
		teams.DELETE("/:team_id", remove, h.DeleteTeam)
	}

	// Read-only by design: an editable audit trail is not an audit trail.
	route.Group("audit/v1").GET("", read, h.ListAuditLogs)
}
