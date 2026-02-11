package tasks

import (
	"encoding/json"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/yuudev14-workflow/workflow-service/db/queries"
	repository "github.com/yuudev14-workflow/workflow-service/internal/infra/base_repository"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/logging"
	"github.com/yuudev14-workflow/workflow-service/internal/types"
)

type TaskRepository interface {
	GetTasksByWorkflowId(workflowId string) []Tasks
	UpsertTasks(tx *sqlx.Tx, workflowId uuid.UUID, tasks []Tasks) ([]Tasks, error)
	DeleteTasks(tx *sqlx.Tx, taskIds []uuid.UUID) error
	CreateTaskHistory(tx *sqlx.Tx, workflowHistoryId string, tasks []Tasks, graph map[uuid.UUID][]uuid.UUID) ([]TaskHistory, error)
	UpdateTaskStatus(workflowHistoryId string, taskId string, status string) (*TaskHistory, error)
	UpdateTaskHistory(workflowHistoryId string, taskId string, taskHistory UpdateTaskHistoryData) (*TaskHistory, error)
	GetTaskHistoryByWorkflowHistoryId(id string, filter TaskHistoryFilter) ([]TaskHistory, error)
	GetTaskHistoryCount(filter TaskHistoryFilter) (int, error)
}

type TaskRepositoryImpl struct {
	*sqlx.DB
}

func NewTaskRepositoryImpl(db *sqlx.DB) TaskRepository {
	return &TaskRepositoryImpl{
		DB: db,
	}
}

// CreateTaskHistory implements TaskRepository.
func (t *TaskRepositoryImpl) CreateTaskHistory(tx *sqlx.Tx, workflowHistoryId string, tasks []Tasks, graph map[uuid.UUID][]uuid.UUID) ([]TaskHistory, error) {
	statement := sq.Insert("task_history").Columns("workflow_history_id", "task_id", "triggered_at", "name", "description", "parameters", "config", "x", "y", "connector_name", "connector_id", "operation", "destination_ids")

	for _, val := range tasks {
		parameters, err := json.Marshal(val.Parameters)
		if err != nil {
			return nil, err
		}

		// get the destination ids in the graph
		destinationIds := graph[val.ID]

		statement = statement.Values(workflowHistoryId, val.ID, time.Now(), val.Name, val.Description, parameters, val.Config, val.X, val.Y, val.ConnectorName, val.ConnectorID, val.Operation, pq.Array(destinationIds))
	}

	statement = statement.Suffix(`RETURNING *`)

	return repository.DbExecAndReturnMany[TaskHistory](
		tx,
		statement,
	)
}

// get tasks by workflow id
func (t *TaskRepositoryImpl) GetTasksByWorkflowId(workflowId string) []Tasks {
	result, _ := repository.DbExecAndReturnManyOld[Tasks](
		t,
		queries.GET_TASK_BY_WORKFLOW_ID,
		workflowId,
	)

	return result
}

// upsert tasks. insert multiple tasks.
// if task does not exist yet add the task in the database
// else update the content of the task
func (t *TaskRepositoryImpl) UpsertTasks(tx *sqlx.Tx, workflowId uuid.UUID, tasks []Tasks) ([]Tasks, error) {

	statement := sq.Insert("tasks").Columns("workflow_id", "name", "description", "parameters", "config", "connector_name", "connector_id", "operation", "x", "y")

	for _, val := range tasks {
		parameters, _ := json.Marshal(val.Parameters)
		statement = statement.Values(workflowId, val.Name, val.Description, parameters, val.Config, val.ConnectorName, val.ConnectorID, val.Operation, val.X, val.Y)
	}

	statement = statement.Suffix(`
		ON CONFLICT (workflow_id, name) DO UPDATE
   	SET description = EXCLUDED.description,
       parameters = EXCLUDED.parameters,
			 config = EXCLUDED.config,
			 connector_name = EXCLUDED.connector_name,
			 connector_id = EXCLUDED.connector_id,
			 operation = EXCLUDED.operation,
			 x = EXCLUDED.x,
			 y = EXCLUDED.y,
       updated_at = NOW()
		RETURNING *`)

	return repository.DbExecAndReturnMany[Tasks](
		tx,
		statement,
	)
}

// Delete multiple tasks based on the taskIds
func (t *TaskRepositoryImpl) DeleteTasks(tx *sqlx.Tx, taskIds []uuid.UUID) error {
	sql, args, err := sq.Delete("tasks").Where(sq.Eq{"id": taskIds}).ToSql()
	logging.Sugar.Debug("DeleteTasks SQL: ", sql)
	logging.Sugar.Debug("DeleteTasks Args: ", args)
	if err != nil {
		logging.Sugar.Error("Failed to build SQL query", err)
		return err
	}
	sql = tx.Rebind(sql)
	_, err = tx.Exec(sql, args...)
	if err != nil {
		logging.Sugar.Warn(err)
	}

	return err
}

// UpdateTaskStatus implements TaskRepository.
func (t *TaskRepositoryImpl) UpdateTaskStatus(workflowHistoryId string, taskId string, status string) (*TaskHistory, error) {
	statement := sq.Update("task_history").Set("status", status).Where("workflow_history_id = ? and task_id = ?", workflowHistoryId, taskId).Suffix("RETURNING *")
	return repository.DbExecAndReturnOne[TaskHistory](
		t.DB,
		statement,
	)
}

// UpdateTaskHistory implements TaskRepository.
func (t *TaskRepositoryImpl) UpdateTaskHistory(workflowHistoryId string, taskId string, taskHistory UpdateTaskHistoryData) (*TaskHistory, error) {
	data := repository.GenerateKeyValueQuery(map[string]types.Nullable[any]{
		"status":         taskHistory.Status.ToNullableAny(),
		"error":          taskHistory.Error.ToNullableAny(),
		"connector_name": taskHistory.ConnectorName.ToNullableAny(),
		"connector_id":   taskHistory.ConnectorID.ToNullableAny(),
		"config":         taskHistory.Config.ToNullableAny(),
	})

	data["name"] = taskHistory.Name
	data["description"] = taskHistory.Description
	data["x"] = taskHistory.X
	data["y"] = taskHistory.Y

	result, err := json.Marshal(taskHistory.Result)
	if err != nil {
		return nil, err
	}
	data["result"] = result

	parameters, err := json.Marshal(taskHistory.Parameters)
	if err != nil {
		return nil, err
	}

	data["parameters"] = parameters

	statement := sq.Update("task_history").SetMap(data).Where("workflow_history_id = ? and task_id = ?", workflowHistoryId, taskId).Suffix("RETURNING *")
	return repository.DbExecAndReturnOne[TaskHistory](
		t.DB,
		statement,
	)
}

// GetTaskHistoryByWorkflowHistoryId implements WorkflowRepository.
func (t *TaskRepositoryImpl) GetTaskHistoryByWorkflowHistoryId(id string, filter TaskHistoryFilter) ([]TaskHistory, error) {
	statement := sq.Select("*").From("task_history").Where("workflow_history_id = ?", id)
	return repository.DbExecAndReturnMany[TaskHistory](
		t.DB,
		statement,
	)
}

// GetTaskHistorGetTaskHistoryCountyByWorkflowId implements WorkflowRepository.
func (w *TaskRepositoryImpl) GetTaskHistoryCount(filter TaskHistoryFilter) (int, error) {
	statement := sq.Select("count(*)").From("task_history")
	if filter.WorkflowID != nil {
		statement = statement.Where("workflow_history_id = ?", filter.WorkflowID)

	}
	return repository.DbExecAndReturnCount(
		w.DB,
		statement,
	)
}
