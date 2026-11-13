package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	rest "github.com/yuudev14/ytsoar/internal/adapters/http/rests"
	"github.com/yuudev14/ytsoar/internal/application/schedules"
	"github.com/yuudev14/ytsoar/internal/logger"
)

type ScheduleHandler struct {
	logger  logger.Logger
	service *schedules.Service
}

func NewScheduleHandler(log logger.Logger, service *schedules.Service) *ScheduleHandler {
	return &ScheduleHandler{logger: log, service: service}
}

func (h *ScheduleHandler) List(c *gin.Context) {
	response := rest.Response{C: c}

	var playbookID *uuid.UUID
	if raw := c.Query("playbook_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.ResponseError(http.StatusBadRequest, "playbook_id must be a uuid")
			return
		}
		playbookID = &id
	}
	list, err := h.service.List(c.Request.Context(), playbookID)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(list)
}

func (h *ScheduleHandler) Get(c *gin.Context) {
	response := rest.Response{C: c}

	id, ok := pathUUID(c, "schedule_id")
	if !ok {
		return
	}
	schedule, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(schedule)
}

func (h *ScheduleHandler) Create(c *gin.Context) {
	response := rest.Response{C: c}

	var payload schedules.CreatePayload
	if ok, code, err := rest.BindFormAndValidate(c, &payload); !ok {
		response.ResponseError(code, err)
		return
	}
	schedule, err := h.service.Create(c.Request.Context(), payload, actorID(c))
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.Response(http.StatusCreated, schedule)
}

func (h *ScheduleHandler) Update(c *gin.Context) {
	response := rest.Response{C: c}

	id, ok := pathUUID(c, "schedule_id")
	if !ok {
		return
	}
	var payload schedules.UpdatePayload
	if ok, code, err := rest.BindFormAndValidate(c, &payload); !ok {
		response.ResponseError(code, err)
		return
	}
	schedule, err := h.service.Update(c.Request.Context(), id, payload)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(schedule)
}

func (h *ScheduleHandler) Delete(c *gin.Context) {
	response := rest.Response{C: c}

	id, ok := pathUUID(c, "schedule_id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(gin.H{"deleted": true})
}

func (h *ScheduleHandler) Preview(c *gin.Context) {
	response := rest.Response{C: c}

	var payload schedules.PreviewPayload
	if ok, code, err := rest.BindFormAndValidate(c, &payload); !ok {
		response.ResponseError(code, err)
		return
	}
	next, err := h.service.Preview(c.Request.Context(), payload)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(gin.H{"next_runs": next})
}
