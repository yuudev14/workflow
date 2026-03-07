package workflows

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14-workflow/workflow-service/internal/edges"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/logging"
	"github.com/yuudev14-workflow/workflow-service/internal/types"
)

type WorkflowService interface {
	GetWorkflows(offset int, limit int, filter WorkflowFilter) ([]Workflows, error)
	GetWorkflowsData(offset int, limit int, filter WorkflowFilter) (types.Entries[Workflows], error)
	GetWorkflowHistoryById(workflowHistoryId uuid.UUID) (*WorkflowHistoryResponse, error)
	GetWorkflowHistory(offset int, limit int, filter WorkflowHistoryFilter) ([]WorkflowHistoryResponse, error)
	GetWorkflowHistoryCount(filter WorkflowHistoryFilter) (int, error)
	GetWorkflowTriggers() ([]WorkflowTriggers, error)
	GetWorkflowsCount(filter WorkflowFilter) (int, error)
	GetWorkflowById(id string) (*Workflows, error)

	GetWorkflowGraphById(id string) (*WorkflowsGraph, error)
	CreateWorkflow(workflow WorkflowPayload) (*Workflows, error)
	UpdateWorkflow(id string, workflow UpdateWorkflowData) (*Workflows, error)
	UpdateWorkflowTx(tx *sqlx.Tx, id string, workflow UpdateWorkflowData) (*Workflows, error)
	CreateWorkflowHistory(tx *sqlx.Tx, id string, edges []edges.ResponseEdges) (*WorkflowHistory, error)
	UpdateWorkflowHistory(workflowHistoryId string, workflowHistory UpdateWorkflowHistoryData) (*WorkflowHistory, error)
	UpdateWorkflowHistoryStatus(workflowHistoryId string, status string) (*WorkflowHistory, error)
}

type WorkflowServiceImpl struct {
	WorkflowRepository WorkflowRepository
}

func NewWorkflowService(WorkflowRepository WorkflowRepository) WorkflowService {
	return &WorkflowServiceImpl{
		WorkflowRepository: WorkflowRepository,
	}
}

// GetWorkflows implements WorkflowService.
func (w *WorkflowServiceImpl) GetWorkflows(offset int, limit int, filter WorkflowFilter) ([]Workflows, error) {
	return w.WorkflowRepository.GetWorkflows(offset, limit, filter)
}

// GetWorkflowsData implements [WorkflowService].
func (w *WorkflowServiceImpl) GetWorkflowsData(offset int, limit int, filter WorkflowFilter) (types.Entries[Workflows], error) {
	workflows, err := w.GetWorkflows(
		offset,
		limit,
		filter,
	)
	if err != nil {
		return types.Entries[Workflows]{}, err
	}

	total, err := w.GetWorkflowsCount(filter)
	if err != nil {
		return types.Entries[Workflows]{}, err
	}

	return types.Entries[Workflows]{
		Entries: workflows,
		Total:   total,
	}, nil
}

func (w *WorkflowServiceImpl) GetWorkflowHistoryById(workflowHistoryId uuid.UUID) (*WorkflowHistoryResponse, error) {
	return w.WorkflowRepository.GetWorkflowHistoryById(workflowHistoryId)
}

// GetWorkflowHistory implements WorkflowService.
func (w *WorkflowServiceImpl) GetWorkflowHistory(offset int, limit int, filter WorkflowHistoryFilter) ([]WorkflowHistoryResponse, error) {
	return w.WorkflowRepository.GetWorkflowHistory(offset, limit, filter)
}

// GetWorkflowHistoryCount implements WorkflowService.
func (w *WorkflowServiceImpl) GetWorkflowHistoryCount(filter WorkflowHistoryFilter) (int, error) {
	return w.WorkflowRepository.GetWorkflowHistoryCount(filter)
}

// GetWorkflowTriggers implements WorkflowService.
func (w *WorkflowServiceImpl) GetWorkflowTriggers() ([]WorkflowTriggers, error) {
	return w.WorkflowRepository.GetWorkflowTriggers()
}

