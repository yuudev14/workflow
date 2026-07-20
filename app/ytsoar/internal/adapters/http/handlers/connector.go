package handlers

import (
	"io"
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
		response.Fail(h.logger, err)
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
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(info)
}

// UploadConnector godoc
// @Summary Upload a connector as a zip (multipart field "file")
// @Tags connectors
// @Router /api/connectors/v1 [post]
func (h *ConnectorHandler) UploadConnector(c *gin.Context) {
	response := rest.Response{C: c}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, connectors.MaxUploadBytes)
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.ResponseError(http.StatusBadRequest, "multipart field 'file' with the connector zip is required")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		response.ResponseError(http.StatusBadRequest, "could not read the uploaded file")
		return
	}
	defer file.Close()
	zipBytes, err := io.ReadAll(file)
	if err != nil {
		response.ResponseError(http.StatusBadRequest, "could not read the uploaded file")
		return
	}

	info, err := h.ConnectorService.UploadConnector(c.Request.Context(), zipBytes, c.PostForm("uploaded_by"))
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	c.JSON(http.StatusCreated, info)
}

// DeleteConnector godoc
// @Summary Delete a connector from the tree
// @Tags connectors
// @Router /api/connectors/v1/{connector_id} [delete]
func (h *ConnectorHandler) DeleteConnector(c *gin.Context) {
	response := rest.Response{C: c}

	err := h.ConnectorService.DeleteConnector(c.Request.Context(), c.Param("connector_id"))
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	c.Status(http.StatusNoContent)
}
