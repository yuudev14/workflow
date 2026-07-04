package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	"github.com/yuudev14/ytsoar/internal/domain"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/jmoiron/sqlx"
	"github.com/yuudev14/ytsoar/internal/types"
	"github.com/yuudev14/ytsoar/internal/utils"
)

type PlaybookRepositoryImpl struct {
	*sqlx.DB
}

func NewPlaybookRepository(db *sqlx.DB) *PlaybookRepositoryImpl {
	return &PlaybookRepositoryImpl{
		DB: db,
	}
}

// GetPlaybooks implements PlaybookRepository.
func (w *PlaybookRepositoryImpl) GetPlaybooks(offset int, limit int, filter playbooks.PlaybookFilter) ([]domain.Playbooks, error) {

	statement := sq.Select("*").From("workflows").OrderBy("updated_at DESC").Offset(uint64(offset)).Limit(uint64(limit))

	if filter.Name != nil {
		statement = statement.Where("name ILIKE ?", fmt.Sprint("%", *filter.Name, "%"))

	}
	return DbExecAndReturnMany[domain.Playbooks](
		w.DB,
		statement,
	)
}

// GetPlaybookHistoryById implements PlaybookRepository.
func (w *PlaybookRepositoryImpl) GetPlaybookHistoryById(workflowHistoryId uuid.UUID) (*domain.PlaybookHistoryResponse, error) {
	statement := sq.
		Select("workflow_history.*, to_jsonb(workflows) AS workflow_data ").
		From("workflow_history").
		Join("workflows on workflows.id = workflow_history.workflow_id").
		Where("workflow_history.id = ?", workflowHistoryId)

	return DbExecAndReturnOne[domain.PlaybookHistoryResponse](
		w.DB,
		statement,
	)
}

// GetPlaybooks implements PlaybookRepository.
func (w *PlaybookRepositoryImpl) GetPlaybookHistory(offset int, limit int, filter playbooks.PlaybookHistoryFilter) ([]domain.PlaybookHistoryResponse, error) {
	// select workflow_history.*, to_jsonb(workflows) AS workflow_data from workflow_history
	// join workflows on workflows.id = workflow_history.workflow_id
	statement := sq.Select("workflow_history.*, to_jsonb(workflows) AS workflow_data ").From("workflow_history").Join("workflows on workflows.id = workflow_history.workflow_id").Offset(uint64(offset)).Limit(uint64(limit)).OrderBy("triggered_at DESC")

	if filter.Name != nil {
		statement = statement.Where("name ILIKE ?", fmt.Sprint("%", filter.Name, "%"))
	}
	if filter.PlaybookID != nil {
		statement = statement.Where("workflow_id = ?", filter.PlaybookID)
	}
	return DbExecAndReturnMany[domain.PlaybookHistoryResponse](
		w.DB,
		statement,
	)
}

// GetPlaybookHistoryCount implements PlaybookRepository.
func (w *PlaybookRepositoryImpl) GetPlaybookHistoryCount(filter playbooks.PlaybookHistoryFilter) (int, error) {
	statement := sq.Select("count(workflow_history.*)").From("workflow_history").Join("workflows on workflows.id = workflow_history.workflow_id")

	if filter.Name != nil {
		statement = statement.Where("workflows.name ILIKE ?", fmt.Sprint("%", filter.Name, "%"))

	}
	return DbExecAndReturnCount(
		w.DB,
		statement,
	)
}

// GetPlaybookTriggers implements PlaybookRepository.
func (w *PlaybookRepositoryImpl) GetPlaybookTriggers() ([]domain.PlaybookTriggers, error) {
	statement := sq.Select("*").From("workflow_triggers")
	return DbExecAndReturnMany[domain.PlaybookTriggers](
		w.DB,
		statement,
	)
}

// GetPlaybooksCount implements PlaybookRepository.
func (w *PlaybookRepositoryImpl) GetPlaybooksCount(filter playbooks.PlaybookFilter) (int, error) {
	statement := sq.Select("count(*)").From("workflows")

	if filter.Name != nil {
		statement = statement.Where("name ILIKE ?", fmt.Sprint("%", filter.Name, "%"))

	}
	return DbExecAndReturnCount(
		w.DB,
		statement,
	)
}

// GetPlaybookById implements PlaybookRepository.
func (w *PlaybookRepositoryImpl) GetPlaybookById(id string) (*domain.Playbooks, error) {
	statement := sq.Select("*").From("workflows").Where("id = ?", id)
	return DbExecAndReturnOne[domain.Playbooks](
		w.DB,
		statement,
	)
}

