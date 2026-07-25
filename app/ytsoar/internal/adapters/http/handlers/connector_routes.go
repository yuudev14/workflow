package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/yuudev14/ytsoar/internal/adapters/http/middleware"
	"github.com/yuudev14/ytsoar/internal/domain"
)

func (h *ConnectorHandler) RegisterRoutes(
	route *gin.RouterGroup,
	requirePermission middleware.PermissionMiddleware,
) {
	group := route.Group("connectors/v1")
	{
		readConnectors := requirePermission(domain.ModuleConnectors, domain.ActionRead)

		group.GET("", readConnectors, h.GetConnectors)
		group.GET("/:connector_id", readConnectors, h.GetConnector)

		// Upload extracts a zip and runs its dependency install, so this grant
		// is effectively "may run code on the platform".
		group.POST("",
			requirePermission(domain.ModuleConnectors, domain.ActionCreate),
			h.UploadConnector)

		group.DELETE("/:connector_id",
			requirePermission(domain.ModuleConnectors, domain.ActionDelete),
			h.DeleteConnector)
	}
}
