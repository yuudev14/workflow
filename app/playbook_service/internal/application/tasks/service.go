package tasks

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14/ytsoar/internal/domain"
)

//go:generate mockgen -destination=mocks/service_mock.go -package=mocks . TaskService
type TaskService interface {
	GetTasksByPlaybookId(workflowId string) ([]domain.Tasks, error)
	UpsertTasks(tx *sqlx.Tx, workflowId uuid.UUID, tasks []domain.Tasks) ([]domain.Tasks, error)
	DeleteTasks(tx *sqlx.Tx, taskIds []uuid.UUID) error
	CreateTaskHistory(tx *sqlx.Tx, workflowHistoryId string, tasks []domain.Tasks, graph map[uuid.UUID][]uuid.UUID) ([]domain.TaskHistory, error)
	UpdateTaskStatus(workflowHistoryId string, taskId string, status string) (*domain.TaskHistory, error)
	UpdateTaskHistory(workflowHistoryId string, taskId string, taskHistory UpdateTaskHistoryData) (*domain.TaskHistory, error)
	GetTaskHistoryByPlaybookHistoryId(id string, filter TaskHistoryFilter) ([]domain.TaskHistory, error)
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
func (t *TaskServiceImpl) CreateTaskHistory(tx *sqlx.Tx, workflowHistoryId string, tasks []domain.Tasks, graph map[uuid.UUID][]uuid.UUID) ([]domain.TaskHistory, error) {
	return t.TaskRepository.CreateTaskHistory(tx, workflowHistoryId, tasks, graph)
}

// get tasks by workflow id
func (t *TaskServiceImpl) GetTasksByPlaybookId(workflowId string) ([]domain.Tasks, error) {
	return t.TaskRepository.GetTasksByPlaybookId(workflowId), nil
}

// upsert tasks. insert multiple tasks.
// if task does not exist yet add the task in the database
// else update the content of the task
func (t *TaskServiceImpl) UpsertTasks(tx *sqlx.Tx, workflowId uuid.UUID, tasks []domain.Tasks) ([]domain.Tasks, error) {
	return t.TaskRepository.UpsertTasks(tx, workflowId, tasks)
}

// Delete multiple tasks based on the taskIds
func (t *TaskServiceImpl) DeleteTasks(tx *sqlx.Tx, taskIds []uuid.UUID) error {
	return t.TaskRepository.DeleteTasks(tx, taskIds)
}

// UpdateTaskStatus implements TaskService.
func (t *TaskServiceImpl) UpdateTaskStatus(workflowHistoryId string, taskId string, status string) (*domain.TaskHistory, error) {
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
func (t *TaskServiceImpl) UpdateTaskHistory(workflowHistoryId string, taskId string, taskHistory UpdateTaskHistoryData) (*domain.TaskHistory, error) {
	res, err := t.TaskRepository.UpdateTaskHistory(workflowHistoryId, taskId, taskHistory)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, fmt.Errorf("no task history was updated")
	}

	return res, nil
}

// GetPlaybookById implements PlaybookService.
func (t *TaskServiceImpl) GetTaskHistoryByPlaybookHistoryId(id string, filter TaskHistoryFilter) ([]domain.TaskHistory, error) {
	return t.TaskRepository.GetTaskHistoryByPlaybookHistoryId(id, filter)
}

// GetTaskHistoryCount implements PlaybookService.
func (t *TaskServiceImpl) GetTaskHistoryCount(filter TaskHistoryFilter) (int, error) {
	return t.TaskRepository.GetTaskHistoryCount(filter)
}
