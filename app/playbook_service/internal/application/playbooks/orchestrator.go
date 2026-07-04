package playbooks

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14/ytsoar/internal/application/contracts"
	"github.com/yuudev14/ytsoar/internal/application/edges"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logging"
)

//go:generate mockgen -destination=mocks/orchestrator_mock.go -package=mocks . PlaybookApplicationService

type PlaybookApplicationService interface {
	TriggerPlaybook(playbookId string) (*domain.TaskMessage, error)
	PreparePlaybookMessage(tasks []domain.Tasks, edges []domain.ResponseEdges) (map[string]domain.Tasks, map[string][]string)
	UpsertTasks(
		tx *sqlx.Tx,
		playbookUUID uuid.UUID,
		nodes []tasks.TaskPayload,
	) ([]domain.Tasks, error)
	InsertEdges(
		tx *sqlx.Tx,
		playbookUUID uuid.UUID,
		edges map[string][]string,
		tasks []domain.Tasks,
		handles *map[string]map[string]domain.EdgeHandle,
	) error
	DeleteTasks(
		tx *sqlx.Tx,
		playbookUUID uuid.UUID,
		nodes []tasks.TaskPayload,
	) error
	DeleteEdges(
		tx *sqlx.Tx,
		playbookUUID uuid.UUID,
		edges map[string][]string,
	) error
	UpdatePlaybookTasks(playbookId string, body UpdatePlaybookTasksPayload) (*domain.PlaybookGraph, error)
}

type PlaybookApplicationServiceImpl struct {
	PlaybookService   PlaybookService
	TaskService       tasks.TaskService
	EdgeService       edges.EdgeService
	DB                *sqlx.DB
	TaskPublisher     contracts.TaskPublisher
	StatusBroadcaster contracts.StatusBroadcaster
}

func NewPlaybookApplicationService(
	playbookService PlaybookService,
	taskService tasks.TaskService,
	edgeService edges.EdgeService,
	db *sqlx.DB,
	taskPublisher contracts.TaskPublisher,
	statusBroadcaster contracts.StatusBroadcaster,
) PlaybookApplicationService {
	return &PlaybookApplicationServiceImpl{
		PlaybookService:   playbookService,
		TaskService:       taskService,
		EdgeService:       edgeService,
		DB:                db,
		TaskPublisher:     taskPublisher,
		StatusBroadcaster: statusBroadcaster,
	}
}

