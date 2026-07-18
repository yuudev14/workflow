package playbooks

import (
	"context"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
)

//go:generate mockgen -destination=mocks/repository_mock.go -package=mocks . PlaybookRepository

type PlaybookRepository interface {
	GetPlaybooks(ctx context.Context, offset int, limit int, filter PlaybookFilter) ([]domain.Playbooks, error)
	GetPlaybookHistoryById(ctx context.Context, playbookHistoryId uuid.UUID) (*domain.PlaybookHistoryResponse, error)
	GetPlaybookHistory(ctx context.Context, offset int, limit int, filter PlaybookHistoryFilter) ([]domain.PlaybookHistoryResponse, error)
	GetPlaybookHistoryCount(ctx context.Context, filter PlaybookHistoryFilter) (int, error)
	GetPlaybookTriggers(ctx context.Context) ([]domain.PlaybookTriggers, error)
	GetPlaybooksCount(ctx context.Context, filter PlaybookFilter) (int, error)
	GetPlaybookById(ctx context.Context, id string) (*domain.Playbooks, error)

	GetPlaybookGraphById(ctx context.Context, id string) (*domain.PlaybookGraph, error)
	CreatePlaybook(ctx context.Context, playbook PlaybookPayload) (*domain.Playbooks, error)
	UpdatePlaybook(ctx context.Context, id string, playbook UpdatePlaybookData) (*domain.Playbooks, error)
	CreatePlaybookHistory(ctx context.Context, id string, edges []domain.ResponseEdges) (*domain.PlaybookHistory, error)
	UpdatePlaybookHistoryStatus(ctx context.Context, playbookHistoryId string, status string) (*domain.PlaybookHistory, error)
	UpdatePlaybookHistory(ctx context.Context, playbookHistoryId string, playbookHistory UpdatePlaybookHistoryData) (*domain.PlaybookHistory, error)
}
