package service

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/yuudev14-workflow/workflow-service/dto"
	"github.com/yuudev14-workflow/workflow-service/models"
	"github.com/yuudev14-workflow/workflow-service/pkg/repository"
)

type WorkflowService interface {
	GetWorkflows(offset int, limit int, filter dto.WorkflowFilter) ([]models.Workflows, error)
	GetWorkflowHistory(offset int, limit int, filter dto.WorkflowHistoryFilter) ([]repository.WorkflowHistoryResponse, error)
	GetWorkflowHistoryCount(filter dto.WorkflowHistoryFilter) (int, error)
	GetWorkflowTriggers() ([]models.WorkflowTriggers, error)
	GetWorkflowsCount(filter dto.WorkflowFilter) (int, error)
	GetWorkflowById(id string) (*models.Workflows, error)
	GetTaskHistoryByWorkflowHistoryId(id string, filter dto.TaskHistoryFilter) ([]models.TaskHistory, error)
	GetTaskHistoryCount(filter dto.TaskHistoryFilter) (int, error)
	GetWorkflowGraphById(id string) (*repository.WorkflowsGraph, error)
	CreateWorkflow(workflow dto.WorkflowPayload) (*models.Workflows, error)
	UpdateWorkflow(id string, workflow dto.UpdateWorkflowData) (*models.Workflows, error)
	UpdateWorkflowTx(tx *sqlx.Tx, id string, workflow dto.UpdateWorkflowData) (*models.Workflows, error)
	CreateWorkflowHistory(tx *sqlx.Tx, id string, edges []models.Edges) (*models.WorkflowHistory, error)
	UpdateWorkflowHistory(workflowHistoryId string, workflowHistory dto.UpdateWorkflowHistoryData) (*models.WorkflowHistory, error)
	UpdateWorkflowHistoryStatus(workflowHistoryId string, status string) (*models.WorkflowHistory, error)
}

type WorkflowServiceImpl struct {
	WorkflowRepository repository.WorkflowRepository
}

func NewWorkflowService(WorkflowRepository repository.WorkflowRepository) WorkflowService {
	return &WorkflowServiceImpl{
		WorkflowRepository: WorkflowRepository,
	}
}

// GetWorkflows implements WorkflowService.
func (w *WorkflowServiceImpl) GetWorkflows(offset int, limit int, filter dto.WorkflowFilter) ([]models.Workflows, error) {
	return w.WorkflowRepository.GetWorkflows(offset, limit, filter)
}

// GetWorkflowHistory implements WorkflowService.
func (w *WorkflowServiceImpl) GetWorkflowHistory(offset int, limit int, filter dto.WorkflowHistoryFilter) ([]repository.WorkflowHistoryResponse, error) {
	return w.WorkflowRepository.GetWorkflowHistory(offset, limit, filter)
}

// GetWorkflowHistoryCount implements WorkflowService.
func (w *WorkflowServiceImpl) GetWorkflowHistoryCount(filter dto.WorkflowHistoryFilter) (int, error) {
	return w.WorkflowRepository.GetWorkflowHistoryCount(filter)
}

// GetWorkflowTriggers implements WorkflowService.
func (w *WorkflowServiceImpl) GetWorkflowTriggers() ([]models.WorkflowTriggers, error) {
	return w.WorkflowRepository.GetWorkflowTriggers()
}

// GetWorkflowsCount implements WorkflowService.
func (w *WorkflowServiceImpl) GetWorkflowsCount(filter dto.WorkflowFilter) (int, error) {
	return w.WorkflowRepository.GetWorkflowsCount(filter)
}

// CreateWorkflowHistory implements WorkflowService.
func (w *WorkflowServiceImpl) CreateWorkflowHistory(tx *sqlx.Tx, id string, edges []models.Edges) (*models.WorkflowHistory, error) {
	return w.WorkflowRepository.CreateWorkflowHistory(tx, id, edges)
}

// GetWorkflowById implements WorkflowService.
func (w *WorkflowServiceImpl) GetWorkflowById(id string) (*models.Workflows, error) {
	workflow, workflowErr := w.WorkflowRepository.GetWorkflowById(id)
	if workflowErr != nil {
		return nil, workflowErr
	}

	if workflow == nil {
		return nil, fmt.Errorf("workflow is not found")
	}
	return workflow, nil
}

// GetWorkflowById implements WorkflowService.
func (w *WorkflowServiceImpl) GetTaskHistoryByWorkflowHistoryId(id string, filter dto.TaskHistoryFilter) ([]models.TaskHistory, error) {
	return w.WorkflowRepository.GetTaskHistoryByWorkflowHistoryId(id, filter)
}

// GetTaskHistoryCount implements WorkflowService.
func (w *WorkflowServiceImpl) GetTaskHistoryCount(filter dto.TaskHistoryFilter) (int, error) {
	return w.WorkflowRepository.GetTaskHistoryCount(filter)
}

// GetWorkflowById implements WorkflowService.
func (w *WorkflowServiceImpl) GetWorkflowGraphById(id string) (*repository.WorkflowsGraph, error) {
	workflow, workflowErr := w.WorkflowRepository.GetWorkflowGraphById(id)
	if workflowErr != nil {
		return nil, workflowErr
	}

	if workflow == nil {
		return nil, fmt.Errorf("workflow is not found")
	}
	return workflow, nil
}

// function for creating a workflow:
func (w *WorkflowServiceImpl) CreateWorkflow(workflow dto.WorkflowPayload) (*models.Workflows, error) {
	return w.WorkflowRepository.CreateWorkflow(workflow)
}

// updateWorkflow implements WorkflowRepository.
func (w *WorkflowServiceImpl) UpdateWorkflow(id string, workflow dto.UpdateWorkflowData) (*models.Workflows, error) {
	return w.WorkflowRepository.UpdateWorkflow(id, workflow)
}

// updateWorkflowTx implements WorkflowRepository.
func (w *WorkflowServiceImpl) UpdateWorkflowTx(tx *sqlx.Tx, id string, workflow dto.UpdateWorkflowData) (*models.Workflows, error) {
	return w.WorkflowRepository.UpdateWorkflowTx(tx, id, workflow)
}

// UpdateWorkflowHistoryStatus implements WorkflowRepository.
func (w *WorkflowServiceImpl) UpdateWorkflowHistoryStatus(workflowHistoryId string, status string) (*models.WorkflowHistory, error) {
	res, err := w.WorkflowRepository.UpdateWorkflowHistoryStatus(workflowHistoryId, status)

	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, fmt.Errorf("no workflow status was updated")
	}

	return res, nil
}

// UpdateWorkflowHistoryStatus implements WorkflowRepository.
func (w *WorkflowServiceImpl) UpdateWorkflowHistory(workflowHistoryId string, workflowHistory dto.UpdateWorkflowHistoryData) (*models.WorkflowHistory, error) {
	res, err := w.WorkflowRepository.UpdateWorkflowHistory(workflowHistoryId, workflowHistory)

	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, fmt.Errorf("no workflow status was updated")
	}

	return res, nil
}
