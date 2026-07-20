package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/adapters/http/common/dto"
	rest "github.com/yuudev14/ytsoar/internal/adapters/http/rests"
	"github.com/yuudev14/ytsoar/internal/application/edges"
	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	"github.com/yuudev14/ytsoar/internal/logger"
)

type PlaybookHandler struct {
	logger                     logger.Logger
	PlaybookService            playbooks.PlaybookService
	TaskService                tasks.TaskService
	EdgeService                edges.EdgeService
	PlaybookApplicationService playbooks.PlaybookApplicationService
}

func NewPlaybookHandler(
	log logger.Logger,
	PlaybookService playbooks.PlaybookService,
	TaskService tasks.TaskService,
	EdgeService edges.EdgeService,
	PlaybookApplicationService playbooks.PlaybookApplicationService,
) *PlaybookHandler {
	return &PlaybookHandler{
		logger:                     log,
		PlaybookService:            PlaybookService,
		TaskService:                TaskService,
		EdgeService:                EdgeService,
		PlaybookApplicationService: PlaybookApplicationService,
	}
}

func (w *PlaybookHandler) GetPlaybooks(c *gin.Context) {
	var (
		query  dto.FilterQuery
		filter playbooks.PlaybookFilter
	)

	response := rest.Response{C: c}

	if ok, code, err := rest.BindQueryAndValidate(c, &query); !ok {
		w.logger.Error(err)
		response.ResponseError(code, err)
		return
	}

	if ok, code, err := rest.BindQueryAndValidate(c, &filter); !ok {
		w.logger.Error(err)
		response.ResponseError(code, err)
		return
	}

	w.logger.Debugw(
		"get playbooks",
		"offset", query.Offset,
		"limit", query.Limit,
		"filter", filter,
	)

	playbooksData, err := w.PlaybookService.GetPlaybooksData(c.Request.Context(),
		query.Offset,
		query.Limit,
		filter,
	)
	if err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	response.ResponseSuccess(playbooksData)
}

func (w *PlaybookHandler) GetPlaybookGraphById(c *gin.Context) {
	response := rest.Response{C: c}
	playbookId := c.Param("playbook_id")

	playbook, playbookErr := w.PlaybookService.GetPlaybookGraphById(c.Request.Context(), playbookId)

	if playbookErr != nil {
		w.logger.Error(playbookErr)

		if errors.Is(playbookErr, playbooks.ErrPlaybookNotFound) {
			response.ResponseError(http.StatusNotFound, playbookErr.Error())
		} else {
			response.ResponseError(http.StatusInternalServerError, playbookErr.Error())
		}
		return
	}

	response.ResponseSuccess(playbook)
}

func (w *PlaybookHandler) GetPlaybookHistory(c *gin.Context) {
	var query dto.FilterQuery
	var filter playbooks.PlaybookHistoryFilter
	response := rest.Response{C: c}
	ok, code, err := rest.BindQueryAndValidate(c, &query)

	if !ok {
		w.logger.Error(err)
		response.ResponseError(code, err)
		return
	}

	ok, code, err = rest.BindQueryAndValidate(c, &filter)

	if !ok {
		w.logger.Error(err)
		response.ResponseError(code, err)
		return
	}

	w.logger.Debugf("queries: %v", query)
	w.logger.Debugf("filter: %v", filter)

	histories, historiesErr := w.PlaybookService.GetPlaybooksHistoryData(c.Request.Context(), query.Offset, query.Limit, filter)

	if historiesErr != nil {
		response.ResponseError(http.StatusBadRequest, historiesErr.Error())
		return
	}

	response.ResponseSuccess(histories)

}

