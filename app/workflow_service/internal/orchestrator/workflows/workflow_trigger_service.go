package workflow_application

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14-workflow/workflow-service/internal/edges"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/logging"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/mq"
	"github.com/yuudev14-workflow/workflow-service/internal/tasks"
	"github.com/yuudev14-workflow/workflow-service/internal/workflows"
)

type WorkflowApplicationService interface {
	TriggerWorkflow(workflowId string) (*TaskMessage, error)
	PrepareWorkflowMessage(tasks []tasks.Tasks, edges []edges.ResponseEdges) (map[string]tasks.Tasks, map[string][]string)
	UpsertTasks(
		tx *sqlx.Tx,
		workflowUUID uuid.UUID,
		nodes []tasks.TaskPayload,
	) ([]tasks.Tasks, error)
	InsertEdges(
		tx *sqlx.Tx,
		workflowUUID uuid.UUID,
		edges map[string][]string,
		tasks []tasks.Tasks,
		handles *map[string]map[string]edges.EdgeHandle,
	) error
	DeleteTasks(
		tx *sqlx.Tx,
		workflowUUID uuid.UUID,
		nodes []tasks.TaskPayload,
	) error
	DeleteEdges(
		tx *sqlx.Tx,
		workflowUUID uuid.UUID,
		edges map[string][]string,
	) error
	UpdateWorkflowTasks(workflowId string, body tasks.UpdateWorkflowtasks) (*workflows.WorkflowsGraph, error)
}

type TaskMessage struct {
	Graph             map[string][]string    `json:"graph"`
	Tasks             map[string]tasks.Tasks `json:"tasks"`
	WorkflowHistoryId uuid.UUID              `json:"workflow_history_id"`
}

type WorkflowApplicationServiceImpl struct {
	WorkflowService workflows.WorkflowService
	TaskService     tasks.TaskService
	EdgeService     edges.EdgeService
	DB              *sqlx.DB
	TaskPubSUb      mq.TaskPubSub
}

func NewWorkflowApplicationService(WorkflowService workflows.WorkflowService, TaskService tasks.TaskService, EdgeService edges.EdgeService, DB *sqlx.DB, TaskPubSUb mq.TaskPubSub) WorkflowApplicationService {
	return &WorkflowApplicationServiceImpl{
		WorkflowService: WorkflowService,
		TaskService:     TaskService,
		EdgeService:     EdgeService,
		DB:              DB,
		TaskPubSUb:      TaskPubSUb,
	}
}

