package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/yuudev14/ytsoar/internal/adapters/http/middleware"
	"github.com/yuudev14/ytsoar/internal/domain"
)

func (h *PlaybookHandler) RegisterRoutes(
	route *gin.RouterGroup,
	requirePermission middleware.PermissionMiddleware,
) {
	r := route.Group("playbooks/v1")
	{
		readPlaybooks := requirePermission(domain.ModulePlaybooks, domain.ActionRead)

		r.GET("", readPlaybooks, h.GetPlaybooks)
		r.GET("/history", readPlaybooks, h.GetPlaybookHistory)
		r.GET("/history/:playbook_history_id/tasks", readPlaybooks, h.GetTaskHistoryByPlaybookHistoryId)
		r.GET("/:playbook_id", readPlaybooks, h.GetPlaybookGraphById)
		r.GET("/:playbook_id/tasks", readPlaybooks, h.GetTasksByPlaybookId)

		r.POST("",
			requirePermission(domain.ModulePlaybooks, domain.ActionCreate),
			h.CreatePlaybook)

		updatePlaybooks := requirePermission(domain.ModulePlaybooks, domain.ActionUpdate)

		r.PUT("/:playbook_id", updatePlaybooks, h.UpdatePlaybook)
		r.PUT("/tasks/:playbook_id", updatePlaybooks, h.UpdatePlaybookTasks)

		// Running a playbook is its own grant: editing and executing are
		// separate privileges.
		r.POST("/trigger/:playbook_id",
			requirePermission(domain.ModulePlaybooks, domain.ActionExecute),
			h.Trigger)
	}
}