func (w *PlaybookHandler) GetTaskHistoryByPlaybookHistoryId(c *gin.Context) {
	response := rest.Response{C: c}
	var taskHistoryFilter tasks.TaskHistoryFilter
	playbookHistoryId := c.Param("playbook_history_id")
	playbookHistoryUUID, uuidErr := uuid.Parse(playbookHistoryId)
	if uuidErr != nil {
		response.ResponseError(http.StatusNotFound, uuidErr)
		return
	}

	if ok, code, err := rest.BindQueryAndValidate(c, &taskHistoryFilter); !ok {
		w.logger.Error(err)
		response.ResponseError(code, err)
		return
	}

	playbookHistory, playbookHistoryErr := w.PlaybookService.GetPlaybookHistoryById(c.Request.Context(), playbookHistoryUUID)

	if playbookHistoryErr != nil {
		w.logger.Error(playbookHistoryErr)
		response.ResponseError(http.StatusNotFound, playbookHistoryErr)
		return
	}

	w.logger.Debugf("filter: %v", taskHistoryFilter)

	tasksHistory, err := w.TaskService.GetTaskHistoryByPlaybookHistoryId(c.Request.Context(), playbookHistoryId, taskHistoryFilter)

	if err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	response.ResponseSuccess(gin.H{
		"tasks": tasksHistory,
		"edges": playbookHistory.Edges,
	})
}

func (w *PlaybookHandler) CreatePlaybook(c *gin.Context) {
	var body playbooks.PlaybookPayload
	response := rest.Response{C: c}

	check, code, validErr := rest.BindFormAndValidate(c, &body)

	if !check {
		w.logger.Errorf("%v", validErr)
		response.ResponseError(code, validErr)
		return
	}

	playbook, err := w.PlaybookService.CreatePlaybook(c.Request.Context(), body)

	if err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	w.logger.Debug("created playbook")
	response.ResponseSuccess(playbook)

}

func (w *PlaybookHandler) UpdatePlaybook(c *gin.Context) {
	var body playbooks.UpdatePlaybookData
	response := rest.Response{C: c}
	playbookId := c.Param("playbook_id")

	check, code, validErr := rest.BindFormAndValidate(c, &body)

	if !check {
		w.logger.Errorf("%v", validErr)
		response.ResponseError(code, validErr)
		return
	}

	playbook, err := w.PlaybookService.UpdatePlaybook(c.Request.Context(), playbookId, body)

	if err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	w.logger.Debug("updated playbook")
	response.ResponseSuccess(playbook)

}

func (w *PlaybookHandler) UpdatePlaybookTasks(c *gin.Context) {
	var body playbooks.UpdatePlaybookTasksPayload
	response := rest.Response{C: c}
	playbookId := c.Param("playbook_id")
	check, code, validErr := rest.BindFormAndValidate(c, &body)

	if !check {
		w.logger.Errorf("%v", validErr)
		response.ResponseError(code, validErr)
		return
	}

	playbook, playbookErr := w.PlaybookApplicationService.UpdatePlaybookTasks(c.Request.Context(), playbookId, body)

	if playbookErr != nil {
		w.logger.Error(playbookErr)
		response.ResponseError(http.StatusInternalServerError, playbookErr.Error())
		return
	}

	response.ResponseSuccess(playbook)
}

func (w *PlaybookHandler) GetTasksByPlaybookId(c *gin.Context) {
	response := rest.Response{C: c}
	playbookId := c.Param("playbook_id")

	_, playbookErr := w.PlaybookService.GetPlaybookById(c.Request.Context(), playbookId)

	if playbookErr != nil {
		w.logger.Error(playbookErr)
		if errors.Is(playbookErr, playbooks.ErrPlaybookNotFound) {
			response.ResponseError(http.StatusNotFound, playbookErr.Error())
		} else {
			response.ResponseError(http.StatusInternalServerError, playbookErr.Error())
		}
		return
	}

	newTasks, newTaskErr := w.TaskService.GetTasksByPlaybookId(c.Request.Context(), playbookId)
	if newTaskErr != nil {
		w.logger.Errorf("error: %v", newTaskErr)
		response.ResponseError(http.StatusBadRequest, newTaskErr.Error())
		return
	}

	response.ResponseSuccess(gin.H{
		"tasks": newTasks,
	})
}