// GetWorkflowsCount implements WorkflowService.
func (w *WorkflowServiceImpl) GetWorkflowsCount(filter WorkflowFilter) (int, error) {
	return w.WorkflowRepository.GetWorkflowsCount(filter)
}

// CreateWorkflowHistory implements WorkflowService.
func (w *WorkflowServiceImpl) CreateWorkflowHistory(tx *sqlx.Tx, id string, edges []edges.ResponseEdges) (*WorkflowHistory, error) {
	return w.WorkflowRepository.CreateWorkflowHistory(tx, id, edges)
}

// GetWorkflowById implements WorkflowService.
func (w *WorkflowServiceImpl) GetWorkflowById(id string) (*Workflows, error) {
	workflow, workflowErr := w.WorkflowRepository.GetWorkflowById(id)
	if workflowErr != nil {
		logging.Sugar.Error(fmt.Sprintf("error fetching workflow by id: %v, error: %v", id, workflowErr))
		return nil, workflowErr
	}

	if workflow == nil {
		return nil, fmt.Errorf("workflow is not found")
	}
	return workflow, nil
}

// GetWorkflowById implements WorkflowService.
func (w *WorkflowServiceImpl) GetWorkflowGraphById(id string) (*WorkflowsGraph, error) {
	workflow, workflowErr := w.WorkflowRepository.GetWorkflowGraphById(id)
	if workflowErr != nil {
		logging.Sugar.Error(fmt.Sprintf("error fetching workflow by id: %s, error: %v", id, workflowErr))
		return nil, fmt.Errorf("error fetching graph by workflow by id: %s", id)
	}

	if workflow == nil {
		return nil, fmt.Errorf("workflow is not found")
	}
	return workflow, nil
}

// function for creating a workflow:
func (w *WorkflowServiceImpl) CreateWorkflow(workflow WorkflowPayload) (*Workflows, error) {
	return w.WorkflowRepository.CreateWorkflow(workflow)
}

// updateWorkflow implements WorkflowRepository.
func (w *WorkflowServiceImpl) UpdateWorkflow(id string, workflow UpdateWorkflowData) (*Workflows, error) {
	return w.WorkflowRepository.UpdateWorkflow(id, workflow)
}

// updateWorkflowTx implements WorkflowRepository.
func (w *WorkflowServiceImpl) UpdateWorkflowTx(tx *sqlx.Tx, id string, workflow UpdateWorkflowData) (*Workflows, error) {
	return w.WorkflowRepository.UpdateWorkflowTx(tx, id, workflow)
}

// UpdateWorkflowHistoryStatus implements WorkflowRepository.
func (w *WorkflowServiceImpl) UpdateWorkflowHistoryStatus(workflowHistoryId string, status string) (*WorkflowHistory, error) {
	res, err := w.WorkflowRepository.UpdateWorkflowHistoryStatus(workflowHistoryId, status)

	if err != nil {
		logging.Sugar.Error(fmt.Sprintf("error updating status of workflowHistoryId by id: %s, error: %v", workflowHistoryId, err))
		return nil, fmt.Errorf("error updating workflowHistoryId by id: %s", workflowHistoryId)
	}

	if res == nil {
		return nil, fmt.Errorf("no workflow status was updated")
	}

	return res, nil
}

// UpdateWorkflowHistoryStatus implements WorkflowRepository.
func (w *WorkflowServiceImpl) UpdateWorkflowHistory(workflowHistoryId string, workflowHistory UpdateWorkflowHistoryData) (*WorkflowHistory, error) {
	res, err := w.WorkflowRepository.UpdateWorkflowHistory(workflowHistoryId, workflowHistory)

	if err != nil {
		logging.Sugar.Error(fmt.Sprintf("error updating workflowHistoryId by id: %s, error: %v", workflowHistoryId, err))
		return nil, fmt.Errorf("error updating workflowHistoryId by id: %s", workflowHistoryId)
	}

	if res == nil {
		return nil, fmt.Errorf("no workflow status was updated")
	}

	return res, nil
}
