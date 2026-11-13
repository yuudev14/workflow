package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/yuudev14/ytsoar/internal/adapters/http/middleware"
	"github.com/yuudev14/ytsoar/internal/domain"
)

func (h *ScheduleHandler) RegisterRoutes(
	route *gin.RouterGroup,
	requirePermission middleware.PermissionMiddleware,
) {
	r := route.Group("schedules")
	{
		readSchedules := requirePermission(domain.ModuleSchedules, domain.ActionRead)
		// a schedule runs a playbook, so a writer needs the grant to run playbooks too
		runPlaybooks := requirePermission(domain.ModulePlaybooks, domain.ActionExecute)

		r.GET("", readSchedules, h.List)
		r.POST("/preview", readSchedules, h.Preview)
		r.GET("/:schedule_id", readSchedules, h.Get)

		r.POST("", requirePermission(domain.ModuleSchedules, domain.ActionCreate), runPlaybooks, h.Create)
		r.PATCH("/:schedule_id", requirePermission(domain.ModuleSchedules, domain.ActionUpdate), runPlaybooks, h.Update)
		r.DELETE("/:schedule_id", requirePermission(domain.ModuleSchedules, domain.ActionDelete), h.Delete)
	}
}