func (w *WorkflowApplicationServiceImpl) UpsertTasks(
	tx *sqlx.Tx,
	workflowUUID uuid.UUID,
	nodes []tasks.TaskPayload,
) ([]tasks.Tasks, error) {
	// node to update
	var nodeToUpsert []tasks.Tasks
	for _, node := range nodes {
		nodeToUpsert = append(nodeToUpsert, tasks.Tasks{
			Name: node.Name,
			Parameters: func() json.RawMessage {
				if node.Parameters != nil {
					b, _ := json.Marshal(node.Parameters)
					return b
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

func (w *WorkflowApplicationServiceImpl) InsertEdges(
	tx *sqlx.Tx,
	workflowUUID uuid.UUID,
	edges_ map[string][]string,
	tasks []tasks.Tasks,
	handles *map[string]map[string]edges.EdgeHandle,
) error {
	// node to update
	var edgeToInsert []edges.Edges
	tasksMap := make(map[string]uuid.UUID)

	// initialize data to insert. in payload we have the name of the tasks but we need
	// to save the id instead of the name that why we need to
	// create a taskmap with name and uuid of the task to easily get the uuid from the edges
	for _, task := range tasks {
		tasksMap[task.Name] = task.ID
	}

	logging.Sugar.Debugf("tasksMap: %v", tasksMap)
	logging.Sugar.Debugf("edges: %v", edges_)

	for key, values := range edges_ {
		for _, val := range values {
			sourceId, sourceIdOk := tasksMap[key]
			destinationID, destinationIdOk := tasksMap[val]
			if sourceIdOk && destinationIdOk {
				edge := edges.Edges{
					SourceID:      sourceId,
					DestinationID: destinationID,
					WorkflowID:    workflowUUID,
				}

				// handle for frontend reference
				if handles != nil {
					handleMap := *handles
					if handleSourceKey, handleSourceKeyOk := handleMap[key]; handleSourceKeyOk {
						if handleDestKey, handleDestKeyOk := handleSourceKey[val]; handleDestKeyOk {
							if handleDestKey.SourceHandle != nil {
								edge.SourceHandle = sql.NullString{String: *handleDestKey.SourceHandle, Valid: true}
							}
							if handleDestKey.DestinationHandle != nil {
								edge.DestinationHandle = sql.NullString{String: *handleDestKey.DestinationHandle, Valid: true}
							}
						}
					}

				}
				edgeToInsert = append(edgeToInsert, edge)
			} else {
				logging.Sugar.Infof("edges data that are not added: %v %v", key, val)
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

func (w *WorkflowApplicationServiceImpl) DeleteTasks(
	tx *sqlx.Tx,
	workflowUUID uuid.UUID,
	nodes []tasks.TaskPayload,
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
	logging.Sugar.Debugf("tasksBodyMap: %v", tasksBodyMap)
	// 2. if node not in new nodes to be updated, delete
	for _, node := range tasks {
		_, ok := tasksBodyMap[node.Name]
		logging.Sugar.Debugf("checking if node to be deleted for: %v", node.Name)
		if !ok {
			nodeToDelete = append(nodeToDelete, node.ID)
		}
	}

	logging.Sugar.Debugf("node to delete: %v", nodeToDelete)
	if len(nodeToDelete) > 0 {
		err := w.TaskService.DeleteTasks(tx, nodeToDelete)
		return err

	}
	return nil
}

// delete edges that doesnt exist in the body payload
func (w *WorkflowApplicationServiceImpl) DeleteEdges(
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

func validateWorkflowTaskPayload(body tasks.UpdateWorkflowtasks) error {
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

func (w *WorkflowApplicationServiceImpl) UpdateWorkflowTasks(
	workflowId string,
	body tasks.UpdateWorkflowtasks,
) (*workflows.WorkflowsGraph, error) {
	tx, err := w.DB.Beginx()
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if body.Task != nil {
		_, errTask := w.WorkflowService.UpdateWorkflowTx(tx, workflowId, *body.Task)

		logging.Sugar.Debug("updated workflow...")

		if errTask != nil {
			return nil, errTask
		}
	}

	// validate if start node in body payload
	payloadErr := validateWorkflowTaskPayload(body)

	if payloadErr != nil {
		return nil, payloadErr
	}

	workflowUUID, err := uuid.Parse(workflowId)

	if err != nil {
		return nil, err
	}

	// delete the edges first
	deleteEdgesErr := w.DeleteEdges(tx, workflowUUID, body.Edges)
	if deleteEdgesErr != nil {
		logging.Sugar.Error(deleteEdgesErr)
		tx.Rollback()
		return nil, deleteEdgesErr
	}

	// upsert the tasks. insert if doesnt exist, update when exist
	insertedTasks, upsertTasksErr := w.UpsertTasks(tx, workflowUUID, body.Nodes)
	if upsertTasksErr != nil {
		logging.Sugar.Error(upsertTasksErr)
		tx.Rollback()
		return nil, upsertTasksErr
	}

	// delete the tasks the we dont need anymore
	deleteTaskError := w.DeleteTasks(tx, workflowUUID, body.Nodes)
	if deleteTaskError != nil {
		logging.Sugar.Error(deleteTaskError)
		tx.Rollback()
		return nil, deleteTaskError
	}

	// insert the new edges
	logging.Sugar.Info("insert edges")
	insertEdgeError := w.InsertEdges(tx, workflowUUID, body.Edges, insertedTasks, body.Handles)
	if insertEdgeError != nil {
		logging.Sugar.Error(insertEdgeError)
		tx.Rollback()
		return nil, insertEdgeError
	}

	logging.Sugar.Debug("added workflow...")
	commitErr := tx.Commit()

	if commitErr != nil {
		logging.Sugar.Error(commitErr)
		tx.Rollback()
		return nil, commitErr
	}

	workflowGraph, workflowErr := w.WorkflowService.GetWorkflowGraphById(workflowId)

	if workflowErr != nil {
		logging.Sugar.Error(workflowErr)
		return nil, workflowErr
	}

	return workflowGraph, nil

}

// TriggerWorkflow implements WorkflowApplicationService.
func (w *WorkflowApplicationServiceImpl) TriggerWorkflow(workflowId string) (*TaskMessage, error) {
	_, workflowErr := w.WorkflowService.GetWorkflowById(workflowId)

	if workflowErr != nil {
		logging.Sugar.Error(workflowErr)
		return nil, workflowErr
	}
	taskData, tasksErr := w.TaskService.GetTasksByWorkflowId(workflowId)
	if tasksErr != nil {
		logging.Sugar.Errorf("error: ", tasksErr)
		return nil, tasksErr
	}

	edges, edgesErr := w.EdgeService.GetEdgesByWorkflowId(workflowId)

	if edgesErr != nil {
		logging.Sugar.Errorf("error: ", edgesErr)
		return nil, edgesErr
	}

	// workflowEdges := []models.Edges{}
	// for _, edge := range edges {
	// 	workflowEdges = append(workflowEdges, models.Edges{
	// 		ID:                edge.ID,
	// 		WorkflowID:        edge.WorkflowID,
	// 		DestinationID:     edge.DestinationID,
	// 		SourceID:          edge.SourceID,
	// 		SourceHandle:      edge.SourceHandle,
	// 		DestinationHandle: edge.DestinationHandle,
	// 	})
	// }
	tasksMap, graph := w.PrepareWorkflowMessage(taskData, edges)

	// create transacton

	tx, txErr := w.DB.Beginx()
	if txErr != nil {
		tx.Rollback()
		return nil, txErr
	}

	workflowHistory, workflowHistoryErr := w.WorkflowService.CreateWorkflowHistory(tx, workflowId, edges)
	if workflowHistoryErr != nil {
		tx.Rollback()
		return nil, workflowHistoryErr
	}

	// Log the ID to verify it's correct
	logging.Sugar.Infof("Created workflow history with ID: %v", workflowHistory.ID)
	_, createTaskHistoryErr := w.TaskService.CreateTaskHistory(tx, workflowHistory.ID.String(), taskData, GetGraphUUIDS(edges))

	if createTaskHistoryErr != nil {
		tx.Rollback()
		return nil, createTaskHistoryErr
	}

	commitErr := tx.Commit()

	if commitErr != nil {
		logging.Sugar.Error(commitErr)
		tx.Rollback()
		return nil, commitErr

	}

	body := TaskMessage{
		Graph:             graph,
		Tasks:             tasksMap,
		WorkflowHistoryId: workflowHistory.ID,
	}

	mqErr := w.TaskPubSUb.SendMessage(body)

	if mqErr != nil {
		logging.Sugar.Errorf("error when sending the message to queue", mqErr)
		return nil, mqErr
	}
	return &body, nil
}

// PrepareWorkflowMessage implements WorkflowTriggerService.
func (w *WorkflowApplicationServiceImpl) PrepareWorkflowMessage(tasksData []tasks.Tasks, edges []edges.ResponseEdges) (map[string]tasks.Tasks, map[string][]string) {
	tasksMap := make(map[string]tasks.Tasks)
	graph := map[string][]string{}

	for _, task := range tasksData {
		tasksMap[task.Name] = task
	}

	for _, edge := range edges {
		children, ok := graph[edge.SourceTaskName]
		if ok {
			graph[edge.SourceTaskName] = append(children, edge.DestinationTaskName)
		} else {
			graph[edge.SourceTaskName] = []string{edge.DestinationTaskName}
		}

		_, taskNameOk := graph[edge.DestinationTaskName]

		if !taskNameOk {
			graph[edge.DestinationTaskName] = []string{}
		}
	}

	return tasksMap, graph
}

func GetGraphUUIDS(edges []edges.ResponseEdges) map[uuid.UUID][]uuid.UUID {
	graph := map[uuid.UUID][]uuid.UUID{}

	for _, edge := range edges {
		children, ok := graph[edge.SourceID]
		if ok {
			graph[edge.SourceID] = append(children, edge.DestinationID)
		} else {
			graph[edge.SourceID] = []uuid.UUID{edge.DestinationID}
		}

		_, taskNameOk := graph[edge.DestinationID]

		if !taskNameOk {
			graph[edge.DestinationID] = []uuid.UUID{}
		}
	}

	return graph

}
