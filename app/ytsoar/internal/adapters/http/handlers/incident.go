package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	rest "github.com/yuudev14/ytsoar/internal/adapters/http/rests"
	"github.com/yuudev14/ytsoar/internal/application/incidents"
	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

type IncidentHandler struct {
	logger          logger.Logger
	incidentService *incidents.Service
	orchestrator    playbooks.PlaybookApplicationService
}

func NewIncidentHandler(log logger.Logger, incidentService *incidents.Service, orchestrator playbooks.PlaybookApplicationService) *IncidentHandler {
	return &IncidentHandler{logger: log, incidentService: incidentService, orchestrator: orchestrator}
}

func (h *IncidentHandler) List(c *gin.Context) {
	response := rest.Response{C: c}

	var filter incidents.IncidentFilter
	if ok, code, err := rest.BindQueryAndValidate(c, &filter); !ok {
		response.ResponseError(code, err)
		return
	}

	page, err := h.incidentService.List(c.Request.Context(), filter)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(page)
}

func (h *IncidentHandler) Summary(c *gin.Context) {
	response := rest.Response{C: c}

	rng, ok := bindRange(c, h.logger)
	if !ok {
		return
	}

	summary, err := h.incidentService.Summary(c.Request.Context(), rng)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(summary)
}

func (h *IncidentHandler) Get(c *gin.Context) {
	response := rest.Response{C: c}

	id, ok := pathUUID(c, "incident_id")
	if !ok {
		return
	}

	detail, err := h.incidentService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(detail)
}

func (h *IncidentHandler) Create(c *gin.Context) {
	response := rest.Response{C: c}

	var payload incidents.CreateIncidentPayload
	if ok, code, err := rest.BindFormAndValidate(c, &payload); !ok {
		response.ResponseError(code, err)
		return
	}

	incident, err := h.incidentService.Create(c.Request.Context(), payload, actorID(c))
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.Response(http.StatusCreated, incident)
}

func (h *IncidentHandler) Update(c *gin.Context) {
	response := rest.Response{C: c}

	id, ok := pathUUID(c, "incident_id")
	if !ok {
		return
	}

	var payload incidents.UpdateIncidentPayload
	if ok, code, err := rest.BindFormAndValidate(c, &payload); !ok {
		response.ResponseError(code, err)
		return
	}

	incident, err := h.incidentService.Update(c.Request.Context(), id, payload, actorID(c))
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(incident)
}

func (h *IncidentHandler) UpdateStatus(c *gin.Context) {
	response := rest.Response{C: c}

	id, ok := pathUUID(c, "incident_id")
	if !ok {
		return
	}

	var payload incidents.UpdateIncidentStatusPayload
	if ok, code, err := rest.BindFormAndValidate(c, &payload); !ok {
		response.ResponseError(code, err)
		return
	}

	incident, err := h.incidentService.UpdateStatus(c.Request.Context(), id, payload, actorID(c))
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(incident)
}

func (h *IncidentHandler) ListNotes(c *gin.Context) {
	response := rest.Response{C: c}

	id, ok := pathUUID(c, "incident_id")
	if !ok {
		return
	}

	notes, err := h.incidentService.ListNotes(c.Request.Context(), id)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(notes)
}

func (h *IncidentHandler) AddNote(c *gin.Context) {
	response := rest.Response{C: c}

	id, ok := pathUUID(c, "incident_id")
	if !ok {
		return
	}

	var payload incidents.AddNotePayload
	if ok, code, err := rest.BindFormAndValidate(c, &payload); !ok {
		response.ResponseError(code, err)
		return
	}

	note, err := h.incidentService.AddNote(c.Request.Context(), id, payload, actorID(c))
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.Response(http.StatusCreated, note)
}

func (h *IncidentHandler) UpdateNote(c *gin.Context) {
	response := rest.Response{C: c}

	noteID, ok := pathUUID(c, "note_id")
	if !ok {
		return
	}

	var payload incidents.UpdateNotePayload
	if ok, code, err := rest.BindFormAndValidate(c, &payload); !ok {
		response.ResponseError(code, err)
		return
	}

	note, err := h.incidentService.UpdateNote(c.Request.Context(), noteID, payload, actorID(c))
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(note)
}

func (h *IncidentHandler) DeleteNote(c *gin.Context) {
	response := rest.Response{C: c}

	noteID, ok := pathUUID(c, "note_id")
	if !ok {
		return
	}

	if err := h.incidentService.DeleteNote(c.Request.Context(), noteID, actorID(c)); err != nil {
		response.Fail(h.logger, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *IncidentHandler) LinkAlert(c *gin.Context) {
	response := rest.Response{C: c}

	id, ok := pathUUID(c, "incident_id")
	if !ok {
		return
	}

	var payload incidents.LinkAlertPayload
	if ok, code, err := rest.BindFormAndValidate(c, &payload); !ok {
		response.ResponseError(code, err)
		return
	}

	if err := h.incidentService.LinkAlert(c.Request.Context(), id, payload, actorID(c)); err != nil {
		response.Fail(h.logger, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *IncidentHandler) UnlinkAlert(c *gin.Context) {
	response := rest.Response{C: c}

	incidentID, ok := pathUUID(c, "incident_id")
	if !ok {
		return
	}
	alertID, ok := pathUUID(c, "alert_id")
	if !ok {
		return
	}

	if err := h.incidentService.UnlinkAlert(c.Request.Context(), incidentID, alertID, actorID(c)); err != nil {
		response.Fail(h.logger, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Run starts one playbook run against the selected incidents.
func (h *IncidentHandler) Run(c *gin.Context) {
	response := rest.Response{C: c}

	var payload playbooks.RunPlaybookPayload
	if ok, code, err := rest.BindFormAndValidate(c, &payload); !ok {
		response.ResponseError(code, err)
		return
	}

	message, err := h.orchestrator.RunPlaybook(
		c.Request.Context(), payload.PlaybookID, domain.ModuleEventIncident, payload, actorID(c))
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.Response(http.StatusAccepted, message)
}
