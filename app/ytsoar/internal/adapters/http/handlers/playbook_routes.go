package handlers

import (
	"github.com/gin-gonic/gin"
)

func (h *PlaybookHandler) RegisterRoutes(route *gin.RouterGroup) {
	r := route.Group("playbooks/v1")
	{
		r.GET("", h.GetPlaybooks)
		r.GET("/history", h.GetPlaybookHistory)
		r.GET("/history/:playbook_history_id/tasks", h.GetTaskHistoryByPlaybookHistoryId)
		r.GET("/:playbook_id", h.GetPlaybookGraphById)
		r.GET("/triggers", h.GetPlaybookTriggerTypes)
		r.POST("/trigger/:playbook_id", h.Trigger)
		r.POST("", h.CreatePlaybook)
		r.GET("/:playbook_id/tasks", h.GetTasksByPlaybookId)
		r.PUT("/:playbook_id", h.UpdatePlaybook)
		r.PUT("/tasks/:playbook_id", h.UpdatePlaybookTasks)
	}
}
