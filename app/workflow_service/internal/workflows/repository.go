package workflows

import (
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/jmoiron/sqlx"
	"github.com/yuudev14-workflow/workflow-service/internal/edges"
	repository "github.com/yuudev14-workflow/workflow-service/internal/infra/base_repository"
	"github.com/yuudev14-workflow/workflow-service/internal/types"
	"github.com/yuudev14-workflow/workflow-service/internal/utils"
)

type WorkflowsGraph struct {
	ID          uuid.UUID        `db:"id" json:"id"`
	Name        string           `db:"name" json:"name"`
	Description *string          `db:"description" json:"description"`
	TriggerType *string          `json:"trigger_type" db:"trigger_type"`
	CreatedAt   time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time        `db:"updated_at" json:"updated_at"`
	Tasks       *json.RawMessage `db:"tasks" json:"tasks"`
	Edges       *json.RawMessage `db:"edges" json:"edges"`
}

type WorkflowHistoryResponse struct {
	ID           uuid.UUID        `db:"id" json:"id"`
	WorkflowID   uuid.UUID        `db:"workflow_id" json:"workflow_id"`
	WorkflowData json.RawMessage  `db:"workflow_data" json:"workflow_data"`
	Status       string           `db:"status" json:"status"`
	Error        *string          `db:"error" json:"error"`
	Result       *json.RawMessage `db:"result" json:"result"`
	TriggeredAt  time.Time        `db:"triggered_at" json:"triggered_at"`
	Edges        json.RawMessage  `db:"edges" json:"edges"`
}

type WorkflowRepository interface {
	GetWorkflows(offset int, limit int, filter WorkflowFilter) ([]Workflows, error)
	GetWorkflowHistoryById(workflowHistoryId uuid.UUID) (*WorkflowHistoryResponse, error)
	GetWorkflowHistory(offset int, limit int, filter WorkflowHistoryFilter) ([]WorkflowHistoryResponse, error)
	GetWorkflowHistoryCount(filter WorkflowHistoryFilter) (int, error)
	GetWorkflowTriggers() ([]WorkflowTriggers, error)
	GetWorkflowsCount(filter WorkflowFilter) (int, error)
	GetWorkflowById(id string) (*Workflows, error)

	GetWorkflowGraphById(id string) (*WorkflowsGraph, error)
	CreateWorkflow(workflow WorkflowPayload) (*Workflows, error)
	UpdateWorkflow(id string, workflow UpdateWorkflowData) (*Workflows, error)
	UpdateWorkflowTx(tx *sqlx.Tx, id string, workflow UpdateWorkflowData) (*Workflows, error)
	CreateWorkflowHistory(tx *sqlx.Tx, id string, edges []edges.ResponseEdges) (*WorkflowHistory, error)
	UpdateWorkflowHistoryStatus(workflow_history_id string, status string) (*WorkflowHistory, error)
	UpdateWorkflowHistory(workflowHistoryId string, workflowHistory UpdateWorkflowHistoryData) (*WorkflowHistory, error)
}

type WorkflowRepositoryImpl struct {
	*sqlx.DB
}

func NewWorkflowRepository(db *sqlx.DB) WorkflowRepository {
	return &WorkflowRepositoryImpl{
		DB: db,
	}
}

// GetWorkflows implements WorkflowRepository.
func (w *WorkflowRepositoryImpl) GetWorkflows(offset int, limit int, filter WorkflowFilter) ([]Workflows, error) {

	statement := sq.Select("*").From("workflows").OrderBy("updated_at DESC").Offset(uint64(offset)).Limit(uint64(limit))

	if filter.Name != nil {
		statement = statement.Where("name ILIKE ?", fmt.Sprint("%", *filter.Name, "%"))

	}
	return repository.DbExecAndReturnMany[Workflows](
		w.DB,
		statement,
	)
}

