package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	rest "github.com/yuudev14/ytsoar/internal/adapters/http/rests"
	"github.com/yuudev14/ytsoar/internal/application/connectors"
	"github.com/yuudev14/ytsoar/internal/logger"
)

type ConnectorHandler struct {
	logger           logger.Logger
	ConnectorService connectors.ConnectorService
}

func NewConnectorHandler(log logger.Logger, connectorService connectors.ConnectorService) *ConnectorHandler {
	return &ConnectorHandler{
		logger:           log,
		ConnectorService: connectorService,
	}
}

// GetConnectors godoc
// @Summary List available connectors
// @Tags connectors
// @Router /api/connectors/v1 [get]
func (h *ConnectorHandler) GetConnectors(c *gin.Context) {
	response := rest.Response{C: c}

	infos, err := h.ConnectorService.GetConnectors(c.Request.Context())
	if err != nil {
		h.logger.Error(err)
		response.ResponseError(http.StatusInternalServerError, err.Error())
		return
	}
	response.ResponseSuccess(infos)
}

// GetConnector godoc
// @Summary Get one connector's metadata
// @Tags connectors
// @Router /api/connectors/v1/{connector_id} [get]
func (h *ConnectorHandler) GetConnector(c *gin.Context) {
	response := rest.Response{C: c}

	info, err := h.ConnectorService.GetConnector(c.Request.Context(), c.Param("connector_id"))
	if err != nil {
		if errors.Is(err, connectors.ErrConnectorNotFound) {
			response.ResponseError(http.StatusNotFound, err.Error())
			return
		}
		h.logger.Error(err)
		response.ResponseError(http.StatusInternalServerError, err.Error())
		return
	}
	response.ResponseSuccess(info)
}
