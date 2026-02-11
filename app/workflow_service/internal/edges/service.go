package edges

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type EdgeService interface {
	InsertEdges(tx *sqlx.Tx, edges []Edges) ([]Edges, error)
	DeleteEdges(tx *sqlx.Tx, edgeIds []uuid.UUID) error
	DeleteAllWorkflowEdges(tx *sqlx.Tx, workflowId string) error
	GetEdgesByWorkflowId(workflowId string) ([]ResponseEdges, error)
}

type EdgeServiceImpl struct {
	EdgeRepository EdgeRepository
}

func NewEdgeServiceImpl(EdgeRepository EdgeRepository) EdgeService {
	return &EdgeServiceImpl{
		EdgeRepository: EdgeRepository,
	}
}

// GetNodesByWorkflowId implements EdgeService.
func (e *EdgeServiceImpl) GetEdgesByWorkflowId(workflowId string) ([]ResponseEdges, error) {
	return e.EdgeRepository.GetEdgesByWorkflowId(workflowId)
}

// accepts multiple edge structs to be added in the database in a transaction matter
// Do nothing if there's already existing source and destination combined
func (e *EdgeServiceImpl) InsertEdges(tx *sqlx.Tx, edges []Edges) ([]Edges, error) {
	return e.EdgeRepository.InsertEdges(tx, edges)
}

// accepts multiple edge ids to be deleted
func (e *EdgeServiceImpl) DeleteEdges(tx *sqlx.Tx, edgeIds []uuid.UUID) error {
	return e.EdgeRepository.DeleteEdges(tx, edgeIds)
}

// accepts multiple edge ids to be deleted
func (e *EdgeServiceImpl) DeleteAllWorkflowEdges(tx *sqlx.Tx, workflowId string) error {
	return e.EdgeRepository.DeleteAllWorkflowEdges(tx, workflowId)
}
