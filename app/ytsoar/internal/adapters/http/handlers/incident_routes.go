package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/yuudev14/ytsoar/internal/adapters/http/middleware"
	"github.com/yuudev14/ytsoar/internal/domain"
)

func (h *IncidentHandler) RegisterRoutes(
	route *gin.RouterGroup,
	requirePermission middleware.PermissionMiddleware,
) {
	r := route.Group("incidents/v1")
	{
		readIncidents := requirePermission(domain.ModuleIncidents, domain.ActionRead)
		updateIncidents := requirePermission(domain.ModuleIncidents, domain.ActionUpdate)

		r.GET("", readIncidents, h.List)
		r.GET("/summary", readIncidents, h.Summary)
		r.GET("/:incident_id", readIncidents, h.Get)
		r.GET("/:incident_id/notes", readIncidents, h.ListNotes)

		r.POST("",
			requirePermission(domain.ModuleIncidents, domain.ActionCreate),
			h.Create)

		// incidents:execute, not playbooks:update - "may run automation on
		// incidents" is a different grant from "may edit playbooks".
		r.POST("/run", requirePermission(domain.ModuleIncidents, domain.ActionExecute), h.Run)

		r.PATCH("/:incident_id", updateIncidents, h.Update)
		r.PATCH("/:incident_id/status", updateIncidents, h.UpdateStatus)

		r.POST("/:incident_id/notes", updateIncidents, h.AddNote)
		r.PATCH("/:incident_id/notes/:note_id", updateIncidents, h.UpdateNote)
		r.DELETE("/:incident_id/notes/:note_id", updateIncidents, h.DeleteNote)

		r.POST("/:incident_id/alerts", updateIncidents, h.LinkAlert)
		r.DELETE("/:incident_id/alerts/:alert_id", updateIncidents, h.UnlinkAlert)
	}
}