func (w *PlaybookApplicationServiceImpl) UpsertTasks(
	tx *sqlx.Tx,
	playbookUUID uuid.UUID,
	nodes []tasks.TaskPayload,
) ([]domain.Tasks, error) {
	// node to update
	var nodeToUpsert []domain.Tasks
	for _, node := range nodes {
		nodeToUpsert = append(nodeToUpsert, domain.Tasks{
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
		return w.TaskService.UpsertTasks(tx, playbookUUID, nodeToUpsert)
	}
	return nil, nil
}

func (w *PlaybookApplicationServiceImpl) InsertEdges(
	tx *sqlx.Tx,
	playbookUUID uuid.UUID,
	edges_ map[string][]string,
	tasks []domain.Tasks,
	handles *map[string]map[string]domain.EdgeHandle,
) error {
	// node to update
	var edgeToInsert []domain.Edges
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
				edge := domain.Edges{
					SourceID:      sourceId,
					DestinationID: destinationID,
					PlaybookID:    playbookUUID,
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

func (w *PlaybookApplicationServiceImpl) DeleteTasks(
	tx *sqlx.Tx,
	playbookUUID uuid.UUID,
	nodes []tasks.TaskPayload,
) error {
	// node to delete
	var nodeToDelete []uuid.UUID
	tasksBodyMap := make(map[string]bool)

	// verify nodes name should be unique
	tasks, tasksErr := w.TaskService.GetTasksByPlaybookId(playbookUUID.String())
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
func (w *PlaybookApplicationServiceImpl) DeleteEdges(
	tx *sqlx.Tx,
	playbookUUID uuid.UUID,
	edges map[string][]string,
) error {

	var edgeToDelete []uuid.UUID
	edgesMap := make(map[[2]string]bool)

	// delete all edges from the playbook if nothing is in the payload
	if len(edges) == 0 {
		return w.EdgeService.DeleteAllPlaybookEdges(tx, playbookUUID.String())
	}

	playbookEdges, playbookEdgesErr := w.EdgeService.GetEdgesByPlaybookId(playbookUUID.String())
	logging.Sugar.Debug("playbook edges", playbookEdges)

	if playbookEdgesErr != nil {
		logging.Sugar.Error(playbookEdgesErr)
		return playbookEdgesErr
	}

	// populate the hashmap
	for key, values := range edges {
		for _, val := range values {
			edgesMap[[2]string{key, val}] = true
		}
	}

	// if the edge does not exist in the hashmap, add to the delete lists
	for _, edge := range playbookEdges {
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

func validatePlaybookTaskPayload(body UpdatePlaybookTasksPayload) error {
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

func (w *PlaybookApplicationServiceImpl) UpdatePlaybookTasks(
	playbookId string,
	body UpdatePlaybookTasksPayload,
) (*domain.PlaybookGraph, error) {
	tx, err := w.DB.Beginx()
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if body.Task != nil {
		_, errTask := w.PlaybookService.UpdatePlaybookTx(tx, playbookId, *body.Task)

		logging.Sugar.Debug("updated playbook...")

		if errTask != nil {
			return nil, errTask
		}
	}

	// validate if start node in body payload
	payloadErr := validatePlaybookTaskPayload(body)

	if payloadErr != nil {
		return nil, payloadErr
	}

	playbookUUID, err := uuid.Parse(playbookId)

	if err != nil {
		return nil, err
	}

	// delete the edges first
	deleteEdgesErr := w.DeleteEdges(tx, playbookUUID, body.Edges)
	if deleteEdgesErr != nil {
		logging.Sugar.Error(deleteEdgesErr)
		tx.Rollback()
		return nil, deleteEdgesErr
	}

	// upsert the tasks. insert if doesnt exist, update when exist
	insertedTasks, upsertTasksErr := w.UpsertTasks(tx, playbookUUID, body.Nodes)
	if upsertTasksErr != nil {
		logging.Sugar.Error(upsertTasksErr)
		tx.Rollback()
		return nil, upsertTasksErr
	}

	// delete the tasks the we dont need anymore
	deleteTaskError := w.DeleteTasks(tx, playbookUUID, body.Nodes)
	if deleteTaskError != nil {
		logging.Sugar.Error(deleteTaskError)
		tx.Rollback()
		return nil, deleteTaskError
	}

	// insert the new edges
	logging.Sugar.Info("insert edges")
	insertEdgeError := w.InsertEdges(tx, playbookUUID, body.Edges, insertedTasks, body.Handles)
	if insertEdgeError != nil {
		logging.Sugar.Error(insertEdgeError)
		tx.Rollback()
		return nil, insertEdgeError
	}

	logging.Sugar.Debug("added playbook...")
	commitErr := tx.Commit()

	if commitErr != nil {
		logging.Sugar.Error(commitErr)
		tx.Rollback()
		return nil, commitErr
	}

	playbookGraph, playbookErr := w.PlaybookService.GetPlaybookGraphById(playbookId)

	if playbookErr != nil {
		logging.Sugar.Error(playbookErr)
		return nil, playbookErr
	}

	return playbookGraph, nil

}

// TriggerPlaybook implements PlaybookApplicationService.
func (w *PlaybookApplicationServiceImpl) TriggerPlaybook(playbookId string) (*domain.TaskMessage, error) {
	_, playbookErr := w.PlaybookService.GetPlaybookById(playbookId)

	if playbookErr != nil {
		logging.Sugar.Error(playbookErr)
		return nil, playbookErr
	}
	taskData, tasksErr := w.TaskService.GetTasksByPlaybookId(playbookId)
	if tasksErr != nil {
		logging.Sugar.Errorf("error: ", tasksErr)
		return nil, tasksErr
	}

	edges, edgesErr := w.EdgeService.GetEdgesByPlaybookId(playbookId)

	if edgesErr != nil {
		logging.Sugar.Errorf("error: ", edgesErr)
		return nil, edgesErr
	}

	tasksMap, graph := w.PreparePlaybookMessage(taskData, edges)

	// create transacton

	tx, txErr := w.DB.Beginx()
	if txErr != nil {
		tx.Rollback()
		return nil, txErr
	}

	playbookHistory, playbookHistoryErr := w.PlaybookService.CreatePlaybookHistory(tx, playbookId, edges)
	if playbookHistoryErr != nil {
		tx.Rollback()
		return nil, playbookHistoryErr
	}
	w.StatusBroadcaster.Broadcast(playbookHistory)

	// Log the ID to verify it's correct
	logging.Sugar.Infof("Created playbook history with ID: %v", playbookHistory.ID)
	_, createTaskHistoryErr := w.TaskService.CreateTaskHistory(tx, playbookHistory.ID.String(), taskData, GetGraphUUIDS(edges))

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

	body := domain.TaskMessage{
		Graph:             graph,
		Tasks:             tasksMap,
		PlaybookHistoryId: playbookHistory.ID,
	}

	mqErr := w.TaskPublisher.SendMessage(body)

	if mqErr != nil {
		logging.Sugar.Errorf("error when sending the message to queue", mqErr)
		return nil, mqErr
	}
	return &body, nil
}

// PreparePlaybookMessage implements PlaybookApplicationService.
func (w *PlaybookApplicationServiceImpl) PreparePlaybookMessage(tasksData []domain.Tasks, edges []domain.ResponseEdges) (map[string]domain.Tasks, map[string][]string) {
	tasksMap := make(map[string]domain.Tasks)
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

func GetGraphUUIDS(edges []domain.ResponseEdges) map[uuid.UUID][]uuid.UUID {
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
