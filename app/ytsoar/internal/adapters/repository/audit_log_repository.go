package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// AuditLogRepositoryImpl implements auth.AuditLogRepository.
type AuditLogRepositoryImpl struct {
	logger logger.Logger
	q      QuerierTx
	pool   *pgxpool.Pool
}

func NewAuditLogRepositoryImpl(log logger.Logger, q QuerierTx, pool *pgxpool.Pool) *AuditLogRepositoryImpl {
	return &AuditLogRepositoryImpl{logger: log, q: q, pool: pool}
}

func (r *AuditLogRepositoryImpl) queriesFromContext(ctx context.Context) db.Querier {
	if tx, ok := txFromContext(ctx); ok {
		return r.q.WithTx(tx)
	}
	return r.q
}

func (r *AuditLogRepositoryImpl) Insert(ctx context.Context, entry domain.AuditEntry) error {
	var detail json.RawMessage
	if entry.Detail != nil {
		encoded, err := json.Marshal(entry.Detail)
		if err != nil {
			return err
		}
		detail = encoded
	}

	actorID := pgtype.UUID{}
	if entry.ActorID != nil {
		actorID = toPgUUID(*entry.ActorID)
	}

	_, err := r.queriesFromContext(ctx).InsertAuditLog(ctx, db.InsertAuditLogParams{
		ActorID:  actorID,
		Module:   entry.Module,
		Action:   entry.Action,
		EntityID: toPgText(entry.EntityID),
		Detail:   detail,
	})
	return err
}