// GetWorkflowHistoryById implements WorkflowRepository.
func (w *WorkflowRepositoryImpl) GetWorkflowHistoryById(workflowHistoryId uuid.UUID) (*WorkflowHistoryResponse, error) {
	statement := sq.
		Select("workflow_history.*, to_jsonb(workflows) AS workflow_data ").
		From("workflow_history").
		Join("workflows on workflows.id = workflow_history.workflow_id").
		Where("workflow_history.id = ?", workflowHistoryId)

	return repository.DbExecAndReturnOne[WorkflowHistoryResponse](
		w.DB,
		statement,
	)
}

// GetWorkflows implements WorkflowRepository.
func (w *WorkflowRepositoryImpl) GetWorkflowHistory(offset int, limit int, filter WorkflowHistoryFilter) ([]WorkflowHistoryResponse, error) {
	// select workflow_history.*, to_jsonb(workflows) AS workflow_data from workflow_history
	// join workflows on workflows.id = workflow_history.workflow_id
	statement := sq.Select("workflow_history.*, to_jsonb(workflows) AS workflow_data ").From("workflow_history").Join("workflows on workflows.id = workflow_history.workflow_id").Offset(uint64(offset)).Limit(uint64(limit)).OrderBy("triggered_at DESC")

	if filter.Name != nil {
		statement = statement.Where("name ILIKE ?", fmt.Sprint("%", filter.Name, "%"))
	}
	if filter.WorkflowID != nil {
		statement = statement.Where("workflow_id = ?", filter.WorkflowID)
	}
	return repository.DbExecAndReturnMany[WorkflowHistoryResponse](
		w.DB,
		statement,
	)
}

// GetWorkflowHistoryCount implements WorkflowRepository.
func (w *WorkflowRepositoryImpl) GetWorkflowHistoryCount(filter WorkflowHistoryFilter) (int, error) {
	statement := sq.Select("count(workflow_history.*)").From("workflow_history").Join("workflows on workflows.id = workflow_history.workflow_id")

	if filter.Name != nil {
		statement = statement.Where("workflows.name ILIKE ?", fmt.Sprint("%", filter.Name, "%"))

	}
	return repository.DbExecAndReturnCount(
		w.DB,
		statement,
	)
}

// GetWorkflowTriggers implements WorkflowRepository.
func (w *WorkflowRepositoryImpl) GetWorkflowTriggers() ([]WorkflowTriggers, error) {
	statement := sq.Select("*").From("workflow_triggers")
	return repository.DbExecAndReturnMany[WorkflowTriggers](
		w.DB,
		statement,
	)
}

// GetWorkflowsCount implements WorkflowRepository.
func (w *WorkflowRepositoryImpl) GetWorkflowsCount(filter WorkflowFilter) (int, error) {
	statement := sq.Select("count(*)").From("workflows")

	if filter.Name != nil {
		statement = statement.Where("name ILIKE ?", fmt.Sprint("%", filter.Name, "%"))

	}
	return repository.DbExecAndReturnCount(
		w.DB,
		statement,
	)
}

// GetWorkflowById implements WorkflowRepository.
func (w *WorkflowRepositoryImpl) GetWorkflowById(id string) (*Workflows, error) {
	statement := sq.Select("*").From("workflows").Where("id = ?", id)
	return repository.DbExecAndReturnOne[Workflows](
		w.DB,
		statement,
	)
}

// GetWorkflowById implements WorkflowRepository.
func (w *WorkflowRepositoryImpl) GetWorkflowGraphById(id string) (*WorkflowsGraph, error) {
	statement := sq.Select(`
	workflows.*,
	(SELECT JSON_AGG(tasks.*)
        FROM tasks
        WHERE tasks.workflow_id = workflows.id) AS tasks,
	(SELECT JSON_AGG(edges.*)
        FROM edges
        WHERE edges.workflow_id = workflows.id) AS edges
	`).From("workflows").Where("id = ?", id)
	return repository.DbExecAndReturnOne[WorkflowsGraph](
		w.DB,
		statement,
	)
}

