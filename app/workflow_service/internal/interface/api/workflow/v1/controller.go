package workflow_http_v1

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14-workflow/workflow-service/internal/edges"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/logging"
	"github.com/yuudev14-workflow/workflow-service/internal/interface/api/common/dto"
	rest "github.com/yuudev14-workflow/workflow-service/internal/interface/api/rests"
	workflow_application "github.com/yuudev14-workflow/workflow-service/internal/orchestrator/workflows"
	"github.com/yuudev14-workflow/workflow-service/internal/tasks"
	"github.com/yuudev14-workflow/workflow-service/internal/workflows"
)

type WorkflowController struct {
	WorkflowService            workflows.WorkflowService
	TaskService                tasks.TaskService
	EdgeService                edges.EdgeService
	WorkflowApplicationService workflow_application.WorkflowApplicationService
	DB                         *sqlx.DB
}

func NewWorkflowController(
	WorkflowService workflows.WorkflowService,
	TaskService tasks.TaskService,
	EdgeService edges.EdgeService,
	WorkflowApplicationService workflow_application.WorkflowApplicationService,
	DB *sqlx.DB,
) *WorkflowController {
	return &WorkflowController{
		WorkflowService:            WorkflowService,
		TaskService:                TaskService,
		EdgeService:                EdgeService,
		WorkflowApplicationService: WorkflowApplicationService,
		DB:                         DB,
	}
}

