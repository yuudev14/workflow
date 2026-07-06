package handlers

import "github.com/gin-gonic/gin"

func (h *ConnectorHandler) RegisterRoutes(route *gin.RouterGroup) {
	group := route.Group("connectors/v1")
	{
		group.GET("", h.GetConnectors)
		group.GET("/:connector_id", h.GetConnector)
	}
}