// CreateWorkflowHistory implements WorkflowRepository.
func (w *WorkflowRepositoryImpl) CreateWorkflowHistory(tx *sqlx.Tx, id string, edges []edges.ResponseEdges) (*WorkflowHistory, error) {
	modifiedEdges := make([]map[string]interface{}, len(edges))

	for i, edge := range edges {
		modifiedEdges[i] = map[string]interface{}{
			"id":                    edge.ID,
			"destination_id":        edge.DestinationID,
			"source_id":             edge.SourceID,
			"workflow_id":           edge.WorkflowID,
			"destination_task_name": edge.DestinationTaskName,
			"source_task_name":      edge.SourceTaskName,
			"destination_handle":    utils.NullStringToInterface(edge.DestinationHandle),
			"source_handle":         utils.NullStringToInterface(edge.SourceHandle),
		}

	}
	edgesJSON, _ := json.Marshal(modifiedEdges)
	statement := sq.Insert("workflow_history").Columns("workflow_id", "triggered_at", "edges").Values(id, time.Now(), edgesJSON).Suffix("RETURNING *")
	return repository.DbExecAndReturnOne[WorkflowHistory](
		tx,
		statement,
	)
}

// function for creating a workflow:
func (w *WorkflowRepositoryImpl) CreateWorkflow(workflow WorkflowPayload) (*Workflows, error) {
	statement := sq.Insert("workflows").Columns("name", "description", "trigger_type").Values(workflow.Name, workflow.Description, workflow.TriggerType).Suffix("RETURNING *")
	return repository.DbExecAndReturnOne[Workflows](
		w.DB,
		statement,
	)
}

// updateWorkflow implements WorkflowRepository.
func (w *WorkflowRepositoryImpl) UpdateWorkflow(id string, workflow UpdateWorkflowData) (*Workflows, error) {

	data := repository.GenerateKeyValueQuery(map[string]types.Nullable[any]{
		"name":         workflow.Name.ToNullableAny(),
		"description":  workflow.Description.ToNullableAny(),
		"trigger_type": workflow.TriggerType.ToNullableAny(),
	})

	statement := sq.Update("workflows").SetMap(data).Where(sq.Eq{"id": id}).Suffix("RETURNING *")

	return repository.DbExecAndReturnOne[Workflows](
		w.DB,
		statement,
	)
}

// updateWorkflow implements WorkflowRepository.
func (w *WorkflowRepositoryImpl) UpdateWorkflowTx(tx *sqlx.Tx, id string, workflow UpdateWorkflowData) (*Workflows, error) {

	data := repository.GenerateKeyValueQuery(map[string]types.Nullable[any]{
		"name":         workflow.Name.ToNullableAny(),
		"description":  workflow.Description.ToNullableAny(),
		"trigger_type": workflow.TriggerType.ToNullableAny(),
	})

	statement := sq.Update("workflows").SetMap(data).Where(sq.Eq{"id": id}).Suffix("RETURNING *")

	return repository.DbExecAndReturnOne[Workflows](
		tx,
		statement,
	)
}

// UpdateWorkflowHistory implements WorkflowRepository.
func (w *WorkflowRepositoryImpl) UpdateWorkflowHistory(workflowHistoryId string, workflowHistory UpdateWorkflowHistoryData) (*WorkflowHistory, error) {
	data := repository.GenerateKeyValueQuery(map[string]types.Nullable[any]{
		"status": workflowHistory.Status.ToNullableAny(),
		"error":  workflowHistory.Error.ToNullableAny(),
	})

	jsonData, err := json.Marshal(workflowHistory.Result)
	if err != nil {
		return nil, err
	}

	data["result"] = jsonData
	statement := sq.Update("workflow_history").SetMap(data).Where(sq.Eq{"id": workflowHistoryId}).Suffix("RETURNING *")
	return repository.DbExecAndReturnOne[WorkflowHistory](
		w.DB,
		statement,
	)
}

// UpdateWorkflowHistoryStatus implements WorkflowRepository.
func (w *WorkflowRepositoryImpl) UpdateWorkflowHistoryStatus(workflowHistoryId string, status string) (*WorkflowHistory, error) {
	statement := sq.Update("workflow_history").Set("status", status).Where(sq.Eq{"id": workflowHistoryId}).Suffix("RETURNING *")
	return repository.DbExecAndReturnOne[WorkflowHistory](
		w.DB,
		statement,
	)
}
