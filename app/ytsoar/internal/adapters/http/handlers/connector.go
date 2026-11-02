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
	ConfigService    connectors.ConfigService
}

func NewConnectorHandler(log logger.Logger, connectorService connectors.ConnectorService, configService connectors.ConfigService) *ConnectorHandler {
	return &ConnectorHandler{
		logger:           log,
		ConnectorService: connectorService,
		ConfigService:    configService,
	}
}

// GetConnectors godoc
// @Summary List available connectors
// @Tags connectors
// @Router /api/v1/connectors [get]
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
// @Router /api/v1/connectors/{connector_id} [get]
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
// @Summary Upload a connector as a zip (multipart field "file"). Returns 409 when the id exists, unless replace=true.
// @Tags connectors
// @Router /api/v1/connectors [post]
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

	info, err := h.ConnectorService.UploadConnector(c.Request.Context(), zipBytes, c.PostForm("uploaded_by"), c.Query("replace") == "true")
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	c.JSON(http.StatusCreated, info)
}

// DeleteConnector godoc
// @Summary Delete a connector from the tree
// @Tags connectors
// @Router /api/v1/connectors/{connector_id} [delete]
func (h *ConnectorHandler) DeleteConnector(c *gin.Context) {
	response := rest.Response{C: c}

	err := h.ConnectorService.DeleteConnector(c.Request.Context(), c.Param("connector_id"))
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListConfigs godoc
// @Summary List the named configs of a connector. Password fields are masked.
// @Tags connectors
// @Router /api/v1/connectors/{connector_id}/configs [get]
func (h *ConnectorHandler) ListConfigs(c *gin.Context) {
	response := rest.Response{C: c}

	configs, err := h.ConfigService.ListConfigs(c.Request.Context(), c.Param("connector_id"))
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(configs)
}

// CreateConfig godoc
// @Summary Add a named config to a connector. Every string value is encrypted.
// @Tags connectors
// @Router /api/v1/connectors/{connector_id}/configs [post]
func (h *ConnectorHandler) CreateConfig(c *gin.Context) {
	response := rest.Response{C: c}

	var body connectors.ConfigInput
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}
	config, err := h.ConfigService.CreateConfig(c.Request.Context(), actorID(c), c.Param("connector_id"), body)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	c.JSON(http.StatusCreated, config)
}

// UpdateConfig godoc
// @Summary Change a config. A password value of "********" keeps the stored secret.
// @Tags connectors
// @Router /api/v1/connectors/{connector_id}/configs/{config_id} [patch]
func (h *ConnectorHandler) UpdateConfig(c *gin.Context) {
	response := rest.Response{C: c}

	configID, ok := pathUUID(c, "config_id")
	if !ok {
		return
	}
	var body connectors.UpdateConfigInput
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}
	config, err := h.ConfigService.UpdateConfig(c.Request.Context(), actorID(c), c.Param("connector_id"), configID, body)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(config)
}

// DeleteConfig godoc
// @Summary Delete a config
// @Tags connectors
// @Router /api/v1/connectors/{connector_id}/configs/{config_id} [delete]
func (h *ConnectorHandler) DeleteConfig(c *gin.Context) {
	response := rest.Response{C: c}

	configID, ok := pathUUID(c, "config_id")
	if !ok {
		return
	}
	if err := h.ConfigService.DeleteConfig(c.Request.Context(), actorID(c), c.Param("connector_id"), configID); err != nil {
		response.Fail(h.logger, err)
		return
	}
	c.Status(http.StatusNoContent)
}
