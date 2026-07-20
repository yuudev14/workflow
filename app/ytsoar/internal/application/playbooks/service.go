package playbooks

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
	"github.com/yuudev14/ytsoar/internal/logger"
	"github.com/yuudev14/ytsoar/internal/types"
)

// ErrPlaybookNotFound lets handlers map a missing playbook to 404 with
// errors.Is instead of matching error strings.
var ErrPlaybookNotFound = apperr.New(apperr.NotFound, "playbook is not found")

//go:generate mockgen -destination=mocks/service_mock.go -package=mocks . PlaybookService
type PlaybookService interface {
	GetPlaybooks(ctx context.Context, offset int, limit int, filter PlaybookFilter) ([]domain.Playbooks, error)
	GetPlaybooksData(ctx context.Context, offset int, limit int, filter PlaybookFilter) (types.Entries[domain.Playbooks], error)
	GetPlaybookHistoryById(ctx context.Context, playbookHistoryId uuid.UUID) (*domain.PlaybookHistoryResponse, error)
	GetPlaybookHistory(ctx context.Context, offset int, limit int, filter PlaybookHistoryFilter) ([]domain.PlaybookHistoryResponse, error)
	GetPlaybookHistoryCount(ctx context.Context, filter PlaybookHistoryFilter) (int, error)
	GetPlaybooksCount(ctx context.Context, filter PlaybookFilter) (int, error)
	GetPlaybookById(ctx context.Context, id string) (*domain.Playbooks, error)
	GetPlaybooksHistoryData(ctx context.Context, offset int, limit int, filter PlaybookHistoryFilter) (types.Entries[domain.PlaybookHistoryResponse], error)
	GetPlaybookGraphById(ctx context.Context, id string) (*domain.PlaybookGraph, error)
	CreatePlaybook(ctx context.Context, playbook PlaybookPayload) (*domain.Playbooks, error)
	UpdatePlaybook(ctx context.Context, id string, playbook UpdatePlaybookData) (*domain.Playbooks, error)
	CreatePlaybookHistory(ctx context.Context, id string, edges []domain.ResponseEdges) (*domain.PlaybookHistory, error)
	UpdatePlaybookHistory(ctx context.Context, playbookHistoryId string, playbookHistory UpdatePlaybookHistoryData) (*domain.PlaybookHistory, error)
	UpdatePlaybookHistoryStatus(ctx context.Context, playbookHistoryId string, status string) (*domain.PlaybookHistory, error)
}

type PlaybookServiceImpl struct {
	logger             logger.Logger
	PlaybookRepository PlaybookRepository
}

func NewPlaybookService(log logger.Logger, playbookRepository PlaybookRepository) PlaybookService {
	return &PlaybookServiceImpl{
		logger:             log,
		PlaybookRepository: playbookRepository,
	}
}

// GetPlaybooks implements PlaybookService.
func (w *PlaybookServiceImpl) GetPlaybooks(ctx context.Context, offset int, limit int, filter PlaybookFilter) ([]domain.Playbooks, error) {
	return w.PlaybookRepository.GetPlaybooks(ctx, offset, limit, filter)
}

// GetPlaybooksData implements [PlaybookService].
func (w *PlaybookServiceImpl) GetPlaybooksData(ctx context.Context, offset int, limit int, filter PlaybookFilter) (types.Entries[domain.Playbooks], error) {
	playbooks, err := w.GetPlaybooks(ctx, offset, limit, filter)
	if err != nil {
		return types.Entries[domain.Playbooks]{}, err
	}

	total, err := w.GetPlaybooksCount(ctx, filter)
	if err != nil {
		return types.Entries[domain.Playbooks]{}, err
	}

	return types.Entries[domain.Playbooks]{
		Entries: playbooks,
		Total:   total,
	}, nil
}

// GetPlaybooksHistoryData implements [PlaybookService].
func (w *PlaybookServiceImpl) GetPlaybooksHistoryData(ctx context.Context, offset int, limit int, filter PlaybookHistoryFilter) (types.Entries[domain.PlaybookHistoryResponse], error) {
	histories, err := w.GetPlaybookHistory(ctx, offset, limit, filter)
	if err != nil {
		return types.Entries[domain.PlaybookHistoryResponse]{}, err
	}

	total, err := w.GetPlaybookHistoryCount(ctx, filter)
	if err != nil {
		return types.Entries[domain.PlaybookHistoryResponse]{}, err
	}

	return types.Entries[domain.PlaybookHistoryResponse]{
		Entries: histories,
		Total:   total,
	}, nil
}

func (w *PlaybookServiceImpl) GetPlaybookHistoryById(ctx context.Context, playbookHistoryId uuid.UUID) (*domain.PlaybookHistoryResponse, error) {
	return w.PlaybookRepository.GetPlaybookHistoryById(ctx, playbookHistoryId)
}

