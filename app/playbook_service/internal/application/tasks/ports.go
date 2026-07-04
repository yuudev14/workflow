package tasks

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14/ytsoar/internal/domain"
)

//go:generate mockgen -destination=mocks/repository_mock.go -package=mocks . TaskRepository

type TaskRepository interface {
	GetTasksByPlaybookId(playbookId string) []domain.Tasks
	UpsertTasks(tx *sqlx.Tx, playbookId uuid.UUID, tasks []domain.Tasks) ([]domain.Tasks, error)
	DeleteTasks(tx *sqlx.Tx, taskIds []uuid.UUID) error
	CreateTaskHistory(tx *sqlx.Tx, playbookHistoryId string, tasks []domain.Tasks, graph map[uuid.UUID][]uuid.UUID) ([]domain.TaskHistory, error)
	UpdateTaskStatus(playbookHistoryId string, taskId string, status string) (*domain.TaskHistory, error)
	UpdateTaskHistory(playbookHistoryId string, taskId string, taskHistory UpdateTaskHistoryData) (*domain.TaskHistory, error)
	GetTaskHistoryByPlaybookHistoryId(id string, filter TaskHistoryFilter) ([]domain.TaskHistory, error)
	GetTaskHistoryCount(filter TaskHistoryFilter) (int, error)
}
