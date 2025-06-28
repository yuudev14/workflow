package service

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14-workflow/workflow-service/models"
	"github.com/yuudev14-workflow/workflow-service/pkg/logging"
	"github.com/yuudev14-workflow/workflow-service/pkg/repository"
)

type EdgeService interface {
	InsertEdges(tx *sqlx.Tx, edges []models.Edges) ([]models.Edges, error)
	DeleteEdges(tx *sqlx.Tx, edgeIds []uuid.UUID) error
	DeleteAllWorkflowEdges(tx *sqlx.Tx, workflowId string) error
	GetEdgesByWorkflowId(workflowId string) ([]repository.Edges, error)
}

type EdgeServiceImpl struct {
	EdgeRepository  repository.EdgeRepository
	WorkflowService WorkflowService
}

func NewEdgeServiceImpl(EdgeRepository repository.EdgeRepository, WorkflowService WorkflowService) EdgeService {
	return &EdgeServiceImpl{
		EdgeRepository:  EdgeRepository,
		WorkflowService: WorkflowService,
	}
}

// GetNodesByWorkflowId implements EdgeService.
func (e *EdgeServiceImpl) GetEdgesByWorkflowId(workflowId string) ([]repository.Edges, error) {
	_, err := e.WorkflowService.GetWorkflowById(workflowId)
	if err != nil {
		return nil, err
	}
	return e.EdgeRepository.GetEdgesByWorkflowId(workflowId)
}

// accepts multiple edge structs to be added in the database in a transaction matter
// Do nothing if there's already existing source and destination combined
func (e *EdgeServiceImpl) InsertEdges(tx *sqlx.Tx, edges []models.Edges) ([]models.Edges, error) {
	return e.EdgeRepository.InsertEdges(tx, edges)
}

// accepts multiple edge ids to be deleted
func (e *EdgeServiceImpl) DeleteEdges(tx *sqlx.Tx, edgeIds []uuid.UUID) error {
	return e.EdgeRepository.DeleteEdges(tx, edgeIds)
}

// accepts multiple edge ids to be deleted
func (e *EdgeServiceImpl) DeleteAllWorkflowEdges(tx *sqlx.Tx, workflowId string) error {
	_, err := e.WorkflowService.GetWorkflowById(workflowId)
	if err != nil {
		logging.Sugar.Errorf("error when deleting workflow edges", err)
		return err
	}
	return e.EdgeRepository.DeleteAllWorkflowEdges(tx, workflowId)
}