// GetPlaybookById implements PlaybookRepository.
func (w *PlaybookRepositoryImpl) GetPlaybookGraphById(id string) (*domain.PlaybookGraph, error) {
	statement := sq.Select(`
	workflows.*,
	(SELECT JSON_AGG(tasks.*)
        FROM tasks
        WHERE tasks.workflow_id = workflows.id) AS tasks,
	(SELECT JSON_AGG(edges.*)
        FROM edges
        WHERE edges.workflow_id = workflows.id) AS edges
	`).From("workflows").Where("id = ?", id)
	return DbExecAndReturnOne[domain.PlaybookGraph](
		w.DB,
		statement,
	)
}

// CreatePlaybookHistory implements PlaybookRepository.
func (w *PlaybookRepositoryImpl) CreatePlaybookHistory(tx *sqlx.Tx, id string, edges []domain.ResponseEdges) (*domain.PlaybookHistory, error) {
	modifiedEdges := make([]map[string]interface{}, len(edges))

	for i, edge := range edges {
		modifiedEdges[i] = map[string]interface{}{
			"id":                    edge.ID,
			"destination_id":        edge.DestinationID,
			"source_id":             edge.SourceID,
			"workflow_id":           edge.PlaybookID,
			"destination_task_name": edge.DestinationTaskName,
			"source_task_name":      edge.SourceTaskName,
			"destination_handle":    utils.NullStringToInterface(edge.DestinationHandle),
			"source_handle":         utils.NullStringToInterface(edge.SourceHandle),
		}

	}
	edgesJSON, _ := json.Marshal(modifiedEdges)
	statement := sq.Insert("workflow_history").Columns("workflow_id", "triggered_at", "edges").Values(id, time.Now(), edgesJSON).Suffix("RETURNING *")
	return DbExecAndReturnOne[domain.PlaybookHistory](
		tx,
		statement,
	)
}

// function for creating a workflow:
func (w *PlaybookRepositoryImpl) CreatePlaybook(workflow playbooks.PlaybookPayload) (*domain.Playbooks, error) {
	statement := sq.Insert("workflows").Columns("name", "description", "trigger_type").Values(workflow.Name, workflow.Description, workflow.TriggerType).Suffix("RETURNING *")
	return DbExecAndReturnOne[domain.Playbooks](
		w.DB,
		statement,
	)
}

// updatePlaybook implements PlaybookRepository.
func (w *PlaybookRepositoryImpl) UpdatePlaybook(id string, workflow playbooks.UpdatePlaybookData) (*domain.Playbooks, error) {

	data := GenerateKeyValueQuery(map[string]types.Nullable[any]{
		"name":         workflow.Name.ToNullableAny(),
		"description":  workflow.Description.ToNullableAny(),
		"trigger_type": workflow.TriggerType.ToNullableAny(),
	})

	statement := sq.Update("workflows").SetMap(data).Where(sq.Eq{"id": id}).Suffix("RETURNING *")

	return DbExecAndReturnOne[domain.Playbooks](
		w.DB,
		statement,
	)
}

// updatePlaybook implements PlaybookRepository.
func (w *PlaybookRepositoryImpl) UpdatePlaybookTx(tx *sqlx.Tx, id string, workflow playbooks.UpdatePlaybookData) (*domain.Playbooks, error) {

	data := GenerateKeyValueQuery(map[string]types.Nullable[any]{
		"name":         workflow.Name.ToNullableAny(),
		"description":  workflow.Description.ToNullableAny(),
		"trigger_type": workflow.TriggerType.ToNullableAny(),
	})

	statement := sq.Update("workflows").SetMap(data).Where(sq.Eq{"id": id}).Suffix("RETURNING *")

	return DbExecAndReturnOne[domain.Playbooks](
		tx,
		statement,
	)
}

// UpdatePlaybookHistory implements PlaybookRepository.
func (w *PlaybookRepositoryImpl) UpdatePlaybookHistory(workflowHistoryId string, workflowHistory playbooks.UpdatePlaybookHistoryData) (*domain.PlaybookHistory, error) {
	data := GenerateKeyValueQuery(map[string]types.Nullable[any]{
		"status": workflowHistory.Status.ToNullableAny(),
		"error":  workflowHistory.Error.ToNullableAny(),
	})

	jsonData, err := json.Marshal(workflowHistory.Result)
	if err != nil {
		return nil, err
	}

	data["result"] = jsonData
	statement := sq.Update("workflow_history").SetMap(data).Where(sq.Eq{"id": workflowHistoryId}).Suffix("RETURNING *")
	return DbExecAndReturnOne[domain.PlaybookHistory](
		w.DB,
		statement,
	)
}

// UpdatePlaybookHistoryStatus implements PlaybookRepository.
func (w *PlaybookRepositoryImpl) UpdatePlaybookHistoryStatus(workflowHistoryId string, status string) (*domain.PlaybookHistory, error) {
	statement := sq.Update("workflow_history").Set("status", status).Where(sq.Eq{"id": workflowHistoryId}).Suffix("RETURNING *")
	return DbExecAndReturnOne[domain.PlaybookHistory](
		w.DB,
		statement,
	)
}
