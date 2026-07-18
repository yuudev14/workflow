package repository

import (
	"github.com/jackc/pgx/v5"
	"github.com/yuudev14/ytsoar/db"
)

//go:generate mockgen -destination=mocks/querier_mock.go -package=mocks . QuerierTx
type QuerierTx interface {
	db.Querier
	WithTx(tx pgx.Tx) *db.Queries
}