func (w *WorkflowController) GetWorkflows(c *gin.Context) {
	var (
		query  dto.FilterQuery
		filter workflows.WorkflowFilter
	)

	response := rest.Response{C: c}

	if ok, code, err := rest.BindQueryAndValidate(c, &query); !ok {
		logging.Sugar.Error(err)
		response.ResponseError(code, err)
		return
	}

	if ok, code, err := rest.BindQueryAndValidate(c, &filter); !ok {
		logging.Sugar.Error(err)
		response.ResponseError(code, err)
		return
	}

	logging.Sugar.Debugw(
		"get workflows",
		"offset", query.Offset,
		"limit", query.Limit,
		"filter", filter,
	)

	workflows, err := w.WorkflowService.GetWorkflowsData(
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

func (w *WorkflowController) GetWorkflowGraphById(c *gin.Context) {
	response := rest.Response{C: c}
	workflowId := c.Param("workflow_id")

	workflow, workflowErr := w.WorkflowService.GetWorkflowGraphById(workflowId)

	if workflowErr != nil {
		logging.Sugar.Error(workflowErr)
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

func (w *WorkflowController) GetWorkflowHistory(c *gin.Context) {
	var query dto.FilterQuery
	var filter workflows.WorkflowHistoryFilter
	response := rest.Response{C: c}
	ok, code, err := rest.BindQueryAndValidate(c, &query)

	if !ok {
		logging.Sugar.Error(err)
		response.ResponseError(code, err)
		return
	}

	ok, code, err = rest.BindQueryAndValidate(c, &filter)

	if !ok {
		logging.Sugar.Error(err)
		response.ResponseError(code, err)
		return
	}

	logging.Sugar.Debugf("queries: %v", query)
	logging.Sugar.Debugf("filter: %v", filter)

	logging.Sugar.Debug("getting worflows")
	histories, historiesErr := w.WorkflowService.GetWorkflowsHistoryData(query.Offset, query.Limit, filter)

	if historiesErr != nil {
		response.ResponseError(http.StatusBadRequest, historiesErr.Error())
		return
	}

	response.ResponseSuccess(histories)

}

func (w *WorkflowController) GetWorkflowTriggerTypes(c *gin.Context) {
	response := rest.Response{C: c}
	workflowTriggers, workflowErr := w.WorkflowService.GetWorkflowTriggers()

	if workflowErr != nil {
		logging.Sugar.Error(workflowErr)
		response.ResponseError(http.StatusInternalServerError, workflowErr.Error())
		return
	}

	response.ResponseSuccess(workflowTriggers)
}

func (w *WorkflowController) GetWorkflowById(c *gin.Context) {
	response := rest.Response{C: c}
	workflowId := c.Param("workflow_id")

	_, workflowErr := w.WorkflowService.GetWorkflowById(workflowId)

	if workflowErr != nil {
		logging.Sugar.Error(workflowErr)
		response.ResponseError(http.StatusInternalServerError, workflowErr.Error())
		return
	}

	newTasks, newTaskErr := w.TaskService.GetTasksByWorkflowId(workflowId)
	if newTaskErr != nil {
		logging.Sugar.Errorf("error: ", newTaskErr)
		response.ResponseError(http.StatusBadRequest, newTaskErr.Error())
		return
	}

	response.ResponseSuccess(gin.H{
		"tasks": newTasks,
	})
}

func (w *WorkflowController) GetTaskHistoryByWorkflowHistoryId(c *gin.Context) {
	response := rest.Response{C: c}
	var taskHistoryFilter tasks.TaskHistoryFilter
	workflowHistoryId := c.Param("workflow_history_id")
	workflowHistoryUUID, uuidErr := uuid.Parse(workflowHistoryId)
	if uuidErr != nil {
		response.ResponseError(http.StatusNotFound, uuidErr)
	}

	workflowHistory, workflowHistoryErr := w.WorkflowService.GetWorkflowHistoryById(workflowHistoryUUID)

	if workflowHistoryErr != nil {
		logging.Sugar.Error(workflowHistoryErr)
		response.ResponseError(http.StatusNotFound, workflowHistoryErr)
		return
	}

	logging.Sugar.Debugf("filter: %v", taskHistoryFilter)

	logging.Sugar.Debug("getting worflows")
	tasksHistory, err := w.TaskService.GetTaskHistoryByWorkflowHistoryId(workflowHistoryId, taskHistoryFilter)

	if err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	response.ResponseSuccess(gin.H{
		"tasks": tasksHistory,
		"edges": workflowHistory.Edges,
	})
}

func (w *WorkflowController) CreateWorkflow(c *gin.Context) {
	var body workflows.WorkflowPayload
	response := rest.Response{C: c}

	check, code, validErr := rest.BindFormAndValidate(c, &body)

	if !check {
		logging.Sugar.Errorf(fmt.Sprintf("%v", validErr))
		response.ResponseError(code, validErr)
		return
	}

	workflow, err := w.WorkflowService.CreateWorkflow(body)

	logging.Sugar.Debug("added workflow...")

	if err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	response.ResponseSuccess(workflow)

}

func (w *WorkflowController) UpdateWorkflow(c *gin.Context) {
	var body workflows.UpdateWorkflowData
	response := rest.Response{C: c}
	workflowId := c.Param("workflow_id")

	check, code, validErr := rest.BindFormAndValidate(c, &body)

	if !check {
		logging.Sugar.Errorf(fmt.Sprintf("%v", validErr))
		response.ResponseError(code, validErr)
		return
	}

	workflow, err := w.WorkflowService.UpdateWorkflow(workflowId, body)

	logging.Sugar.Debug("added workflow...")

	if err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	response.ResponseSuccess(workflow)

}

func (w *WorkflowController) UpdateWorkflowTasks(c *gin.Context) {
	var body tasks.UpdateWorkflowtasks
	response := rest.Response{C: c}
	workflowId := c.Param("workflow_id")
	check, code, validErr := rest.BindFormAndValidate(c, &body)

	if !check {
		logging.Sugar.Errorf(fmt.Sprintf("%v", validErr))
		response.ResponseError(code, validErr)
		return
	}

	workflow, workflowErr := w.WorkflowApplicationService.UpdateWorkflowTasks(workflowId, body)

	if workflowErr != nil {
		logging.Sugar.Error(workflowErr)
		response.ResponseError(http.StatusInternalServerError, workflowErr.Error())
		return
	}

	response.ResponseSuccess(workflow)
}

func (w *WorkflowController) GetTasksByWorkflowId(c *gin.Context) {
	response := rest.Response{C: c}
	workflowId := c.Param("workflow_id")

	_, workflowErr := w.WorkflowService.GetWorkflowById(workflowId)

	if workflowErr != nil {
		logging.Sugar.Error(workflowErr)
		response.ResponseError(http.StatusInternalServerError, workflowErr.Error())
		return
	}

	newTasks, newTaskErr := w.TaskService.GetTasksByWorkflowId(workflowId)
	if newTaskErr != nil {
		logging.Sugar.Errorf("error: ", newTaskErr)
		response.ResponseError(http.StatusBadRequest, newTaskErr.Error())
		return
	}

	response.ResponseSuccess(gin.H{
		"tasks": newTasks,
	})
}

func (w *WorkflowController) Trigger(c *gin.Context) {
	response := rest.Response{C: c}
	workflowId := c.Param("workflow_id")
	data, triggerErr := w.WorkflowApplicationService.TriggerWorkflow(workflowId)
	if triggerErr != nil {
		logging.Sugar.Errorf("error when sending the message to queue", triggerErr)
		response.ResponseError(http.StatusBadGateway, triggerErr.Error())
		return
	}

	response.Response(http.StatusAccepted, data)
}

func validateTaskStateChangePayload(body tasks.UpdateWorkflowTaskHistoryStatus) error {
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

func validateWorkflowStateChangePayload(body tasks.UpdateWorkflowTaskHistoryStatus) error {
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

func (w *WorkflowController) UpdateTaskStatus(c *gin.Context) {
	response := rest.Response{C: c}
	workflowHistoryId := c.Param("workflow_history_id")
	taskId := c.Param("task_id")
	var body tasks.UpdateWorkflowTaskHistoryStatus

	check, code, validErr := rest.BindFormAndValidate(c, &body)

	if !check {
		logging.Sugar.Errorf(fmt.Sprintf("%v", validErr))
		response.ResponseError(code, validErr)
		return
	}

	payloadErr := validateTaskStateChangePayload(body)

	if payloadErr != nil {
		logging.Sugar.Errorf(fmt.Sprintf("%v", payloadErr))
		response.ResponseError(http.StatusBadRequest, payloadErr.Error())
		return
	}

	task, updateTaskErr := w.TaskService.UpdateTaskStatus(workflowHistoryId, taskId, body.Status)

	if updateTaskErr != nil {
		logging.Sugar.Errorf(fmt.Sprintf("%v", updateTaskErr))
		response.ResponseError(http.StatusBadRequest, updateTaskErr.Error())
		return
	}

	response.Response(http.StatusAccepted, task)
}

func (w *WorkflowController) UpdateWorkflowStatus(c *gin.Context) {
	response := rest.Response{C: c}
	workflowHistoryId := c.Param("workflow_history_id")
	var body tasks.UpdateWorkflowTaskHistoryStatus

	check, code, validErr := rest.BindFormAndValidate(c, &body)

	if !check {
		logging.Sugar.Errorf(fmt.Sprintf("%v", validErr))
		response.ResponseError(code, validErr)
		return
	}

	payloadErr := validateWorkflowStateChangePayload(body)

	if payloadErr != nil {
		logging.Sugar.Errorf(fmt.Sprintf("%v", payloadErr))
		response.ResponseError(http.StatusBadRequest, payloadErr.Error())
		return
	}

	workflow, updateWorkflowErr := w.WorkflowService.UpdateWorkflowHistoryStatus(workflowHistoryId, body.Status)

	if updateWorkflowErr != nil {
		logging.Sugar.Errorf(fmt.Sprintf("%v", updateWorkflowErr))
		response.ResponseError(http.StatusBadRequest, updateWorkflowErr.Error())
		return
	}

	response.Response(http.StatusAccepted, workflow)
}
