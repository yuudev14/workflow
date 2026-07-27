package playbooks

import (
	"context"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/types"
)

//go:generate mockgen -destination=mocks/repository_mock.go -package=mocks . PlaybookRepository,RecordResolver

// RecordResolver hydrates the module rows a run acts on. Declared here, in the
// consumer's package, so playbooks does not import alerts/incidents - the same
// shape as IncidentLinker in application/alerts/ports.go.
//
// The caller sends ids and the server hydrates: list queries never select
// `payload`, which is exactly what templates read, and a client-supplied record
// would be spoofable.
type RecordResolver interface {
	Resolve(ctx context.Context, moduleType string, ids []uuid.UUID) ([]map[string]any, error)
}

type PlaybookRepository interface {
	GetPlaybooks(ctx context.Context, offset int, limit int, filter PlaybookFilter) ([]domain.Playbooks, error)
	GetPlaybookHistoryById(ctx context.Context, playbookHistoryId uuid.UUID) (*domain.PlaybookHistoryResponse, error)
	GetPlaybookHistory(ctx context.Context, offset int, limit int, filter PlaybookHistoryFilter) ([]domain.PlaybookHistoryResponse, error)
	GetPlaybookHistoryCount(ctx context.Context, filter PlaybookHistoryFilter) (int, error)
	GetPlaybooksCount(ctx context.Context, filter PlaybookFilter) (int, error)
	Summary(ctx context.Context, rng types.ResolvedRange) (PlaybooksSummary, error)
	GetPlaybookById(ctx context.Context, id string) (*domain.Playbooks, error)

	GetPlaybookGraphById(ctx context.Context, id string) (*domain.PlaybookGraph, error)
	CreatePlaybook(ctx context.Context, playbook PlaybookPayload) (*domain.Playbooks, error)
	UpdatePlaybook(ctx context.Context, id string, playbook UpdatePlaybookData) (*domain.Playbooks, error)
	CreatePlaybookHistory(ctx context.Context, id string, edges []domain.ResponseEdges, run RunStamp) (*domain.PlaybookHistory, error)
	CreatePlaybookRunRecords(ctx context.Context, historyID uuid.UUID, moduleType string, recordIDs []uuid.UUID) error
	UpdatePlaybookHistoryStatus(ctx context.Context, playbookHistoryId string, status string) (*domain.PlaybookHistory, error)
	UpdatePlaybookHistory(ctx context.Context, playbookHistoryId string, playbookHistory UpdatePlaybookHistoryData) (*domain.PlaybookHistory, error)
}
