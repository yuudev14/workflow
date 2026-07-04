package edges

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14/ytsoar/internal/domain"
)

//go:generate mockgen -destination=mocks/repository_mock.go -package=mocks . EdgeRepository

type EdgeRepository interface {
	InsertEdges(tx *sqlx.Tx, edges []domain.Edges) ([]domain.Edges, error)
	DeleteEdges(tx *sqlx.Tx, edgeIds []uuid.UUID) error
	DeleteAllPlaybookEdges(tx *sqlx.Tx, playbookId string) error
	GetEdgesByPlaybookId(playbookId string) ([]domain.ResponseEdges, error)
}
