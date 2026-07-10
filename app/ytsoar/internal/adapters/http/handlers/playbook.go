package handlers

import (
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
		"get workflows",
		"offset", query.Offset,
		"limit", query.Limit,
		"filter", filter,
	)

	workflows, err := w.PlaybookService.GetPlaybooksData(c.Request.Context(),
		query.Offset,
		query.Limit,
		filter,
	)
	if err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	response.ResponseSuccess(workflows)
}

func (w *PlaybookHandler) GetPlaybookGraphById(c *gin.Context) {
	response := rest.Response{C: c}
	workflowId := c.Param("playbook_id")

	workflow, workflowErr := w.PlaybookService.GetPlaybookGraphById(c.Request.Context(), workflowId)

	if workflowErr != nil {
		w.logger.Error(workflowErr)
		errMsg := workflowErr.Error()

		if errMsg == "workflow is not found" {
			response.ResponseError(http.StatusNotFound, errMsg)
		} else {
			response.ResponseError(http.StatusInternalServerError, errMsg)
		}
		return
	}

	response.ResponseSuccess(workflow)
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

	w.logger.Debug("getting worflows")
	histories, historiesErr := w.PlaybookService.GetPlaybooksHistoryData(c.Request.Context(), query.Offset, query.Limit, filter)

	if historiesErr != nil {
		response.ResponseError(http.StatusBadRequest, historiesErr.Error())
		return
	}

	response.ResponseSuccess(histories)

}

func (w *PlaybookHandler) GetPlaybookById(c *gin.Context) {
	response := rest.Response{C: c}
	workflowId := c.Param("playbook_id")

	_, workflowErr := w.PlaybookService.GetPlaybookById(c.Request.Context(), workflowId)

	if workflowErr != nil {
		w.logger.Error(workflowErr)
		response.ResponseError(http.StatusInternalServerError, workflowErr.Error())
		return
	}

	newTasks, newTaskErr := w.TaskService.GetTasksByPlaybookId(c.Request.Context(), workflowId)
	if newTaskErr != nil {
		w.logger.Errorf("error: ", newTaskErr)
		response.ResponseError(http.StatusBadRequest, newTaskErr.Error())
		return
	}

	response.ResponseSuccess(gin.H{
		"tasks": newTasks,
	})
}

func (w *PlaybookHandler) GetTaskHistoryByPlaybookHistoryId(c *gin.Context) {
	response := rest.Response{C: c}
	var taskHistoryFilter tasks.TaskHistoryFilter
	workflowHistoryId := c.Param("playbook_history_id")
	workflowHistoryUUID, uuidErr := uuid.Parse(workflowHistoryId)
	if uuidErr != nil {
		response.ResponseError(http.StatusNotFound, uuidErr)
	}

	workflowHistory, workflowHistoryErr := w.PlaybookService.GetPlaybookHistoryById(c.Request.Context(), workflowHistoryUUID)

	if workflowHistoryErr != nil {
		w.logger.Error(workflowHistoryErr)
		response.ResponseError(http.StatusNotFound, workflowHistoryErr)
		return
	}

	w.logger.Debugf("filter: %v", taskHistoryFilter)

	w.logger.Debug("getting worflows")
	tasksHistory, err := w.TaskService.GetTaskHistoryByPlaybookHistoryId(c.Request.Context(), workflowHistoryId, taskHistoryFilter)

	if err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	response.ResponseSuccess(gin.H{
		"tasks": tasksHistory,
		"edges": workflowHistory.Edges,
	})
}

func (w *PlaybookHandler) CreatePlaybook(c *gin.Context) {
	var body playbooks.PlaybookPayload
	response := rest.Response{C: c}

	check, code, validErr := rest.BindFormAndValidate(c, &body)

	if !check {
		w.logger.Errorf(fmt.Sprintf("%v", validErr))
		response.ResponseError(code, validErr)
		return
	}

	workflow, err := w.PlaybookService.CreatePlaybook(c.Request.Context(), body)

	w.logger.Debug("added workflow...")

	if err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	response.ResponseSuccess(workflow)

}

func (w *PlaybookHandler) UpdatePlaybook(c *gin.Context) {
	var body playbooks.UpdatePlaybookData
	response := rest.Response{C: c}
	workflowId := c.Param("playbook_id")

	check, code, validErr := rest.BindFormAndValidate(c, &body)

	if !check {
		w.logger.Errorf(fmt.Sprintf("%v", validErr))
		response.ResponseError(code, validErr)
		return
	}

	workflow, err := w.PlaybookService.UpdatePlaybook(c.Request.Context(), workflowId, body)

	w.logger.Debug("added workflow...")

	if err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	response.ResponseSuccess(workflow)

}

