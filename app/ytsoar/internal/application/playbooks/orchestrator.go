package playbooks

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/application/contracts"
	"github.com/yuudev14/ytsoar/internal/application/edges"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
	"github.com/yuudev14/ytsoar/internal/logger"
)

//go:generate mockgen -destination=mocks/orchestrator_mock.go -package=mocks . PlaybookApplicationService

type PlaybookApplicationService interface {
	TriggerPlaybook(ctx context.Context, playbookId string) (*domain.TaskMessage, error)
	RunPlaybook(ctx context.Context, playbookId string, moduleType string, payload RunPlaybookPayload, actorID *uuid.UUID) (*domain.TaskMessage, error)
	RunForEvent(ctx context.Context, moduleType string, event string, recordID uuid.UUID, occurredAt time.Time) error
	WebhookAuth(ctx context.Context, playbookId string) (domain.WebhookAuth, error)
	RunWebhook(ctx context.Context, playbookId string, parameters map[string]any, actorID *uuid.UUID) (uuid.UUID, error)
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
	RunEnqueuer       contracts.RunEnqueuer
	StatusBroadcaster contracts.StatusBroadcaster
	Records           RecordResolver
}

func NewPlaybookApplicationService(
	log logger.Logger,
	playbookService PlaybookService,
	taskService tasks.TaskService,
	edgeService edges.EdgeService,
	tx contracts.TxManager,
	runEnqueuer contracts.RunEnqueuer,
	statusBroadcaster contracts.StatusBroadcaster,
	records RecordResolver,
) PlaybookApplicationService {
	return &PlaybookApplicationServiceImpl{
		Logger:            log,
		PlaybookService:   playbookService,
		TaskService:       taskService,
		EdgeService:       edgeService,
		Tx:                tx,
		RunEnqueuer:       runEnqueuer,
		StatusBroadcaster: statusBroadcaster,
		Records:           records,
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

	// the payload has task names, but we save ids. Build a name -> uuid map to get the id
	// from the edges
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
	for _, node := range body.Nodes {
		if node.Name == "start" {
			return nil
		}
	}

	return apperr.New(apperr.Invalid, "the playbook must have a start step")
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
		// tasks and edges have no deleted_at, so a deleted playbook must fail here, before they are written
		if body.Task != nil {
			if _, errTask := w.PlaybookService.UpdatePlaybook(ctx, playbookId, *body.Task); errTask != nil {
				return errTask
			}
			w.Logger.Debug("updated playbook...")
		} else if _, err := w.PlaybookService.GetPlaybookById(ctx, playbookId); err != nil {
			return err
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

// TriggerPlaybook implements PlaybookApplicationService. The editor trigger has no
// records and no parameters.
func (w *PlaybookApplicationServiceImpl) TriggerPlaybook(ctx context.Context, playbookId string) (*domain.TaskMessage, error) {
	playbook, err := w.PlaybookService.GetPlaybookById(ctx, playbookId)
	if err != nil {
		return nil, err
	}
	taskList, err := w.TaskService.GetTasksByPlaybookId(ctx, playbookId)
	if err != nil {
		return nil, err
	}
	if !slices.ContainsFunc(taskList, func(t domain.Tasks) bool { return t.Name == "start" }) {
		return nil, apperr.New(apperr.Invalid, "the playbook must have a start step")
	}
	if playbook.TriggerType == nil {
		return nil, apperr.New(apperr.Invalid, "the playbook must have a trigger type")
	}
	return w.runPlaybook(ctx, playbookId, RunStamp{
		TriggerType: new(string(domain.TriggerTypeManual)),
		Input:       domain.NewEmptyRunInput(),
	}, "", nil)
}

// RunPlaybook implements PlaybookApplicationService. The caller sends record ids and we
// load the records here, so a fake payload cannot target a record the analyst did not pick.
func (w *PlaybookApplicationServiceImpl) RunPlaybook(ctx context.Context, playbookId string, moduleType string, payload RunPlaybookPayload, actorID *uuid.UUID) (*domain.TaskMessage, error) {
	recordIDs, err := parseRecordIDs(payload.RecordIDs)
	if err != nil {
		return nil, err
	}

	records, err := w.Records.Resolve(ctx, moduleType, recordIDs)
	if err != nil {
		return nil, err
	}

	parameters := payload.Parameters
	if parameters == nil {
		parameters = map[string]any{}
	}

	return w.runPlaybook(ctx, playbookId, RunStamp{
		TriggerType: new(string(domain.TriggerTypeManual)),
		TriggeredBy: actorID,
		Input: &domain.RunInput{
			ModuleType: &moduleType,
			Records:    records,
			Parameters: parameters,
		},
	}, moduleType, recordIDs)
}

// WebhookAuth implements PlaybookApplicationService. A playbook whose trigger is not
// webhook is not found, so a caller cannot tell it from a playbook that does not exist.
func (w *PlaybookApplicationServiceImpl) WebhookAuth(ctx context.Context, playbookId string) (domain.WebhookAuth, error) {
	playbook, err := w.PlaybookService.GetPlaybookById(ctx, playbookId)
	if err != nil {
		return "", err
	}
	if playbook.TriggerType == nil || *playbook.TriggerType != string(domain.TriggerTypeWebhook) {
		return "", ErrPlaybookNotFound
	}
	return domain.WebhookAuthOf(playbook.TriggerParameters)
}

// RunWebhook implements PlaybookApplicationService. The caller is checked by WebhookAuth
// first. The request body is the run parameters.
func (w *PlaybookApplicationServiceImpl) RunWebhook(ctx context.Context, playbookId string, parameters map[string]any, actorID *uuid.UUID) (uuid.UUID, error) {
	input := domain.NewEmptyRunInput()
	if parameters != nil {
		input.Parameters = parameters
	}
	msg, err := w.runPlaybook(ctx, playbookId, RunStamp{
		TriggerType: new(string(domain.TriggerTypeWebhook)),
		TriggeredBy: actorID,
		Input:       input,
	}, "", nil)
	if err != nil {
		return uuid.Nil, err
	}
	return msg.PlaybookHistoryId, nil
}

var eventTriggers = map[string]domain.TriggerType{
	domain.ModuleEventCreated: domain.TriggerTypeOnCreate,
	domain.ModuleEventUpdated: domain.TriggerTypeOnUpdate,
}

// RunForEvent implements PlaybookApplicationService. It queues one run for each playbook
// that the event triggers. All runs share one transaction, so a retried job does not
// queue them twice.
func (w *PlaybookApplicationServiceImpl) RunForEvent(ctx context.Context, moduleType string, event string, recordID uuid.UUID, occurredAt time.Time) error {
	triggerType, ok := eventTriggers[event]
	if !ok {
		return nil
	}

	return w.Tx.WithinTransaction(ctx, func(ctx context.Context) error {
		playbookIDs, err := w.PlaybookService.ListEventPlaybookIDs(ctx, triggerType, moduleType, recordID, occurredAt)
		if err != nil || len(playbookIDs) == 0 {
			return err
		}

		recordIDs := []uuid.UUID{recordID}
		records, err := w.Records.Resolve(ctx, moduleType, recordIDs)
		if err != nil {
			return err
		}

		for _, playbookID := range playbookIDs {
			if _, _, err := w.queueRun(ctx, playbookID.String(), RunStamp{
				TriggerType: new(string(triggerType)),
				Input: &domain.RunInput{
					ModuleType: &moduleType,
					Records:    records,
					Parameters: map[string]any{},
				},
			}, moduleType, recordIDs); err != nil {
				return err
			}
		}
		return nil
	})
}

func parseRecordIDs(raw []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(raw))
	for _, s := range raw {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, apperr.New(apperr.Invalid, "record_ids must be uuids")
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (w *PlaybookApplicationServiceImpl) runPlaybook(ctx context.Context, playbookId string, run RunStamp, moduleType string, recordIDs []uuid.UUID) (*domain.TaskMessage, error) {
	var (
		msg     *domain.TaskMessage
		history *domain.PlaybookHistory
	)
	txErr := w.Tx.WithinTransaction(ctx, func(ctx context.Context) error {
		var err error
		msg, history, err = w.queueRun(ctx, playbookId, run, moduleType, recordIDs)
		return err
	})
	if txErr != nil {
		return nil, txErr
	}

	// broadcast after the commit: a rollback must not leak a phantom history, and a slow
	// client must not hold the tx open. Use the same {event, data} envelope as the status
	// consumer, or clients drop the row.
	w.StatusBroadcaster.Broadcast(map[string]any{
		"event": "playbook_status",
		"data":  history,
	})

	return msg, nil
}

// queueRun must run in a transaction: the run is queued only if its history exists.
func (w *PlaybookApplicationServiceImpl) queueRun(ctx context.Context, playbookId string, run RunStamp, moduleType string, recordIDs []uuid.UUID) (*domain.TaskMessage, *domain.PlaybookHistory, error) {
	_, playbookErr := w.PlaybookService.GetPlaybookById(ctx, playbookId)
	if playbookErr != nil {
		return nil, nil, playbookErr
	}

	taskData, tasksErr := w.TaskService.GetTasksByPlaybookId(ctx, playbookId)
	if tasksErr != nil {
		return nil, nil, tasksErr
	}

	edges, edgesErr := w.EdgeService.GetEdgesByPlaybookId(ctx, playbookId)
	if edgesErr != nil {
		return nil, nil, edgesErr
	}

	tasksMap, graph, edgeRefs := w.PreparePlaybookMessage(taskData, edges)

	playbookHistory, historyErr := w.PlaybookService.CreatePlaybookHistory(ctx, playbookId, edges, run)
	if historyErr != nil {
		return nil, nil, historyErr
	}

	if len(recordIDs) > 0 {
		if err := w.PlaybookService.CreatePlaybookRunRecords(ctx, playbookHistory.ID, moduleType, recordIDs); err != nil {
			return nil, nil, err
		}
	}

	w.Logger.Infof("Created playbook history with ID: %v", playbookHistory.ID)
	if _, err := w.TaskService.CreateTaskHistory(ctx, playbookHistory.ID.String(), taskData, GetGraphUUIDS(edges)); err != nil {
		return nil, nil, err
	}
	if err := w.RunEnqueuer.EnqueueRun(ctx, playbookHistory.ID); err != nil {
		return nil, nil, err
	}

	return &domain.TaskMessage{
		Graph:             graph,
		Tasks:             tasksMap,
		PlaybookHistoryId: playbookHistory.ID,
		Edges:             edgeRefs,
		Input:             run.Input.Resolved(),
	}, playbookHistory, nil
}

// RunLoader builds the TaskMessage of a queued run from its history rows (edges and input
// in playbook_history, task snapshots in task_history). The queue holds only the run ID,
// not a copy of the graph.
type RunLoader struct {
	playbooks PlaybookService
	tasks     tasks.TaskService
}

func NewRunLoader(playbookService PlaybookService, taskService tasks.TaskService) *RunLoader {
	return &RunLoader{playbooks: playbookService, tasks: taskService}
}

func (l *RunLoader) LoadRunMessage(ctx context.Context, historyID uuid.UUID) (domain.TaskMessage, error) {
	history, err := l.playbooks.GetPlaybookHistoryById(ctx, historyID)
	if err != nil {
		return domain.TaskMessage{}, err
	}
	if history == nil { // the repository returns nil when the row is gone
		return domain.TaskMessage{}, fmt.Errorf("run %s does not exist", historyID)
	}
	taskRows, err := l.tasks.GetTaskHistoryByPlaybookHistoryId(ctx, historyID.String(), tasks.TaskHistoryFilter{})
	if err != nil {
		return domain.TaskMessage{}, err
	}

	taskList := make([]domain.Tasks, len(taskRows))
	for i, row := range taskRows {
		taskList[i] = domain.Tasks{
			ID: row.TaskID, PlaybookID: history.PlaybookID.String(), Name: row.Name,
			Config: row.Config, ConnectorName: row.ConnectorName, ConnectorID: row.ConnectorID,
			Operation: row.Operation, Description: row.Description, Parameters: row.Parameters,
			X: row.X, Y: row.Y,
		}
	}

	// edge handles are stored as plain JSON strings or null
	var stored []struct {
		SourceTaskName      string  `json:"source_task_name"`
		DestinationTaskName string  `json:"destination_task_name"`
		SourceHandle        *string `json:"source_handle"`
	}
	if err := json.Unmarshal(history.Edges, &stored); err != nil {
		return domain.TaskMessage{}, fmt.Errorf("could not decode the stored edges: %w", err)
	}
	responseEdges := make([]domain.ResponseEdges, len(stored))
	for i, edge := range stored {
		responseEdges[i] = domain.ResponseEdges{SourceTaskName: edge.SourceTaskName, DestinationTaskName: edge.DestinationTaskName}
		if edge.SourceHandle != nil {
			responseEdges[i].SourceHandle = sql.NullString{String: *edge.SourceHandle, Valid: true}
		}
	}

	var input *domain.RunInput
	if len(history.Input) > 0 {
		if err := json.Unmarshal(history.Input, &input); err != nil {
			return domain.TaskMessage{}, fmt.Errorf("could not decode the stored input: %w", err)
		}
	}

	tasksMap, graph, edgeRefs := prepareMessage(taskList, responseEdges)
	return domain.TaskMessage{
		Graph: graph, Tasks: tasksMap, PlaybookHistoryId: historyID, Edges: edgeRefs, Input: input.Resolved(),
	}, nil
}

// PreparePlaybookMessage implements PlaybookApplicationService.
func (w *PlaybookApplicationServiceImpl) PreparePlaybookMessage(tasksData []domain.Tasks, edges []domain.ResponseEdges) (map[string]domain.Tasks, map[string][]string, []domain.EdgeRef) {
	return prepareMessage(tasksData, edges)
}

func prepareMessage(tasksData []domain.Tasks, edges []domain.ResponseEdges) (map[string]domain.Tasks, map[string][]string, []domain.EdgeRef) {
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
		} else if source, ok := tasksMap[edge.SourceTaskName]; ok &&
			source.ConnectorID != nil && *source.ConnectorID == domain.ConditionConnectorID {
			// a nil handle always follows in the executor, which would skip branch gating on a
			// condition edge. Use the "output" handle of the editor instead (never a followed branch).
			handle := domain.ConditionOutputHandle
			ref.SourceHandle = &handle
		}
		edgeRefs = append(edgeRefs, ref)
	}

	// the graph only has nodes with an edge so far. Add the rest so single-node and
	// disconnected tasks still run.
	for name := range tasksMap {
		if _, ok := graph[name]; !ok {
			graph[name] = []string{}
		}
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
