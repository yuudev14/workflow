package tasks

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type TaskService interface {
	GetTasksByWorkflowId(workflowId string) ([]Tasks, error)
	UpsertTasks(tx *sqlx.Tx, workflowId uuid.UUID, tasks []Tasks) ([]Tasks, error)
	DeleteTasks(tx *sqlx.Tx, taskIds []uuid.UUID) error
	CreateTaskHistory(tx *sqlx.Tx, workflowHistoryId string, tasks []Tasks, graph map[uuid.UUID][]uuid.UUID) ([]TaskHistory, error)
	UpdateTaskStatus(workflowHistoryId string, taskId string, status string) (*TaskHistory, error)
	UpdateTaskHistory(workflowHistoryId string, taskId string, taskHistory UpdateTaskHistoryData) (*TaskHistory, error)
	GetTaskHistoryByWorkflowHistoryId(id string, filter TaskHistoryFilter) ([]TaskHistory, error)
	GetTaskHistoryCount(filter TaskHistoryFilter) (int, error)
}

type TaskServiceImpl struct {
	TaskRepository TaskRepository
}

func NewTaskServiceImpl(TaskService TaskRepository) TaskService {
	return &TaskServiceImpl{
		TaskRepository: TaskService,
	}
}

// CreateTaskHistory implements TaskService.
func (t *TaskServiceImpl) CreateTaskHistory(tx *sqlx.Tx, workflowHistoryId string, tasks []Tasks, graph map[uuid.UUID][]uuid.UUID) ([]TaskHistory, error) {
	return t.TaskRepository.CreateTaskHistory(tx, workflowHistoryId, tasks, graph)
}

// get tasks by workflow id
func (t *TaskServiceImpl) GetTasksByWorkflowId(workflowId string) ([]Tasks, error) {
	return t.TaskRepository.GetTasksByWorkflowId(workflowId), nil
}

// upsert tasks. insert multiple tasks.
// if task does not exist yet add the task in the database
// else update the content of the task
func (t *TaskServiceImpl) UpsertTasks(tx *sqlx.Tx, workflowId uuid.UUID, tasks []Tasks) ([]Tasks, error) {
	return t.TaskRepository.UpsertTasks(tx, workflowId, tasks)
}

// Delete multiple tasks based on the taskIds
func (t *TaskServiceImpl) DeleteTasks(tx *sqlx.Tx, taskIds []uuid.UUID) error {
	return t.TaskRepository.DeleteTasks(tx, taskIds)
}

// UpdateTaskStatus implements TaskService.
func (t *TaskServiceImpl) UpdateTaskStatus(workflowHistoryId string, taskId string, status string) (*TaskHistory, error) {
	res, err := t.TaskRepository.UpdateTaskStatus(workflowHistoryId, taskId, status)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, fmt.Errorf("no task history was updated")
	}

	return res, nil
}

// UpdateTaskStatus implements TaskService.
func (t *TaskServiceImpl) UpdateTaskHistory(workflowHistoryId string, taskId string, taskHistory UpdateTaskHistoryData) (*TaskHistory, error) {
	res, err := t.TaskRepository.UpdateTaskHistory(workflowHistoryId, taskId, taskHistory)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, fmt.Errorf("no task history was updated")
	}

	return res, nil
}

// GetWorkflowById implements WorkflowService.
func (t *TaskServiceImpl) GetTaskHistoryByWorkflowHistoryId(id string, filter TaskHistoryFilter) ([]TaskHistory, error) {
	return t.TaskRepository.GetTaskHistoryByWorkflowHistoryId(id, filter)
}

// GetTaskHistoryCount implements WorkflowService.
func (t *TaskServiceImpl) GetTaskHistoryCount(filter TaskHistoryFilter) (int, error) {
	return t.TaskRepository.GetTaskHistoryCount(filter)
}
