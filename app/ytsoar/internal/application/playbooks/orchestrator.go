package playbooks

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/application/contracts"
	"github.com/yuudev14/ytsoar/internal/application/edges"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

//go:generate mockgen -destination=mocks/orchestrator_mock.go -package=mocks . PlaybookApplicationService

type PlaybookApplicationService interface {
	TriggerPlaybook(ctx context.Context, playbookId string) (*domain.TaskMessage, error)
	PreparePlaybookMessage(tasks []domain.Tasks, edges []domain.ResponseEdges) (map[string]domain.Tasks, map[string][]string, []domain.EdgeRef)
	UpsertTasks(ctx context.Context, playbookUUID uuid.UUID, nodes []tasks.TaskPayload) ([]domain.Tasks, error)
	InsertEdges(ctx context.Context, playbookUUID uuid.UUID, edges map[string][]string, tasks []domain.Tasks, handles map[string]map[string]domain.EdgeHandle) error
	DeleteTasks(ctx context.Context, playbookUUID uuid.UUID, nodes []tasks.TaskPayload) error
	DeleteEdges(ctx context.Context, playbookUUID uuid.UUID, edges map[string][]string) error
	UpdatePlaybookTasks(ctx context.Context, playbookId string, body UpdatePlaybookTasksPayload) (*domain.PlaybookGraph, error)
}

type PlaybookApplicationServiceImpl struct {
	Logger            logger.Logger
	PlaybookService   PlaybookService
	TaskService       tasks.TaskService
	EdgeService       edges.EdgeService
	Tx                contracts.TxManager
	TaskPublisher     contracts.TaskPublisher
	StatusBroadcaster contracts.StatusBroadcaster
}

func NewPlaybookApplicationService(
	log logger.Logger,
	playbookService PlaybookService,
	taskService tasks.TaskService,
	edgeService edges.EdgeService,
	tx contracts.TxManager,
	taskPublisher contracts.TaskPublisher,
	statusBroadcaster contracts.StatusBroadcaster,
) PlaybookApplicationService {
	return &PlaybookApplicationServiceImpl{
		Logger:            log,
		PlaybookService:   playbookService,
		TaskService:       taskService,
		EdgeService:       edgeService,
		Tx:                tx,
		TaskPublisher:     taskPublisher,
		StatusBroadcaster: statusBroadcaster,
	}
}

func (w *PlaybookApplicationServiceImpl) UpsertTasks(
	ctx context.Context,
	playbookUUID uuid.UUID,
	nodes []tasks.TaskPayload,
) ([]domain.Tasks, error) {
	// node to update
	var nodeToUpsert []domain.Tasks
	for _, node := range nodes {
		var parameters json.RawMessage
		if node.Parameters != nil {
			b, err := json.Marshal(node.Parameters)
			if err != nil {
				return nil, fmt.Errorf("could not encode parameters for task %q: %w", node.Name, err)
			}
			parameters = b
		}
		nodeToUpsert = append(nodeToUpsert, domain.Tasks{
			Name:          node.Name,
			Parameters:    parameters,
			Description:   node.Description,
			Config:        node.Config.Value,
			ConnectorName: node.ConnectorName,
			ConnectorID:   node.ConnectorID,
			Operation:     node.Operation,
			X:             node.X,
			Y:             node.Y,
		})
	}

	w.Logger.Debugf("node to add: %v", nodeToUpsert)
	// save the tasks
	if len(nodeToUpsert) > 0 {
		return w.TaskService.UpsertTasks(ctx, playbookUUID, nodeToUpsert)
	}
	return nil, nil
}

