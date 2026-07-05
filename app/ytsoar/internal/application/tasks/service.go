package tasks

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
)

//go:generate mockgen -destination=mocks/service_mock.go -package=mocks . TaskService
type TaskService interface {
	GetTasksByPlaybookId(ctx context.Context, playbookId string) ([]domain.Tasks, error)
	UpsertTasks(ctx context.Context, playbookId uuid.UUID, tasks []domain.Tasks) ([]domain.Tasks, error)
	DeleteTasks(ctx context.Context, taskIds []uuid.UUID) error
	CreateTaskHistory(ctx context.Context, playbookHistoryId string, tasks []domain.Tasks, graph map[uuid.UUID][]uuid.UUID) ([]domain.TaskHistory, error)
	UpdateTaskStatus(ctx context.Context, playbookHistoryId string, taskId string, status string) (*domain.TaskHistory, error)
	UpdateTaskHistory(ctx context.Context, playbookHistoryId string, taskId string, taskHistory UpdateTaskHistoryData) (*domain.TaskHistory, error)
	GetTaskHistoryByPlaybookHistoryId(ctx context.Context, id string, filter TaskHistoryFilter) ([]domain.TaskHistory, error)
	GetTaskHistoryCount(ctx context.Context, filter TaskHistoryFilter) (int, error)
}

type TaskServiceImpl struct {
	TaskRepository TaskRepository
}

func NewTaskServiceImpl(taskRepository TaskRepository) TaskService {
	return &TaskServiceImpl{
		TaskRepository: taskRepository,
	}
}

// CreateTaskHistory implements TaskService.
func (t *TaskServiceImpl) CreateTaskHistory(ctx context.Context, playbookHistoryId string, tasks []domain.Tasks, graph map[uuid.UUID][]uuid.UUID) ([]domain.TaskHistory, error) {
	return t.TaskRepository.CreateTaskHistory(ctx, playbookHistoryId, tasks, graph)
}

// GetTasksByPlaybookId implements TaskService.
func (t *TaskServiceImpl) GetTasksByPlaybookId(ctx context.Context, playbookId string) ([]domain.Tasks, error) {
	return t.TaskRepository.GetTasksByPlaybookId(ctx, playbookId), nil
}

// UpsertTasks inserts the tasks or updates their content when they exist.
func (t *TaskServiceImpl) UpsertTasks(ctx context.Context, playbookId uuid.UUID, tasks []domain.Tasks) ([]domain.Tasks, error) {
	return t.TaskRepository.UpsertTasks(ctx, playbookId, tasks)
}

// DeleteTasks deletes multiple tasks by id.
func (t *TaskServiceImpl) DeleteTasks(ctx context.Context, taskIds []uuid.UUID) error {
	return t.TaskRepository.DeleteTasks(ctx, taskIds)
}

// UpdateTaskStatus implements TaskService.
func (t *TaskServiceImpl) UpdateTaskStatus(ctx context.Context, playbookHistoryId string, taskId string, status string) (*domain.TaskHistory, error) {
	res, err := t.TaskRepository.UpdateTaskStatus(ctx, playbookHistoryId, taskId, status)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, fmt.Errorf("no task history was updated")
	}

	return res, nil
}

// UpdateTaskHistory implements TaskService.
func (t *TaskServiceImpl) UpdateTaskHistory(ctx context.Context, playbookHistoryId string, taskId string, taskHistory UpdateTaskHistoryData) (*domain.TaskHistory, error) {
	res, err := t.TaskRepository.UpdateTaskHistory(ctx, playbookHistoryId, taskId, taskHistory)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, fmt.Errorf("no task history was updated")
	}

	return res, nil
}

// GetTaskHistoryByPlaybookHistoryId implements TaskService.
func (t *TaskServiceImpl) GetTaskHistoryByPlaybookHistoryId(ctx context.Context, id string, filter TaskHistoryFilter) ([]domain.TaskHistory, error) {
	return t.TaskRepository.GetTaskHistoryByPlaybookHistoryId(ctx, id, filter)
}

// GetTaskHistoryCount implements TaskService.
func (t *TaskServiceImpl) GetTaskHistoryCount(ctx context.Context, filter TaskHistoryFilter) (int, error) {
	return t.TaskRepository.GetTaskHistoryCount(ctx, filter)
}
