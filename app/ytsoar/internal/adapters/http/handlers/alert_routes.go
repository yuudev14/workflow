package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/yuudev14/ytsoar/internal/adapters/http/middleware"
	"github.com/yuudev14/ytsoar/internal/domain"
)

func (h *AlertHandler) RegisterRoutes(
	route *gin.RouterGroup,
	requirePermission middleware.PermissionMiddleware,
) {
	r := route.Group("alerts/v1")
	{
		readAlerts := requirePermission(domain.ModuleAlerts, domain.ActionRead)
		createAlerts := requirePermission(domain.ModuleAlerts, domain.ActionCreate)
		updateAlerts := requirePermission(domain.ModuleAlerts, domain.ActionUpdate)

		r.GET("", readAlerts, h.List)
		r.GET("/summary", readAlerts, h.Summary)
		r.GET("/:alert_id", readAlerts, h.Get)
		r.GET("/:alert_id/notes", readAlerts, h.ListNotes)

		r.POST("", createAlerts, h.Create)
		r.POST("/batch", createAlerts, h.CreateBatch)

		// alerts:execute, not playbooks:update - "may run automation on alerts"
		// is a different grant from "may edit playbooks".
		r.POST("/run", requirePermission(domain.ModuleAlerts, domain.ActionExecute), h.Run)

		r.PATCH("/:alert_id", updateAlerts, h.Update)
		r.PATCH("/:alert_id/status", updateAlerts, h.UpdateStatus)

		r.POST("/:alert_id/notes", updateAlerts, h.AddNote)
		r.PATCH("/:alert_id/notes/:note_id", updateAlerts, h.UpdateNote)
		r.DELETE("/:alert_id/notes/:note_id", updateAlerts, h.DeleteNote)

		r.POST("/:alert_id/escalate",
			requirePermission(domain.ModuleIncidents, domain.ActionCreate),
			h.Escalate)
	}
}
