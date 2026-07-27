package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/yuudev14/ytsoar/internal/adapters/http/middleware"
	rest "github.com/yuudev14/ytsoar/internal/adapters/http/rests"
	"github.com/yuudev14/ytsoar/internal/application/alerts"
	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
	"github.com/yuudev14/ytsoar/internal/types"
)

type AlertHandler struct {
	logger       logger.Logger
	alertService *alerts.Service
	orchestrator playbooks.PlaybookApplicationService
}

func NewAlertHandler(log logger.Logger, alertService *alerts.Service, orchestrator playbooks.PlaybookApplicationService) *AlertHandler {
	return &AlertHandler{logger: log, alertService: alertService, orchestrator: orchestrator}
}

// actorID is optional on ingest: a forwarder posting alerts is authenticated
// but its user id is not the analyst who acted on them.
func actorID(c *gin.Context) *uuid.UUID {
	actor, ok := middleware.CurrentUser(c)
	if !ok {
		return nil
	}
	id := actor.ID
	return &id
}

// gin cannot bind a path param to uuid.UUID ([16]byte), so every id is parsed
// here rather than through the binder.
func pathUUID(c *gin.Context, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		response := rest.Response{C: c}
		response.ResponseError(http.StatusBadRequest, param+" must be a uuid")
		return uuid.Nil, false
	}
	return id, true
}

// bindRange is shared by every /summary handler: bind the raw bounds, then let
// Resolve apply the defaults and reject an impossible window.
func bindRange(c *gin.Context, log logger.Logger) (types.ResolvedRange, bool) {
	response := rest.Response{C: c}

	var raw types.DateRange
	if ok, code, err := rest.BindQueryAndValidate(c, &raw); !ok {
		response.ResponseError(code, err)
		return types.ResolvedRange{}, false
	}

	rng, err := raw.Resolve()
	if err != nil {
		response.Fail(log, err)
		return types.ResolvedRange{}, false
	}
	return rng, true
}

func (h *AlertHandler) List(c *gin.Context) {
	response := rest.Response{C: c}

	var filter alerts.AlertFilter
	if ok, code, err := rest.BindQueryAndValidate(c, &filter); !ok {
		response.ResponseError(code, err)
		return
	}

	page, err := h.alertService.List(c.Request.Context(), filter)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(page)
}

func (h *AlertHandler) Summary(c *gin.Context) {
	response := rest.Response{C: c}

	rng, ok := bindRange(c, h.logger)
	if !ok {
		return
	}

	summary, err := h.alertService.Summary(c.Request.Context(), rng)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(summary)
}

func (h *AlertHandler) Get(c *gin.Context) {
	response := rest.Response{C: c}

	id, ok := pathUUID(c, "alert_id")
	if !ok {
		return
	}

	detail, err := h.alertService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(detail)
}

func (h *AlertHandler) Create(c *gin.Context) {
	response := rest.Response{C: c}

	var payload alerts.CreateAlertPayload
	if ok, code, err := rest.BindFormAndValidate(c, &payload); !ok {
		response.ResponseError(code, err)
		return
	}

	alert, err := h.alertService.Create(c.Request.Context(), payload, actorID(c))
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.Response(http.StatusCreated, alert)
}

func (h *AlertHandler) CreateBatch(c *gin.Context) {
	response := rest.Response{C: c}

	var payload alerts.BatchCreateAlertPayload
	if ok, code, err := rest.BindFormAndValidate(c, &payload); !ok {
		response.ResponseError(code, err)
		return
	}

	created, failed, err := h.alertService.CreateBatch(c.Request.Context(), payload, actorID(c))
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.Response(http.StatusCreated, gin.H{
		"created": created,
		"failed":  failed,
	})
}

func (h *AlertHandler) Update(c *gin.Context) {
	response := rest.Response{C: c}

	id, ok := pathUUID(c, "alert_id")
	if !ok {
		return
	}

	var payload alerts.UpdateAlertPayload
	if ok, code, err := rest.BindFormAndValidate(c, &payload); !ok {
		response.ResponseError(code, err)
		return
	}

	alert, err := h.alertService.Update(c.Request.Context(), id, payload, actorID(c))
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(alert)
}

func (h *AlertHandler) UpdateStatus(c *gin.Context) {
	response := rest.Response{C: c}

	id, ok := pathUUID(c, "alert_id")
	if !ok {
		return
	}

	var payload alerts.UpdateAlertStatusPayload
	if ok, code, err := rest.BindFormAndValidate(c, &payload); !ok {
		response.ResponseError(code, err)
		return
	}

	alert, err := h.alertService.UpdateStatus(c.Request.Context(), id, payload, actorID(c))
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(alert)
}

func (h *AlertHandler) Escalate(c *gin.Context) {
	response := rest.Response{C: c}

	id, ok := pathUUID(c, "alert_id")
	if !ok {
		return
	}

	var payload alerts.EscalateAlertPayload
	if ok, code, err := rest.BindFormAndValidate(c, &payload); !ok {
		response.ResponseError(code, err)
		return
	}

	incident, err := h.alertService.Escalate(c.Request.Context(), id, payload, actorID(c))
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.Response(http.StatusCreated, incident)
}

func (h *AlertHandler) ListNotes(c *gin.Context) {
	response := rest.Response{C: c}

	id, ok := pathUUID(c, "alert_id")
	if !ok {
		return
	}

	notes, err := h.alertService.ListNotes(c.Request.Context(), id)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(notes)
}

func (h *AlertHandler) AddNote(c *gin.Context) {
	response := rest.Response{C: c}

	id, ok := pathUUID(c, "alert_id")
	if !ok {
		return
	}

	var payload alerts.AddNotePayload
	if ok, code, err := rest.BindFormAndValidate(c, &payload); !ok {
		response.ResponseError(code, err)
		return
	}

	note, err := h.alertService.AddNote(c.Request.Context(), id, payload, actorID(c))
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.Response(http.StatusCreated, note)
}

func (h *AlertHandler) UpdateNote(c *gin.Context) {
	response := rest.Response{C: c}

	noteID, ok := pathUUID(c, "note_id")
	if !ok {
		return
	}

	var payload alerts.UpdateNotePayload
	if ok, code, err := rest.BindFormAndValidate(c, &payload); !ok {
		response.ResponseError(code, err)
		return
	}

	note, err := h.alertService.UpdateNote(c.Request.Context(), noteID, payload, actorID(c))
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(note)
}

func (h *AlertHandler) DeleteNote(c *gin.Context) {
	response := rest.Response{C: c}

	noteID, ok := pathUUID(c, "note_id")
	if !ok {
		return
	}

	if err := h.alertService.DeleteNote(c.Request.Context(), noteID, actorID(c)); err != nil {
		response.Fail(h.logger, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Run starts one playbook run against the selected alerts. The body carries ids
// only; the records are hydrated server-side.
func (h *AlertHandler) Run(c *gin.Context) {
	response := rest.Response{C: c}

	var payload playbooks.RunPlaybookPayload
	if ok, code, err := rest.BindFormAndValidate(c, &payload); !ok {
		response.ResponseError(code, err)
		return
	}

	message, err := h.orchestrator.RunPlaybook(
		c.Request.Context(), payload.PlaybookID, domain.ModuleEventAlert, payload, actorID(c))
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.Response(http.StatusAccepted, message)
}
