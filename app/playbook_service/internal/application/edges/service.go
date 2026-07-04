package edges

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14/ytsoar/internal/domain"
)

//go:generate mockgen -destination=mocks/service_mock.go -package=mocks . EdgeService
type EdgeService interface {
	InsertEdges(tx *sqlx.Tx, edges []domain.Edges) ([]domain.Edges, error)
	DeleteEdges(tx *sqlx.Tx, edgeIds []uuid.UUID) error
	DeleteAllPlaybookEdges(tx *sqlx.Tx, workflowId string) error
	GetEdgesByPlaybookId(workflowId string) ([]domain.ResponseEdges, error)
}

type EdgeServiceImpl struct {
	EdgeRepository EdgeRepository
}

func NewEdgeServiceImpl(EdgeRepository EdgeRepository) EdgeService {
	return &EdgeServiceImpl{
		EdgeRepository: EdgeRepository,
	}
}

// GetNodesByPlaybookId implements EdgeService.
func (e *EdgeServiceImpl) GetEdgesByPlaybookId(workflowId string) ([]domain.ResponseEdges, error) {
	return e.EdgeRepository.GetEdgesByPlaybookId(workflowId)
}

// accepts multiple edge structs to be added in the database in a transaction matter
// Do nothing if there's already existing source and destination combined
func (e *EdgeServiceImpl) InsertEdges(tx *sqlx.Tx, edges []domain.Edges) ([]domain.Edges, error) {
	return e.EdgeRepository.InsertEdges(tx, edges)
}

// accepts multiple edge ids to be deleted
func (e *EdgeServiceImpl) DeleteEdges(tx *sqlx.Tx, edgeIds []uuid.UUID) error {
	return e.EdgeRepository.DeleteEdges(tx, edgeIds)
}

// accepts multiple edge ids to be deleted
func (e *EdgeServiceImpl) DeleteAllPlaybookEdges(tx *sqlx.Tx, workflowId string) error {
	return e.EdgeRepository.DeleteAllPlaybookEdges(tx, workflowId)
}
