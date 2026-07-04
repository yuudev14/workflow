package edges

import (
	"context"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
)

//go:generate mockgen -destination=mocks/repository_mock.go -package=mocks . EdgeRepository

type EdgeRepository interface {
	InsertEdges(ctx context.Context, edges []domain.Edges) ([]domain.Edges, error)
	DeleteEdges(ctx context.Context, edgeIds []uuid.UUID) error
	DeleteAllPlaybookEdges(ctx context.Context, playbookId string) error
	GetEdgesByPlaybookId(ctx context.Context, playbookId string) ([]domain.ResponseEdges, error)
}