func (w *PlaybookApplicationServiceImpl) InsertEdges(
	ctx context.Context,
	playbookUUID uuid.UUID,
	edges_ map[string][]string,
	tasks []domain.Tasks,
	handles map[string]map[string]domain.EdgeHandle,
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

	w.Logger.Debugf("tasksMap: %v", tasksMap)
	w.Logger.Debugf("edges: %v", edges_)

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
				if edgeHandle, ok := handles[key][val]; ok {
					if edgeHandle.SourceHandle != nil {
						edge.SourceHandle = sql.NullString{String: *edgeHandle.SourceHandle, Valid: true}
					}
					if edgeHandle.DestinationHandle != nil {
						edge.DestinationHandle = sql.NullString{String: *edgeHandle.DestinationHandle, Valid: true}
					}
				}
				edgeToInsert = append(edgeToInsert, edge)
			} else {
				w.Logger.Infof("edges data that are not added: %v %v", key, val)
			}
		}
	}

	w.Logger.Debugf("edges to add: %v", edgeToInsert)
	// save the edges
	if len(edgeToInsert) > 0 {
		_, err := w.EdgeService.InsertEdges(ctx, edgeToInsert)
		return err
	}
	return nil
}

func (w *PlaybookApplicationServiceImpl) DeleteTasks(
	ctx context.Context,
	playbookUUID uuid.UUID,
	nodes []tasks.TaskPayload,
) error {
	// node to delete
	var nodeToDelete []uuid.UUID
	tasksBodyMap := make(map[string]bool)

	// verify nodes name should be unique
	tasks, tasksErr := w.TaskService.GetTasksByPlaybookId(ctx, playbookUUID.String())
	if tasksErr != nil {
		return tasksErr
	}
	w.Logger.Debugf("tasks: %v", tasks)

	for _, node := range nodes {
		tasksBodyMap[node.Name] = true
	}
	w.Logger.Debugf("tasksBodyMap: %v", tasksBodyMap)
	// 2. if node not in new nodes to be updated, delete
	for _, node := range tasks {
		_, ok := tasksBodyMap[node.Name]
		w.Logger.Debugf("checking if node to be deleted for: %v", node.Name)
		if !ok {
			nodeToDelete = append(nodeToDelete, node.ID)
		}
	}

	w.Logger.Debugf("node to delete: %v", nodeToDelete)
	if len(nodeToDelete) > 0 {
		return w.TaskService.DeleteTasks(ctx, nodeToDelete)
	}
	return nil
}

