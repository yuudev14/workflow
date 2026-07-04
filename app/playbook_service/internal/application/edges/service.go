package edges

import (
	"context"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
)

//go:generate mockgen -destination=mocks/service_mock.go -package=mocks . EdgeService
type EdgeService interface {
	InsertEdges(ctx context.Context, edges []domain.Edges) ([]domain.Edges, error)
	DeleteEdges(ctx context.Context, edgeIds []uuid.UUID) error
	DeleteAllPlaybookEdges(ctx context.Context, playbookId string) error
	GetEdgesByPlaybookId(ctx context.Context, playbookId string) ([]domain.ResponseEdges, error)
}

type EdgeServiceImpl struct {
	EdgeRepository EdgeRepository
}

func NewEdgeServiceImpl(EdgeRepository EdgeRepository) EdgeService {
	return &EdgeServiceImpl{
		EdgeRepository: EdgeRepository,
	}
}

// GetEdgesByPlaybookId implements EdgeService.
func (e *EdgeServiceImpl) GetEdgesByPlaybookId(ctx context.Context, playbookId string) ([]domain.ResponseEdges, error) {
	return e.EdgeRepository.GetEdgesByPlaybookId(ctx, playbookId)
}

// InsertEdges upserts multiple edges.
func (e *EdgeServiceImpl) InsertEdges(ctx context.Context, edges []domain.Edges) ([]domain.Edges, error) {
	return e.EdgeRepository.InsertEdges(ctx, edges)
}

// DeleteEdges deletes multiple edges by id.
func (e *EdgeServiceImpl) DeleteEdges(ctx context.Context, edgeIds []uuid.UUID) error {
	return e.EdgeRepository.DeleteEdges(ctx, edgeIds)
}

// DeleteAllPlaybookEdges deletes every edge belonging to the playbook.
func (e *EdgeServiceImpl) DeleteAllPlaybookEdges(ctx context.Context, playbookId string) error {
	return e.EdgeRepository.DeleteAllPlaybookEdges(ctx, playbookId)
}
