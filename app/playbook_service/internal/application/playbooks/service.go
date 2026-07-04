package playbooks

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logging"
	"github.com/yuudev14/ytsoar/internal/types"
)

//go:generate mockgen -destination=mocks/service_mock.go -package=mocks . PlaybookService
type PlaybookService interface {
	GetPlaybooks(offset int, limit int, filter PlaybookFilter) ([]domain.Playbooks, error)
	GetPlaybooksData(offset int, limit int, filter PlaybookFilter) (types.Entries[domain.Playbooks], error)
	GetPlaybookHistoryById(workflowHistoryId uuid.UUID) (*domain.PlaybookHistoryResponse, error)
	GetPlaybookHistory(offset int, limit int, filter PlaybookHistoryFilter) ([]domain.PlaybookHistoryResponse, error)
	GetPlaybookHistoryCount(filter PlaybookHistoryFilter) (int, error)
	GetPlaybookTriggers() ([]domain.PlaybookTriggers, error)
	GetPlaybooksCount(filter PlaybookFilter) (int, error)
	GetPlaybookById(id string) (*domain.Playbooks, error)
	GetPlaybooksHistoryData(offset int, limit int, filter PlaybookHistoryFilter) (types.Entries[domain.PlaybookHistoryResponse], error)
	GetPlaybookGraphById(id string) (*domain.PlaybookGraph, error)
	CreatePlaybook(workflow PlaybookPayload) (*domain.Playbooks, error)
	UpdatePlaybook(id string, workflow UpdatePlaybookData) (*domain.Playbooks, error)
	UpdatePlaybookTx(tx *sqlx.Tx, id string, workflow UpdatePlaybookData) (*domain.Playbooks, error)
	CreatePlaybookHistory(tx *sqlx.Tx, id string, edges []domain.ResponseEdges) (*domain.PlaybookHistory, error)
	UpdatePlaybookHistory(workflowHistoryId string, workflowHistory UpdatePlaybookHistoryData) (*domain.PlaybookHistory, error)
	UpdatePlaybookHistoryStatus(workflowHistoryId string, status string) (*domain.PlaybookHistory, error)
}

type PlaybookServiceImpl struct {
	PlaybookRepository PlaybookRepository
}

func NewPlaybookService(PlaybookRepository PlaybookRepository) PlaybookService {
	return &PlaybookServiceImpl{
		PlaybookRepository: PlaybookRepository,
	}
}

// GetPlaybooks implements PlaybookService.
func (w *PlaybookServiceImpl) GetPlaybooks(offset int, limit int, filter PlaybookFilter) ([]domain.Playbooks, error) {
	return w.PlaybookRepository.GetPlaybooks(offset, limit, filter)
}

// GetPlaybooksData implements [PlaybookService].
func (w *PlaybookServiceImpl) GetPlaybooksData(offset int, limit int, filter PlaybookFilter) (types.Entries[domain.Playbooks], error) {
	workflows, err := w.GetPlaybooks(
		offset,
		limit,
		filter,
	)
	if err != nil {
		return types.Entries[domain.Playbooks]{}, err
	}

	total, err := w.GetPlaybooksCount(filter)
	if err != nil {
		return types.Entries[domain.Playbooks]{}, err
	}

	return types.Entries[domain.Playbooks]{
		Entries: workflows,
		Total:   total,
	}, nil
}

// GetPlaybooksData implements [PlaybookService].
func (w *PlaybookServiceImpl) GetPlaybooksHistoryData(offset int, limit int, filter PlaybookHistoryFilter) (types.Entries[domain.PlaybookHistoryResponse], error) {
	histories, err := w.GetPlaybookHistory(
		offset,
		limit,
		filter,
	)
	if err != nil {
		return types.Entries[domain.PlaybookHistoryResponse]{}, err
	}

	total, err := w.GetPlaybookHistoryCount(filter)
	if err != nil {
		return types.Entries[domain.PlaybookHistoryResponse]{}, err
	}

	return types.Entries[domain.PlaybookHistoryResponse]{
		Entries: histories,
		Total:   total,
	}, nil
}

func (w *PlaybookServiceImpl) GetPlaybookHistoryById(workflowHistoryId uuid.UUID) (*domain.PlaybookHistoryResponse, error) {
	return w.PlaybookRepository.GetPlaybookHistoryById(workflowHistoryId)
}

