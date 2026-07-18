package tasks

import (
	"context"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
)

//go:generate mockgen -destination=mocks/repository_mock.go -package=mocks . TaskRepository

type TaskRepository interface {
	GetTasksByPlaybookId(ctx context.Context, playbookId string) []domain.Tasks
	UpsertTasks(ctx context.Context, playbookId uuid.UUID, tasks []domain.Tasks) ([]domain.Tasks, error)
	DeleteTasks(ctx context.Context, taskIds []uuid.UUID) error
	CreateTaskHistory(ctx context.Context, playbookHistoryId string, tasks []domain.Tasks, graph map[uuid.UUID][]uuid.UUID) ([]domain.TaskHistory, error)
	UpdateTaskStatus(ctx context.Context, playbookHistoryId string, taskId string, status string) (*domain.TaskHistory, error)
	UpdateTaskHistory(ctx context.Context, playbookHistoryId string, taskId string, taskHistory UpdateTaskHistoryData) (*domain.TaskHistory, error)
	GetTaskHistoryByPlaybookHistoryId(ctx context.Context, id string, filter TaskHistoryFilter) ([]domain.TaskHistory, error)
	GetTaskHistoryCount(ctx context.Context, filter TaskHistoryFilter) (int, error)
}