// GetPlaybookHistory implements PlaybookService.
func (w *PlaybookServiceImpl) GetPlaybookHistory(ctx context.Context, offset int, limit int, filter PlaybookHistoryFilter) ([]domain.PlaybookHistoryResponse, error) {
	return w.PlaybookRepository.GetPlaybookHistory(ctx, offset, limit, filter)
}

// GetPlaybookHistoryCount implements PlaybookService.
func (w *PlaybookServiceImpl) GetPlaybookHistoryCount(ctx context.Context, filter PlaybookHistoryFilter) (int, error) {
	return w.PlaybookRepository.GetPlaybookHistoryCount(ctx, filter)
}

// GetPlaybooksCount implements PlaybookService.
func (w *PlaybookServiceImpl) GetPlaybooksCount(ctx context.Context, filter PlaybookFilter) (int, error) {
	return w.PlaybookRepository.GetPlaybooksCount(ctx, filter)
}

// CreatePlaybookHistory implements PlaybookService.
func (w *PlaybookServiceImpl) CreatePlaybookHistory(ctx context.Context, id string, edges []domain.ResponseEdges) (*domain.PlaybookHistory, error) {
	return w.PlaybookRepository.CreatePlaybookHistory(ctx, id, edges)
}

// GetPlaybookById implements PlaybookService.
func (w *PlaybookServiceImpl) GetPlaybookById(ctx context.Context, id string) (*domain.Playbooks, error) {
	playbook, playbookErr := w.PlaybookRepository.GetPlaybookById(ctx, id)
	if playbookErr != nil {
		w.logger.Error(fmt.Sprintf("error fetching playbook by id: %v, error: %v", id, playbookErr))
		return nil, playbookErr
	}

	if playbook == nil {
		return nil, ErrPlaybookNotFound
	}
	return playbook, nil
}

// GetPlaybookGraphById implements PlaybookService.
func (w *PlaybookServiceImpl) GetPlaybookGraphById(ctx context.Context, id string) (*domain.PlaybookGraph, error) {
	playbook, playbookErr := w.PlaybookRepository.GetPlaybookGraphById(ctx, id)
	if playbookErr != nil {
		w.logger.Error(fmt.Sprintf("error fetching playbook by id: %s, error: %v", id, playbookErr))
		return nil, fmt.Errorf("error fetching graph by playbook by id: %s", id)
	}

	if playbook == nil {
		return nil, ErrPlaybookNotFound
	}
	return playbook, nil
}

// CreatePlaybook implements PlaybookService.
func (w *PlaybookServiceImpl) CreatePlaybook(ctx context.Context, playbook PlaybookPayload) (*domain.Playbooks, error) {
	return w.PlaybookRepository.CreatePlaybook(ctx, playbook)
}

// UpdatePlaybook implements PlaybookService.
func (w *PlaybookServiceImpl) UpdatePlaybook(ctx context.Context, id string, playbook UpdatePlaybookData) (*domain.Playbooks, error) {
	if playbook.TriggerType.Set && playbook.TriggerType.Value != nil && !domain.IsValidTriggerType(*playbook.TriggerType.Value) {
		return nil, fmt.Errorf("unknown trigger type: %s", *playbook.TriggerType.Value)
	}
	return w.PlaybookRepository.UpdatePlaybook(ctx, id, playbook)
}

// UpdatePlaybookHistoryStatus implements PlaybookService.
func (w *PlaybookServiceImpl) UpdatePlaybookHistoryStatus(ctx context.Context, playbookHistoryId string, status string) (*domain.PlaybookHistory, error) {
	res, err := w.PlaybookRepository.UpdatePlaybookHistoryStatus(ctx, playbookHistoryId, status)
	if err != nil {
		w.logger.Error(fmt.Sprintf("error updating status of playbookHistoryId by id: %s, error: %v", playbookHistoryId, err))
		return nil, fmt.Errorf("error updating playbookHistoryId by id: %s", playbookHistoryId)
	}

	if res == nil {
		return nil, fmt.Errorf("no playbook status was updated")
	}

	return res, nil
}

// UpdatePlaybookHistory implements PlaybookService.
func (w *PlaybookServiceImpl) UpdatePlaybookHistory(ctx context.Context, playbookHistoryId string, playbookHistory UpdatePlaybookHistoryData) (*domain.PlaybookHistory, error) {
	res, err := w.PlaybookRepository.UpdatePlaybookHistory(ctx, playbookHistoryId, playbookHistory)
	if err != nil {
		w.logger.Error(fmt.Sprintf("error updating playbookHistoryId by id: %s, error: %v", playbookHistoryId, err))
		return nil, fmt.Errorf("error updating playbookHistoryId by id: %s", playbookHistoryId)
	}

	if res == nil {
		return nil, fmt.Errorf("no playbook status was updated")
	}

	return res, nil
}