// GetPlaybookHistory implements PlaybookService.
func (w *PlaybookServiceImpl) GetPlaybookHistory(offset int, limit int, filter PlaybookHistoryFilter) ([]domain.PlaybookHistoryResponse, error) {
	return w.PlaybookRepository.GetPlaybookHistory(offset, limit, filter)
}

// GetPlaybookHistoryCount implements PlaybookService.
func (w *PlaybookServiceImpl) GetPlaybookHistoryCount(filter PlaybookHistoryFilter) (int, error) {
	return w.PlaybookRepository.GetPlaybookHistoryCount(filter)
}

// GetPlaybookTriggers implements PlaybookService.
func (w *PlaybookServiceImpl) GetPlaybookTriggers() ([]domain.PlaybookTriggers, error) {
	return w.PlaybookRepository.GetPlaybookTriggers()
}

// GetPlaybooksCount implements PlaybookService.
func (w *PlaybookServiceImpl) GetPlaybooksCount(filter PlaybookFilter) (int, error) {
	return w.PlaybookRepository.GetPlaybooksCount(filter)
}

// CreatePlaybookHistory implements PlaybookService.
func (w *PlaybookServiceImpl) CreatePlaybookHistory(tx *sqlx.Tx, id string, edges []domain.ResponseEdges) (*domain.PlaybookHistory, error) {
	return w.PlaybookRepository.CreatePlaybookHistory(tx, id, edges)
}

// GetPlaybookById implements PlaybookService.
func (w *PlaybookServiceImpl) GetPlaybookById(id string) (*domain.Playbooks, error) {
	workflow, workflowErr := w.PlaybookRepository.GetPlaybookById(id)
	if workflowErr != nil {
		logging.Sugar.Error(fmt.Sprintf("error fetching workflow by id: %v, error: %v", id, workflowErr))
		return nil, workflowErr
	}

	if workflow == nil {
		return nil, fmt.Errorf("workflow is not found")
	}
	return workflow, nil
}

// GetPlaybookById implements PlaybookService.
func (w *PlaybookServiceImpl) GetPlaybookGraphById(id string) (*domain.PlaybookGraph, error) {
	workflow, workflowErr := w.PlaybookRepository.GetPlaybookGraphById(id)
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
func (w *PlaybookServiceImpl) CreatePlaybook(workflow PlaybookPayload) (*domain.Playbooks, error) {
	return w.PlaybookRepository.CreatePlaybook(workflow)
}

// updatePlaybook implements PlaybookRepository.
func (w *PlaybookServiceImpl) UpdatePlaybook(id string, workflow UpdatePlaybookData) (*domain.Playbooks, error) {
	return w.PlaybookRepository.UpdatePlaybook(id, workflow)
}

// updatePlaybookTx implements PlaybookRepository.
func (w *PlaybookServiceImpl) UpdatePlaybookTx(tx *sqlx.Tx, id string, workflow UpdatePlaybookData) (*domain.Playbooks, error) {
	return w.PlaybookRepository.UpdatePlaybookTx(tx, id, workflow)
}

// UpdatePlaybookHistoryStatus implements PlaybookRepository.
func (w *PlaybookServiceImpl) UpdatePlaybookHistoryStatus(workflowHistoryId string, status string) (*domain.PlaybookHistory, error) {
	res, err := w.PlaybookRepository.UpdatePlaybookHistoryStatus(workflowHistoryId, status)

	if err != nil {
		logging.Sugar.Error(fmt.Sprintf("error updating status of workflowHistoryId by id: %s, error: %v", workflowHistoryId, err))
		return nil, fmt.Errorf("error updating workflowHistoryId by id: %s", workflowHistoryId)
	}

	if res == nil {
		return nil, fmt.Errorf("no workflow status was updated")
	}

	return res, nil
}

// UpdatePlaybookHistoryStatus implements PlaybookRepository.
func (w *PlaybookServiceImpl) UpdatePlaybookHistory(workflowHistoryId string, workflowHistory UpdatePlaybookHistoryData) (*domain.PlaybookHistory, error) {
	res, err := w.PlaybookRepository.UpdatePlaybookHistory(workflowHistoryId, workflowHistory)

	if err != nil {
		logging.Sugar.Error(fmt.Sprintf("error updating workflowHistoryId by id: %s, error: %v", workflowHistoryId, err))
		return nil, fmt.Errorf("error updating workflowHistoryId by id: %s", workflowHistoryId)
	}

	if res == nil {
		return nil, fmt.Errorf("no workflow status was updated")
	}

	return res, nil
}