func (w *PlaybookHandler) Trigger(c *gin.Context) {
	response := rest.Response{C: c}
	playbookId := c.Param("playbook_id")

	// Without this the raw string reaches postgres and comes back as an
	// invalid-uuid-syntax error, which the branch below would report as 502.
	if _, uuidErr := uuid.Parse(playbookId); uuidErr != nil {
		response.ResponseError(http.StatusBadRequest, "playbook_id must be a uuid")
		return
	}

	data, triggerErr := w.PlaybookApplicationService.TriggerPlaybook(c.Request.Context(), playbookId)
	if triggerErr != nil {
		w.logger.Errorf("error when triggering the playbook: %v", triggerErr)
		if errors.Is(triggerErr, playbooks.ErrPlaybookNotFound) {
			response.ResponseError(http.StatusNotFound, triggerErr.Error())
		} else {
			response.ResponseError(http.StatusBadGateway, "could not trigger the playbook")
		}
		return
	}

	response.Response(http.StatusAccepted, data)
}

// "skipped" is deliberately absent: only the executor sets it, never a client.
func validateTaskStateChangePayload(body tasks.UpdatePlaybookTaskHistoryStatus) error {
	status := []string{
		"pending",
		"in_progress",
		"success",
		"failed",
	}

	for _, _status := range status {
		if body.Status == _status {
			return nil
		}
	}

	return fmt.Errorf("status does not exist in the choices %v", status)
}

func validatePlaybookStateChangePayload(body tasks.UpdatePlaybookTaskHistoryStatus) error {
	status := []string{
		"in_progress",
		"success",
		"failed",
	}

	for _, _status := range status {
		if body.Status == _status {
			return nil
		}
	}

	return fmt.Errorf("status does not exist in the choices %v", status)
}

func (w *PlaybookHandler) UpdateTaskStatus(c *gin.Context) {
	response := rest.Response{C: c}
	playbookHistoryId := c.Param("playbook_history_id")
	taskId := c.Param("task_id")
	var body tasks.UpdatePlaybookTaskHistoryStatus

	check, code, validErr := rest.BindFormAndValidate(c, &body)

	if !check {
		w.logger.Errorf("%v", validErr)
		response.ResponseError(code, validErr)
		return
	}

	payloadErr := validateTaskStateChangePayload(body)

	if payloadErr != nil {
		w.logger.Errorf("%v", payloadErr)
		response.ResponseError(http.StatusBadRequest, payloadErr.Error())
		return
	}

	task, updateTaskErr := w.TaskService.UpdateTaskStatus(c.Request.Context(), playbookHistoryId, taskId, body.Status)

	if updateTaskErr != nil {
		w.logger.Errorf("%v", updateTaskErr)
		response.ResponseError(http.StatusBadRequest, updateTaskErr.Error())
		return
	}

	response.Response(http.StatusAccepted, task)
}

func (w *PlaybookHandler) UpdatePlaybookStatus(c *gin.Context) {
	response := rest.Response{C: c}
	playbookHistoryId := c.Param("playbook_history_id")
	var body tasks.UpdatePlaybookTaskHistoryStatus

	check, code, validErr := rest.BindFormAndValidate(c, &body)

	if !check {
		w.logger.Errorf("%v", validErr)
		response.ResponseError(code, validErr)
		return
	}

	payloadErr := validatePlaybookStateChangePayload(body)

	if payloadErr != nil {
		w.logger.Errorf("%v", payloadErr)
		response.ResponseError(http.StatusBadRequest, payloadErr.Error())
		return
	}

	playbook, updatePlaybookErr := w.PlaybookService.UpdatePlaybookHistoryStatus(c.Request.Context(), playbookHistoryId, body.Status)

	if updatePlaybookErr != nil {
		w.logger.Errorf("%v", updatePlaybookErr)
		response.ResponseError(http.StatusBadRequest, updatePlaybookErr.Error())
		return
	}

	response.Response(http.StatusAccepted, playbook)
}
