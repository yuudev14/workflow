package workflow_controller_v1

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14-workflow/workflow-service/db"
	"github.com/yuudev14-workflow/workflow-service/dto"
	"github.com/yuudev14-workflow/workflow-service/models"
	"github.com/yuudev14-workflow/workflow-service/pkg/logging"
	rest "github.com/yuudev14-workflow/workflow-service/pkg/rests"
	"github.com/yuudev14-workflow/workflow-service/pkg/types"
	"github.com/yuudev14-workflow/workflow-service/service"
)

type WorkflowController struct {
	WorkflowService        service.WorkflowService
	TaskService            service.TaskService
	EdgeService            service.EdgeService
	WorkflowTriggerService service.WorkflowTriggerService
}

func NewWorkflowController(
	WorkflowService service.WorkflowService,
	TaskService service.TaskService,
	EdgeService service.EdgeService,
	WorkflowTriggerService service.WorkflowTriggerService,
) *WorkflowController {
	return &WorkflowController{
		WorkflowService:        WorkflowService,
		TaskService:            TaskService,
		EdgeService:            EdgeService,
		WorkflowTriggerService: WorkflowTriggerService,
	}
}

func (w *WorkflowController) GetWorkflows(c *gin.Context) {
	var query dto.FilterQuery
	var filter dto.WorkflowFilter
	response := rest.Response{C: c}

	checkQuery, codeQuery, validQueryErr := rest.BindQueryAndValidate(c, &query)

	if !checkQuery {
		logging.Sugar.Errorf(fmt.Sprintf("%v", validQueryErr))
		response.ResponseError(codeQuery, validQueryErr)
		return
	}

	checkFilter, codeFilter, validFilterErr := rest.BindQueryAndValidate(c, &filter)

	if !checkFilter {
		logging.Sugar.Errorf(fmt.Sprintf("%v", validFilterErr))
		response.ResponseError(codeFilter, validFilterErr)
		return
	}

	logging.Sugar.Debugf("queries: %v", query)
	logging.Sugar.Debugf("filter: %v", filter)

	logging.Sugar.Debug("getting worflows")
	workflows, err := w.WorkflowService.GetWorkflows(query.Offset, query.Limit, filter)

	if err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	workflowsCount, workflowsCountErr := w.WorkflowService.GetWorkflowsCount(filter)

	if workflowsCountErr != nil {
		response.ResponseError(http.StatusBadRequest, workflowsCountErr.Error())
		return
	}

	response.ResponseSuccess(gin.H{
		"entries": workflows,
		"total":   workflowsCount,
	})

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
	var filter dto.WorkflowHistoryFilter
	response := rest.Response{C: c}

	checkQuery, codeQuery, validQueryErr := rest.BindQueryAndValidate(c, &query)

	if !checkQuery {
		logging.Sugar.Errorf(fmt.Sprintf("%v", validQueryErr))
		response.ResponseError(codeQuery, validQueryErr)
		return
	}

	checkFilter, codeFilter, validFilterErr := rest.BindQueryAndValidate(c, &filter)

	if !checkFilter {
		logging.Sugar.Errorf(fmt.Sprintf("%v", validFilterErr))
		response.ResponseError(codeFilter, validFilterErr)
		return
	}

	logging.Sugar.Debugf("queries: %v", query)
	logging.Sugar.Debugf("filter: %v", filter)

	logging.Sugar.Debug("getting worflows")
	workflows, err := w.WorkflowService.GetWorkflowHistory(query.Offset, query.Limit, filter)

	if err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	workflowsCount, workflowsCountErr := w.WorkflowService.GetWorkflowHistoryCount(filter)

	if workflowsCountErr != nil {
		response.ResponseError(http.StatusBadRequest, workflowsCountErr.Error())
		return
	}

	response.ResponseSuccess(gin.H{
		"entries": workflows,
		"total":   workflowsCount,
	})

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

func (w *WorkflowController) GetWorkflowHistoryById(c *gin.Context) {
	response := rest.Response{C: c}
	workflowHistoryId := c.Param("workflow_history_id")

	workflowHistory, workflowErr := w.WorkflowService.GetWorkflowHistoryById(workflowHistoryId)

	if workflowErr != nil {
		logging.Sugar.Error(workflowErr)
		response.ResponseError(http.StatusInternalServerError, workflowErr.Error())
		return
	}

	response.ResponseSuccess(workflowHistory)
}

func (w *WorkflowController) CreateWorkflow(c *gin.Context) {
	var body dto.WorkflowPayload
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
	var body dto.UpdateWorkflowData
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

func (w *WorkflowController) UpsertTasks(
	tx *sqlx.Tx,
	workflowUUID uuid.UUID,
	nodes []dto.Task,
) ([]models.Tasks, error) {
	// node to update
	var nodeToUpsert []models.Tasks
	for _, node := range nodes {
		nodeToUpsert = append(nodeToUpsert, models.Tasks{
			Name: node.Name,
			Parameters: func() types.JsonType {
				if node.Parameters != nil {
					return types.JsonType(*node.Parameters)
				}
				return nil
			}(),
			Description:   node.Description,
			Config:        node.Config.Value,
			ConnectorName: node.ConnectorName,
			ConnectorID:   node.ConnectorID,
			Operation:     node.Operation,
			X:             node.X,
			Y:             node.Y,
		})
	}

	logging.Sugar.Debugf("node to add: %v", nodeToUpsert)
	// save the tasks
	if len(nodeToUpsert) > 0 {
		return w.TaskService.UpsertTasks(tx, workflowUUID, nodeToUpsert)
	}
	return nil, nil
}

func (w *WorkflowController) InsertEdges(
	tx *sqlx.Tx,
	workflowUUID uuid.UUID,
	edges map[string][]string,
	tasks []models.Tasks,
	handles *map[string]map[string]dto.EdgeHandle,
) error {
	// node to update
	var edgeToInsert []models.Edges
	tasksMap := make(map[string]uuid.UUID)

	// initialize data to insert. in payload we have the name of the tasks but we need
	// to save the id instead of the name that why we need to
	// create a taskmap with name and uuid of the task to easily get the uuid from the edges
	for _, task := range tasks {
		tasksMap[task.Name] = task.ID
	}

	logging.Sugar.Debugf("tasksMap: %v", tasksMap)
	logging.Sugar.Debugf("edges: %v", edges)

	for key, values := range edges {
		for _, val := range values {
			sourceId, sourceIdOk := tasksMap[key]
			destinationID, destinationIdOk := tasksMap[val]
			if sourceIdOk && destinationIdOk {
				edge := models.Edges{
					SourceID:      sourceId,
					DestinationID: destinationID,
					WorkflowID:    workflowUUID,
				}

				// handle for frontend reference
				if handles != nil {
					handleMap := *handles
					if handleSourceKey, handleSourceKeyOk := handleMap[key]; handleSourceKeyOk {
						if handleDestKey, handleDestKeyOk := handleSourceKey[val]; handleDestKeyOk {
							edge.SourceHandle = sql.NullString{String: *handleDestKey.SourceHandle, Valid: true}
							edge.DestinationHandle = sql.NullString{String: *handleDestKey.DestinationHandle, Valid: true}
						}
					}

				}
				edgeToInsert = append(edgeToInsert, edge)
			} else {
				logging.Sugar.Debugf("edges data that are not added: %v %v", key, val)
			}
		}
	}

	logging.Sugar.Debugf("edges to add: %v", edgeToInsert)
	// save the edges
	if len(edgeToInsert) > 0 {
		_, err := w.EdgeService.InsertEdges(tx, edgeToInsert)
		return err
	}
	return nil
}

func (w *WorkflowController) DeleteTasks(
	tx *sqlx.Tx,
	workflowUUID uuid.UUID,
	nodes []dto.Task,
) error {
	// node to delete
	var nodeToDelete []uuid.UUID
	tasksBodyMap := make(map[string]bool)

	// verify nodes name should be unique
	tasks, tasksErr := w.TaskService.GetTasksByWorkflowId(workflowUUID.String())
	if tasksErr != nil {
		return tasksErr
	}
	logging.Sugar.Debugf("tasks: %v", tasks)

	for _, node := range nodes {
		tasksBodyMap[node.Name] = true
	}
	// 2. if node not in new nodes to be updated, delete
	for _, node := range tasks {
		_, ok := tasksBodyMap[node.Name]
		if !ok {
			nodeToDelete = append(nodeToDelete, node.ID)
		}
	}

	logging.Sugar.Debugf("node to delete: %v", nodeToDelete)
	if len(nodeToDelete) > 0 {
		logging.Sugar.Debugf("node to delete: %v", nodeToDelete)
		err := w.TaskService.DeleteTasks(tx, nodeToDelete)
		return err

	}
	return nil
}

// delete edges that doesnt exist in the body payload
func (w *WorkflowController) DeleteEdges(
	tx *sqlx.Tx,
	workflowUUID uuid.UUID,
	edges map[string][]string,
) error {

	var edgeToDelete []uuid.UUID
	edgesMap := make(map[[2]string]bool)

	// delete all edges from the workflow if nothing is in the payload
	if len(edges) == 0 {
		return w.EdgeService.DeleteAllWorkflowEdges(tx, workflowUUID.String())
	}

	workflowEdges, workflowEdgesErr := w.EdgeService.GetEdgesByWorkflowId(workflowUUID.String())
	logging.Sugar.Debug("workflow edges", workflowEdges)

	if workflowEdgesErr != nil {
		logging.Sugar.Error(workflowEdgesErr)
		return workflowEdgesErr
	}

	// populate the hashmap
	for key, values := range edges {
		for _, val := range values {
			edgesMap[[2]string{key, val}] = true
		}
	}

	// if the edge does not exist in the hashmap, add to the delete lists
	for _, edge := range workflowEdges {
		_, ok := edgesMap[[2]string{edge.SourceTaskName, edge.DestinationTaskName}]
		if !ok {
			edgeToDelete = append(edgeToDelete, edge.ID)
		}
	}

	logging.Sugar.Debugf("edge to delete: %v", edgeToDelete)
	if len(edgeToDelete) > 0 {
		deleteEdgesError := w.EdgeService.DeleteEdges(tx, edgeToDelete)
		return deleteEdgesError

	}
	return nil

}

func validateWorkflowTaskPayload(body dto.UpdateWorkflowtasks) error {
	_, ok := body.Edges["start"]
	if !ok {
		return fmt.Errorf("'Start' doesnt exist in edges")
	}

	for _, node := range body.Nodes {
		if node.Name == "start" {
			return nil
		}
	}

	return fmt.Errorf("'Start' doesnt exist in nodes")
}

func (w *WorkflowController) UpdateWorkflowTasks(c *gin.Context) {
	var body dto.UpdateWorkflowtasks
	response := rest.Response{C: c}
	workflowId := c.Param("workflow_id")
	tx, err := db.DB.Beginx()
	if err != nil {
		tx.Rollback()
		response.ResponseError(http.StatusInternalServerError, err)
		return
	}

	check, code, validErr := rest.BindFormAndValidate(c, &body)

	if !check {
		logging.Sugar.Errorf(fmt.Sprintf("%v", validErr))
		response.ResponseError(code, validErr)
		return
	}

	if body.Task != nil {
		_, errTask := w.WorkflowService.UpdateWorkflowTx(tx, workflowId, *body.Task)

		logging.Sugar.Debug("updated workflow...")

		if errTask != nil {
			response.ResponseError(http.StatusBadRequest, errTask.Error())
			return
		}
	}

	// validate if start node in body payload
	// payloadErr := validateWorkflowTaskPayload(body)

	// if payloadErr != nil {
	// 	response.ResponseError(code, payloadErr.Error())
	// 	return
	// }

	workflowUUID, err := uuid.Parse(workflowId)

	if err != nil {
		response.ResponseError(http.StatusInternalServerError, err.Error())
		return
	}

	// delete the edges first
	deleteEdgesErr := w.DeleteEdges(tx, workflowUUID, body.Edges)
	if deleteEdgesErr != nil {
		logging.Sugar.Error(deleteEdgesErr)
		tx.Rollback()
		response.ResponseError(http.StatusBadRequest, deleteEdgesErr.Error())
		return
	}

	// upsert the tasks. insert if doesnt exist, update when exist
	insertedTasks, upsertTasksErr := w.UpsertTasks(tx, workflowUUID, body.Nodes)
	if upsertTasksErr != nil {
		logging.Sugar.Error(upsertTasksErr)
		tx.Rollback()
		response.ResponseError(http.StatusBadRequest, upsertTasksErr.Error())
		return
	}

	// delete the tasks the we dont need anymore
	deleteTaskError := w.DeleteTasks(tx, workflowUUID, body.Nodes)
	if deleteTaskError != nil {
		logging.Sugar.Error(deleteTaskError)
		tx.Rollback()
		response.ResponseError(http.StatusInternalServerError, deleteTaskError.Error())
		return
	}

	// insert the new edges
	insertEdgeError := w.InsertEdges(tx, workflowUUID, body.Edges, insertedTasks, body.Handles)
	if insertEdgeError != nil {
		logging.Sugar.Error(insertEdgeError)
		tx.Rollback()
		response.ResponseError(http.StatusInternalServerError, insertEdgeError.Error())
		return
	}

	logging.Sugar.Debug("added workflow...")
	commitErr := tx.Commit()

	if commitErr != nil {
		logging.Sugar.Error(commitErr)
		tx.Rollback()
		response.ResponseError(http.StatusInternalServerError, commitErr.Error())
		return
	}

	workflow, workflowErr := w.WorkflowService.GetWorkflowGraphById(workflowId)

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
	data, triggerErr := w.WorkflowTriggerService.TriggerWorkflow(workflowId)
	if triggerErr != nil {
		logging.Sugar.Errorf("error when sending the message to queue", triggerErr)
		response.ResponseError(http.StatusBadGateway, triggerErr.Error())
		return
	}

	response.Response(http.StatusAccepted, data)
}

func validateTaskStateChangePayload(body dto.UpdateWorkflowTaskHistoryStatus) error {
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

func validateWorkflowStateChangePayload(body dto.UpdateWorkflowTaskHistoryStatus) error {
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
	var body dto.UpdateWorkflowTaskHistoryStatus

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
	var body dto.UpdateWorkflowTaskHistoryStatus

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