func (w *PlaybookHandler) UpdatePlaybookTasks(c *gin.Context) {
	var body playbooks.UpdatePlaybookTasksPayload
	response := rest.Response{C: c}
	workflowId := c.Param("playbook_id")
	check, code, validErr := rest.BindFormAndValidate(c, &body)

	if !check {
		w.logger.Errorf(fmt.Sprintf("%v", validErr))
		response.ResponseError(code, validErr)
		return
	}

	workflow, workflowErr := w.PlaybookApplicationService.UpdatePlaybookTasks(c.Request.Context(), workflowId, body)

	if workflowErr != nil {
		w.logger.Error(workflowErr)
		response.ResponseError(http.StatusInternalServerError, workflowErr.Error())
		return
	}

	response.ResponseSuccess(workflow)
}

func (w *PlaybookHandler) GetTasksByPlaybookId(c *gin.Context) {
	response := rest.Response{C: c}
	workflowId := c.Param("playbook_id")

	_, workflowErr := w.PlaybookService.GetPlaybookById(c.Request.Context(), workflowId)

	if workflowErr != nil {
		w.logger.Error(workflowErr)
		response.ResponseError(http.StatusInternalServerError, workflowErr.Error())
		return
	}

	newTasks, newTaskErr := w.TaskService.GetTasksByPlaybookId(c.Request.Context(), workflowId)
	if newTaskErr != nil {
		w.logger.Errorf("error: ", newTaskErr)
		response.ResponseError(http.StatusBadRequest, newTaskErr.Error())
		return
	}

	response.ResponseSuccess(gin.H{
		"tasks": newTasks,
	})
}

func (w *PlaybookHandler) Trigger(c *gin.Context) {
	response := rest.Response{C: c}
	workflowId := c.Param("playbook_id")
	data, triggerErr := w.PlaybookApplicationService.TriggerPlaybook(c.Request.Context(), workflowId)
	if triggerErr != nil {
		w.logger.Errorf("error when sending the message to queue", triggerErr)
		response.ResponseError(http.StatusBadGateway, triggerErr.Error())
		return
	}

	response.Response(http.StatusAccepted, data)
}

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
	workflowHistoryId := c.Param("playbook_history_id")
	taskId := c.Param("task_id")
	var body tasks.UpdatePlaybookTaskHistoryStatus

	check, code, validErr := rest.BindFormAndValidate(c, &body)

	if !check {
		w.logger.Errorf(fmt.Sprintf("%v", validErr))
		response.ResponseError(code, validErr)
		return
	}

	payloadErr := validateTaskStateChangePayload(body)

	if payloadErr != nil {
		w.logger.Errorf(fmt.Sprintf("%v", payloadErr))
		response.ResponseError(http.StatusBadRequest, payloadErr.Error())
		return
	}

	task, updateTaskErr := w.TaskService.UpdateTaskStatus(c.Request.Context(), workflowHistoryId, taskId, body.Status)

	if updateTaskErr != nil {
		w.logger.Errorf(fmt.Sprintf("%v", updateTaskErr))
		response.ResponseError(http.StatusBadRequest, updateTaskErr.Error())
		return
	}

	response.Response(http.StatusAccepted, task)
}

func (w *PlaybookHandler) UpdatePlaybookStatus(c *gin.Context) {
	response := rest.Response{C: c}
	workflowHistoryId := c.Param("playbook_history_id")
	var body tasks.UpdatePlaybookTaskHistoryStatus

	check, code, validErr := rest.BindFormAndValidate(c, &body)

	if !check {
		w.logger.Errorf(fmt.Sprintf("%v", validErr))
		response.ResponseError(code, validErr)
		return
	}

	payloadErr := validatePlaybookStateChangePayload(body)

	if payloadErr != nil {
		w.logger.Errorf(fmt.Sprintf("%v", payloadErr))
		response.ResponseError(http.StatusBadRequest, payloadErr.Error())
		return
	}

	workflow, updatePlaybookErr := w.PlaybookService.UpdatePlaybookHistoryStatus(c.Request.Context(), workflowHistoryId, body.Status)

	if updatePlaybookErr != nil {
		w.logger.Errorf(fmt.Sprintf("%v", updatePlaybookErr))
		response.ResponseError(http.StatusBadRequest, updatePlaybookErr.Error())
		return
	}

	response.Response(http.StatusAccepted, workflow)
}
