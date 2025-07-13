package repository

import (
	"encoding/json"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14-workflow/workflow-service/db/queries"
	"github.com/yuudev14-workflow/workflow-service/dto"
	"github.com/yuudev14-workflow/workflow-service/models"
	"github.com/yuudev14-workflow/workflow-service/pkg/logging"
	"github.com/yuudev14-workflow/workflow-service/pkg/types"
)

type TaskRepository interface {
	GetTasksByWorkflowId(workflowId string) []models.Tasks
	UpsertTasks(tx *sqlx.Tx, workflowId uuid.UUID, tasks []models.Tasks) ([]models.Tasks, error)
	DeleteTasks(tx *sqlx.Tx, taskIds []uuid.UUID) error
	CreateTaskHistory(tx *sqlx.Tx, workflowHistoryId string, tasks []models.Tasks) ([]models.TaskHistory, error)
	UpdateTaskStatus(workflowHistoryId string, taskId string, status string) (*models.TaskHistory, error)
	UpdateTaskHistory(workflowHistoryId string, taskId string, taskHistory dto.UpdateTaskHistoryData) (*models.TaskHistory, error)
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
func (t *TaskRepositoryImpl) CreateTaskHistory(tx *sqlx.Tx, workflowHistoryId string, tasks []models.Tasks) ([]models.TaskHistory, error) {
	statement := sq.Insert("task_history").Columns("workflow_history_id", "task_id", "triggered_at", "name", "description", "parameters", "config", "x", "y", "connector_name", "connector_id", "operation")

	for _, val := range tasks {
		parameters, err := json.Marshal(val.Parameters)
		if err != nil {
			return nil, err
		}

		statement = statement.Values(workflowHistoryId, val.ID, time.Now(), val.Name, val.Description, parameters, val.Config, val.X, val.Y, val.ConnectorName, val.ConnectorID, val.Operation)
	}

	statement = statement.Suffix(`RETURNING *`)

	return DbExecAndReturnMany[models.TaskHistory](
		tx,
		statement,
	)
}

// get tasks by workflow id
func (t *TaskRepositoryImpl) GetTasksByWorkflowId(workflowId string) []models.Tasks {
	result, _ := DbExecAndReturnManyOld[models.Tasks](
		t,
		queries.GET_TASK_BY_WORKFLOW_ID,
		workflowId,
	)

	return result
}

// upsert tasks. insert multiple tasks.
// if task does not exist yet add the task in the database
// else update the content of the task
func (t *TaskRepositoryImpl) UpsertTasks(tx *sqlx.Tx, workflowId uuid.UUID, tasks []models.Tasks) ([]models.Tasks, error) {

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

	return DbExecAndReturnMany[models.Tasks](
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
func (t *TaskRepositoryImpl) UpdateTaskStatus(workflowHistoryId string, taskId string, status string) (*models.TaskHistory, error) {
	statement := sq.Update("task_history").Set("status", status).Where("workflow_history_id = ? and task_id = ?", workflowHistoryId, taskId).Suffix("RETURNING *")
	return DbExecAndReturnOne[models.TaskHistory](
		t.DB,
		statement,
	)
}

// UpdateTaskHistory implements TaskRepository.
func (t *TaskRepositoryImpl) UpdateTaskHistory(workflowHistoryId string, taskId string, taskHistory dto.UpdateTaskHistoryData) (*models.TaskHistory, error) {
	data := GenerateKeyValueQuery(map[string]types.Nullable[any]{
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
	return DbExecAndReturnOne[models.TaskHistory](
		t.DB,
		statement,
	)
}
