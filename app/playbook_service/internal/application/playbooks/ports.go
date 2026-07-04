package playbooks

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14/ytsoar/internal/domain"
)

//go:generate mockgen -destination=mocks/repository_mock.go -package=mocks . PlaybookRepository

type PlaybookRepository interface {
	GetPlaybooks(offset int, limit int, filter PlaybookFilter) ([]domain.Playbooks, error)
	GetPlaybookHistoryById(playbookHistoryId uuid.UUID) (*domain.PlaybookHistoryResponse, error)
	GetPlaybookHistory(offset int, limit int, filter PlaybookHistoryFilter) ([]domain.PlaybookHistoryResponse, error)
	GetPlaybookHistoryCount(filter PlaybookHistoryFilter) (int, error)
	GetPlaybookTriggers() ([]domain.PlaybookTriggers, error)
	GetPlaybooksCount(filter PlaybookFilter) (int, error)
	GetPlaybookById(id string) (*domain.Playbooks, error)

	GetPlaybookGraphById(id string) (*domain.PlaybookGraph, error)
	CreatePlaybook(playbook PlaybookPayload) (*domain.Playbooks, error)
	UpdatePlaybook(id string, playbook UpdatePlaybookData) (*domain.Playbooks, error)
	UpdatePlaybookTx(tx *sqlx.Tx, id string, playbook UpdatePlaybookData) (*domain.Playbooks, error)
	CreatePlaybookHistory(tx *sqlx.Tx, id string, edges []domain.ResponseEdges) (*domain.PlaybookHistory, error)
	UpdatePlaybookHistoryStatus(playbookHistoryId string, status string) (*domain.PlaybookHistory, error)
	UpdatePlaybookHistory(playbookHistoryId string, playbookHistory UpdatePlaybookHistoryData) (*domain.PlaybookHistory, error)
}