// delete edges that doesnt exist in the body payload
func (w *PlaybookApplicationServiceImpl) DeleteEdges(
	ctx context.Context,
	playbookUUID uuid.UUID,
	edges map[string][]string,
) error {

	var edgeToDelete []uuid.UUID
	edgesMap := make(map[[2]string]bool)

	// delete all edges from the playbook if nothing is in the payload
	if len(edges) == 0 {
		return w.EdgeService.DeleteAllPlaybookEdges(ctx, playbookUUID.String())
	}

	playbookEdges, playbookEdgesErr := w.EdgeService.GetEdgesByPlaybookId(ctx, playbookUUID.String())
	w.Logger.Debug("playbook edges", playbookEdges)

	if playbookEdgesErr != nil {
		w.Logger.Error(playbookEdgesErr)
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

	w.Logger.Debugf("edge to delete: %v", edgeToDelete)
	if len(edgeToDelete) > 0 {
		return w.EdgeService.DeleteEdges(ctx, edgeToDelete)
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
	ctx context.Context,
	playbookId string,
	body UpdatePlaybookTasksPayload,
) (*domain.PlaybookGraph, error) {
	// validate if start node in body payload
	if payloadErr := validatePlaybookTaskPayload(body); payloadErr != nil {
		return nil, payloadErr
	}

	playbookUUID, err := uuid.Parse(playbookId)
	if err != nil {
		return nil, err
	}

	txErr := w.Tx.WithinTransaction(ctx, func(ctx context.Context) error {
		if body.Task != nil {
			if _, errTask := w.PlaybookService.UpdatePlaybook(ctx, playbookId, *body.Task); errTask != nil {
				return errTask
			}
			w.Logger.Debug("updated playbook...")
		}

		// delete the edges first
		if deleteEdgesErr := w.DeleteEdges(ctx, playbookUUID, body.Edges); deleteEdgesErr != nil {
			w.Logger.Error(deleteEdgesErr)
			return deleteEdgesErr
		}

		// upsert the tasks. insert if doesnt exist, update when exist
		insertedTasks, upsertTasksErr := w.UpsertTasks(ctx, playbookUUID, body.Nodes)
		if upsertTasksErr != nil {
			w.Logger.Error(upsertTasksErr)
			return upsertTasksErr
		}

		// delete the tasks the we dont need anymore
		if deleteTaskError := w.DeleteTasks(ctx, playbookUUID, body.Nodes); deleteTaskError != nil {
			w.Logger.Error(deleteTaskError)
			return deleteTaskError
		}

		// insert the new edges
		w.Logger.Info("insert edges")
		if insertEdgeError := w.InsertEdges(ctx, playbookUUID, body.Edges, insertedTasks, body.Handles); insertEdgeError != nil {
			w.Logger.Error(insertEdgeError)
			return insertEdgeError
		}

		w.Logger.Debug("added playbook...")
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	playbookGraph, playbookErr := w.PlaybookService.GetPlaybookGraphById(ctx, playbookId)
	if playbookErr != nil {
		w.Logger.Error(playbookErr)
		return nil, playbookErr
	}

	return playbookGraph, nil
}

// TriggerPlaybook implements PlaybookApplicationService.
func (w *PlaybookApplicationServiceImpl) TriggerPlaybook(ctx context.Context, playbookId string) (*domain.TaskMessage, error) {
	_, playbookErr := w.PlaybookService.GetPlaybookById(ctx, playbookId)
	if playbookErr != nil {
		w.Logger.Error(playbookErr)
		return nil, playbookErr
	}

	taskData, tasksErr := w.TaskService.GetTasksByPlaybookId(ctx, playbookId)
	if tasksErr != nil {
		w.Logger.Errorf("error: %v", tasksErr)
		return nil, tasksErr
	}

	edges, edgesErr := w.EdgeService.GetEdgesByPlaybookId(ctx, playbookId)
	if edgesErr != nil {
		w.Logger.Errorf("error: %v", edgesErr)
		return nil, edgesErr
	}

	tasksMap, graph, edgeRefs := w.PreparePlaybookMessage(taskData, edges)

	var playbookHistory *domain.PlaybookHistory
	txErr := w.Tx.WithinTransaction(ctx, func(ctx context.Context) error {
		history, historyErr := w.PlaybookService.CreatePlaybookHistory(ctx, playbookId, edges)
		if historyErr != nil {
			return historyErr
		}
		playbookHistory = history

		w.Logger.Infof("Created playbook history with ID: %v", playbookHistory.ID)
		_, createTaskHistoryErr := w.TaskService.CreateTaskHistory(ctx, playbookHistory.ID.String(), taskData, GetGraphUUIDS(edges))
		return createTaskHistoryErr
	})
	if txErr != nil {
		return nil, txErr
	}

	// broadcast only after the tx commits — a rollback must not leak a
	// phantom history to WS clients, and a slow client must not hold the tx open
	w.StatusBroadcaster.Broadcast(playbookHistory)

	body := domain.TaskMessage{
		Graph:             graph,
		Tasks:             tasksMap,
		PlaybookHistoryId: playbookHistory.ID,
		Edges:             edgeRefs,
	}

	if mqErr := w.TaskPublisher.SendMessage(body); mqErr != nil {
		w.Logger.Errorf("error when sending the message to queue: %v", mqErr)
		return nil, mqErr
	}
	return &body, nil
}

// PreparePlaybookMessage implements PlaybookApplicationService.
func (w *PlaybookApplicationServiceImpl) PreparePlaybookMessage(tasksData []domain.Tasks, edges []domain.ResponseEdges) (map[string]domain.Tasks, map[string][]string, []domain.EdgeRef) {
	tasksMap := make(map[string]domain.Tasks)
	graph := map[string][]string{}
	edgeRefs := make([]domain.EdgeRef, 0, len(edges))

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

		ref := domain.EdgeRef{
			Source:      edge.SourceTaskName,
			Destination: edge.DestinationTaskName,
		}
		if edge.SourceHandle.Valid {
			handle := edge.SourceHandle.String
			ref.SourceHandle = &handle
		}
		edgeRefs = append(edgeRefs, ref)
	}

	return tasksMap, graph, edgeRefs
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
